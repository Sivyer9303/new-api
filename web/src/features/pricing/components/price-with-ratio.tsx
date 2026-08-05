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
import { cn } from '@/lib/utils'

import { formatPriceWithSiteRatio } from '../lib/site-recharge-display'

export interface PriceWithRatioProps {
  /** Upstream already-formatted list price (e.g. from formatPrice). */
  value: string
  className?: string
  originalClassName?: string
  actualClassName?: string
}

/**
 * Display-only dual price: strikethrough list price + site recharge actual.
 */
export function PriceWithRatio(props: PriceWithRatioProps) {
  const parts = formatPriceWithSiteRatio(props.value)

  if (!parts.scalable) {
    return (
      <span
        className={cn(
          'text-amber-700 dark:text-amber-300 font-mono font-semibold tabular-nums',
          props.actualClassName,
          props.className
        )}
      >
        {parts.original}
      </span>
    )
  }

  return (
    <span
      className={cn(
        'inline-flex flex-wrap items-baseline gap-1.5 font-mono tabular-nums',
        props.className
      )}
    >
      <span
        className={cn(
          'text-muted-foreground/70 line-through',
          props.originalClassName
        )}
      >
        {parts.original}
      </span>
      <span
        className={cn(
          'font-semibold text-amber-700 dark:text-amber-300',
          props.actualClassName
        )}
      >
        {parts.actual}
      </span>
    </span>
  )
}
