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
    'Provider task failed': 'Provider task failed',
    'Provider completed the task without a usable result; administrator review is required':
      'Provider completed the task without a usable result; administrator review is required',
    'Provider returned an unknown task status; administrator review is required':
      'Provider returned an unknown task status; administrator review is required',
    'Optional companion audio. Numbered as @音频1. MP3 or WAV; cannot be the only reference.':
      'Optional companion audio. Numbered as @音频1. MP3 or WAV; cannot be the only reference.',
    'Use @视频N / @图片N in the prompt. MP4 or MOV; 1–3 clips; total duration ≤{{seconds}}s.':
      'Use @视频N / @图片N in the prompt. MP4 or MOV; 1–3 clips; total duration ≤{{seconds}}s.',
  },
  zh: {
    'Provider task failed': '视频任务失败',
    'Provider completed the task without a usable result; administrator review is required':
      '视频已完成但没有可用结果，请联系管理员审核。',
    'Provider returned an unknown task status; administrator review is required':
      '视频任务状态未知，请联系管理员审核。',
    'Optional companion audio. Numbered as @音频1. MP3 or WAV; cannot be the only reference.':
      '可选配套音频。按 @音频1 编号。支持 MP3 或 WAV；不能作为唯一参考。',
    'Use @视频N / @图片N in the prompt. MP4 or MOV; 1–3 clips; total duration ≤{{seconds}}s.':
      '请在提示词中使用 @视频N / @图片N。支持 MP4 或 MOV；1–3 段；总时长 ≤{{seconds}} 秒。',
  },
  'zh-TW': {
    'Provider task failed': '影片任務失敗',
    'Provider completed the task without a usable result; administrator review is required':
      '影片已完成但沒有可用結果，請聯絡管理員審核。',
    'Provider returned an unknown task status; administrator review is required':
      '影片任務狀態未知，請聯絡管理員審核。',
    'Optional companion audio. Numbered as @音频1. MP3 or WAV; cannot be the only reference.':
      '可選搭配音訊。以 @音频1 編號。支援 MP3 或 WAV；不能作為唯一參考。',
    'Use @视频N / @图片N in the prompt. MP4 or MOV; 1–3 clips; total duration ≤{{seconds}}s.':
      '請在提示詞使用 @视频N / @图片N。支援 MP4 或 MOV；1–3 段；總時長 ≤{{seconds}} 秒。',
  },
  fr: {
    'Provider task failed': 'Échec de la tâche vidéo',
    'Provider completed the task without a usable result; administrator review is required':
      'La tâche est terminée sans résultat utilisable ; une revue administrateur est requise.',
    'Provider returned an unknown task status; administrator review is required':
      'Statut de tâche inconnu ; une revue administrateur est requise.',
    'Optional companion audio. Numbered as @音频1. MP3 or WAV; cannot be the only reference.':
      'Audio d’accompagnement optionnel. Numéroté @音频1. MP3 ou WAV ; ne peut pas être la seule référence.',
    'Use @视频N / @图片N in the prompt. MP4 or MOV; 1–3 clips; total duration ≤{{seconds}}s.':
      'Utilisez @视频N / @图片N dans le prompt. MP4 ou MOV ; 1–3 clips ; durée totale ≤{{seconds}} s.',
  },
  ja: {
    'Provider task failed': '動画タスクに失敗しました',
    'Provider completed the task without a usable result; administrator review is required':
      'タスクは完了しましたが利用可能な結果がありません。管理者による確認が必要です。',
    'Provider returned an unknown task status; administrator review is required':
      'タスクの状態が不明です。管理者による確認が必要です。',
    'Optional companion audio. Numbered as @音频1. MP3 or WAV; cannot be the only reference.':
      '任意の補助音声です。@音频1 と番号付けされます。MP3 または WAV。音声だけでは参照にできません。',
    'Use @视频N / @图片N in the prompt. MP4 or MOV; 1–3 clips; total duration ≤{{seconds}}s.':
      'プロンプトでは @视频N / @图片N を使ってください。MP4 または MOV、1–3 本、合計時間 ≤{{seconds}} 秒。',
  },
  ru: {
    'Provider task failed': 'Ошибка видеозадачи',
    'Provider completed the task without a usable result; administrator review is required':
      'Задача завершена без пригодного результата; требуется проверка администратора.',
    'Provider returned an unknown task status; administrator review is required':
      'Неизвестный статус задачи; требуется проверка администратора.',
    'Optional companion audio. Numbered as @音频1. MP3 or WAV; cannot be the only reference.':
      'Необязательное сопутствующее аудио. Нумерация: @音频1. MP3 или WAV; не может быть единственной ссылкой.',
    'Use @视频N / @图片N in the prompt. MP4 or MOV; 1–3 clips; total duration ≤{{seconds}}s.':
      'Используйте @视频N / @图片N в запросе. MP4 или MOV; 1–3 клипа; общая длительность ≤{{seconds}} с.',
  },
  vi: {
    'Provider task failed': 'Tác vụ video thất bại',
    'Provider completed the task without a usable result; administrator review is required':
      'Tác vụ đã hoàn tất nhưng không có kết quả dùng được; cần quản trị viên xem xét.',
    'Provider returned an unknown task status; administrator review is required':
      'Trạng thái tác vụ không xác định; cần quản trị viên xem xét.',
    'Optional companion audio. Numbered as @音频1. MP3 or WAV; cannot be the only reference.':
      'Audio kèm theo tùy chọn. Đánh số @音频1. MP3 hoặc WAV; không thể là tham chiếu duy nhất.',
    'Use @视频N / @图片N in the prompt. MP4 or MOV; 1–3 clips; total duration ≤{{seconds}}s.':
      'Dùng @视频N / @图片N trong prompt. MP4 hoặc MOV; 1–3 clip; tổng thời lượng ≤{{seconds}} giây.',
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
