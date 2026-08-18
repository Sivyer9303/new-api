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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import i18next from 'i18next'
import { toast } from 'sonner'

import { updateSystemOption, updateVideoProviderOption } from '../api'
import type {
  UpdateOptionRequest,
  UpdateVideoProviderOptionRequest,
} from '../types'

// Configuration keys that require status refresh
const STATUS_RELATED_KEYS = new Set([
  'HeaderNavModules',
  'SidebarModulesAdmin',
  'Notice',
  'LogConsumeEnabled',
  'QuotaPerUnit',
  'USDExchangeRate',
  'DisplayInCurrencyEnabled',
  'DisplayTokenStatEnabled',
  'general_setting.quota_display_type',
  'general_setting.custom_currency_symbol',
  'general_setting.custom_currency_exchange_rate',
  'oidc.display_name',
  'console_setting.custom_pages',
  'console_setting.availability_monitor_enabled',
  'console_setting.availability_monitor_visibility',
  'console_setting.availability_monitor_refresh_interval',
  // Turnstile is read from /api/status on lottery / login pages
  'TurnstileCheckEnabled',
  'TurnstileSiteKey',
  'TurnstileSecretKey',
])

function isVideoToolRelatedKey(key: string) {
  return (
    key === 'video_setting.enabled' ||
    key.startsWith('silkroad_setting.') ||
    key.startsWith('brioi_setting.') ||
    key.startsWith('compatvideo_setting.') ||
    key.startsWith('aistarslab_setting.')
  )
}

export function useUpdateOption() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (request: UpdateOptionRequest) => updateSystemOption(request),
    onSuccess: (data, variables) => {
      if (data.success) {
        // Always refresh system-options
        queryClient.invalidateQueries({ queryKey: ['system-options'] })

        // If updating frontend-display-related config, also refresh status
        if (
          STATUS_RELATED_KEYS.has(variables.key) ||
          isVideoToolRelatedKey(variables.key)
        ) {
          queryClient.invalidateQueries({ queryKey: ['status'] })
          try {
            window.localStorage.removeItem('status')
          } catch {
            /* empty */
          }
        }
        if (isVideoToolRelatedKey(variables.key)) {
          queryClient.invalidateQueries({ queryKey: ['video-tool-config'] })
        }

        // Backend may return a non-empty message on success as a warning
        // (e.g. per_second billing mode bound to a model without a SilkRoad
        // profile). Surface it prominently instead of the generic success toast.
        if (data.message) {
          toast.warning(data.message, { duration: 10000 })
        } else {
          toast.success(i18next.t('Setting updated successfully'))
        }
      } else {
        toast.error(data.message || i18next.t('Failed to update setting'))
      }
    },
    onError: (error: Error) => {
      toast.error(error.message || i18next.t('Failed to update setting'))
    },
  })
}

export function useUpdateVideoProviderOption() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (request: UpdateVideoProviderOptionRequest) =>
      updateVideoProviderOption(request),
    onSuccess: (data) => {
      if (!data.success) return
      queryClient.invalidateQueries({ queryKey: ['system-options'] })
      queryClient.invalidateQueries({ queryKey: ['status'] })
      queryClient.invalidateQueries({ queryKey: ['video-tool-config'] })
      try {
        window.localStorage.removeItem('status')
      } catch {
        /* empty */
      }
    },
  })
}
