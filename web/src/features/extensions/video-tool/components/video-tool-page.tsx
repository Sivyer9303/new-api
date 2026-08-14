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
import { Link } from '@tanstack/react-router'
import { useEffect, useId, useMemo, useRef, useState } from 'react'
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
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { fetchTokenKey } from '@/features/keys/api'
import type { ApiKey } from '@/features/keys/types'
import { usePricingData } from '@/features/pricing/hooks/use-pricing-data'
import { getConfiguredGroupRatio } from '@/features/pricing/lib/model-helpers'
import { formatCurrencyFromUSD } from '@/lib/currency'
import { cn } from '@/lib/utils'

import { fetchVideoModelsForToken, submitVideoGeneration } from '../api'
import { useVideoTaskPolling } from '../hooks/use-video-task-polling'
import { useVideoToolBootstrap } from '../hooks/use-video-tool-bootstrap'
import {
  generationTypeDisableReason,
  generationTypesForProfile,
  resolutionFromModelName,
  retainCompatibleVideoModel,
  resolveProviderVideoProfile,
  resolveSelectedOption,
  type GenerationTypeDisableReason,
} from '../lib/capabilities'
import { estimateVideoPrice } from '../lib/pricing'
import {
  buildPromptMentionOptions,
  mentionToken,
  type PromptMentionKind,
} from '../lib/prompt-mentions'
import { resolveVideoProviderByID } from '../lib/provider-config'
import {
  revokeReferenceImageItems,
  type ReferenceImageItem,
} from '../lib/reference-image'
import { buildVideoGenerationRequest } from '../lib/request'
import type { PublicGenerationType, VideoToolModel } from '../types'
import { PromptMentionEditor } from './prompt-mention-editor'
import { ReferenceImageGrid } from './reference-image-grid'
import { VideoTaskResultCard } from './video-task-result-card'
import { VideoToolStateCard } from './video-tool-state-card'

function generationTypeDisplayLabel(
  gt: PublicGenerationType,
  translate: (key: string) => string
): string {
  switch (gt.value) {
    case 'text2video':
      return translate('Text to video')
    case 'image2video':
    case 'image_reference':
      return translate('Image reference')
    case 'multi_image':
      return translate('Multi-image reference')
    case 'first_frame':
    case 'start_frame':
      return translate('First frame')
    case 'start_end':
    case 'first_last':
    case 'first_last_frame':
      return translate('First & last frame')
    case 'reference_audio':
      return translate('Reference audio')
    case 'reference_videos':
      return translate('Reference video')
    default:
      return gt.label
  }
}

function generationTypeDisableMessage(
  reason: GenerationTypeDisableReason | null,
  translate: (key: string) => string
): string {
  if (reason === 'requires_ref_model') {
    return translate('This mode requires a model whose name contains -ref.')
  }
  if (reason === 'requires_non_ref_model') {
    return translate(
      'This mode requires a model whose name does not contain -ref.'
    )
  }
  return ''
}

function VideoGenerationModeButton(props: {
  generationType: PublicGenerationType
  selected: boolean
  disabled: boolean
  disableReason: GenerationTypeDisableReason | null
  label: string
  disableMessage: string
  onSelect: (value: string) => void
}) {
  const button = (
    <Button
      type='button'
      size='sm'
      variant={props.selected ? 'default' : 'outline'}
      className={cn(
        'h-auto min-h-9 max-w-full whitespace-normal px-3 py-2 text-left leading-snug',
        props.selected && 'bg-primary text-primary-foreground'
      )}
      onClick={() => props.onSelect(props.generationType.value)}
      disabled={props.disabled}
      role='radio'
      aria-checked={props.selected}
      aria-disabled={props.disabled}
    >
      {props.label}
    </Button>
  )
  if (!props.disableReason) return button
  return (
    <Tooltip>
      <TooltipTrigger render={<span className='inline-flex' />}>
        {button}
      </TooltipTrigger>
      <TooltipContent>{props.disableMessage}</TooltipContent>
    </Tooltip>
  )
}

function apiKeySelectLabel(key: ApiKey): string {
  const name = key.name?.trim() || `Key #${key.id}`
  const group = key.group?.trim() || 'default'
  return `${name} (${group})`
}

async function filesToDataUrls(files: File[]): Promise<string[]> {
  if (files.length === 0) return []
  const readers = files.map(
    (file) =>
      new Promise<string>((resolve, reject) => {
        const reader = new FileReader()
        reader.addEventListener('load', () =>
          resolve(String(reader.result || ''))
        )
        reader.addEventListener('error', () =>
          reject(reader.error ?? new Error('read failed'))
        )
        reader.readAsDataURL(file)
      })
  )
  return Promise.all(readers)
}

async function fileToDataURL(file: File): Promise<string> {
  const [url] = await filesToDataUrls([file])
  return url
}

const MAX_REFERENCE_VIDEO_SECONDS = 15

async function readVideoDurationSeconds(file: File): Promise<number> {
  const objectURL = URL.createObjectURL(file)
  try {
    return await new Promise<number>((resolve, reject) => {
      const video = document.createElement('video')
      video.preload = 'metadata'
      video.onloadedmetadata = () => {
        resolve(Number.isFinite(video.duration) ? video.duration : 0)
      }
      video.onerror = () => reject(new Error('Failed to read video metadata'))
      video.src = objectURL
    })
  } finally {
    URL.revokeObjectURL(objectURL)
  }
}

