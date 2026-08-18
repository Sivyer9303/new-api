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
    AIStarsLab: 'AIStarsLab',
    'Configure resolutions for each public model name you configured on the channel. Unlisted models keep 720p, 1080p, and 1K.':
      'Configure resolutions for each public model name you configured on the channel. Unlisted models keep 720p, 1080p, and 1K.',
    'Use the channel public model name or the upstream model name. Unlisted models keep 720p, 1080p, and 1K.':
      'Use the channel public model name or the upstream model name. Unlisted models keep 720p, 1080p, and 1K.',
    'Comma-separated, for example 720p, 1080p':
      'Comma-separated, for example 720p, 1080p',
    'Provider task failed': 'Provider task failed',
    'Provider completed the task without a usable result; administrator review is required':
      'Provider completed the task without a usable result; administrator review is required',
    'Provider returned an unknown task status; administrator review is required':
      'Provider returned an unknown task status; administrator review is required',
    'Optional companion audio. Numbered as @音频1. MP3 or WAV; cannot be the only reference.':
      'Optional companion audio. Numbered as @音频1. MP3 or WAV; cannot be the only reference.',
    'Use @视频N / @图片N in the prompt. MP4 or MOV; 1–3 clips; total duration ≤{{seconds}}s.':
      'Use @视频N / @图片N in the prompt. MP4 or MOV; 1–3 clips; total duration ≤{{seconds}}s.',
    'Compatible Video': 'Compatible Video',
    xtoken: 'xtoken',
    'Task submission did not complete.': 'Task submission did not complete.',
    'Task submission or billing reservation outcome is uncertain; administrator review is required.':
      'Task submission or billing reservation outcome is uncertain; administrator review is required.',
    'Task submission outcome is uncertain after timeout; administrator review is required':
      'Task submission outcome is uncertain after timeout; administrator review is required',
    'Video billing settlement requires administrator review':
      'Video billing settlement requires administrator review',
    'Video billing settlement recovery requires administrator review':
      'Video billing settlement recovery requires administrator review',
    'Generate audio': 'Generate audio',
    'Inherit built-in default': 'Inherit built-in default',
    'Select an upstream format': 'Select an upstream format',
    'Override capability defaults per built-in model profile. Leave a field empty to keep the built-in default.':
      'Override capability defaults per built-in model profile. Leave a field empty to keep the built-in default.',
  },
  zh: {
    AIStarsLab: 'AIStarsLab',
    'Configure resolutions for each public model name you configured on the channel. Unlisted models keep 720p, 1080p, and 1K.':
      '按你在渠道里配置的对外模型名设置分辨率。未列出的模型仍使用 720p、1080p 和 1K。',
    'Use the channel public model name or the upstream model name. Unlisted models keep 720p, 1080p, and 1K.':
      '可填写渠道对外模型名或上游模型名。未列出的模型仍使用 720p、1080p 和 1K。',
    'Comma-separated, for example 720p, 1080p': '用逗号分隔，例如 720p, 1080p',
    'Provider task failed': '视频任务失败',
    'Provider completed the task without a usable result; administrator review is required':
      '视频已完成但没有可用结果，请联系管理员审核。',
    'Provider returned an unknown task status; administrator review is required':
      '视频任务状态未知，请联系管理员审核。',
    'Optional companion audio. Numbered as @音频1. MP3 or WAV; cannot be the only reference.':
      '可选配套音频。按 @音频1 编号。支持 MP3 或 WAV；不能作为唯一参考。',
    'Use @视频N / @图片N in the prompt. MP4 or MOV; 1–3 clips; total duration ≤{{seconds}}s.':
      '请在提示词中使用 @视频N / @图片N。支持 MP4 或 MOV；1–3 段；总时长 ≤{{seconds}} 秒。',
    'Compatible Video': '兼容视频',
    xtoken: 'xtoken',
    'Task submission did not complete.': '任务提交未完成。',
    'Task submission or billing reservation outcome is uncertain; administrator review is required.':
      '任务提交或预扣结果不确定，请管理员复核。',
    'Task submission outcome is uncertain after timeout; administrator review is required':
      '任务提交超时后结果不确定，请管理员复核。',
    'Video billing settlement requires administrator review':
      '视频计费结算需要管理员复核。',
    'Video billing settlement recovery requires administrator review':
      '视频计费结算恢复需要管理员复核。',
    'Generate audio': '生成音频',
    'Inherit built-in default': '继承内置默认值',
    'Select an upstream format': '选择上游格式',
    'Override capability defaults per built-in model profile. Leave a field empty to keep the built-in default.':
      '按内置模型配置覆盖能力默认值。留空则使用内置默认值。',
  },
  'zh-TW': {
    AIStarsLab: 'AIStarsLab',
    'Configure resolutions for each public model name you configured on the channel. Unlisted models keep 720p, 1080p, and 1K.':
      '依你在渠道中設定的對外模型名稱設定解析度。未列出的模型仍使用 720p、1080p 和 1K。',
    'Use the channel public model name or the upstream model name. Unlisted models keep 720p, 1080p, and 1K.':
      '可填寫渠道對外模型名或上游模型名。未列出的模型仍使用 720p、1080p 和 1K。',
    'Comma-separated, for example 720p, 1080p': '以逗號分隔，例如 720p, 1080p',
    'Provider task failed': '影片任務失敗',
    'Provider completed the task without a usable result; administrator review is required':
      '影片已完成但沒有可用結果，請聯絡管理員審核。',
    'Provider returned an unknown task status; administrator review is required':
      '影片任務狀態未知，請聯絡管理員審核。',
    'Optional companion audio. Numbered as @音频1. MP3 or WAV; cannot be the only reference.':
      '可選搭配音訊。以 @音频1 編號。支援 MP3 或 WAV；不能作為唯一參考。',
    'Use @视频N / @图片N in the prompt. MP4 or MOV; 1–3 clips; total duration ≤{{seconds}}s.':
      '請在提示詞使用 @视频N / @图片N。支援 MP4 或 MOV；1–3 段；總時長 ≤{{seconds}} 秒。',
    'Compatible Video': '相容影片',
    xtoken: 'xtoken',
    'Task submission did not complete.': '任務提交未完成。',
    'Task submission or billing reservation outcome is uncertain; administrator review is required.':
      '任務提交或預扣結果不確定，請管理員複核。',
    'Task submission outcome is uncertain after timeout; administrator review is required':
      '任務提交逾時後結果不確定，請管理員複核。',
    'Video billing settlement requires administrator review':
      '影片計費結算需要管理員複核。',
    'Video billing settlement recovery requires administrator review':
      '影片計費結算恢復需要管理員複核。',
    'Generate audio': '產生音訊',
    'Inherit built-in default': '繼承內建預設值',
    'Select an upstream format': '選擇上游格式',
    'Override capability defaults per built-in model profile. Leave a field empty to keep the built-in default.':
      '依內建模型設定覆寫能力預設值。留空即可保留內建預設值。',
  },
  fr: {
    AIStarsLab: 'AIStarsLab',
    'Configure resolutions for each public model name you configured on the channel. Unlisted models keep 720p, 1080p, and 1K.':
      'Configurez les résolutions pour chaque nom de modèle public défini sur le canal. Les modèles non listés conservent 720p, 1080p et 1K.',
    'Use the channel public model name or the upstream model name. Unlisted models keep 720p, 1080p, and 1K.':
      'Utilisez le nom de modèle public du canal ou le nom de modèle amont. Les modèles non listés conservent 720p, 1080p et 1K.',
    'Comma-separated, for example 720p, 1080p':
      'Séparées par des virgules, par exemple 720p, 1080p',
    'Provider task failed': 'Échec de la tâche vidéo',
    'Provider completed the task without a usable result; administrator review is required':
      'La tâche est terminée sans résultat utilisable ; une revue administrateur est requise.',
    'Provider returned an unknown task status; administrator review is required':
      'Statut de tâche inconnu ; une revue administrateur est requise.',
    'Optional companion audio. Numbered as @音频1. MP3 or WAV; cannot be the only reference.':
      'Audio d’accompagnement optionnel. Numéroté @音频1. MP3 ou WAV ; ne peut pas être la seule référence.',
    'Use @视频N / @图片N in the prompt. MP4 or MOV; 1–3 clips; total duration ≤{{seconds}}s.':
      'Utilisez @视频N / @图片N dans le prompt. MP4 ou MOV ; 1–3 clips ; durée totale ≤{{seconds}} s.',
    'Compatible Video': 'Vidéo compatible',
    xtoken: 'xtoken',
    'Task submission did not complete.': 'La soumission de la tâche n’est pas terminée.',
    'Task submission or billing reservation outcome is uncertain; administrator review is required.':
      'Le résultat de la soumission ou de la réservation de quota est incertain ; une revue administrateur est requise.',
    'Task submission outcome is uncertain after timeout; administrator review is required':
      'Le résultat de la soumission est incertain après expiration du délai ; une revue administrateur est requise.',
    'Video billing settlement requires administrator review':
      'Le règlement de facturation vidéo nécessite une revue administrateur.',
    'Video billing settlement recovery requires administrator review':
      'La reprise du règlement de facturation vidéo nécessite une revue administrateur.',
    'Generate audio': 'Générer l’audio',
    'Inherit built-in default': 'Hériter de la valeur par défaut intégrée',
    'Select an upstream format': 'Sélectionner un format amont',
    'Override capability defaults per built-in model profile. Leave a field empty to keep the built-in default.':
      'Remplacez les capacités par défaut pour chaque profil intégré. Laissez un champ vide pour conserver sa valeur par défaut.',
  },
  ja: {
    AIStarsLab: 'AIStarsLab',
    'Configure resolutions for each public model name you configured on the channel. Unlisted models keep 720p, 1080p, and 1K.':
      'チャネルに設定した公開モデル名ごとに解像度を指定します。未設定のモデルは 720p、1080p、1K のままです。',
    'Use the channel public model name or the upstream model name. Unlisted models keep 720p, 1080p, and 1K.':
      'チャネルの公開モデル名または上流モデル名を指定します。未設定のモデルは 720p、1080p、1K のままです。',
    'Comma-separated, for example 720p, 1080p': 'カンマ区切り、例: 720p, 1080p',
    'Provider task failed': '動画タスクに失敗しました',
    'Provider completed the task without a usable result; administrator review is required':
      'タスクは完了しましたが利用可能な結果がありません。管理者による確認が必要です。',
    'Provider returned an unknown task status; administrator review is required':
      'タスクの状態が不明です。管理者による確認が必要です。',
    'Optional companion audio. Numbered as @音频1. MP3 or WAV; cannot be the only reference.':
      '任意の補助音声です。@音频1 と番号付けされます。MP3 または WAV。音声だけでは参照にできません。',
    'Use @视频N / @图片N in the prompt. MP4 or MOV; 1–3 clips; total duration ≤{{seconds}}s.':
      'プロンプトでは @视频N / @图片N を使ってください。MP4 または MOV、1–3 本、合計時間 ≤{{seconds}} 秒。',
    'Compatible Video': '互換ビデオ',
    xtoken: 'xtoken',
    'Task submission did not complete.': 'タスクの送信が完了しませんでした。',
    'Task submission or billing reservation outcome is uncertain; administrator review is required.':
      'タスク送信または課金予約の結果が不明です。管理者による確認が必要です。',
    'Task submission outcome is uncertain after timeout; administrator review is required':
      'タイムアウト後のタスク送信結果が不明です。管理者による確認が必要です。',
    'Video billing settlement requires administrator review':
      '動画の課金精算は管理者による確認が必要です。',
    'Video billing settlement recovery requires administrator review':
      '動画の課金精算の復旧は管理者による確認が必要です。',
    'Generate audio': '音声を生成',
    'Inherit built-in default': '組み込みの既定値を継承',
    'Select an upstream format': '上流形式を選択',
    'Override capability defaults per built-in model profile. Leave a field empty to keep the built-in default.':
      '組み込みモデルプロファイルごとに機能の既定値を上書きします。空欄の項目は組み込みの既定値を使用します。',
  },
  ru: {
    AIStarsLab: 'AIStarsLab',
    'Configure resolutions for each public model name you configured on the channel. Unlisted models keep 720p, 1080p, and 1K.':
      'Настройте разрешения для каждого публичного имени модели, заданного на канале. Для остальных моделей остаются 720p, 1080p и 1K.',
    'Use the channel public model name or the upstream model name. Unlisted models keep 720p, 1080p, and 1K.':
      'Укажите публичное имя модели канала или имя модели провайдера. Для остальных моделей остаются 720p, 1080p и 1K.',
    'Comma-separated, for example 720p, 1080p':
      'Через запятую, например 720p, 1080p',
    'Provider task failed': 'Ошибка видеозадачи',
    'Provider completed the task without a usable result; administrator review is required':
      'Задача завершена без пригодного результата; требуется проверка администратора.',
    'Provider returned an unknown task status; administrator review is required':
      'Неизвестный статус задачи; требуется проверка администратора.',
    'Optional companion audio. Numbered as @音频1. MP3 or WAV; cannot be the only reference.':
      'Необязательное сопутствующее аудио. Нумерация: @音频1. MP3 или WAV; не может быть единственной ссылкой.',
    'Use @视频N / @图片N in the prompt. MP4 or MOV; 1–3 clips; total duration ≤{{seconds}}s.':
      'Используйте @视频N / @图片N в запросе. MP4 или MOV; 1–3 клипа; общая длительность ≤{{seconds}} с.',
    'Compatible Video': 'Совместимое видео',
    xtoken: 'xtoken',
    'Task submission did not complete.': 'Отправка задачи не завершена.',
    'Task submission or billing reservation outcome is uncertain; administrator review is required.':
      'Результат отправки задачи или резервирования квоты неясен; требуется проверка администратора.',
    'Task submission outcome is uncertain after timeout; administrator review is required':
      'Результат отправки задачи после тайм-аута неясен; требуется проверка администратора.',
    'Video billing settlement requires administrator review':
      'Расчёт оплаты видео требует проверки администратора.',
    'Video billing settlement recovery requires administrator review':
      'Восстановление расчёта оплаты видео требует проверки администратора.',
    'Generate audio': 'Создать аудио',
    'Inherit built-in default': 'Использовать встроенное значение по умолчанию',
    'Select an upstream format': 'Выберите формат провайдера',
    'Override capability defaults per built-in model profile. Leave a field empty to keep the built-in default.':
      'Переопределите возможности для каждого встроенного профиля модели. Оставьте поле пустым, чтобы сохранить встроенное значение.',
  },
  vi: {
    AIStarsLab: 'AIStarsLab',
    'Configure resolutions for each public model name you configured on the channel. Unlisted models keep 720p, 1080p, and 1K.':
      'Cấu hình độ phân giải theo tên mô hình công khai bạn đặt trên kênh. Các mô hình chưa liệt kê vẫn dùng 720p, 1080p và 1K.',
    'Use the channel public model name or the upstream model name. Unlisted models keep 720p, 1080p, and 1K.':
      'Dùng tên mô hình công khai trên kênh hoặc tên mô hình thượng nguồn. Các mô hình chưa liệt kê vẫn dùng 720p, 1080p và 1K.',
    'Comma-separated, for example 720p, 1080p':
      'Phân tách bằng dấu phẩy, ví dụ 720p, 1080p',
    'Provider task failed': 'Tác vụ video thất bại',
    'Provider completed the task without a usable result; administrator review is required':
      'Tác vụ đã hoàn tất nhưng không có kết quả dùng được; cần quản trị viên xem xét.',
    'Provider returned an unknown task status; administrator review is required':
      'Trạng thái tác vụ không xác định; cần quản trị viên xem xét.',
    'Optional companion audio. Numbered as @音频1. MP3 or WAV; cannot be the only reference.':
      'Audio kèm theo tùy chọn. Đánh số @音频1. MP3 hoặc WAV; không thể là tham chiếu duy nhất.',
    'Use @视频N / @图片N in the prompt. MP4 or MOV; 1–3 clips; total duration ≤{{seconds}}s.':
      'Dùng @视频N / @图片N trong prompt. MP4 hoặc MOV; 1–3 clip; tổng thời lượng ≤{{seconds}} giây.',
    'Compatible Video': 'Video tương thích',
    xtoken: 'xtoken',
    'Task submission did not complete.': 'Gửi tác vụ chưa hoàn tất.',
    'Task submission or billing reservation outcome is uncertain; administrator review is required.':
      'Kết quả gửi tác vụ hoặc giữ hạn ngạch không rõ; cần quản trị viên xem xét.',
    'Task submission outcome is uncertain after timeout; administrator review is required':
      'Kết quả gửi tác vụ sau khi hết thời gian chờ không rõ; cần quản trị viên xem xét.',
    'Video billing settlement requires administrator review':
      'Quyết toán cước video cần quản trị viên xem xét.',
    'Video billing settlement recovery requires administrator review':
      'Khôi phục quyết toán cước video cần quản trị viên xem xét.',
    'Generate audio': 'Tạo âm thanh',
    'Inherit built-in default': 'Kế thừa giá trị mặc định tích hợp',
    'Select an upstream format': 'Chọn định dạng thượng nguồn',
    'Override capability defaults per built-in model profile. Leave a field empty to keep the built-in default.':
      'Ghi đè khả năng mặc định cho từng cấu hình mẫu tích hợp. Để trống để giữ giá trị mặc định tích hợp.',
  },
}

async function main() {
  let totalAdded = 0

  for (const [locale, trans] of Object.entries(newKeys)) {
    const filePath = path.join(LOCALES_DIR, `${locale}.json`)
    const json = JSON.parse(await fs.readFile(filePath, 'utf8'))

    let count = 0
    for (const [key, value] of Object.entries(trans)) {
      if (!Object.hasOwn(json.translation, key)) {
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
