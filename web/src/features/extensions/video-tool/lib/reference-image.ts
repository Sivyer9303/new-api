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

export type ReferenceImageItem = {
  id: string
  file: File
  previewUrl: string
}

const THUMBNAIL_MAX_EDGE = 320

export async function createReferenceImageItem(
  file: File
): Promise<ReferenceImageItem> {
  const sourceUrl = URL.createObjectURL(file)
  let previewUrl = sourceUrl
  let image: HTMLImageElement | null = null

  try {
    image = new Image()
    image.src = sourceUrl
    await image.decode()

    const scale = Math.min(
      1,
      THUMBNAIL_MAX_EDGE / Math.max(image.naturalWidth, image.naturalHeight)
    )
    const canvas = document.createElement('canvas')
    canvas.width = Math.max(1, Math.round(image.naturalWidth * scale))
    canvas.height = Math.max(1, Math.round(image.naturalHeight * scale))
    const context = canvas.getContext('2d')
    if (context) {
      context.drawImage(image, 0, 0, canvas.width, canvas.height)
      const thumbnail = await new Promise<Blob | null>((resolve) => {
        canvas.toBlob(resolve, 'image/webp', 0.82)
      })
      if (thumbnail) {
        previewUrl = URL.createObjectURL(thumbnail)
        URL.revokeObjectURL(sourceUrl)
      }
    }
  } catch {
    // The original object URL remains a safe fallback if thumbnailing fails.
  } finally {
    if (image) image.src = ''
  }

  return {
    id: `${file.name}-${file.size}-${file.lastModified}-${Math.random().toString(36).slice(2, 8)}`,
    file,
    previewUrl,
  }
}

export function revokeReferenceImageItems(items: ReferenceImageItem[]) {
  for (const item of items) {
    URL.revokeObjectURL(item.previewUrl)
  }
}
