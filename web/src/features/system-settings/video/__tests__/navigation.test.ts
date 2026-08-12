import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import type { TFunction } from 'i18next'

import {
  getVideoSectionNavItems,
  VIDEO_DEFAULT_SECTION,
  VIDEO_SECTION_IDS,
} from '../section-registry'

describe('video settings navigation', () => {
  test('exposes general, storage, and SilkRoad as dedicated child routes', () => {
    const translate = ((key: string) => key) as TFunction
    const items = getVideoSectionNavItems(translate)

    assert.equal(VIDEO_DEFAULT_SECTION, 'general')
    assert.deepEqual(VIDEO_SECTION_IDS, ['general', 'storage', 'silkroad'])
    assert.deepEqual(
      items.map((item) => item.url),
      [
        '/system-settings/video/general',
        '/system-settings/video/storage',
        '/system-settings/video/silkroad',
      ]
    )
  })
})
