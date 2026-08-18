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
  CHANNEL_TYPE_AISTARSLAB,
  CHANNEL_TYPE_COMPAT_VIDEO,
  CHANNEL_TYPE_OPTIONS,
  MODEL_FETCHABLE_TYPES,
} from '../../constants'
import {
  buildSettingJSON,
  CHANNEL_FORM_DEFAULT_VALUES,
  channelFormSchema,
} from '../channel-form'
import { getChannelTypeConfig } from '../channel-type-config'
import { getChannelTypeIcon, getKeyPromptForType } from '../channel-utils'

function aiStarsLabForm(baseUrl: string) {
  return {
    ...CHANNEL_FORM_DEFAULT_VALUES,
    name: 'AIStarsLab video',
    type: CHANNEL_TYPE_AISTARSLAB,
    base_url: baseUrl,
    key: 'test-key',
    models: 'test:test-video',
  }
}

describe('AIStarsLab channel', () => {
  test('registers after xtoken with model discovery and official Base URL', () => {
    assert.deepEqual(
      CHANNEL_TYPE_OPTIONS.find(
        (item) => item.value === CHANNEL_TYPE_AISTARSLAB
      ),
      { value: CHANNEL_TYPE_AISTARSLAB, label: 'AIStarsLab' }
    )
    assert.equal(
      CHANNEL_TYPE_OPTIONS.findIndex(
        (item) => item.value === CHANNEL_TYPE_COMPAT_VIDEO
      ) + 1,
      CHANNEL_TYPE_OPTIONS.findIndex(
        (item) => item.value === CHANNEL_TYPE_AISTARSLAB
      )
    )
    assert.equal(MODEL_FETCHABLE_TYPES.has(CHANNEL_TYPE_AISTARSLAB), true)
    assert.equal(getChannelTypeIcon(CHANNEL_TYPE_AISTARSLAB), 'AIStarsLab')
    assert.equal(
      getKeyPromptForType(CHANNEL_TYPE_AISTARSLAB),
      'Enter API key for this channel'
    )
    assert.equal(
      getChannelTypeConfig(CHANNEL_TYPE_AISTARSLAB).defaultBaseUrl,
      'https://api.video.aistarslab.com/openai'
    )
  })

  test('requires an administrator-supplied non-blank Base URL', () => {
    const blankResult = channelFormSchema.safeParse(aiStarsLabForm('  '))
    assert.equal(blankResult.success, false)
    assert.equal(
      channelFormSchema.safeParse(
        aiStarsLabForm('https://api.video.aistarslab.com/openai')
      ).success,
      true
    )
  })

  test('forces R2 signed URL delivery when serializing channel settings', () => {
    const setting = JSON.parse(
      buildSettingJSON({
        ...aiStarsLabForm('https://api.video.aistarslab.com/openai'),
        video_input_media_delivery: 'inline_base64',
      })
    ) as Record<string, unknown>

    assert.equal(setting.video_input_media_delivery, 'r2_presigned_url')
  })
})
