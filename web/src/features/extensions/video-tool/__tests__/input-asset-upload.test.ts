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
  collectReadyHttpsSources,
  isVideoInputExpired,
  resolveVideoInputContentType,
} from '../lib/input-asset-upload'
import { buildVideoGenerationRequest } from '../lib/request'

describe('video input asset upload helpers', () => {
  test('resolves common image audio and video content types', () => {
    assert.equal(
      resolveVideoInputContentType(
        new File(['x'], 'a.jpg', { type: 'image/jpeg' }),
        'image'
      ),
      'image/jpeg'
    )
    assert.equal(
      resolveVideoInputContentType(
        new File(['x'], 'a.mp3', { type: '' }),
        'audio'
      ),
      'audio/mpeg'
    )
    assert.equal(
      resolveVideoInputContentType(
        new File(['x'], 'a.mov', { type: 'video/quicktime' }),
        'video'
      ),
      'video/quicktime'
    )
  })

  test('detects expiry with a small safety skew', () => {
    assert.equal(isVideoInputExpired(100, 90), false)
    assert.equal(isVideoInputExpired(100, 96), true)
    assert.equal(isVideoInputExpired(undefined, 1), true)
  })

  test('gates generation sources until HTTPS uploads are ready', () => {
    assert.deepEqual(
      collectReadyHttpsSources([
        {
          uploadStatus: 'ready',
          sourceUrl: 'https://r2.example/a.png',
          expiresAt: Math.floor(Date.now() / 1000) + 3600,
        },
      ]),
      {
        ok: true,
        urls: ['https://r2.example/a.png'],
      }
    )
    assert.equal(
      collectReadyHttpsSources([{ uploadStatus: 'uploading' }]).ok,
      false
    )
    assert.equal(
      collectReadyHttpsSources([
        {
          uploadStatus: 'ready',
          sourceUrl: 'data:image/png;base64,xx',
          expiresAt: Math.floor(Date.now() / 1000) + 3600,
        },
      ]).ok,
      false
    )
    assert.equal(
      collectReadyHttpsSources([
        {
          uploadStatus: 'ready',
          sourceUrl: 'https://r2.example/a.png',
          expiresAt: Math.floor(Date.now() / 1000) - 10,
        },
      ]).reason,
      'expired'
    )
  })

  test('builds generation body with HTTPS media sources only', () => {
    const body = buildVideoGenerationRequest({
      model: 'demo',
      prompt: 'Use uploaded refs',
      generationType: 'reference_videos',
      aspectRatio: '16:9',
      durationValue: '5',
      durationFieldKey: 'seconds',
      images: ['https://r2.example/img.png'],
      audioURL: 'https://r2.example/a.mp3',
      videos: ['https://r2.example/v.mp4'],
    })
    assert.deepEqual(body.media, [
      {
        type: 'image',
        role: 'reference',
        source: 'https://r2.example/img.png',
      },
      {
        type: 'audio',
        source: 'https://r2.example/a.mp3',
      },
      {
        type: 'video',
        role: 'reference',
        source: 'https://r2.example/v.mp4',
      },
    ])
  })
})
