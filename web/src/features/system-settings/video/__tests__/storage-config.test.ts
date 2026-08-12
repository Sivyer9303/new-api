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
  activeRetentionDays,
  DEFAULT_LOCAL_RETENTION_DAYS,
  DEFAULT_R2_INPUT_PREFIX,
  DEFAULT_R2_RESULT_PREFIX,
  DEFAULT_R2_RETENTION_DAYS,
  defaultVideoStorageValues,
  formatStorageBytes,
  parseVideoStorage,
  R2_FREE_TIER_BYTES,
  R2_SOFT_LIMIT_RATIO,
  serializeVideoStorage,
  usagePercent,
  videoStorageSchema,
} from '../storage-config'

describe('video storage config', () => {
  test('fills R2 defaults when parsing a local-only payload', () => {
    const parsed = parseVideoStorage(
      '{"driver":"local","local_dir":"data/videos","max_retry":5,"ingest_node_name":"node-1","public_download_base_url":"https://video.example.com"}'
    )

    assert.equal(parsed.driver, 'local')
    assert.equal(parsed.ingest_node_name, 'node-1')
    assert.equal(parsed.local_retention_days, DEFAULT_LOCAL_RETENTION_DAYS)
    assert.equal(parsed.r2.retention_days, DEFAULT_R2_RETENTION_DAYS)
    assert.equal(parsed.r2.result_prefix, DEFAULT_R2_RESULT_PREFIX)
    assert.equal(parsed.r2.input_prefix, DEFAULT_R2_INPUT_PREFIX)
    assert.equal(parsed.r2.result_presign_ttl_seconds, 900)
  })

  test('round-trips an R2 payload through serialize and parse', () => {
    const values = defaultVideoStorageValues()
    values.driver = 'r2'
    values.public_download_base_url = 'https://video.example.com'
    values.r2 = {
      ...values.r2,
      account_id: 'acct',
      access_key_id: 'ak',
      secret_access_key: 'sk',
      api_token: 'token',
      bucket: 'videos',
      retention_days: 3,
    }

    const parsed = parseVideoStorage(serializeVideoStorage(values))

    assert.deepEqual(parsed, values)
  })

  test('falls back to defaults for unparsable payloads', () => {
    assert.deepEqual(parseVideoStorage('not-json'), defaultVideoStorageValues())
  })

  test('reports the retention of the selected driver', () => {
    const values = defaultVideoStorageValues()
    values.local_retention_days = 7
    values.r2.retention_days = 3

    assert.equal(activeRetentionDays(values), 7)
    assert.equal(activeRetentionDays({ ...values, driver: 'r2' }), 3)
  })

  test('rejects retention and presign values outside the allowed range', () => {
    const values = defaultVideoStorageValues()

    assert.equal(videoStorageSchema.safeParse(values).success, true)
    assert.equal(
      videoStorageSchema.safeParse({ ...values, local_retention_days: 0 })
        .success,
      false
    )
    assert.equal(
      videoStorageSchema.safeParse({ ...values, local_retention_days: 31 })
        .success,
      false
    )
    assert.equal(
      videoStorageSchema.safeParse({
        ...values,
        r2: { ...values.r2, result_presign_ttl_seconds: 30 },
      }).success,
      false
    )
    assert.equal(
      videoStorageSchema.safeParse({ ...values, driver: 's3' }).success,
      false
    )
  })

  test('describes the free tier soft limit for the usage card', () => {
    assert.equal(R2_FREE_TIER_BYTES, 10 * 1024 ** 3)
    assert.equal(R2_SOFT_LIMIT_RATIO, 0.9)
    assert.equal(formatStorageBytes(R2_FREE_TIER_BYTES), '10.00 GiB')
    assert.equal(formatStorageBytes(0), '0 B')
    assert.equal(usagePercent(R2_FREE_TIER_BYTES * 0.9, R2_FREE_TIER_BYTES), 90)
    assert.equal(usagePercent(1, 0), 0)
  })
})
