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
  parseCompatibleVideoProfiles,
  serializeCompatibleVideoProfiles,
} from '../compatvideo-profiles'

describe('Compatible Video settings overrides', () => {
  test('parses an empty override list into all-inherit values', () => {
    const values = parseCompatibleVideoProfiles('[]')
    for (const profile of [
      'seedance2',
      'grok-image-video',
      'grok-video-1.5',
      'unknown',
    ]) {
      assert.equal(values.profiles[profile].durations, '')
      assert.equal(values.profiles[profile].resolutions, '')
      assert.equal(values.profiles[profile].aspect_ratios, '')
      assert.equal(values.profiles[profile].dialect, '')
    }
  })

  test('parses persisted overrides per profile', () => {
    const values = parseCompatibleVideoProfiles(
      JSON.stringify([
        {
          id: 'seedance2',
          durations: [5, 10],
          resolutions: ['1080p'],
          aspect_ratios: ['16:9', '9:16'],
          dialect: 'newapi_generations',
        },
      ])
    )
    assert.equal(values.profiles.seedance2.durations, '5, 10')
    assert.equal(values.profiles.seedance2.resolutions, '1080p')
    assert.equal(values.profiles.seedance2.aspect_ratios, '16:9, 9:16')
    assert.equal(values.profiles.seedance2.dialect, 'newapi_generations')
    assert.equal(values.profiles['grok-image-video'].durations, '')
  })

  test('serializes only configured fields and numeric durations', () => {
    const values = parseCompatibleVideoProfiles('[]')
    values.profiles.seedance2.durations = '4, 6, 8'
    values.profiles.seedance2.aspect_ratios = '16:9'
    values.profiles['grok-video-1.5'].dialect = 'openai_videos'

    const payload = serializeCompatibleVideoProfiles(values) as Record<
      string,
      unknown
    >[]

    const seedance = payload.find((entry) => entry.id === 'seedance2')
    assert.deepEqual(seedance?.durations, [4, 6, 8])
    assert.equal(seedance?.resolutions, undefined)
    assert.deepEqual(seedance?.aspect_ratios, ['16:9'])
    assert.equal(seedance?.dialect, undefined)

    const grokVideo = payload.find((entry) => entry.id === 'grok-video-1.5')
    assert.equal(grokVideo?.dialect, 'openai_videos')
    assert.equal(grokVideo?.durations, undefined)
  })

  test('inherits the built-in dialect when one is not configured', () => {
    const values = parseCompatibleVideoProfiles('[]')
    const payload = serializeCompatibleVideoProfiles(values) as Record<
      string,
      unknown
    >[]
    const seedance = payload.find((entry) => entry.id === 'seedance2')
    assert.equal(seedance?.dialect, undefined)
  })
})
