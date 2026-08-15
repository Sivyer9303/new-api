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

  test('always sends normalized media and numeric seconds', () => {
    assert.deepEqual(
      buildVideoGenerationRequest({
        model: 'seedance-ref',
        prompt: 'Use normalized media fields',
        generationType: 'reference_audio',
        aspectRatio: '16:9',
        durationValue: '5',
        durationFieldKey: 'seconds',
        images: ['image-data-url'],
        audioURL: 'audio-data-url',
      }),
      {
        model: 'seedance-ref',
        prompt: 'Use normalized media fields',
        generation_type: 'reference_audio',
        aspect_ratio: '16:9',
        seconds: 5,
        media: [
          {
            type: 'image',
            role: 'reference',
            source: 'image-data-url',
          },
          {
            type: 'audio',
            source: 'audio-data-url',
          },
        ],
      }
    )
  })

  test('sends reference videos through normalized media', () => {
    assert.deepEqual(
      buildVideoGenerationRequest({
        model: 'seedance-ref',
        prompt: 'Follow @Video1',
        generationType: 'reference_videos',
        aspectRatio: '16:9',
        durationValue: '5',
        durationFieldKey: 'seconds',
        images: ['image-data-url'],
        videos: ['video-data-url'],
      }),
      {
        model: 'seedance-ref',
        prompt: 'Follow @Video1',
        generation_type: 'reference_videos',
        aspect_ratio: '16:9',
        seconds: 5,
        media: [
          {
            type: 'image',
            role: 'reference',
            source: 'image-data-url',
          },
          {
            type: 'video',
            role: 'reference',
            source: 'video-data-url',
          },
        ],
      }
    )
  })

  test('builds Brioi mixed image, video, and audio media refs', () => {
    assert.deepEqual(
      buildVideoGenerationRequest({
        model: 'seedance-2-0',
        prompt: '保持 @图片1，衔接 @视频1，声音参考 @音频1',
        generationType: 'reference_videos',
        aspectRatio: '16:9',
        durationValue: '10',
        durationFieldKey: 'duration',
        resolution: '720p',
        images: ['image-data-url'],
        imageRoles: ['reference'],
        audioURL: 'audio-data-url',
        videos: ['video-data-url'],
      }),
      {
        model: 'seedance-2-0',
        prompt: '保持 @图片1，衔接 @视频1，声音参考 @音频1',
        generation_type: 'reference_videos',
        aspect_ratio: '16:9',
        resolution: '720p',
        duration: 10,
        media: [
          {
            type: 'image',
            role: 'reference',
            source: 'image-data-url',
          },
          {
            type: 'audio',
            source: 'audio-data-url',
          },
          {
            type: 'video',
            role: 'reference',
            source: 'video-data-url',
          },
        ],
      }
    )
  })

  test('includes generate_audio when the profile allows it', () => {
    assert.deepEqual(
      buildVideoGenerationRequest({
        model: 'seedance-2-0',
        prompt: 'A cat walks',
        generationType: 'text2video',
        aspectRatio: '16:9',
        durationValue: '8',
        durationFieldKey: 'seconds',
        resolution: '720p',
        generateAudio: true,
      }),
      {
        model: 'seedance-2-0',
        prompt: 'A cat walks',
        generation_type: 'text2video',
        aspect_ratio: '16:9',
        resolution: '720p',
        seconds: 8,
        generate_audio: true,
      }
    )
  })
})
