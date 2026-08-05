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

/** Site top-up bonus shown on Model Square: pay 1 get 10. Display only. */
export const SITE_RECHARGE_RATIO = 10

export type SiteRatioPriceParts = {
  original: string
  actual: string
  /** False when the string cannot be scaled (e.g. "-" or unparsable). */
  scalable: boolean
}

/**
 * Scale an already-formatted currency string by 1 / SITE_RECHARGE_RATIO.
 * Preserves leading currency symbols and optional trailing `k` suffix.
 * Display-only — does not change billing math.
 */
export function formatPriceWithSiteRatio(
  formattedOriginal: string
): SiteRatioPriceParts {
  const original = formattedOriginal
  const match = formattedOriginal.match(/^([^\d-]*)([-\d,]+\.?\d*)(k?)$/i)
  if (!match) {
    return { original, actual: original, scalable: false }
  }

  const [, symbol, number, suffix] = match
  const parsed = Number.parseFloat(number.replaceAll(',', ''))
  if (!Number.isFinite(parsed)) {
    return { original, actual: original, scalable: false }
  }

  const scaled = parsed / SITE_RECHARGE_RATIO
  let digits = scaled.toString()
  if (digits.includes('e')) {
    digits = scaled.toFixed(20).replace(/\.?0+$/, '')
  }

  return {
    original,
    actual: `${symbol}${digits}${suffix}`,
    scalable: true,
  }
}
