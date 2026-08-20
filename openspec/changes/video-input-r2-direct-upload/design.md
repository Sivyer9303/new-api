## Context

Today the Video Generation page converts selected files to base64 `data:` URLs and posts them inside `/v1/video/generations`. Adaptors that require R2 delivery call `service.StageVideoInputMedia`, which uploads inline data URLs to the R2 input prefix and returns a short-lived presigned GET URL for upstream. `StageVideoInputMedia` already **passes through** `http(s)` sources unchanged.

R2 is already configured (`input_prefix`, `input_presign_ttl_seconds` default 21600 / 6h, `input_ttl_hours` default 24h, upload size limits, free-tier soft block). Result delivery already uses short-lived GET redirects; this change adds **client-side PUT** for inputs.

Stakeholders: Video tool users (latency), gateway operators (bandwidth/memory), channel adaptors (unchanged contract if they already accept HTTPS media).

## Goals / Non-Goals

**Goals:**

- Move large media bytes off the generation request path via browser → R2 direct PUT.
- Return a short-lived R2 **presigned GET** URL from complete (scheme B1) for use as `media[].source`.
- Show per-file upload progress; block submit until media is ready or clear expired media.
- Enforce anti-abuse limits by **`user_id`** (not per API token).
- Keep generations compatible with `data:` and third-party `https://` sources.

**Non-Goals:**

- User-typed external URL UI in the video tool.
- Stable gateway URLs for inputs (`/v1/video/input-assets/.../content`) or auto-renewal of expired GET URLs.
- Custom domain to hide `*.r2.cloudflarestorage.com`.
- Daily per-user byte quotas or a reusable media library UI (may follow later).
- Changing upstream channel protocols.

## Decisions

### 1. Scheme B1: short-lived R2 GET after complete

**Choice:** Complete returns `{ url, expires_at, ... }` where `url` is an R2 (or S3-compatible) **presigned GET** with TTL = configured `input_presign_ttl_seconds`.

**Rationale:** Generation can pass the URL straight through existing staging pass-through; no asset content proxy in P1; bandwidth stays on user↔R2 and upstream↔R2.

**Alternative (rejected for P1):** Stable gateway URL + resolve-on-submit (scheme A). Better for multi-hour delays, more code (asset resolve path, content route). Product accepted “re-upload if expired”.

**Alternative (rejected):** Public-read bucket URLs. Avoids expiry but weakens access control.

### 2. No secrets in the browser; Presigned PUT only

**Choice:** Server signs PUT with server-side R2 credentials. Client never receives `access_key`, `secret`, or `api_token`.

Visible to clients: temporary signed URLs, object host, object key (UUID under `input_prefix`). Treat signed URLs as bearer credentials for that object until expiry.

### 3. API shape (dashboard first; `/v1` optional same handlers)

```text
POST /api/video/input-assets/presign
  Auth: UserAuth (dashboard)
  Body: { kind: "image"|"audio"|"video", content_type, size }
  → { asset_id, upload_url, upload_headers, expires_at }

PUT {upload_url}   # browser → R2, progress tracked client-side

POST /api/video/input-assets/{asset_id}/complete
  Auth: UserAuth
  → { asset_id, url, expires_at, kind, content_type, size }

DELETE /api/video/input-assets/{asset_id}   # optional P1: cancel/abandon
```

Optional Phase 1.1: mirror under `/v1/video/input-assets/...` with token auth, still rate-limited by owning `user_id`.

Generation request unchanged:

```json
"media": [{ "type": "image", "role": "reference", "source": "https://..." }]
```

### 4. Asset bookkeeping

**Choice:** Persist a lightweight `video_input_assets` row (or equivalent) with: `asset_id`, `user_id`, `object_key`, `kind`, `content_type`, `size`, `status` (`presigned`|`ready`|`failed`|`expired`), `expires_at`, timestamps.

**Rationale:** Pending-count limits and complete ownership checks need server state; object key must not be client-chosen.

Orphan objects rely on existing input lifecycle / TTL cleanup where available; failed complete deletes the object when possible.

### 5. Anti-abuse keyed by `user_id`

| Control | Initial default | Notes |
|---|---|---|
| Presign rate | 20 / minute / user | Redis if available, else in-process |
| Max incomplete (`presigned`) | 5 / user | |
| Max unused `ready` assets | 30 / user | Soft cap before generation |
| Size / MIME | Existing `upload_limits` + kind whitelist | Declared size bound on presign |
| R2 soft limit | Existing `VideoStorageUploadBlocked` | Reject new presigns |

Same user sharing multiple tokens shares one bucket of limits.

### 6. Frontend flow

```text
select file → local validate → presign → PUT (progress) → complete → store {url, expires_at}
submit → if any asset past expires_at → toast + require re-upload
       → else generations with https sources only
```

Keep local blob previews; do not depend on GET URL for preview in P1 if CORS GET is awkward—optional later.

### 7. Expiry UX

No server-side refresh of GET URLs. If submit detects expiry or generation fails with a clear expired-media signal, prompt: refresh / re-select and upload again.

### 8. R2 CORS

Administrators MUST configure the R2 bucket CORS to allow PUT (and OPTIONS) from the dashboard origin(s). Document required CORS in ops notes; without it browser upload fails.

## Risks / Trade-offs

- **[Risk] GET URL expires before submit** → Mitigation: show `expires_at`; pre-submit check; clear error copy. No auto-renew in P1.
- **[Risk] Presigned URL leak** → Mitigation: short TTL, random keys, no logging of full signed URLs in access logs where avoidable.
- **[Risk] CORS misconfiguration** → Mitigation: document setup; surface upload failure clearly.
- **[Risk] Client lies about size then uploads larger body** → Mitigation: prefer conditional/presign constraints; complete HEAD + size check; delete on mismatch.
- **[Trade-off] Hostname reveals R2** → Accepted for P1; custom domain later.
- **[Trade-off] Multi-instance rate limit without Redis** → Per-node limits only; document Redis recommendation for production.

## Migration Plan

1. Deploy backend APIs behind UserAuth; no behavior change until UI ships.
2. Configure R2 CORS for dashboard origins.
3. Ship video tool UI using upload-then-URL path.
4. Leave `data:` path intact for API clients and rollback (feature flag or UI revert).

Rollback: disable UI upload path / revert frontend; generations still accept `data:`.

## Open Questions

- Whether `/v1` upload routes ship in the same PR as dashboard routes (lean: same handlers, `/v1` in same change if low cost).
- Whether DELETE abandon is required in P1 or TTL-only cleanup is enough (lean: DELETE nice-to-have).
