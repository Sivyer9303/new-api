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
  image_roles: VideoMediaRole[]
}

export type VideoMediaType = 'image' | 'audio' | 'video'
export type VideoMediaRole = 'reference' | 'first_frame' | 'last_frame'

export type PublicMediaLimits = {
  min_items: number
  max_items: number
  accepted_types: VideoMediaType[]
  allowed_roles: VideoMediaRole[]
  allow_audio: boolean
}

export type PublicProfile = {
  id: string
  label: string
  exact_models: string[]
  model_prefixes: string[]
  durations: PublicOption[]
  resolutions: PublicOption[]
  aspect_ratios: PublicOption[]
  generation_types: string[]
  media: PublicMediaLimits
  media_limits: Record<string, PublicMediaLimits>
}

export type VideoProviderConfig = {
  id: string
  label: string
  groups: string[]
  generation_types: PublicGenerationType[]
  profiles: PublicProfile[]
  default_profile_id: string
  strict_model_matching: boolean
}

export type VideoToolConfig = {
  version: number
  enabled: boolean
  providers: VideoProviderConfig[]
  provider_by_group: Record<string, string>
  video_tool_groups: string[]
}

export type VideoToolModel = {
  id: string
  profile_model: string
  provider_id?: string
}

export type VideoToolModelDiscovery = {
  group: string
  resolved_groups: string[]
  provider?: string
  reason?: string
  models: VideoToolModel[]
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
