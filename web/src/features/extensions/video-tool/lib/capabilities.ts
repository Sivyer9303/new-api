/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import type { PublicProfile } from '../types'

export function resolveVideoProfile(
  profiles: PublicProfile[],
  modelID: string,
  defaultProfileID: string
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
  return (
    profiles.find((profile) => profile.id === defaultProfileID) ??
    profiles[0] ??
    null
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