function isMp4VideoFile(file: File): boolean {
  return (
    /video\/mp4/i.test(file.type) || file.name.toLowerCase().endsWith('.mp4')
  )
}

function isBrioiReferenceVideoFile(file: File): boolean {
  const name = file.name.toLowerCase()
  return (
    isMp4VideoFile(file) ||
    /video\/quicktime/i.test(file.type) ||
    name.endsWith('.mov')
  )
}

function isBrioiReferenceAudioDataURL(value: string): boolean {
  const lower = value.toLowerCase()
  return (
    lower.startsWith('data:audio/mpeg;base64,') ||
    lower.startsWith('data:audio/mp3;base64,') ||
    lower.startsWith('data:audio/wav;base64,') ||
    lower.startsWith('data:audio/x-wav;base64,') ||
    lower.startsWith('data:audio/wave;base64,')
  )
}

function isBrioiReferenceVideoDataURL(value: string): boolean {
  const lower = value.toLowerCase()
  return (
    lower.startsWith('data:video/mp4;base64,') ||
    lower.startsWith('data:video/quicktime;base64,')
  )
}

function isAllowedReferenceAudioFile(file: File, allowWav: boolean): boolean {
  const name = file.name.toLowerCase()
  if (allowWav) {
    return (
      /audio\/(mpeg|mp3|wav|x-wav|wave)/i.test(file.type) ||
      name.endsWith('.mp3') ||
      name.endsWith('.wav')
    )
  }
  return /audio\/(mpeg|mp3)/i.test(file.type) || name.endsWith('.mp3')
}

function isAllowedReferenceVideoFile(file: File, allowMov: boolean): boolean {
  if (allowMov) return isBrioiReferenceVideoFile(file)
  return isMp4VideoFile(file)
}

type TranslateText = (
  key: string,
  options?: Record<string, number | string>
) => string

function referenceAudioLabel(
  translate: TranslateText,
  requireAudio: boolean,
  isBrioi: boolean
): string {
  if (requireAudio) return translate('Reference audio (required, MP3)')
  if (isBrioi) return translate('Reference audio (optional, MP3/WAV)')
  return translate('Reference audio (optional, MP3)')
}

function referenceAudioHelp(
  translate: TranslateText,
  requireAudio: boolean,
  isBrioi: boolean
): string {
  if (isBrioi) {
    return translate(
      'Optional companion audio. Brioi numbers it as @音频1. MP3 or WAV; cannot be the only reference.'
    )
  }
  if (requireAudio) {
    return translate(
      'Required. Sent as data:audio/mpeg;base64,… in audio_url. MP3 only. Images are optional.'
    )
  }
  return translate(
    'Optional. Sent as data:audio/mpeg;base64,… in audio_url. MP3 only.'
  )
}

function referenceVideoLabel(
  translate: TranslateText,
  requireVideo: boolean,
  isBrioi: boolean,
  min: number,
  max: number
): string {
  if (!requireVideo) return translate('Reference videos (optional, MP4)')
  const key = isBrioi
    ? 'Reference videos (required, MP4/MOV, {{min}}-{{max}}, ≤{{seconds}}s each)'
    : 'Reference videos (required, MP4, {{min}}-{{max}}, ≤{{seconds}}s each)'
  return translate(key, { min, max, seconds: MAX_REFERENCE_VIDEO_SECONDS })
}

