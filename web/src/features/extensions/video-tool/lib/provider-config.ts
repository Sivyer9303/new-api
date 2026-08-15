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
  PublicOption,
  PublicProfile,
  VideoMediaRole,
  VideoMediaType,
  VideoProviderConfig,
  VideoToolConfig,
} from '../types'

type UnknownRecord = Record<string, unknown>

const EMPTY_MEDIA_LIMITS: PublicMediaLimits = {
  min_items: 0,
  max_items: 0,
  accepted_types: [],
  allowed_roles: [],
  allow_audio: false,
  allow_video: false,
}

function asRecord(value: unknown): UnknownRecord | null {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return null
  return value as UnknownRecord
}

function firstDefined(record: UnknownRecord, keys: string[]): unknown {
  for (const key of keys) {
    if (record[key] !== undefined) return record[key]
  }
  return undefined
}

function asString(value: unknown): string {
  if (typeof value === 'string') return value.trim()
  if (typeof value === 'number') return String(value)
  return ''
}

function stringList(value: unknown): string[] {
  if (typeof value === 'string') {
    return value
      .split(/[,，\n]/)
      .map((item) => item.trim())
      .filter(Boolean)
  }
  if (!Array.isArray(value)) return []
  const result: string[] = []
  for (const item of value) {
    const direct = asString(item)
    if (direct) {
      result.push(direct)
      continue
    }
    const record = asRecord(item)
    if (!record || record.enabled === false) continue
    const candidate = asString(
      firstDefined(record, ['value', 'id', 'name', 'type'])
    )
    if (candidate) result.push(candidate)
  }
  return [...new Set(result)]
}

function integerValue(value: unknown, fallback: number): number {
  const number = typeof value === 'number' ? value : Number(value)
  return Number.isInteger(number) ? number : fallback
}

function publicProviderLabel(value: unknown): string {
  const label = asString(value)
  if (
    !label ||
    /^(?:brioi|silk[\s_-]*road|compat(?:ible)?[\s_-]*video)$/i.test(label)
  ) {
    return ''
  }
  return label
}

export function canonicalVideoProviderID(value: unknown): string {
  const normalized = asString(value)
    .toLowerCase()
    .replaceAll(/[\s_-]+/g, '')
  if (
    normalized === '61' ||
    normalized === 'silkroad' ||
    normalized === 'channelsilkroad'
  ) {
    return 'silkroad'
  }
  if (
    normalized === '62' ||
    normalized === 'brioi' ||
    normalized === 'channelbrioi'
  ) {
    return 'brioi'
  }
  if (
    normalized === '63' ||
    normalized === 'compatvideo' ||
    normalized === 'compatiblevideo' ||
    normalized === 'channelcompatvideo'
  ) {
    return 'compat_video'
  }
  return asString(value).toLowerCase()
}

function normalizeOptions(
  value: unknown,
  defaultUpstreamKey: string
): PublicOption[] {
  if (!Array.isArray(value)) return []
  const options: PublicOption[] = []
  for (let index = 0; index < value.length; index += 1) {
    const item = value[index]
    const record = asRecord(item)
    if (record?.enabled === false) continue
    const optionValue = record
      ? asString(firstDefined(record, ['value', 'id', 'name']))
      : asString(item)
    if (!optionValue) continue
    options.push({
      label: record ? asString(record.label) || optionValue : optionValue,
      value: optionValue,
      upstream_key:
        (record && asString(record.upstream_key)) || defaultUpstreamKey,
      sort: record ? integerValue(record.sort, index + 1) : index + 1,
    })
  }
  return options.sort((left, right) => left.sort - right.sort)
}

function mediaTypeList(value: unknown): VideoMediaType[] {
  return stringList(value).filter(
    (item): item is VideoMediaType =>
      item === 'image' || item === 'audio' || item === 'video'
  )
}

function mediaRoleList(value: unknown): VideoMediaRole[] {
  return stringList(value).filter(
    (item): item is VideoMediaRole =>
      item === 'reference' || item === 'first_frame' || item === 'last_frame'
  )
}

