/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { createElement } from 'react'

import { BrioiSettingsSection } from '../extensions/brioi-settings-section'
import { SilkRoadSettingsSection } from '../extensions/silkroad-settings-section'
import type { VideoSettings } from '../types'
import { createSectionRegistry } from '../utils/section-registry'
import { VideoGeneralSettingsSection } from './general-settings-section'
import { VideoStorageSettingsSection } from './storage-settings-section'

const VIDEO_SECTIONS = [
  {
    id: 'general',
    titleKey: 'General',
    build: (settings: VideoSettings) =>
      createElement(VideoGeneralSettingsSection, {
        defaultValues: {
          enabled: settings['video_setting.enabled'],
          uploadLimitsJson: settings['video_setting.upload_limits'],
        },
      }),
  },
  {
    id: 'storage',
    titleKey: 'Storage',
    build: (settings: VideoSettings) =>
      createElement(VideoStorageSettingsSection, {
        storageJson: settings['video_setting.storage'],
      }),
  },
  {
    id: 'silkroad',
    titleKey: 'SilkRoad',
    build: (settings: VideoSettings) =>
      createElement(SilkRoadSettingsSection, {
        defaultValues: {
          commonJson: settings['silkroad_setting.common'],
          profilesJson: settings['silkroad_setting.profiles'],
          defaultProfileID: settings['silkroad_setting.default_profile_id'],
          groupsJson: settings['silkroad_setting.video_tool_groups'],
        },
      }),
  },
  {
    id: 'brioi',
    titleKey: 'Brioi',
    build: (settings: VideoSettings) =>
      createElement(BrioiSettingsSection, {
        defaultValues: {
          groupsJson: settings['brioi_setting.video_tool_groups'],
          profilesJson: settings['brioi_setting.profiles'],
        },
      }),
  },
] as const

export type VideoSectionID = (typeof VIDEO_SECTIONS)[number]['id']

const videoRegistry = createSectionRegistry<VideoSectionID, VideoSettings>({
  sections: VIDEO_SECTIONS,
  defaultSection: 'general',
  basePath: '/system-settings/video',
  urlStyle: 'path',
})

export const VIDEO_SECTION_IDS = videoRegistry.sectionIds
export const VIDEO_DEFAULT_SECTION = videoRegistry.defaultSection
export const getVideoSectionNavItems = videoRegistry.getSectionNavItems
export const getVideoSectionContent = videoRegistry.getSectionContent
export const getVideoSectionMeta = videoRegistry.getSectionMeta
