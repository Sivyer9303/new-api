import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')

function stableStringify(obj) {
  return JSON.stringify(obj, null, 2) + '\n'
}

const newKeys = {
  en: {
    'Preview only — image/audio base64 is shortened here. On submit, full data:…;base64,… payloads are sent.':
      'Preview only — image/audio base64 is shortened here. On submit, full data:…;base64,… payloads are sent.',
  },
  zh: {
    'Preview only — image/audio base64 is shortened here. On submit, full data:…;base64,… payloads are sent.':
      '仅预览：图片/音频的 base64 在此缩短显示。真正提交时会发送完整的 data:…;base64,… 内容。',
  },
  'zh-TW': {
    'Preview only — image/audio base64 is shortened here. On submit, full data:…;base64,… payloads are sent.':
      '僅預覽：圖片/音訊的 base64 在此縮短顯示。真正送出時會傳送完整的 data:…;base64,… 內容。',
  },
  fr: {
    'Preview only — image/audio base64 is shortened here. On submit, full data:…;base64,… payloads are sent.':
      'Aperçu uniquement — le base64 image/audio est raccourci ici. À l’envoi, les payloads data:…;base64,… complets sont transmis.',
  },
  ja: {
    'Preview only — image/audio base64 is shortened here. On submit, full data:…;base64,… payloads are sent.':
      'プレビューのみ — 画像/音声の base64 はここでは短縮表示します。送信時は完全な data:…;base64,… を送ります。',
  },
  ru: {
    'Preview only — image/audio base64 is shortened here. On submit, full data:…;base64,… payloads are sent.':
      'Только превью — base64 изображений/аудио здесь сокращён. При отправке уходят полные data:…;base64,….',
  },
  vi: {
    'Preview only — image/audio base64 is shortened here. On submit, full data:…;base64,… payloads are sent.':
      'Chỉ xem trước — base64 ảnh/âm thanh được rút gọn tại đây. Khi gửi sẽ dùng đầy đủ data:…;base64,….',
  },
}

async function main() {
  let totalAdded = 0

  for (const [locale, trans] of Object.entries(newKeys)) {
    const filePath = path.join(LOCALES_DIR, `${locale}.json`)
    const json = JSON.parse(await fs.readFile(filePath, 'utf8'))

    let count = 0
    for (const [key, value] of Object.entries(trans)) {
      if (!(key in json.translation) || json.translation[key] !== value) {
        json.translation[key] = value
        count++
      }
    }

    const sorted = Object.keys(json.translation)
      .sort((a, b) => a.localeCompare(b))
      .reduce((acc, k) => {
        acc[k] = json.translation[k]
        return acc
      }, {})
    json.translation = sorted

    await fs.writeFile(filePath, stableStringify(json))
    console.log(`${locale}: upserted ${count} keys`)
    totalAdded += count
  }

  console.log(`Done. Total upserts: ${totalAdded}`)
}

await main()
