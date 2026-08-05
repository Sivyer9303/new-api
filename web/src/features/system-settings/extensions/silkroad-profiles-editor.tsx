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
import { ChevronDown, Plus, Trash2 } from 'lucide-react'
import type { ReactNode } from 'react'
import {
  useFieldArray,
  useWatch,
  type Control,
  type FieldArrayPath,
  type FieldValues,
  type Path,
} from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { cn } from '@/lib/utils'

import {
  emptyGenerationType,
  emptyOptionItem,
  emptyProfile,
  type ProfileForm,
} from './silkroad-profile-schemas'

type ProfilesFormValues = {
  profiles: ProfileForm[]
}

type SilkRoadProfilesEditorProps<T extends FieldValues & ProfilesFormValues> = {
  control: Control<T>
  disabled?: boolean
}

function FoldSection({
  title,
  description,
  count,
  defaultOpen = false,
  headerRight,
  children,
}: {
  title: string
  description?: string
  count?: number
  defaultOpen?: boolean
  headerRight?: ReactNode
  children: ReactNode
}) {
  return (
    <Collapsible defaultOpen={defaultOpen} className='rounded-xl border'>
      <div className='flex items-center gap-2 px-3'>
        <CollapsibleTrigger className='group flex min-w-0 flex-1 items-center gap-2 py-3 text-left'>
          <ChevronDown className='text-muted-foreground size-4 shrink-0 transition-transform group-data-[panel-open]:rotate-180' />
          <span className='min-w-0 flex-1 truncate text-sm font-medium'>
            {title}
          </span>
          {typeof count === 'number' ? (
            <Badge variant='secondary' className='shrink-0'>
              {count}
            </Badge>
          ) : null}
        </CollapsibleTrigger>
        {headerRight ? <div className='shrink-0 py-2'>{headerRight}</div> : null}
      </div>
      <CollapsibleContent>
        <div className='space-y-3 border-t px-3 py-3'>
          {description ? (
            <p className='text-muted-foreground text-xs'>{description}</p>
          ) : null}
          {children}
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}

function OptionRowsEditor<T extends FieldValues & ProfilesFormValues>({
  control,
  name,
  title,
  description,
  disabled,
  minRows = 0,
  defaultOpen = false,
}: {
  control: Control<T>
  name: FieldArrayPath<T>
  title: string
  description?: string
  disabled?: boolean
  minRows?: number
  defaultOpen?: boolean
}) {
  const { t } = useTranslation()
  const array = useFieldArray({ control, name })

  return (
    <FoldSection
      title={title}
      description={description}
      count={array.fields.length}
      defaultOpen={defaultOpen}
      headerRight={
        <Button
          type='button'
          variant='outline'
          size='sm'
          disabled={disabled}
          onClick={() =>
            array.append(emptyOptionItem(array.fields.length + 1) as never)
          }
        >
          <Plus className='mr-1 h-4 w-4' />
          {t('Add option')}
        </Button>
      }
    >
      <div className='space-y-3'>
        {array.fields.map((field, index) => (
          <div
            key={field.id}
            className='grid gap-3 rounded-lg border p-3 sm:grid-cols-[1.2fr_1fr_1.2fr_auto_auto_auto] sm:items-end'
          >
            <FormField
              control={control}
              name={`${name}.${index}.label` as Path<T>}
              render={({ field: f }) => (
                <FormItem>
                  <FormLabel>{t('Label')}</FormLabel>
                  <FormControl>
                    <Input {...f} disabled={disabled} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={control}
              name={`${name}.${index}.value` as Path<T>}
              render={({ field: f }) => (
                <FormItem>
                  <FormLabel>{t('Value')}</FormLabel>
                  <FormControl>
                    <Input {...f} disabled={disabled} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={control}
              name={`${name}.${index}.upstream_key` as Path<T>}
              render={({ field: f }) => (
                <FormItem>
                  <FormLabel>{t('Upstream key')}</FormLabel>
                  <FormControl>
                    <Input {...f} disabled={disabled} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={control}
              name={`${name}.${index}.sort` as Path<T>}
              render={({ field: f }) => (
                <FormItem>
                  <FormLabel>{t('Sort')}</FormLabel>
                  <FormControl>
                    <Input type='number' {...f} disabled={disabled} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={control}
              name={`${name}.${index}.enabled` as Path<T>}
              render={({ field: f }) => (
                <FormItem className='flex flex-col gap-2'>
                  <FormLabel>{t('Enabled')}</FormLabel>
                  <FormControl>
                    <Switch
                      checked={Boolean(f.value)}
                      onCheckedChange={f.onChange}
                      disabled={disabled}
                    />
                  </FormControl>
                </FormItem>
              )}
            />
            <Button
              type='button'
              variant='ghost'
              size='icon'
              className='text-destructive'
              disabled={disabled || array.fields.length <= minRows}
              onClick={() => array.remove(index)}
              aria-label={t('Remove option')}
            >
              <Trash2 className='h-4 w-4' />
            </Button>
          </div>
        ))}
        {array.fields.length === 0 ? (
          <p className='text-muted-foreground text-xs'>{t('No options yet')}</p>
        ) : null}
      </div>
    </FoldSection>
  )
}

function GenerationTypeCard<T extends FieldValues & ProfilesFormValues>({
  control,
  profileIndex,
  index,
  disabled,
  canRemove,
  onRemove,
}: {
  control: Control<T>
  profileIndex: number
  index: number
  disabled?: boolean
  canRemove: boolean
  onRemove: () => void
}) {
  const { t } = useTranslation()
  const base = `profiles.${profileIndex}.generation_types.${index}`
  const setsName =
    `profiles.${profileIndex}.generation_types.${index}.upstream_sets` as FieldArrayPath<T>
  const sets = useFieldArray({ control, name: setsName })
  const modeLabel = useWatch({
    control,
    name: `${base}.label` as Path<T>,
  })
  const modeValue = useWatch({
    control,
    name: `${base}.value` as Path<T>,
  })
  const title =
    (typeof modeLabel === 'string' && modeLabel.trim()) ||
    (typeof modeValue === 'string' && modeValue.trim()) ||
    t('Untitled mode')

  return (
    <FoldSection
      title={title}
      defaultOpen={false}
      headerRight={
        <Button
          type='button'
          variant='ghost'
          size='icon'
          className='text-destructive'
          disabled={disabled || !canRemove}
          onClick={onRemove}
          aria-label={t('Remove mode')}
        >
          <Trash2 className='h-4 w-4' />
        </Button>
      }
    >
      <div className='grid gap-3 sm:grid-cols-[1.2fr_1fr_auto_auto] sm:items-end'>
        <FormField
          control={control}
          name={`${base}.label` as Path<T>}
          render={({ field: f }) => (
            <FormItem>
              <FormLabel>{t('Label')}</FormLabel>
              <FormControl>
                <Input {...f} disabled={disabled} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={control}
          name={`${base}.value` as Path<T>}
          render={({ field: f }) => (
            <FormItem>
              <FormLabel>{t('Value')}</FormLabel>
              <FormControl>
                <Input {...f} disabled={disabled} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={control}
          name={`${base}.sort` as Path<T>}
          render={({ field: f }) => (
            <FormItem>
              <FormLabel>{t('Sort')}</FormLabel>
              <FormControl>
                <Input type='number' {...f} disabled={disabled} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={control}
          name={`${base}.enabled` as Path<T>}
          render={({ field: f }) => (
            <FormItem className='flex flex-col gap-2'>
              <FormLabel>{t('Enabled')}</FormLabel>
              <FormControl>
                <Switch
                  checked={Boolean(f.value)}
                  onCheckedChange={f.onChange}
                  disabled={disabled}
                />
              </FormControl>
            </FormItem>
          )}
        />
      </div>

      <div className='grid gap-3 sm:grid-cols-3'>
        <FormField
          control={control}
          name={`${base}.require_ref_model` as Path<T>}
          render={({ field: f }) => (
            <FormItem className='flex flex-col gap-2'>
              <FormLabel>{t('Require -ref model')}</FormLabel>
              <FormControl>
                <Switch
                  checked={Boolean(f.value)}
                  onCheckedChange={f.onChange}
                  disabled={disabled}
                />
              </FormControl>
            </FormItem>
          )}
        />
        <FormField
          control={control}
          name={`${base}.media_requirements.images_min` as Path<T>}
          render={({ field: f }) => (
            <FormItem>
              <FormLabel>{t('Images min')}</FormLabel>
              <FormControl>
                <Input type='number' min={0} {...f} disabled={disabled} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={control}
          name={`${base}.media_requirements.images_max` as Path<T>}
          render={({ field: f }) => (
            <FormItem>
              <FormLabel>{t('Images max')}</FormLabel>
              <FormControl>
                <Input type='number' min={0} {...f} disabled={disabled} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
      </div>

      <div className='space-y-2'>
        <div className='flex items-center justify-between gap-3'>
          <h5 className='text-muted-foreground text-xs font-medium'>
            {t('Upstream field mappings')}
          </h5>
          <Button
            type='button'
            variant='outline'
            size='sm'
            disabled={disabled}
            onClick={() =>
              sets.append({
                upstream_key: '',
                value: '',
                from: '',
              } as never)
            }
          >
            <Plus className='mr-1 h-4 w-4' />
            {t('Add mapping')}
          </Button>
        </div>
        {sets.fields.map((setField, setIndex) => (
          <div
            key={setField.id}
            className='grid gap-3 rounded-lg border border-dashed p-3 sm:grid-cols-[1.2fr_1fr_1fr_auto] sm:items-end'
          >
            <FormField
              control={control}
              name={`${setsName}.${setIndex}.upstream_key` as Path<T>}
              render={({ field: f }) => (
                <FormItem>
                  <FormLabel>{t('Upstream key')}</FormLabel>
                  <FormControl>
                    <Input {...f} disabled={disabled} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={control}
              name={`${setsName}.${setIndex}.from` as Path<T>}
              render={({ field: f }) => (
                <FormItem>
                  <FormLabel>{t('From')}</FormLabel>
                  <FormControl>
                    <Input
                      placeholder='images[0]'
                      {...f}
                      disabled={disabled}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={control}
              name={`${setsName}.${setIndex}.value` as Path<T>}
              render={({ field: f }) => (
                <FormItem>
                  <FormLabel>{t('Fixed value')}</FormLabel>
                  <FormControl>
                    <Input placeholder='auto' {...f} disabled={disabled} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <Button
              type='button'
              variant='ghost'
              size='icon'
              className='text-destructive'
              disabled={disabled}
              onClick={() => sets.remove(setIndex)}
              aria-label={t('Remove mapping')}
            >
              <Trash2 className='h-4 w-4' />
            </Button>
          </div>
        ))}
      </div>
    </FoldSection>
  )
}

function GenerationTypesEditor<T extends FieldValues & ProfilesFormValues>({
  control,
  profileIndex,
  disabled,
}: {
  control: Control<T>
  profileIndex: number
  disabled?: boolean
}) {
  const { t } = useTranslation()
  const name = `profiles.${profileIndex}.generation_types` as FieldArrayPath<T>
  const array = useFieldArray({ control, name })

  return (
    <FoldSection
      title={t('Generation modes')}
      description={t(
        'Each mode maps friendly fields to upstream request keys and image limits.'
      )}
      count={array.fields.length}
      defaultOpen={false}
      headerRight={
        <Button
          type='button'
          variant='outline'
          size='sm'
          disabled={disabled}
          onClick={() =>
            array.append(
              emptyGenerationType(array.fields.length + 1) as never
            )
          }
        >
          <Plus className='mr-1 h-4 w-4' />
          {t('Add mode')}
        </Button>
      }
    >
      <div className='space-y-3'>
        {array.fields.map((field, index) => (
          <GenerationTypeCard
            key={field.id}
            control={control}
            profileIndex={profileIndex}
            index={index}
            disabled={disabled}
            canRemove={array.fields.length > 1}
            onRemove={() => array.remove(index)}
          />
        ))}
      </div>
    </FoldSection>
  )
}

export function SilkRoadProfilesEditor<
  T extends FieldValues & ProfilesFormValues,
>({ control, disabled }: SilkRoadProfilesEditorProps<T>) {
  const { t } = useTranslation()
  const profiles = useFieldArray({
    control,
    name: 'profiles' as FieldArrayPath<T>,
  })

  return (
    <div className='space-y-4'>
      <div className='flex items-center justify-between gap-3'>
        <div>
          <h3 className='text-sm font-medium'>{t('Model profiles')}</h3>
          <p className='text-muted-foreground text-xs'>
            {t(
              'Configure durations, aspect ratios, generation modes, and extras per model family. No raw JSON required.'
            )}
          </p>
        </div>
        <Button
          type='button'
          variant='outline'
          size='sm'
          disabled={disabled}
          onClick={() =>
            profiles.append(emptyProfile(profiles.fields.length) as never)
          }
        >
          <Plus className='mr-1 h-4 w-4' />
          {t('Add profile')}
        </Button>
      </div>

      <div className='space-y-3'>
        {profiles.fields.map((field, index) => (
          <ProfileCard
            key={field.id}
            control={control}
            index={index}
            disabled={disabled}
            canRemove={profiles.fields.length > 1}
            defaultOpen={index === 0}
            onRemove={() => profiles.remove(index)}
          />
        ))}
      </div>
    </div>
  )
}

function ProfileCard<T extends FieldValues & ProfilesFormValues>({
  control,
  index,
  disabled,
  canRemove,
  defaultOpen,
  onRemove,
}: {
  control: Control<T>
  index: number
  disabled?: boolean
  canRemove: boolean
  defaultOpen: boolean
  onRemove: () => void
}) {
  const { t } = useTranslation()
  const label = useWatch({
    control,
    name: `profiles.${index}.label` as Path<T>,
  })
  const id = useWatch({
    control,
    name: `profiles.${index}.id` as Path<T>,
  })
  const title =
    (typeof label === 'string' && label.trim()) ||
    (typeof id === 'string' && id.trim()) ||
    t('Untitled profile')

  return (
    <Collapsible
      defaultOpen={defaultOpen}
      className='bg-card rounded-xl border'
    >
      <div className='flex items-center gap-2 px-4'>
        <CollapsibleTrigger className='group flex min-w-0 flex-1 items-center gap-2 py-3.5 text-left'>
          <ChevronDown
            className={cn(
              'text-muted-foreground size-4 shrink-0 transition-transform',
              'group-data-[panel-open]:rotate-180'
            )}
          />
          <span className='min-w-0 flex-1 truncate text-base font-semibold'>
            {title}
          </span>
        </CollapsibleTrigger>
        <Button
          type='button'
          variant='ghost'
          size='sm'
          className='text-destructive shrink-0'
          disabled={disabled || !canRemove}
          onClick={onRemove}
        >
          <Trash2 className='mr-1 h-4 w-4' />
          {t('Remove profile')}
        </Button>
      </div>

      <CollapsibleContent>
        <div className='space-y-3 border-t px-4 py-4'>
          <FoldSection title={t('Basic info')} defaultOpen>
            <div className='grid gap-4 sm:grid-cols-2'>
              <FormField
                control={control}
                name={`profiles.${index}.id` as Path<T>}
                render={({ field: f }) => (
                  <FormItem>
                    <FormLabel>{t('Profile ID')}</FormLabel>
                    <FormControl>
                      <Input {...f} disabled={disabled} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={control}
                name={`profiles.${index}.label` as Path<T>}
                render={({ field: f }) => (
                  <FormItem>
                    <FormLabel>{t('Profile label')}</FormLabel>
                    <FormControl>
                      <Input {...f} disabled={disabled} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <FormField
              control={control}
              name={`profiles.${index}.model_prefixes_text` as Path<T>}
              render={({ field: f }) => (
                <FormItem>
                  <FormLabel>{t('Model prefixes')}</FormLabel>
                  <FormControl>
                    <Input
                      placeholder='dreamina-seedance-2-0-'
                      {...f}
                      disabled={disabled}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Comma-separated model ID prefixes for this profile.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </FoldSection>

          <OptionRowsEditor
            control={control}
            name={`profiles.${index}.durations` as FieldArrayPath<T>}
            title={t('Durations')}
            description={t(
              'Allowed duration values and the upstream field name (seconds / duration).'
            )}
            disabled={disabled}
            minRows={1}
          />

          <OptionRowsEditor
            control={control}
            name={`profiles.${index}.aspect_ratios` as FieldArrayPath<T>}
            title={t('Aspect ratios')}
            disabled={disabled}
            minRows={1}
          />

          <GenerationTypesEditor
            control={control}
            profileIndex={index}
            disabled={disabled}
          />

          <OptionRowsEditor
            control={control}
            name={`profiles.${index}.extra_options` as FieldArrayPath<T>}
            title={t('Extra options')}
            description={t(
              'Optional toggles such as generate_audio. Leave empty if unused.'
            )}
            disabled={disabled}
            minRows={0}
          />
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}
