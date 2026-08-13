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

import { buildVideoGenerationRequest } from '../lib/request'

describe('video generation request construction', () => {
  test('preserves provider duration fields and optional media capabilities', () => {
    assert.deepEqual(
      buildVideoGenerationRequest({
        model: 'seedance-pro-ref',
        prompt: '  Generate a launch clip  ',
        generationType: 'reference_audio',
        aspectRatio: '16:9',
        durationValue: '10',
        durationFieldKey: 'duration',
        resolution: '1080p',
        images: ['data:image/png;base64,image'],
        imageRoles: ['first_frame'],
        audioURL: 'data:audio/mpeg;base64,audio',
      }),
      {
        model: 'seedance-pro-ref',
        prompt: 'Generate a launch clip',
        generation_type: 'reference_audio',
        aspect_ratio: '16:9',
        resolution: '1080p',
        duration: 10,
        media: [
          {
            type: 'image',
            role: 'first_frame',
            source: 'data:image/png;base64,image',
          },
          {
            type: 'audio',
            source: 'data:audio/mpeg;base64,audio',
          },
        ],
      }
    )
  })

  test('assigns first and last frame roles while omitting stale resolution', () => {
    assert.deepEqual(
      buildVideoGenerationRequest({
        model: 'seedance-2-0',
        prompt: 'Transition between frames',
        generationType: 'start_end',
        aspectRatio: '16:9',
        durationValue: '8',
        durationFieldKey: 'duration',
        images: ['first-data-url', 'last-data-url'],
        imageRoles: ['first_frame', 'last_frame'],
      }),
      {
        model: 'seedance-2-0',
        prompt: 'Transition between frames',
        generation_type: 'start_end',
        aspect_ratio: '16:9',
        duration: 8,
        media: [
          {
            type: 'image',
            role: 'first_frame',
            source: 'first-data-url',
          },
          {
            type: 'image',
            role: 'last_frame',
            source: 'last-data-url',
          },
        ],
      }
    )
  })

  test('preserves SilkRoad legacy image and audio fields', () => {
    assert.deepEqual(
      buildVideoGenerationRequest({
        model: 'seedance-ref',
        prompt: 'Use legacy media fields',
        generationType: 'reference_audio',
        aspectRatio: '16:9',
        durationValue: '5',
        durationFieldKey: 'seconds',
        images: ['image-data-url'],
        audioURL: 'audio-data-url',
        mediaFormat: 'legacy',
      }),
      {
        model: 'seedance-ref',
        prompt: 'Use legacy media fields',
        generation_type: 'reference_audio',
        aspect_ratio: '16:9',
        seconds: '5',
        images: ['image-data-url'],
        audio_url: 'audio-data-url',
      }
    )
  })
})