function normalizeMediaLimits(value: unknown): PublicMediaLimits {
  const record = asRecord(value)
  if (!record) return { ...EMPTY_MEDIA_LIMITS }
  return {
    min_items: Math.max(
      0,
      integerValue(
        firstDefined(record, ['min_items', 'images_min', 'min_images']),
        0
      )
    ),
    max_items: Math.max(
      0,
      integerValue(
        firstDefined(record, ['max_items', 'images_max', 'max_images']),
        0
      )
    ),
    accepted_types: mediaTypeList(
      firstDefined(record, ['accepted_types', 'types', 'media_types'])
    ),
    allowed_roles: mediaRoleList(
      firstDefined(record, ['allowed_roles', 'roles', 'image_roles'])
    ),
    allow_audio: record.allow_audio === true,
    allow_video: record.allow_video === true,
  }
}

function inferredImageRoles(value: string): VideoMediaRole[] {
  switch (value) {
    case 'start_end':
    case 'first_last':
    case 'first_last_frame':
      return ['first_frame', 'last_frame']
    case 'first_frame':
    case 'start_frame':
      return ['first_frame']
    case 'image2video':
    case 'multi_image':
    case 'image_reference':
    case 'reference_audio':
    case 'reference_videos':
      return ['reference']
    default:
      return []
  }
}

function defaultImageBounds(value: string): {
  min: number
  max: number
} {
  switch (value) {
    case 'image2video':
    case 'image_reference':
    case 'first_frame':
    case 'start_frame':
      return { min: 1, max: 1 }
    case 'start_end':
    case 'first_last':
    case 'first_last_frame':
      return { min: 2, max: 2 }
    case 'multi_image':
      return { min: 2, max: 30 }
    case 'reference_audio':
      // Upstream requires at least one companion image with reference audio.
      return { min: 1, max: 9 }
    case 'reference_videos':
      // Companion images are optional; videos carry the required media.
      return { min: 0, max: 9 }
    default:
      return { min: 0, max: 0 }
  }
}

function defaultVideoBounds(value: string): {
  min: number
  max: number
} {
  if (value === 'reference_videos') return { min: 1, max: 3 }
  return { min: 0, max: 0 }
}

function normalizeGenerationType(
  value: unknown,
  index: number,
  providerID: string
): PublicGenerationType | null {
  const record = asRecord(value)
  if (record?.enabled === false) return null
  const modeValue = record
    ? asString(firstDefined(record, ['value', 'id', 'name', 'type']))
    : asString(value)
  if (!modeValue) return null
  const media = normalizeMediaLimits(
    record && firstDefined(record, ['media', 'media_limits'])
  )
  const bounds = defaultImageBounds(modeValue)
  const videoBounds = defaultVideoBounds(modeValue)
  const imagesMin = record
    ? integerValue(
        firstDefined(record, ['images_min', 'min_images']),
        media.min_items || bounds.min
      )
    : bounds.min
  const imagesMax = record
    ? integerValue(
        firstDefined(record, ['images_max', 'max_images']),
        media.max_items || bounds.max
      )
    : bounds.max
  const videosMin = record
    ? integerValue(firstDefined(record, ['videos_min', 'min_videos']), videoBounds.min)
    : videoBounds.min
  const videosMax = record
    ? integerValue(firstDefined(record, ['videos_max', 'max_videos']), videoBounds.max)
    : videoBounds.max
  const explicitRoles = mediaRoleList(
    record &&
      firstDefined(record, ['image_roles', 'media_roles', 'allowed_roles'])
  )
  const imageRoles =
    explicitRoles.length > 0 ? explicitRoles : inferredImageRoles(modeValue)
  const allowVideo =
    record?.allow_video === true ||
    media.allow_video ||
    videosMax > 0 ||
    modeValue === 'reference_videos'
  const requireVideo =
    record?.require_video === true || modeValue === 'reference_videos'
  const imageMode = imagesMax > 0
  const videoMode = allowVideo || requireVideo || videosMax > 0
  return {
    label: (record && asString(record.label)) || modeValue,
    value: modeValue,
    sort: record ? integerValue(record.sort, index + 1) : index + 1,
    require_ref_model:
      record?.require_ref_model === true ||
      (providerID === 'silkroad' && (imageMode || videoMode)),
    require_audio: record?.require_audio === true,
    allow_audio: record?.allow_audio === true || media.allow_audio,
    require_video: requireVideo,
    allow_video: allowVideo,
    images_min: Math.max(0, imagesMin),
    images_max: Math.max(0, imagesMax),
    videos_min: Math.max(0, videosMin),
    videos_max: Math.max(0, videosMax),
    image_roles: imageRoles,
  }
}

