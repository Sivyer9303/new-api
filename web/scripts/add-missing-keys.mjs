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
    'Assign each group to only one video provider. Keys in these groups use this provider for models, capabilities, and task routing.':
      'Assign each group to only one video provider. Keys in these groups use this provider for models, capabilities, and task routing.',
    'Based on fixed model price × group ratio. Duration does not change this estimate. Reference only — final charge follows actual billing.':
      'Based on fixed model price × group ratio. Duration does not change this estimate. Reference only — final charge follows actual billing.',
    Brioi: 'Brioi',
    'Brioi requires R2 signed URLs because the upstream does not accept Base64 media.':
      'Brioi requires R2 signed URLs because the upstream does not accept Base64 media.',
    'Brioi profiles match exact upstream model names. Disable supported capabilities as needed; unsupported values cannot be added.':
      'Brioi profiles match exact upstream model names. Disable supported capabilities as needed; unsupported values cannot be added.',
    'Brioi Video': 'Brioi Video',
    'Enable at least one generation mode':
      'Enable at least one generation mode',
    'Enable at least one supported option':
      'Enable at least one supported option',
    'Failed to load API key group settings':
      'Failed to load API key group settings',
    'Failed to load video configuration': 'Failed to load video configuration',
    'First frame': 'First frame',
    'Image limit must be between {{min}} and {{max}} for this profile':
      'Image limit must be between {{min}} and {{max}} for this profile',
    'Image reference': 'Image reference',
    'Last frame': 'Last frame',
    'Maximum multi-image references': 'Maximum multi-image references',
    'Maximum reference images': 'Maximum reference images',
    'May be lowered for this provider profile, but cannot exceed the Brioi limit of {{max}}.':
      'May be lowered for this provider profile, but cannot exceed the Brioi limit of {{max}}.',
    'No eligible video models are available for this key and provider.':
      'No eligible video models are available for this key and provider.',
    'Option is outside the Brioi model hard capabilities':
      'Option is outside the Brioi model hard capabilities',
    'Preview only — image/audio base64 is shortened here. On submit, full data:…;base64,… payloads are sent.':
      'Preview only — image/audio base64 is shortened here. On submit, full data:…;base64,… payloads are sent.',
    'Provider groups': 'Provider groups',
    Reference: 'Reference',
    'Reference audio was cleared because it is not supported by the selected mode.':
      'Reference audio was cleared because it is not supported by the selected mode.',
    'Reference images were cleared because they are not supported by the selected mode.':
      'Reference images were cleared because they are not supported by the selected mode.',
    Resolution: 'Resolution',
    Resolutions: 'Resolutions',
    'Selected generation mode was cleared because it is not supported by the selected model.':
      'Selected generation mode was cleared because it is not supported by the selected model.',
    'Selected model was cleared because it does not support this generation mode.':
      'Selected model was cleared because it does not support this generation mode.',
    'Status refresh failed temporarily; retrying automatically.':
      'Status refresh failed temporarily; retrying automatically.',
    'Unknown Brioi model profile': 'Unknown Brioi model profile',
    '{{seconds}} seconds': '{{seconds}} seconds',
  },
  zh: {
    'Assign each group to only one video provider. Keys in these groups use this provider for models, capabilities, and task routing.':
      '每个分组只能分配给一个视频服务商。这些分组中的密钥将使用该服务商的模型、能力配置和任务路由。',
    'Based on fixed model price × group ratio. Duration does not change this estimate. Reference only — final charge follows actual billing.':
      '按模型固定价格 × 分组倍率估算，时长不会改变此价格。仅供参考，最终费用以实际计费为准。',
    Brioi: 'Brioi',
    'Brioi requires R2 signed URLs because the upstream does not accept Base64 media.':
      'Brioi 必须使用 R2 签名链接，因为上游不接受 Base64 素材。',
    'Brioi profiles match exact upstream model names. Disable supported capabilities as needed; unsupported values cannot be added.':
      'Brioi 档案按上游模型名称精确匹配。可按需禁用受支持的能力，但不能添加不支持的值。',
    'Brioi Video': 'Brioi 视频',
    'Enable at least one generation mode': '请至少启用一种生成模式',
    'Enable at least one supported option': '请至少启用一个支持的选项',
    'Failed to load API key group settings': '加载 API 密钥分组设置失败',
    'Failed to load video configuration': '加载视频配置失败',
    'First frame': '首帧',
    'Image limit must be between {{min}} and {{max}} for this profile':
      '此档案的图片数量限制必须在 {{min}} 到 {{max}} 之间',
    'Image reference': '图片参考',
    'Last frame': '尾帧',
    'Maximum multi-image references': '最大多图参考数',
    'Maximum reference images': '最大参考图片数',
    'May be lowered for this provider profile, but cannot exceed the Brioi limit of {{max}}.':
      '可为此服务商档案调低，但不能超过 Brioi 的 {{max}} 张限制。',
    'No eligible video models are available for this key and provider.':
      '此密钥和服务商没有可用的视频模型。',
    'Option is outside the Brioi model hard capabilities':
      '选项超出 Brioi 模型的硬性能力范围',
    'Preview only — image/audio base64 is shortened here. On submit, full data:…;base64,… payloads are sent.':
      '仅预览：图片/音频的 base64 在此缩短显示。真正提交时会发送完整的 data:…;base64,… 内容。',
    'Provider groups': '服务商分组',
    Reference: '参考',
    'Reference audio was cleared because it is not supported by the selected mode.':
      '参考音频不受所选模式支持，已清除。',
    'Reference images were cleared because they are not supported by the selected mode.':
      '参考图片不受所选模式支持，已清除。',
    Resolution: '分辨率',
    Resolutions: '分辨率',
    'Selected generation mode was cleared because it is not supported by the selected model.':
      '所选生成模式不受当前模型支持，已清除选择。',
    'Selected model was cleared because it does not support this generation mode.':
      '所选模型不支持此生成模式，已清除选择。',
    'Status refresh failed temporarily; retrying automatically.':
      '状态刷新暂时失败，正在自动重试。',
    'Unknown Brioi model profile': '未知的 Brioi 模型档案',
    '{{seconds}} seconds': '{{seconds}} 秒',
  },
  'zh-TW': {
    'Assign each group to only one video provider. Keys in these groups use this provider for models, capabilities, and task routing.':
      '每個群組只能指派給一個影片供應商。這些群組中的金鑰將使用該供應商的模型、能力設定與任務路由。',
    'Based on fixed model price × group ratio. Duration does not change this estimate. Reference only — final charge follows actual billing.':
      '依模型固定價格 × 群組倍率估算，時長不會改變此價格。僅供參考，最終費用以實際計費為準。',
    Brioi: 'Brioi',
    'Brioi requires R2 signed URLs because the upstream does not accept Base64 media.':
      'Brioi 必須使用 R2 簽名連結，因為上游不接受 Base64 素材。',
    'Brioi profiles match exact upstream model names. Disable supported capabilities as needed; unsupported values cannot be added.':
      'Brioi 設定檔會精確比對上游模型名稱。可視需要停用支援的能力，但無法加入不支援的值。',
    'Brioi Video': 'Brioi 影片',
    'Enable at least one generation mode': '請至少啟用一種生成模式',
    'Enable at least one supported option': '請至少啟用一個支援的選項',
    'Failed to load API key group settings': '載入 API 金鑰群組設定失敗',
    'Failed to load video configuration': '載入影片設定失敗',
    'First frame': '首幀',
    'Image limit must be between {{min}} and {{max}} for this profile':
      '此設定檔的圖片數量限制必須介於 {{min}} 與 {{max}} 之間',
    'Image reference': '圖片參考',
    'Last frame': '尾幀',
    'Maximum multi-image references': '多圖參考上限',
    'Maximum reference images': '參考圖片上限',
    'May be lowered for this provider profile, but cannot exceed the Brioi limit of {{max}}.':
      '可為此供應商設定檔調低，但不得超過 Brioi 的 {{max}} 張限制。',
    'No eligible video models are available for this key and provider.':
      '此金鑰與供應商沒有可用的影片模型。',
    'Option is outside the Brioi model hard capabilities':
      '選項超出 Brioi 模型的硬性能力範圍',
    'Preview only — image/audio base64 is shortened here. On submit, full data:…;base64,… payloads are sent.':
      '僅預覽：圖片/音訊的 base64 在此縮短顯示。真正送出時會傳送完整的 data:…;base64,… 內容。',
    'Provider groups': '供應商群組',
    Reference: '參考',
    'Reference audio was cleared because it is not supported by the selected mode.':
      '參考音訊不受所選模式支援，已清除。',
    'Reference images were cleared because they are not supported by the selected mode.':
      '參考圖片不受所選模式支援，已清除。',
    Resolution: '解析度',
    Resolutions: '解析度',
    'Selected generation mode was cleared because it is not supported by the selected model.':
      '所選生成模式不受目前模型支援，已清除選取。',
    'Selected model was cleared because it does not support this generation mode.':
      '所選模型不支援此生成模式，已清除選取。',
    'Status refresh failed temporarily; retrying automatically.':
      '狀態重新整理暫時失敗，正在自動重試。',
    'Unknown Brioi model profile': '未知的 Brioi 模型設定檔',
    '{{seconds}} seconds': '{{seconds}} 秒',
  },
  fr: {
    'Assign each group to only one video provider. Keys in these groups use this provider for models, capabilities, and task routing.':
      'Attribuez chaque groupe à un seul fournisseur vidéo. Les clés de ces groupes utilisent ce fournisseur pour les modèles, capacités et tâches.',
    'Based on fixed model price × group ratio. Duration does not change this estimate. Reference only — final charge follows actual billing.':
      'Estimation : prix fixe du modèle × coefficient du groupe. La durée ne la modifie pas. Le coût final suit la facturation réelle.',
    Brioi: 'Brioi',
    'Brioi requires R2 signed URLs because the upstream does not accept Base64 media.':
      'Brioi exige des URL R2 signées, car le fournisseur en amont n’accepte pas les médias en Base64.',
    'Brioi profiles match exact upstream model names. Disable supported capabilities as needed; unsupported values cannot be added.':
      'Les profils Brioi correspondent exactement aux modèles amont. Vous pouvez désactiver des capacités prises en charge, mais pas en ajouter.',
    'Brioi Video': 'Vidéo Brioi',
    'Enable at least one generation mode':
      'Activez au moins un mode de génération',
    'Enable at least one supported option':
      'Activez au moins une option prise en charge',
    'Failed to load API key group settings':
      'Échec du chargement des groupes de clés API',
    'Failed to load video configuration':
      'Échec du chargement de la configuration vidéo',
    'First frame': 'Première image',
    'Image limit must be between {{min}} and {{max}} for this profile':
      'La limite d’images de ce profil doit être comprise entre {{min}} et {{max}}',
    'Image reference': 'Image de référence',
    'Last frame': 'Dernière image',
    'Maximum multi-image references':
      'Nombre maximal d’images en mode multi-image',
    'Maximum reference images': 'Nombre maximal d’images de référence',
    'May be lowered for this provider profile, but cannot exceed the Brioi limit of {{max}}.':
      'Peut être réduit pour ce profil, sans dépasser la limite Brioi de {{max}}.',
    'No eligible video models are available for this key and provider.':
      'Aucun modèle vidéo admissible pour cette clé et ce fournisseur.',
    'Option is outside the Brioi model hard capabilities':
      'Cette option dépasse les capacités strictes du modèle Brioi',
    'Preview only — image/audio base64 is shortened here. On submit, full data:…;base64,… payloads are sent.':
      'Aperçu uniquement — le base64 image/audio est raccourci ici. À l’envoi, les payloads data:…;base64,… complets sont transmis.',
    'Provider groups': 'Groupes de fournisseur',
    Reference: 'Référence',
    'Reference audio was cleared because it is not supported by the selected mode.':
      'L’audio de référence a été supprimé car le mode sélectionné ne le prend pas en charge.',
    'Reference images were cleared because they are not supported by the selected mode.':
      'Les images de référence ont été supprimées car le mode sélectionné ne les prend pas en charge.',
    Resolution: 'Résolution',
    Resolutions: 'Résolutions',
    'Selected generation mode was cleared because it is not supported by the selected model.':
      'Le mode de génération a été désélectionné car le modèle choisi ne le prend pas en charge.',
    'Selected model was cleared because it does not support this generation mode.':
      'Le modèle sélectionné a été désélectionné car il ne prend pas en charge ce mode.',
    'Status refresh failed temporarily; retrying automatically.':
      'Échec temporaire de l’actualisation du statut. Nouvelle tentative automatique.',
    'Unknown Brioi model profile': 'Profil de modèle Brioi inconnu',
    '{{seconds}} seconds': '{{seconds}} secondes',
  },
  ja: {
    'Assign each group to only one video provider. Keys in these groups use this provider for models, capabilities, and task routing.':
      '各グループは 1 つの動画プロバイダーだけに割り当ててください。グループ内のキーは、そのプロバイダーのモデル、機能、タスクルーティングを使用します。',
    'Based on fixed model price × group ratio. Duration does not change this estimate. Reference only — final charge follows actual billing.':
      'モデル固定価格 × グループ倍率による見積もりです。時間によって変わりません。最終料金は実際の課金に従います。',
    Brioi: 'Brioi',
    'Brioi requires R2 signed URLs because the upstream does not accept Base64 media.':
      'Brioi はアップストリームが Base64 メディアを受け付けないため、R2 署名付き URL が必要です。',
    'Brioi profiles match exact upstream model names. Disable supported capabilities as needed; unsupported values cannot be added.':
      'Brioi プロファイルは上流モデル名と完全一致します。対応機能は無効化できますが、未対応の値は追加できません。',
    'Brioi Video': 'Brioi 動画',
    'Enable at least one generation mode':
      '生成モードを 1 つ以上有効にしてください',
    'Enable at least one supported option':
      '対応オプションを 1 つ以上有効にしてください',
    'Failed to load API key group settings':
      'API キーのグループ設定を読み込めませんでした',
    'Failed to load video configuration': '動画設定を読み込めませんでした',
    'First frame': '開始フレーム',
    'Image limit must be between {{min}} and {{max}} for this profile':
      'このプロファイルの画像上限は {{min}}～{{max}} にしてください',
    'Image reference': '参照画像',
    'Last frame': '終了フレーム',
    'Maximum multi-image references': '複数画像参照の上限',
    'Maximum reference images': '参照画像の上限',
    'May be lowered for this provider profile, but cannot exceed the Brioi limit of {{max}}.':
      'このプロバイダープロファイルでは引き下げられますが、Brioi の上限 {{max}} を超えることはできません。',
    'No eligible video models are available for this key and provider.':
      'このキーとプロバイダーで利用可能な動画モデルはありません。',
    'Option is outside the Brioi model hard capabilities':
      'オプションが Brioi モデルの上限を超えています',
    'Preview only — image/audio base64 is shortened here. On submit, full data:…;base64,… payloads are sent.':
      'プレビューのみ — 画像/音声の base64 はここでは短縮表示します。送信時は完全な data:…;base64,… を送ります。',
    'Provider groups': 'プロバイダーグループ',
    Reference: '参照',
    'Reference audio was cleared because it is not supported by the selected mode.':
      '選択したモードでは参照音声を使用できないため、削除しました。',
    'Reference images were cleared because they are not supported by the selected mode.':
      '選択したモードでは参照画像を使用できないため、削除しました。',
    Resolution: '解像度',
    Resolutions: '解像度',
    'Selected generation mode was cleared because it is not supported by the selected model.':
      '選択したモデルはこの生成モードに対応していないため、モードの選択を解除しました。',
    'Selected model was cleared because it does not support this generation mode.':
      '選択したモデルはこの生成モードに対応していないため、選択を解除しました。',
    'Status refresh failed temporarily; retrying automatically.':
      'ステータスの更新に一時的に失敗しました。自動的に再試行します。',
    'Unknown Brioi model profile': '不明な Brioi モデルプロファイル',
    '{{seconds}} seconds': '{{seconds}} 秒',
  },
  ru: {
    'Assign each group to only one video provider. Keys in these groups use this provider for models, capabilities, and task routing.':
      'Назначайте каждой группе только одного видеопровайдера. Ключи группы используют его модели, возможности и маршрутизацию задач.',
    'Based on fixed model price × group ratio. Duration does not change this estimate. Reference only — final charge follows actual billing.':
      'Расчёт: фиксированная цена модели × коэффициент группы. Длительность не влияет. Итоговая сумма определяется фактическим биллингом.',
    Brioi: 'Brioi',
    'Brioi requires R2 signed URLs because the upstream does not accept Base64 media.':
      'Brioi требует подписанные URL R2, поскольку вышестоящий сервис не принимает медиа в Base64.',
    'Brioi profiles match exact upstream model names. Disable supported capabilities as needed; unsupported values cannot be added.':
      'Профили Brioi точно соответствуют именам моделей провайдера. Поддерживаемые возможности можно отключать, но нельзя добавлять неподдерживаемые.',
    'Brioi Video': 'Видео Brioi',
    'Enable at least one generation mode':
      'Включите хотя бы один режим генерации',
    'Enable at least one supported option':
      'Включите хотя бы один поддерживаемый параметр',
    'Failed to load API key group settings':
      'Не удалось загрузить настройки групп API-ключей',
    'Failed to load video configuration':
      'Не удалось загрузить настройки видео',
    'First frame': 'Первый кадр',
    'Image limit must be between {{min}} and {{max}} for this profile':
      'Лимит изображений для профиля должен быть от {{min}} до {{max}}',
    'Image reference': 'Референсное изображение',
    'Last frame': 'Последний кадр',
    'Maximum multi-image references':
      'Максимум изображений для мульти-референса',
    'Maximum reference images': 'Максимум референсных изображений',
    'May be lowered for this provider profile, but cannot exceed the Brioi limit of {{max}}.':
      'Лимит можно уменьшить для этого профиля, но нельзя превысить предел Brioi: {{max}}.',
    'No eligible video models are available for this key and provider.':
      'Для этого ключа и провайдера нет подходящих видеомоделей.',
    'Option is outside the Brioi model hard capabilities':
      'Параметр выходит за жёсткие ограничения модели Brioi',
    'Preview only — image/audio base64 is shortened here. On submit, full data:…;base64,… payloads are sent.':
      'Только превью — base64 изображений/аудио здесь сокращён. При отправке уходят полные data:…;base64,….',
    'Provider groups': 'Группы провайдера',
    Reference: 'Референс',
    'Reference audio was cleared because it is not supported by the selected mode.':
      'Референсное аудио удалено: выбранный режим его не поддерживает.',
    'Reference images were cleared because they are not supported by the selected mode.':
      'Референсные изображения удалены: выбранный режим их не поддерживает.',
    Resolution: 'Разрешение',
    Resolutions: 'Разрешения',
    'Selected generation mode was cleared because it is not supported by the selected model.':
      'Режим генерации сброшен: выбранная модель его не поддерживает.',
    'Selected model was cleared because it does not support this generation mode.':
      'Выбранная модель не поддерживает этот режим генерации, поэтому выбор сброшен.',
    'Status refresh failed temporarily; retrying automatically.':
      'Временная ошибка обновления статуса. Выполняется автоматическая повторная попытка.',
    'Unknown Brioi model profile': 'Неизвестный профиль модели Brioi',
    '{{seconds}} seconds': '{{seconds}} секунд',
  },
  vi: {
    'Assign each group to only one video provider. Keys in these groups use this provider for models, capabilities, and task routing.':
      'Chỉ gán mỗi nhóm cho một nhà cung cấp video. Khóa trong nhóm sẽ dùng mô hình, khả năng và định tuyến tác vụ của nhà cung cấp đó.',
    'Based on fixed model price × group ratio. Duration does not change this estimate. Reference only — final charge follows actual billing.':
      'Ước tính theo giá cố định của mô hình × hệ số nhóm. Thời lượng không làm thay đổi giá. Chi phí cuối cùng theo cách tính phí thực tế.',
    Brioi: 'Brioi',
    'Brioi requires R2 signed URLs because the upstream does not accept Base64 media.':
      'Brioi yêu cầu URL R2 có chữ ký vì nhà cung cấp thượng nguồn không chấp nhận nội dung Base64.',
    'Brioi profiles match exact upstream model names. Disable supported capabilities as needed; unsupported values cannot be added.':
      'Hồ sơ Brioi khớp chính xác tên mô hình nguồn. Có thể tắt khả năng được hỗ trợ nhưng không thể thêm giá trị không hỗ trợ.',
    'Brioi Video': 'Video Brioi',
    'Enable at least one generation mode': 'Bật ít nhất một chế độ tạo',
    'Enable at least one supported option':
      'Bật ít nhất một tùy chọn được hỗ trợ',
    'Failed to load API key group settings':
      'Không thể tải cài đặt nhóm khóa API',
    'Failed to load video configuration': 'Không thể tải cấu hình video',
    'First frame': 'Khung hình đầu',
    'Image limit must be between {{min}} and {{max}} for this profile':
      'Giới hạn ảnh của hồ sơ này phải từ {{min}} đến {{max}}',
    'Image reference': 'Ảnh tham chiếu',
    'Last frame': 'Khung hình cuối',
    'Maximum multi-image references': 'Số ảnh tham chiếu đa ảnh tối đa',
    'Maximum reference images': 'Số ảnh tham chiếu tối đa',
    'May be lowered for this provider profile, but cannot exceed the Brioi limit of {{max}}.':
      'Có thể giảm cho hồ sơ nhà cung cấp này nhưng không được vượt quá giới hạn {{max}} của Brioi.',
    'No eligible video models are available for this key and provider.':
      'Không có mô hình video phù hợp cho khóa và nhà cung cấp này.',
    'Option is outside the Brioi model hard capabilities':
      'Tùy chọn vượt quá giới hạn cứng của mô hình Brioi',
    'Preview only — image/audio base64 is shortened here. On submit, full data:…;base64,… payloads are sent.':
      'Chỉ xem trước — base64 ảnh/âm thanh được rút gọn tại đây. Khi gửi sẽ dùng đầy đủ data:…;base64,….',
    'Provider groups': 'Nhóm nhà cung cấp',
    Reference: 'Tham chiếu',
    'Reference audio was cleared because it is not supported by the selected mode.':
      'Đã xóa âm thanh tham chiếu vì chế độ đã chọn không hỗ trợ.',
    'Reference images were cleared because they are not supported by the selected mode.':
      'Đã xóa ảnh tham chiếu vì chế độ đã chọn không hỗ trợ.',
    Resolution: 'Độ phân giải',
    Resolutions: 'Độ phân giải',
    'Selected generation mode was cleared because it is not supported by the selected model.':
      'Đã bỏ chọn chế độ tạo vì mô hình đã chọn không hỗ trợ.',
    'Selected model was cleared because it does not support this generation mode.':
      'Đã bỏ chọn mô hình vì mô hình không hỗ trợ chế độ tạo này.',
    'Status refresh failed temporarily; retrying automatically.':
      'Tạm thời không thể làm mới trạng thái; đang tự động thử lại.',
    'Unknown Brioi model profile': 'Hồ sơ mô hình Brioi không xác định',
    '{{seconds}} seconds': '{{seconds}} giây',
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
