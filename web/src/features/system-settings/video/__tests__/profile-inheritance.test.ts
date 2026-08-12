import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  defaultProfileExists,
  parseProfilesToForm,
  profilesFormToApi,
} from '../../extensions/silkroad-profile-schemas'

describe('SilkRoad profile inheritance', () => {
  test('preserves exact matches and omits empty capability overrides', () => {
    const form = parseProfilesToForm(
      JSON.stringify([
        {
          id: 'seedance',
          label: 'Seedance',
          exact_models: ['seedance-2.0-pro'],
          model_prefixes: ['seedance-2.0-'],
        },
      ])
    )

    assert.equal(form[0].exact_models_text, 'seedance-2.0-pro')
    assert.deepEqual(form[0].durations, [])
    assert.deepEqual(form[0].aspect_ratios, [])
    assert.deepEqual(profilesFormToApi(form), [
      {
        id: 'seedance',
        label: 'Seedance',
        exact_models: ['seedance-2.0-pro'],
        model_prefixes: ['seedance-2.0-'],
      },
    ])
  })

  test('requires the selected default profile to remain in the profile list', () => {
    const profiles = parseProfilesToForm(
      '[{"id":"default","label":"Default","model_prefixes":[]}]'
    )

    assert.equal(defaultProfileExists('default', profiles), true)
    assert.equal(defaultProfileExists('deleted', profiles), false)
  })
})
