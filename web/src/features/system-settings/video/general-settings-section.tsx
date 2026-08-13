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
  FormLabel,
} from '@/components/ui/form'
import { Switch } from '@/components/ui/switch'

import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

const schema = z.object({
  enabled: z.boolean(),
})

type Values = z.infer<typeof schema>

export function VideoGeneralSettingsSection(props: {
  defaultValues: {
    enabled: boolean
  }
}) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      enabled: props.defaultValues.enabled,
    },
  })

  useEffect(() => {
    form.reset({
      enabled: props.defaultValues.enabled,
    })
  }, [form, props.defaultValues])

  async function onSubmit(values: Values) {
    try {
      const enabledResult = await updateOption.mutateAsync({
        key: 'video_setting.enabled',
        value: values.enabled,
      })
      if (!enabledResult.success) return
      toast.success(t('Settings saved'))
      form.reset({ enabled: values.enabled })
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
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
