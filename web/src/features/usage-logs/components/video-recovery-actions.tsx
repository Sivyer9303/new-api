/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { handleServerError } from '@/lib/handle-server-error'

import {
  confirmVideoTaskProvider,
  getVideoTaskDiagnostics,
  refundVideoTask,
  retryVideoTaskStorage,
} from '../api'
import { getVideoRecoveryAvailability } from '../lib/video-recovery'
import type { VideoProviderConfirmation } from '../types'

function DiagnosticRow(props: {
  label: string
  value: string | number | undefined
}) {
  return (
    <div className='grid grid-cols-[8rem_minmax(0,1fr)] gap-2 text-xs'>
      <span className='text-muted-foreground'>{props.label}</span>
      <span className='min-w-0 font-mono break-all'>{props.value ?? '-'}</span>
    </div>
  )
}

export function VideoRecoveryActions(props: { taskID: string }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [open, setOpen] = useState(false)
  const [refundOpen, setRefundOpen] = useState(false)
  const [refundReason, setRefundReason] = useState('')
  const [confirmation, setConfirmation] =
    useState<VideoProviderConfirmation | null>(null)

  const diagnosticsQuery = useQuery({
    queryKey: ['video-task-diagnostics', props.taskID],
    queryFn: () => getVideoTaskDiagnostics(props.taskID),
    enabled: open,
  })

  function refreshTaskData() {
    void queryClient.invalidateQueries({
      queryKey: ['video-task-diagnostics', props.taskID],
    })
    void queryClient.invalidateQueries({ queryKey: ['logs'] })
  }

  const retryMutation = useMutation({
    mutationFn: () => retryVideoTaskStorage(props.taskID),
    onSuccess: () => {
      toast.success(t('Video storage retry queued'))
      refreshTaskData()
    },
    onError: handleServerError,
  })
  const confirmMutation = useMutation({
    mutationFn: () => confirmVideoTaskProvider(props.taskID),
    onSuccess: (result) => {
      setConfirmation(result)
      toast.success(t('Upstream result confirmed'))
      refreshTaskData()
    },
    onError: handleServerError,
  })
  const refundMutation = useMutation({
    mutationFn: () => refundVideoTask(props.taskID, refundReason.trim()),
    onSuccess: (result) => {
      setRefundOpen(false)
      setRefundReason('')
      toast.success(
        result.already_refunded
          ? t('This video was already refunded')
          : t('Full video refund completed')
      )
      refreshTaskData()
    },
    onError: handleServerError,
  })

  const diagnostics = diagnosticsQuery.data
  const availability = diagnostics
    ? getVideoRecoveryAvailability(diagnostics)
    : null
  const busy =
    retryMutation.isPending ||
    confirmMutation.isPending ||
    refundMutation.isPending

  return (
    <>
      <Button
        type='button'
        size='sm'
        variant='outline'
        className='h-7 px-2 text-xs'
        onClick={() => setOpen(true)}
      >
        {t('Video recovery')}
      </Button>
      <Dialog
        open={open}
        onOpenChange={setOpen}
        title={t('Video delivery diagnostics')}
        description={t(
          'Private provider and storage details are visible only to administrators.'
        )}
        contentClassName='sm:max-w-2xl'
        contentHeight='min(72dvh, 720px)'
      >
        {diagnosticsQuery.isLoading && (
          <p className='text-muted-foreground text-sm'>
            {t('Loading diagnostics...')}
          </p>
        )}
        {!diagnosticsQuery.isLoading &&
          (diagnosticsQuery.isError || !diagnostics) && (
            <p className='text-destructive text-sm'>
              {t('Failed to load video diagnostics')}
            </p>
          )}
        {!diagnosticsQuery.isLoading &&
          !diagnosticsQuery.isError &&
          diagnostics && (
            <div className='space-y-4'>
              <div className='space-y-1 rounded-md border p-3'>
                <DiagnosticRow
                  label={t('Task ID')}
                  value={diagnostics.task_id}
                />
                <DiagnosticRow
                  label={t('Upstream Task ID')}
                  value={diagnostics.upstream_task_id}
                />
                <DiagnosticRow label={t('Status')} value={diagnostics.status} />
                <DiagnosticRow
                  label={t('Progress')}
                  value={diagnostics.progress}
                />
                <DiagnosticRow
                  label={t('Storage status')}
                  value={diagnostics.storage.status}
                />
                <DiagnosticRow
                  label={t('Storage attempts')}
                  value={diagnostics.storage.retry_count}
                />
                <DiagnosticRow
                  label={t('Last storage error')}
                  value={diagnostics.storage.last_error}
                />
                <DiagnosticRow
                  label={t('Private upstream result URL')}
                  value={diagnostics.storage.upstream_result_url}
                />
                <DiagnosticRow
                  label={t('Refunded quota')}
                  value={diagnostics.manual_refund.quota}
                />
              </div>
              {confirmation && (
                <div className='space-y-1 rounded-md border p-3'>
                  <p className='text-sm font-medium'>
                    {t('Latest upstream confirmation')}
                  </p>
                  <DiagnosticRow
                    label={t('Status')}
                    value={confirmation.status}
                  />
                  <DiagnosticRow
                    label={t('Failure reason')}
                    value={confirmation.failure_reason}
                  />
                  <DiagnosticRow
                    label={t('Private upstream result URL')}
                    value={confirmation.result_url}
                  />
                </div>
              )}
              {availability?.refunded && (
                <p className='text-muted-foreground text-sm'>
                  {t(
                    'This task has been fully refunded. Storage retry and content delivery are permanently disabled.'
                  )}
                </p>
              )}
              <div className='flex flex-wrap gap-2'>
                <Button
                  type='button'
                  size='sm'
                  variant='outline'
                  disabled={!availability?.canConfirmProvider || busy}
                  onClick={() => confirmMutation.mutate()}
                >
                  {confirmMutation.isPending
                    ? t('Confirming...')
                    : t('Confirm upstream result')}
                </Button>
                <Button
                  type='button'
                  size='sm'
                  variant='outline'
                  disabled={!availability?.canRetryStorage || busy}
                  onClick={() => retryMutation.mutate()}
                >
                  {retryMutation.isPending
                    ? t('Retrying...')
                    : t('Retry storage')}
                </Button>
                <Button
                  type='button'
                  size='sm'
                  variant='destructive'
                  disabled={!availability?.canRefund || busy}
                  onClick={() => setRefundOpen(true)}
                >
                  {t('Issue full refund')}
                </Button>
              </div>
            </div>
          )}
      </Dialog>
      <ConfirmDialog
        open={refundOpen}
        onOpenChange={setRefundOpen}
        title={t('Confirm full video refund')}
        desc={t(
          'This action refunds the full task charge and permanently disables storage retry and content delivery.'
        )}
        confirmText={
          refundMutation.isPending ? t('Refunding...') : t('Issue full refund')
        }
        destructive
        disabled={!refundReason.trim()}
        isLoading={refundMutation.isPending}
        handleConfirm={() => refundMutation.mutate()}
      >
        <Textarea
          value={refundReason}
          onChange={(event) => setRefundReason(event.target.value)}
          placeholder={t('Required refund reason')}
          aria-label={t('Refund reason')}
        />
      </ConfirmDialog>
    </>
  )
}
