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
import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')

function stableStringify(obj) {
  return `${JSON.stringify(obj, null, 2)}\n`
}

const newKeys = {
  en: {
    'View request': 'View request',
    'Request parameters': 'Request parameters',
    'Request parameters for this task': 'Request parameters for this task',
    'No request parameters were recorded for this task.':
      'No request parameters were recorded for this task.',
    'Generation type': 'Generation type',
    Size: 'Size',
    Attachments: 'Attachments',
  },
  zh: {
    'View request': '查看请求',
    'Request parameters': '请求参数',
    'Request parameters for this task': '查看该任务提交时的请求参数',
    'No request parameters were recorded for this task.':
      '该任务没有记录请求参数。',
    'Generation type': '生成类型',
    Size: '尺寸',
    Attachments: '附件',
  },
  'zh-TW': {
    'View request': '查看請求',
    'Request parameters': '請求參數',
    'Request parameters for this task': '查看此任務送出時的請求參數',
    'No request parameters were recorded for this task.':
      '此任務沒有記錄請求參數。',
    'Generation type': '產生類型',
    Size: '尺寸',
    Attachments: '附件',
  },
  fr: {
    'View request': 'Voir la requête',
    'Request parameters': 'Paramètres de la requête',
    'Request parameters for this task':
      'Paramètres envoyés pour cette tâche',
    'No request parameters were recorded for this task.':
      'Aucun paramètre de requête n’a été enregistré pour cette tâche.',
    'Generation type': 'Type de génération',
    Size: 'Taille',
    Attachments: 'Pièces jointes',
  },
  ja: {
    'View request': 'リクエストを表示',
    'Request parameters': 'リクエストパラメータ',
    'Request parameters for this task':
      'このタスク送信時のリクエストパラメータ',
    'No request parameters were recorded for this task.':
      'このタスクのリクエストパラメータは記録されていません。',
    'Generation type': '生成タイプ',
    Size: 'サイズ',
    Attachments: '添付',
  },
  ru: {
    'View request': 'Посмотреть запрос',
    'Request parameters': 'Параметры запроса',
    'Request parameters for this task':
      'Параметры запроса, отправленные для этой задачи',
    'No request parameters were recorded for this task.':
      'Для этой задачи параметры запроса не сохранены.',
    'Generation type': 'Тип генерации',
    Size: 'Размер',
    Attachments: 'Вложения',
  },
  vi: {
    'View request': 'Xem yêu cầu',
    'Request parameters': 'Tham số yêu cầu',
    'Request parameters for this task':
      'Tham số yêu cầu đã gửi cho tác vụ này',
    'No request parameters were recorded for this task.':
      'Tác vụ này không ghi lại tham số yêu cầu.',
    'Generation type': 'Kiểu tạo',
    Size: 'Kích thước',
    Attachments: 'Tệp đính kèm',
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
