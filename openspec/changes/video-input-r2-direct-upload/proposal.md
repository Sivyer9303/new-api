## Why

Video Generation currently packs reference images, audio, and video into `/v1/video/generations` as base64 `data:` URLs. The gateway then stages those bytes to R2 on the submit path, so large payloads make generation slow and burn gateway bandwidth and memory. Users need to upload media first, then submit a small generation request that only carries HTTPS URLs.

## What Changes

- Add a browser → R2 **presigned PUT** upload flow (scheme B1): `presign` → client PUT → `complete` → short-lived R2 **presigned GET** URL.
- Video tool UI: upload on file select with progress; generation body uses those HTTPS URLs only (no bulk base64 on the tool path).
- Keep `/v1/video/generations` **compatible**: continue to accept `data:` URLs and arbitrary `https://` sources; prefer URLs when present.
- Rate-limit and quota upload APIs by **`user_id`** (presign rate, pending count, size/type checks, R2 soft-limit guard).
- On expired upload GET URLs: do **not** auto-refresh; tell the user to re-select and upload again (TTL ≈ existing `input_presign_ttl_seconds`, default 6h).
- **Out of scope this change:** user-typed external URL UI; custom CDN domain for hiding `*.r2.cloudflarestorage.com`; daily byte quotas / asset library management.

## Capabilities

### New Capabilities

- `video-input-direct-upload`: Presign PUT, client upload, complete, short-lived GET URL, `user_id` anti-abuse limits, and video-tool + generations consumption of HTTPS media sources.

### Modified Capabilities

None. There is no synchronized main-spec capability for input staging yet; this change builds on the existing R2 input staging helpers and video tool APIs.

## Impact

- Backend: new dashboard (and optionally `/v1`) input-asset routes; R2 PresignPut; complete validation; Redis/memory rate limits keyed by `user_id`; reuse `StageVideoInputMedia` https pass-through on generation.
- Frontend: video tool upload state machine, progress UI, expiry checks before submit, i18n.
- Ops: R2 CORS must allow browser PUT from the dashboard origin(s); no new third-party dependencies beyond existing AWS S3-compatible signing used for R2.
