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
  buildPromptMentionOptions,
  filterPromptMentionOptions,
  findActiveMentionQuery,
  mentionToken,
  parseMentionToken,
} from '../lib/prompt-mentions'

describe('prompt mentions', () => {
  test('builds ordered Image/Audio/Video tokens', () => {
    const options = buildPromptMentionOptions({
      images: [{ fileName: 'a.png' }, { fileName: 'b.png' }],
      audio: { fileName: 'beat.mp3' },
      videos: [{ fileName: 'cam.mp4' }],
      labelFor: (kind, index) => `${kind}-${index}`,
    })
    assert.deepEqual(
      options.map((option) => option.token),
      ['@Image1', '@Image2', '@Audio1', '@Video1']
    )
  })

  test('parses and formats mention tokens', () => {
    assert.equal(mentionToken('image', 1), '@Image1')
    assert.equal(mentionToken('image', 1, 'zh'), '@图片1')
    assert.deepEqual(parseMentionToken('@Video2'), {
      kind: 'video',
      index: 2,
    })
    assert.deepEqual(parseMentionToken('@视频2'), {
      kind: 'video',
      index: 2,
    })
    assert.equal(parseMentionToken('@image1'), null)
  })

  test('detects an open @ query before the caret', () => {
    assert.deepEqual(findActiveMentionQuery('hello @Im'), {
      start: 6,
      query: 'Im',
    })
    assert.deepEqual(findActiveMentionQuery('角色随@'), {
      start: 3,
      query: '',
    })
    assert.deepEqual(findActiveMentionQuery('运镜参考@Vid'), {
      start: 4,
      query: 'Vid',
    })
    assert.equal(findActiveMentionQuery('email@x'), null)
    assert.equal(findActiveMentionQuery('done @Image1 more'), null)
  })

  test('filters mention options by query', () => {
    const options = buildPromptMentionOptions({
      images: [{ fileName: 'cat.png' }],
      videos: [{ fileName: 'pan.mp4' }],
      labelFor: (kind, index) => `${kind}${index}`,
    })
    assert.deepEqual(
      filterPromptMentionOptions(options, 'vid').map((item) => item.token),
      ['@Video1']
    )
  })

  test('builds Brioi Chinese mention tokens', () => {
    const options = buildPromptMentionOptions({
      images: [{ fileName: 'a.png' }],
      audio: { fileName: 'voice.mp3' },
      videos: [{ fileName: 'cam.mp4' }],
      dialect: 'zh',
      labelFor: (kind, index) => `${kind}-${index}`,
    })
    assert.deepEqual(
      options.map((option) => option.token),
      ['@图片1', '@音频1', '@视频1']
    )
  })
})
