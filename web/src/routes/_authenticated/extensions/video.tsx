/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { createFileRoute, redirect } from '@tanstack/react-router'

import { VideoToolPage } from '@/features/extensions/video-tool'
import { getStatus } from '@/lib/api'

export const Route = createFileRoute('/_authenticated/extensions/video')({
  beforeLoad: async () => {
    try {
      const status = await getStatus()
      if (
        !(status?.video_tool_enabled ?? status?.silkroad_video_tool_enabled)
      ) {
        throw redirect({ to: '/dashboard' })
      }
    } catch (error) {
      if (error && typeof error === 'object' && 'to' in error) {
        throw error
      }
      throw redirect({ to: '/dashboard' })
    }
  },
  component: VideoToolPage,
})
