/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AlertTriangle, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'

import { fetchVideoStorageStatus, refreshVideoStorageUsage } from './api'
import {
  formatStorageBytes,
  R2_SOFT_LIMIT_RATIO,
  usagePercent,
} from './storage-config'

const STORAGE_STATUS_QUERY_KEY = ['video-storage-status'] as const

export function R2UsageCard() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const status = useQuery({
    queryKey: STORAGE_STATUS_QUERY_KEY,
    queryFn: fetchVideoStorageStatus,
  })
  const refresh = useMutation({
    mutationFn: refreshVideoStorageUsage,
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: STORAGE_STATUS_QUERY_KEY,
      })
      toast.success(t('Storage usage refreshed'))
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Failed to refresh storage usage'))
    },
  })

  const usage = status.data?.r2
  const percent = usage ? usagePercent(usage.usage_bytes, usage.quota_bytes) : 0
  const checkedAt = usage?.checked_at
    ? new Date(usage.checked_at * 1000).toLocaleString()
    : t('Never')

  return (
    <div className='border-border bg-muted/40 space-y-3 rounded-lg border p-4'>
      <div className='flex items-center justify-between gap-3'>
        <div className='space-y-1'>
          <p className='text-sm font-medium'>
            {t('Cloudflare R2 free tier usage')}
          </p>
          <p className='text-muted-foreground text-sm'>
            {t(
              'Uploads stop automatically once usage reaches {{percent}}% of the {{quota}} free allowance.',
              {
                percent: Math.round(R2_SOFT_LIMIT_RATIO * 100),
                quota: formatStorageBytes(usage?.quota_bytes ?? 0),
              }
            )}
          </p>
        </div>
        <Button
          type='button'
          variant='outline'
          size='sm'
          onClick={() => refresh.mutate()}
          disabled={refresh.isPending || status.isLoading}
        >
          <RefreshCw
            className={refresh.isPending ? 'size-4 animate-spin' : 'size-4'}
            aria-hidden='true'
          />
          {t('Check now')}
        </Button>
      </div>

      {status.isLoading ? (
        <p className='text-muted-foreground text-sm' role='status'>
          {t('Loading storage usage...')}
        </p>
      ) : null}

      {usage ? (
        <div className='space-y-2'>
          <div
            className='bg-muted h-2 w-full overflow-hidden rounded-full'
            role='progressbar'
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={Math.round(percent)}
            aria-label={t('Cloudflare R2 free tier usage')}
          >
            <div
              className={
                usage.upload_blocked ? 'h-full bg-red-500' : 'bg-primary h-full'
              }
              style={{ width: `${percent}%` }}
            />
          </div>
          <dl className='grid gap-x-6 gap-y-1 text-sm sm:grid-cols-2'>
            <div className='flex justify-between gap-2'>
              <dt className='text-muted-foreground'>{t('Used')}</dt>
              <dd>
                {formatStorageBytes(usage.usage_bytes)} ({percent.toFixed(1)}%)
              </dd>
            </div>
            <div className='flex justify-between gap-2'>
              <dt className='text-muted-foreground'>{t('Upload limit')}</dt>
              <dd>{formatStorageBytes(usage.soft_limit_bytes)}</dd>
            </div>
            <div className='flex justify-between gap-2'>
              <dt className='text-muted-foreground'>{t('Last checked')}</dt>
              <dd>{checkedAt}</dd>
            </div>
            <div className='flex justify-between gap-2'>
              <dt className='text-muted-foreground'>{t('Uploads')}</dt>
              <dd>
                {usage.upload_blocked ? (
                  <Badge variant='destructive'>{t('Blocked')}</Badge>
                ) : (
                  <Badge variant='outline'>{t('Allowed')}</Badge>
                )}
              </dd>
            </div>
          </dl>
          {usage.upload_blocked ? (
            <p className='flex gap-2 text-sm text-red-600' role='alert'>
              <AlertTriangle
                className='mt-0.5 size-4 shrink-0'
                aria-hidden='true'
              />
              {t(
                'R2 bucket full: new video uploads are refused until objects expire or are deleted.'
              )}
            </p>
          ) : null}
          {usage.last_error ? (
            <p className='text-muted-foreground text-sm' role='status'>
              {t('Last usage check failed: {{error}}', {
                error: usage.last_error,
              })}
            </p>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}
