/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { Plus, Trash2 } from 'lucide-react'
import { useEffect, useMemo } from 'react'
import { useFieldArray, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
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
import { useUpdateVideoProviderOption } from '../hooks/use-update-option'
import {
  AISTARSLAB_PROVIDER,
  DEFAULT_AISTARSLAB_RESOLUTIONS,
  emptyAIStarsLabProfiles,
  parseAIStarsLabProfiles,
  serializeAIStarsLabProfiles,
  type AIStarsLabSettingsValues,
} from './aistarslab-profiles'

export function AIStarsLabSettingsSection(props: {
  defaultValues: {
    profilesJson: string
  }
}) {
  const { t } = useTranslation()
  const updateProvider = useUpdateVideoProviderOption()
  const defaults = useMemo<AIStarsLabSettingsValues>(
    () => parseAIStarsLabProfiles(props.defaultValues.profilesJson),
    [props.defaultValues.profilesJson]
  )
  const form = useForm<AIStarsLabSettingsValues>({
    defaultValues: emptyAIStarsLabProfiles(),
  })
  const profiles = useFieldArray({
    control: form.control,
    name: 'profiles',
  })

  useEffect(() => {
    form.reset(defaults)
  }, [defaults, form])

  async function onSubmit(values: AIStarsLabSettingsValues) {
    try {
      const result = await updateProvider.mutateAsync({
        provider: AISTARSLAB_PROVIDER,
        profiles: serializeAIStarsLabProfiles(values),
      })
      if (!result.success) {
        toast.error(result.message || t('Failed to save settings'))
        return
      }
      toast.success(t('Settings saved'))
      form.reset(values)
    } catch {
      toast.error(t('Failed to save settings'))
    }
  }

  const busy = updateProvider.isPending || form.formState.isSubmitting

  return (
    <SettingsSection title={t('AIStarsLab')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)} autoComplete='off'>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={busy}
            isSaveDisabled={!form.formState.isDirty}
          />
          <p className='text-muted-foreground text-sm'>
            {t(
              'Configure resolutions for each public model name you configured on the channel. Unlisted models keep 720p, 1080p, and 1K.'
            )}
          </p>
          <div className='flex justify-end'>
            <Button
              type='button'
              variant='outline'
              size='sm'
              disabled={busy}
              onClick={() =>
                profiles.append({
                  model: '',
                  resolutions: DEFAULT_AISTARSLAB_RESOLUTIONS.join(', '),
                })
              }
            >
              <Plus className='mr-1 h-4 w-4' />
              {t('Add model')}
            </Button>
          </div>
          {profiles.fields.length === 0 ? (
            <p className='text-muted-foreground text-sm'>
              {t('No models configured. Use Add model to get started.')}
            </p>
          ) : (
            <div className='space-y-4'>
              {profiles.fields.map((field, index) => (
                <div
                  key={field.id}
                  className='grid gap-3 rounded-lg border p-3 sm:grid-cols-[1.2fr_1.4fr_auto] sm:items-end'
                >
                  <FormField
                    control={form.control}
                    name={`profiles.${index}.model`}
                    render={({ field: modelField }) => (
                      <FormItem>
                        <FormLabel>{t('Model name')}</FormLabel>
                        <FormControl>
                          <Input
                            placeholder='seedance-2.0-fast'
                            {...modelField}
                            disabled={busy}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name={`profiles.${index}.resolutions`}
                    render={({ field: resolutionField }) => (
                      <FormItem>
                        <FormLabel>{t('Resolutions')}</FormLabel>
                        <FormControl>
                          <Input
                            placeholder={DEFAULT_AISTARSLAB_RESOLUTIONS.join(
                              ', '
                            )}
                            {...resolutionField}
                            disabled={busy}
                          />
                        </FormControl>
                        <FormDescription>
                          {t('Comma-separated, for example 720p, 1080p')}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <Button
                    type='button'
                    variant='ghost'
                    size='icon'
                    className='self-end'
                    disabled={busy}
                    onClick={() => profiles.remove(index)}
                    aria-label={t('Remove')}
                  >
                    <Trash2 className='h-4 w-4' />
                  </Button>
                </div>
              ))}
            </div>
          )}
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
