/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  CHANNEL_TYPE_BRIOI,
  CHANNEL_TYPE_OPTIONS,
  CHANNEL_TYPE_SILKROAD,
  MODEL_FETCHABLE_TYPES,
} from '../../constants'
import {
  buildSettingJSON,
  CHANNEL_FORM_DEFAULT_VALUES,
  channelFormSchema,
} from '../channel-form'
import { getChannelTypeConfig } from '../channel-type-config'
import { getChannelTypeIcon, getKeyPromptForType } from '../channel-utils'

function brioiForm(baseUrl: string) {
  return {
    ...CHANNEL_FORM_DEFAULT_VALUES,
    name: 'Brioi video',
    type: CHANNEL_TYPE_BRIOI,
    base_url: baseUrl,
    key: 'test-key',
    models: 'seedance-2-0',
  }
}

describe('Brioi channel', () => {
  test('registers after SilkRoad with model discovery and fallback icon metadata', () => {
    assert.deepEqual(
      CHANNEL_TYPE_OPTIONS.find((item) => item.value === CHANNEL_TYPE_BRIOI),
      { value: CHANNEL_TYPE_BRIOI, label: 'Brioi' }
    )
    assert.equal(
      CHANNEL_TYPE_OPTIONS.findIndex(
        (item) => item.value === CHANNEL_TYPE_SILKROAD
      ) + 1,
      CHANNEL_TYPE_OPTIONS.findIndex(
        (item) => item.value === CHANNEL_TYPE_BRIOI
      )
    )
    assert.equal(MODEL_FETCHABLE_TYPES.has(CHANNEL_TYPE_BRIOI), true)
    assert.equal(getChannelTypeIcon(CHANNEL_TYPE_BRIOI), 'Brioi')
    assert.equal(
      getKeyPromptForType(CHANNEL_TYPE_BRIOI),
      'Enter API key for this channel'
    )
    assert.equal(getChannelTypeConfig(CHANNEL_TYPE_BRIOI).icon, 'Brioi')
    assert.equal(
      getChannelTypeConfig(CHANNEL_TYPE_BRIOI).defaultBaseUrl,
      undefined
    )
  })

  test('requires an administrator-supplied non-blank Base URL', () => {
    const blankResult = channelFormSchema.safeParse(brioiForm('  '))

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
      channelFormSchema.safeParse(brioiForm('https://brioi.example')).success,
      true
    )
  })

  test('forces R2 signed URL delivery when serializing channel settings', () => {
    const setting = JSON.parse(
      buildSettingJSON(brioiForm('https://brioi.example'))
    ) as Record<string, unknown>

    assert.equal(setting.video_input_media_delivery, 'r2_presigned_url')
  })
})
