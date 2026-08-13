/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { z } from 'zod'

export const BRIOI_GENERATION_TYPES = [
  'text2video',
  'image2video',
  'multi_image',
  'first_frame',
  'start_end',
] as const

export type BrioiGenerationType = (typeof BRIOI_GENERATION_TYPES)[number]

export type BrioiHardProfile = {
  id: string
  label: string
  model: string
  durations: string[]
  resolutions: string[]
  aspectRatios: string[]
  maxImages: number
}

function integerRange(min: number, max: number): string[] {
  return Array.from({ length: max - min + 1 }, (_, index) =>
    String(min + index)
  )
}

export const BRIOI_HARD_PROFILES: readonly BrioiHardProfile[] = [
  {
    id: 'seedance-2-0-fast',
    label: 'Seedance 2.0 Fast',
    model: 'seedance-2-0-fast',
    durations: integerRange(4, 15),
    resolutions: ['480p', '720p'],
    aspectRatios: ['21:9', '16:9', '4:3', '1:1', '3:4', '9:16'],
    maxImages: 9,
  },
  {
    id: 'seedance-2-0',
    label: 'Seedance 2.0',
    model: 'seedance-2-0',
    durations: integerRange(4, 15),
    resolutions: ['480p', '720p', '1080p', '4K'],
    aspectRatios: ['21:9', '16:9', '4:3', '1:1', '3:4', '9:16'],
    maxImages: 9,
  },
  {
    id: 'seedance-2-5',
    label: 'Seedance 2.5',
    model: 'seedance-2-5',
    durations: integerRange(4, 29),
    resolutions: ['480p', '720p'],
    aspectRatios: ['16:9', '9:16'],
    maxImages: 30,
  },
] as const

export type BrioiProfileForm = {
  id: string
  label: string
  model: string
  enabled: boolean
  durations: string[]
  resolutions: string[]
  aspect_ratios: string[]
  generation_types: BrioiGenerationType[]
  max_images: number
}

export type BrioiSettingsValues = {
  groups_text: string
  profiles: BrioiProfileForm[]
}

type Translate = (
  key: string,
  options?: Record<string, number | string>
) => string

function stringArray(raw: unknown): string[] {
  if (!Array.isArray(raw)) return []
  const values: string[] = []
  for (const item of raw) {
    if (typeof item === 'string' || typeof item === 'number') {
      values.push(String(item))
      continue
    }
    if (!item || typeof item !== 'object') continue
    const option = item as Record<string, unknown>
    if (option.enabled === false) continue
    const value = option.value ?? option.id ?? option.name
    if (typeof value === 'string' || typeof value === 'number') {
      values.push(String(value))
    }
  }
  return [...new Set(values.map((value) => value.trim()).filter(Boolean))]
}

function recordValue(raw: unknown, key: string): unknown {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return undefined
  return (raw as Record<string, unknown>)[key]
}

function configuredValues(
  raw: unknown,
  key: string,
  defaults: readonly string[]
): string[] {
  const value = recordValue(raw, key)
  return value === undefined ? [...defaults] : stringArray(value)
}

function rawProfiles(raw: string | undefined): unknown[] {
  if (!raw?.trim()) return []
  try {
    const value = JSON.parse(raw) as unknown
    return Array.isArray(value) ? value : []
  } catch {
    return []
  }
}

function profileMatches(raw: unknown, hard: BrioiHardProfile): boolean {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return false
  const profile = raw as Record<string, unknown>
  if (profile.id === hard.id || profile.model === hard.model) return true
  return stringArray(profile.exact_models).includes(hard.model)
}

function defaultBrioiProfile(hard: BrioiHardProfile): BrioiProfileForm {
  return {
    id: hard.id,
    label: hard.label,
    model: hard.model,
    enabled: true,
    durations: [...hard.durations],
    resolutions: [...hard.resolutions],
    aspect_ratios: [...hard.aspectRatios],
    generation_types: [...BRIOI_GENERATION_TYPES],
    max_images: hard.maxImages,
  }
}

export function defaultBrioiProfiles(): BrioiProfileForm[] {
  return BRIOI_HARD_PROFILES.map(defaultBrioiProfile)
}

