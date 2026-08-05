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
import { zodResolver } from '@hookform/resolvers/zod'
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
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'

import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import { formatJsonForTextarea } from '../models/utils'

const DEFAULT_STORAGE = {
  enabled: true,
  driver: 'local' as const,
  local_dir: 'data/silkroad-videos',
  retention_days: 7,
  max_retry: 5,
  ingest_node_name: '',
  public_download_base_url: '',
}

type StorageValues = {
  enabled: boolean
  driver: 'local'
  local_dir: string
  retention_days: number
  max_retry: number
  ingest_node_name: string
  public_download_base_url: string
}

function parseStorage(raw: string | undefined): StorageValues {
  if (!raw || !raw.trim()) return { ...DEFAULT_STORAGE }
  try {
    const parsed = JSON.parse(raw) as Partial<StorageValues>
    return {
      enabled: Boolean(parsed.enabled ?? DEFAULT_STORAGE.enabled),
      driver: 'local',
      local_dir:
        typeof parsed.local_dir === 'string' && parsed.local_dir.trim()
          ? parsed.local_dir
          : DEFAULT_STORAGE.local_dir,
      retention_days:
        typeof parsed.retention_days === 'number' && parsed.retention_days >= 1
          ? parsed.retention_days
          : DEFAULT_STORAGE.retention_days,
      max_retry:
        typeof parsed.max_retry === 'number' && parsed.max_retry >= 1
          ? parsed.max_retry
          : DEFAULT_STORAGE.max_retry,
      ingest_node_name:
        typeof parsed.ingest_node_name === 'string'
          ? parsed.ingest_node_name
          : '',
      public_download_base_url:
        typeof parsed.public_download_base_url === 'string'
          ? parsed.public_download_base_url
          : '',
    }
  } catch {
    return { ...DEFAULT_STORAGE }
  }
}

function parseProfilesJson(raw: string | undefined): string {
  if (!raw || !raw.trim()) return '[]'
  return formatJsonForTextarea(raw)
}

const schema = z
  .object({
    enabled: z.boolean(),
    driver: z.literal('local'),
    local_dir: z.string(),
    retention_days: z.coerce.number().int().min(1),
    max_retry: z.coerce.number().int().min(1),
    ingest_node_name: z.string(),
    public_download_base_url: z.string(),
    profilesJson: z.string().min(1),
  })
  .superRefine((values, ctx) => {
    if (values.enabled) {
      if (!values.local_dir.trim()) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: 'Local directory is required when storage is enabled',
          path: ['local_dir'],
        })
      }
      if (!values.ingest_node_name.trim()) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: 'Ingest node name is required when storage is enabled',
          path: ['ingest_node_name'],
        })
      }
      if (!values.public_download_base_url.trim()) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message:
            'Public download base URL is required when storage is enabled',
          path: ['public_download_base_url'],
        })
      }
    }

    try {
      const parsed = JSON.parse(values.profilesJson) as unknown
      if (!Array.isArray(parsed)) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: 'Profiles must be a JSON array',
          path: ['profilesJson'],
        })
      } else if (parsed.length === 0) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: 'Profiles cannot be empty',
          path: ['profilesJson'],
        })
      }
    } catch {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: 'Invalid JSON',
        path: ['profilesJson'],
      })
    }
  })

type Values = z.infer<typeof schema>

export function SilkRoadSettingsSection({
  defaultValues,
}: {
  defaultValues: {
    storageJson: string
    profilesJson: string
  }
}) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const storage = parseStorage(defaultValues.storageJson)

  const form = useForm<Values>({
    resolver: zodResolver(schema) as unknown as Resolver<Values>,
    defaultValues: {
      ...storage,
      profilesJson: parseProfilesJson(defaultValues.profilesJson),
    },
  })

  useEffect(() => {
    const next = parseStorage(defaultValues.storageJson)
    form.reset({
      ...next,
      profilesJson: parseProfilesJson(defaultValues.profilesJson),
    })
  }, [defaultValues, form])

  const { isDirty, isSubmitting } = form.formState

  async function onSubmit(values: Values) {
    let profilesNormalized: string
    try {
      profilesNormalized = JSON.stringify(JSON.parse(values.profilesJson))
    } catch {
      toast.error(t('Invalid JSON'))
      return
    }

    const storagePayload = {
      enabled: values.enabled,
      driver: 'local' as const,
      local_dir: values.local_dir.trim(),
      retention_days: values.retention_days,
      max_retry: values.max_retry,
      ingest_node_name: values.ingest_node_name.trim(),
      public_download_base_url: values.public_download_base_url.trim(),
    }

    const updates: Array<{ key: string; value: string }> = [
      {
        key: 'silkroad_setting.storage',
        value: JSON.stringify(storagePayload),
      },
      {
        key: 'silkroad_setting.profiles',
        value: profilesNormalized,
      },
    ]

    try {
      for (const item of updates) {
        await updateOption.mutateAsync(item)
      }
      toast.success(t('Settings saved'))
      form.reset({
        ...values,
        driver: 'local',
        profilesJson: formatJsonForTextarea(profilesNormalized),
      })
    } catch {
      toast.error(t('Failed to save settings'))
    }
  }

  return (
    <SettingsSection title={t('SilkRoad Video')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)} autoComplete='off'>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={isSubmitting || updateOption.isPending}
            isSaveDisabled={!isDirty}
          />

          <FormField
            control={form.control}
            name='enabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Enable SilkRoad video storage')}</FormLabel>
                  <FormDescription>
                    {t(
                      'When enabled, completed NewAPI SilkRoad videos are ingested on the configured node and served from local storage.'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    disabled={updateOption.isPending || isSubmitting}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <div className='space-y-6'>
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
                  <FormMessage />
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
                    <Input
                      placeholder='data/silkroad-videos'
                      {...field}
                      disabled={updateOption.isPending || isSubmitting}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Directory on the ingest node where video files are stored.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <div className='grid gap-4 sm:grid-cols-2'>
              <FormField
                control={form.control}
                name='retention_days'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Retention days')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={1}
                        {...field}
                        disabled={updateOption.isPending || isSubmitting}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Stored videos older than this many days are deleted.')}
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
                    <FormLabel>{t('Max ingest retries')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={1}
                        {...field}
                        disabled={updateOption.isPending || isSubmitting}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Maximum download attempts before marking ingest as failed.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <FormField
              control={form.control}
              name='ingest_node_name'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Ingest node name')}</FormLabel>
                  <FormControl>
                    <Input
                      placeholder='node-1'
                      {...field}
                      disabled={updateOption.isPending || isSubmitting}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Must match NODE_NAME on the node that downloads and stores videos. Required when storage is enabled.'
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
                      disabled={updateOption.isPending || isSubmitting}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Public origin used to build client download URLs. Required when storage is enabled.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='profilesJson'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Model profiles (JSON)')}</FormLabel>
                  <FormControl>
                    <Textarea
                      className='min-h-64 font-mono text-sm'
                      spellCheck={false}
                      {...field}
                      disabled={updateOption.isPending || isSubmitting}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'JSON array of SilkRoad profiles (durations, aspect ratios, generation types). Must be non-empty and valid.'
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
