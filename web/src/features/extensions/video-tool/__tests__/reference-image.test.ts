/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import assert from 'node:assert/strict'
import { test } from 'node:test'

import {
  createReferenceImageItem,
  revokeReferenceImageItems,
} from '../lib/reference-image'

test('reference previews are bounded thumbnails with explicit URL cleanup', async () => {
  const originalImage = globalThis.Image
  const originalDocument = globalThis.document
  const originalCreateObjectURL = URL.createObjectURL
  const originalRevokeObjectURL = URL.revokeObjectURL
  const revoked: string[] = []
  const canvas = {
    width: 0,
    height: 0,
    getContext: () => ({
      drawImage: () => undefined,
    }),
    toBlob: (callback: BlobCallback) =>
      callback(new Blob(['thumbnail'], { type: 'image/webp' })),
  }
  let objectURLCount = 0

  class TestImage {
    naturalWidth = 4000
    naturalHeight = 2000
    src = ''

    async decode() {}
  }

  Object.defineProperty(globalThis, 'Image', {
    configurable: true,
    value: TestImage,
  })
  Object.defineProperty(globalThis, 'document', {
    configurable: true,
    value: {
      createElement: (tag: string) => {
        assert.equal(tag, 'canvas')
        return canvas
      },
    },
  })
  URL.createObjectURL = () => {
    objectURLCount += 1
    return objectURLCount === 1 ? 'blob:source' : 'blob:thumbnail'
  }
  URL.revokeObjectURL = (url) => {
    revoked.push(url)
  }

  try {
    const file = new File(['full-size-image'], 'reference.png', {
      type: 'image/png',
    })
    const item = await createReferenceImageItem(file)

    assert.equal(canvas.width, 320)
    assert.equal(canvas.height, 160)
    assert.equal(item.previewUrl, 'blob:thumbnail')
    assert.deepEqual(revoked, ['blob:source'])

    revokeReferenceImageItems([item])
    assert.deepEqual(revoked, ['blob:source', 'blob:thumbnail'])
  } finally {
    Object.defineProperty(globalThis, 'Image', {
      configurable: true,
      value: originalImage,
    })
    Object.defineProperty(globalThis, 'document', {
      configurable: true,
      value: originalDocument,
    })
    URL.createObjectURL = originalCreateObjectURL
    URL.revokeObjectURL = originalRevokeObjectURL
  }
})