export function parseBrioiProfiles(
  raw: string | undefined
): BrioiProfileForm[] {
  const configured = rawProfiles(raw)
  return BRIOI_HARD_PROFILES.map((hard) => {
    const source = configured.find((item) => profileMatches(item, hard))
    if (!source || typeof source !== 'object' || Array.isArray(source)) {
      return defaultBrioiProfile(hard)
    }
    const profile = source as Record<string, unknown>
    const overrides =
      profile.overrides &&
      typeof profile.overrides === 'object' &&
      !Array.isArray(profile.overrides)
        ? profile.overrides
        : profile
    const media =
      recordValue(overrides, 'media') ??
      recordValue(overrides, 'media_limits') ??
      recordValue(profile, 'media')
    const generationModes =
      recordValue(profile, 'generation_modes') ??
      recordValue(overrides, 'generation_modes')
    const multiImageMode = Array.isArray(generationModes)
      ? generationModes
          .map((mode) =>
            recordValue(mode, 'value') === 'multi_image' ? mode : null
          )
          .find((mode) => mode !== null)
      : null
    const configuredMax =
      recordValue(multiImageMode, 'images_max') ??
      recordValue(media, 'max_items') ??
      recordValue(media, 'max_images') ??
      profile.max_images
    const maxImages =
      typeof configuredMax === 'number' && Number.isInteger(configuredMax)
        ? configuredMax
        : hard.maxImages
    const generationTypes = (
      generationModes === undefined
        ? configuredValues(
            overrides,
            'generation_types',
            BRIOI_GENERATION_TYPES
          )
        : stringArray(generationModes)
    ).filter((value): value is BrioiGenerationType =>
      BRIOI_GENERATION_TYPES.includes(value as BrioiGenerationType)
    )

    return {
      id: hard.id,
      label:
        typeof profile.label === 'string' && profile.label.trim()
          ? profile.label
          : hard.label,
      model: hard.model,
      enabled: profile.enabled !== false,
      durations: configuredValues(overrides, 'durations', hard.durations),
      resolutions: configuredValues(overrides, 'resolutions', hard.resolutions),
      aspect_ratios: configuredValues(
        overrides,
        'aspect_ratios',
        hard.aspectRatios
      ),
      generation_types: generationTypes,
      max_images: maxImages,
    }
  })
}

export function serializeBrioiProfiles(profiles: BrioiProfileForm[]): string {
  const payload = profiles.map((profile) => ({
    model: profile.model,
    label: profile.label,
    enabled: profile.enabled,
    durations: profile.durations.map(Number),
    resolutions: profile.resolutions,
    aspect_ratios: profile.aspect_ratios,
    generation_modes: BRIOI_GENERATION_TYPES.map((value, index) => {
      let imagesMax = 0
      if (value === 'image2video' || value === 'first_frame') imagesMax = 1
      if (value === 'start_end') imagesMax = 2
      if (value === 'multi_image') imagesMax = profile.max_images
      return {
        value,
        enabled: profile.generation_types.includes(value),
        images_max: imagesMax,
        sort: index + 1,
      }
    }),
  }))
  return JSON.stringify(payload)
}

export function createBrioiSettingsSchema(translate: Translate) {
  const profileSchema = z.object({
    id: z.string().min(1),
    label: z.string().min(1),
    model: z.string().min(1),
    enabled: z.boolean(),
    durations: z.array(z.string()),
    resolutions: z.array(z.string()),
    aspect_ratios: z.array(z.string()),
    generation_types: z.array(z.enum(BRIOI_GENERATION_TYPES)),
    max_images: z.coerce.number().int(),
  })

  return z
    .object({
      groups_text: z.string(),
      profiles: z.array(profileSchema).length(BRIOI_HARD_PROFILES.length),
    })
    .superRefine((values, context) => {
      values.profiles.forEach((profile, index) => {
        const hard = BRIOI_HARD_PROFILES.find(
          (candidate) => candidate.id === profile.id
        )
        if (!hard || hard.model !== profile.model) {
          context.addIssue({
            code: z.ZodIssueCode.custom,
            message: translate('Unknown Brioi model profile'),
            path: ['profiles', index, 'id'],
          })
          return
        }
        const optionSets: Array<{
          key: 'durations' | 'resolutions' | 'aspect_ratios'
          allowed: readonly string[]
        }> = [
          { key: 'durations', allowed: hard.durations },
          { key: 'resolutions', allowed: hard.resolutions },
          { key: 'aspect_ratios', allowed: hard.aspectRatios },
        ]
        for (const optionSet of optionSets) {
          const valuesForOption = profile[optionSet.key]
          if (profile.enabled && valuesForOption.length === 0) {
            context.addIssue({
              code: z.ZodIssueCode.custom,
              message: translate('Enable at least one supported option'),
              path: ['profiles', index, optionSet.key],
            })
          }
          if (
            valuesForOption.some((value) => !optionSet.allowed.includes(value))
          ) {
            context.addIssue({
              code: z.ZodIssueCode.custom,
              message: translate(
                'Option is outside the Brioi model hard capabilities'
              ),
              path: ['profiles', index, optionSet.key],
            })
          }
        }
        if (profile.enabled && profile.generation_types.length === 0) {
          context.addIssue({
            code: z.ZodIssueCode.custom,
            message: translate('Enable at least one generation mode'),
            path: ['profiles', index, 'generation_types'],
          })
        }
        if (profile.max_images < 2 || profile.max_images > hard.maxImages) {
          context.addIssue({
            code: z.ZodIssueCode.custom,
            message: translate(
              'Image limit must be between {{min}} and {{max}} for this profile',
              { min: 2, max: hard.maxImages }
            ),
            path: ['profiles', index, 'max_images'],
          })
        }
      })
    })
}
