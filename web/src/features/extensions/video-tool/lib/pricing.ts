/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

export type VideoBillingMode = 'fixed' | 'per_second'

export type VideoPriceEstimate = {
  usd: number
  unitPrice: number
  ratio: number
  seconds: number
  billingMode: VideoBillingMode
}

export function resolveVideoBillingMode(
  billingMode: string | undefined,
  quotaType?: number
): VideoBillingMode | null {
  if (billingMode) {
    if (billingMode === 'per_second') return 'per_second'
    if (billingMode === 'fixed' || billingMode === 'ratio') return 'fixed'
    return null
  }
  return quotaType === 2 ? 'per_second' : 'fixed'
}

export function estimateVideoPrice(input: {
  modelPrice: number
  billingMode?: string
  quotaType?: number
  durationSeconds: number
  groupRatio: number
}): VideoPriceEstimate | null {
  if (
    !Number.isFinite(input.modelPrice) ||
    input.modelPrice <= 0 ||
    !Number.isFinite(input.groupRatio) ||
    input.groupRatio < 0
  ) {
    return null
  }
  const mode = resolveVideoBillingMode(input.billingMode, input.quotaType)
  if (!mode) return null
  if (
    mode === 'per_second' &&
    (!Number.isFinite(input.durationSeconds) || input.durationSeconds <= 0)
  ) {
    return null
  }
  const seconds = mode === 'per_second' ? input.durationSeconds : 1
  return {
    usd: input.modelPrice * seconds * input.groupRatio,
    unitPrice: input.modelPrice,
    ratio: input.groupRatio,
    seconds: input.durationSeconds,
    billingMode: mode,
  }
}
