/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { zodResolver } from '@hookform/resolvers/zod'
import { Info } from 'lucide-react'
import { useEffect } from 'react'
import { useForm, type Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import { R2UsageCard } from './r2-usage-card'
import {
  activeRetentionDays,
  parseVideoStorage,
  serializeVideoStorage,
  videoStorageSchema,
  type VideoStorageValues,
} from './storage-config'

export function VideoStorageSettingsSection(props: { storageJson: string }) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const form = useForm<VideoStorageValues>({
    resolver: zodResolver(videoStorageSchema) as Resolver<VideoStorageValues>,
    defaultValues: parseVideoStorage(props.storageJson),
  })

  useEffect(() => {
    form.reset(parseVideoStorage(props.storageJson))
  }, [form, props.storageJson])

  const driver = form.watch('driver')
  const isR2 = driver === 'r2'

  async function onSubmit(values: VideoStorageValues) {
    try {
      const result = await updateOption.mutateAsync({
        key: 'video_setting.storage',
        value: serializeVideoStorage(values),
      })
      if (!result.success) return
      toast.success(t('Settings saved'))
      form.reset(values)
    } catch {
      toast.error(t('Failed to save settings'))
    }
  }

  const busy = updateOption.isPending || form.formState.isSubmitting

  return (
    <SettingsSection title={t('Video Storage')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)} autoComplete='off'>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={busy}
            isSaveDisabled={!form.formState.isDirty}
          />
          <div
            className='border-border bg-muted/40 flex gap-3 rounded-lg border p-4'
            role='note'
          >
            <Info
              className='text-muted-foreground mt-0.5 size-4 shrink-0'
              aria-hidden='true'
            />
            <div className='space-y-1'>
              <p className='text-sm font-medium'>
                {t('Stored videos expire after {{days}} days', {
                  days: activeRetentionDays(form.getValues()),
                })}
              </p>
              <p className='text-muted-foreground text-sm'>
                {t(
                  'Every generated video is copied into the configured storage before delivery, and the upstream address is never exposed to users. Each driver keeps its own retention period.'
                )}
              </p>
            </div>
          </div>

          <FormField
            control={form.control}
            name='driver'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Storage driver')}</FormLabel>
                <Select
                  items={[
                    { value: 'local', label: t('Local disk') },
                    { value: 'r2', label: t('Cloudflare R2') },
                  ]}
                  value={field.value}
                  onValueChange={field.onChange}
                  disabled={busy}
                >
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent alignItemWithTrigger={false}>
                    <SelectGroup>
                      <SelectItem value='local'>{t('Local disk')}</SelectItem>
                      <SelectItem value='r2'>{t('Cloudflare R2')}</SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
                <FormDescription>
                  {t(
                    'Local disk streams videos through this application. Cloudflare R2 stores them in a private bucket and redirects viewers to a short-lived signed URL.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='public_download_base_url'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Public download base URL')}</FormLabel>
                <FormControl>
                  <Input
                    placeholder='https://video.example.com'
                    {...field}
                    disabled={busy}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Public origin of this site, used to build the video content URL returned to clients.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='max_retry'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Max storage retries')}</FormLabel>
                <FormControl>
                  <Input type='number' min={1} {...field} disabled={busy} />
                </FormControl>
                <FormDescription>
                  {t(
                    'Maximum automatic transfer attempts before administrator recovery is required.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          {isR2 ? null : (
            <>
              <FormField
                control={form.control}
                name='local_dir'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Local directory')}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder='data/videos'
                        {...field}
                        disabled={busy}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Directory on the ingest node where all provider video files are stored.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='ingest_node_name'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Ingest node name')}</FormLabel>
                    <FormControl>
                      <Input placeholder='node-1' {...field} disabled={busy} />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Must match NODE_NAME on the node that downloads and stores videos.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='local_retention_days'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Local retention (days)')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={1}
                        max={30}
                        {...field}
                        disabled={busy}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Days a locally stored video stays playable before it is deleted. Between 1 and 30.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </>
          )}

          {isR2 ? (
            <>
              <R2UsageCard />
              <FormField
                control={form.control}
                name='r2.account_id'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('R2 account ID')}</FormLabel>
                    <FormControl>
                      <Input {...field} disabled={busy} />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Cloudflare account identifier. Used to derive the S3 endpoint and to read bucket usage.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='r2.bucket'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('R2 bucket name')}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder='new-api-videos'
                        {...field}
                        disabled={busy}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Keep this bucket private. Viewers only ever receive signed URLs.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='r2.access_key_id'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('R2 access key ID')}</FormLabel>
                    <FormControl>
                      <Input autoComplete='off' {...field} disabled={busy} />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'S3-compatible access key used to upload and sign objects.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='r2.secret_access_key'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('R2 secret access key')}</FormLabel>
                    <FormControl>
                      <Input
                        type='password'
                        autoComplete='new-password'
                        {...field}
                        disabled={busy}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Stored securely and never shown to clients. Re-enter it if you rotate the key.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='r2.api_token'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Cloudflare API token')}</FormLabel>
                    <FormControl>
                      <Input
                        type='password'
                        autoComplete='new-password'
                        {...field}
                        disabled={busy}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Token with R2 read permission. Used once per hour to read bucket usage so uploads can stop before the free tier is exhausted.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='r2.endpoint'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('R2 S3 endpoint')}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder='https://<account_id>.r2.cloudflarestorage.com'
                        {...field}
                        disabled={busy}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Leave empty to derive it from the account ID. Set it only for jurisdiction-specific endpoints.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='r2.region'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('R2 region')}</FormLabel>
                    <FormControl>
                      <Input placeholder='auto' {...field} disabled={busy} />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'R2 uses the auto region unless Cloudflare tells you otherwise.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='r2.result_prefix'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Result object prefix')}</FormLabel>
                    <FormControl>
                      <Input placeholder='videos/' {...field} disabled={busy} />
                    </FormControl>
                    <FormDescription>
                      {t('Key prefix for finished videos delivered to users.')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='r2.input_prefix'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Input object prefix')}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder='video-inputs/'
                        {...field}
                        disabled={busy}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Key prefix for reference images and audio staged for upstreams that cannot accept base64. Must differ from the result prefix.'
                      )}
                    </FormDescription>
                    <FormDescription>
                      {t(
                        'Browser direct upload requires R2 bucket CORS allowing PUT and OPTIONS from your dashboard origin(s).'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='r2.retention_days'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('R2 retention (days)')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={1}
                        max={30}
                        {...field}
                        disabled={busy}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Days a video stays in R2 before it is deleted. Shorter retention keeps you inside the free tier.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='r2.result_presign_ttl_seconds'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('Playback link lifetime (seconds)')}
                    </FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={60}
                        {...field}
                        disabled={busy}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'How long a signed playback URL stays valid. Viewers simply request the video again to get a fresh link.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='r2.input_presign_ttl_seconds'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('Upstream input link lifetime (seconds)')}
                    </FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={60}
                        {...field}
                        disabled={busy}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'How long the signed URL handed to the upstream provider stays valid. It must outlive the provider fetch.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='r2.input_ttl_hours'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Staged input retention (hours)')}</FormLabel>
                    <FormControl>
                      <Input type='number' min={1} {...field} disabled={busy} />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Staged reference media older than this is deleted by the hourly cleanup. Only the input prefix is affected.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </>
          ) : null}
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
