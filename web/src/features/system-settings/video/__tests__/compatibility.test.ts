/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import type { SystemOption, VideoSettings } from '../../types'
import {
  DEFAULT_VIDEO_UPLOAD_LIMITS,
  resolveVideoSettings,
} from '../compatibility'
import {
  DEFAULT_LOCAL_RETENTION_DAYS,
  DEFAULT_R2_RETENTION_DAYS,
} from '../storage-config'

const legacySettings: VideoSettings = {
  'video_setting.enabled': false,
  'video_setting.video_tool_groups': '[]',
  'video_setting.storage': '{}',
  'video_setting.upload_limits': DEFAULT_VIDEO_UPLOAD_LIMITS,
  'silkroad_setting.common': '{}',
  'silkroad_setting.profiles': '[{"id":"legacy"}]',
  'silkroad_setting.default_profile_id': 'legacy',
  'brioi_setting.profiles': '[]',
  'compatvideo_setting.profiles': '[]',
  'aistarslab_setting.profiles': '[]',
  'silkroad_setting.storage':
    '{"enabled":true,"driver":"local","local_dir":"legacy/videos","retention_days":7,"max_retry":9,"ingest_node_name":"legacy-node","public_download_base_url":"https://video.example.com"}',
}

describe('video settings compatibility', () => {
  test('shows legacy values until generic options are explicitly saved', () => {
    const resolved = resolveVideoSettings(legacySettings, [])

    assert.equal(resolved['video_setting.enabled'], true)
    const storage = JSON.parse(resolved['video_setting.storage'])
    assert.equal(storage.driver, 'local')
    assert.equal(storage.local_dir, 'legacy/videos')
    assert.equal(storage.max_retry, 9)
    assert.equal(storage.ingest_node_name, 'legacy-node')
    assert.equal(storage.public_download_base_url, 'https://video.example.com')
    assert.equal(storage.local_retention_days, 7)
  })

  test('carries a custom legacy retention into the local driver', () => {
    const resolved = resolveVideoSettings(
      {
        ...legacySettings,
        'silkroad_setting.storage':
          '{"enabled":true,"local_dir":"legacy/videos","retention_days":14}',
      },
      []
    )

    const storage = JSON.parse(resolved['video_setting.storage'])
    assert.equal(storage.local_retention_days, 14)
  })

  test('falls back to driver defaults when the legacy row is unusable', () => {
    const resolved = resolveVideoSettings(
      { ...legacySettings, 'silkroad_setting.storage': 'not-json' },
      []
    )

    const storage = JSON.parse(resolved['video_setting.storage'])
    assert.equal(storage.driver, 'local')
    assert.equal(storage.local_retention_days, DEFAULT_LOCAL_RETENTION_DAYS)
    assert.equal(storage.r2.retention_days, DEFAULT_R2_RETENTION_DAYS)
  })

  test('keeps explicit generic values instead of legacy fallbacks', () => {
    const explicit = {
      ...legacySettings,
      'video_setting.enabled': false,
      'video_setting.video_tool_groups': '["video"]',
      'video_setting.storage':
        '{"driver":"r2","local_dir":"new/videos","max_retry":3,"ingest_node_name":"","public_download_base_url":"https://new.example.com"}',
    }
    const raw: SystemOption[] = [
      { key: 'video_setting.enabled', value: 'false' },
      { key: 'video_setting.video_tool_groups', value: '["video"]' },
      {
        key: 'video_setting.storage',
        value: explicit['video_setting.storage'],
      },
    ]

    assert.deepEqual(resolveVideoSettings(explicit, raw), explicit)
  })
})
