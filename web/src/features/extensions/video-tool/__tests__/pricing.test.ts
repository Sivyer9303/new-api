/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { estimateVideoPrice } from '../lib/pricing'

describe('video price estimation', () => {
  test('does not multiply fixed-price models by duration', () => {
    const estimate = estimateVideoPrice({
      modelPrice: 0.25,
      billingMode: 'fixed',
      quotaType: 2,
      durationSeconds: 12,
      groupRatio: 2,
    })

    assert.equal(estimate?.billingMode, 'fixed')
    assert.equal(estimate?.usd, 0.5)
  })

  test('multiplies per-second models by validated duration', () => {
    const estimate = estimateVideoPrice({
      modelPrice: 0.25,
      billingMode: 'per_second',
      durationSeconds: 12,
      groupRatio: 2,
    })

    assert.equal(estimate?.billingMode, 'per_second')
    assert.equal(estimate?.usd, 6)
  })

  test('rejects a non-positive model price', () => {
    assert.equal(
      estimateVideoPrice({
        modelPrice: 0,
        durationSeconds: 8,
        groupRatio: 1,
      }),
      null
    )
  })

  test('preserves a configured free group ratio', () => {
    const estimate = estimateVideoPrice({
      modelPrice: 0.25,
      billingMode: 'per_second',
      durationSeconds: 8,
      groupRatio: 0,
    })

    assert.equal(estimate?.usd, 0)
    assert.equal(estimate?.ratio, 0)
  })

  test('does not claim a fixed estimate for tiered expressions', () => {
    assert.equal(
      estimateVideoPrice({
        modelPrice: 0.25,
        billingMode: 'tiered_expr',
        durationSeconds: 8,
        groupRatio: 1,
      }),
      null
    )
  })
})
