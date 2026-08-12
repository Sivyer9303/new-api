import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { LEGACY_VIDEO_TOOL_ROUTE, VIDEO_TOOL_ROUTE } from '../lib/routes'

describe('video tool routes', () => {
  test('uses the provider-neutral route and retains the legacy redirect source', () => {
    assert.equal(VIDEO_TOOL_ROUTE, '/extensions/video')
    assert.equal(LEGACY_VIDEO_TOOL_ROUTE, '/extensions/seedance')
  })
})
