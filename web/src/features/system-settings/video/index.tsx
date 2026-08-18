/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { SettingsPage } from '../components/settings-page'
import {
  defaultBrioiProfiles,
  serializeBrioiProfiles,
} from '../extensions/brioi-profile-schemas'
import type { VideoSettings } from '../types'
import { DEFAULT_VIDEO_STORAGE, resolveVideoSettings } from './compatibility'
import {
  getVideoSectionContent,
  getVideoSectionMeta,
  VIDEO_DEFAULT_SECTION,
} from './section-registry'
import {
  defaultVideoUploadLimitsValues,
  serializeVideoUploadLimits,
} from './upload-limits'

const DEFAULT_COMMON =
  '{"durations":[{"label":"4 秒","value":"4","upstream_key":"seconds","enabled":true,"sort":1},{"label":"5 秒","value":"5","upstream_key":"seconds","enabled":true,"sort":2},{"label":"10 秒","value":"10","upstream_key":"seconds","enabled":true,"sort":3},{"label":"15 秒","value":"15","upstream_key":"seconds","enabled":true,"sort":4}],"aspect_ratios":[{"label":"16:9","value":"16:9","upstream_key":"aspect_ratio","enabled":true,"sort":1},{"label":"9:16","value":"9:16","upstream_key":"aspect_ratio","enabled":true,"sort":2},{"label":"1:1","value":"1:1","upstream_key":"aspect_ratio","enabled":true,"sort":3}]}'
const DEFAULT_PROFILES =
  '[{"id":"seedance_reverse","label":"Seedance","model_prefixes":["seedance-2.0-"]}]'

const defaultVideoSettings: VideoSettings = {
  'video_setting.enabled': false,
  'video_setting.video_tool_groups': '[]',
  'video_setting.storage': DEFAULT_VIDEO_STORAGE,
  'video_setting.upload_limits': serializeVideoUploadLimits(
    defaultVideoUploadLimitsValues()
  ),
  'silkroad_setting.common': DEFAULT_COMMON,
  'silkroad_setting.profiles': DEFAULT_PROFILES,
  'silkroad_setting.default_profile_id': 'seedance_reverse',
  'brioi_setting.profiles': serializeBrioiProfiles(defaultBrioiProfiles()),
  'compatvideo_setting.profiles': '[]',
  'aistarslab_setting.profiles': '[]',
  'silkroad_setting.storage':
    '{"enabled":false,"driver":"local","local_dir":"data/silkroad-videos","retention_days":7,"max_retry":5,"ingest_node_name":"","public_download_base_url":""}',
}

export function VideoSettingsPage() {
  return (
    <SettingsPage
      routePath='/_authenticated/system-settings/video/$section'
      defaultSettings={defaultVideoSettings}
      defaultSection={VIDEO_DEFAULT_SECTION}
      getSectionContent={getVideoSectionContent}
      getSectionMeta={getVideoSectionMeta}
      loadingMessage='Loading video settings...'
      resolveSettings={resolveVideoSettings}
    />
  )
}
