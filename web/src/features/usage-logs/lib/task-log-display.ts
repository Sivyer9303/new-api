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
import type {
  TaskLog,
  TaskLogProperties,
  TaskMediaSnapshot,
  TaskRequestSnapshot,
} from '../types'
import { taskActionMapper } from './mappers'

export const GENERATION_TYPE_LABELS: Record<string, string> = {
  text2video: 'Text to video',
  image2video: 'Image reference',
  multi_image: 'Multi-image reference',
  first_frame: 'First frame',
  start_end: 'First & last frame',
  reference_videos: 'Reference video',
  reference_audio: 'Reference audio',
}

const MEDIA_TYPE_LABELS: Record<string, string> = {
  image: 'Image',
  video: 'Video',
  audio: 'Audio',
}

const MEDIA_ROLE_LABELS: Record<string, string> = {
  reference: 'Reference',
  first_frame: 'First frame',
  last_frame: 'Last frame',
  first: 'First frame',
  last: 'Last frame',
}

export type TaskRequestField = {
  labelKey: string
  value: string
  translateValue?: boolean
}

export type TaskRequestMediaItem = {
  key: string
  typeKey: string
  roleKey?: string
}

export function parseTaskLogProperties(
  properties: TaskLog['properties']
): TaskLogProperties | undefined {
  if (!properties) {
    return undefined
  }
  if (typeof properties === 'string') {
    try {
      const parsed: unknown = JSON.parse(properties)
      if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
        return undefined
      }
      return parsed as TaskLogProperties
    } catch {
      return undefined
    }
  }
  return properties
}

export function getTaskRequestSnapshot(
  log: Pick<TaskLog, 'properties'>
): TaskRequestSnapshot | undefined {
  return parseTaskLogProperties(log.properties)?.request
}

export function taskLogActionLabel(
  log: Pick<TaskLog, 'action' | 'properties'>
): string {
  const generationType = getTaskRequestSnapshot(log)?.generation_type
  if (generationType) {
    const mapped = GENERATION_TYPE_LABELS[generationType]
    if (mapped) {
      return mapped
    }
  }
  return taskActionMapper.getLabel(log.action, log.action)
}

export function getTaskRequestFields(
  log: Pick<TaskLog, 'properties'>
): TaskRequestField[] {
  const properties = parseTaskLogProperties(log.properties)
  const request = properties?.request
  if (!request) {
    return []
  }

  const fields: TaskRequestField[] = []
  const model = (properties?.origin_model_name || request.model || '').trim()
  if (model) {
    fields.push({ labelKey: 'Model', value: model })
  }
  const generationType = request.generation_type?.trim()
  if (generationType) {
    fields.push({
      labelKey: 'Generation type',
      value: GENERATION_TYPE_LABELS[generationType] || generationType,
      translateValue: Boolean(GENERATION_TYPE_LABELS[generationType]),
    })
  }
  const seconds =
    request.seconds?.trim() ||
    (request.duration && request.duration > 0 ? String(request.duration) : '')
  if (seconds) {
    fields.push({ labelKey: 'Seconds', value: seconds })
  }
  if (request.resolution?.trim()) {
    fields.push({ labelKey: 'Resolution', value: request.resolution.trim() })
  }
  if (request.aspect_ratio?.trim()) {
    fields.push({
      labelKey: 'Aspect ratio',
      value: request.aspect_ratio.trim(),
    })
  }
  if (request.size?.trim()) {
    fields.push({ labelKey: 'Size', value: request.size.trim() })
  }
  if (request.prompt) {
    fields.push({ labelKey: 'Prompt', value: request.prompt })
  }
  return fields
}

export function getTaskRequestMedia(
  log: Pick<TaskLog, 'properties'>
): TaskRequestMediaItem[] {
  const media = getTaskRequestSnapshot(log)?.media
  if (!media || media.length === 0) {
    return []
  }
  const seen = new Map<string, number>()
  return media.map((item) => {
    const labeled = mediaItemLabel(item)
    const base = `${labeled.typeKey}:${labeled.roleKey ?? ''}`
    const next = (seen.get(base) ?? 0) + 1
    seen.set(base, next)
    return { ...labeled, key: `${base}:${next}` }
  })
}

export function hasTaskRequestSnapshot(
  log: Pick<TaskLog, 'properties'>
): boolean {
  return (
    getTaskRequestFields(log).length > 0 || getTaskRequestMedia(log).length > 0
  )
}

export function getTaskRequestCopyJSON(
  log: Pick<TaskLog, 'properties'>
): string {
  const properties = parseTaskLogProperties(log.properties)
  const request = properties?.request
  if (!request) {
    return '{}'
  }
  const payload: Record<string, unknown> = {}
  const model = (properties?.origin_model_name || request.model || '').trim()
  if (model) {
    payload.model = model
  }
  if (request.generation_type?.trim()) {
    payload.generation_type = request.generation_type.trim()
  }
  if (request.prompt) {
    payload.prompt = request.prompt
  }
  if (request.seconds?.trim()) {
    payload.seconds = request.seconds.trim()
  } else if (request.duration && request.duration > 0) {
    payload.duration = request.duration
  }
  if (request.resolution?.trim()) {
    payload.resolution = request.resolution.trim()
  }
  if (request.aspect_ratio?.trim()) {
    payload.aspect_ratio = request.aspect_ratio.trim()
  }
  if (request.size?.trim()) {
    payload.size = request.size.trim()
  }
  if (request.media && request.media.length > 0) {
    payload.media = request.media.map((item) => {
      const summary: TaskMediaSnapshot = { type: item.type }
      if (item.role) {
        summary.role = item.role
      }
      return summary
    })
  }
  return JSON.stringify(payload, null, 2)
}

function mediaItemLabel(
  item: TaskMediaSnapshot
): Omit<TaskRequestMediaItem, 'key'> {
  const typeKey = MEDIA_TYPE_LABELS[item.type] || item.type
  if (!item.role) {
    return { typeKey }
  }
  return {
    typeKey,
    roleKey: MEDIA_ROLE_LABELS[item.role] || item.role,
  }
}
