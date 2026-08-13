## Context

The generic video foundation already provides a persisted asynchronous task lifecycle, polling, billing settlement, mandatory result storage, R2 input staging, URL redaction, content delivery, and administrator recovery. SilkRoad currently implements `channel.TaskAdaptor`, while `relay/channel/task/videocommon` contains provider-neutral request/result concepts and parsing helpers. Some storage and URL-redaction entry points are still SilkRoad-named and identify video tasks with hardcoded legacy platform values `"60"` and `"61"`; those checks cannot safely recognize Brioi without being generalized.

Brioi exposes `POST /v1/videos` and `GET /v1/videos/{task_id}` at an administrator-supplied base URL. Its Seedance endpoints reject base64 media and require publicly readable HTTPS links. Brioi and SilkRoad can expose identical model names but different capabilities, so model name alone cannot select a provider configuration.

The Video Generation page currently selects an API key, calls `/v1/models` with that key, and applies one SilkRoad-derived capability set. The backend correctly scopes the general model list and task distribution by token group, but it does not bind a Video Generation group to one provider type. The frontend price estimate also always multiplies model price by duration, even for fixed-price models.

## Goals / Non-Goals

**Goals:**

- Add Brioi as a standalone video-only channel without protocol or settings branches inside NewAPI or SilkRoad.
- Reuse `channel.TaskAdaptor`, `videocommon`, and the existing task/storage lifecycle.
- Support Brioi `seedance-2-0-fast`, `seedance-2-0`, and `seedance-2-5` from the Video Generation page.
- Support text-to-video, ordinary single/multi-image reference, strict first-frame, and strict first/last-frame generation.
- Upload every user-selected image to R2 before creating the Brioi task and ensure the upstream request contains no base64 media.
- Give each provider type an independent Video Configuration page.
- Bind each Video Generation group to exactly one video provider type and enforce that binding in capability selection, model discovery, and distribution.
- Keep fixed/per-second pricing in the existing global model pricing configuration and make the frontend estimate respect it.

**Non-Goals:**

- Do not add Brioi support to the public client `/v1/videos` route in the first phase.
- Do not add Brioi image, text, audio, or other non-video interfaces.
- Do not support Brioi `/v1/realperson` assets.
- Do not support video or audio reference media.
- Do not add a default Brioi base URL.
- Do not share Brioi settings or request branches with SilkRoad.
- Do not add a runtime JSON request-mapping language.

## Decisions

### 1. Implement Brioi as a sibling `TaskAdaptor`

Add a dedicated `ChannelTypeBrioi` and a `relay/channel/task/brioi` package that implements the existing `channel.TaskAdaptor` contract. Brioi and SilkRoad packages MUST NOT import one another.

```text
channel.TaskAdaptor
        ├── silkroad.TaskAdaptor
        └── brioi.TaskAdaptor

videocommon
        ├── normalized video request/media types
        ├── task ID, status, result URL, and error parsing
        └── reusable validation helpers
```

The common task orchestration remains responsible for persistence, polling schedules, billing, provider-success versus delivery-success states, result storage, and public responses. Brioi owns only its URLs, headers, request fields, model limits, and response details. Provider-neutral storage and URL-redaction paths classify new tasks by `TaskPrivateData.VideoTask`; preexisting action/platform fallbacks, including platforms `"60"` and `"61"`, remain for historical tasks created before that marker existed.

This uses the existing unified interface instead of introducing another broad `VideoProvider` interface with overlapping responsibilities.

**Alternative considered:** add Brioi branches to SilkRoad. Rejected because provider protocols and settings would become mutually dependent.

**Alternative considered:** create a configurable field-mapping DSL. Rejected because media limits, billing bounds, and response semantics must be compile-time, testable behavior.

### 2. Register a video-only channel with no default base URL

`ChannelTypeBrioi` declares only `EndpointTypeOpenAIVideo` for distributor compatibility, but the first phase accepts submissions only from `/v1/video/generations`. It has no synchronous relay adaptor.

The administrator MUST provide a non-empty channel base URL. The adaptor trims a trailing slash and appends `/v1/videos` or `/v1/videos/{task_id}`. Channel testing calls `<base>/v1/models` with the configured Bearer key and reports authentication/model availability without generating a billable task.

Normal channel model lists, model mappings, groups, priorities, weights, proxies, and keys remain channel-instance concerns.

### 3. Keep provider capabilities in independent settings modules

Add a `brioi_setting` module and a Brioi child page under Video Configuration. The menu is:

```text
Video Configuration
├── General
├── Storage
├── SilkRoad
└── Brioi
```

SilkRoad and Brioi each own their provider groups and model profiles. Storage and the global Video Generation enable switch remain generic.

Brioi defaults include exact upstream profiles:

- `seedance-2-0-fast`: integer durations 4–15, 480p/720p, and 21:9, 16:9, 4:3, 1:1, 3:4, 9:16.
- `seedance-2-0`: the same durations and ratios, with 480p/720p/1080p/4K.
- `seedance-2-5`: integer durations 4–29, 480p/720p, and 16:9/9:16.

