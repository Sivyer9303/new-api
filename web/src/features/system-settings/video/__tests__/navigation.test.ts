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

import type { TFunction } from 'i18next'

import {
  getVideoSectionNavItems,
  VIDEO_DEFAULT_SECTION,
  VIDEO_SECTION_IDS,
} from '../section-registry'

describe('video settings navigation', () => {
  test('keeps generic settings separate from SilkRoad and Brioi routes', () => {
    const translate = ((key: string) => key) as TFunction
    const items = getVideoSectionNavItems(translate)

    assert.equal(VIDEO_DEFAULT_SECTION, 'general')
    assert.deepEqual(VIDEO_SECTION_IDS, [
      'general',
      'storage',
      'silkroad',
      'brioi',
      'compatvideo',
    ])
    assert.deepEqual(
      items.map((item) => item.url),
      [
        '/system-settings/video/general',
        '/system-settings/video/storage',
        '/system-settings/video/silkroad',
        '/system-settings/video/brioi',
        '/system-settings/video/compatvideo',
      ]
    )
  })
})
