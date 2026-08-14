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
  VIDEO_DELIVERY_FAILURE_I18N_KEY,
  localizeTaskFailReason,
} from '../localize-task-fail-reason'

describe('localizeTaskFailReason', () => {
  test('translates the video delivery failure message with the embedded task id', () => {
    const translated = localizeTaskFailReason(
      'Video delivery failed after generation. Contact an administrator with task ID task_abc123 for review.',
      (key, options) => {
        assert.equal(key, VIDEO_DELIVERY_FAILURE_I18N_KEY)
        assert.deepEqual(options, { taskId: 'task_abc123' })
        return `localized:${options?.taskId}`
      }
    )
    assert.equal(translated, 'localized:task_abc123')
  })

  test('leaves unknown fail reasons unchanged', () => {
    const raw = 'upstream video result blocked by fetch policy'
    assert.equal(
      localizeTaskFailReason(raw, () => {
        throw new Error('should not translate unknown reasons')
      }),
      raw
    )
  })
})
