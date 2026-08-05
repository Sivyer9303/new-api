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
    const res = await axios.get<VideoFetchResponse>(
      `/v1/video/generations/${encodeURIComponent(taskId)}`,
      {
        headers: {
          Authorization: `Bearer ${bearerKey(tokenKey)}`,
        },
      }
    )
    return res.data
  } catch (err) {
    throw axiosErrorMessage(err, 'Failed to fetch video task')
  }
}
