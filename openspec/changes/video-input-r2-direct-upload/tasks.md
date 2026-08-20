## 1. Backend foundation

- [x] 1.1 Add `video_input_assets` persistence (model + migration-safe GORM) with user ownership, object key, kind, size, content type, status, expires_at
- [x] 1.2 Implement R2 PresignPut helper on the existing S3-compatible object store (bounded content-type / size where supported)
- [x] 1.3 Implement `CreateInputAssetPresign(userID, kind, contentType, size)` with upload_limits + MIME whitelist + R2 soft-limit checks
- [x] 1.4 Implement `CompleteInputAsset(userID, assetID)` with HEAD/size/type validation, delete-on-failure, and PresignGet response (`url`, `expires_at`)
- [x] 1.5 Implement optional `DeleteInputAsset(userID, assetID)` for abandon/cancel
- [x] 1.6 Add `user_id`-keyed rate limit (default 20 presign/min) and pending cap (default 5 incomplete); prefer Redis when configured

## 2. HTTP API

- [x] 2.1 Wire dashboard routes: `POST /api/video/input-assets/presign`, `POST /api/video/input-assets/:id/complete`, optional `DELETE /api/video/input-assets/:id` under `UserAuth`
- [x] 2.2 (Same change if low cost) Mirror routes under `/v1/video/input-assets/...` with token auth, still limited by owning `user_id`
- [x] 2.3 Return clear 4xx/429 error codes/messages for validation, rate limit, soft-block, and ownership failures
- [x] 2.4 Add Go tests for presign validation, complete success/failure, ownership, and rate-limit behavior

## 3. Generation compatibility

- [x] 3.1 Confirm HTTPS `media[].source` continues to pass through `StageVideoInputMedia` without re-upload
- [x] 3.2 Keep `data:` path working for non-tool / legacy clients; add regression tests if gaps appear
- [ ] 3.3 Optionally map expired-upstream / fetch failures to a user-visible “media expired, re-upload” message when detectable

## 4. Video tool frontend

- [x] 4.1 Add upload client helpers (presign → PUT with progress → complete) and per-file state (`pending|uploading|ready|failed`)
- [x] 4.2 On file select: local validate then start upload; show progress UI
- [x] 4.3 Before submit: reject expired assets (`expires_at`) with toast to re-select/upload; disable submit while uploads in flight
- [x] 4.4 Build generation body with HTTPS URLs only for uploaded assets (no bulk `data:` on the tool path)
- [x] 4.5 Add i18n strings for upload progress, failures, rate limit, and expiry prompts
- [x] 4.6 Add frontend unit tests for request construction and expiry gating

## 5. Ops and verification

- [x] 5.1 Document required R2 bucket CORS (PUT/OPTIONS from dashboard origins) in the change notes or admin docs
- [ ] 5.2 Manual verify: large image/video upload shows progress; generation request payload stays small; expired URL prompts re-upload
- [ ] 5.3 Manual verify: rapid presign returns 429; soft-blocked R2 rejects new uploads
