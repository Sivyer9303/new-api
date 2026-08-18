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
  AISTARSLAB_PROVIDER,
  parseAIStarsLabProfiles,
  serializeAIStarsLabProfiles,
} from '../aistarslab-profiles'

describe('AIStarsLab model resolution overrides', () => {
  test('uses the backend provider wire value', () => {
    assert.equal(AISTARSLAB_PROVIDER, 'aistarslab')
  })

  test('parses an empty override list', () => {
    assert.deepEqual(parseAIStarsLabProfiles('[]'), { profiles: [] })
    assert.deepEqual(parseAIStarsLabProfiles('not-json'), { profiles: [] })
  })

  test('parses and serializes public model names with resolution order preserved', () => {
    const values = parseAIStarsLabProfiles(
      JSON.stringify([
        {
          model: 'seedance-2.0-fast',
          resolutions: ['1080p', '720p'],
        },
      ])
    )
    assert.equal(values.profiles[0]?.model, 'seedance-2.0-fast')
    assert.equal(values.profiles[0]?.resolutions, '1080p, 720p')
    assert.deepEqual(serializeAIStarsLabProfiles(values), [
      {
        model: 'seedance-2.0-fast',
        resolutions: ['1080p', '720p'],
      },
    ])
  })

  test('drops blank rows when serializing', () => {
    assert.deepEqual(
      serializeAIStarsLabProfiles({
        profiles: [
          { model: '  ', resolutions: '720p' },
          { model: '  ', resolutions: '  ' },
        ],
      }),
      []
    )
  })
})
