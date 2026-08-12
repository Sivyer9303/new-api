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
import { useUpdateOption } from '../hooks/use-update-option'
import {
  defaultProfileExists,
  optionItemSchema,
  parseProfilesToForm,
  profileFormSchema,
  profilesFormToApi,
  type OptionItemForm,
} from './silkroad-profile-schemas'
import {
  OptionRowsEditor,
  SilkRoadProfilesEditor,
} from './silkroad-profiles-editor'

const MAX_DURATION_SECONDS = 3600
const ALLOWED_ASPECT_RATIOS = new Set([
  '16:9',
  '9:16',
  '1:1',
  '4:3',
  '3:4',
  '21:9',
])

function createSilkRoadSettingsSchema(
  translate: (key: string, options?: Record<string, number>) => string
) {
  return z
    .object({
      default_profile_id: z.string().min(1),
      common_durations: z.array(optionItemSchema).min(1),
      common_aspect_ratios: z.array(optionItemSchema).min(1),
      profiles: z.array(profileFormSchema).min(1),
    })
    .superRefine((values, context) => {
      const profileIDs = new Set<string>()
      const exactModels = new Set<string>()
      const prefixes = new Set<string>()
      values.profiles.forEach((profile, index) => {
        if (profileIDs.has(profile.id.trim())) {
          context.addIssue({
            code: z.ZodIssueCode.custom,
            message: translate('Profile IDs must be unique'),
            path: ['profiles', index, 'id'],
          })
        }
        profileIDs.add(profile.id.trim())
        for (const exact of splitList(profile.exact_models_text)) {
          if (exactModels.has(exact)) {
            context.addIssue({
              code: z.ZodIssueCode.custom,
              message: translate('Exact model IDs must be unique'),
              path: ['profiles', index, 'exact_models_text'],
            })
          }
          exactModels.add(exact)
        }
        for (const prefix of splitList(profile.model_prefixes_text)) {
          if (prefixes.has(prefix)) {
            context.addIssue({
              code: z.ZodIssueCode.custom,
              message: translate('Model prefixes must be unique'),
              path: ['profiles', index, 'model_prefixes_text'],
            })
          }
          prefixes.add(prefix)
        }
        validateDurations(
          profile.durations,
          ['profiles', index, 'durations'],
          context,
          translate
        )
        validateAspectRatios(
          profile.aspect_ratios,
          ['profiles', index, 'aspect_ratios'],
          context,
          translate
        )
      })
      if (!defaultProfileExists(values.default_profile_id, values.profiles)) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          message: translate(
            'Default profile must reference an existing profile'
          ),
          path: ['default_profile_id'],
        })
      }
      validateDurations(
        values.common_durations,
        ['common_durations'],
        context,
        translate
      )
      validateAspectRatios(
        values.common_aspect_ratios,
        ['common_aspect_ratios'],
        context,
        translate
      )
    })
}

type Values = z.infer<ReturnType<typeof createSilkRoadSettingsSchema>>

function splitList(value: string): string[] {
  return value
    .split(/[,，\n]/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function validateDurations(
  items: OptionItemForm[],
  path: Array<string | number>,
  context: z.RefinementCtx,
  translate: (key: string, options?: Record<string, number>) => string
) {
  items.forEach((item, index) => {
    if (!item.enabled) return
    const seconds = Number(item.value)
    if (
      !Number.isInteger(seconds) ||
      seconds < 1 ||
      seconds > MAX_DURATION_SECONDS
    ) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        message: translate('Duration must be between 1 and {{max}} seconds', {
          max: MAX_DURATION_SECONDS,
        }),
        path: [...path, index, 'value'],
      })
    }
  })
}

function validateAspectRatios(
  items: OptionItemForm[],
  path: Array<string | number>,
  context: z.RefinementCtx,
  translate: (key: string) => string
) {
  items.forEach((item, index) => {
    if (item.enabled && !ALLOWED_ASPECT_RATIOS.has(item.value)) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        message: translate(
          'Aspect ratio is outside SilkRoad hard capabilities'
        ),
        path: [...path, index, 'value'],
      })
    }
  })
}

function parseOptions(raw: string, key: 'durations' | 'aspect_ratios') {
  try {
    const common = JSON.parse(raw) as Record<string, unknown>
    const values = common[key]
    return Array.isArray(values) ? (values as OptionItemForm[]) : []
  } catch {
    return []
  }
}

