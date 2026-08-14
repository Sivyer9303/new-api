/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { z } from 'zod'

export const DEFAULT_MAX_IMAGE_UPLOAD_MB = 10
export const DEFAULT_MAX_AUDIO_UPLOAD_MB = 24
export const DEFAULT_MAX_VIDEO_UPLOAD_MB = 50
export const MIN_UPLOAD_LIMIT_MB = 1
export const MAX_UPLOAD_LIMIT_MB = 200

export const videoUploadLimitsSchema = z.object({
  max_image_mb: z.coerce
    .number()
    .int()
    .min(MIN_UPLOAD_LIMIT_MB)
    .max(MAX_UPLOAD_LIMIT_MB),
  max_audio_mb: z.coerce
    .number()
    .int()
    .min(MIN_UPLOAD_LIMIT_MB)
    .max(MAX_UPLOAD_LIMIT_MB),
  max_video_mb: z.coerce
    .number()
    .int()
    .min(MIN_UPLOAD_LIMIT_MB)
    .max(MAX_UPLOAD_LIMIT_MB),
})

export type VideoUploadLimitsValues = z.infer<typeof videoUploadLimitsSchema>

export function defaultVideoUploadLimitsValues(): VideoUploadLimitsValues {
  return {
    max_image_mb: DEFAULT_MAX_IMAGE_UPLOAD_MB,
    max_audio_mb: DEFAULT_MAX_AUDIO_UPLOAD_MB,
    max_video_mb: DEFAULT_MAX_VIDEO_UPLOAD_MB,
  }
}

export function parseVideoUploadLimitsJson(
  raw: string | undefined | null
): VideoUploadLimitsValues {
  const defaults = defaultVideoUploadLimitsValues()
  if (!raw?.trim()) return defaults
  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>
    const result = videoUploadLimitsSchema.safeParse({
      max_image_mb: parsed.max_image_mb ?? defaults.max_image_mb,
      max_audio_mb: parsed.max_audio_mb ?? defaults.max_audio_mb,
      max_video_mb: parsed.max_video_mb ?? defaults.max_video_mb,
    })
    return result.success ? result.data : defaults
  } catch {
    return defaults
  }
}

export function serializeVideoUploadLimits(
  values: VideoUploadLimitsValues
): string {
  return JSON.stringify({
    max_image_mb: values.max_image_mb,
    max_audio_mb: values.max_audio_mb,
    max_video_mb: values.max_video_mb,
  })
}
