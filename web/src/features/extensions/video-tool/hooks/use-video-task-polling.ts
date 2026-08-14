/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { localizeTaskFailReason } from '@/lib/localize-task-fail-reason'

import { fetchVideoGeneration, resolveVideoPlaybackUrl } from '../api'

const POLL_INTERVAL_MS = 3000
const POLL_MAX_ATTEMPTS = 120

function isTerminalSuccess(status: string | undefined): boolean {
  const normalized = (status || '').toLowerCase()
  return normalized === 'completed' || normalized === 'success'
}

function isTerminalFailure(status: string | undefined): boolean {
  const normalized = (status || '').toLowerCase()
  return (
    normalized === 'failed' ||
    normalized === 'failure' ||
    normalized === 'cancelled' ||
    normalized === 'canceled'
  )
}

function isTerminalStatus(status: string | undefined): boolean {
  return isTerminalSuccess(status) || isTerminalFailure(status)
}

export function useVideoTaskPolling() {
  const { t } = useTranslation()
  const [taskId, setTaskId] = useState('')
  const [taskStatus, setTaskStatus] = useState('')
  const [taskProgress, setTaskProgress] = useState('')
  const [previewUrl, setPreviewUrl] = useState('')
  const [pollError, setPollError] = useState('')
  const [pollingTokenKey, setPollingTokenKey] = useState('')
  const [pollingPaused, setPollingPaused] = useState(false)
  const pollAttemptRef = useRef(0)
  const seenTerminalToastRef = useRef('')

  const isPolling = useMemo(
    () =>
      Boolean(taskId) &&
      Boolean(pollingTokenKey) &&
      !pollingPaused &&
      !isTerminalStatus(taskStatus),
    [pollingPaused, pollingTokenKey, taskId, taskStatus]
  )

  const reset = useCallback(() => {
    setTaskId('')
    setTaskStatus('')
    setTaskProgress('')
    setPreviewUrl('')
    setPollError('')
    setPollingTokenKey('')
    setPollingPaused(false)
    pollAttemptRef.current = 0
    seenTerminalToastRef.current = ''
  }, [])

  const start = useCallback(
    (input: { taskId: string; status: string; tokenKey: string }) => {
      setTaskId(input.taskId)
      setTaskStatus(input.status)
      setTaskProgress('')
      setPreviewUrl('')
      setPollError('')
      setPollingTokenKey(input.tokenKey)
      setPollingPaused(false)
      pollAttemptRef.current = 0
      seenTerminalToastRef.current = ''
    },
    []
  )

  const failSubmission = useCallback((message: string) => {
    setPollError(message)
  }, [])

  const resume = useCallback(() => {
    pollAttemptRef.current = 0
    setPollError('')
    setPollingPaused(false)
  }, [])

  useEffect(() => {
    if (!previewUrl.startsWith('blob:')) return
    return () => URL.revokeObjectURL(previewUrl)
  }, [previewUrl])

  useEffect(() => {
    if (!taskId || !pollingTokenKey || pollingPaused) return

    let cancelled = false
    let timer: ReturnType<typeof setTimeout> | undefined

    const pollOnce = async () => {
      if (cancelled) return
      if (pollAttemptRef.current >= POLL_MAX_ATTEMPTS) {
        const message = t('Timed out waiting for video. Check Task Logs.')
        setPollError(message)
        toast.message(message)
        setPollingPaused(true)
        return
      }
      pollAttemptRef.current += 1
      try {
        const statusResponse = await fetchVideoGeneration(
          pollingTokenKey,
          taskId
        )
        if (cancelled) return
        setPollError('')
        const status = statusResponse.status || ''
        if (status) {
          setTaskStatus(status)
        }
        if (
          statusResponse.progress != null &&
          String(statusResponse.progress).trim()
        ) {
          setTaskProgress(String(statusResponse.progress))
        }
        if (isTerminalSuccess(status)) {
          const siteContent = `/v1/videos/${taskId}/content`
          let playableUrl = siteContent
          try {
            const resolved = await resolveVideoPlaybackUrl(
              pollingTokenKey,
              taskId
            )
            if (cancelled) {
              resolved.revoke?.()
              return
            }
            playableUrl = resolved.url
          } catch {
            playableUrl = siteContent
          }
          setPreviewUrl(playableUrl)
          if (seenTerminalToastRef.current !== taskId) {
            seenTerminalToastRef.current = taskId
            toast.success(t('Video generation completed'))
          }
          return
        }
        if (isTerminalFailure(status)) {
          const rawMessage =
            statusResponse.error?.message ||
            statusResponse.fail_reason ||
            ''
          const message = rawMessage
            ? localizeTaskFailReason(rawMessage, t)
            : t('Video generation failed')
          setPollError(message)
          if (seenTerminalToastRef.current !== taskId) {
            seenTerminalToastRef.current = taskId
            toast.error(message)
          }
          return
        }
      } catch (error) {
        if (cancelled) return
        setPollError(
          error instanceof Error
            ? error.message
            : t('Failed to fetch video task')
        )
      }
      if (!cancelled) {
        timer = setTimeout(() => {
          void pollOnce()
        }, POLL_INTERVAL_MS)
      }
    }

    timer = setTimeout(() => {
      void pollOnce()
    }, 800)

    return () => {
      cancelled = true
      if (timer) clearTimeout(timer)
    }
  }, [pollingPaused, pollingTokenKey, t, taskId])

  return {
    taskId,
    taskStatus,
    taskProgress,
    previewUrl,
    pollError,
    pollingPaused,
    pollingTokenKey,
    isPolling,
    reset,
    start,
    failSubmission,
    resume,
  }
}
