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
import axios from 'axios'

import { api } from '@/lib/api'

import type {
  VideoFetchResponse,
  VideoSubmitResponse,
  VideoToolConfig,
} from './types'

function bearerKey(tokenKey: string): string {
  return tokenKey.startsWith('sk-') ? tokenKey : `sk-${tokenKey}`
}

function axiosErrorMessage(err: unknown, fallback: string): Error {
  if (axios.isAxiosError(err)) {
    const data = err.response?.data as
      | { error?: { message?: string }; message?: string }
      | undefined
    const msg =
      data?.error?.message || data?.message || err.message || fallback
    return new Error(msg)
  }
  if (err instanceof Error) return err
  return new Error(fallback)
}

export async function fetchVideoToolConfig(): Promise<{
  success: boolean
  message?: string
  data?: VideoToolConfig
}> {
  const res = await api.get('/api/silkroad/video-tool')
  return res.data
}

export async function fetchModelsWithTokenKey(tokenKey: string): Promise<
  string[]
> {
  try {
    const res = await axios.get<{ data?: Array<{ id?: string }> }>(
      '/v1/models',
      {
        headers: {
          Authorization: `Bearer ${bearerKey(tokenKey)}`,
        },
      }
    )
    const items = res.data?.data ?? []
    return items
      .map((m) => m.id?.trim() ?? '')
      .filter((id) => id.length > 0)
  } catch (err) {
    throw axiosErrorMessage(err, 'Failed to load models')
  }
}

export async function submitVideoGeneration(
  tokenKey: string,
  body: Record<string, unknown>
): Promise<VideoSubmitResponse> {
  try {
    const res = await axios.post<VideoSubmitResponse>(
      '/v1/video/generations',
      body,
      {
        headers: {
          Authorization: `Bearer ${bearerKey(tokenKey)}`,
          'Content-Type': 'application/json',
        },
      }
    )
    return res.data
  } catch (err) {
    throw axiosErrorMessage(err, 'Failed to submit video task')
  }
}

export async function fetchVideoGeneration(
  tokenKey: string,
  taskId: string
): Promise<VideoFetchResponse> {
  try {
    const res = await axios.get<unknown>(
      `/v1/video/generations/${encodeURIComponent(taskId)}`,
      {
        headers: {
          Authorization: `Bearer ${bearerKey(tokenKey)}`,
        },
      }
    )
    return normalizeVideoFetchResponse(res.data)
  } catch (err) {
    throw axiosErrorMessage(err, 'Failed to fetch video task')
  }
}

export async function fetchVideoContentBlob(
  tokenKey: string,
  taskId: string
): Promise<Blob> {
  try {
    const res = await axios.get<Blob>(
      `/v1/videos/${encodeURIComponent(taskId)}/content`,
      {
        headers: {
          Authorization: `Bearer ${bearerKey(tokenKey)}`,
        },
        responseType: 'blob',
      }
    )
    return res.data
  } catch (err) {
    throw axiosErrorMessage(err, 'Failed to load video preview')
  }
}

function isSiteVideoContentURL(value: string): boolean {
  return value.includes('/v1/videos/') && /\/content\/?$/.test(value)
}

function pickSiteContentURL(...values: unknown[]): string | undefined {
  for (const value of values) {
    if (typeof value !== 'string') continue
    const trimmed = value.trim()
    if (!trimmed) continue
    // Only accept this site's content proxy — never upstream CDN hosts.
    if (isSiteVideoContentURL(trimmed)) return trimmed
  }
  return undefined
}

function collectNestedURLs(
  node: unknown,
  depth = 0,
  out: string[] = []
): string[] {
  if (node == null || depth > 6) return out
  if (typeof node === 'string') {
    const trimmed = node.trim()
    if (!trimmed.startsWith('{') && !trimmed.startsWith('[')) return out
    try {
      return collectNestedURLs(JSON.parse(trimmed) as unknown, depth + 1, out)
    } catch {
      return out
    }
  }
  if (Array.isArray(node)) {
    for (const item of node) collectNestedURLs(item, depth + 1, out)
    return out
  }
  if (typeof node !== 'object') return out
  const obj = node as Record<string, unknown>
  for (const key of ['video_url', 'url', 'result_url'] as const) {
    const value = obj[key]
    if (typeof value === 'string' && value.trim()) out.push(value.trim())
  }
  for (const value of Object.values(obj)) {
    collectNestedURLs(value, depth + 1, out)
  }
  return out
}

/** Normalize both flat OpenAI-style and `{ code, data: TaskDto }` fetch payloads. */
export function normalizeVideoFetchResponse(raw: unknown): VideoFetchResponse {
  if (!raw || typeof raw !== 'object') {
    return {}
  }
  const root = raw as Record<string, unknown>
  const nested =
    root.data && typeof root.data === 'object' && !Array.isArray(root.data)
      ? (root.data as Record<string, unknown>)
      : null

  // Prefer nested TaskDto when present (GET /v1/video/generations/:id).
  const src =
    nested && (nested.status != null || nested.task_id != null) ? nested : root

  const status =
    typeof src.status === 'string'
      ? src.status
      : typeof root.status === 'string'
        ? root.status
        : undefined

  const progress = src.progress ?? root.progress
  const failReason =
    typeof src.fail_reason === 'string'
      ? src.fail_reason
      : typeof root.fail_reason === 'string'
        ? root.fail_reason
        : undefined

  const errorObj =
    src.error && typeof src.error === 'object'
      ? (src.error as { message?: string })
      : root.error && typeof root.error === 'object'
        ? (root.error as { message?: string })
        : failReason
          ? { message: failReason }
          : undefined

  const nestedURLs = collectNestedURLs(src.data)
  const siteURL = pickSiteContentURL(
    src.result_url,
    src.video_url,
    src.url,
    ...nestedURLs
  )

  return {
    id: typeof src.id === 'string' ? src.id : undefined,
    task_id:
      typeof src.task_id === 'string'
        ? src.task_id
        : typeof root.task_id === 'string'
          ? root.task_id
          : undefined,
    status,
    progress:
      typeof progress === 'string' || typeof progress === 'number'
        ? progress
        : undefined,
    // Never propagate upstream CDN URLs to the UI.
    video_url: siteURL,
    url: siteURL,
    result_url: siteURL,
    fail_reason: failReason,
    error: errorObj,
  }
}
