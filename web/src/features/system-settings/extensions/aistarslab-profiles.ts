/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
export const AISTARSLAB_PROVIDER = 'aistarslab'
export const DEFAULT_AISTARSLAB_RESOLUTIONS = ['720p', '1080p', '1K'] as const

export type AIStarsLabModelOverrideForm = {
  model: string
  resolutions: string
}

export type AIStarsLabSettingsValues = {
  profiles: AIStarsLabModelOverrideForm[]
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

export function emptyAIStarsLabProfiles(): AIStarsLabSettingsValues {
  return { profiles: [] }
}

export function parseAIStarsLabProfiles(raw: string): AIStarsLabSettingsValues {
  let parsed: unknown
  try {
    parsed = JSON.parse(raw) as unknown
  } catch {
    return emptyAIStarsLabProfiles()
  }
  if (!Array.isArray(parsed)) return emptyAIStarsLabProfiles()

  const profiles: AIStarsLabModelOverrideForm[] = []
  for (const entry of parsed) {
    if (!entry || typeof entry !== 'object') continue
    const record = entry as Record<string, unknown>
    const model = typeof record.model === 'string' ? record.model.trim() : ''
    if (!model) continue
    const resolutions = Array.isArray(record.resolutions)
      ? record.resolutions
          .filter((item): item is string => typeof item === 'string')
          .map((item) => item.trim())
          .filter(Boolean)
      : []
    profiles.push({
      model,
      resolutions: resolutions.join(', '),
    })
  }
  return { profiles }
}

export function serializeAIStarsLabProfiles(
  values: AIStarsLabSettingsValues
): unknown[] {
  const payload: Record<string, unknown>[] = []
  for (const profile of values.profiles) {
    const model = profile.model.trim()
    if (!model) continue
    payload.push({
      model,
      resolutions: splitList(profile.resolutions),
    })
  }
  return payload
}
