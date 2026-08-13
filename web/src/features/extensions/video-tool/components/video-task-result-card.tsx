/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { Button, buttonVariants } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { cn } from '@/lib/utils'

import { isVideoStoragePhase } from '../lib/capabilities'

export function VideoTaskResultCard(props: {
  taskId: string
  taskStatus: string
  taskProgress: string
  previewUrl: string
  pollError: string
  isPolling: boolean
  pollingPaused: boolean
  canResumePolling: boolean
  onResumePolling: () => void
}) {
  const { t } = useTranslation()

  return (
    <Card>
      <CardHeader className='pb-3'>
        <CardTitle className='text-base'>{t('Result')}</CardTitle>
        <CardDescription>
          {props.taskId ? (
            <>
              {t('Task ID')}: <span className='font-mono'>{props.taskId}</span>
              {props.taskStatus ? ` · ${props.taskStatus}` : ''}
            </>
          ) : null}
        </CardDescription>
      </CardHeader>
      <CardContent className='space-y-3' aria-live='polite' aria-atomic='false'>
        {props.isPolling ? (
          <div className='space-y-1 text-sm'>
            <p className='text-muted-foreground'>
              {isVideoStoragePhase(props.taskProgress)
                ? t('Storing generated video locally...')
                : t('Refreshing task status automatically...')}
            </p>
            <p>
              {t('Progress')}:{' '}
              <span className='font-medium'>
                {props.taskProgress || t('Waiting for update')}
              </span>
            </p>
          </div>
        ) : null}
        {!props.isPolling &&
        props.taskProgress &&
        !props.pollError &&
        !props.previewUrl ? (
          <p className='text-sm'>
            {t('Progress')}:{' '}
            <span className='font-medium'>{props.taskProgress}</span>
          </p>
        ) : null}
        {props.pollError && props.isPolling ? (
          <p className='text-sm text-amber-700 dark:text-amber-300'>
            {t('Status refresh failed temporarily; retrying automatically.')}{' '}
            {props.pollError}
          </p>
        ) : null}
        {props.pollError && !props.isPolling ? (
          <p className='text-destructive text-sm'>{props.pollError}</p>
        ) : null}
        {props.pollingPaused && props.canResumePolling ? (
          <Button
            type='button'
            size='sm'
            variant='outline'
            onClick={props.onResumePolling}
          >
            {t('Retry')}
          </Button>
        ) : null}
        {props.previewUrl ? (
          <div className='space-y-2'>
            <video
              className='bg-muted aspect-video w-full rounded-md'
              src={props.previewUrl}
              controls
              playsInline
            />
            <a
              href={props.previewUrl}
              target='_blank'
              rel='noopener noreferrer'
              className='text-muted-foreground text-sm hover:underline'
            >
              {t('Download link')}
            </a>
          </div>
        ) : null}
        {props.taskId ? (
          <Link
            to='/usage-logs/$section'
            params={{ section: 'task' }}
            search={{ filter: props.taskId }}
            className={cn(buttonVariants({ variant: 'link' }), 'h-auto p-0')}
          >
            {t('View this task in Task Logs')}
          </Link>
        ) : null}
      </CardContent>
    </Card>
  )
}
