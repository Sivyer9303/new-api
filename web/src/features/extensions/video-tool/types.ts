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
export type PublicOption = {
  label: string
  value: string
  upstream_key: string
  sort: number
}

export type PublicGenerationType = {
  label: string
  value: string
  sort: number
  require_ref_model: boolean
  require_audio: boolean
  allow_audio: boolean
  images_min: number
  images_max: number
}

export type PublicProfile = {
  id: string
  label: string
  model_prefixes: string[]
  durations: PublicOption[]
  aspect_ratios: PublicOption[]
}

export type VideoToolConfig = {
  enabled: boolean
  video_tool_groups: string[]
  generation_types: PublicGenerationType[]
  profiles: PublicProfile[]
}

export type VideoTaskStatus =
  | 'queued'
  | 'submitted'
  | 'in_progress'
  | 'completed'
  | 'success'
  | 'failed'
  | 'failure'
  | string

export type VideoSubmitResponse = {
  id?: string
  task_id?: string
  status?: VideoTaskStatus
}

export type VideoFetchResponse = {
  id?: string
  task_id?: string
  status?: VideoTaskStatus
  progress?: number | string
  video_url?: string
  url?: string
  result_url?: string
  fail_reason?: string
  error?: { message?: string }
}

/** Fixed Seedance / Dreamina generation modes (mirrors backend hardcoding). */
export const HARDCODED_GENERATION_TYPES: PublicGenerationType[] = [
  {
    label: 'Text to video',
    value: 'text2video',
    sort: 1,
    require_ref_model: false,
    require_audio: false,
    allow_audio: false,
    images_min: 0,
    images_max: 0,
  },
  {
    label: 'Image to video',
    value: 'image2video',
    sort: 2,
    require_ref_model: true,
    require_audio: false,
    allow_audio: false,
    images_min: 1,
    images_max: 1,
  },
  {
    label: 'Multi-image reference',
    value: 'multi_image',
    sort: 3,
    require_ref_model: true,
    require_audio: false,
    allow_audio: false,
    images_min: 2,
    images_max: 9,
  },
  {
    label: 'First & last frame',
    value: 'start_end',
    sort: 4,
    require_ref_model: true,
    require_audio: false,
    allow_audio: false,
    images_min: 2,
    images_max: 2,
  },
  {
    label: 'Reference audio',
    value: 'reference_audio',
    sort: 5,
    require_ref_model: true,
    require_audio: true,
    allow_audio: true,
    images_min: 0,
    images_max: 9,
  },
]
