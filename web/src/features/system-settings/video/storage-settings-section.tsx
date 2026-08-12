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
import { z } from 'zod'

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

import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import { VIDEO_RETENTION_DAYS } from './compatibility'

const DEFAULT_STORAGE = {
  driver: 'local' as const,
  local_dir: 'data/videos',
  max_retry: 5,
  ingest_node_name: '',
  public_download_base_url: '',
}

const schema = z.object({
  driver: z.literal('local'),
  local_dir: z.string().trim().min(1),
  max_retry: z.coerce.number().int().min(1),
  ingest_node_name: z.string(),
  public_download_base_url: z.union([z.literal(''), z.string().trim().url()]),
})

type Values = z.infer<typeof schema>

function parseStorage(raw: string): Values {
  try {
    const value = JSON.parse(raw) as Partial<Values>
    return {
      driver: 'local',
      local_dir:
        typeof value.local_dir === 'string' && value.local_dir.trim()
          ? value.local_dir
          : DEFAULT_STORAGE.local_dir,
      max_retry:
        typeof value.max_retry === 'number' && value.max_retry >= 1
          ? value.max_retry
          : DEFAULT_STORAGE.max_retry,
      ingest_node_name:
        typeof value.ingest_node_name === 'string'
          ? value.ingest_node_name
          : '',
      public_download_base_url:
        typeof value.public_download_base_url === 'string'
          ? value.public_download_base_url
          : '',
    }
  } catch {
    return { ...DEFAULT_STORAGE }
  }
}

export function VideoStorageSettingsSection(props: { storageJson: string }) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const form = useForm<Values>({
    resolver: zodResolver(schema) as Resolver<Values>,
    defaultValues: parseStorage(props.storageJson),
  })

  useEffect(() => {
    form.reset(parseStorage(props.storageJson))
  }, [form, props.storageJson])

  async function onSubmit(values: Values) {
    try {
      const result = await updateOption.mutateAsync({
        key: 'video_setting.storage',
        value: JSON.stringify({
          driver: 'local',
          local_dir: values.local_dir.trim(),
          max_retry: values.max_retry,
          ingest_node_name: values.ingest_node_name.trim(),
          public_download_base_url: values.public_download_base_url.trim(),
        }),
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
                {t('Video retention is fixed at {{days}} days', {
                  days: VIDEO_RETENTION_DAYS,
                })}
              </p>
              <p className='text-muted-foreground text-sm'>
                {t(
                  'All generated videos are stored locally before delivery and are permanently deleted after seven days. This period cannot be extended.'
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
                <FormControl>
                  <Input {...field} value='local' disabled readOnly />
                </FormControl>
                <FormDescription>
                  {t('Only the local driver is supported.')}
                </FormDescription>
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name='local_dir'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Local directory')}</FormLabel>
                <FormControl>
                  <Input placeholder='data/videos' {...field} disabled={busy} />
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
                  {t('Public origin used to build local video content URLs.')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
