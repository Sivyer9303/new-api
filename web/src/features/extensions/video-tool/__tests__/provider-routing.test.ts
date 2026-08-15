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
  normalizeVideoModelDiscovery,
  normalizeVideoModelList,
  VIDEO_TOOL_MODELS_ENDPOINT,
} from '../api'
import {
  generationTypesForProfile,
  resolveProviderVideoProfile,
} from '../lib/capabilities'
import {
  isVideoTokenGroupCandidate,
  normalizeVideoToolConfig,
  resolveVideoProviderByID,
  resolveVideoProviderForGroup,
} from '../lib/provider-config'

const sharedModel = 'shared-seedance'

describe('video provider routing', () => {
  test('resolves the same public model against the selected key group provider', () => {
    const config = normalizeVideoToolConfig({
      version: 2,
      enabled: true,
      providers: {
        silkroad: {
          label: 'SilkRoad',
          groups: ['silkroad-group'],
          default_profile_id: 'silkroad-default',
          generation_types: ['text2video'],
          profiles: [
            {
              id: 'silkroad-default',
              exact_models: [sharedModel],
              durations: [5],
              aspect_ratios: ['16:9'],
            },
          ],
        },
        brioi: {
          label: 'Brioi',
          groups: ['brioi-group'],
          generation_types: ['text2video'],
          profiles: [
            {
              id: 'brioi-exact',
              exact_models: [sharedModel],
              durations: [8],
              resolutions: ['720p'],
              aspect_ratios: ['9:16'],
              media: {
                max_items: 30,
                accepted_types: ['image'],
                allowed_roles: ['reference'],
              },
            },
          ],
        },
      },
    })

    const silkRoad = resolveVideoProviderForGroup(config, 'silkroad-group')
    const brioi = resolveVideoProviderForGroup(config, 'brioi-group')

    assert.equal(silkRoad?.id, 'silkroad')
    assert.equal(brioi?.id, 'brioi')
    assert.equal(silkRoad?.label, '')
    assert.equal(brioi?.label, '')
    assert.ok(silkRoad)
    assert.ok(brioi)
    assert.equal(
      resolveProviderVideoProfile(silkRoad, sharedModel)?.durations[0]?.value,
      '5'
    )
    assert.equal(
      resolveProviderVideoProfile(brioi, sharedModel)?.resolutions[0]?.value,
      '720p'
    )
  })

  test('excludes groups claimed by more than one provider', () => {
    const config = normalizeVideoToolConfig({
      providers: [
        { id: 'silkroad', groups: ['shared'], profiles: [] },
        { id: 'brioi', groups: ['shared'], profiles: [] },
      ],
    })

    assert.equal(resolveVideoProviderForGroup(config, 'shared'), null)
    assert.deepEqual(config.video_tool_groups, [])
  })

  test('keeps only currently selectable token groups with one effective owner', () => {
    const config = normalizeVideoToolConfig({
      providers: {
        brioi: {
          groups: ['video'],
          profiles: [],
        },
        silkroad: {
          groups: ['legacy-video'],
          profiles: [],
        },
      },
    })

    assert.equal(
      isVideoTokenGroupCandidate(config, 'auto', '', ['video']),
      true
    )
    assert.equal(
      isVideoTokenGroupCandidate(config, 'auto', '', ['video', 'legacy-video']),
      false
    )
    assert.equal(isVideoTokenGroupCandidate(config, '', 'video'), true)
    assert.equal(isVideoTokenGroupCandidate(config, '', 'unowned'), false)
    assert.equal(isVideoTokenGroupCandidate(config, 'video'), true)
    assert.equal(isVideoTokenGroupCandidate(config, 'unowned'), false)
    assert.equal(
      isVideoTokenGroupCandidate(config, 'legacy-video', '', [], {
        selectableGroups: new Set(['video']),
      }),
      false
    )
    assert.equal(
      isVideoTokenGroupCandidate(
        config,
        'auto',
        '',
        ['revoked', 'video', 'legacy-video'],
        {
          selectableGroups: new Set(['video', 'legacy-video']),
          maxAutoGroups: 1,
        }
      ),
      true
    )
    assert.equal(
      isVideoTokenGroupCandidate(
        config,
        'auto',
        '',
        ['revoked', 'video', 'legacy-video'],
        {
          selectableGroups: new Set(['video', 'legacy-video']),
          maxAutoGroups: 2,
        }
      ),
      false
    )
    assert.equal(
      isVideoTokenGroupCandidate(
        config,
        'auto',
        '',
        ['selectable-unowned', 'video'],
        {
          selectableGroups: new Set(['selectable-unowned', 'video']),
          maxAutoGroups: 1,
        }
      ),
      false
    )
    assert.equal(resolveVideoProviderByID(config, 62)?.id, 'brioi')
  })

  test('adapts the legacy single-provider payload without exposing another provider', () => {
    const config = normalizeVideoToolConfig({
      version: 1,
      enabled: true,
      video_tool_groups: ['legacy'],
      generation_types: [
        {
          value: 'image2video',
          images_min: 1,
          images_max: 1,
        },
      ],
      profiles: [
        {
          id: 'legacy-default',
          model_prefixes: ['seedance-'],
          durations: [5],
          aspect_ratios: ['16:9'],
        },
      ],
      default_profile_id: 'legacy-default',
    })

    assert.equal(resolveVideoProviderForGroup(config, 'legacy')?.id, 'silkroad')
    assert.equal(config.providers.length, 1)
  })

  test('keeps public and profile model names distinct in token-scoped discovery', () => {
    assert.equal(VIDEO_TOOL_MODELS_ENDPOINT, '/api/video/models')
    assert.deepEqual(
      normalizeVideoModelList({
        models: [
          {
            id: 'public-seedance',
            upstream_model: 'seedance-2-0',
            provider_id: 'brioi',
          },
        ],
      }),
      [
        {
          id: 'public-seedance',
          profile_model: 'seedance-2-0',
          provider_id: 'brioi',
        },
      ]
    )
  })

  test('preserves backend-resolved provider and group metadata', () => {
    assert.deepEqual(
      normalizeVideoModelDiscovery({
        group: ' auto ',
        resolved_groups: ['video', 'video', ''],
        provider: ' brioi ',
        models: [
          {
            id: 'public-seedance',
            profile_model: 'seedance-2-0',
            provider_id: 'brioi',
          },
        ],
      }),
      {
        group: 'auto',
        resolved_groups: ['video'],
        provider: 'brioi',
        reason: undefined,
        models: [
          {
            id: 'public-seedance',
            profile_model: 'seedance-2-0',
            provider_id: 'brioi',
          },
        ],
      }
    )
  })

  test('keeps silkroad reference_audio visible with required companion images', () => {
    const config = normalizeVideoToolConfig({
      version: 2,
      enabled: true,
      providers: {
        silkroad: {
          groups: ['silkroad-group'],
          default_profile_id: 'silkroad-default',
          generation_types: [
            {
              label: '文生视频',
              value: 'text2video',
              sort: 1,
              require_ref_model: false,
              images_min: 0,
              images_max: 0,
            },
            {
              label: '参考音频',
              value: 'reference_audio',
              sort: 5,
              require_ref_model: true,
              require_audio: true,
              allow_audio: true,
              images_min: 1,
              images_max: 9,
            },
          ],
          profiles: [
            {
              id: 'silkroad-default',
              exact_models: ['dreamina-seedance-2-0-480p-ref'],
              durations: [4],
              aspect_ratios: ['16:9'],
            },
          ],
        },
      },
    })
    const provider = resolveVideoProviderForGroup(config, 'silkroad-group')
    assert.ok(provider)
    const profile = resolveProviderVideoProfile(
      provider,
      'dreamina-seedance-2-0-480p-ref'
    )
    assert.ok(profile)
    const generationTypes = generationTypesForProfile(provider, profile)
    assert.deepEqual(
      generationTypes.map((mode) => mode.value),
      ['text2video', 'reference_audio']
    )
    const audioMode = generationTypes.find(
      (mode) => mode.value === 'reference_audio'
    )
    assert.ok(audioMode)
    assert.equal(audioMode.allow_audio, true)
    assert.equal(audioMode.require_audio, true)
    assert.deepEqual(audioMode.image_roles, ['reference'])
  })

  test('keeps silkroad reference_videos visible with optional companion images', () => {
    const config = normalizeVideoToolConfig({
      version: 2,
      enabled: true,
      providers: {
        silkroad: {
          groups: ['silkroad-group'],
          default_profile_id: 'silkroad-default',
          generation_types: [
            {
              label: '文生视频',
              value: 'text2video',
              sort: 1,
              require_ref_model: false,
              images_min: 0,
              images_max: 0,
            },
            {
              label: '参考视频',
              value: 'reference_videos',
              sort: 6,
              require_ref_model: true,
              require_video: true,
              allow_video: true,
              images_min: 0,
              images_max: 9,
              videos_min: 1,
              videos_max: 3,
            },
          ],
          profiles: [
            {
              id: 'silkroad-default',
              exact_models: ['dreamina-seedance-2-0-480p-ref'],
              durations: [4],
              aspect_ratios: ['16:9'],
            },
          ],
        },
      },
    })
    const provider = resolveVideoProviderForGroup(config, 'silkroad-group')
    assert.ok(provider)
    const profile = resolveProviderVideoProfile(
      provider,
      'dreamina-seedance-2-0-480p-ref'
    )
    assert.ok(profile)
    const generationTypes = generationTypesForProfile(provider, profile)
    assert.deepEqual(
      generationTypes.map((mode) => mode.value),
      ['text2video', 'reference_videos']
    )
    const videoMode = generationTypes.find(
      (mode) => mode.value === 'reference_videos'
    )
    assert.ok(videoMode)
    assert.equal(videoMode.allow_video, true)
    assert.equal(videoMode.require_video, true)
    assert.equal(videoMode.videos_min, 1)
    assert.equal(videoMode.videos_max, 3)
    assert.deepEqual(videoMode.image_roles, ['reference'])
  })

  test('keeps brioi reference_videos mixed media with optional audio', () => {
    const config = normalizeVideoToolConfig({
      version: 2,
      enabled: true,
      providers: {
        brioi: {
          groups: ['brioi-group'],
          default_profile_id: 'seedance-2-0',
          generation_types: [
            {
              label: 'Text to video',
              value: 'text2video',
              sort: 1,
              images_min: 0,
              images_max: 0,
            },
            {
              label: 'Reference video',
              value: 'reference_videos',
              sort: 6,
              require_video: true,
              allow_video: true,
              allow_audio: true,
              images_min: 0,
              images_max: 9,
              videos_min: 1,
              videos_max: 3,
            },
          ],
          profiles: [
            {
              id: 'seedance-2-0',
              exact_models: ['seedance-2-0'],
              durations: [4],
              aspect_ratios: ['16:9'],
              media_limits_by_mode: {
                reference_videos: {
                  min_items: 0,
                  max_items: 15,
                  accepted_types: ['image', 'video', 'audio'],
                  allowed_roles: ['reference'],
                  allow_audio: true,
                  allow_video: true,
                },
              },
            },
          ],
        },
      },
    })
    const provider = resolveVideoProviderForGroup(config, 'brioi-group')
    assert.ok(provider)
    const profile = resolveProviderVideoProfile(provider, 'seedance-2-0')
    assert.ok(profile)
    const generationTypes = generationTypesForProfile(provider, profile)
    const videoMode = generationTypes.find(
      (mode) => mode.value === 'reference_videos'
    )
    assert.ok(videoMode)
    assert.equal(videoMode.allow_video, true)
    assert.equal(videoMode.require_video, true)
    assert.equal(videoMode.allow_audio, true)
    assert.equal(videoMode.require_audio, false)
    assert.equal(videoMode.images_min, 0)
    assert.equal(videoMode.images_max, 9)
    assert.equal(videoMode.videos_min, 1)
    assert.equal(videoMode.videos_max, 3)
  })
})