Administrators can disable supported options but cannot configure values outside documented hard bounds. Capability matching uses the model-mapped upstream name. Unknown models are rejected rather than assigned a default profile.

Provider settings are returned to users only as sanitized public capabilities; channel keys, storage credentials, and administrative metadata are excluded.

### 4. Make Video Generation groups uniquely provider-owned

Each group exposed in the Video Generation tool belongs to exactly one provider type. Saving a provider configuration fails with a targeted conflict if one of its groups is already assigned to another provider.

The provider resolver is used consistently:

```text
selected API key
  → token group
  → unique provider type
  → provider-specific public capabilities
  → provider-constrained video models
  → provider-constrained task distribution
```

The Video Generation page must not use the unrestricted group model union as its authoritative model source. A video-tool model query returns models that satisfy group, provider channel type, video endpoint eligibility, token model limits, and billing configuration. Submission repeats the provider-type constraint server-side, so client tampering cannot cross-route an identically named model.

Provider group lists are aggregated only for backward-compatible generic visibility. Provider-specific behavior never derives from that union.

**Alternative considered:** allow multiple provider types per group and use channel priority. Rejected because failover could select a provider whose request capabilities differ from those shown in the UI.

**Alternative considered:** add a provider selector. Rejected because the operator explicitly uses groups to separate upstreams.

### 5. Normalize UI requests, then build Brioi-native `ref` payloads

The Video Generation page continues posting its provider-neutral friendly request to `/v1/video/generations`. The Brioi adaptor parses it into `videocommon.VideoGenerateRequest`, validates the matched upstream profile, and maps image media as follows:

- no images: omit `ref`;
- ordinary image reference: `{"url": signedURL, "type": "image"}`;
- strict first frame: add `role: "first_frame"`;
- strict last frame: add `role: "last_frame"` and require a first frame;
- strict frame modes cannot mix with ordinary reference images.

The body sent to Brioi contains `model`, `prompt`, integer `duration`, `resolution`, `aspect_ratio`, and optional `ref`. It never contains the client-only `generation_type`, base64 data, SilkRoad fields, or unsupported chat fields.

Seedance 2.0 ordinary multi-reference requests require at least one image and allow at most nine. Seedance 2.5 image reference requests allow at most thirty. The shared frontend mode can expose lower administrative limits but never higher provider limits.

### 6. Stage every uploaded image before upstream submission

The frontend converts selected files to data URLs as it does today. Before building the Brioi body, the adaptor calls the provider-neutral R2 input staging service for each image, preserving order and role. Staging writes under `video-inputs/<channel-id>/`, returns a presigned HTTPS URL, and uses the configured input URL lifetime and cleanup period.

If any upload, content-type validation, quota check, or URL signing step fails, the request fails before Brioi receives a create-task request. The already staged objects remain bounded by the normal input cleanup job.

Signed input URLs are not written to public task data or normal logs. Brioi receives them because it must fetch the media; users receive neither those URLs nor upstream result URLs.

### 7. Normalize Brioi submit and polling responses

Submit parsing accepts non-empty `id` or `task_id` and persists only the upstream ID privately. Polling maps:

- `queued` and `pending` to queued;
- `processing` and `in_progress` to running;
- `completed` to provider success only when `metadata.url` or `result_url` is non-empty;
- `failed` and `cancelled` to failure;
- unknown statuses to failure with `NoRefund=true` for administrator review.

Temporary polling transport errors and 5xx responses retry the same upstream ID; they never create a duplicate task. Provider success enters the existing storage state, downloads the result through the SSRF-protected result source, stores it through the active R2 driver, and becomes public success only after storage verification.

A completed response without a result URL is treated as a provider/delivery anomaly and must not be exposed as successful. Existing no-automatic-refund and administrator recovery policy applies where provider charging may already have occurred.

### 8. Reuse central billing and correct the UI estimate

The Brioi adaptor validates duration against the selected model's hard bounds and reports a bounded `seconds` multiplier through the existing billing hook. The global model billing mode decides whether model price is fixed per request or multiplied by seconds; Brioi settings do not duplicate pricing mode.

The Video Generation estimate changes to:

- fixed price: `model price × group ratio`;
- per-second: `model price × duration × group ratio`.

Backend pre-consume and settlement remain authoritative and continue using checked quota conversion.

### 9. Remove provider IDs from generic storage and redaction gates

Rename or wrap SilkRoad-named success-store, ingest, and URL-redaction entry points behind provider-neutral video functions. New video tasks are identified by `TaskPrivateData.VideoTask` and their persisted storage metadata rather than an allow-list of provider platform IDs. Historical rows that predate `VideoTask` retain the baseline action/platform compatibility heuristics, including platforms `"60"` and `"61"`.

The polling path sends every provider-normalized video success through this common transition. Adding Brioi therefore requires adaptor registration, not another provider-specific branch in storage, cleanup, content delivery, diagnostics, or administrative recovery.

