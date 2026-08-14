/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

export type PromptMentionKind = 'image' | 'audio' | 'video'

export type PromptMentionDialect = 'latin' | 'zh'

export type PromptMentionOption = {
  kind: PromptMentionKind
  /** 1-based index matching upstream @ImageN / @AudioN / @VideoN or Brioi @图片N */
  index: number
  token: string
  label: string
  previewUrl?: string
  fileName?: string
}

const LATIN_MENTION_NAMES = {
  image: 'Image',
  audio: 'Audio',
  video: 'Video',
} as const

const ZH_MENTION_NAMES = {
  image: '图片',
  audio: '音频',
  video: '视频',
} as const

export const PROMPT_MENTION_TOKEN_RE =
  /@(Image|Audio|Video|图片|音频|视频)([1-9]\d*)\b/g

export function mentionToken(
  kind: PromptMentionKind,
  index: number,
  dialect: PromptMentionDialect = 'latin'
): string {
  const names = dialect === 'zh' ? ZH_MENTION_NAMES : LATIN_MENTION_NAMES
  return `@${names[kind]}${index}`
}

export function parseMentionToken(
  token: string
): { kind: PromptMentionKind; index: number } | null {
  const match = /^@(Image|Audio|Video|图片|音频|视频)([1-9]\d*)$/.exec(
    token.trim()
  )
  if (!match) return null
  let kind: PromptMentionKind = 'video'
  if (match[1] === 'Image' || match[1] === '图片') kind = 'image'
  else if (match[1] === 'Audio' || match[1] === '音频') kind = 'audio'
  return { kind, index: Number(match[2]) }
}

export function buildPromptMentionOptions(input: {
  images?: Array<{ previewUrl?: string; fileName?: string }>
  audio?: { fileName?: string } | null
  videos?: Array<{ previewUrl?: string; fileName?: string }>
  dialect?: PromptMentionDialect
  labelFor: (kind: PromptMentionKind, index: number) => string
}): PromptMentionOption[] {
  const dialect = input.dialect ?? 'latin'
  const options: PromptMentionOption[] = []
  input.images?.forEach((image, offset) => {
    const index = offset + 1
    options.push({
      kind: 'image',
      index,
      token: mentionToken('image', index, dialect),
      label: input.labelFor('image', index),
      previewUrl: image.previewUrl,
      fileName: image.fileName,
    })
  })
  if (input.audio) {
    options.push({
      kind: 'audio',
      index: 1,
      token: mentionToken('audio', 1, dialect),
      label: input.labelFor('audio', 1),
      fileName: input.audio.fileName,
    })
  }
  input.videos?.forEach((video, offset) => {
    const index = offset + 1
    options.push({
      kind: 'video',
      index,
      token: mentionToken('video', index, dialect),
      label: input.labelFor('video', index),
      previewUrl: video.previewUrl,
      fileName: video.fileName,
    })
  })
  return options
}

export function filterPromptMentionOptions(
  options: PromptMentionOption[],
  query: string
): PromptMentionOption[] {
  const normalized = query.trim().toLowerCase()
  if (!normalized) return options
  return options.filter((option) => {
    const haystack = [
      option.token,
      option.label,
      option.fileName ?? '',
      option.kind,
      String(option.index),
    ]
      .join(' ')
      .toLowerCase()
    return haystack.includes(normalized)
  })
}

export function groupPromptMentionOptions(options: PromptMentionOption[]): {
  images: PromptMentionOption[]
  audios: PromptMentionOption[]
  videos: PromptMentionOption[]
} {
  return {
    images: options.filter((option) => option.kind === 'image'),
    audios: options.filter((option) => option.kind === 'audio'),
    videos: options.filter((option) => option.kind === 'video'),
  }
}

/** Detect an open @-query immediately before the caret in plain text. */
export function findActiveMentionQuery(
  textBeforeCaret: string
): { start: number; query: string } | null {
  const at = textBeforeCaret.lastIndexOf('@')
  if (at < 0) return null
  if (at > 0) {
    const prev = textBeforeCaret[at - 1]
    // Only suppress email-like forms (user@…); allow @ after CJK, punctuation,
    // or whitespace so prompts like "角色随@Audio1" still open the picker.
    if (prev && /[A-Za-z0-9._-]/.test(prev)) return null
  }
  const query = textBeforeCaret.slice(at + 1)
  if (/[\s\n]/.test(query)) return null
  if (query.length > 32) return null
  return { start: at, query }
}
