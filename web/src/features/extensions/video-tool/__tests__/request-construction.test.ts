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
        images: ['data:image/png;base64,image'],
        audioURL: 'data:audio/mpeg;base64,audio',
      }),
      {
        model: 'seedance-pro-ref',
        prompt: 'Generate a launch clip',
        generation_type: 'reference_audio',
        aspect_ratio: '16:9',
        duration: 10,
        images: ['data:image/png;base64,image'],
        audio_url: 'data:audio/mpeg;base64,audio',
      }
    )
  })
})
