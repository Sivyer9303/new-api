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

/**
 * Resolve a playable video URL for authenticated dashboard/API preview.
 * Object storage (R2) returns a short-lived signed URL via JSON; local storage
 * streams bytes that become a blob URL.
 */
export async function resolveAuthenticatedVideoPlaybackUrl(
  contentPath: string,
  authorization: string
): Promise<{ url: string; revoke?: () => void }> {
  const jsonRes = await fetch(contentPath, {
    headers: {
      Authorization: authorization,
      Accept: 'application/json',
    },
  })

  if (!jsonRes.ok) {
    throw new Error(`Failed to load video preview (${jsonRes.status})`)
  }

  const contentType = (jsonRes.headers.get('content-type') || '').toLowerCase()
  if (contentType.includes('application/json')) {
    const body = (await jsonRes.json()) as { url?: unknown }
    if (typeof body.url === 'string' && body.url.trim()) {
      return { url: body.url.trim() }
    }
    throw new Error('Failed to load video preview')
  }

  const blob = await jsonRes.blob()
  const url = URL.createObjectURL(blob)
  return { url, revoke: () => URL.revokeObjectURL(url) }
}
