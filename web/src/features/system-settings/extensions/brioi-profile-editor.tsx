/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useWatch, type Control } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
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

import {
  BRIOI_GENERATION_TYPES,
  BRIOI_HARD_PROFILES,
  type BrioiGenerationType,
  type BrioiSettingsValues,
} from './brioi-profile-schemas'

type CapabilitySwitchesProps = {
  values: readonly string[]
  selected: string[]
  disabled: boolean
  label: string
  describeValue?: (value: string) => string
  onChange: (values: string[]) => void
}

function CapabilitySwitches(props: CapabilitySwitchesProps) {
  return (
    <fieldset className='space-y-2'>
      <legend className='text-sm font-medium'>{props.label}</legend>
      <div className='grid gap-2 sm:grid-cols-2 lg:grid-cols-3'>
        {props.values.map((value) => {
          const checked = props.selected.includes(value)
          return (
            <label
              key={value}
              className='border-border flex items-center justify-between gap-3 rounded-lg border px-3 py-2 text-sm'
            >
              <span>{props.describeValue?.(value) ?? value}</span>
              <Switch
                checked={checked}
                disabled={props.disabled}
                onCheckedChange={(nextChecked) => {
                  if (nextChecked) {
                    props.onChange([...props.selected, value])
                    return
                  }
                  props.onChange(
                    props.selected.filter((selected) => selected !== value)
                  )
                }}
              />
            </label>
          )
        })}
      </div>
    </fieldset>
  )
}

function generationTypeLabel(
  value: BrioiGenerationType,
  translate: (key: string) => string
): string {
  switch (value) {
    case 'text2video':
      return translate('Text to video')
    case 'image2video':
      return translate('Image reference')
    case 'multi_image':
      return translate('Multi-image reference')
    case 'first_frame':
      return translate('First frame')
    case 'start_end':
      return translate('First & last frame')
  }
}

export function BrioiProfileEditor(props: {
  control: Control<BrioiSettingsValues>
  disabled?: boolean
}) {
  const { t } = useTranslation()
  const profiles = useWatch({ control: props.control, name: 'profiles' }) ?? []

  return (
    <div className='space-y-4'>
      {BRIOI_HARD_PROFILES.map((hard, index) => {
        const profile = profiles[index]
        const profileDisabled = Boolean(props.disabled || !profile?.enabled)
        return (
          <Card key={hard.id}>
            <CardHeader>
              <div className='flex items-start justify-between gap-4'>
                <div>
                  <CardTitle>{hard.label}</CardTitle>
                  <CardDescription className='font-mono'>
                    {hard.model}
                  </CardDescription>
                </div>
                <FormField
                  control={props.control}
                  name={`profiles.${index}.enabled`}
                  render={({ field }) => (
                    <FormItem className='flex items-center gap-2'>
                      <FormLabel>{t('Enabled')}</FormLabel>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                          disabled={props.disabled}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />
              </div>
            </CardHeader>
            <CardContent className='space-y-5'>
              <FormField
                control={props.control}
                name={`profiles.${index}.durations`}
                render={({ field }) => (
                  <FormItem>
                    <FormControl>
                      <CapabilitySwitches
                        values={hard.durations}
                        selected={field.value}
                        disabled={profileDisabled}
                        label={t('Durations')}
                        describeValue={(value) =>
                          t('{{seconds}} seconds', { seconds: value })
                        }
                        onChange={field.onChange}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={props.control}
                name={`profiles.${index}.resolutions`}
                render={({ field }) => (
                  <FormItem>
                    <FormControl>
                      <CapabilitySwitches
                        values={hard.resolutions}
                        selected={field.value}
                        disabled={profileDisabled}
                        label={t('Resolutions')}
                        onChange={field.onChange}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={props.control}
                name={`profiles.${index}.aspect_ratios`}
                render={({ field }) => (
                  <FormItem>
                    <FormControl>
                      <CapabilitySwitches
                        values={hard.aspectRatios}
                        selected={field.value}
                        disabled={profileDisabled}
                        label={t('Aspect ratios')}
                        onChange={field.onChange}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={props.control}
                name={`profiles.${index}.generation_types`}
                render={({ field }) => (
                  <FormItem>
                    <FormControl>
                      <CapabilitySwitches
                        values={BRIOI_GENERATION_TYPES}
                        selected={field.value}
                        disabled={profileDisabled}
                        label={t('Generation modes')}
                        describeValue={(value) =>
                          generationTypeLabel(value as BrioiGenerationType, t)
                        }
                        onChange={(values) =>
                          field.onChange(values as BrioiGenerationType[])
                        }
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={props.control}
                name={`profiles.${index}.max_images`}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Maximum multi-image references')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={2}
                        max={hard.maxImages}
                        value={field.value}
                        onChange={(event) =>
                          field.onChange(event.target.valueAsNumber)
                        }
                        disabled={profileDisabled}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'May be lowered for this provider profile, but cannot exceed the Brioi limit of {{max}}.',
                        { max: hard.maxImages }
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </CardContent>
          </Card>
        )
      })}
    </div>
  )
}
