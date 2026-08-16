/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import type { SystemOption, VideoSettings } from '../types'
import {
  defaultVideoStorageValues,
  MIN_RETENTION_DAYS,
  serializeVideoStorage,
  type VideoStorageValues,
} from './storage-config'
import {
  defaultVideoUploadLimitsValues,
  serializeVideoUploadLimits,
} from './upload-limits'

export const DEFAULT_VIDEO_STORAGE = serializeVideoStorage(
  defaultVideoStorageValues()
)

export const DEFAULT_VIDEO_UPLOAD_LIMITS = serializeVideoUploadLimits(
  defaultVideoUploadLimitsValues()
)

function hasOption(options: SystemOption[] | undefined, key: string) {
  return options?.some((option) => option.key === key) ?? false
}

/**
 * Maps a legacy `silkroad_setting.storage` row onto the generic storage payload.
 * Legacy installs only ever used the local driver.
 */
function resolveLegacyStorage(raw: string): string {
  let legacy: Record<string, unknown>
  try {
    legacy = JSON.parse(raw) as Record<string, unknown>
  } catch {
    return DEFAULT_VIDEO_STORAGE
  }
  const defaults = defaultVideoStorageValues()
  const values: VideoStorageValues = {
    ...defaults,
    local_dir:
      typeof legacy.local_dir === 'string' && legacy.local_dir.trim()
        ? legacy.local_dir
        : defaults.local_dir,
    max_retry:
      typeof legacy.max_retry === 'number' && legacy.max_retry >= 1
        ? legacy.max_retry
        : defaults.max_retry,
    ingest_node_name:
      typeof legacy.ingest_node_name === 'string'
        ? legacy.ingest_node_name
        : '',
    public_download_base_url:
      typeof legacy.public_download_base_url === 'string'
        ? legacy.public_download_base_url
        : '',
    local_retention_days:
      typeof legacy.retention_days === 'number' &&
      legacy.retention_days >= MIN_RETENTION_DAYS
        ? legacy.retention_days
        : defaults.local_retention_days,
  }
  return serializeVideoStorage(values)
}

export function resolveVideoSettings(
  settings: VideoSettings,
  raw: SystemOption[] | undefined
): VideoSettings {
  const resolved = { ...settings }
  if (!hasOption(raw, 'video_setting.storage')) {
    resolved['video_setting.storage'] = resolveLegacyStorage(
      settings['silkroad_setting.storage']
    )
  }
  if (!hasOption(raw, 'video_setting.upload_limits')) {
    resolved['video_setting.upload_limits'] = DEFAULT_VIDEO_UPLOAD_LIMITS
  }
  if (!hasOption(raw, 'video_setting.enabled')) {
    try {
      const profiles = JSON.parse(
        settings['silkroad_setting.profiles']
      ) as unknown
      resolved['video_setting.enabled'] =
        Array.isArray(profiles) && profiles.length > 0
    } catch {
      resolved['video_setting.enabled'] = false
    }
  }
  return resolved
}
