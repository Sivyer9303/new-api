/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button, buttonVariants } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { getApiKeys, fetchTokenKey } from '@/features/keys/api'
import type { ApiKey } from '@/features/keys/types'
import { usePricingData } from '@/features/pricing/hooks/use-pricing-data'
import { getConfiguredGroupRatio } from '@/features/pricing/lib/model-helpers'
import { formatCurrencyFromUSD } from '@/lib/currency'
import { cn } from '@/lib/utils'

import {
  fetchModelsWithTokenKey,
  fetchVideoContentBlob,
  fetchVideoGeneration,
  fetchVideoToolConfig,
  submitVideoGeneration,
} from '../api'
import {
  filterModelsForProfile,
  isVideoStoragePhase,
  modelHasConfiguredMatch,
  resolveVideoProfile,
} from '../lib/capabilities'
import {
  revokeReferenceImageItems,
  type ReferenceImageItem,
} from '../lib/reference-image'
import { buildVideoGenerationRequest } from '../lib/request'
import type { PublicGenerationType } from '../types'
import { ReferenceImageGrid } from './reference-image-grid'

function generationTypeDisplayLabel(
  gt: PublicGenerationType,
  translate: (key: string) => string
): string {
  switch (gt.value) {
    case 'text2video':
      return translate('Text to video')
    case 'image2video':
      return translate('Image to video')
    case 'multi_image':
      return translate('Multi-image reference')
    case 'start_end':
      return translate('First & last frame')
    case 'reference_audio':
      return translate('Reference audio')
    default:
      return gt.label
  }
}

function isTerminalSuccess(status: string | undefined): boolean {
  const s = (status || '').toLowerCase()
  return s === 'completed' || s === 'success'
}

function isTerminalFailure(status: string | undefined): boolean {
  const s = (status || '').toLowerCase()
  return (
    s === 'failed' || s === 'failure' || s === 'cancelled' || s === 'canceled'
  )
}

function isTerminalStatus(status: string | undefined): boolean {
  return isTerminalSuccess(status) || isTerminalFailure(status)
}

function apiKeySelectLabel(key: ApiKey): string {
  const name = key.name?.trim() || `Key #${key.id}`
  const group = key.group?.trim() || 'default'
  return `${name} (${group})`
}

const POLL_INTERVAL_MS = 3000
const POLL_MAX_ATTEMPTS = 120

async function filesToDataUrls(files: File[]): Promise<string[]> {
  if (files.length === 0) return []
  const readers = files.map(
    (file) =>
      new Promise<string>((resolve, reject) => {
        const reader = new FileReader()
        reader.onload = () => resolve(String(reader.result || ''))
        reader.onerror = () => reject(reader.error ?? new Error('read failed'))
        reader.readAsDataURL(file)
      })
  )
  return Promise.all(readers)
}

async function fileToDataURL(file: File): Promise<string> {
  const [url] = await filesToDataUrls([file])
  return url
}

