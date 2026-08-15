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

/** Stable i18n key; backend embeds the concrete task ID in fail_reason. */
export const VIDEO_DELIVERY_FAILURE_I18N_KEY =
  'Video delivery failed after generation. Contact an administrator with task ID {{taskId}} for review.'

export const PROVIDER_TASK_FAILED_I18N_KEY = 'Provider task failed'

export const PROVIDER_NO_USABLE_RESULT_I18N_KEY =
  'Provider completed the task without a usable result; administrator review is required'

export const PROVIDER_UNKNOWN_STATUS_I18N_KEY =
  'Provider returned an unknown task status; administrator review is required'

const VIDEO_DELIVERY_FAILURE_PATTERN =
  /^Video delivery failed after generation\. Contact an administrator with task ID (.+) for review\.$/

const VIDEO_UPSTREAM_BRAND_PATTERN = /\b(?:brioi|silk[\s_-]*road)\b/i

const GENERIC_FAIL_REASON_KEYS: Record<string, string> = {
  'Provider task failed': PROVIDER_TASK_FAILED_I18N_KEY,
  'Brioi task failed': PROVIDER_TASK_FAILED_I18N_KEY,
  'SilkRoad task failed': PROVIDER_TASK_FAILED_I18N_KEY,
  'Provider completed the task without a usable result; administrator review is required':
    PROVIDER_NO_USABLE_RESULT_I18N_KEY,
  'Brioi completed the task without a result URL; administrator review is required':
    PROVIDER_NO_USABLE_RESULT_I18N_KEY,
  'Provider completed the task without a result URL; administrator review is required':
    PROVIDER_NO_USABLE_RESULT_I18N_KEY,
  'Provider returned an unknown task status; administrator review is required':
    PROVIDER_UNKNOWN_STATUS_I18N_KEY,
  'Brioi returned an unknown task status; administrator review is required':
    PROVIDER_UNKNOWN_STATUS_I18N_KEY,
  'SilkRoad returned an unknown task status; administrator review is required':
    PROVIDER_UNKNOWN_STATUS_I18N_KEY,
}

type TranslateFn = (key: string, options?: Record<string, unknown>) => string

/**
 * Localizes known machine-written task fail_reason strings for UI display.
 * Unknown messages pass through unless they still name a video upstream.
 */
export function localizeTaskFailReason(
  failReason: string,
  translate: TranslateFn
): string {
  const trimmed = failReason.trim()
  if (!trimmed) return failReason

  const deliveryMatch = trimmed.match(VIDEO_DELIVERY_FAILURE_PATTERN)
  if (deliveryMatch) {
    return translate(VIDEO_DELIVERY_FAILURE_I18N_KEY, {
      taskId: deliveryMatch[1],
    })
  }

  const mappedKey = GENERIC_FAIL_REASON_KEYS[trimmed]
  if (mappedKey) {
    return translate(mappedKey)
  }

  if (VIDEO_UPSTREAM_BRAND_PATTERN.test(trimmed)) {
    return translate('Video generation failed')
  }

  return failReason
}
