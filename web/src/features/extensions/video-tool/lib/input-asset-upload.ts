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

export type VideoInputAssetKind = 'image' | 'audio' | 'video'

export type VideoInputUploadStatus =
  | 'pending'
  | 'uploading'
  | 'ready'
  | 'failed'

export type VideoInputUploadResult = {
  assetId: string
  url: string
  expiresAt: number
  kind: VideoInputAssetKind
  contentType: string
  size: number
}

type PresignResponse = {
  success: boolean
  message?: string
  data?: {
    asset_id: string
    upload_url: string
    upload_headers?: Record<string, string>
    expires_at: number
  }
}

type CompleteResponse = {
  success: boolean
  message?: string
  data?: {
    asset_id: string
    url: string
    expires_at: number
    kind: string
    content_type: string
    size: number
  }
}

function apiErrorMessage(err: unknown, fallback: string): Error {
  if (axios.isAxiosError(err)) {
    const data = err.response?.data as
      | { error?: { message?: string }; message?: string }
      | undefined
    const msg = data?.error?.message || data?.message || err.message || fallback
    return new Error(msg)
  }
  if (err instanceof Error) return err
  return new Error(fallback)
}

/** Resolve a browser File MIME into a whitelist type accepted by the backend. */
export function resolveVideoInputContentType(
  file: File,
  kind: VideoInputAssetKind
): string {
  const raw = (file.type || '').toLowerCase().trim()
  const name = file.name.toLowerCase()
  if (kind === 'image') {
    if (raw === 'image/jpg' || raw === 'image/jpeg' || name.endsWith('.jpg') || name.endsWith('.jpeg')) {
      return 'image/jpeg'
    }
    if (raw === 'image/png' || name.endsWith('.png')) return 'image/png'
    if (raw === 'image/webp' || name.endsWith('.webp')) return 'image/webp'
    if (raw === 'image/gif' || name.endsWith('.gif')) return 'image/gif'
    if (raw.startsWith('image/')) return raw
    throw new Error('Unsupported image type')
  }
  if (kind === 'audio') {
    if (
      raw === 'audio/mpeg' ||
      raw === 'audio/mp3' ||
      name.endsWith('.mp3')
    ) {
      return 'audio/mpeg'
    }
    if (
      raw === 'audio/wav' ||
      raw === 'audio/x-wav' ||
      raw === 'audio/wave' ||
      name.endsWith('.wav')
    ) {
      return 'audio/wav'
    }
    throw new Error('Unsupported audio type')
  }
  if (raw === 'video/mp4' || name.endsWith('.mp4')) return 'video/mp4'
  if (raw === 'video/quicktime' || name.endsWith('.mov')) {
    return 'video/quicktime'
  }
  throw new Error('Unsupported video type')
}

export function isVideoInputExpired(
  expiresAt: number | undefined,
  nowSec = Math.floor(Date.now() / 1000)
): boolean {
  if (expiresAt == null || expiresAt <= 0) return true
  // Small skew so submit does not race the exact expiry second.
  return expiresAt <= nowSec + 5
}

export function collectReadyHttpsSources(
  items: Array<{
    uploadStatus: VideoInputUploadStatus
    sourceUrl?: string
    expiresAt?: number
  }>
):
  | { ok: true; urls: string[] }
  | { ok: false; reason: 'uploading' | 'failed' | 'expired' | 'missing' } {
  const urls: string[] = []
  for (const item of items) {
    if (item.uploadStatus === 'pending' || item.uploadStatus === 'uploading') {
      return { ok: false, reason: 'uploading' }
    }
    if (item.uploadStatus === 'failed') {
      return { ok: false, reason: 'failed' }
    }
    if (!item.sourceUrl || !/^https:\/\//i.test(item.sourceUrl)) {
      return { ok: false, reason: 'missing' }
    }
    if (isVideoInputExpired(item.expiresAt)) {
      return { ok: false, reason: 'expired' }
    }
    urls.push(item.sourceUrl)
  }
  return { ok: true, urls }
}

function putWithProgress(
  uploadUrl: string,
  file: File,
  headers: Record<string, string>,
  onProgress?: (percent: number) => void,
  signal?: AbortSignal
): Promise<void> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open('PUT', uploadUrl)
    for (const [name, value] of Object.entries(headers)) {
      if (!name || value == null) continue
      // Host/content-length are controlled by the browser.
      if (/^(host|content-length)$/i.test(name)) continue
      xhr.setRequestHeader(name, value)
    }
    xhr.upload.onprogress = (event) => {
      if (!event.lengthComputable || !onProgress) return
      onProgress(Math.min(100, Math.round((event.loaded / event.total) * 100)))
    }
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        onProgress?.(100)
        resolve()
        return
      }
      reject(new Error(`Upload failed with status ${xhr.status}`))
    }
    xhr.onerror = () => reject(new Error('Upload failed'))
    xhr.onabort = () => reject(new Error('Upload aborted'))
    if (signal) {
      if (signal.aborted) {
        reject(new Error('Upload aborted'))
        return
      }
      signal.addEventListener(
        'abort',
        () => {
          xhr.abort()
        },
        { once: true }
      )
    }
    xhr.send(file)
  })
}

export async function uploadVideoInputAsset(
  file: File,
  kind: VideoInputAssetKind,
  options?: {
    onProgress?: (percent: number) => void
    signal?: AbortSignal
  }
): Promise<VideoInputUploadResult> {
  const contentType = resolveVideoInputContentType(file, kind)
  try {
    const presignRes = await api.post<PresignResponse>(
      '/api/video/input-assets/presign',
      {
        kind,
        content_type: contentType,
        size: file.size,
      },
      { signal: options?.signal }
    )
    if (!presignRes.data.success || !presignRes.data.data) {
      throw new Error(presignRes.data.message || 'Failed to prepare upload')
    }
    const presign = presignRes.data.data
    const uploadHeaders: Record<string, string> = {
      ...(presign.upload_headers || {}),
      'Content-Type': contentType,
    }
    await putWithProgress(
      presign.upload_url,
      file,
      uploadHeaders,
      options?.onProgress,
      options?.signal
    )
    const completeRes = await api.post<CompleteResponse>(
      `/api/video/input-assets/${encodeURIComponent(presign.asset_id)}/complete`,
      {},
      { signal: options?.signal }
    )
    if (!completeRes.data.success || !completeRes.data.data) {
      throw new Error(completeRes.data.message || 'Failed to finalize upload')
    }
    const complete = completeRes.data.data
    return {
      assetId: complete.asset_id,
      url: complete.url,
      expiresAt: complete.expires_at,
      kind: kind,
      contentType: complete.content_type || contentType,
      size: complete.size || file.size,
    }
  } catch (err) {
    throw apiErrorMessage(err, 'Failed to upload media')
  }
}

export async function deleteVideoInputAsset(assetId: string): Promise<void> {
  if (!assetId) return
  try {
    await api.delete(
      `/api/video/input-assets/${encodeURIComponent(assetId)}`
    )
  } catch {
    // Best-effort cleanup; TTL cleanup covers orphans.
  }
}
