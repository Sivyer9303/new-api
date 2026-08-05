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
import { describe, expect, it } from 'vitest'

import {
  SITE_RECHARGE_RATIO,
  formatPriceWithSiteRatio,
} from '../site-recharge-display'

describe('formatPriceWithSiteRatio', () => {
  it('keeps the site ratio at 10', () => {
    expect(SITE_RECHARGE_RATIO).toBe(10)
  })

  it('scales a dollar list price to one tenth', () => {
    expect(formatPriceWithSiteRatio('$10')).toEqual({
      original: '$10',
      actual: '$1',
      scalable: true,
    })
  })

  it('scales fractional prices', () => {
    expect(formatPriceWithSiteRatio('$12.5')).toEqual({
      original: '$12.5',
      actual: '$1.25',
      scalable: true,
    })
  })

  it('preserves currency symbols other than dollar', () => {
    expect(formatPriceWithSiteRatio('¥70')).toEqual({
      original: '¥70',
      actual: '¥7',
      scalable: true,
    })
  })

  it('does not scale placeholder dashes', () => {
    expect(formatPriceWithSiteRatio('-')).toEqual({
      original: '-',
      actual: '-',
      scalable: false,
    })
  })
})
