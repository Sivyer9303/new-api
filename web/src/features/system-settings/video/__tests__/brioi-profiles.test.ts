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
  createBrioiSettingsSchema,
  defaultBrioiProfiles,
  parseBrioiProfiles,
  serializeBrioiProfiles,
} from '../../extensions/brioi-profile-schemas'

describe('Brioi provider profiles', () => {
  test('uses the exact documented model duration and resolution bounds', () => {
    const profiles = defaultBrioiProfiles()

    assert.deepEqual(
      profiles.map((profile) => profile.model),
      ['seedance-2-0-fast', 'seedance-2-0', 'seedance-2-5']
    )
    assert.deepEqual(profiles[0].durations, [
      '4',
      '5',
      '6',
      '7',
      '8',
      '9',
      '10',
      '11',
      '12',
      '13',
      '14',
      '15',
    ])
    assert.deepEqual(profiles[1].resolutions, ['480p', '720p', '1080p', '4K'])
    assert.equal(profiles[2].durations.at(-1), '29')
    assert.deepEqual(profiles[2].aspect_ratios, ['16:9', '9:16'])
    assert.equal(profiles[2].max_images, 30)
  })

  test('serializes the backend profile shape while preserving disabled options', () => {
    const profiles = defaultBrioiProfiles()
    profiles[0].resolutions = ['720p']
    profiles[0].generation_types = ['text2video', 'first_frame']
    profiles[0].max_images = 1

    const serialized = serializeBrioiProfiles(profiles)
    const payload = JSON.parse(serialized)
    const parsed = parseBrioiProfiles(serialized)

    assert.equal(payload[0].model, 'seedance-2-0-fast')
    assert.equal(payload[0].durations[0], 4)
    assert.equal(payload[0].generation_modes.length, 6)
    assert.deepEqual(parsed[0].resolutions, ['720p'])
    assert.deepEqual(parsed[0].generation_types, ['text2video', 'first_frame'])
    assert.equal(parsed[0].max_images, 1)
  })

  test('caps reference_videos companion images at 9 for Seedance 2.5', () => {
    const profiles = defaultBrioiProfiles()
    assert.equal(profiles[2].max_images, 30)

    const payload = JSON.parse(serializeBrioiProfiles(profiles))
    const referenceVideos = payload[2].generation_modes.find(
      (mode: { value: string }) => mode.value === 'reference_videos'
    )

    assert.equal(referenceVideos.images_max, 9)
    assert.equal(
      payload[2].generation_modes.find(
        (mode: { value: string }) => mode.value === 'multi_image'
      ).images_max,
      30
    )
  })

  test('rejects values outside Brioi hard capabilities', () => {
    const profiles = defaultBrioiProfiles()
    profiles[0].resolutions.push('8K')
    const schema = createBrioiSettingsSchema((key) => key)
    const result = schema.safeParse({ profiles })

    assert.equal(result.success, false)
    if (!result.success) {
      assert.equal(
        result.error.issues.some(
          (issue) =>
            issue.path.join('.') === 'profiles.0.resolutions' &&
            issue.message ===
              'Option is outside the Brioi model hard capabilities'
        ),
        true
      )
    }
  })
})
