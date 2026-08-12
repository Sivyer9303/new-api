/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { createFileRoute, redirect } from '@tanstack/react-router'

import { VIDEO_DEFAULT_SECTION } from '@/features/system-settings/video/section-registry'

export const Route = createFileRoute('/_authenticated/system-settings/video/')({
  beforeLoad: () => {
    throw redirect({
      to: '/system-settings/video/$section',
      params: { section: VIDEO_DEFAULT_SECTION },
    })
  },
})
