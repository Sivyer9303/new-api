import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  filterModelsForProfile,
  isVideoStoragePhase,
  resolveVideoProfile,
} from '../lib/capabilities'
import type { PublicProfile } from '../types'

const profiles: PublicProfile[] = [
  {
    id: 'default',
    label: 'Default',
    exact_models: [],
    model_prefixes: ['seedance-'],
    durations: [],
    aspect_ratios: [],
  },
  {
    id: 'pro',
    label: 'Pro',
    exact_models: ['special-model'],
    model_prefixes: ['seedance-pro-'],
    durations: [],
    aspect_ratios: [],
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
})
