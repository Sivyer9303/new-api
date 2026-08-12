/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { api } from '@/lib/api'

export type VideoStorageStatus = {
  driver: string
  storage_enabled: boolean
  retention_days: number
  ready: boolean
  ready_error: string
  r2: {
    configured: boolean
    bucket: string
    input_ttl_hours: number
    usage_bytes: number
    quota_bytes: number
    soft_limit_bytes: number
    soft_limit_ratio: number
    upload_blocked: boolean
    checked_at: number
    last_error: string
  }
}

export async function fetchVideoStorageStatus(): Promise<VideoStorageStatus> {
  const res = await api.get<{
    success: boolean
    message?: string
    data?: VideoStorageStatus
  }>('/api/video/storage-status')
  if (!res.data.success || !res.data.data) {
    throw new Error(res.data.message || 'Failed to load storage status')
  }
  return res.data.data
}

export async function refreshVideoStorageUsage(): Promise<void> {
  const res = await api.post<{ success: boolean; message?: string }>(
    '/api/video/storage-status/refresh'
  )
  if (!res.data.success) {
    throw new Error(res.data.message || 'Failed to refresh storage usage')
  }
}
