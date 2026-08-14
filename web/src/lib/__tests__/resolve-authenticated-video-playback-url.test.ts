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

import { resolveAuthenticatedVideoPlaybackUrl } from '../resolve-authenticated-video-playback-url'

describe('resolveAuthenticatedVideoPlaybackUrl', () => {
  test('opens the signed URL returned as JSON for object storage', async () => {
    const signed =
      'https://acct.r2.cloudflarestorage.com/videos/task_x?X-Amz-Signature=abc'
    const originalFetch = globalThis.fetch
    globalThis.fetch = (async () =>
      new Response(JSON.stringify({ url: signed }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })) as typeof fetch
    try {
      const result = await resolveAuthenticatedVideoPlaybackUrl(
        '/v1/videos/task_x/content',
        'Bearer tok'
      )
      assert.equal(result.url, signed)
      assert.equal(result.revoke, undefined)
    } finally {
      globalThis.fetch = originalFetch
    }
  })

  test('builds a blob URL when the content endpoint streams bytes', async () => {
    const originalFetch = globalThis.fetch
    const originalCreate = URL.createObjectURL
    globalThis.fetch = (async () =>
      new Response(new Blob(['mp4'], { type: 'video/mp4' }), {
        status: 200,
        headers: { 'Content-Type': 'video/mp4' },
      })) as typeof fetch
    URL.createObjectURL = () => 'blob:preview'
    try {
      const result = await resolveAuthenticatedVideoPlaybackUrl(
        '/v1/videos/task_local/content',
        'Bearer tok'
      )
      assert.equal(result.url, 'blob:preview')
      assert.equal(typeof result.revoke, 'function')
    } finally {
      globalThis.fetch = originalFetch
      URL.createObjectURL = originalCreate
    }
  })
})