export function SilkRoadSettingsSection(props: {
  defaultValues: {
    commonJson: string
    profilesJson: string
    defaultProfileID: string
  }
}) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const formSchema = useMemo(
    () => createSilkRoadSettingsSchema((key, options) => t(key, options)),
    [t]
  )
  const defaults = useMemo<Values>(() => {
    const profiles = parseProfilesToForm(props.defaultValues.profilesJson)
    return {
      default_profile_id:
        props.defaultValues.defaultProfileID || profiles[0]?.id || '',
      common_durations: parseOptions(
        props.defaultValues.commonJson,
        'durations'
      ),
      common_aspect_ratios: parseOptions(
        props.defaultValues.commonJson,
        'aspect_ratios'
      ),
      profiles,
    }
  }, [props.defaultValues])
  const form = useForm<Values>({
    resolver: zodResolver(formSchema) as Resolver<Values>,
    defaultValues: defaults,
  })

  useEffect(() => {
    form.reset(defaults)
  }, [defaults, form])

  async function onSubmit(values: Values) {
    const profilePayload = profilesFormToApi(values.profiles)
    const currentProfiles = profilesFormToApi(
      parseProfilesToForm(props.defaultValues.profilesJson)
    )
    const currentProfileIDs = new Set(
      currentProfiles.map((profile) => profile.id)
    )
    const updates: Array<{ key: string; value: string }> = [
      {
        key: 'silkroad_setting.common',
        value: JSON.stringify({
          durations: values.common_durations,
          aspect_ratios: values.common_aspect_ratios,
        }),
      },
    ]
    if (
      values.default_profile_id !== props.defaultValues.defaultProfileID &&
      currentProfileIDs.has(values.default_profile_id)
    ) {
      updates.push({
        key: 'silkroad_setting.default_profile_id',
        value: values.default_profile_id,
      })
      updates.push({
        key: 'silkroad_setting.profiles',
        value: JSON.stringify(profilePayload),
      })
    } else if (
      values.default_profile_id !== props.defaultValues.defaultProfileID
    ) {
      const oldDefault = currentProfiles.find(
        (profile) => profile.id === props.defaultValues.defaultProfileID
      )
      const interimProfiles =
        oldDefault &&
        !profilePayload.some((profile) => profile.id === oldDefault.id)
          ? [...profilePayload, oldDefault]
          : profilePayload
      updates.push({
        key: 'silkroad_setting.profiles',
        value: JSON.stringify(interimProfiles),
      })
      updates.push({
        key: 'silkroad_setting.default_profile_id',
        value: values.default_profile_id,
      })
      if (interimProfiles !== profilePayload) {
        updates.push({
          key: 'silkroad_setting.profiles',
          value: JSON.stringify(profilePayload),
        })
      }
    } else {
      updates.push({
        key: 'silkroad_setting.profiles',
        value: JSON.stringify(profilePayload),
      })
    }
    try {
      for (const update of updates) {
        const result = await updateOption.mutateAsync(update)
        if (!result.success) return
      }
      toast.success(t('Settings saved'))
      form.reset(values)
    } catch {
      toast.error(t('Failed to save settings'))
    }
  }

  const busy = updateOption.isPending || form.formState.isSubmitting
  const profiles = form.watch('profiles')

  return (
    <SettingsSection title={t('SilkRoad Video')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)} autoComplete='off'>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={busy}
            isSaveDisabled={!form.formState.isDirty}
          />
          <FormField
            control={form.control}
            name='default_profile_id'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Default profile')}</FormLabel>
                <Select
                  value={field.value}
                  onValueChange={field.onChange}
                  disabled={busy}
                >
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue
                        placeholder={t('Select a default profile')}
                      />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    {profiles.map((profile) => (
                      <SelectItem key={profile.id} value={profile.id}>
                        {profile.label || profile.id}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <FormDescription>
                  {t(
                    'Models without an exact or prefix match use this profile. Review fallback diagnostics before saving.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <div className='space-y-4'>
            <div>
              <h3 className='text-sm font-medium'>
                {t('Common capabilities')}
              </h3>
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Profiles inherit these options unless they define their own overrides.'
                )}
              </p>
            </div>
            <OptionRowsEditor
              control={form.control}
              name='common_durations'
              title={t('Common durations')}
              disabled={busy}
              minRows={1}
              defaultOpen
            />
            <OptionRowsEditor
              control={form.control}
              name='common_aspect_ratios'
              title={t('Common aspect ratios')}
              disabled={busy}
              minRows={1}
            />
          </div>
          <SilkRoadProfilesEditor control={form.control} disabled={busy} />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
