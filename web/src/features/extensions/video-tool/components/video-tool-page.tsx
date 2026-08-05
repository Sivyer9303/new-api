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
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

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
import { SectionPageLayout } from '@/components/layout'
import { getApiKeys, fetchTokenKey } from '@/features/keys/api'
import type { ApiKey } from '@/features/keys/types'
import { cn } from '@/lib/utils'

import {
  fetchModelsWithTokenKey,
  fetchVideoGeneration,
  fetchVideoToolConfig,
  submitVideoGeneration,
} from '../api'
import type { PublicProfile } from '../types'

function matchProfile(
  profiles: PublicProfile[],
  modelId: string
): PublicProfile | null {
  for (const p of profiles) {
    if (p.model_prefixes.some((prefix) => modelId.startsWith(prefix))) {
      return p
    }
  }
  return null
}

function filterModelsForProfile(
  models: string[],
  profile: PublicProfile,
  requireRef: boolean
): string[] {
  const matched = models.filter((id) =>
    profile.model_prefixes.some((prefix) => id.startsWith(prefix))
  )
  if (!requireRef) {
    return matched
  }
  const refs = matched.filter((id) => id.includes('-ref'))
  return refs.length > 0 ? refs : matched
}

function isTerminalSuccess(status: string | undefined): boolean {
  const s = (status || '').toLowerCase()
  return s === 'completed' || s === 'success'
}

function isTerminalFailure(status: string | undefined): boolean {
  const s = (status || '').toLowerCase()
  return s === 'failed' || s === 'failure' || s === 'cancelled'
}

