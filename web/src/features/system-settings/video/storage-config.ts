/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { z } from 'zod'

export const VIDEO_STORAGE_DRIVERS = ['local', 'r2'] as const

export type VideoStorageDriver = (typeof VIDEO_STORAGE_DRIVERS)[number]

export const DEFAULT_LOCAL_RETENTION_DAYS = 7
export const DEFAULT_R2_RETENTION_DAYS = 3
export const MIN_RETENTION_DAYS = 1
export const MAX_RETENTION_DAYS = 30

export const DEFAULT_R2_RESULT_PREFIX = 'videos/'
export const DEFAULT_R2_INPUT_PREFIX = 'video-inputs/'
export const DEFAULT_R2_RESULT_PRESIGN_TTL_SECONDS = 900
export const DEFAULT_R2_INPUT_PRESIGN_TTL_SECONDS = 21600
export const DEFAULT_R2_INPUT_TTL_HOURS = 24
export const MIN_PRESIGN_TTL_SECONDS = 60
export const MAX_PRESIGN_TTL_SECONDS = 7 * 24 * 3600
export const MIN_INPUT_TTL_HOURS = 1
export const MAX_INPUT_TTL_HOURS = 30 * 24

/** Cloudflare's free storage allowance and the ratio at which uploads stop. */
export const R2_FREE_TIER_BYTES = 10 * 1024 ** 3
export const R2_SOFT_LIMIT_RATIO = 0.9

export const videoStorageSchema = z.object({
  driver: z.enum(VIDEO_STORAGE_DRIVERS),
  max_retry: z.coerce.number().int().min(1),
  local_dir: z.string().trim().min(1),
  ingest_node_name: z.string(),
  public_download_base_url: z.union([z.literal(''), z.string().trim().url()]),
  local_retention_days: z.coerce
    .number()
    .int()
    .min(MIN_RETENTION_DAYS)
    .max(MAX_RETENTION_DAYS),
  r2: z.object({
    account_id: z.string(),
    access_key_id: z.string(),
    secret_access_key: z.string(),
    api_token: z.string(),
    bucket: z.string(),
    endpoint: z.union([z.literal(''), z.string().trim().url()]),
    region: z.string(),
    result_prefix: z.string().trim().min(1),
    input_prefix: z.string().trim().min(1),
    retention_days: z.coerce
      .number()
      .int()
      .min(MIN_RETENTION_DAYS)
      .max(MAX_RETENTION_DAYS),
    result_presign_ttl_seconds: z.coerce
      .number()
      .int()
      .min(MIN_PRESIGN_TTL_SECONDS)
      .max(MAX_PRESIGN_TTL_SECONDS),
    input_presign_ttl_seconds: z.coerce
      .number()
      .int()
      .min(MIN_PRESIGN_TTL_SECONDS)
      .max(MAX_PRESIGN_TTL_SECONDS),
    input_ttl_hours: z.coerce
      .number()
      .int()
      .min(MIN_INPUT_TTL_HOURS)
      .max(MAX_INPUT_TTL_HOURS),
  }),
})

export type VideoStorageValues = z.infer<typeof videoStorageSchema>

export function defaultVideoStorageValues(): VideoStorageValues {
  return {
    driver: 'local',
    max_retry: 5,
    local_dir: 'data/videos',
    ingest_node_name: '',
    public_download_base_url: '',
    local_retention_days: DEFAULT_LOCAL_RETENTION_DAYS,
    r2: {
      account_id: '',
      access_key_id: '',
      secret_access_key: '',
      api_token: '',
      bucket: '',
      endpoint: '',
      region: 'auto',
      result_prefix: DEFAULT_R2_RESULT_PREFIX,
      input_prefix: DEFAULT_R2_INPUT_PREFIX,
      retention_days: DEFAULT_R2_RETENTION_DAYS,
      result_presign_ttl_seconds: DEFAULT_R2_RESULT_PRESIGN_TTL_SECONDS,
      input_presign_ttl_seconds: DEFAULT_R2_INPUT_PRESIGN_TTL_SECONDS,
      input_ttl_hours: DEFAULT_R2_INPUT_TTL_HOURS,
    },
  }
}

function readString(value: unknown, fallback: string): string {
  return typeof value === 'string' && value.trim() ? value : fallback
}

function readNumber(value: unknown, fallback: number, min: number): number {
  return typeof value === 'number' && Number.isFinite(value) && value >= min
    ? value
    : fallback
}