export function normalizeGenerationTypes(
  value: unknown,
  providerID: string
): PublicGenerationType[] {
  if (!Array.isArray(value)) return []
  const result: PublicGenerationType[] = []
  value.forEach((item, index) => {
    const generationType = normalizeGenerationType(item, index, providerID)
    if (generationType) result.push(generationType)
  })
  return result.sort((left, right) => left.sort - right.sort)
}

function generationTypesFromProfiles(
  value: unknown,
  providerID: string
): PublicGenerationType[] {
  if (!Array.isArray(value)) return []
  const byValue = new Map<string, PublicGenerationType>()
  for (const profileValue of value) {
    const profile = asRecord(profileValue)
    const generationModes =
      profile && firstDefined(profile, ['generation_modes', 'generation_types'])
    for (const mode of normalizeGenerationTypes(generationModes, providerID)) {
      const current = byValue.get(mode.value)
      if (!current) {
        byValue.set(mode.value, mode)
        continue
      }
      byValue.set(mode.value, {
        ...current,
        images_min: Math.min(current.images_min, mode.images_min),
        images_max: Math.max(current.images_max, mode.images_max),
        videos_min: Math.min(current.videos_min, mode.videos_min),
        videos_max: Math.max(current.videos_max, mode.videos_max),
        image_roles: [
          ...new Set([...current.image_roles, ...mode.image_roles]),
        ],
        allow_audio: current.allow_audio || mode.allow_audio,
        require_audio: current.require_audio || mode.require_audio,
        allow_video: current.allow_video || mode.allow_video,
        require_video: current.require_video || mode.require_video,
      })
    }
  }
  return [...byValue.values()].sort((left, right) => left.sort - right.sort)
}

function normalizeMediaLimitMap(
  value: unknown
): Record<string, PublicMediaLimits> {
  const record = asRecord(value)
  if (!record) return {}
  const directKeys = [
    'min_items',
    'max_items',
    'accepted_types',
    'allowed_roles',
    'allow_audio',
    'allow_video',
  ]
  if (directKeys.some((key) => record[key] !== undefined)) return {}
  return Object.fromEntries(
    Object.entries(record).map(([mode, limits]) => [
      mode,
      normalizeMediaLimits(limits),
    ])
  )
}

export function normalizeProfile(value: unknown, index: number): PublicProfile | null {
  const profile = asRecord(value)
  if (!profile || profile.enabled === false) return null
  const capabilities =
    asRecord(firstDefined(profile, ['capabilities', 'overrides'])) ?? profile
  const exactModels = stringList(
    firstDefined(profile, ['exact_models', 'models'])
  )
  const singleModel = asString(
    firstDefined(profile, ['model', 'upstream_model'])
  )
  if (singleModel && !exactModels.includes(singleModel)) {
    exactModels.push(singleModel)
  }
  const id =
    asString(firstDefined(profile, ['id', 'profile_id'])) ||
    singleModel ||
    exactModels[0] ||
    `profile-${index + 1}`
  const mediaValue = firstDefined(capabilities, ['media', 'media_limits'])
  const mediaLimits = normalizeMediaLimitMap(
    firstDefined(capabilities, ['media_limits_by_mode', 'mode_media_limits']) ??
      mediaValue
  )
  const generationModes = firstDefined(capabilities, [
    'generation_modes',
    'generation_types',
  ])
  if (Array.isArray(generationModes)) {
    generationModes.forEach((modeValue, modeIndex) => {
      const mode = normalizeGenerationType(modeValue, modeIndex, '')
      if (!mode) return
      mediaLimits[mode.value] = {
        min_items: mode.images_min,
        max_items: mode.images_max,
        accepted_types: [
          ...(mode.images_max > 0 ? (['image'] as const) : []),
          ...(mode.allow_video || mode.videos_max > 0
            ? (['video'] as const)
            : []),
          ...(mode.allow_audio ? (['audio'] as const) : []),
        ],
        allowed_roles: mode.image_roles,
        allow_audio: mode.allow_audio,
        allow_video: mode.allow_video || mode.videos_max > 0,
      }
    })
  }
  return {
    id,
    label: asString(profile.label) || id,
    exact_models: exactModels,
    model_prefixes: stringList(profile.model_prefixes),
    durations: normalizeOptions(capabilities.durations, 'duration'),
    resolutions: normalizeOptions(capabilities.resolutions, 'resolution'),
    aspect_ratios: normalizeOptions(capabilities.aspect_ratios, 'aspect_ratio'),
    generation_types: stringList(generationModes),
    require_ref_model_suffix: profile.require_ref_model_suffix !== false,
    allow_generate_audio: profile.allow_generate_audio === true,
    generate_audio_default: profile.generate_audio_default === true,
    multi_image_max_duration: Math.max(
      0,
      integerValue(profile.multi_image_max_duration, 0)
    ),
    mention_dialect:
      asString(profile.mention_dialect) === 'zh' ? 'zh' : 'latin',
    media: normalizeMediaLimits(mediaValue),
    media_limits: mediaLimits,
  }
}