async function filesToDataUrls(files: FileList | null): Promise<string[]> {
  if (!files || files.length === 0) return []
  const readers = Array.from(files).map(
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
  const [imageFiles, setImageFiles] = useState<FileList | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [taskId, setTaskId] = useState('')
  const [taskStatus, setTaskStatus] = useState('')
  const [previewUrl, setPreviewUrl] = useState('')
  const [pollError, setPollError] = useState('')

  const configQuery = useQuery({
    queryKey: ['silkroad-video-tool-config'],
    queryFn: async () => {
      const res = await fetchVideoToolConfig()
      if (!res.success || !res.data) {
        throw new Error(res.message || 'Failed to load video tool config')
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

  const profiles = configQuery.data?.profiles ?? []
  const selectedProfile = useMemo(
    () => (modelId ? matchProfile(profiles, modelId) : null),
    [modelId, profiles]
  )

  const selectedGenType = useMemo(() => {
    if (!selectedProfile || !generationType) return null
    return (
      selectedProfile.generation_types.find((g) => g.value === generationType) ??
      null
    )
  }, [selectedProfile, generationType])

  const durationOptions = selectedProfile?.durations ?? []
  const aspectOptions = selectedProfile?.aspect_ratios ?? []
  const generationTypes = selectedProfile?.generation_types ?? []

  const durationFieldKey =
    durationOptions.find((d) => d.value === durationValue)?.upstream_key ||
    durationOptions[0]?.upstream_key ||
    'seconds'

  useEffect(() => {
    if (!selectedProfile) return
    if (
      !generationType ||
      !selectedProfile.generation_types.some((g) => g.value === generationType)
    ) {
      setGenerationType(selectedProfile.generation_types[0]?.value ?? '')
    }
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
  }, [selectedProfile, generationType, durationValue, aspectRatio])

  const filteredModels = useMemo(() => {
    if (!selectedProfile) {
      return models.filter((id) =>
        profiles.some((p) =>
          p.model_prefixes.some((prefix) => id.startsWith(prefix))
        )
      )
    }
    return filterModelsForProfile(
      models,
      selectedProfile,
      Boolean(selectedGenType?.require_ref_model)
    )
  }, [models, profiles, selectedProfile, selectedGenType])

  useEffect(() => {
    if (filteredModels.length === 0) return
    if (!modelId || !filteredModels.includes(modelId)) {
      setModelId(filteredModels[0])
    }
  }, [filteredModels, modelId])

  const requestPreview = useMemo(() => {
    const body: Record<string, unknown> = {
      model: modelId,
      prompt,
      generation_type: generationType,
      aspect_ratio: aspectRatio,
    }
    if (durationFieldKey === 'duration') {
      body.duration = Number(durationValue)
    } else {
      body.seconds = durationValue
    }
    if ((selectedGenType?.images_max ?? 0) > 0) {
      body.images = ['<selected files will be attached>']
    }
    return JSON.stringify(body, null, 2)
  }, [
    modelId,
    prompt,
    generationType,
    aspectRatio,
    durationFieldKey,
    durationValue,
    selectedGenType,
  ])

  async function handleLoadModels() {
    if (!tokenId) {
      toast.error(t('Please select an API key'))
      return
    }
    setLoadingModels(true)
    try {
      const keyRes = await fetchTokenKey(Number(tokenId))
      if (!keyRes.success || !keyRes.data?.key) {
        throw new Error(keyRes.message || 'Failed to fetch API key')
      }
      const all = await fetchModelsWithTokenKey(keyRes.data.key)
      const matched = all.filter((id) =>
        profiles.some((p) =>
          p.model_prefixes.some((prefix) => id.startsWith(prefix))
        )
      )
      setModels(matched)
      if (matched.length === 0) {
        toast.error(t('No Seedance models available for this key'))
        setModelId('')
      } else {
        setModelId(matched[0])
        toast.success(t('Models loaded'))
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t('Failed to load models'))
    } finally {
      setLoadingModels(false)
    }
  }

  async function handleSubmit() {
    if (!tokenId) {
      toast.error(t('Please select an API key'))
      return
    }
    if (!modelId || !generationType || !prompt.trim() || !durationValue || !aspectRatio) {
      toast.error(t('Please fill in all required fields'))
      return
    }
    if (!selectedProfile || !selectedGenType) {
      toast.error(t('Selected model is not supported by SilkRoad video profiles'))
      return
    }

    const minImg = selectedGenType.images_min
    const maxImg = selectedGenType.images_max
    const fileCount = imageFiles?.length ?? 0
    if (maxImg > 0 && (fileCount < minImg || fileCount > maxImg)) {
      toast.error(
        t('This mode requires {{min}}-{{max}} reference image(s)', {
          min: minImg,
          max: maxImg,
        })
      )
      return
    }

    setSubmitting(true)
    setPollError('')
    setPreviewUrl('')
    setTaskStatus('')
    setTaskId('')
    try {
      const keyRes = await fetchTokenKey(Number(tokenId))
      if (!keyRes.success || !keyRes.data?.key) {
        throw new Error(keyRes.message || 'Failed to fetch API key')
      }
      const tokenKey = keyRes.data.key
      const images = await filesToDataUrls(imageFiles)

      let submitModel = modelId
      if (selectedGenType.require_ref_model && !submitModel.includes('-ref')) {
        const refCandidate = filteredModels.find((id) => id.includes('-ref'))
        if (refCandidate) {
          submitModel = refCandidate
        }
      }

      const body: Record<string, unknown> = {
        model: submitModel,
        prompt: prompt.trim(),
        generation_type: generationType,
        aspect_ratio: aspectRatio,
      }
      if (durationFieldKey === 'duration') {
        body.duration = Number(durationValue)
      } else {
        body.seconds = durationValue
      }
      if (images.length > 0) {
        body.images = images
      }

      const submitRes = await submitVideoGeneration(tokenKey, body)
      const publicId = submitRes.id || submitRes.task_id
      if (!publicId) {
        throw new Error(t('Submit succeeded but no task id returned'))
      }
      setTaskId(publicId)
      setTaskStatus(submitRes.status || 'queued')
      toast.success(t('Video task submitted'))

      // Poll until terminal
      for (let i = 0; i < 120; i++) {
        await new Promise((r) => setTimeout(r, 3000))
        const statusRes = await fetchVideoGeneration(tokenKey, publicId)
        const st = statusRes.status || ''
        setTaskStatus(st)
        if (isTerminalSuccess(st)) {
          // Prefer local proxy path so session cookie auth works in <video>.
          setPreviewUrl(`/v1/videos/${publicId}/content`)
          toast.success(t('Video generation completed'))
          return
        }
        if (isTerminalFailure(st)) {
          const msg =
            statusRes.error?.message || t('Video generation failed')
          setPollError(msg)
          toast.error(msg)
          return
        }
      }
      setPollError(t('Timed out waiting for video. Check Task Logs.'))
      toast.message(t('Timed out waiting for video. Check Task Logs.'))
    } catch (err) {
      const message =
        err instanceof Error ? err.message : t('Failed to submit video task')
      toast.error(message)
      setPollError(message)
    } finally {
      setSubmitting(false)
    }
  }

  if (configQuery.isLoading || keysQuery.isLoading) {
    return (
      <SectionPageLayout>
        <SectionPageLayout.Title>
          {t('Seedance video tool')}
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
          {t('Seedance video tool')}
        </SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <Card>
            <CardHeader>
              <CardTitle>{t('Video generation unavailable')}</CardTitle>
              <CardDescription>
                {t(
                  'SilkRoad video profiles are not configured or enabled. Ask an admin to set up the SilkRoad extension.'
                )}
              </CardDescription>
            </CardHeader>
          </Card>
        </SectionPageLayout.Content>
      </SectionPageLayout>
    )
  }

  const keys = keysQuery.data ?? []

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Seedance video tool')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='mx-auto flex w-full max-w-3xl flex-col gap-4 pb-8'>
          <p className='text-muted-foreground text-sm'>
            {t(
              'Select an API key, load models, choose a mode, and generate video. Results also appear in Task Logs.'
            )}
          </p>
      <Card>
        <CardHeader className='pb-3'>
          <CardTitle className='text-base'>{t('API key')}</CardTitle>
          <CardDescription>
            {t(
              'You must select a key. The key is fetched only for this session request and is not stored in the page.'
            )}
          </CardDescription>
        </CardHeader>
        <CardContent className='flex flex-col gap-3 sm:flex-row sm:items-end'>
          <div className='min-w-0 flex-1 space-y-2'>
            <Label>{t('Your API key')}</Label>
            <Select
              value={tokenId || null}
              onValueChange={(v) => setTokenId(v ?? '')}
            >
              <SelectTrigger className='w-full'>
                <SelectValue placeholder={t('Select an API key')} />
              </SelectTrigger>
              <SelectContent>
                {keys.length === 0 ? (
                  <div className='text-muted-foreground px-2 py-1.5 text-sm'>
                    {t('No enabled API keys. Create one on the API Keys page.')}
                  </div>
                ) : (
                  keys.map((k) => (
                    <SelectItem key={k.id} value={String(k.id)}>
                      {k.name || `Key #${k.id}`} ({k.key})
                    </SelectItem>
                  ))
                )}
              </SelectContent>
            </Select>
          </div>
          <Button
            type='button'
            onClick={() => void handleLoadModels()}
            disabled={!tokenId || loadingModels}
          >
            {loadingModels ? t('Loading...') : t('Load models')}
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className='pb-3'>
          <CardTitle className='text-base'>{t('Model')}</CardTitle>
          {selectedProfile && (
            <CardDescription>
              {t('Tier')}: {selectedProfile.label} · {t('Duration field')}:{' '}
              {durationFieldKey}
            </CardDescription>
          )}
        </CardHeader>
        <CardContent>
          <Select
            value={modelId || null}
            onValueChange={(v) => setModelId(v ?? '')}
            disabled={filteredModels.length === 0}
          >
            <SelectTrigger className='w-full'>
              <SelectValue placeholder={t('Load models first')} />
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
          <CardTitle className='text-base'>{t('Generation mode')}</CardTitle>
        </CardHeader>
        <CardContent className='space-y-4'>
          <div className='flex flex-wrap gap-2'>
            {generationTypes.map((gt) => (
              <Button
                key={gt.value}
                type='button'
                size='sm'
                variant={generationType === gt.value ? 'default' : 'outline'}
                className={cn(
                  generationType === gt.value && 'bg-primary text-primary-foreground'
                )}
                onClick={() => setGenerationType(gt.value)}
                disabled={!selectedProfile}
              >
                {gt.label}
              </Button>
            ))}
          </div>

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
              <Input
                type='file'
                accept='image/*'
                multiple={(selectedGenType?.images_max ?? 1) > 1}
                onChange={(e) => setImageFiles(e.target.files)}
              />
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader className='pb-3'>
          <CardTitle className='text-base'>
            {t('Request JSON (auto-generated)')}
          </CardTitle>
          <CardDescription>
            {t('Built from the form above. Submitted fields follow site validation rules.')}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <pre className='bg-muted max-h-64 overflow-auto rounded-md p-3 text-xs'>
            {requestPreview}
          </pre>
        </CardContent>
      </Card>

      <div className='flex flex-wrap items-center gap-3'>
        <Button
          type='button'
          size='lg'
          onClick={() => void handleSubmit()}
          disabled={submitting || !tokenId || !modelId}
        >
          {submitting ? t('Generating...') : t('Generate video')}
        </Button>
        <Link
          to='/usage-logs/$section'
          params={{ section: 'task' }}
          className={cn(buttonVariants({ variant: 'outline' }))}
        >
          {t('Open Task Logs')}
        </Link>
      </div>

      {(taskId || pollError || previewUrl) && (
        <Card>
          <CardHeader className='pb-3'>
            <CardTitle className='text-base'>{t('Result')}</CardTitle>
            <CardDescription>
              {taskId && (
                <>
                  {t('Task ID')}: <span className='font-mono'>{taskId}</span>
                  {taskStatus ? ` · ${taskStatus}` : ''}
                </>
              )}
            </CardDescription>
          </CardHeader>
          <CardContent className='space-y-3'>
            {pollError && (
              <p className='text-destructive text-sm'>{pollError}</p>
            )}
            {previewUrl && (
              <video
                className='bg-muted aspect-video w-full rounded-md'
                src={previewUrl}
                controls
                playsInline
              />
            )}
            {taskId && (
              <Link
                to='/usage-logs/$section'
                params={{ section: 'task' }}
                search={{ filter: taskId }}
                className={cn(buttonVariants({ variant: 'link' }), 'h-auto p-0')}
              >
                {t('View this task in Task Logs')}
              </Link>
            )}
          </CardContent>
        </Card>
      )}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
