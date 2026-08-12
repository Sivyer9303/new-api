## Why

Video generation is currently implemented as SilkRoad-specific behavior attached to the generic NewAPI channel type, with provider assumptions duplicated across task adaptation, settings, storage, and the frontend. A provider-neutral video foundation is needed now so SilkRoad can become a dedicated video-only channel and future video providers can be added without modifying the task core or rebuilding the UI and result-delivery flow.

## What Changes

- Introduce a provider-neutral asynchronous video task foundation covering normalized requests, submission, polling, status normalization, billing hooks, result delivery, and provider registration.
- Add a dedicated video-only SilkRoad channel type and move new SilkRoad video submissions to its own adapter and channel-type configuration.
- Keep both `/v1/video/generations` and `/v1/videos` client APIs, normalize both into the same internal task flow, and preserve route-specific response compatibility.
- Retain legacy NewAPI/SilkRoad task query compatibility without migrating existing channel or task records; operators will create new SilkRoad channels manually.
- Generalize SilkRoad output ingest into mandatory storage for every video channel. A task remains externally `processing` until the stored file is ready, and upstream result URLs are never exposed to users.
- Fix video retention at seven days from successful storage. Expiry deletes the local file and removes playable local and upstream result addresses for all users, including administrators.
- Treat exhausted storage retries as a non-refundable delivery failure. Users are instructed to contact an administrator; administrators can retry storage, confirm the upstream result, or issue one idempotent full refund.
- Allow administrators to preview any user's unexpired stored video while preserving owner-only access for ordinary users and recording administrative access and recovery actions in audit logs.
- Add a top-level Video Configuration area with generic settings and a SilkRoad child section. Provider adapters define hard capability limits; common provider defaults and optional model profiles may narrow them, with an administrator-selected default profile used when no exact or prefix match exists.
- Rename the user-facing Seedance extension to Video Generation, move it to `/extensions/video`, and redirect the old `/extensions/seedance` route.
- Add a storage-driver boundary with a local-disk implementation using the existing designated ingest-node/public-base model, leaving room for future cloud object-storage drivers.
- Do not add Brioi, generic input-asset storage, shared local filesystems, or automatic SilkRoad channel migration in this change.

## Capabilities

### New Capabilities

- `video-provider-framework`: Provider-neutral asynchronous video APIs, task lifecycle, provider registration, dedicated SilkRoad channel behavior, legacy compatibility, and model capability/profile resolution.
- `video-result-storage`: Mandatory provider-neutral result ingest, private upstream URL handling, local storage driver behavior, seven-day retention, delivery-state semantics, and storage failure policy.
- `video-admin-operations`: Cross-user administrator preview, storage recovery, upstream confirmation, idempotent full refunds, authorization, and audit requirements.
- `video-configuration`: Top-level video settings navigation, generic/provider-specific configuration ownership, default profile selection, and the renamed generic Video Generation extension.

### Modified Capabilities

None. This repository has no existing OpenSpec capability specifications.

## Impact

- Backend channel constants, endpoint capability registration, task adapter selection, request/response conversion, polling, billing, task persistence, video proxying, result sanitization, storage cleanup, settings, and administrator APIs.
- Frontend channel metadata, system-settings navigation and forms, extension route and branding, capability-driven video form rendering, task-log preview and recovery controls, and i18n locales.
- Existing NewAPI channels and stored tasks remain in place. Legacy task handling must remain available while all new SilkRoad submissions use the dedicated channel type.
- No new external storage dependency is introduced; the initial driver uses local disk on a configured ingest node.
