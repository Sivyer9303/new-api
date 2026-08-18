import assert from 'node:assert/strict'
import { test } from 'node:test'

import { COMPAT_VIDEO_PROVIDER } from '../compatvideo-profiles'

test('Compatible Video settings use the backend provider wire value', () => {
  assert.equal(COMPAT_VIDEO_PROVIDER, 'compat_video')
})
