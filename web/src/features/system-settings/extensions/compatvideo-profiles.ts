/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

// Built-in Compatible Video profiles and their default capabilities. Mirrored
// from setting/compatvideo_setting/profiles.go (builtInProfiles) so the admin
// page can show each profile's inherited defaults as placeholders. Keep these
// in sync when builtInProfiles changes.
export const COMPAT_VIDEO_PROFILES = [
  {
    id: 'seedance2',
    label: 'Seedance 2',
    dialect: 'openai_videos',
    defaultDurations: [4, 6, 8, 10, 12, 15],
    defaultResolutions: ['480p', '720p'],
    defaultAspectRatios: ['auto', '21:9', '16:9', '4:3', '1:1', '3:4', '9:16'],
  },
  {
    id: 'grok-image-video',
    label: 'Grok Image Video',
    dialect: 'newapi_generations',
    defaultDurations: [4, 6, 8, 10, 12, 15],
    defaultResolutions: ['480p', '720p'],
    defaultAspectRatios: ['1:1', '16:9', '9:16', '4:3', '3:4', '3:2', '2:3'],
  },
  {
    id: 'grok-video-1.5',
    label: 'Grok Video 1.5',
    dialect: 'newapi_generations',
    defaultDurations: [4, 6, 8, 10, 12, 15],
    defaultResolutions: ['480p', '720p'],
    defaultAspectRatios: ['16:9', '9:16'],
  },
  {
    id: 'unknown',
    label: 'Compatible Video',
    dialect: 'newapi_generations',
    defaultDurations: [4, 6, 8, 10, 12, 15],
    defaultResolutions: ['480p', '720p'],
    defaultAspectRatios: ['16:9', '9:16', '1:1'],
  },
] as const

export type ProfileOverrideForm = {
  durations: string
  resolutions: string
  aspect_ratios: string
  dialect: string
}

export type CompatVideoSettingsValues = {
  profiles: Record<string, ProfileOverrideForm>
}

function splitList(value: string): string[] {
  return [
    ...new Set(
      value
        .split(/[,，\s]+/)
        .map((item) => item.trim())
        .filter(Boolean)
    ),
  ]
}

function toText(values: readonly number[] | readonly string[] | undefined): string {
  if (!values?.length) return ''
  return values.join(', ')
}

export function emptyCompatVideoProfiles(): CompatVideoSettingsValues {
  return {
    profiles: Object.fromEntries(
      COMPAT_VIDEO_PROFILES.map((profile) => [
        profile.id,
        { durations: '', resolutions: '', aspect_ratios: '', dialect: '' },
      ])
    ),
  }
}

export function parseCompatibleVideoProfiles(
  raw: string
): CompatVideoSettingsValues {
  let parsed: unknown
  try {
    parsed = JSON.parse(raw) as unknown
  } catch {
    return emptyCompatVideoProfiles()
  }
  if (!Array.isArray(parsed)) return emptyCompatVideoProfiles()

  const values = emptyCompatVideoProfiles()
  for (const entry of parsed) {
    if (!entry || typeof entry !== 'object') continue
    const record = entry as Record<string, unknown>
    const id = typeof record.id === 'string' ? record.id : ''
    const form = values.profiles[id]
    if (!form) continue
    if (Array.isArray(record.durations)) form.durations = toText(record.durations as number[])
    if (Array.isArray(record.resolutions)) form.resolutions = toText(record.resolutions as string[])
    if (Array.isArray(record.aspect_ratios)) {
      form.aspect_ratios = toText(record.aspect_ratios as string[])
    }
    if (typeof record.dialect === 'string') form.dialect = record.dialect
  }
  return values
}

export function serializeCompatibleVideoProfiles(
  values: CompatVideoSettingsValues
): unknown[] {
  const overrides: Record<string, unknown>[] = []
  for (const profile of COMPAT_VIDEO_PROFILES) {
    const form = values.profiles[profile.id]
    const payload: Record<string, unknown> = { id: profile.id }
    const durations = splitList(form.durations)
      .map((item) => Number(item))
      .filter((n) => !Number.isNaN(n))
    const resolutions = splitList(form.resolutions)
    const aspectRatios = splitList(form.aspect_ratios)
    if (durations.length > 0) payload.durations = durations
    if (resolutions.length > 0) payload.resolutions = resolutions
    if (aspectRatios.length > 0) payload.aspect_ratios = aspectRatios
    if (form.dialect) payload.dialect = form.dialect
    overrides.push(payload)
  }
  return overrides
}
