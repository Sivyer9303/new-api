import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { normalizeVideoFetchResponse } from '../api'

describe('video task response normalization', () => {
  test('normalizes a flat OpenAI-style response without exposing upstream URLs', () => {
    const result = normalizeVideoFetchResponse({
      id: 'task_flat',
      status: 'completed',
      progress: 100,
      video_url: 'https://upstream.example/private.mp4',
    })

    assert.equal(result.id, 'task_flat')
    assert.equal(result.status, 'completed')
    assert.equal(result.progress, 100)
    assert.equal(result.video_url, undefined)
  })

  test('normalizes a nested task response and keeps only local content URLs', () => {
    const result = normalizeVideoFetchResponse({
      code: 0,
      data: {
        task_id: 'task_nested',
        status: 'SUCCESS',
        progress: '100%',
        result_url: '/v1/videos/task_nested/content',
        data: {
          video_url: 'https://upstream.example/private.mp4',
        },
      },
    })

    assert.equal(result.task_id, 'task_nested')
    assert.equal(result.status, 'SUCCESS')
    assert.equal(result.result_url, '/v1/videos/task_nested/content')
    assert.equal(result.video_url, '/v1/videos/task_nested/content')
  })

  test('rejects an external URL even when its path resembles the local content route', () => {
    const result = normalizeVideoFetchResponse({
      task_id: 'task_external',
      status: 'SUCCESS',
      result_url: 'https://upstream.example/v1/videos/provider-task/content',
    })

    assert.equal(result.result_url, undefined)
    assert.equal(result.video_url, undefined)
  })
})
