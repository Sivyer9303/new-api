/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import type {
  PublicGenerationType,
  PublicMediaLimits,
  PublicProfile,
  VideoProviderConfig,
} from '../types'

export function resolveSelectedOption(
  currentValue: string,
  options: Array<{ value: string }>
): string {
  return options.some((option) => option.value === currentValue)
    ? currentValue
    : (options[0]?.value ?? '')
}

/** Derive Brioi resolution from a local/public model alias (e.g. seedance-2-0-480p). */
export function resolutionFromModelName(modelName: string): string {
  const normalized = modelName.trim().toLowerCase().replace(/-ref$/, '')
  if (!normalized) return ''
  const suffixes: Array<[string, string]> = [
    ['-1080p', '1080p'],
    ['-720p', '720p'],
    ['-480p', '480p'],
    ['-4k', '4K'],
  ]
  for (const [suffix, value] of suffixes) {
    if (normalized.endsWith(suffix)) return value
  }
  return ''
}

export function retainCompatibleVideoModel(
  selectedModelID: string,
  compatibleModels: Array<{ id: string }>
): string {
  return compatibleModels.some((model) => model.id === selectedModelID)
    ? selectedModelID
    : ''
}

function isReferenceVideoModel(modelName: string): boolean {
  return modelName.includes('-ref')
}

export function modelSupportsGenerationType(
  modelName: string,
  generationType: PublicGenerationType
): boolean {
  const isRefModel = isReferenceVideoModel(modelName)
  if (generationType.require_ref_model) return isRefModel
  return !isRefModel
}

export type GenerationTypeDisableReason =
  | 'requires_ref_model'
  | 'requires_non_ref_model'

export function generationTypeDisableReason(
  modelName: string,
  generationType: PublicGenerationType
): GenerationTypeDisableReason | null {
  if (!modelName || modelSupportsGenerationType(modelName, generationType)) {
    return null
  }
  if (generationType.require_ref_model) return 'requires_ref_model'
  return 'requires_non_ref_model'
}

export function resolveVideoProfile(
  profiles: PublicProfile[],
  modelID: string,
  defaultProfileID: string,
  allowFallback = true
): PublicProfile | null {
  const exact = profiles.find((profile) =>
    profile.exact_models.includes(modelID)
  )
  if (exact) return exact

  let prefixMatch: PublicProfile | null = null
  let prefixLength = -1
  for (const profile of profiles) {
    for (const prefix of profile.model_prefixes) {
      if (modelID.startsWith(prefix) && prefix.length > prefixLength) {
        prefixMatch = profile
        prefixLength = prefix.length
      }
    }
  }
  if (prefixMatch) return prefixMatch
  if (!allowFallback) return null
  return (
    profiles.find((profile) => profile.id === defaultProfileID) ??
    profiles[0] ??
    null
  )
}

function applyMediaLimits(
  generationType: PublicGenerationType,
  limits: PublicMediaLimits
): PublicGenerationType {
  if (generationType.images_max <= 0 && !generationType.allow_video) {
    return {
      ...generationType,
      allow_audio: generationType.allow_audio && limits.allow_audio,
      allow_video:
        generationType.allow_video &&
        (limits.allow_video || limits.accepted_types.includes('video')),
    }
  }
  const maxItems =
    limits.max_items > 0
      ? Math.min(generationType.images_max, limits.max_items)
      : generationType.images_max
  const minItems = Math.min(
    maxItems,
    Math.max(generationType.images_min, limits.min_items)
  )
  const roles =
    limits.allowed_roles.length > 0
      ? generationType.image_roles.filter((role) =>
          limits.allowed_roles.includes(role)
        )
      : generationType.image_roles
  return {
    ...generationType,
    images_min: minItems,
    images_max: maxItems,
    image_roles: roles,
    allow_audio: generationType.allow_audio && limits.allow_audio,
    allow_video:
      generationType.allow_video &&
      (limits.allow_video ||
        limits.accepted_types.length === 0 ||
        limits.accepted_types.includes('video')),
  }
}

function hasMediaLimits(limits: PublicMediaLimits): boolean {
  return (
    limits.min_items > 0 ||
    limits.max_items > 0 ||
    limits.accepted_types.length > 0 ||
    limits.allowed_roles.length > 0 ||
    limits.allow_audio ||
    limits.allow_video
  )
}

export function generationTypesForProfile(
  provider: VideoProviderConfig,
  profile: PublicProfile
): PublicGenerationType[] {
  const enabledTypes =
    profile.generation_types.length > 0
      ? new Set(profile.generation_types)
      : null
  const enforceRefSuffix = profile.require_ref_model_suffix !== false
  return provider.generation_types
    .filter((generationType) => enabledTypes?.has(generationType.value) ?? true)
    .map((generationType) => {
      const modeLimits = profile.media_limits[generationType.value]
      const limits = modeLimits ?? profile.media
      const limited = hasMediaLimits(limits)
        ? applyMediaLimits(generationType, limits)
        : generationType
      if (enforceRefSuffix || !limited.require_ref_model) {
        return limited
      }
      return { ...limited, require_ref_model: false }
    })
    .filter((generationType) => {
      if (generationType.allow_video || generationType.require_video) {
        return true
      }
      if (generationType.images_max === 0) return true
      if (generationType.image_roles.length > 0) return true
      return generationType.allow_audio || generationType.require_audio
    })
}

export function resolveProviderVideoProfile(
  provider: VideoProviderConfig,
  modelID: string
): PublicProfile | null {
  return resolveVideoProfile(
    provider.profiles,
    modelID,
    provider.default_profile_id,
    !provider.strict_model_matching
  )
}

export function modelHasConfiguredMatch(
  profiles: PublicProfile[],
  modelID: string
): boolean {
  return profiles.some(
    (profile) =>
      profile.exact_models.includes(modelID) ||
      profile.model_prefixes.some((prefix) => modelID.startsWith(prefix))
  )
}

export function filterModelsForProfile(
  models: string[],
  profile: PublicProfile,
  requireReferenceModel: boolean,
  includeUnmatchedFallback = false,
  profiles: PublicProfile[] = [profile]
): string[] {
  const matched = models.filter(
    (modelID) =>
      profile.exact_models.includes(modelID) ||
      profile.model_prefixes.some((prefix) => modelID.startsWith(prefix)) ||
      (includeUnmatchedFallback && !modelHasConfiguredMatch(profiles, modelID))
  )
  if (!requireReferenceModel) return matched
  const references = matched.filter((modelID) => modelID.includes('-ref'))
  return references.length > 0 ? references : matched
}

export function isVideoStoragePhase(progress: string): boolean {
  return progress.trim().replace('%', '') === '99'
}
