/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import type { VideoTaskDiagnostics } from '../types'

export type VideoRecoveryAvailability = {
  canRetryStorage: boolean
  canConfirmProvider: boolean
  canRefund: boolean
  refunded: boolean
}

export function getVideoRecoveryAvailability(
  diagnostics: VideoTaskDiagnostics
): VideoRecoveryAvailability {
  const refunded =
    diagnostics.status === 'REFUNDED' ||
    (diagnostics.manual_refund.refunded_at ?? 0) > 0
  const deliveryFailure =
    diagnostics.status === 'FAILURE' &&
    diagnostics.storage.status === 'failed' &&
    diagnostics.storage.no_automatic_refund === true

  return {
    canRetryStorage: deliveryFailure && !refunded,
    canConfirmProvider: !refunded,
    canRefund: deliveryFailure && diagnostics.quota > 0 && !refunded,
    refunded,
  }
}
