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
  filterModelsForProfile,
  isVideoStoragePhase,
  retainCompatibleVideoModel,
  resolveVideoProfile,
  resolveSelectedOption,
} from '../lib/capabilities'
import type { PublicProfile } from '../types'

const profiles: PublicProfile[] = [
  {
    id: 'default',
    label: 'Default',
    exact_models: [],
    model_prefixes: ['seedance-'],
    durations: [],
    resolutions: [],
    aspect_ratios: [],
    generation_types: [],
    media: {
      min_items: 0,
      max_items: 0,
      accepted_types: [],
      allowed_roles: [],
      allow_audio: false,
    },
    media_limits: {},
  },
  {
    id: 'pro',
    label: 'Pro',
    exact_models: ['special-model'],
    model_prefixes: ['seedance-pro-'],
    durations: [],
    resolutions: [],
    aspect_ratios: [],
    generation_types: [],
    media: {
      min_items: 0,
      max_items: 0,
      accepted_types: [],
      allowed_roles: [],
      allow_audio: false,
    },
    media_limits: {},
  },
]

describe('video tool capability resolution', () => {
  test('uses exact, longest-prefix, then selected default precedence', () => {
    assert.equal(
      resolveVideoProfile(profiles, 'special-model', 'default')?.id,
      'pro'
    )
    assert.equal(
      resolveVideoProfile(profiles, 'seedance-pro-fast', 'default')?.id,
      'pro'
    )
    assert.equal(
      resolveVideoProfile(profiles, 'future-video-model', 'default')?.id,
      'default'
    )
  })

  test('filters reference modes within the resolved profile models', () => {
    assert.deepEqual(
      filterModelsForProfile(
        ['special-model', 'seedance-pro-fast', 'seedance-pro-fast-ref'],
        profiles[1],
        true
      ),
      ['seedance-pro-fast-ref']
    )
  })

  test('includes unmatched models only for the selected default profile', () => {
    assert.deepEqual(
      filterModelsForProfile(
        ['seedance-basic', 'special-model', 'future-video-model'],
        profiles[0],
        false,
        true,
        profiles
      ),
      ['seedance-basic', 'future-video-model']
    )
  })

  test('recognizes the reserved local-storage progress phase', () => {
    assert.equal(isVideoStoragePhase('99%'), true)
    assert.equal(isVideoStoragePhase('100%'), false)
  })

  test('replaces a stale provider option and clears unsupported fields', () => {
    assert.equal(
      resolveSelectedOption('4K', [{ value: '480p' }, { value: '720p' }]),
      '480p'
    )
    assert.equal(resolveSelectedOption('720p', []), '')
  })

  test('clears a model that becomes incompatible instead of replacing it', () => {
    const compatible = [{ id: 'seedance-2.5-ref' }]

    assert.equal(retainCompatibleVideoModel('seedance-2.5', compatible), '')
    assert.equal(
      retainCompatibleVideoModel('seedance-2.5-ref', compatible),
      'seedance-2.5-ref'
    )
    assert.equal(retainCompatibleVideoModel('', compatible), '')
  })
})
