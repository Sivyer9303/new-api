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
import {
  useEffect,
  useId,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from 'react'

import { cn } from '@/lib/utils'

import {
  filterPromptMentionOptions,
  findActiveMentionQuery,
  groupPromptMentionOptions,
  parseMentionToken,
  PROMPT_MENTION_TOKEN_RE,
  type PromptMentionOption,
} from '../lib/prompt-mentions'

type PromptMentionEditorProps = {
  id?: string
  value: string
  onChange: (value: string) => void
  options: PromptMentionOption[]
  placeholder?: string
  disabled?: boolean
  className?: string
  emptyMenuLabel: string
  imageGroupLabel: string
  audioGroupLabel: string
  videoGroupLabel: string
}

type MentionMenuState = {
  query: string
  left: number
  top: number
}

function mentionKindBadge(kind: PromptMentionOption['kind']): string {
  if (kind === 'audio') return 'A'
  if (kind === 'video') return 'V'
  return 'I'
}

function MentionPreview({ option }: { option: PromptMentionOption }) {
  if (!option.previewUrl) {
    return (
      <span className='bg-muted text-muted-foreground flex size-6 items-center justify-center rounded text-[10px] font-semibold'>
        {mentionKindBadge(option.kind)}
      </span>
    )
  }
  if (option.kind === 'video') {
    return (
      <video
        src={option.previewUrl}
        muted
        playsInline
        preload='metadata'
        className='size-6 rounded object-cover'
      />
    )
  }
  return (
    <img
      src={option.previewUrl}
      alt=''
      className='size-6 rounded object-cover'
    />
  )
}

function optionsByTokenMap(
  options: PromptMentionOption[]
): Map<string, PromptMentionOption> {
  return new Map(options.map((option) => [option.token, option]))
}

function serializeNode(node: Node): string {
  if (node.nodeType === Node.TEXT_NODE) {
    return node.textContent ?? ''
  }
  if (node.nodeType !== Node.ELEMENT_NODE) return ''
  const el = node as HTMLElement
  if (el.dataset.mentionToken) return el.dataset.mentionToken
  if (el.tagName === 'BR') return '\n'
  let out = ''
  for (const child of el.childNodes) {
    out += serializeNode(child)
  }
  return out
}

function serializeEditor(root: HTMLElement): string {
  const children = [...root.childNodes]
  if (children.length === 0) return ''
  const hasBlocks = children.some(
    (node) =>
      node.nodeType === Node.ELEMENT_NODE &&
      ['DIV', 'P'].includes((node as HTMLElement).tagName)
  )
  if (!hasBlocks) {
    return children.map(serializeNode).join('')
  }
  return children
    .map((node) => {
      if (
        node.nodeType === Node.ELEMENT_NODE &&
        ['DIV', 'P'].includes((node as HTMLElement).tagName)
      ) {
        const line = serializeNode(node)
        return line === '\n' ? '' : line.replace(/\n$/, '')
      }
      return serializeNode(node)
    })
    .join('\n')
}

function createMentionChip(
  option:
    | PromptMentionOption
    | {
        token: string
        label: string
        previewUrl?: string
        kind?: PromptMentionOption['kind']
      },
  missing: boolean
): HTMLSpanElement {
  const chip = document.createElement('span')
  chip.dataset.mentionToken = option.token
  chip.contentEditable = 'false'
  chip.setAttribute('role', 'inline')
  chip.className = cn(
    'mx-0.5 inline-flex max-w-full items-center gap-1 rounded-md border px-1.5 py-0 align-middle text-[0.925em] font-medium leading-none',
    missing
      ? 'border-destructive/40 bg-destructive/10 text-destructive'
      : 'border-primary/20 bg-primary/10 text-primary'
  )

  const label = document.createElement('span')
  label.className = 'whitespace-nowrap'
  label.textContent = `${option.label}：`
  chip.appendChild(label)

  const kind =
    'kind' in option && option.kind
      ? option.kind
      : parseMentionToken(option.token)?.kind
  if (option.previewUrl && kind === 'image') {
    const img = document.createElement('img')
    img.src = option.previewUrl
    img.alt = ''
    img.draggable = false
    img.className =
      'inline-block h-[1.1em] w-[1.1em] shrink-0 rounded-sm object-cover align-middle'
    chip.appendChild(img)
  } else if (option.previewUrl && kind === 'video') {
    const video = document.createElement('video')
    video.src = option.previewUrl
    video.muted = true
    video.playsInline = true
    video.preload = 'metadata'
    video.className =
      'inline-block h-[1.1em] w-[1.1em] shrink-0 rounded-sm object-cover align-middle'
    chip.appendChild(video)
  }

  return chip
}

function renderPlainValue(
  root: HTMLElement,
  value: string,
  optionsByToken: Map<string, PromptMentionOption>
) {
  root.replaceChildren()
  if (!value) return

  let lastIndex = 0
  const pattern = new RegExp(PROMPT_MENTION_TOKEN_RE.source, 'g')
  for (const match of value.matchAll(pattern)) {
    const index = match.index ?? 0
    if (index > lastIndex) {
      root.appendChild(document.createTextNode(value.slice(lastIndex, index)))
    }
    const token = match[0]
    const option = optionsByToken.get(token)
    const parsed = parseMentionToken(token)
    let fallbackLabel = token
    if (parsed) {
      if (parsed.kind === 'image') fallbackLabel = `Image${parsed.index}`
      else if (parsed.kind === 'audio') fallbackLabel = `Audio${parsed.index}`
      else fallbackLabel = `Video${parsed.index}`
    }
    root.appendChild(
      createMentionChip(
        option ?? {
          token,
          label: fallbackLabel,
          previewUrl: undefined,
        },
        !option
      )
    )
    lastIndex = index + token.length
  }
  if (lastIndex < value.length) {
    root.appendChild(document.createTextNode(value.slice(lastIndex)))
  }
}

function placeCaretAfter(node: Node) {
  const selection = window.getSelection()
  if (!selection) return
  const range = document.createRange()
  range.setStartAfter(node)
  range.collapse(true)
  selection.removeAllRanges()
  selection.addRange(range)
}

function getCaretClientPoint(root: HTMLElement): { left: number; top: number } {
  const selection = window.getSelection()
  if (!selection || selection.rangeCount === 0) {
    const rect = root.getBoundingClientRect()
    return { left: 12, top: rect.height }
  }
  const range = selection.getRangeAt(0).cloneRange()
  range.collapse(true)
  const rects = range.getClientRects()
  const rect = rects.item(0) ?? range.getBoundingClientRect()
  const rootRect = root.getBoundingClientRect()
  if (rect.width === 0 && rect.height === 0) {
    return { left: 12, top: 28 }
  }
  return {
    left: Math.max(8, rect.left - rootRect.left),
    top: Math.max(24, rect.bottom - rootRect.top + 6),
  }
}

function readTextBeforeCaretInTextNode(): string | null {
  const selection = window.getSelection()
  if (!selection || selection.rangeCount === 0) return null
  const node = selection.anchorNode
  if (!node || node.nodeType !== Node.TEXT_NODE) return null
  return (node.textContent ?? '').slice(0, selection.anchorOffset)
}

export function PromptMentionEditor({
  id,
  value,
  onChange,
  options,
  placeholder,
  disabled,
  className,
  emptyMenuLabel,
  imageGroupLabel,
  audioGroupLabel,
  videoGroupLabel,
}: PromptMentionEditorProps) {
  const rootRef = useRef<HTMLDivElement>(null)
  const skipSyncRef = useRef(false)
  const optionsByToken = useMemo(() => optionsByTokenMap(options), [options])
  const [menu, setMenu] = useState<MentionMenuState | null>(null)
  const [highlight, setHighlight] = useState(0)
  const listId = useId()

  const filtered = useMemo(() => {
    if (!menu) return [] as PromptMentionOption[]
    return filterPromptMentionOptions(options, menu.query)
  }, [menu, options])
  const grouped = useMemo(() => groupPromptMentionOptions(filtered), [filtered])
  const flatFiltered = useMemo(
    () => [...grouped.images, ...grouped.audios, ...grouped.videos],
    [grouped]
  )

  useLayoutEffect(() => {
    const root = rootRef.current
    if (!root) return
    if (skipSyncRef.current) {
      skipSyncRef.current = false
      return
    }
    if (serializeEditor(root) === value) {
      // Refresh chip previews/labels when media list changes.
      root
        .querySelectorAll<HTMLElement>('[data-mention-token]')
        .forEach((chip) => {
          const token = chip.dataset.mentionToken
          if (!token) return
          const option = optionsByToken.get(token)
          const next = createMentionChip(
            option ?? {
              token,
              label: token.replace(/^@/, ''),
            },
            !option
          )
          chip.replaceWith(next)
        })
      return
    }
    renderPlainValue(root, value, optionsByToken)
  }, [value, optionsByToken])

  useEffect(() => {
    setHighlight(0)
  }, [menu?.query, filtered.length])

  function emitChange() {
    const root = rootRef.current
    if (!root) return
    skipSyncRef.current = true
    onChange(serializeEditor(root))
  }

  function refreshMentionMenu() {
    if (disabled) {
      setMenu(null)
      return
    }
    const textBefore = readTextBeforeCaretInTextNode()
    if (textBefore == null) {
      setMenu(null)
      return
    }
    const active = findActiveMentionQuery(textBefore)
    if (!active) {
      setMenu(null)
      return
    }
    const root = rootRef.current
    if (!root) return
    const point = getCaretClientPoint(root)
    setMenu({
      query: active.query,
      left: point.left,
      top: point.top,
    })
  }

  function insertMention(option: PromptMentionOption) {
    const root = rootRef.current
    const selection = window.getSelection()
    if (!root || !selection || selection.rangeCount === 0) return
    const node = selection.anchorNode
    if (!node || node.nodeType !== Node.TEXT_NODE) return
    const textNode = node as Text
    const before = (textNode.textContent ?? '').slice(0, selection.anchorOffset)
    const active = findActiveMentionQuery(before)
    if (!active) return

    const after = (textNode.textContent ?? '').slice(selection.anchorOffset)
    const prefix = before.slice(0, active.start)
    textNode.textContent = prefix
    const chip = createMentionChip(option, false)
    if (textNode.textContent === '') {
      textNode.replaceWith(chip)
    } else {
      textNode.after(chip)
    }
    const trailing = document.createTextNode(
      after.startsWith(' ') ? after : ` ${after}`
    )
    chip.after(trailing)
    placeCaretAfter(chip)
    // Move caret into the trailing text at offset 1 (after inserted space) when present.
    if (trailing.textContent?.startsWith(' ')) {
      const range = document.createRange()
      range.setStart(trailing, 1)
      range.collapse(true)
      selection.removeAllRanges()
      selection.addRange(range)
    }
    setMenu(null)
    emitChange()
  }

  function onKeyDown(event: React.KeyboardEvent<HTMLDivElement>) {
    if (!menu) {
      if (event.key === 'Enter' && !event.shiftKey) {
        // Keep multiline: Enter inserts newline via contentEditable default.
      }
      return
    }
    if (event.key === 'Escape') {
      event.preventDefault()
      setMenu(null)
      return
    }
    if (event.key === 'ArrowDown') {
      event.preventDefault()
      if (flatFiltered.length === 0) return
      setHighlight((current) => (current + 1) % flatFiltered.length)
      return
    }
    if (event.key === 'ArrowUp') {
      event.preventDefault()
      if (flatFiltered.length === 0) return
      setHighlight(
        (current) => (current - 1 + flatFiltered.length) % flatFiltered.length
      )
      return
    }
    if (event.key === 'Enter' || event.key === 'Tab') {
      const selected = flatFiltered[highlight]
      if (!selected) return
      event.preventDefault()
      insertMention(selected)
    }
  }

  function renderGroup(
    label: string,
    items: PromptMentionOption[],
    offset: number
  ) {
    if (items.length === 0) return null
    return (
      <div key={label} className='py-1'>
        <p className='text-muted-foreground px-2 py-1 text-[11px] font-medium tracking-wide uppercase'>
          {label}
        </p>
        <ul className='flex flex-col gap-0.5'>
          {items.map((item, index) => {
            const flatIndex = offset + index
            return (
              <li key={item.token}>
                <button
                  type='button'
                  id={`${listId}-${item.token}`}
                  className={cn(
                    'flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm',
                    flatIndex === highlight
                      ? 'bg-accent text-accent-foreground'
                      : 'hover:bg-muted'
                  )}
                  onMouseDown={(event) => {
                    event.preventDefault()
                    insertMention(item)
                  }}
                  onMouseEnter={() => setHighlight(flatIndex)}
                >
                  <MentionPreview option={item} />
                  <span className='min-w-0 flex-1 truncate'>
                    <span className='font-medium'>{item.label}</span>
                    <span className='text-muted-foreground ml-2 font-mono text-xs'>
                      {item.token}
                    </span>
                  </span>
                </button>
              </li>
            )
          })}
        </ul>
      </div>
    )
  }

  const showPlaceholder = !value

  return (
    <div className='relative'>
      <div
        id={id}
        ref={rootRef}
        role='textbox'
        aria-multiline='true'
        aria-autocomplete='list'
        aria-expanded={menu != null}
        aria-controls={menu ? listId : undefined}
        aria-activedescendant={
          menu && flatFiltered[highlight]
            ? `${listId}-${flatFiltered[highlight].token}`
            : undefined
        }
        contentEditable={!disabled}
        suppressContentEditableWarning
        data-placeholder={placeholder}
        className={cn(
          'border-input focus-visible:border-ring focus-visible:ring-ring/50 dark:bg-input/30 empty:before:text-muted-foreground max-h-96 min-h-24 w-full overflow-y-auto rounded-lg border bg-transparent px-2.5 py-2 text-base transition-colors outline-none focus-visible:ring-3 md:text-sm',
          'whitespace-pre-wrap break-words',
          disabled && 'cursor-not-allowed opacity-50',
          showPlaceholder &&
            'empty:before:pointer-events-none empty:before:content-[attr(data-placeholder)]',
          className
        )}
        onInput={() => {
          const root = rootRef.current
          if (root && serializeEditor(root).trim() === '') {
            root.replaceChildren()
          }
          emitChange()
          refreshMentionMenu()
        }}
        onKeyUp={refreshMentionMenu}
        onClick={refreshMentionMenu}
        onKeyDown={onKeyDown}
        onPaste={(event) => {
          event.preventDefault()
          const text = event.clipboardData.getData('text/plain')
          document.execCommand('insertText', false, text)
          emitChange()
          refreshMentionMenu()
        }}
        onBlur={() => {
          // Delay so mousedown on menu items can run first.
          window.setTimeout(() => setMenu(null), 120)
        }}
      />

      {menu ? (
        <div
          id={listId}
          role='listbox'
          className='bg-popover text-popover-foreground absolute z-30 w-[min(100%,20rem)] overflow-hidden rounded-xl border shadow-md'
          style={{ left: menu.left, top: menu.top }}
        >
          <div className='max-h-64 overflow-y-auto p-1'>
            {flatFiltered.length === 0 ? (
              <p className='text-muted-foreground px-2 py-2 text-sm'>
                {emptyMenuLabel}
              </p>
            ) : (
              <>
                {renderGroup(imageGroupLabel, grouped.images, 0)}
                {renderGroup(
                  audioGroupLabel,
                  grouped.audios,
                  grouped.images.length
                )}
                {renderGroup(
                  videoGroupLabel,
                  grouped.videos,
                  grouped.images.length + grouped.audios.length
                )}
              </>
            )}
          </div>
        </div>
      ) : null}
    </div>
  )
}
