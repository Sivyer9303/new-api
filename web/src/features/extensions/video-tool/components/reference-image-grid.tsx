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
import { Cancel01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useEffect, useId, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { cn } from '@/lib/utils'

import {
  createReferenceImageItem,
  type ReferenceImageItem,
} from '../lib/reference-image'
import type { VideoMediaRole } from '../types'

type ReferenceImageGridProps = {
  items: ReferenceImageItem[]
  onChange: (items: ReferenceImageItem[]) => void
  min: number
  max: number
  roles: VideoMediaRole[]
  disabled?: boolean
}

/**
 * Three-column reference image picker. Empty cells open a file dialog; multi-select
 * appends until max. Existing thumbnails can be removed individually.
 */
export function ReferenceImageGrid(props: ReferenceImageGridProps) {
  const { t } = useTranslation()
  const inputRef = useRef<HTMLInputElement>(null)
  const replaceIndexRef = useRef<number | null>(null)
  const mountedRef = useRef(true)
  const previewRequestRef = useRef(0)
  const [creatingPreviews, setCreatingPreviews] = useState(false)
  const inputId = useId()
  const maxSlots = Math.min(Math.max(props.max, 0), 30)
  const remaining = Math.max(0, props.max - props.items.length)
  const disabled = props.disabled || creatingPreviews

  useEffect(() => {
    return () => {
      mountedRef.current = false
    }
  }, [])

  useEffect(() => {
    previewRequestRef.current += 1
    setCreatingPreviews(false)
  }, [props.max])

  const openPicker = (replaceIndex: number | null) => {
    if (disabled) return
    if (replaceIndex === null && remaining <= 0) {
      toast.error(
        t('You can upload at most {{max}} image(s)', { max: props.max })
      )
      return
    }
    replaceIndexRef.current = replaceIndex
    if (inputRef.current) {
      // Multi-select only when appending into empty capacity.
      inputRef.current.multiple = replaceIndex === null && remaining > 1
      inputRef.current.value = ''
      inputRef.current.click()
    }
  }

  const handleFilesSelected = async (fileList: FileList | null) => {
    if (!fileList || fileList.length === 0) return
    const picked = [...fileList].filter((f) => f.type.startsWith('image/'))
    if (picked.length === 0) {
      toast.error(t('Please select image files'))
      return
    }

    const replaceIndex = replaceIndexRef.current
    replaceIndexRef.current = null

    if (
      replaceIndex !== null &&
      replaceIndex >= 0 &&
      replaceIndex < props.items.length
    ) {
      // Replace a single slot (first picked file only).
      const requestId = ++previewRequestRef.current
      setCreatingPreviews(true)
      const created = await createReferenceImageItem(picked[0])
      if (!mountedRef.current || requestId !== previewRequestRef.current) {
        URL.revokeObjectURL(created.previewUrl)
        return
      }
      const next = [...props.items]
      const prev = next[replaceIndex]
      URL.revokeObjectURL(prev.previewUrl)
      next[replaceIndex] = created
      props.onChange(next)
      setCreatingPreviews(false)
      if (picked.length > 1) {
        toast.message(
          t('Replaced one image. Extra selected files were ignored.')
        )
      }
      return
    }

    const room = props.max - props.items.length
    if (room <= 0) {
      toast.error(
        t('You can upload at most {{max}} image(s)', { max: props.max })
      )
      return
    }
    const accepted = picked.slice(0, room)
    const requestId = ++previewRequestRef.current
    setCreatingPreviews(true)
    const created = await Promise.all(accepted.map(createReferenceImageItem))
    if (!mountedRef.current || requestId !== previewRequestRef.current) {
      for (const item of created) URL.revokeObjectURL(item.previewUrl)
      return
    }
    props.onChange([...props.items, ...created])
    setCreatingPreviews(false)
    if (picked.length > room) {
      toast.message(
        t('Only {{count}} more image(s) could be added (max {{max}}).', {
          count: room,
          max: props.max,
        })
      )
    }
  }

  const removeAt = (index: number) => {
    const next = [...props.items]
    const [removed] = next.splice(index, 1)
    if (removed) URL.revokeObjectURL(removed.previewUrl)
    props.onChange(next)
  }

  if (maxSlots <= 0) return null

  const visibleSlotCount = Math.min(
    maxSlots,
    Math.max(props.items.length + (remaining > 0 ? 1 : 0), props.min)
  )
  const slots: Array<ReferenceImageItem | null> = [
    ...Array(visibleSlotCount),
  ].map((_, i) => props.items[i] ?? null)

  return (
    <div className='space-y-2'>
      <input
        id={inputId}
        ref={inputRef}
        type='file'
        accept='image/*'
        multiple
        className='sr-only'
        disabled={disabled}
        onChange={(e) => void handleFilesSelected(e.target.files)}
      />
      <div className='grid w-fit grid-cols-3 gap-2'>
        {slots.map((item, index) => {
          if (item) {
            const role = props.roles[index] ?? props.roles[0] ?? 'reference'
            let roleLabel = t('Reference')
            if (role === 'first_frame') roleLabel = t('First frame')
            if (role === 'last_frame') roleLabel = t('Last frame')
            return (
              <div
                key={item.id}
                className='bg-muted/40 group relative size-28 overflow-hidden rounded-md border sm:size-32'
              >
                <button
                  type='button'
                  className='size-full cursor-pointer'
                  onClick={() => openPicker(index)}
                  disabled={disabled}
                  aria-label={t('Replace image {{n}}', { n: index + 1 })}
                >
                  <img
                    src={item.previewUrl}
                    alt={item.file.name}
                    className='size-full object-cover'
                  />
                </button>
                <button
                  type='button'
                  className={cn(
                    'bg-background/90 text-foreground absolute top-1 right-1 inline-flex size-6 items-center justify-center rounded-full border shadow-sm',
                    'opacity-100 sm:opacity-0 sm:group-hover:opacity-100'
                  )}
                  onClick={() => removeAt(index)}
                  disabled={disabled}
                  aria-label={t('Remove image {{n}}', { n: index + 1 })}
                >
                  <HugeiconsIcon icon={Cancel01Icon} className='size-3' />
                </button>
                <span className='bg-background/80 absolute bottom-1 left-1 rounded px-1 py-0.5 text-[10px] font-medium'>
                  {index + 1} · {roleLabel}
                </span>
              </div>
            )
          }

          const canAdd = props.items.length < props.max
          // Fixed grid slot positions use positional keys by design.
          return (
            <button
              key={`empty-slot-${index + 1}-of-${maxSlots}`}
              type='button'
              disabled={disabled || !canAdd}
              onClick={() => openPicker(null)}
              className={cn(
                'border-muted-foreground/30 text-muted-foreground flex size-28 flex-col items-center justify-center gap-0.5 rounded-md border border-dashed text-xs transition-colors sm:size-32',
                canAdd && !disabled
                  ? 'hover:border-primary hover:text-primary cursor-pointer'
                  : 'cursor-not-allowed opacity-40'
              )}
              aria-label={t('Add image')}
            >
              <span className='text-xl leading-none font-light'>+</span>
              <span>{t('Add')}</span>
            </button>
          )
        })}
      </div>
      <p className='text-muted-foreground text-xs'>
        {t(
          '{{count}} / {{max}} selected (min {{min}}). Click a tile to add or replace; multi-select appends until the limit.',
          {
            count: props.items.length,
            max: props.max,
            min: props.min,
          }
        )}
      </p>
    </div>
  )
}