**Alternative considered:** add Brioi platform `"62"` to every SilkRoad allow-list. Rejected because each future provider would require editing generic lifecycle code and a missed list would expose an upstream URL or skip mandatory storage.

### 10. Make asynchronous acceptance and delivery transitions durable

The initial implementation inherited a task flow that wrote the client success response before billing settlement and task insertion. That ordering can acknowledge a task that the system failed to persist after the upstream already accepted it. Video task submission MUST instead create a durable `SUBMITTING` row before the create-task request, then persist the upstream identifier and public response data before settlement and before writing client success.

Task adaptors parse provider responses but do not write HTTP responses themselves. The common controller owns the final response after persistence and settlement. Once an upstream has accepted a task, later local failures MUST NOT trigger an automatic refund; the durable row remains recoverable or enters administrator review.

Provider success, billing settlement, and result storage use explicit recoverable states:

```text
SUBMITTING
  → provider accepted / upstream ID persisted
  → provider success with usable result URL
  → SETTLING
  → STORING
  → SUCCESS
```

`SETTLING` is a resumable state, not a one-way transient marker. Its watchdog retries an idempotent settlement transition and then exposes the task to storage. A provider success without a usable result URL never enters settlement or storage; it becomes `provider_review` with automatic refund withheld.

Polling channel lookup distinguishes transient cache/database failures from definitive channel removal. Transient failures leave tasks pollable; a definitively unavailable channel moves the task to administrator review without pretending that the provider failed.

Task HTTP handling accepts every successful 2xx submission response, closes every response body, bounds and sanitizes provider error bodies, and retries only retryable statuses. This prevents duplicate create-task requests after valid 201/202 responses and avoids leaking provider bodies or connections.

### 11. Keep status, settings writes, and frontend states consistent

The generic Video Generation enabled flag is computed once from global enablement plus any enabled provider and reused by `/api/status` and `/api/video/tool-config`. The legacy SilkRoad status field remains an alias for compatibility. Provider/global video setting changes invalidate the frontend status query and its local-storage snapshot immediately.

Each provider settings page saves its complete provider configuration through one backend operation. Validation, cross-provider ownership checks, and persistence are atomic so a failed save cannot leave groups and profiles from different revisions. The frontend emits one success/error notification per save.

The Video Generation page distinguishes administrator-disabled state from configuration, API-key, and group-loading failures and offers a retry path. Dependent selections change atomically; an incompatible mode clears or explicitly announces a model change instead of silently replacing the user's choice. Form controls use associated labels and selection-group semantics. Multi-image previews use bounded thumbnails and render only populated slots plus available add controls.

## Risks / Trade-offs

- **[Group configuration conflicts]** A group assigned to two providers would make identical model names ambiguous. → Reject overlapping provider group saves with a precise conflict message and repeat provider resolution during submission.
- **[Configured group has no matching Brioi channel]** The UI could expose a provider with no route. → Provider-constrained model discovery returns no models and explains the missing eligible channel.
- **[R2 input URL expires before Brioi fetches it]** Long upstream queueing could make the image unavailable. → Use the configurable input presign TTL, defaulting to six hours, and document that it must exceed the provider fetch window.
- **[Large multi-image payloads consume R2 quota]** Seedance 2.5 allows many images. → Reuse per-object limits, R2 usage gating, provider item limits, and automatic input cleanup.
- **[Brioi changes model capabilities]** Hard bounds could become stale. → Keep provider settings able to disable documented options, validate against hard safety limits, and update defaults/tests when the contract changes.
- **[First phase omits `/v1/videos`]** API clients cannot use the new channel yet. → Keep parsing and provider logic based on normalized video types so a later route parser can reuse the adaptor without changing the provider protocol.
- **[Existing deployments use global video groups]** Moving to provider ownership could hide keys until configured. → Preserve SilkRoad's current groups as its compatibility source and require explicit Brioi group assignment; expose clear admin validation and migration guidance.
- **[Historical tasks lack the video marker]** Narrowing the baseline action/platform heuristics could expose old upstream URLs or skip storage. → Preserve the existing compatibility heuristics for unmarked historical tasks, explicitly cover platforms `"60"` and `"61"`, and use `VideoTask` for every new provider.

## Migration Plan

1. Add the new channel type, settings defaults, provider resolver, and sanitized configuration output without assigning existing groups to Brioi.
2. Preserve existing SilkRoad group behavior by treating current configured video groups as SilkRoad-owned until explicitly changed.
3. Deploy backend and frontend together so provider-specific configuration and model resolution agree.
4. Administrators add a Brioi channel with an explicit base URL/key/models and assign a group on the Brioi settings page.
5. Validate the channel with `/v1/models`, then test one text-to-video and one image-to-video task through the Video Generation page.
6. Rollback disables/removes Brioi group ownership and channel rows; existing tasks remain queryable through their persisted Brioi platform adaptor while the version containing that adaptor is running.

## Open Questions

None. The first-phase scope and provider/group boundaries were confirmed during design review.
