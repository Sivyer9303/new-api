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

const schema = z.object({
  enabled: z.boolean(),
  groupsText: z.string(),
})

type Values = z.infer<typeof schema>

function parseGroups(raw: string): string {
  try {
    const groups = JSON.parse(raw) as unknown
    if (!Array.isArray(groups)) return ''
    return groups
      .filter((group): group is string => typeof group === 'string')
      .map((group) => group.trim())
      .filter(Boolean)
      .join(', ')
  } catch {
    return ''
  }
}

function normalizeGroups(value: string): string[] {
  return [
    ...new Set(
      value
        .split(/[,，\n]/)
        .map((group) => group.trim())
        .filter(Boolean)
    ),
  ]
}

export function VideoGeneralSettingsSection(props: {
  defaultValues: {
    enabled: boolean
    groupsJson: string
  }
}) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      enabled: props.defaultValues.enabled,
      groupsText: parseGroups(props.defaultValues.groupsJson),
    },
  })

  useEffect(() => {
    form.reset({
      enabled: props.defaultValues.enabled,
      groupsText: parseGroups(props.defaultValues.groupsJson),
    })
  }, [form, props.defaultValues])

  async function onSubmit(values: Values) {
    const groups = normalizeGroups(values.groupsText)
    try {
      const groupsResult = await updateOption.mutateAsync({
        key: 'video_setting.video_tool_groups',
        value: JSON.stringify(groups),
      })
      if (!groupsResult.success) return
      const enabledResult = await updateOption.mutateAsync({
        key: 'video_setting.enabled',
        value: values.enabled,
      })
      if (!enabledResult.success) return
      toast.success(t('Settings saved'))
      form.reset({ enabled: values.enabled, groupsText: groups.join(', ') })
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
          <FormField
            control={form.control}
            name='groupsText'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Video tool allowed groups')}</FormLabel>
                <FormControl>
                  <Input
                    placeholder='default, video'
                    {...field}
                    disabled={busy}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Comma-separated group names whose API keys can use the Video Generation tool. Leave empty to allow no keys.'
                  )}
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
