/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useEffect, useMemo } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateVideoProviderOption } from '../hooks/use-update-option'
import {
  COMPAT_VIDEO_PROFILES,
  COMPAT_VIDEO_PROVIDER,
  emptyCompatVideoProfiles,
  parseCompatibleVideoProfiles,
  serializeCompatibleVideoProfiles,
  type CompatVideoSettingsValues,
} from './compatvideo-profiles'

const dialectOptions = [
  { value: 'newapi_generations', label: 'POST /v1/video/generations' },
  { value: 'openai_videos', label: 'POST /v1/videos' },
] as const

function toText(
  values: readonly number[] | readonly string[] | undefined
): string {
  if (!values?.length) return ''
  return values.join(', ')
}

type ProfileOverrideField =
  | 'durations'
  | 'resolutions'
  | 'aspect_ratios'
  | 'dialect'

export function CompatVideoSettingsSection(props: {
  defaultValues: {
    profilesJson: string
  }
}) {
  const { t } = useTranslation()
  const updateProvider = useUpdateVideoProviderOption()

  const defaults = useMemo<CompatVideoSettingsValues>(
    () => parseCompatibleVideoProfiles(props.defaultValues.profilesJson),
    [props.defaultValues.profilesJson]
  )
  const form = useForm<CompatVideoSettingsValues>({
    defaultValues: emptyCompatVideoProfiles(),
  })

  useEffect(() => {
    form.reset(defaults)
  }, [defaults, form])

  async function onSubmit(values: CompatVideoSettingsValues) {
    const payload = serializeCompatibleVideoProfiles(values)
    try {
      const result = await updateProvider.mutateAsync({
        provider: COMPAT_VIDEO_PROVIDER,
        profiles: payload,
      })
      if (!result.success) {
        toast.error(result.message || t('Failed to save settings'))
        return
      }
      toast.success(t('Settings saved'))
    } catch {
      toast.error(t('Failed to save settings'))
    }
  }

  const busy = updateProvider.isPending || form.formState.isSubmitting

  return (
    <SettingsSection title={t('xtoken')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)} autoComplete='off'>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={busy}
            isSaveDisabled={!form.formState.isDirty}
          />
          <p className='text-muted-foreground text-sm'>
            {t(
              'Override capability defaults per built-in model profile. Leave a field empty to keep the built-in default.'
            )}
          </p>
          <div className='space-y-4'>
            {COMPAT_VIDEO_PROFILES.map((profile) => {
              const path = `profiles.${profile.id}` as const
              const fieldSpecs: {
                id: ProfileOverrideField
                label: string
                placeholder: string
              }[] = [
                {
                  id: 'durations',
                  label: t('Durations'),
                  placeholder: toText(profile.defaultDurations),
                },
                {
                  id: 'resolutions',
                  label: t('Resolutions'),
                  placeholder: toText(profile.defaultResolutions),
                },
                {
                  id: 'aspect_ratios',
                  label: t('Aspect ratios'),
                  placeholder: toText(profile.defaultAspectRatios),
                },
              ]
              return (
                <Card key={profile.id}>
                  <CardHeader>
                    <CardTitle>{profile.label}</CardTitle>
                    <CardDescription className='font-mono text-xs'>
                      {profile.id}
                    </CardDescription>
                  </CardHeader>
                  <CardContent className='space-y-4'>
                    {fieldSpecs.map((spec) => (
                      <FormField
                        key={spec.id}
                        control={form.control}
                        name={`${path}.${spec.id}`}
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{spec.label}</FormLabel>
                            <FormControl>
                              <Input
                                placeholder={spec.placeholder}
                                {...field}
                                disabled={busy}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    ))}
                    <FormField
                      control={form.control}
                      name={`${path}.dialect`}
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>{t('Upstream request dialect')}</FormLabel>
                          <Select
                            value={field.value || 'inherit'}
                            onValueChange={(value) =>
                              field.onChange(value === 'inherit' ? '' : value)
                            }
                            disabled={busy}
                          >
                            <FormControl>
                              <SelectTrigger>
                                <SelectValue
                                  placeholder={t('Select an upstream format')}
                                />
                              </SelectTrigger>
                            </FormControl>
                            <SelectContent>
                              <SelectItem value='inherit'>
                                {t('Inherit built-in default')}
                              </SelectItem>
                              {dialectOptions.map((option) => (
                                <SelectItem
                                  key={option.value}
                                  value={option.value}
                                >
                                  {option.label}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </CardContent>
                </Card>
              )
            })}
          </div>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
