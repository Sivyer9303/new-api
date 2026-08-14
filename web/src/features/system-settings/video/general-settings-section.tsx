/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
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
import { Switch } from '@/components/ui/switch'

import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import {
  MAX_UPLOAD_LIMIT_MB,
  MIN_UPLOAD_LIMIT_MB,
  parseVideoUploadLimitsJson,
  serializeVideoUploadLimits,
  videoUploadLimitsSchema,
} from './upload-limits'

const schema = z.object({
  enabled: z.boolean(),
  upload_limits: videoUploadLimitsSchema,
})

type Values = z.infer<typeof schema>

export function VideoGeneralSettingsSection(props: {
  defaultValues: {
    enabled: boolean
    uploadLimitsJson?: string
  }
}) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      enabled: props.defaultValues.enabled,
      upload_limits: parseVideoUploadLimitsJson(
        props.defaultValues.uploadLimitsJson
      ),
    },
  })

  useEffect(() => {
    form.reset({
      enabled: props.defaultValues.enabled,
      upload_limits: parseVideoUploadLimitsJson(
        props.defaultValues.uploadLimitsJson
      ),
    })
  }, [form, props.defaultValues])

  async function onSubmit(values: Values) {
    try {
      const enabledResult = await updateOption.mutateAsync({
        key: 'video_setting.enabled',
        value: values.enabled,
      })
      if (!enabledResult.success) return
      const limitsResult = await updateOption.mutateAsync({
        key: 'video_setting.upload_limits',
        value: serializeVideoUploadLimits(values.upload_limits),
      })
      if (!limitsResult.success) return
      toast.success(t('Settings saved'))
      form.reset(values)
    } catch {
      toast.error(t('Failed to save settings'))
    }
  }

  const busy = updateOption.isPending || form.formState.isSubmitting

  return (
    <SettingsSection title={t('General Video Settings')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)} autoComplete='off'>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={busy}
            isSaveDisabled={!form.formState.isDirty}
          />
          <FormField
            control={form.control}
            name='enabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Enable video generation')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Allow configured video channels and the Video Generation tool to accept new tasks.'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    disabled={busy}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />
          <div className='grid gap-4 sm:grid-cols-3'>
            <FormField
              control={form.control}
              name='upload_limits.max_image_mb'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Max image upload size (MB)')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={MIN_UPLOAD_LIMIT_MB}
                      max={MAX_UPLOAD_LIMIT_MB}
                      disabled={busy}
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Maximum size for each reference image uploaded in the video tool.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='upload_limits.max_audio_mb'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Max audio upload size (MB)')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={MIN_UPLOAD_LIMIT_MB}
                      max={MAX_UPLOAD_LIMIT_MB}
                      disabled={busy}
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Maximum size for each reference audio file uploaded in the video tool.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='upload_limits.max_video_mb'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Max video upload size (MB)')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={MIN_UPLOAD_LIMIT_MB}
                      max={MAX_UPLOAD_LIMIT_MB}
                      disabled={busy}
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Maximum size for each reference video file uploaded in the video tool.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
