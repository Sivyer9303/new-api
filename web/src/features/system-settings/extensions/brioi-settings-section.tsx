/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useMemo } from 'react'
import { useForm, type Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Form } from '@/components/ui/form'

import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateVideoProviderOption } from '../hooks/use-update-option'
import { BrioiProfileEditor } from './brioi-profile-editor'
import {
  createBrioiSettingsSchema,
  parseBrioiProfiles,
  serializeBrioiProfiles,
  type BrioiSettingsValues,
} from './brioi-profile-schemas'

export function BrioiSettingsSection(props: {
  defaultValues: {
    profilesJson: string
  }
}) {
  const { t } = useTranslation()
  const updateProvider = useUpdateVideoProviderOption()
  const formSchema = useMemo(
    () => createBrioiSettingsSchema((key, options) => t(key, options)),
    [t]
  )
  const defaults = useMemo<BrioiSettingsValues>(
    () => ({
      profiles: parseBrioiProfiles(props.defaultValues.profilesJson),
    }),
    [props.defaultValues.profilesJson]
  )
  const form = useForm<BrioiSettingsValues>({
    resolver: zodResolver(formSchema) as Resolver<BrioiSettingsValues>,
    defaultValues: defaults,
  })

  useEffect(() => {
    form.reset(defaults)
  }, [defaults, form])

  async function onSubmit(values: BrioiSettingsValues) {
    try {
      const result = await updateProvider.mutateAsync({
        provider: 'brioi',
        profiles: JSON.parse(
          serializeBrioiProfiles(values.profiles)
        ) as unknown[],
      })
      if (!result.success) {
        toast.error(result.message || t('Failed to save settings'))
        return
      }

      toast.success(t('Settings saved'))
      form.reset({
        profiles: values.profiles,
      })
    } catch {
      toast.error(t('Failed to save settings'))
    }
  }

  const busy = updateProvider.isPending || form.formState.isSubmitting

  return (
    <SettingsSection title={t('Brioi Video')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)} autoComplete='off'>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={busy}
            isSaveDisabled={!form.formState.isDirty}
          />
          <div className='space-y-1'>
            <h3 className='text-sm font-medium'>{t('Model profiles')}</h3>
            <p className='text-muted-foreground text-sm'>
              {t(
                'Brioi profiles match exact upstream model names. Disable supported capabilities as needed; unsupported values cannot be added.'
              )}
            </p>
          </div>
          <BrioiProfileEditor control={form.control} disabled={busy} />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