export function VideoToolPage() {
  const { t } = useTranslation()
  const controlId = useId()
  const {
    config,
    keys,
    videoToolGroups,
    isLoading: bootstrapLoading,
    error: bootstrapError,
    retry: retryBootstrap,
  } = useVideoToolBootstrap()
  const [tokenId, setTokenId] = useState<string>('')
  const [models, setModels] = useState<VideoToolModel[]>([])
  const [modelProviderId, setModelProviderId] = useState('')
  const [modelPricingGroup, setModelPricingGroup] = useState('')
  const [loadingModels, setLoadingModels] = useState(false)
  const [modelLoadError, setModelLoadError] = useState('')
  const [modelLoadRevision, setModelLoadRevision] = useState(0)
  const [modelId, setModelId] = useState('')
  const [generationType, setGenerationType] = useState('')
  const [prompt, setPrompt] = useState('')
  const [durationValue, setDurationValue] = useState('')
  const [resolution, setResolution] = useState('')
  const [aspectRatio, setAspectRatio] = useState('')
  const [referenceImages, setReferenceImages] = useState<ReferenceImageItem[]>(
    []
  )
  const [audioFile, setAudioFile] = useState<File | null>(null)
  const [audioPreviewUrl, setAudioPreviewUrl] = useState('')
  const [referenceVideoFiles, setReferenceVideoFiles] = useState<File[]>([])
  const [referenceVideoPreviewUrls, setReferenceVideoPreviewUrls] = useState<
    string[]
  >([])
  const [submitting, setSubmitting] = useState(false)
  const {
    taskId,
    taskStatus,
    taskProgress,
    previewUrl,
    pollError,
    pollingPaused,
    pollingTokenKey,
    isPolling,
    reset: resetTaskPolling,
    start: startTaskPolling,
    failSubmission,
    resume: resumeTaskPolling,
  } = useVideoTaskPolling()
  const loadModelsRequestRef = useRef(0)
  const referenceImagesRef = useRef(referenceImages)
  referenceImagesRef.current = referenceImages

  const { models: pricingModels, groupRatio } = usePricingData()

  // Models with a configured positive ModelPrice. Unpriced models are hidden
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

  const selectedApiKey = useMemo(
    () => keys.find((key) => String(key.id) === tokenId) ?? null,
    [keys, tokenId]
  )
  const activeProvider = useMemo(() => {
    if (!config || !selectedApiKey) return null
    return resolveVideoProviderByID(
      config,
      modelProviderId || models[0]?.provider_id
    )
  }, [config, modelProviderId, models, selectedApiKey])

  useEffect(() => {
    if (!tokenId) return
    if (!keys.some((k) => String(k.id) === tokenId)) {
      setTokenId('')
      setModels([])
      setModelProviderId('')
      setModelPricingGroup('')
      setModelId('')
    }
  }, [keys, tokenId])

  // Selecting / changing an API key automatically loads models.
  useEffect(() => {
    if (!tokenId) {
      setModels([])
      setModelProviderId('')
      setModelPricingGroup('')
      setModelId('')
      setLoadingModels(false)
      setModelLoadError('')
      return
    }
    if (!config || !selectedApiKey) {
      setModels([])
      setModelProviderId('')
      setModelPricingGroup('')
      setModelId('')
      setLoadingModels(false)
      setModelLoadError('')
      return
    }

    const requestId = ++loadModelsRequestRef.current
    const abortController = new AbortController()

    const load = async () => {
      setModels([])
      setModelProviderId('')
      setModelPricingGroup('')
      setModelId('')
      setLoadingModels(true)
      setModelLoadError('')
      try {
        const discovery = await fetchVideoModelsForToken(
          Number(tokenId),
          abortController.signal
        )
        if (
          abortController.signal.aborted ||
          requestId !== loadModelsRequestRef.current
        ) {
          return
        }
        const currentConfig = config
        const discoveredProvider = resolveVideoProviderByID(
          currentConfig,
          discovery.provider
        )
        const matched = discovery.models.filter((model) => {
          const modelProvider =
            resolveVideoProviderByID(currentConfig, model.provider_id) ??
            discoveredProvider
          if (!modelProvider) return false
          return (
            resolveProviderVideoProfile(modelProvider, model.profile_model) !==
            null
          )
        })
        let pricingGroup = discovery.group
        if (discovery.group === 'auto') {
          pricingGroup =
            discovery.resolved_groups.length === 1
              ? discovery.resolved_groups[0]
              : ''
        }
        setModels(matched)
        setModelProviderId(discovery.provider || matched[0]?.provider_id || '')
        setModelPricingGroup(pricingGroup)
        if (matched.length === 0) {
          setModelId('')
          toast.error(t('No video models available for this key'))
        }
        // Selection is synced from available priced models via safeModelId.
      } catch (err) {
        if (
          abortController.signal.aborted ||
          requestId !== loadModelsRequestRef.current
        ) {
          return
        }
        setModels([])
        setModelProviderId('')
        setModelPricingGroup('')
        setModelId('')
        const message =
          err instanceof Error ? err.message : t('Failed to load models')
        setModelLoadError(message)
        toast.error(message)
      } finally {
        if (
          !abortController.signal.aborted &&
          requestId === loadModelsRequestRef.current
        ) {
          setLoadingModels(false)
        }
      }
    }

    void load()
    return () => {
      abortController.abort()
    }
  }, [config, modelLoadRevision, selectedApiKey, t, tokenId])

  const selectedModel = useMemo(
    () => models.find((model) => model.id === modelId) ?? null,
    [modelId, models]
  )
  const selectedProfile = useMemo(() => {
    return selectedModel && activeProvider
      ? resolveProviderVideoProfile(activeProvider, selectedModel.profile_model)
      : null
  }, [activeProvider, selectedModel])

  const generationTypes = useMemo(() => {
    if (!activeProvider) return []
    return selectedProfile
      ? generationTypesForProfile(activeProvider, selectedProfile)
      : activeProvider.generation_types
  }, [activeProvider, selectedProfile])

  const selectedGenType = useMemo(() => {
    if (!generationType) return null
    return generationTypes.find((g) => g.value === generationType) ?? null
  }, [generationType, generationTypes])

  const promptMentionOptions = useMemo(() => {
    const labelFor = (kind: PromptMentionKind, index: number) => {
      if (kind === 'image') return t('Image {{index}}', { index })
      if (kind === 'audio') return t('Audio {{index}}', { index })
      return t('Video {{index}}', { index })
    }
    return buildPromptMentionOptions({
      images: referenceImages.map((item) => ({
        previewUrl: item.previewUrl,
        fileName: item.file.name,
      })),
      audio: audioFile ? { fileName: audioFile.name } : null,
      videos: referenceVideoFiles.map((file, index) => ({
        previewUrl: referenceVideoPreviewUrls[index],
        fileName: file.name,
      })),
      dialect: activeProvider?.id === 'brioi' ? 'zh' : 'latin',
      labelFor,
    })
  }, [
    activeProvider?.id,
    audioFile,
    referenceImages,
    referenceVideoFiles,
    referenceVideoPreviewUrls,
    t,
  ])

  const durationOptions = selectedProfile?.durations ?? []
  // Brioi maps shared upstream models (e.g. seedance-2-0) and encodes tier in
  // the local alias; SilkRoad encodes resolution in the model name itself and
  // rejects a separate resolution field.
  const modelEncodedResolution =
    activeProvider?.id === 'brioi'
      ? resolutionFromModelName(selectedModel?.id ?? '')
      : ''
  const resolutionOptions = modelEncodedResolution
    ? []
    : (selectedProfile?.resolutions ?? [])
  const aspectOptions = selectedProfile?.aspect_ratios ?? []
  const effectiveResolution = modelEncodedResolution || resolution

  const durationFieldKey =
    durationOptions.find((d) => d.value === durationValue)?.upstream_key ||
    durationOptions[0]?.upstream_key ||
    'seconds'

  useEffect(() => {
    const modelName = selectedModel?.profile_model || selectedModel?.id || ''
    const enabledModes = generationTypes.filter(
      (candidate) => !generationTypeDisableReason(modelName, candidate)
    )
    if (enabledModes.length === 0) {
      if (generationType) setGenerationType('')
      return
    }
    if (
      generationType &&
      enabledModes.some((candidate) => candidate.value === generationType)
    ) {
      return
    }
    const hadSelection = Boolean(generationType)
    setGenerationType(enabledModes[0].value)
    if (hadSelection) {
      toast.message(
        t(
          'Selected generation mode was cleared because it is not supported by the selected model.'
        )
      )
    }
  }, [generationType, generationTypes, selectedModel, t])

  useEffect(() => {
    if (!selectedProfile) {
      setDurationValue('')
      setResolution('')
      setAspectRatio('')
      return
    }
    const encodedResolution =
      activeProvider?.id === 'brioi'
        ? resolutionFromModelName(selectedModel?.id ?? '')
        : ''
    const nextDuration = resolveSelectedOption(
      durationValue,
      selectedProfile.durations
    )
    const nextResolution = encodedResolution
      ? encodedResolution
      : resolveSelectedOption(resolution, selectedProfile.resolutions)
    const nextAspectRatio = resolveSelectedOption(
      aspectRatio,
      selectedProfile.aspect_ratios
    )
    if (nextDuration !== durationValue) setDurationValue(nextDuration)
    if (nextResolution !== resolution) setResolution(nextResolution)
    if (nextAspectRatio !== aspectRatio) setAspectRatio(nextAspectRatio)
  }, [
    activeProvider?.id,
    selectedProfile,
    selectedModel,
    durationValue,
    resolution,
    aspectRatio,
  ])

  // Trim or clear reference images when the selected mode's image limit changes.
  useEffect(() => {
    const maxImg = selectedGenType?.images_max ?? 0
    if (maxImg <= 0) {
      if (referenceImages.length > 0) {
        revokeReferenceImageItems(referenceImages)
        setReferenceImages([])
        toast.message(
          t(
            'Reference images were cleared because they are not supported by the selected mode.'
          )
        )
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
  }, [selectedGenType?.images_max, selectedGenType?.value, t])

  // Clear reference audio when the mode does not allow it.
  useEffect(() => {
    if (selectedGenType?.allow_audio) return
    if (audioFile) {
      toast.message(
        t(
          'Reference audio was cleared because it is not supported by the selected mode.'
        )
      )
    }
    setAudioFile(null)
    setAudioPreviewUrl('')
  }, [audioFile, selectedGenType?.allow_audio, selectedGenType?.value, t])

  // Clear reference videos when the mode does not allow them.
  useEffect(() => {
    if (selectedGenType?.allow_video) return
    if (referenceVideoFiles.length === 0) return
    toast.message(
      t(
        'Reference videos were cleared because they are not supported by the selected mode.'
      )
    )
    referenceVideoPreviewUrls.forEach((url) => {
      if (url) URL.revokeObjectURL(url)
    })
    setReferenceVideoFiles([])
    setReferenceVideoPreviewUrls([])
  }, [
    referenceVideoFiles.length,
    referenceVideoPreviewUrls,
    selectedGenType?.allow_video,
    selectedGenType?.value,
    t,
  ])

  useEffect(() => {
    return () => {
      revokeReferenceImageItems(referenceImagesRef.current)
    }
  }, [])

  useEffect(() => {
    if (!audioPreviewUrl) return
    return () => URL.revokeObjectURL(audioPreviewUrl)
  }, [audioPreviewUrl])

  const availableModels = useMemo(
    () => models.filter((model) => pricedModelIds.has(model.id)),
    [models, pricedModelIds]
  )

  const noPricedModelsToastKeyRef = useRef('')
  // The key returned video models, but none have ModelPrice configured.
  useEffect(() => {
    if (loadingModels || !tokenId) return
    if (models.length === 0 || availableModels.length > 0) return
    if (pricingModels.length === 0) return
    const toastKey = `${tokenId}:${models.map((model) => model.id).join('\0')}`
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

  // Base UI Select requires its value to remain among the rendered items.
  const safeModelId = useMemo(
    () => retainCompatibleVideoModel(modelId, availableModels),
    [availableModels, modelId]
  )

  useEffect(() => {
    if (!modelId || safeModelId === modelId) return
    setModelId('')
  }, [modelId, safeModelId])

  const priceEstimate = useMemo(() => {
    if (!safeModelId || !durationValue) return null
    const pricing = pricingModels.find((m) => m.model_name === safeModelId)
    if (!pricing || pricing.model_price == null || pricing.model_price <= 0) {
      return null
    }
    const seconds = Number(durationValue)
    if (!Number.isFinite(seconds) || seconds <= 0) return null
    if (!modelPricingGroup) return null
    const ratios = pricing.group_ratio || groupRatio || {}
    const ratio = getConfiguredGroupRatio(ratios, modelPricingGroup)
    const estimate = estimateVideoPrice({
      modelPrice: pricing.model_price,
      billingMode: pricing.billing_mode,
      quotaType: pricing.quota_type,
      durationSeconds: seconds,
      groupRatio: ratio,
    })
    if (!estimate) return null
    let formatted = '-'
    try {
      formatted = formatCurrencyFromUSD(estimate.usd)
    } catch {
      formatted = `$${estimate.usd}`
    }
    return {
      ...estimate,
      group: modelPricingGroup,
      formatted,
    }
  }, [safeModelId, durationValue, pricingModels, modelPricingGroup, groupRatio])

  const requestPreview = useMemo(() => {
    const images =
      (selectedGenType?.images_max ?? 0) > 0
        ? referenceImages.map((item) => {
            const mime = item.file.type || 'image/jpeg'
            return `data:${mime};base64,…(${item.file.name || 'image'})`
          })
        : []
    const audioURL =
      selectedGenType?.allow_audio && audioFile
        ? `data:audio/mpeg;base64,…(${audioFile.name})`
        : undefined
    const videos =
      selectedGenType?.allow_video && referenceVideoFiles.length > 0
        ? referenceVideoFiles.map(
            (file) => `data:video/mp4;base64,…(${file.name})`
          )
        : undefined
    const body = buildVideoGenerationRequest({
      model: safeModelId,
      prompt,
      generationType,
      aspectRatio,
      durationFieldKey,
      durationValue,
      resolution: effectiveResolution,
      images,
      imageRoles: selectedGenType?.image_roles,
      audioURL,
      videos,
      mediaFormat:
        activeProvider?.id === 'silkroad' ? 'legacy' : 'normalized',
    })
    return JSON.stringify(body, null, 2)
  }, [
    safeModelId,
    prompt,
    generationType,
    aspectRatio,
    effectiveResolution,
    durationFieldKey,
    durationValue,
    selectedGenType,
    referenceImages,
    audioFile,
    referenceVideoFiles,
    activeProvider?.id,
  ])

  let priceEstimateDescription = ''
  if (priceEstimate?.billingMode === 'fixed') {
    priceEstimateDescription = t(
      'Based on fixed model price × group ratio. Duration does not change this estimate. Reference only — final charge follows actual billing.'
    )
  } else if (priceEstimate) {
    priceEstimateDescription = t(
      'Based on model unit price × {{seconds}}s × group ratio. Reference only — final charge follows actual billing.',
      { seconds: priceEstimate.seconds }
    )
  }

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
      !aspectRatio ||
      (resolutionOptions.length > 0 && !effectiveResolution)
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
    const minVideos = selectedGenType.videos_min
    const maxVideos = selectedGenType.videos_max
    if (
      selectedGenType.allow_video &&
      (referenceVideoFiles.length < minVideos ||
        referenceVideoFiles.length > maxVideos)
    ) {
      toast.error(
        t('This mode requires {{min}}-{{max}} reference video(s)', {
          min: minVideos,
          max: maxVideos,
        })
      )
      return
    }

    setSubmitting(true)
    resetTaskPolling()
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
        const allowWav = activeProvider?.id === 'brioi'
        if (allowWav) {
          if (!isBrioiReferenceAudioDataURL(audioURL)) {
            throw new Error(t('Reference audio must be an MP3 or WAV file'))
          }
        } else if (
          !audioURL.toLowerCase().startsWith('data:audio/mpeg;base64,') &&
          !audioURL.toLowerCase().startsWith('data:audio/mp3;base64,')
        ) {
          throw new Error(t('Reference audio must be an MP3 file'))
        }
      }
      let videos: string[] = []
      if (selectedGenType.allow_video && referenceVideoFiles.length > 0) {
        const allowMov = activeProvider?.id === 'brioi'
        let totalSeconds = 0
        for (const file of referenceVideoFiles) {
          if (!isAllowedReferenceVideoFile(file, allowMov)) {
            throw new Error(
              allowMov
                ? t('Reference video must be an MP4 or MOV file')
                : t('Reference video must be an MP4 file')
            )
          }
          const duration = await readVideoDurationSeconds(file)
          if (duration > MAX_REFERENCE_VIDEO_SECONDS) {
            throw new Error(
              t('Each reference video must be at most {{max}} seconds', {
                max: MAX_REFERENCE_VIDEO_SECONDS,
              })
            )
          }
          totalSeconds += duration
        }
        if (totalSeconds > MAX_REFERENCE_VIDEO_SECONDS) {
          throw new Error(
            t(
              'Total reference video duration must be at most {{max}} seconds',
              { max: MAX_REFERENCE_VIDEO_SECONDS }
            )
          )
        }
        videos = await filesToDataUrls(referenceVideoFiles)
        for (const video of videos) {
          const ok = allowMov
            ? isBrioiReferenceVideoDataURL(video)
            : video.toLowerCase().startsWith('data:video/mp4;base64,')
          if (!ok) {
            throw new Error(
              allowMov
                ? t('Reference video must be an MP4 or MOV file')
                : t('Reference video must be an MP4 file')
            )
          }
        }
      }

      const body = buildVideoGenerationRequest({
        model: safeModelId,
        prompt: prompt.trim(),
        generationType,
        aspectRatio,
        durationFieldKey,
        durationValue,
        resolution: effectiveResolution,
        images,
        imageRoles: selectedGenType.image_roles,
        audioURL,
        videos,
        mediaFormat:
          activeProvider?.id === 'silkroad' ? 'legacy' : 'normalized',
      })

      const submitRes = await submitVideoGeneration(tokenKey, body)
      const publicId = submitRes.id || submitRes.task_id
      if (!publicId) {
        throw new Error(t('Submit succeeded but no task id returned'))
      }
      startTaskPolling({
        taskId: publicId,
        status: submitRes.status || 'queued',
        tokenKey,
      })
      toast.success(t('Video task submitted'))
    } catch (err) {
      const message =
        err instanceof Error ? err.message : t('Failed to submit video task')
      toast.error(message)
      failSubmission(message)
    } finally {
      setSubmitting(false)
    }
  }

  if (bootstrapLoading) {
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

  if (bootstrapError) {
    return (
      <SectionPageLayout>
        <SectionPageLayout.Title>
          {t('Video Generation')}
        </SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <VideoToolStateCard
            title={bootstrapError.title}
            description={
              bootstrapError.cause instanceof Error
                ? bootstrapError.cause.message
                : bootstrapError.title
            }
            onRetry={retryBootstrap}
          />
        </SectionPageLayout.Content>
      </SectionPageLayout>
    )
  }

  if (!config?.enabled) {
    return (
      <SectionPageLayout>
        <SectionPageLayout.Title>
          {t('Video Generation')}
        </SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <VideoToolStateCard
            title={t('Video generation unavailable')}
            description={t(
              'Video generation is not configured or enabled. Ask an administrator to review Video Configuration.'
            )}
          />
        </SectionPageLayout.Content>
      </SectionPageLayout>
    )
  }

  const allowedGroupsLabel =
    videoToolGroups.length > 0 ? videoToolGroups.join(', ') : t('(none)')
  let modelPlaceholder = t('Select an API key first')
  if (tokenId) {
    modelPlaceholder = t('Select a model')
  }
  if (loadingModels) {
    modelPlaceholder = t('Loading models...')
  }

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
                  <Label htmlFor={`${controlId}-api-key`}>
                    {t('Your API key')}
                  </Label>
                  <Select
                    value={tokenId || null}
                    onValueChange={(v) => setTokenId(v ?? '')}
                    disabled={keys.length === 0}
                  >
                    <SelectTrigger
                      id={`${controlId}-api-key`}
                      className='w-full'
                    >
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
                  {selectedProfile && activeProvider && (
                    <CardDescription>
                      {t('Provider')}: {activeProvider.label} · {t('Profile')}:{' '}
                      {selectedProfile.label} · {t('Duration field')}:{' '}
                      {durationFieldKey}
                    </CardDescription>
                  )}
                </CardHeader>
                <CardContent>
                  <Label htmlFor={`${controlId}-model`} className='sr-only'>
                    {t('Model')}
                  </Label>
                  <Select
                    value={safeModelId || null}
                    onValueChange={(v) => setModelId(v ?? '')}
                    disabled={availableModels.length === 0 || loadingModels}
                  >
                    <SelectTrigger id={`${controlId}-model`} className='w-full'>
                      <SelectValue placeholder={modelPlaceholder} />
                    </SelectTrigger>
                    <SelectContent>
                      {availableModels.map((model) => (
                        <SelectItem key={model.id} value={model.id}>
                          {model.id}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {modelLoadError ? (
                    <Alert variant='destructive' className='mt-3'>
                      <AlertTitle>{t('Failed to load models')}</AlertTitle>
                      <AlertDescription className='space-y-2'>
                        <p>{modelLoadError}</p>
                        <Button
                          type='button'
                          size='sm'
                          variant='outline'
                          onClick={() =>
                            setModelLoadRevision((revision) => revision + 1)
                          }
                        >
                          {t('Retry')}
                        </Button>
                      </AlertDescription>
                    </Alert>
                  ) : null}
                  {!loadingModels &&
                  !modelLoadError &&
                  tokenId &&
                  models.length === 0 ? (
                    <p className='text-muted-foreground mt-2 text-sm'>
                      {t(
                        'No eligible video models are available for this key and provider.'
                      )}
                    </p>
                  ) : null}
                  {!loadingModels &&
                  !modelLoadError &&
                  models.length > 0 &&
                  availableModels.length === 0 ? (
                    <p className='text-muted-foreground mt-2 text-sm'>
                      {t(
                        'No video models with configured pricing are available. Set a model price in Model Pricing first.'
                      )}
                    </p>
                  ) : null}
                </CardContent>
              </Card>

              <Card>
                <CardHeader className='pb-3'>
                  <CardTitle className='text-base'>
                    {t('Generation mode')}
                  </CardTitle>
                </CardHeader>
                <CardContent className='space-y-4'>
                  <TooltipProvider>
                    <div
                      className='flex flex-wrap gap-2'
                      role='radiogroup'
                      aria-label={t('Generation mode')}
                    >
                      {generationTypes.map((gt) => {
                        const modelName =
                          selectedModel?.profile_model ||
                          selectedModel?.id ||
                          ''
                        const disableReason = selectedProfile
                          ? generationTypeDisableReason(modelName, gt)
                          : null
                        return (
                          <VideoGenerationModeButton
                            key={gt.value}
                            generationType={gt}
                            selected={generationType === gt.value}
                            disabled={!selectedProfile || disableReason != null}
                            disableReason={disableReason}
                            disableMessage={generationTypeDisableMessage(
                              disableReason,
                              t
                            )}
                            label={generationTypeDisplayLabel(gt, t)}
                            onSelect={setGenerationType}
                          />
                        )
                      })}
                    </div>
                  </TooltipProvider>
                  {selectedGenType?.require_ref_model && (
                    <p className='text-muted-foreground text-sm'>
                      {t(
                        'This mode requires a model whose name contains -ref.'
                      )}
                    </p>
                  )}

                  <div className='space-y-2'>
                    <Label htmlFor={`${controlId}-prompt`}>{t('Prompt')}</Label>
                    <PromptMentionEditor
                      id={`${controlId}-prompt`}
                      value={prompt}
                      onChange={setPrompt}
                      options={promptMentionOptions}
                      disabled={submitting || isPolling}
                      placeholder={t(
                        'Describe the video you want to generate. Type @ to mention uploaded media.'
                      )}
                      emptyMenuLabel={t(
                        'Upload images, audio, or video first, then type @ to insert.'
                      )}
                      imageGroupLabel={t('Images')}
                      audioGroupLabel={t('Audio')}
                      videoGroupLabel={t('Videos')}
                    />
                    <p className='text-muted-foreground text-xs'>
                      {activeProvider?.id === 'brioi'
                        ? t(
                            'Type @ to insert @图片1 / @视频1 / @音频1. Chips are display-only; the submitted prompt keeps the @ tokens.'
                          )
                        : t(
                            'Type @ to insert @Image1 / @Audio1 / @Video1. Chips are display-only; the submitted prompt keeps the @ tokens.'
                          )}
                    </p>
                  </div>

                  <div
                    className={cn(
                      'grid gap-4',
                      resolutionOptions.length > 0
                        ? 'sm:grid-cols-3'
                        : 'sm:grid-cols-2'
                    )}
                  >
                    <div className='space-y-2'>
                      <Label htmlFor={`${controlId}-duration`}>
                        {durationFieldKey === 'duration'
                          ? t('Duration (seconds)')
                          : t('Seconds')}
                      </Label>
                      <Select
                        value={durationValue || null}
                        onValueChange={(v) => setDurationValue(v ?? '')}
                      >
                        <SelectTrigger
                          id={`${controlId}-duration`}
                          className='w-full'
                        >
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
                    {resolutionOptions.length > 0 ? (
                      <div className='space-y-2'>
                        <Label htmlFor={`${controlId}-resolution`}>
                          {t('Resolution')}
                        </Label>
                        <Select
                          value={resolution || null}
                          onValueChange={(value) => setResolution(value ?? '')}
                        >
                          <SelectTrigger
                            id={`${controlId}-resolution`}
                            className='w-full'
                          >
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            {resolutionOptions.map((option) => (
                              <SelectItem
                                key={option.value}
                                value={option.value}
                              >
                                {option.label}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                    ) : null}
                    <div className='space-y-2'>
                      <Label htmlFor={`${controlId}-aspect-ratio`}>
                        {t('Aspect ratio')}
                      </Label>
                      <Select
                        value={aspectRatio || null}
                        onValueChange={(v) => setAspectRatio(v ?? '')}
                      >
                        <SelectTrigger
                          id={`${controlId}-aspect-ratio`}
                          className='w-full'
                        >
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
                    <div
                      className='space-y-2'
                      role='group'
                      aria-labelledby={`${controlId}-reference-images`}
                    >
                      <Label id={`${controlId}-reference-images`}>
                        {t('Reference images')} ({selectedGenType?.images_min}-
                        {selectedGenType?.images_max})
                      </Label>
                      <ReferenceImageGrid
                        items={referenceImages}
                        onChange={setReferenceImages}
                        min={selectedGenType?.images_min ?? 0}
                        max={selectedGenType?.images_max ?? 1}
                        roles={selectedGenType?.image_roles ?? []}
                        disabled={submitting || isPolling}
                        maxBytes={
                          (config?.upload_limits.max_image_mb ?? 10) *
                          1024 *
                          1024
                        }
                      />
                    </div>
                  )}

                  {selectedGenType?.allow_audio && (
                    <div className='space-y-2'>
                      <Label htmlFor={`${controlId}-reference-audio`}>
                        {referenceAudioLabel(
                          t,
                          selectedGenType.require_audio,
                          activeProvider?.id === 'brioi'
                        )}
                      </Label>
                      <Input
                        id={`${controlId}-reference-audio`}
                        type='file'
                        accept={
                          activeProvider?.id === 'brioi'
                            ? 'audio/mpeg,audio/mp3,audio/wav,audio/x-wav,.mp3,.wav'
                            : 'audio/mpeg,audio/mp3,.mp3'
                        }
                        disabled={submitting || isPolling}
                        onChange={(e) => {
                          const file = e.target.files?.[0] ?? null
                          const allowWav = activeProvider?.id === 'brioi'
                          if (file && !isAllowedReferenceAudioFile(file, allowWav)) {
                            toast.error(
                              allowWav
                                ? t('Reference audio must be an MP3 or WAV file')
                                : t('Reference audio must be an MP3 file')
                            )
                            e.target.value = ''
                            setAudioFile(null)
                            setAudioPreviewUrl('')
                            return
                          }
                          const maxAudioBytes =
                            (config?.upload_limits.max_audio_mb ?? 24) *
                            1024 *
                            1024
                          if (file && file.size > maxAudioBytes) {
                            toast.error(
                              t('Each audio file must be at most {{max}} MB', {
                                max: config?.upload_limits.max_audio_mb ?? 24,
                              })
                            )
                            e.target.value = ''
                            setAudioFile(null)
                            setAudioPreviewUrl('')
                            return
                          }
                          setAudioPreviewUrl(
                            file ? URL.createObjectURL(file) : ''
                          )
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
                              setAudioPreviewUrl('')
                            }}
                          >
                            {t('Remove')}
                          </Button>
                        </div>
                      )}
                      <p className='text-muted-foreground text-xs'>
                        {referenceAudioHelp(
                          t,
                          selectedGenType.require_audio,
                          activeProvider?.id === 'brioi'
                        )}
                      </p>
                    </div>
                  )}

                  {selectedGenType?.allow_video && (
                    <div className='space-y-2'>
                      <Label htmlFor={`${controlId}-reference-videos`}>
                        {referenceVideoLabel(
                          t,
                          selectedGenType.require_video,
                          activeProvider?.id === 'brioi',
                          selectedGenType.videos_min,
                          selectedGenType.videos_max
                        )}
                      </Label>
                      <Input
                        id={`${controlId}-reference-videos`}
                        type='file'
                        accept={
                          activeProvider?.id === 'brioi'
                            ? 'video/mp4,video/quicktime,.mp4,.mov'
                            : 'video/mp4,.mp4'
                        }
                        multiple
                        disabled={submitting || isPolling}
                        onChange={async (e) => {
                          const picked = [...(e.target.files ?? [])]
                          e.target.value = ''
                          if (picked.length === 0) return
                          const maxCount = selectedGenType.videos_max || 3
                          const next = [...referenceVideoFiles, ...picked].slice(
                            0,
                            maxCount
                          )
                          const maxVideoBytes =
                            (config?.upload_limits.max_video_mb ?? 50) *
                            1024 *
                            1024
                          for (const file of next) {
                            const allowMov = activeProvider?.id === 'brioi'
                            if (!isAllowedReferenceVideoFile(file, allowMov)) {
                              toast.error(
                                allowMov
                                  ? t('Reference video must be an MP4 or MOV file')
                                  : t('Reference video must be an MP4 file')
                              )
                              return
                            }
                            if (file.size > maxVideoBytes) {
                              toast.error(
                                t(
                                  'Each video file must be at most {{max}} MB',
                                  {
                                    max:
                                      config?.upload_limits.max_video_mb ?? 50,
                                  }
                                )
                              )
                              return
                            }
                            try {
                              const duration =
                                await readVideoDurationSeconds(file)
                              if (duration > MAX_REFERENCE_VIDEO_SECONDS) {
                                toast.error(
                                  t(
                                    'Each reference video must be at most {{max}} seconds',
                                    { max: MAX_REFERENCE_VIDEO_SECONDS }
                                  )
                                )
                                return
                              }
                            } catch {
                              toast.error(
                                t('Failed to read reference video metadata')
                              )
                              return
                            }
                          }
                          referenceVideoPreviewUrls.forEach((url) => {
                            if (url) URL.revokeObjectURL(url)
                          })
                          setReferenceVideoFiles(next)
                          setReferenceVideoPreviewUrls(
                            next.map((file) => URL.createObjectURL(file))
                          )
                        }}
                      />
                      {referenceVideoFiles.length > 0 && (
                        <div className='space-y-2'>
                          {referenceVideoFiles.map((file, index) => (
                            <div
                              key={`${file.name}-${file.size}-${file.lastModified}`}
                              className='flex items-center gap-2 rounded-md border px-3 py-2'
                            >
                              <p className='text-muted-foreground min-w-0 flex-1 truncate text-sm'>
                                {mentionToken(
                                  'video',
                                  index + 1,
                                  activeProvider?.id === 'brioi' ? 'zh' : 'latin'
                                )}
                                : {file.name}
                              </p>
                              <Button
                                type='button'
                                size='sm'
                                variant='outline'
                                disabled={submitting || isPolling}
                                onClick={() => {
                                  const nextFiles = referenceVideoFiles.filter(
                                    (_, i) => i !== index
                                  )
                                  const nextUrls =
                                    referenceVideoPreviewUrls.filter(
                                      (_, i) => i !== index
                                    )
                                  const removed =
                                    referenceVideoPreviewUrls[index]
                                  if (removed) URL.revokeObjectURL(removed)
                                  setReferenceVideoFiles(nextFiles)
                                  setReferenceVideoPreviewUrls(nextUrls)
                                }}
                              >
                                {t('Remove')}
                              </Button>
                            </div>
                          ))}
                        </div>
                      )}
                      <p className='text-muted-foreground text-xs'>
                        {activeProvider?.id === 'brioi'
                          ? t(
                              'Brioi sends these as ref type=video. Use @视频N / @图片N in the prompt. MP4 or MOV; 1–3 clips; total duration ≤{{seconds}}s.',
                              { seconds: MAX_REFERENCE_VIDEO_SECONDS }
                            )
                          : t(
                              'Sent as reference_videos. Use @Video1 / @Image1 in the prompt. MP4 only; total duration ≤{{seconds}}s.',
                              { seconds: MAX_REFERENCE_VIDEO_SECONDS }
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
                        {priceEstimateDescription}
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
                <VideoTaskResultCard
                  taskId={taskId}
                  taskStatus={taskStatus}
                  taskProgress={taskProgress}
                  previewUrl={previewUrl}
                  pollError={pollError}
                  isPolling={isPolling}
                  pollingPaused={pollingPaused}
                  canResumePolling={Boolean(taskId && pollingTokenKey)}
                  onResumePolling={resumeTaskPolling}
                />
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
