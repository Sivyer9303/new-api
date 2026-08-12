/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import type { SystemOption, VideoSettings } from '../types'

const DEFAULT_STORAGE =
  '{"driver":"local","local_dir":"data/videos","max_retry":5,"ingest_node_name":"","public_download_base_url":""}'

export const VIDEO_RETENTION_DAYS = 7

function hasOption(options: SystemOption[] | undefined, key: string) {
  return options?.some((option) => option.key === key) ?? false
}

function resolveLegacyStorage(raw: string): string {
  try {
    const legacy = JSON.parse(raw) as Record<string, unknown>
    return JSON.stringify({
      driver: 'local',
      local_dir:
        typeof legacy.local_dir === 'string' ? legacy.local_dir : 'data/videos',
      max_retry: typeof legacy.max_retry === 'number' ? legacy.max_retry : 5,
      ingest_node_name:
        typeof legacy.ingest_node_name === 'string'
          ? legacy.ingest_node_name
          : '',
      public_download_base_url:
        typeof legacy.public_download_base_url === 'string'
          ? legacy.public_download_base_url
          : '',
    })
  } catch {
    return DEFAULT_STORAGE
  }
}

export function resolveVideoSettings(
  settings: VideoSettings,
  raw: SystemOption[] | undefined
): VideoSettings {
  const resolved = { ...settings }
  if (!hasOption(raw, 'video_setting.video_tool_groups')) {
    resolved['video_setting.video_tool_groups'] =
      settings['silkroad_setting.video_tool_groups']
  }
  if (!hasOption(raw, 'video_setting.storage')) {
    resolved['video_setting.storage'] = resolveLegacyStorage(
      settings['silkroad_setting.storage']
    )
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
