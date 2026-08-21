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
  generationTypeDisableReason,
  generationTypesForProfile,
  isVideoStoragePhase,
  modelSupportsGenerationType,
  retainCompatibleVideoModel,
  resolutionFromModelName,
  resolveVideoProfile,
  resolveSelectedOption,
  videoPlayModeModelName,
  videoRequestResolution,
} from '../lib/capabilities'
import type {
  PublicGenerationType,
  PublicProfile,
  VideoProviderConfig,
} from '../types'

function generationType(
  value: string,
  requireRefModel: boolean
): PublicGenerationType {
  return {
    label: value,
    value,
    sort: 1,
    require_ref_model: requireRefModel,
    require_audio: false,
    allow_audio: false,
    require_video: false,
    allow_video: false,
    images_min: requireRefModel ? 1 : 0,
    images_max: requireRefModel ? 1 : 0,
    videos_min: 0,
    videos_max: 0,
    image_roles: requireRefModel ? ['reference'] : [],
  }
}

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
    require_ref_model_suffix: true,
    media: {
      min_items: 0,
      max_items: 0,
      accepted_types: [],
      allowed_roles: [],
      allow_audio: false,
      allow_video: false,
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
    require_ref_model_suffix: true,
    media: {
      min_items: 0,
      max_items: 0,
      accepted_types: [],
      allowed_roles: [],
      allow_audio: false,
      allow_video: false,
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

  test('disables image modes for a non-ref model and text-to-video for a ref model', () => {
    const textToVideo = generationType('text2video', false)
    const imageToVideo = generationType('image2video', true)

    assert.equal(
      modelSupportsGenerationType('dreamina-seedance-2-0-480p', textToVideo),
      true
    )
    assert.equal(
      modelSupportsGenerationType('dreamina-seedance-2-0-480p', imageToVideo),
      false
    )
    assert.equal(
      generationTypeDisableReason('dreamina-seedance-2-0-480p', imageToVideo),
      'requires_ref_model'
    )

    assert.equal(
      modelSupportsGenerationType(
        'dreamina-seedance-2-0-480p-ref',
        textToVideo
      ),
      false
    )
    assert.equal(
      modelSupportsGenerationType(
        'dreamina-seedance-2-0-480p-ref',
        imageToVideo
      ),
      true
    )
    assert.equal(
      generationTypeDisableReason(
        'dreamina-seedance-2-0-480p-ref',
        textToVideo
      ),
      'requires_non_ref_model'
    )
  })

  test('uses the public model id for -ref play modes when upstream has no -ref', () => {
    const imageToVideo = generationType('image2video', true)
    const textToVideo = generationType('text2video', false)
    const modelName = videoPlayModeModelName({
      id: 'seedance-2-0-fast-ref',
      profile_model: '48:seedance-2.0-fast',
    })

    assert.equal(modelName, 'seedance-2-0-fast-ref')
    assert.equal(generationTypeDisableReason(modelName, imageToVideo), null)
    assert.equal(
      generationTypeDisableReason(modelName, textToVideo),
      'requires_non_ref_model'
    )
    assert.equal(
      generationTypeDisableReason(
        videoPlayModeModelName({
          id: 'seedance-2-0-fast',
          profile_model: '48:seedance-2.0-fast',
        }),
        imageToVideo
      ),
      'requires_ref_model'
    )
  })

  test('keeps every mode enabled for providers that do not require -ref names', () => {
    const textToVideo = generationType('text2video', false)
    const imageToVideo = generationType('image2video', false)

    assert.equal(modelSupportsGenerationType('seedance-2-0', textToVideo), true)
    assert.equal(
      modelSupportsGenerationType('seedance-2-0', imageToVideo),
      true
    )
    assert.equal(
      generationTypeDisableReason('seedance-2-0', imageToVideo),
      null
    )
  })

  test('clears require_ref_model when the profile disables the -ref suffix rule', () => {
    const provider: VideoProviderConfig = {
      id: 'silkroad',
      label: 'SilkRoad',
      groups: ['default'],
      default_profile_id: 'grok',
      strict_model_matching: false,
      generation_types: [
        generationType('text2video', false),
        generationType('image2video', true),
      ],
      profiles: [
        {
          ...profiles[0],
          id: 'grok',
          label: 'Grok',
          exact_models: ['grok-image-video'],
          model_prefixes: ['grok-'],
          require_ref_model_suffix: false,
        },
      ],
    }
    const generationTypes = generationTypesForProfile(
      provider,
      provider.profiles[0]
    )
    const imageMode = generationTypes.find(
      (mode) => mode.value === 'image2video'
    )
    assert.ok(imageMode)
    assert.equal(imageMode.require_ref_model, false)
    assert.equal(
      generationTypeDisableReason('grok-image-video', imageMode),
      null
    )
  })
})

describe('resolutionFromModelName', () => {
  test('reads resolution suffixes from local aliases', () => {
    assert.equal(resolutionFromModelName('seedance-2-0-480p'), '480p')
    assert.equal(
      resolutionFromModelName('dreamina-seedance-2-0-720p-ref'),
      '720p'
    )
    assert.equal(resolutionFromModelName('minimax-h3-480p-ref'), '480p')
    assert.equal(resolutionFromModelName('seedance-2-0-1080p'), '1080p')
    assert.equal(resolutionFromModelName('seedance-2-0-4k'), '4K')
    assert.equal(resolutionFromModelName('minimax-h3-1k'), '1K')
    assert.equal(resolutionFromModelName('seedance-2-0'), '')
  })
})

describe('videoRequestResolution', () => {
  test('sends the model-encoded resolution, using Seedance 4k casing for SilkRoad', () => {
    assert.equal(
      videoRequestResolution('minimax-h3-480p-ref', 'aistarslab'),
      '480p'
    )
    assert.equal(videoRequestResolution('seedance-2-0-720p', 'brioi'), '720p')
    assert.equal(videoRequestResolution('seedance-2-0-720p', 'silkroad'), '720p')
    assert.equal(videoRequestResolution('seedance-2-0-4k', 'silkroad'), '4k')
    assert.equal(videoRequestResolution('seedance-2-0-4k', 'brioi'), '4K')
    assert.equal(videoRequestResolution('seedance-2-0', 'brioi'), '')
  })
})
