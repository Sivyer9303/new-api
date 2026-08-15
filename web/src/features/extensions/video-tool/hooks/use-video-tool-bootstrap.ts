/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { getApiKeys, getTokenAutoGroups } from '@/features/keys/api'
import type { ApiKey } from '@/features/keys/types'
import { getUserGroups } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { fetchVideoToolConfig } from '../api'
import { isVideoTokenGroupCandidate } from '../lib/provider-config'

export function useVideoToolBootstrap() {
  const { t } = useTranslation()
  const userGroup = useAuthStore((state) => state.auth.user?.group ?? '')

  const configQuery = useQuery({
    queryKey: ['video-tool-config', 1],
    queryFn: async () => {
      const response = await fetchVideoToolConfig()
      if (!response.success || !response.data) {
        throw new Error(response.message || 'Failed to load video tool config')
      }
      if (response.data.version < 1 || response.data.version > 3) {
        throw new Error(t('Unsupported video tool capability version'))
      }
      return response.data
    },
  })

  const keysQuery = useQuery({
    queryKey: ['video-tool-api-keys'],
    queryFn: async () => {
      const response = await getApiKeys({ p: 1, size: 100 })
      if (!response.success || !response.data?.items) {
        throw new Error(response.message || 'Failed to load API keys')
      }
      return response.data.items.filter((key: ApiKey) => key.status === 1)
    },
  })

  const tokenAutoGroupsQuery = useQuery({
    queryKey: ['token-auto-groups'],
    queryFn: async () => {
      const response = await getTokenAutoGroups()
      if (!response.success || !response.data) {
        throw new Error(
          response.message || 'Failed to load Auto group settings'
        )
      }
      return response
    },
  })

  const userGroupsQuery = useQuery({
    queryKey: ['user-groups'],
    queryFn: async () => {
      const response = await getUserGroups()
      if (!response.success || !response.data) {
        throw new Error(response.message || 'Failed to load selectable groups')
      }
      return response
    },
  })

  const selectableTokenGroups = useMemo(
    () =>
      new Set(
        Object.keys(userGroupsQuery.data?.data ?? {}).filter(
          (group) => group !== 'auto'
        )
      ),
    [userGroupsQuery.data?.data]
  )

  const keys = useMemo(() => {
    const config = configQuery.data
    const autoGroupConfig = tokenAutoGroupsQuery.data?.data
    const allKeys = keysQuery.data ?? []
    if (!config || !autoGroupConfig || !userGroupsQuery.data?.data) return []

    return allKeys.filter((key) => {
      const storedAutoGroups = key.auto_groups ?? []
      const hasCustomAutoGroups = storedAutoGroups.length > 0
      const tokenAutoGroups = hasCustomAutoGroups
        ? storedAutoGroups
        : autoGroupConfig.groups
      return isVideoTokenGroupCandidate(
        config,
        key.group?.trim() ?? '',
        userGroup,
        tokenAutoGroups,
        {
          selectableGroups: selectableTokenGroups,
          maxAutoGroups: hasCustomAutoGroups
            ? autoGroupConfig.max_count
            : undefined,
        }
      )
    })
  }, [
    configQuery.data,
    keysQuery.data,
    selectableTokenGroups,
    tokenAutoGroupsQuery.data?.data,
    userGroup,
    userGroupsQuery.data?.data,
  ])

  let error: { title: string; cause: unknown } | null = null
  if (configQuery.isError) {
    error = {
      title: t('Failed to load video configuration'),
      cause: configQuery.error,
    }
  } else if (keysQuery.isError) {
    error = {
      title: t('Failed to load API keys'),
      cause: keysQuery.error,
    }
  } else if (tokenAutoGroupsQuery.isError || userGroupsQuery.isError) {
    error = {
      title: t('Failed to load API key group settings'),
      cause: tokenAutoGroupsQuery.error ?? userGroupsQuery.error,
    }
  }

  return {
    config: configQuery.data,
    keys,
    videoToolGroups: configQuery.data?.video_tool_groups ?? [],
    isLoading:
      configQuery.isLoading ||
      keysQuery.isLoading ||
      tokenAutoGroupsQuery.isLoading ||
      userGroupsQuery.isLoading,
    error,
    retry: () => {
      if (configQuery.isError) void configQuery.refetch()
      if (keysQuery.isError) void keysQuery.refetch()
      if (tokenAutoGroupsQuery.isError) void tokenAutoGroupsQuery.refetch()
      if (userGroupsQuery.isError) void userGroupsQuery.refetch()
    },
  }
}
