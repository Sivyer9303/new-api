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

import { renderAuditContent } from '../format'

describe('renderAuditContent video actions', () => {
  test('localizes video diagnostics and storage retry audits', () => {
    const t = (key: string, opts?: Record<string, unknown>) => {
      if (
        key ===
        'Viewed diagnostics for video {{task_id}} owned by user {{target_user_id}}'
      ) {
        return `查看视频诊断 ${opts?.task_id}（所属用户 ${opts?.target_user_id}）`
      }
      if (
        key ===
        'Retried local storage for video {{task_id}} owned by user {{target_user_id}}'
      ) {
        return `重试视频本地转存 ${opts?.task_id}（所属用户 ${opts?.target_user_id}）`
      }
      return key
    }

    assert.equal(
      renderAuditContent(
        {
          op: {
            action: 'video.diagnostics',
            params: {
              task_id: 'task_abc',
              target_user_id: 1,
            },
          },
        },
        t
      ),
      '查看视频诊断 task_abc（所属用户 1）'
    )

    assert.equal(
      renderAuditContent(
        {
          op: {
            action: 'video.storage_retry',
            params: {
              task_id: 'task_def',
              target_user_id: 2,
            },
          },
        },
        t
      ),
      '重试视频本地转存 task_def（所属用户 2）'
    )
  })
})