export function VideoToolPage() {
  const { t } = useTranslation()
  const [tokenId, setTokenId] = useState<string>('')
  const [models, setModels] = useState<string[]>([])
  const [loadingModels, setLoadingModels] = useState(false)
  const [modelId, setModelId] = useState('')
  const [generationType, setGenerationType] = useState('')
  const [prompt, setPrompt] = useState('')
  const [durationValue, setDurationValue] = useState('')
  const [aspectRatio, setAspectRatio] = useState('')
  const [referenceImages, setReferenceImages] = useState<ReferenceImageItem[]>(
    []
  )
  const [audioFile, setAudioFile] = useState<File | null>(null)
  const [audioPreviewUrl, setAudioPreviewUrl] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [taskId, setTaskId] = useState('')
  const [taskStatus, setTaskStatus] = useState('')
  const [taskProgress, setTaskProgress] = useState('')
  const [previewUrl, setPreviewUrl] = useState('')
  const [pollError, setPollError] = useState('')
  const [pollingTokenKey, setPollingTokenKey] = useState('')
  const pollAttemptRef = useRef(0)
  const seenTerminalToastRef = useRef('')
  const loadModelsRequestRef = useRef(0)

  const configQuery = useQuery({
    queryKey: ['video-tool-config', 1],
    queryFn: async () => {
      const res = await fetchVideoToolConfig()
      if (!res.success || !res.data) {
        throw new Error(res.message || 'Failed to load video tool config')
      }
      if (res.data.version !== 1) {
        throw new Error(t('Unsupported video tool capability version'))
      }
      return res.data
    },
  })

  const keysQuery = useQuery({
    queryKey: ['video-tool-api-keys'],
    queryFn: async () => {
      const res = await getApiKeys({ p: 1, size: 100 })
      if (!res.success || !res.data?.items) {
        throw new Error(res.message || 'Failed to load API keys')
      }
      return res.data.items.filter((k: ApiKey) => k.status === 1)
    },
  })

  const { models: pricingModels, groupRatio } = usePricingData()

  // Models with a configured ModelPrice (> 0). Unpriced models are hidden
  // from the selector so users cannot submit without an estimate path.
  const pricedModelIds = useMemo(() => {
    const ids = new Set<string>()
    for (const m of pricingModels) {
      if (m.model_price != null && m.model_price > 0) {
        ids.add(m.model_name)
      }
    }
    return ids
  }, [pricingModels])

  const profiles = useMemo(
    () => configQuery.data?.profiles ?? [],
    [configQuery.data?.profiles]
  )
  const videoToolGroups = useMemo(
    () => configQuery.data?.video_tool_groups ?? [],
    [configQuery.data?.video_tool_groups]
  )
  const allowedGroupSet = useMemo(
    () => new Set(videoToolGroups.map((g) => g.trim()).filter(Boolean)),
    [videoToolGroups]
  )
  const keys = useMemo(() => {
    const all = keysQuery.data ?? []
    if (allowedGroupSet.size === 0) return []
    return all.filter((k) => allowedGroupSet.has((k.group || '').trim()))
  }, [keysQuery.data, allowedGroupSet])

  useEffect(() => {
    if (!tokenId) return
    if (!keys.some((k) => String(k.id) === tokenId)) {
      setTokenId('')
      setModels([])
      setModelId('')
    }
  }, [keys, tokenId])

  // Selecting / changing an API key automatically loads models.
  useEffect(() => {
    if (!tokenId) {
      setModels([])
      setModelId('')
      setLoadingModels(false)
      return
    }
    if (profiles.length === 0) return

    const requestId = ++loadModelsRequestRef.current
    let cancelled = false

    const load = async () => {
      setLoadingModels(true)
      try {
        const keyRes = await fetchTokenKey(Number(tokenId))
        if (cancelled || requestId !== loadModelsRequestRef.current) return
        if (!keyRes.success || !keyRes.data?.key) {
          throw new Error(keyRes.message || 'Failed to fetch API key')
        }
        const all = await fetchModelsWithTokenKey(keyRes.data.key)
        if (cancelled || requestId !== loadModelsRequestRef.current) return
        const defaultProfileID = configQuery.data?.default_profile_id ?? ''
        const hasDefaultProfile = profiles.some(
          (profile) => profile.id === defaultProfileID
        )
        const matched = hasDefaultProfile
          ? all
          : all.filter((id) => modelHasConfiguredMatch(profiles, id))
        setModels(matched)
        if (matched.length === 0) {
          setModelId('')
          toast.error(t('No video models available for this key'))
        }
        // Selection is synced from filteredModels (priced + mode) via safeModelId.
      } catch (err) {
        if (cancelled || requestId !== loadModelsRequestRef.current) return
        setModels([])
        setModelId('')
        toast.error(
          err instanceof Error ? err.message : t('Failed to load models')
        )
      } finally {
        if (!cancelled && requestId === loadModelsRequestRef.current) {
          setLoadingModels(false)
        }
      }
    }

    void load()
    return () => {
      cancelled = true
    }
  }, [tokenId, profiles, configQuery.data?.default_profile_id, t])

  const selectedApiKey = useMemo(
    () => keys.find((k) => String(k.id) === tokenId) ?? null,
    [keys, tokenId]
  )

  const selectedProfile = useMemo(() => {
    const id = modelId || ''
    return id
      ? resolveVideoProfile(
          profiles,
          id,
          configQuery.data?.default_profile_id ?? ''
        )
      : null
  }, [modelId, profiles, configQuery.data?.default_profile_id])

  const generationTypes = useMemo(
    () => configQuery.data?.generation_types ?? [],
    [configQuery.data?.generation_types]
  )

  const selectedGenType = useMemo(() => {
    if (!generationType) return null
    return generationTypes.find((g) => g.value === generationType) ?? null
  }, [generationType, generationTypes])

  const durationOptions = selectedProfile?.durations ?? []
  const aspectOptions = selectedProfile?.aspect_ratios ?? []

  const durationFieldKey =
    durationOptions.find((d) => d.value === durationValue)?.upstream_key ||
    durationOptions[0]?.upstream_key ||
    'seconds'

  useEffect(() => {
    if (
      !generationType ||
      !generationTypes.some((g) => g.value === generationType)
    ) {
      setGenerationType(generationTypes[0]?.value ?? '')
    }
  }, [generationType, generationTypes])

  useEffect(() => {
    if (!selectedProfile) return
    if (
      !durationValue ||
      !selectedProfile.durations.some((d) => d.value === durationValue)
    ) {
      setDurationValue(selectedProfile.durations[0]?.value ?? '')
    }
    if (
      !aspectRatio ||
      !selectedProfile.aspect_ratios.some((a) => a.value === aspectRatio)
    ) {
      setAspectRatio(selectedProfile.aspect_ratios[0]?.value ?? '')
    }
  }, [selectedProfile, durationValue, aspectRatio])

  // Trim or clear reference images when the selected mode's image limit changes.
  useEffect(() => {
    const maxImg = selectedGenType?.images_max ?? 0
    if (maxImg <= 0) {
      if (referenceImages.length > 0) {
        revokeReferenceImageItems(referenceImages)
        setReferenceImages([])
      }
      return
    }
    if (referenceImages.length > maxImg) {
      const keep = referenceImages.slice(0, maxImg)
      const dropped = referenceImages.slice(maxImg)
      revokeReferenceImageItems(dropped)
      setReferenceImages(keep)
      toast.message(
        t('Removed extra images to fit this mode (max {{max}}).', {
          max: maxImg,
        })
      )
    }
    // intentionally only react to generation-type limit changes
    // eslint-disable-next-line react-hooks/exhaustive-deps -- trim on mode change, not every image edit
  }, [selectedGenType?.images_max, selectedGenType?.value])

  // Clear reference audio when the mode does not allow it.
  useEffect(() => {
    if (selectedGenType?.allow_audio) return
    setAudioFile(null)
    setAudioPreviewUrl((prev) => {
      if (prev) URL.revokeObjectURL(prev)
      return ''
    })
  }, [selectedGenType?.allow_audio, selectedGenType?.value])

  useEffect(() => {
    return () => {
      revokeReferenceImageItems(referenceImages)
      if (audioPreviewUrl) URL.revokeObjectURL(audioPreviewUrl)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- revoke current list on unmount only
  }, [])

  const availableModels = useMemo(
    () => models.filter((id) => pricedModelIds.has(id)),
    [models, pricedModelIds]
  )

  const noPricedModelsToastKeyRef = useRef('')
  // The key returned video models, but none have ModelPrice configured.
  useEffect(() => {
    if (loadingModels || !tokenId) return
    if (models.length === 0 || availableModels.length > 0) return
    if (pricingModels.length === 0) return
    const toastKey = `${tokenId}:${models.join('\0')}`
    if (noPricedModelsToastKeyRef.current === toastKey) return
    noPricedModelsToastKeyRef.current = toastKey
    toast.error(
      t(
        'No video models with configured pricing are available. Set a model price in Model Pricing first.'
      )
    )
  }, [
    loadingModels,
    tokenId,
    models,
    availableModels.length,
    pricingModels.length,
    t,
  ])

  const filteredModels = useMemo(() => {
    const defaultProfileID = configQuery.data?.default_profile_id ?? ''
    const hasDefaultProfile = profiles.some(
      (profile) => profile.id === defaultProfileID
    )
    if (!selectedProfile) {
      return hasDefaultProfile
        ? availableModels
        : availableModels.filter((id) => modelHasConfiguredMatch(profiles, id))
    }
    return filterModelsForProfile(
      availableModels,
      selectedProfile,
      Boolean(selectedGenType?.require_ref_model),
      selectedProfile.id === defaultProfileID,
      profiles
    )
  }, [
    availableModels,
    profiles,
    selectedProfile,
    selectedGenType,
    configQuery.data?.default_profile_id,
  ])

  // Base UI Select throws if value is not among items. When switching to a
  // require_ref generation type, filteredModels shrinks to *-ref ids before
  // the sync effect can update modelId — keep a render-safe selection.
  const safeModelId = useMemo(() => {
    if (filteredModels.length === 0) return ''
    if (modelId && filteredModels.includes(modelId)) return modelId
    return filteredModels[0]
  }, [filteredModels, modelId])

  useEffect(() => {
    if (safeModelId === modelId) return
    setModelId(safeModelId)
  }, [safeModelId, modelId])

  const priceEstimate = useMemo(() => {
    if (!safeModelId || !durationValue) return null
    const pricing = pricingModels.find((m) => m.model_name === safeModelId)
    if (!pricing || pricing.model_price == null || pricing.model_price <= 0) {
      return null
    }
    const seconds = Number(durationValue)
    if (!Number.isFinite(seconds) || seconds <= 0) return null
    const group =
      (selectedApiKey?.group || '').trim() || videoToolGroups[0] || 'default'
    const ratios = pricing.group_ratio || groupRatio || {}
    const ratio = getConfiguredGroupRatio(ratios, group)
    // NewAPI video adaptor always multiplies ModelPrice by duration seconds.
    const usd = pricing.model_price * seconds * ratio
    let formatted = '-'
    try {
      formatted = formatCurrencyFromUSD(usd)
    } catch {
      formatted = `$${usd}`
    }
    return {
      usd,
      seconds,
      unitPrice: pricing.model_price,
      group,
      ratio,
      formatted,
    }
  }, [
    safeModelId,
    durationValue,
    pricingModels,
    selectedApiKey,
    videoToolGroups,
    groupRatio,
  ])

  const requestPreview = useMemo(() => {
    const body: Record<string, unknown> = {
      model: safeModelId,
      prompt,
      generation_type: generationType,
      aspect_ratio: aspectRatio,
    }
    if (durationFieldKey === 'duration') {
      body.duration = Number(durationValue)
    } else {
      body.seconds = durationValue
    }
    if ((selectedGenType?.images_max ?? 0) > 0 && referenceImages.length > 0) {
      // Preview only: full base64 payloads are substituted on submit.
      body.images = referenceImages.map((item) => {
        const mime = item.file.type || 'image/jpeg'
        return `data:${mime};base64,…(${item.file.name || 'image'})`
      })
    }
    if (selectedGenType?.allow_audio) {
      body.audio_url = audioFile
        ? `data:audio/mpeg;base64,…(${audioFile.name})`
        : 'data:audio/mpeg;base64,…'
    }
    return JSON.stringify(body, null, 2)
  }, [
    safeModelId,
    prompt,
    generationType,
    aspectRatio,
    durationFieldKey,
    durationValue,
    selectedGenType,
    referenceImages,
    audioFile,
  ])

  async function handleSubmit() {
    if (!tokenId) {
      toast.error(t('Please select an API key'))
      return
    }
    if (
      !safeModelId ||
      !generationType ||
      !prompt.trim() ||
      !durationValue ||
      !aspectRatio
    ) {
      toast.error(t('Please fill in all required fields'))
      return
    }
    if (!selectedProfile || !selectedGenType) {
      toast.error(t('Selected model is not supported by video capabilities'))
      return
    }

    const minImg = selectedGenType.images_min
    const maxImg = selectedGenType.images_max
    const fileCount = referenceImages.length
    if (fileCount < minImg || fileCount > maxImg) {
      toast.error(
        t('This mode requires {{min}}-{{max}} reference image(s)', {
          min: minImg,
          max: maxImg,
        })
      )
      return
    }
    if (selectedGenType.require_audio && !audioFile) {
      toast.error(t('This mode requires an MP3 reference audio file'))
      return
    }

    setSubmitting(true)
    setPollError('')
    setPreviewUrl('')
    setTaskStatus('')
    setTaskProgress('')
    setTaskId('')
    setPollingTokenKey('')
    pollAttemptRef.current = 0
    seenTerminalToastRef.current = ''
    try {
      const keyRes = await fetchTokenKey(Number(tokenId))
      if (!keyRes.success || !keyRes.data?.key) {
        throw new Error(keyRes.message || 'Failed to fetch API key')
      }
      const tokenKey = keyRes.data.key
      const images = await filesToDataUrls(
        referenceImages.map((item) => item.file)
      )
      let audioURL = ''
      if (selectedGenType.allow_audio && audioFile) {
        audioURL = await fileToDataURL(audioFile)
        const lower = audioURL.toLowerCase()
        if (
          !lower.startsWith('data:audio/mpeg;base64,') &&
          !lower.startsWith('data:audio/mp3;base64,')
        ) {
          throw new Error(t('Reference audio must be an MP3 file'))
        }
      }

      let submitModel = safeModelId
      if (selectedGenType.require_ref_model && !submitModel.includes('-ref')) {
        const refCandidate = filteredModels.find((id) => id.includes('-ref'))
        if (refCandidate) {
          submitModel = refCandidate
        }
      }

      const body = buildVideoGenerationRequest({
        model: submitModel,
        prompt: prompt.trim(),
        generationType,
        aspectRatio,
        durationFieldKey,
        durationValue,
        images,
        audioURL,
      })

      const submitRes = await submitVideoGeneration(tokenKey, body)
      const publicId = submitRes.id || submitRes.task_id
      if (!publicId) {
        throw new Error(t('Submit succeeded but no task id returned'))
      }
      setTaskId(publicId)
      setTaskStatus(submitRes.status || 'queued')
      setTaskProgress('')
      setPollingTokenKey(tokenKey)
      toast.success(t('Video task submitted'))
    } catch (err) {
      const message =
        err instanceof Error ? err.message : t('Failed to submit video task')
      toast.error(message)
      setPollError(message)
    } finally {
      setSubmitting(false)
    }
  }

  const isPolling =
    Boolean(taskId) && Boolean(pollingTokenKey) && !isTerminalStatus(taskStatus)

  useEffect(() => {
    if (!taskId || !pollingTokenKey) return

    let cancelled = false
    let timer: ReturnType<typeof setTimeout> | undefined

    const pollOnce = async () => {
      if (cancelled) return
      if (pollAttemptRef.current >= POLL_MAX_ATTEMPTS) {
        setPollError(t('Timed out waiting for video. Check Task Logs.'))
        toast.message(t('Timed out waiting for video. Check Task Logs.'))
        setPollingTokenKey('')
        return
      }
      pollAttemptRef.current += 1
      try {
        const statusRes = await fetchVideoGeneration(pollingTokenKey, taskId)
        if (cancelled) return
        const st = statusRes.status || ''
        if (st) {
          setTaskStatus(st)
        }
        if (statusRes.progress != null && String(statusRes.progress).trim()) {
          setTaskProgress(String(statusRes.progress))
        }
        if (isTerminalSuccess(st)) {
          // Only this site's content endpoint — never upstream CDN.
          const siteContent = `/v1/videos/${taskId}/content`
          let playable = siteContent
          try {
            const blob = await fetchVideoContentBlob(pollingTokenKey, taskId)
            if (cancelled) return
            playable = URL.createObjectURL(blob)
          } catch {
            playable = siteContent
          }
          setPreviewUrl(playable)
          if (seenTerminalToastRef.current !== taskId) {
            seenTerminalToastRef.current = taskId
            toast.success(t('Video generation completed'))
          }
          return
        }
        if (isTerminalFailure(st)) {
          const msg =
            statusRes.error?.message ||
            statusRes.fail_reason ||
            t('Video generation failed')
          setPollError(msg)
          if (seenTerminalToastRef.current !== taskId) {
            seenTerminalToastRef.current = taskId
            toast.error(msg)
          }
          return
        }
      } catch (err) {
        if (cancelled) return
        // Keep polling on transient fetch errors; surface the latest message.
        setPollError(
          err instanceof Error ? err.message : t('Failed to fetch video task')
        )
      }
      if (!cancelled) {
        timer = setTimeout(() => {
          void pollOnce()
        }, POLL_INTERVAL_MS)
      }
    }

    // First poll shortly after submit so UI updates without waiting a full interval.
    timer = setTimeout(() => {
      void pollOnce()
    }, 800)

    return () => {
      cancelled = true
      if (timer) clearTimeout(timer)
    }
  }, [taskId, pollingTokenKey, t])

  if (configQuery.isLoading || keysQuery.isLoading) {
    return (
      <SectionPageLayout>
        <SectionPageLayout.Title>
          {t('Video Generation')}
        </SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <div className='text-muted-foreground p-4 text-sm'>
            {t('Loading...')}
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>
    )
  }

  if (configQuery.isError || !configQuery.data?.enabled) {
    return (
      <SectionPageLayout>
        <SectionPageLayout.Title>
          {t('Video Generation')}
        </SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <Card>
            <CardHeader>
              <CardTitle>{t('Video generation unavailable')}</CardTitle>
              <CardDescription>
                {t(
                  'Video generation is not configured or enabled. Ask an administrator to review Video Configuration.'
                )}
              </CardDescription>
            </CardHeader>
          </Card>
        </SectionPageLayout.Content>
      </SectionPageLayout>
    )
  }

  const allowedGroupsLabel =
    videoToolGroups.length > 0 ? videoToolGroups.join(', ') : t('(none)')

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Video Generation')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='mx-auto flex w-full max-w-6xl flex-col gap-4 pb-8'>
          <Alert className='border-amber-500/40 bg-amber-500/10 px-4 py-3'>
            <AlertTitle className='text-base font-semibold text-amber-950 dark:text-amber-100'>
              {t('API key group required')}
            </AlertTitle>
            <AlertDescription className='text-base leading-relaxed text-amber-950/90 dark:text-amber-50/90'>
              <p>
                {t(
                  'You must select an API key from an allowed group. If none appear, create a key for those groups on the API Keys page.'
                )}
              </p>
              <p className='mt-1 font-medium'>
                {t('Allowed groups')}: {allowedGroupsLabel}
              </p>
            </AlertDescription>
          </Alert>
          <p className='text-muted-foreground text-sm'>
            {t(
              'Select an API key, choose a mode, and generate video. Models load automatically after you pick a key. Results also appear in Task Logs.'
            )}
          </p>

          <div className='grid items-start gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(320px,0.9fr)]'>
            <div className='flex min-w-0 flex-col gap-4'>
              <Card>
                <CardHeader className='pb-3'>
                  <CardTitle className='text-base'>{t('API key')}</CardTitle>
                  <CardDescription>
                    {t(
                      'You must select a key. The key is fetched only for this session request and is not stored in the page.'
                    )}
                  </CardDescription>
                </CardHeader>
                <CardContent className='space-y-2'>
                  <Label>{t('Your API key')}</Label>
                  <Select
                    value={tokenId || null}
                    onValueChange={(v) => setTokenId(v ?? '')}
                    disabled={keys.length === 0}
                  >
                    <SelectTrigger className='w-full'>
                      <SelectValue
                        placeholder={
                          loadingModels
                            ? t('Loading models...')
                            : t('Select an API key')
                        }
                      >
                        {selectedApiKey
                          ? apiKeySelectLabel(selectedApiKey)
                          : null}
                      </SelectValue>
                    </SelectTrigger>
                    <SelectContent>
                      {keys.length === 0 ? (
                        <div className='text-muted-foreground px-2 py-1.5 text-sm'>
                          {t(
                            'No API keys in the allowed groups. Create a key for those groups on the API Keys page.'
                          )}
                        </div>
                      ) : (
                        keys.map((k) => (
                          <SelectItem key={k.id} value={String(k.id)}>
                            {apiKeySelectLabel(k)}
                            {k.key ? ` · ${k.key}` : ''}
                          </SelectItem>
                        ))
                      )}
                    </SelectContent>
                  </Select>
                  {loadingModels && (
                    <p className='text-muted-foreground text-sm'>
                      {t('Loading models...')}
                    </p>
                  )}
                </CardContent>
              </Card>

              <Card>
                <CardHeader className='pb-3'>
                  <CardTitle className='text-base'>{t('Model')}</CardTitle>
                  {selectedProfile && (
                    <CardDescription>
                      {t('Tier')}: {selectedProfile.label} ·{' '}
                      {t('Duration field')}: {durationFieldKey}
                    </CardDescription>
                  )}
                </CardHeader>
                <CardContent>
                  <Select
                    value={safeModelId || null}
                    onValueChange={(v) => setModelId(v ?? '')}
                    disabled={filteredModels.length === 0 || loadingModels}
                  >
                    <SelectTrigger className='w-full'>
                      <SelectValue
                        placeholder={
                          loadingModels
                            ? t('Loading models...')
                            : t('Select an API key first')
                        }
                      />
                    </SelectTrigger>
                    <SelectContent>
                      {filteredModels.map((id) => (
                        <SelectItem key={id} value={id}>
                          {id}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </CardContent>
              </Card>

              <Card>
                <CardHeader className='pb-3'>
                  <CardTitle className='text-base'>
                    {t('Generation mode')}
                  </CardTitle>
                </CardHeader>
                <CardContent className='space-y-4'>
                  <div className='flex flex-wrap gap-2'>
                    {generationTypes.map((gt) => (
                      <Button
                        key={gt.value}
                        type='button'
                        size='sm'
                        variant={
                          generationType === gt.value ? 'default' : 'outline'
                        }
                        className={cn(
                          'h-auto min-h-9 max-w-full whitespace-normal px-3 py-2 text-left leading-snug',
                          generationType === gt.value &&
                            'bg-primary text-primary-foreground'
                        )}
                        onClick={() => {
                          setGenerationType(gt.value)
                          // Keep model selection inside the filtered list for this mode
                          // (e.g. image/reference requires *-ref) in the same click.
                          if (selectedProfile) {
                            const next = filterModelsForProfile(
                              availableModels,
                              selectedProfile,
                              Boolean(gt.require_ref_model)
                            )
                            if (
                              next.length > 0 &&
                              (!modelId || !next.includes(modelId))
                            ) {
                              setModelId(next[0])
                            }
                          }
                        }}
                        disabled={!selectedProfile}
                      >
                        {generationTypeDisplayLabel(gt, t)}
                      </Button>
                    ))}
                  </div>
                  {selectedGenType?.require_ref_model && (
                    <p className='text-muted-foreground text-sm'>
                      {t(
                        'This mode requires a model whose name contains -ref.'
                      )}
                    </p>
                  )}

                  <div className='space-y-2'>
                    <Label>{t('Prompt')}</Label>
                    <Textarea
                      value={prompt}
                      onChange={(e) => setPrompt(e.target.value)}
                      rows={4}
                      placeholder={t('Describe the video you want to generate')}
                    />
                  </div>

                  <div className='grid gap-4 sm:grid-cols-2'>
                    <div className='space-y-2'>
                      <Label>
                        {durationFieldKey === 'duration'
                          ? t('Duration (seconds)')
                          : t('Seconds')}
                      </Label>
                      <Select
                        value={durationValue || null}
                        onValueChange={(v) => setDurationValue(v ?? '')}
                      >
                        <SelectTrigger className='w-full'>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {durationOptions.map((d) => (
                            <SelectItem key={d.value} value={d.value}>
                              {d.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                    <div className='space-y-2'>
                      <Label>{t('Aspect ratio')}</Label>
                      <Select
                        value={aspectRatio || null}
                        onValueChange={(v) => setAspectRatio(v ?? '')}
                      >
                        <SelectTrigger className='w-full'>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {aspectOptions.map((a) => (
                            <SelectItem key={a.value} value={a.value}>
                              {a.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  </div>

                  {(selectedGenType?.images_max ?? 0) > 0 && (
                    <div className='space-y-2'>
                      <Label>
                        {t('Reference images')} ({selectedGenType?.images_min}-
                        {selectedGenType?.images_max})
                      </Label>
                      <ReferenceImageGrid
                        items={referenceImages}
                        onChange={setReferenceImages}
                        min={selectedGenType?.images_min ?? 0}
                        max={selectedGenType?.images_max ?? 1}
                        disabled={submitting || isPolling}
                      />
                    </div>
                  )}

                  {selectedGenType?.allow_audio && (
                    <div className='space-y-2'>
                      <Label>
                        {selectedGenType.require_audio
                          ? t('Reference audio (required, MP3)')
                          : t('Reference audio (optional, MP3)')}
                      </Label>
                      <Input
                        type='file'
                        accept='audio/mpeg,audio/mp3,.mp3'
                        disabled={submitting || isPolling}
                        onChange={(e) => {
                          const file = e.target.files?.[0] ?? null
                          setAudioPreviewUrl((prev) => {
                            if (prev) URL.revokeObjectURL(prev)
                            return file ? URL.createObjectURL(file) : ''
                          })
                          if (
                            file &&
                            !/audio\/(mpeg|mp3)/i.test(file.type) &&
                            !file.name.toLowerCase().endsWith('.mp3')
                          ) {
                            toast.error(
                              t('Reference audio must be an MP3 file')
                            )
                            e.target.value = ''
                            setAudioFile(null)
                            return
                          }
                          setAudioFile(file)
                        }}
                      />
                      {audioFile && (
                        <div className='flex flex-col gap-2 rounded-md border px-3 py-2 sm:flex-row sm:items-center'>
                          <p className='text-muted-foreground min-w-0 flex-1 truncate text-sm'>
                            {audioFile.name}
                          </p>
                          {audioPreviewUrl && (
                            <audio
                              controls
                              src={audioPreviewUrl}
                              className='h-8 w-full max-w-xs'
                            />
                          )}
                          <Button
                            type='button'
                            size='sm'
                            variant='outline'
                            disabled={submitting || isPolling}
                            onClick={() => {
                              setAudioFile(null)
                              setAudioPreviewUrl((prev) => {
                                if (prev) URL.revokeObjectURL(prev)
                                return ''
                              })
                            }}
                          >
                            {t('Remove')}
                          </Button>
                        </div>
                      )}
                      <p className='text-muted-foreground text-xs'>
                        {selectedGenType.require_audio
                          ? t(
                              'Required. Sent as data:audio/mpeg;base64,… in audio_url. MP3 only. Images are optional.'
                            )
                          : t(
                              'Optional. Sent as data:audio/mpeg;base64,… in audio_url. MP3 only.'
                            )}
                      </p>
                    </div>
                  )}
                </CardContent>
              </Card>
            </div>

            <aside className='flex min-w-0 flex-col gap-4 lg:sticky lg:top-6'>
              <Card>
                <CardHeader className='pb-3'>
                  <CardTitle className='text-base'>
                    {t('Estimated price')}
                  </CardTitle>
                </CardHeader>
                <CardContent className='space-y-4'>
                  <div>
                    <p className='text-lg font-semibold tabular-nums'>
                      {priceEstimate ? priceEstimate.formatted : t('—')}
                    </p>
                    {priceEstimate ? (
                      <p className='text-muted-foreground mt-1 text-sm'>
                        {t(
                          'Based on model unit price × {{seconds}}s × group ratio. Reference only — final charge follows actual billing.',
                          { seconds: priceEstimate.seconds }
                        )}
                      </p>
                    ) : (
                      <p className='text-muted-foreground mt-1 text-sm'>
                        {t(
                          'Unable to estimate for the current selection. Configure model pricing first.'
                        )}
                      </p>
                    )}
                  </div>
                  <div className='flex flex-wrap items-center gap-3'>
                    <Button
                      type='button'
                      size='lg'
                      onClick={() => void handleSubmit()}
                      disabled={
                        submitting || isPolling || !tokenId || !safeModelId
                      }
                    >
                      {submitting || isPolling
                        ? t('Generating...')
                        : t('Generate video')}
                    </Button>
                    <Link
                      to='/usage-logs/$section'
                      params={{ section: 'task' }}
                      className={cn(buttonVariants({ variant: 'outline' }))}
                    >
                      {t('Open Task Logs')}
                    </Link>
                  </div>
                </CardContent>
              </Card>

              {(taskId || pollError || previewUrl) && (
                <Card>
                  <CardHeader className='pb-3'>
                    <CardTitle className='text-base'>{t('Result')}</CardTitle>
                    <CardDescription>
                      {taskId && (
                        <>
                          {t('Task ID')}:{' '}
                          <span className='font-mono'>{taskId}</span>
                          {taskStatus ? ` · ${taskStatus}` : ''}
                        </>
                      )}
                    </CardDescription>
                  </CardHeader>
                  <CardContent className='space-y-3'>
                    {isPolling && (
                      <div className='space-y-1 text-sm'>
                        <p className='text-muted-foreground'>
                          {isVideoStoragePhase(taskProgress)
                            ? t('Storing generated video locally...')
                            : t('Refreshing task status automatically...')}
                        </p>
                        <p>
                          {t('Progress')}:{' '}
                          <span className='font-medium'>
                            {taskProgress || t('Waiting for update')}
                          </span>
                        </p>
                      </div>
                    )}
                    {!isPolling &&
                      taskProgress &&
                      !pollError &&
                      !previewUrl && (
                        <p className='text-sm'>
                          {t('Progress')}:{' '}
                          <span className='font-medium'>{taskProgress}</span>
                        </p>
                      )}
                    {pollError && (
                      <p className='text-destructive text-sm'>{pollError}</p>
                    )}
                    {previewUrl && (
                      <div className='space-y-2'>
                        <video
                          className='bg-muted aspect-video w-full rounded-md'
                          src={previewUrl}
                          controls
                          playsInline
                        />
                        <a
                          href={previewUrl}
                          target='_blank'
                          rel='noopener noreferrer'
                          className='text-muted-foreground text-sm hover:underline'
                        >
                          {t('Download link')}
                        </a>
                      </div>
                    )}
                    {taskId && (
                      <Link
                        to='/usage-logs/$section'
                        params={{ section: 'task' }}
                        search={{ filter: taskId }}
                        className={cn(
                          buttonVariants({ variant: 'link' }),
                          'h-auto p-0'
                        )}
                      >
                        {t('View this task in Task Logs')}
                      </Link>
                    )}
                  </CardContent>
                </Card>
              )}

              <Card>
                <CardHeader className='pb-3'>
                  <CardTitle className='text-base'>
                    {t('Request JSON (auto-generated)')}
                  </CardTitle>
                  <CardDescription>
                    {t(
                      'Preview only — image/audio base64 is shortened here. On submit, full data:…;base64,… payloads are sent.'
                    )}
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <pre className='bg-muted max-h-[28rem] overflow-auto rounded-md p-3 text-xs'>
                    {requestPreview}
                  </pre>
                </CardContent>
              </Card>
            </aside>
          </div>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
