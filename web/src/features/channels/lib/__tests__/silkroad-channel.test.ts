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

import {
  CHANNEL_TYPE_OPTIONS,
  CHANNEL_TYPE_SILKROAD,
  MODEL_FETCHABLE_TYPES,
} from '../../constants'
import { CHANNEL_FORM_DEFAULT_VALUES, channelFormSchema } from '../channel-form'
import { getChannelTypeConfig } from '../channel-type-config'
import { getChannelTypeIcon, getKeyPromptForType } from '../channel-utils'

function silkRoadForm(baseUrl: string) {
  return {
    ...CHANNEL_FORM_DEFAULT_VALUES,
    name: 'SilkRoad video',
    type: CHANNEL_TYPE_SILKROAD,
    base_url: baseUrl,
    key: 'test-key',
    models: 'seedance-2.0',
  }
}

describe('SilkRoad channel', () => {
  test('registers selection, ordering, model discovery, and icon metadata', () => {
    const option = CHANNEL_TYPE_OPTIONS.find(
      (item) => item.value === CHANNEL_TYPE_SILKROAD
    )

    assert.deepEqual(option, {
      value: CHANNEL_TYPE_SILKROAD,
      label: 'SilkRoad',
    })
    assert.equal(
      CHANNEL_TYPE_OPTIONS.findIndex((item) => item.value === 60) + 1,
      CHANNEL_TYPE_OPTIONS.findIndex(
        (item) => item.value === CHANNEL_TYPE_SILKROAD
      )
    )
    assert.equal(MODEL_FETCHABLE_TYPES.has(CHANNEL_TYPE_SILKROAD), true)
    assert.equal(getChannelTypeIcon(CHANNEL_TYPE_SILKROAD), 'SilkRoad')
    assert.equal(
      getKeyPromptForType(CHANNEL_TYPE_SILKROAD),
      'Enter API key for this channel'
    )
    assert.equal(getChannelTypeConfig(CHANNEL_TYPE_SILKROAD).icon, 'SilkRoad')
  })

  test('requires a non-blank Base URL', () => {
    const blankResult = channelFormSchema.safeParse(silkRoadForm('  '))

    assert.equal(blankResult.success, false)
    if (!blankResult.success) {
      assert.equal(
        blankResult.error.issues.some(
          (issue) =>
            issue.path[0] === 'base_url' &&
            issue.message === 'Base URL is required for this channel type'
        ),
        true
      )
    }

    assert.equal(
      channelFormSchema.safeParse(silkRoadForm('https://silkroad.example'))
        .success,
      true
    )
  })
})
