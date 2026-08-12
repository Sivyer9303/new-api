/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { createFileRoute, redirect } from '@tanstack/react-router'

import { VideoSettingsPage } from '@/features/system-settings/video'
import {
  VIDEO_DEFAULT_SECTION,
  VIDEO_SECTION_IDS,
} from '@/features/system-settings/video/section-registry'

export const Route = createFileRoute(
  '/_authenticated/system-settings/video/$section'
)({
  beforeLoad: ({ params }) => {
    const validSections = VIDEO_SECTION_IDS as unknown as string[]
    if (!validSections.includes(params.section)) {
      throw redirect({
        to: '/system-settings/video/$section',
        params: { section: VIDEO_DEFAULT_SECTION },
      })
    }
  },
  component: VideoSettingsPage,
})
