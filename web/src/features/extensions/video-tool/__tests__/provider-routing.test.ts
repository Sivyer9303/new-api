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

  test('adapts the provider map, ownership, and per-profile Brioi modes', () => {
    const config = normalizeVideoToolConfig({
      version: 2,
      enabled: true,
      group_ownership: { video: 'brioi' },
      providers: {
        brioi: {
          enabled: true,
          video_tool_groups: ['video'],
          profiles: [
            {
              model: 'seedance-2-5',
              label: 'Seedance 2.5',
              durations: [4, 29],
              resolutions: ['480p', '720p'],
              aspect_ratios: ['16:9', '9:16'],
              generation_modes: [
                {
                  value: 'multi_image',
                  sort: 3,
                  images_min: 2,
                  images_max: 30,
                  image_roles: [],
                },
                {
                  value: 'start_end',
                  sort: 5,
                  images_min: 2,
                  images_max: 2,
                  image_roles: ['first_frame', 'last_frame'],
                },
              ],
            },
          ],
        },
      },
    })
    const provider = resolveVideoProviderForGroup(config, 'video')
    assert.ok(provider)
    const profile = resolveProviderVideoProfile(provider, 'seedance-2-5')
    assert.ok(profile)
    const generationTypes = generationTypesForProfile(provider, profile)

    assert.equal(provider.id, 'brioi')
    assert.equal(provider.label, 'Brioi')
    assert.deepEqual(
      generationTypes.map((mode) => mode.value),
      ['multi_image', 'start_end']
    )
    assert.equal(generationTypes[0].images_max, 30)
    assert.deepEqual(generationTypes[1].image_roles, [
      'first_frame',
      'last_frame',
    ])
  })
})
