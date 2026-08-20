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
import {
  deleteVideoInputAsset,
  uploadVideoInputAsset,
  type VideoInputAssetKind,
  type VideoInputUploadStatus,
} from './input-asset-upload'

export type UploadTrackedMedia = {
  id: string
  file: File
  uploadStatus: VideoInputUploadStatus
  uploadProgress: number
  assetId?: string
  sourceUrl?: string
  expiresAt?: number
  uploadError?: string
}

type UploadPatch = Partial<
  Pick<
    UploadTrackedMedia,
    | 'uploadStatus'
    | 'uploadProgress'
    | 'assetId'
    | 'sourceUrl'
    | 'expiresAt'
    | 'uploadError'
  >
>

/**
 * Starts a direct R2 upload for every pending item. Callers pass a stable
 * in-flight set so React effect re-runs do not duplicate work.
 */
export function kickPendingMediaUploads<T extends UploadTrackedMedia>(
  items: T[],
  kind: VideoInputAssetKind,
  inFlight: Set<string>,
  patchItem: (id: string, patch: UploadPatch) => void
) {
  for (const item of items) {
    if (item.uploadStatus !== 'pending') continue
    if (inFlight.has(item.id)) continue
    inFlight.add(item.id)
    patchItem(item.id, {
      uploadStatus: 'uploading',
      uploadProgress: 0,
      uploadError: undefined,
    })
    void uploadVideoInputAsset(item.file, kind, {
      onProgress: (percent) => {
        patchItem(item.id, { uploadProgress: percent })
      },
    })
      .then((result) => {
        patchItem(item.id, {
          uploadStatus: 'ready',
          uploadProgress: 100,
          assetId: result.assetId,
          sourceUrl: result.url,
          expiresAt: result.expiresAt,
          uploadError: undefined,
        })
      })
      .catch((err: unknown) => {
        const message =
          err instanceof Error ? err.message : 'Failed to upload media'
        patchItem(item.id, {
          uploadStatus: 'failed',
          uploadProgress: 0,
          uploadError: message,
        })
      })
      .finally(() => {
        inFlight.delete(item.id)
      })
  }
}

export async function abandonMediaUpload(
  item: Pick<UploadTrackedMedia, 'assetId'> | null | undefined
) {
  if (item?.assetId) {
    await deleteVideoInputAsset(item.assetId)
  }
}
