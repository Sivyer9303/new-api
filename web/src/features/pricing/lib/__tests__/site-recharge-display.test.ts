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

import {
  SITE_RECHARGE_RATIO,
  formatPriceWithSiteRatio,
} from '../site-recharge-display'

describe('formatPriceWithSiteRatio', () => {
  test('keeps the site ratio at 10', () => {
    assert.equal(SITE_RECHARGE_RATIO, 10)
  })

  test('scales a dollar list price to one tenth', () => {
    assert.deepEqual(formatPriceWithSiteRatio('$10'), {
      original: '$10',
      actual: '$1',
      scalable: true,
    })
  })

  test('scales fractional prices', () => {
    assert.deepEqual(formatPriceWithSiteRatio('$12.5'), {
      original: '$12.5',
      actual: '$1.25',
      scalable: true,
    })
  })

  test('preserves currency symbols other than dollar', () => {
    assert.deepEqual(formatPriceWithSiteRatio('¥70'), {
      original: '¥70',
      actual: '¥7',
      scalable: true,
    })
  })

  test('does not scale placeholder dashes', () => {
    assert.deepEqual(formatPriceWithSiteRatio('-'), {
      original: '-',
      actual: '-',
      scalable: false,
    })
  })
})