/**
 * Parses a stored `video_setting.storage` payload. Older rows only contain the
 * flat local-driver fields, so every R2 value falls back to its default.
 */
export function parseVideoStorage(raw: string): VideoStorageValues {
  const defaults = defaultVideoStorageValues()
  let value: Record<string, unknown>
  try {
    value = JSON.parse(raw) as Record<string, unknown>
  } catch {
    return defaults
  }
  const r2 = (value.r2 ?? {}) as Record<string, unknown>
  const driver = value.driver === 'r2' ? 'r2' : 'local'

  return {
    driver,
    max_retry: readNumber(value.max_retry, defaults.max_retry, 1),
    local_dir: readString(value.local_dir, defaults.local_dir),
    ingest_node_name:
      typeof value.ingest_node_name === 'string' ? value.ingest_node_name : '',
    public_download_base_url:
      typeof value.public_download_base_url === 'string'
        ? value.public_download_base_url
        : '',
    local_retention_days: readNumber(
      value.local_retention_days,
      defaults.local_retention_days,
      MIN_RETENTION_DAYS
    ),
    r2: {
      account_id: typeof r2.account_id === 'string' ? r2.account_id : '',
      access_key_id:
        typeof r2.access_key_id === 'string' ? r2.access_key_id : '',
      secret_access_key:
        typeof r2.secret_access_key === 'string' ? r2.secret_access_key : '',
      api_token: typeof r2.api_token === 'string' ? r2.api_token : '',
      bucket: typeof r2.bucket === 'string' ? r2.bucket : '',
      endpoint: typeof r2.endpoint === 'string' ? r2.endpoint : '',
      region: readString(r2.region, defaults.r2.region),
      result_prefix: readString(r2.result_prefix, defaults.r2.result_prefix),
      input_prefix: readString(r2.input_prefix, defaults.r2.input_prefix),
      retention_days: readNumber(
        r2.retention_days,
        defaults.r2.retention_days,
        MIN_RETENTION_DAYS
      ),
      result_presign_ttl_seconds: readNumber(
        r2.result_presign_ttl_seconds,
        defaults.r2.result_presign_ttl_seconds,
        MIN_PRESIGN_TTL_SECONDS
      ),
      input_presign_ttl_seconds: readNumber(
        r2.input_presign_ttl_seconds,
        defaults.r2.input_presign_ttl_seconds,
        MIN_PRESIGN_TTL_SECONDS
      ),
      input_ttl_hours: readNumber(
        r2.input_ttl_hours,
        defaults.r2.input_ttl_hours,
        MIN_INPUT_TTL_HOURS
      ),
    },
  }
}

/** Serializes form values into the option payload the backend validates. */
export function serializeVideoStorage(values: VideoStorageValues): string {
  return JSON.stringify({
    driver: values.driver,
    max_retry: values.max_retry,
    local_dir: values.local_dir.trim(),
    ingest_node_name: values.ingest_node_name.trim(),
    public_download_base_url: values.public_download_base_url.trim(),
    local_retention_days: values.local_retention_days,
    r2: {
      account_id: values.r2.account_id.trim(),
      access_key_id: values.r2.access_key_id.trim(),
      secret_access_key: values.r2.secret_access_key.trim(),
      api_token: values.r2.api_token.trim(),
      bucket: values.r2.bucket.trim(),
      endpoint: values.r2.endpoint.trim(),
      region: values.r2.region.trim() || 'auto',
      result_prefix: values.r2.result_prefix.trim(),
      input_prefix: values.r2.input_prefix.trim(),
      retention_days: values.r2.retention_days,
      result_presign_ttl_seconds: values.r2.result_presign_ttl_seconds,
      input_presign_ttl_seconds: values.r2.input_presign_ttl_seconds,
      input_ttl_hours: values.r2.input_ttl_hours,
    },
  })
}

/** Retention that applies to the selected driver, for display and hints. */
export function activeRetentionDays(values: VideoStorageValues): number {
  return values.driver === 'r2'
    ? values.r2.retention_days
    : values.local_retention_days
}

export function formatStorageBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B'
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB']
  let value = bytes
  let unit = 0
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024
    unit += 1
  }
  return `${value.toFixed(unit === 0 ? 0 : 2)} ${units[unit]}`
}

export function usagePercent(usageBytes: number, quotaBytes: number): number {
  if (!Number.isFinite(usageBytes) || !Number.isFinite(quotaBytes)) return 0
  if (quotaBytes <= 0) return 0
  return Math.min(100, Math.max(0, (usageBytes / quotaBytes) * 100))
}
