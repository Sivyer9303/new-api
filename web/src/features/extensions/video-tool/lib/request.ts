/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
export type VideoRequestInput = {
  model: string
  prompt: string
  generationType: string
  aspectRatio: string
  durationValue: string
  durationFieldKey: string
  images?: string[]
  audioURL?: string
}

export function buildVideoGenerationRequest(
  input: VideoRequestInput
): Record<string, unknown> {
  const request: Record<string, unknown> = {
    model: input.model,
    prompt: input.prompt.trim(),
    generation_type: input.generationType,
    aspect_ratio: input.aspectRatio,
  }
  if (input.durationFieldKey === 'duration') {
    request.duration = Number(input.durationValue)
  } else {
    request.seconds = input.durationValue
  }
  if (input.images && input.images.length > 0) {
    request.images = input.images
  }
  if (input.audioURL) {
    request.audio_url = input.audioURL
  }
  return request
}
