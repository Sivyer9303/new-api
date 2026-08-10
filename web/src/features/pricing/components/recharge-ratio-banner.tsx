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
import { BadgePercent } from 'lucide-react'
import { Trans, useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'

import { SITE_RECHARGE_RATIO } from '../lib/site-recharge-display'

export interface RechargeRatioBannerProps {
  className?: string
}

export function RechargeRatioBanner(props: RechargeRatioBannerProps) {
  const { t } = useTranslation()
  const ratioLabel = `1:${SITE_RECHARGE_RATIO}`

  return (
    <div
      role='status'
      className={cn(
        'relative overflow-hidden rounded-xl border border-amber-300 bg-amber-50 px-3 py-3.5 sm:px-4 sm:py-4',
        'dark:border-amber-500/40 dark:bg-amber-500/15',
        props.className
      )}
    >
      <div className='relative flex items-start gap-3 sm:items-center'>
        <div className='flex size-12 shrink-0 items-center justify-center rounded-full bg-amber-500 text-white shadow-sm dark:bg-amber-400 dark:text-amber-950'>
          <BadgePercent className='size-6' aria-hidden />
        </div>
        <div className='min-w-0 flex-1 space-y-1.5'>
          <p className='text-lg font-semibold text-amber-950 sm:text-xl dark:text-amber-50'>
            <Trans
              i18nKey='Site quota conversion ratio is <ratio>{{ratio}}</ratio>'
              values={{ ratio: ratioLabel }}
              components={{
                ratio: (
                  <span className='font-bold text-red-600 dark:text-red-400' />
                ),
              }}
            />
          </p>
          <p className='text-base leading-relaxed text-amber-900/85 sm:text-lg dark:text-amber-100/85'>
            {t(
              'Top-ups and deductions both use this ratio. Strikethrough is the display price; the colored price is the real price after conversion.'
            )}
          </p>
        </div>
        <span className='hidden shrink-0 rounded-lg bg-amber-500 px-3.5 py-2 font-mono text-lg font-bold tracking-tight text-white shadow-sm sm:inline-flex dark:bg-amber-400 dark:text-amber-950'>
          {ratioLabel}
        </span>
      </div>
    </div>
  )
}
