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
  CHANNEL_TYPE_COMPAT_VIDEO,
  CHANNEL_TYPE_OPTIONS,
  MODEL_FETCHABLE_TYPES,
} from '../../constants'
import {
  CHANNEL_FORM_DEFAULT_VALUES,
  channelFormSchema,
} from '../channel-form'
import { getChannelTypeConfig } from '../channel-type-config'
import { getChannelTypeIcon, getKeyPromptForType } from '../channel-utils'

function compatForm(baseUrl: string) {
  return {
    ...CHANNEL_FORM_DEFAULT_VALUES,
    name: 'Compatible video',
    type: CHANNEL_TYPE_COMPAT_VIDEO,
    base_url: baseUrl,
    key: 'test-key',
    models: 'grok-image-video',
  }
}

describe('Compatible Video channel', () => {
  test('registers after Brioi with model discovery and required Base URL', () => {
    assert.deepEqual(
      CHANNEL_TYPE_OPTIONS.find(
        (item) => item.value === CHANNEL_TYPE_COMPAT_VIDEO
      ),
      { value: CHANNEL_TYPE_COMPAT_VIDEO, label: 'Compatible Video' }
    )
    assert.equal(
      CHANNEL_TYPE_OPTIONS.findIndex(
        (item) => item.value === CHANNEL_TYPE_BRIOI
      ) + 1,
      CHANNEL_TYPE_OPTIONS.findIndex(
        (item) => item.value === CHANNEL_TYPE_COMPAT_VIDEO
      )
    )
    assert.equal(MODEL_FETCHABLE_TYPES.has(CHANNEL_TYPE_COMPAT_VIDEO), true)
    assert.equal(getChannelTypeIcon(CHANNEL_TYPE_COMPAT_VIDEO), 'CompatibleVideo')
    assert.equal(
      getKeyPromptForType(CHANNEL_TYPE_COMPAT_VIDEO),
      'Enter API key for this channel'
    )
    assert.equal(
      getChannelTypeConfig(CHANNEL_TYPE_COMPAT_VIDEO).defaultBaseUrl,
      undefined
    )
  })

  test('requires an administrator-supplied non-blank Base URL', () => {
    const blankResult = channelFormSchema.safeParse(compatForm('  '))
    assert.equal(blankResult.success, false)
    assert.equal(
      channelFormSchema.safeParse(compatForm('https://gateway.example')).success,
      true
    )
  })
})

