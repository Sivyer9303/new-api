import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import type { SystemOption, VideoSettings } from '../../types'
import { resolveVideoSettings, VIDEO_RETENTION_DAYS } from '../compatibility'

const legacySettings: VideoSettings = {
  'video_setting.enabled': false,
  'video_setting.video_tool_groups': '[]',
  'video_setting.storage': '{}',
  'silkroad_setting.common': '{}',
  'silkroad_setting.profiles': '[{"id":"legacy"}]',
  'silkroad_setting.default_profile_id': 'legacy',
  'silkroad_setting.storage':
    '{"enabled":true,"driver":"local","local_dir":"legacy/videos","retention_days":7,"max_retry":9,"ingest_node_name":"legacy-node","public_download_base_url":"https://video.example.com"}',
  'silkroad_setting.video_tool_groups': '["default","vip"]',
}

describe('video settings compatibility', () => {
  test('keeps the video retention policy fixed at seven days', () => {
    assert.equal(VIDEO_RETENTION_DAYS, 7)
  })

  test('shows legacy values until generic options are explicitly saved', () => {
    const resolved = resolveVideoSettings(legacySettings, [])

    assert.equal(resolved['video_setting.enabled'], true)
    assert.equal(
      resolved['video_setting.video_tool_groups'],
      '["default","vip"]'
    )
    assert.deepEqual(JSON.parse(resolved['video_setting.storage']), {
      driver: 'local',
      local_dir: 'legacy/videos',
      max_retry: 9,
      ingest_node_name: 'legacy-node',
      public_download_base_url: 'https://video.example.com',
    })
  })

  test('keeps explicit generic values instead of legacy fallbacks', () => {
    const explicit = {
      ...legacySettings,
      'video_setting.enabled': false,
      'video_setting.video_tool_groups': '["video"]',
      'video_setting.storage':
        '{"driver":"local","local_dir":"new/videos","max_retry":3,"ingest_node_name":"new-node","public_download_base_url":"https://new.example.com"}',
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
