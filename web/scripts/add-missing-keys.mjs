import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')

function stableStringify(obj) {
  return JSON.stringify(obj, null, 2) + '\n'
}

const newKeys = {
  en: {
    'Fixed per-second price': 'Fixed per-second price',
    'Per Second': 'Per Second',
    'Per second': 'Per second',
    'Per-second': 'Per-second',
    'USD price per second of generated video.':
      'USD price per second of generated video.',
    Unit: 'Unit',
    'per second': 'per second',
    sec: 'sec',
  },
  zh: {
    'Fixed per-second price': '固定按秒价格',
    'Per Second': '按秒',
    'Per second': '按秒',
    'Per-second': '按秒',
    'USD price per second of generated video.': '生成视频的每秒美元单价。',
    Unit: '单位',
    'per second': '每秒',
    sec: '秒',
  },
  'zh-TW': {
    'Fixed per-second price': '固定按秒價格',
    'Per Second': '按秒',
    'Per second': '按秒',
    'Per-second': '按秒',
    'USD price per second of generated video.': '產生影片的每秒美元單價。',
    Unit: '單位',
    'per second': '每秒',
    sec: '秒',
  },
  fr: {
    'Fixed per-second price': 'Prix fixe à la seconde',
    'Per Second': 'À la seconde',
    'Per second': 'À la seconde',
    'Per-second': 'À la seconde',
    'USD price per second of generated video.':
      'Prix en USD par seconde de vidéo générée.',
    Unit: 'Unité',
    'per second': 'par seconde',
    sec: 's',
  },
  ja: {
    'Fixed per-second price': '秒単位の固定価格',
    'Per Second': '秒単位',
    'Per second': '秒単位',
    'Per-second': '秒単位',
    'USD price per second of generated video.':
      '生成動画の1秒あたりの米ドル単価。',
    Unit: '単位',
    'per second': '毎秒',
    sec: '秒',
  },
  ru: {
    'Fixed per-second price': 'Фиксированная цена за секунду',
    'Per Second': 'Посекундно',
    'Per second': 'Посекундно',
    'Per-second': 'Посекундно',
    'USD price per second of generated video.':
      'Цена в USD за секунду сгенерированного видео.',
    Unit: 'Единица',
    'per second': 'за секунду',
    sec: 'сек',
  },
  vi: {
    'Fixed per-second price': 'Giá cố định theo giây',
    'Per Second': 'Theo giây',
    'Per second': 'Theo giây',
    'Per-second': 'Theo giây',
    'USD price per second of generated video.':
      'Giá USD cho mỗi giây video được tạo.',
    Unit: 'Đơn vị',
    'per second': 'mỗi giây',
    sec: 'giây',
  },
}

async function main() {
  let totalAdded = 0

  for (const [locale, trans] of Object.entries(newKeys)) {
    const filePath = path.join(LOCALES_DIR, `${locale}.json`)
    const json = JSON.parse(await fs.readFile(filePath, 'utf8'))

    let count = 0
    for (const [key, value] of Object.entries(trans)) {
      if (!Object.prototype.hasOwnProperty.call(json.translation, key)) {
        json.translation[key] = value
        count++
      } else if (json.translation[key] !== value) {
        json.translation[key] = value
        count++
      }
    }

    if (count > 0) {
      json.translation = Object.fromEntries(
        Object.entries(json.translation).sort(([a], [b]) => a.localeCompare(b))
      )
      await fs.writeFile(filePath, stableStringify(json), 'utf8')
    }

    console.log(`${locale}: ${count} translations applied`)
    totalAdded += count
  }

  console.log(`\nTotal: ${totalAdded} translations applied`)
}

main().catch((err) => {
  console.error(err)
  process.exitCode = 1
})
