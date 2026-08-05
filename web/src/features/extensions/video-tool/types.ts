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
  images_min: number
  images_max: number
}

export type PublicProfile = {
  id: string
  label: string
  model_prefixes: string[]
  durations: PublicOption[]
  aspect_ratios: PublicOption[]
  generation_types: PublicGenerationType[]
}

export type VideoToolConfig = {
  enabled: boolean
  video_tool_groups: string[]
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
