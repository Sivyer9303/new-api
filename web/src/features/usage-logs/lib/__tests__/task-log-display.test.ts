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

import type { TaskLog } from '../../types'
import {
  getTaskRequestCopyJSON,
  getTaskRequestFields,
  getTaskRequestMedia,
  hasTaskRequestSnapshot,
  taskLogActionLabel,
} from '../task-log-display'

function taskLog(overrides: Partial<TaskLog> = {}): TaskLog {
  return {
    id: 1,
    user_id: 1,
    platform: '62',
    task_id: 'task_abc',
    action: 'generate',
    channel_id: 8,
    submit_time: 1,
    status: 'SUCCESS',
    ...overrides,
  }
}

describe('task log subtitle labels', () => {
  test('prefers generation type over the stored generate action', () => {
    assert.equal(
      taskLogActionLabel(
        taskLog({
          properties: {
            request: { generation_type: 'reference_videos' },
          },
        })
      ),
      'Reference video'
    )
    assert.equal(
      taskLogActionLabel(
        taskLog({
          properties: { request: { generation_type: 'text2video' } },
        })
      ),
      'Text to video'
    )
    assert.equal(taskLogActionLabel(taskLog()), 'Image to Video')
  })
})

describe('task request snapshot display', () => {
  test('builds user-visible fields without media payloads', () => {
    const log = taskLog({
      properties: {
        origin_model_name: 'seedance-2-0',
        request: {
          model: 'seedance-2-0',
          prompt: '保持 @图片1，衔接 @视频1',
          generation_type: 'reference_videos',
          duration: 10,
          resolution: '720p',
          aspect_ratio: '16:9',
          media: [
            { type: 'image', role: 'reference' },
            { type: 'video', role: 'reference' },
          ],
        },
      },
    })

    assert.equal(hasTaskRequestSnapshot(log), true)
    assert.deepEqual(getTaskRequestFields(log), [
      { labelKey: 'Model', value: 'seedance-2-0' },
      {
        labelKey: 'Generation type',
        value: 'Reference video',
        translateValue: true,
      },
      { labelKey: 'Seconds', value: '10' },
      { labelKey: 'Resolution', value: '720p' },
      { labelKey: 'Aspect ratio', value: '16:9' },
      { labelKey: 'Prompt', value: '保持 @图片1，衔接 @视频1' },
    ])
    assert.deepEqual(getTaskRequestMedia(log), [
      { key: 'Image:Reference:1', typeKey: 'Image', roleKey: 'Reference' },
      { key: 'Video:Reference:1', typeKey: 'Video', roleKey: 'Reference' },
    ])

    const copied = getTaskRequestCopyJSON(log)
    assert.equal(copied.includes('data:'), false)
    assert.deepEqual(JSON.parse(copied), {
      model: 'seedance-2-0',
      generation_type: 'reference_videos',
      prompt: '保持 @图片1，衔接 @视频1',
      duration: 10,
      resolution: '720p',
      aspect_ratio: '16:9',
      media: [
        { type: 'image', role: 'reference' },
        { type: 'video', role: 'reference' },
      ],
    })
  })

  test('hides the request viewer when no snapshot was stored', () => {
    assert.equal(hasTaskRequestSnapshot(taskLog()), false)
    assert.equal(getTaskRequestCopyJSON(taskLog()), '{}')
  })
})