function normalizeProfiles(value: unknown): PublicProfile[] {
  if (!Array.isArray(value)) return []
  const result: PublicProfile[] = []
  value.forEach((item, index) => {
    const profile = normalizeProfile(item, index)
    if (profile) result.push(profile)
  })
  return result
}

function providerEntries(value: unknown): Array<[string, unknown]> {
  if (Array.isArray(value)) {
    return value.map((provider, index) => {
      const record = asRecord(provider)
      const id =
        (record &&
          asString(
            firstDefined(record, ['id', 'provider_id', 'provider', 'type'])
          )) ||
        String(index)
      return [id, provider]
    })
  }
  const record = asRecord(value)
  return record ? Object.entries(record) : []
}

function normalizeProvider(
  rawID: string,
  value: unknown,
  root: UnknownRecord
): VideoProviderConfig | null {
  const provider = asRecord(value)
  if (!provider || provider.enabled === false) return null
  const providerID = canonicalVideoProviderID(
    firstDefined(provider, ['id', 'provider_id', 'provider', 'type']) ?? rawID
  )
  if (!providerID) return null
  const profilesValue =
    firstDefined(provider, ['profiles', 'model_profiles']) ?? []
  const generationTypesValue =
    firstDefined(provider, ['generation_types', 'modes']) ??
    root.generation_types
  const strictValue = firstDefined(provider, [
    'strict_model_matching',
    'require_profile_match',
  ])
  const generationTypes = normalizeGenerationTypes(
    generationTypesValue,
    providerID
  )
  return {
    id: providerID,
    label: publicProviderLabel(provider.label) || publicProviderLabel(provider.name),
    groups: stringList(firstDefined(provider, ['groups', 'video_tool_groups'])),
    generation_types:
      generationTypes.length > 0
        ? generationTypes
        : generationTypesFromProfiles(profilesValue, providerID),
    profiles: normalizeProfiles(profilesValue),
    default_profile_id: asString(provider.default_profile_id),
    strict_model_matching:
      strictValue === true || (strictValue !== false && providerID === 'brioi'),
  }
}

function providerIDFromOwnership(value: unknown): string {
  if (Array.isArray(value)) {
    const providers = [
      ...new Set(
        value.map(canonicalVideoProviderID).filter((provider) => provider)
      ),
    ]
    return providers.length === 1 ? providers[0] : ''
  }
  const record = asRecord(value)
  if (record) {
    return canonicalVideoProviderID(
      firstDefined(record, ['provider_id', 'provider', 'id', 'type'])
    )
  }
  return canonicalVideoProviderID(value)
}

function normalizeExplicitOwnership(value: unknown): Record<string, string> {
  const record = asRecord(value)
  if (!record) return {}
  const result: Record<string, string> = {}
  for (const [rawGroup, owner] of Object.entries(record)) {
    const group = rawGroup.trim()
    const providerID = providerIDFromOwnership(owner)
    if (group && providerID) result[group] = providerID
  }
  return result
}

function deriveOwnership(
  providers: VideoProviderConfig[]
): Record<string, string> {
  const claims = new Map<string, Set<string>>()
  for (const provider of providers) {
    for (const rawGroup of provider.groups) {
      const group = rawGroup.trim()
      if (!group) continue
      const owners = claims.get(group) ?? new Set<string>()
      owners.add(provider.id)
      claims.set(group, owners)
    }
  }
  const result: Record<string, string> = {}
  for (const [group, owners] of claims) {
    if (owners.size === 1) result[group] = [...owners][0]
  }
  return result
}

