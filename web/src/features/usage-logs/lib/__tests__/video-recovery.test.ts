import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import type { VideoTaskDiagnostics } from '../../types'
import { getVideoRecoveryAvailability } from '../video-recovery'

function diagnostics(
  overrides: Partial<VideoTaskDiagnostics> = {}
): VideoTaskDiagnostics {
  return {
    task_id: 'task_video',
    user_id: 1,
    channel_id: 2,
    platform: '61',
    status: 'FAILURE',
    quota: 100,
    storage: {
      status: 'failed',
      no_automatic_refund: true,
    },
    manual_refund: {},
    ...overrides,
  }
}

describe('video recovery action visibility', () => {
  test('enables retry and full refund only for retained delivery failures', () => {
    assert.deepEqual(getVideoRecoveryAvailability(diagnostics()), {
      canRetryStorage: true,
      canConfirmProvider: true,
      canRefund: true,
      refunded: false,
    })
  })

  test('disables every mutating recovery action after refund', () => {
    assert.deepEqual(
      getVideoRecoveryAvailability(
        diagnostics({
          status: 'REFUNDED',
          quota: 0,
          storage: { status: 'refunded' },
          manual_refund: { refunded_at: 1, quota: 100 },
        })
      ),
      {
        canRetryStorage: false,
        canConfirmProvider: false,
        canRefund: false,
        refunded: true,
      }
    )
  })

  test('keeps generation failures out of storage refund controls', () => {
    const result = getVideoRecoveryAvailability(
      diagnostics({
        storage: { status: 'failed', no_automatic_refund: false },
      })
    )

    assert.equal(result.canRetryStorage, false)
    assert.equal(result.canRefund, false)
  })
})
