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
import { Copy, Check } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'

import {
  getTaskRequestCopyJSON,
  getTaskRequestFields,
  getTaskRequestMedia,
  hasTaskRequestSnapshot,
} from '../../lib/task-log-display'
import type { TaskLog } from '../../types'

interface RequestParamsDialogProps {
  log: TaskLog
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function RequestParamsDialog(props: RequestParamsDialogProps) {
  const { t } = useTranslation()
  const { copiedText, copyToClipboard } = useCopyToClipboard({ notify: false })
  const fields = getTaskRequestFields(props.log)
  const media = getTaskRequestMedia(props.log)
  const copyText = getTaskRequestCopyJSON(props.log)
  const hasSnapshot = hasTaskRequestSnapshot(props.log)

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={t('Request parameters')}
      description={t('Request parameters for this task')}
      contentClassName='sm:max-w-lg'
      contentHeight='auto'
      bodyClassName='space-y-4'
    >
      <ScrollArea className='max-h-[500px] pr-4'>
        <div className='space-y-4 py-4'>
          {!hasSnapshot ? (
            <p className='text-muted-foreground text-sm'>
              {t('No request parameters were recorded for this task.')}
            </p>
          ) : (
            <div className='relative space-y-3'>
              <Button
                variant='ghost'
                size='sm'
                className='absolute top-0 right-0 h-8 w-8 p-0'
                onClick={() => copyToClipboard(copyText)}
                title={t('Copy to clipboard')}
              >
                {copiedText === copyText ? (
                  <Check className='size-4 text-green-600' />
                ) : (
                  <Copy className='size-4' />
                )}
              </Button>
              {fields.map((field) => {
                let value = field.value
                if (field.translateValue) {
                  value = t(field.value)
                }
                return (
                  <div key={field.labelKey} className='space-y-1 pr-10'>
                    <Label className='text-sm font-semibold'>
                      {t(field.labelKey)}
                    </Label>
                    <p className='text-sm leading-relaxed break-words whitespace-pre-wrap'>
                      {value}
                    </p>
                  </div>
                )
              })}
              {media.length > 0 ? (
                <div className='space-y-1 pr-10'>
                  <Label className='text-sm font-semibold'>
                    {t('Attachments')}
                  </Label>
                  <ul className='text-muted-foreground list-inside list-disc text-sm'>
                    {media.map((item) => {
                      const typeLabel = t(item.typeKey)
                      if (!item.roleKey) {
                        return <li key={item.key}>{typeLabel}</li>
                      }
                      return (
                        <li key={item.key}>
                          {typeLabel} ({t(item.roleKey)})
                        </li>
                      )
                    })}
                  </ul>
                </div>
              ) : null}
            </div>
          )}
        </div>
      </ScrollArea>
    </Dialog>
  )
}

export function TaskRequestParamsButton(props: { log: TaskLog }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)

  if (!hasTaskRequestSnapshot(props.log)) {
    return null
  }

  return (
    <>
      <button
        type='button'
        className='text-foreground hover:underline'
        onClick={() => setOpen(true)}
      >
        {t('View request')}
      </button>
      <RequestParamsDialog log={props.log} open={open} onOpenChange={setOpen} />
    </>
  )
}