export function normalizeVideoToolConfig(value: unknown): VideoToolConfig {
  const root = asRecord(value) ?? {}
  const rawProviders = firstDefined(root, [
    'providers',
    'provider_configs',
    'video_providers',
  ])
  let providers = providerEntries(rawProviders)
    .map(([id, provider]) => normalizeProvider(id, provider, root))
    .filter((provider): provider is VideoProviderConfig => provider !== null)

  if (providers.length === 0) {
    const legacyProvider = normalizeProvider(
      'silkroad',
      {
        id: 'silkroad',
        groups: root.video_tool_groups,
        generation_types: root.generation_types,
        profiles: root.profiles,
        default_profile_id: root.default_profile_id,
        strict_model_matching: false,
      },
      root
    )
    providers = legacyProvider ? [legacyProvider] : []
  }

  const explicitOwnership = normalizeExplicitOwnership(
    firstDefined(root, [
      'provider_by_group',
      'group_ownership',
      'group_owners',
      'group_provider_map',
      'provider_map',
    ])
  )
  const providerByGroup =
    Object.keys(explicitOwnership).length > 0
      ? explicitOwnership
      : deriveOwnership(providers)
  const rawVersion = integerValue(root.version, providers.length > 1 ? 2 : 1)
  const uploadLimitsRecord = asRecord(
    firstDefined(root, ['upload_limits', 'uploadLimits'])
  )
  const explicitGroups = stringList(
    firstDefined(root, ['video_tool_groups', 'groups'])
  )
  const derivedGroups = Object.keys(providerByGroup)
  const videoToolGroups =
    explicitGroups.length > 0 ? [...new Set(explicitGroups)] : derivedGroups

  return {
    version: rawVersion,
    enabled: root.enabled !== false,
    providers,
    provider_by_group: providerByGroup,
    video_tool_groups: videoToolGroups,
    upload_limits: {
      max_image_mb: Math.max(
        1,
        integerValue(uploadLimitsRecord?.max_image_mb, 10)
      ),
      max_audio_mb: Math.max(
        1,
        integerValue(uploadLimitsRecord?.max_audio_mb, 24)
      ),
      max_video_mb: Math.max(
        1,
        integerValue(uploadLimitsRecord?.max_video_mb, 50)
      ),
    },
  }
}

export function resolveVideoProviderForGroup(
  config: VideoToolConfig,
  rawGroup: string
): VideoProviderConfig | null {
  const group = rawGroup.trim() || 'default'
  const providerID = canonicalVideoProviderID(config.provider_by_group[group])
  if (!providerID) return null
  return resolveVideoProviderByID(config, providerID)
}

export function resolveVideoProviderByID(
  config: VideoToolConfig,
  rawProviderID: unknown
): VideoProviderConfig | null {
  const providerID = canonicalVideoProviderID(rawProviderID)
  if (!providerID) return null
  return config.providers.find((provider) => provider.id === providerID) ?? null
}

export function isVideoTokenGroupCandidate(
  config: VideoToolConfig,
  rawGroup: string,
  inheritedGroup = '',
  autoGroups: string[] = [],
  constraints: {
    selectableGroups?: ReadonlySet<string>
    maxAutoGroups?: number
  } = {}
): boolean {
  const { selectableGroups, maxAutoGroups } = constraints
  const videoGroups = new Set(
    config.video_tool_groups.map((group) => group.trim()).filter(Boolean)
  )
  const group = rawGroup.trim()
  if (!group) {
    const inherited = inheritedGroup.trim() || 'default'
    return videoGroups.has(inherited)
  }
  if (group === 'auto') {
    let limit = Number.POSITIVE_INFINITY
    if (maxAutoGroups !== undefined) {
      limit =
        Number.isInteger(maxAutoGroups) && maxAutoGroups > 0 ? maxAutoGroups : 0
    }
    if (limit === 0) return false

    const seenGroups = new Set<string>()
    let acceptedGroups = 0
    for (const rawCandidate of autoGroups) {
      const candidate = rawCandidate.trim()
      if (
        !candidate ||
        candidate === 'auto' ||
        seenGroups.has(candidate) ||
        (selectableGroups && !selectableGroups.has(candidate))
      ) {
        continue
      }
      seenGroups.add(candidate)
      acceptedGroups++
      if (videoGroups.has(candidate)) return true
      if (acceptedGroups >= limit) break
    }
    return false
  }
  if (selectableGroups && !selectableGroups.has(group)) return false
  return videoGroups.has(group)
}
