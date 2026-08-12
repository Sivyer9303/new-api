## Context

The repository already has a reusable asynchronous task pipeline:

```text
router/video-router.go
  → controller.RelayTask
  → relay.RelayTaskSubmit
  → channel.TaskAdaptor
  → model.Task
  → service.RunTaskPollingOnce / UpdateVideoTasks
  → provider polling, billing settlement, and content delivery
```

SilkRoad video support was added by registering a SilkRoad-specific task adapter under `ChannelTypeNewAPI = 60`. That adapter depends on `silkroad_setting`, accepts a custom friendly request, calls `/v1/video/generations`, applies per-second billing, and triggers optional local result ingest. The generic NewAPI channel simultaneously remains a multi-format synchronous upstream for OpenAI, Claude, Gemini, Responses, and alpha-search traffic.

This creates four kinds of coupling:

1. The NewAPI channel type has both generic gateway semantics and SilkRoad video semantics.
2. Provider-specific profile matching and generation modes are duplicated between backend and frontend.
3. Result storage and URL redaction are named for SilkRoad and are conditionally applied only to platform `"60"`.
4. A provider result is currently considered successful before mandatory local delivery is guaranteed.

The change must preserve the existing layered architecture, all supported databases, quota safety invariants, and the current local ingest-node deployment model. `relaykit/` independence must remain unaffected. The initial implementation supports only SilkRoad; Brioi and generic input-asset staging are explicitly deferred.

## Goals / Non-Goals

**Goals:**

- Make the asynchronous video lifecycle provider-neutral and stable enough that a future provider needs only a channel type, adapter, capability definition, provider configuration, and registration.
- Give SilkRoad a dedicated video-only channel type without changing generic NewAPI synchronous behavior.
- Keep both public video API families and one persisted task lifecycle.
- Make stored delivery mandatory for every video provider and prevent every user-facing upstream URL leak.
- Expose completion only after the stored result is verified.
- Preserve the seven-day retention rule without administrator bypass.
- Provide administrator recovery, full-refund, cross-user preview, and audit workflows.
- Separate generic Video Configuration from provider-specific configuration.
- Preserve legacy NewAPI/SilkRoad tasks and settings without automatic channel or task migration.

**Non-Goals:**

- Add Brioi or any other new video provider.
- Build generic input media hosting or convert client data URLs into public provider-readable URLs.
- Build a shared filesystem.
- Add an S3/OSS/COS/R2 driver in this change.
- Migrate existing channel rows or rewrite historical task platform values.
- Make arbitrary upstream request transformation configurable as a JSON mapping language.
- Allow configurable retention beyond or below the fixed seven-day rule.
- Add chat, image, audio, embedding, or text endpoints to the SilkRoad channel.

## Decisions

### 1. Keep the existing task orchestration and extract provider-neutral video concepts

`channel.TaskAdaptor`, `RelayTaskSubmit`, task persistence, polling, and billing hooks remain the orchestration backbone. The change will add a focused provider-neutral video package alongside task adapters rather than replacing the task framework.

The neutral package will own durable video concepts:

```go
type VideoGenerateRequest struct {
    Model          string
    Prompt         string
    GenerationType string
    Duration       *int
    Resolution     string
    AspectRatio    string
    Media          []VideoMedia
}

type VideoMedia struct {
    Type   VideoMediaType
    Role   VideoMediaRole
    Source string
}

type VideoProviderResult struct {
    UpstreamTaskID string
    Status         ProviderTaskStatus
    Progress       int
    ResultURL      string
    FailureReason  string
    NoRefund       bool
}
```

Optional scalar request fields use pointers where absence and explicit zero must be distinguished. All provider adapters validate values before they become billing multipliers.

The neutral package will also provide tested helpers for:

- submit ID extraction and public ID replacement;
- status synonym normalization;
- nested result URL extraction, including `metadata.url`;
- provider error extraction;
- capability/profile resolution;
- route-specific public status conversion.

Provider adapters remain responsible for provider URLs, headers, request encoding, exact validation, response envelopes, status semantics, and billing. Helpers will express stable domain behavior and be shared by multiple adapters or framework paths; mechanical single-use functions will remain inline.

**Alternative considered:** make SilkRoad's current adapter the base class and embed it in future adapters. Rejected because SilkRoad request fields, model naming, billing, and endpoints are provider behavior, and Go embedding does not provide safe virtual dispatch.

**Alternative considered:** build a fully configurable request-mapping DSL. Rejected because protocol correctness, billing bounds, media constraints, and error handling would become runtime configuration risks instead of testable code.

### 2. Introduce a video-only SilkRoad channel type

Add `ChannelTypeSilkRoad` at the next available stable numeric channel ID before `ChannelTypeDummy`. Update backend and frontend channel names, ordering, base URL storage, and tests. The channel declares only `EndpointTypeOpenAIVideo`.

The dedicated channel:

- has no synchronous `channel.Adaptor`;
- registers a SilkRoad `TaskAdaptor`;
- accepts normal channel instance fields: base URL, API key, proxy, models, and model mapping;
- uses channel-type-wide SilkRoad capability/profile settings;
- rejects routing for all non-video endpoint types.

`ChannelTypeNewAPI` retains its existing synchronous adapter and endpoint capabilities. It will no longer be eligible for new video submissions.

The adapter lookup used by background polling must continue resolving historical platform `"60"` tasks to the legacy SilkRoad task protocol. Submission eligibility and historical polling compatibility must therefore be separated: retaining a legacy polling adapter for platform 60 must not make NewAPI channels eligible for new video requests.

**Alternative considered:** automatically convert channels whose base URL looks like SilkRoad. Rejected because base URLs are operator-defined and may use reverse proxies; the operator will manually add a new SilkRoad channel.

### 3. Normalize both public APIs into one internal request

Both route families remain:

```text
POST/GET /v1/video/generations
POST/GET /v1/videos
GET      /v1/videos/{task_id}/content
```

The legacy route parser accepts the current friendly JSON contract. The OpenAI-style route parser accepts its supported JSON or multipart contract. Both produce `VideoGenerateRequest` and enter the same distributor, billing, task adapter, task row, polling, and storage flow.

Responses remain route-specific:

- `/v1/video/generations/{id}` returns the existing task response envelope;
- `/v1/videos/{id}` returns a flat OpenAI-style video object through a common converter contract;
- both use the public task ID and sanitized local result state.

The provider's upstream path is independent from the client route. The SilkRoad adapter always uses its upstream `/v1/video/generations` path.

### 4. Use a two-phase provider-success and delivery-success state machine

Provider completion is not public completion. The persisted lifecycle becomes:

```text
NOT_START → SUBMITTED/QUEUED → IN_PROGRESS
                                  │
                                  ▼
                               STORING
                              /       \
                         SUCCESS     FAILURE
                      storage ready  delivery failed
```

Add an internal `STORING` task status or an equivalent explicit persisted state. Public API converters map `STORING` to `processing`. Provider polling excludes `STORING`; the storage worker owns it.

On provider success:

1. Parse and privately save the upstream result address.
2. Redact playable addresses from task data.
3. Perform provider-success billing settlement.
4. Atomically transition the task to `STORING` with storage `pending`.
5. Let the storage worker claim it.

On storage ready:

1. Verify that the stored file can be opened and has nonzero expected content.
2. Persist storage metadata and expiry.
3. Clear the no-longer-needed upstream result address when safe.
4. Atomically transition to `SUCCESS`.

On exhausted storage retries:

1. Persist storage `failed`, last error, and retry count.
2. Transition to `FAILURE` with a delivery-specific failure code.
3. Persist a no-automatic-refund marker.
4. Return user-safe contact-administrator guidance.

This distinguishes provider cost from delivery state and prevents repeated upstream polling after generation already succeeded.

### 5. Generalize output ingest behind a storage driver

Replace SilkRoad-named result storage entry points with a provider-neutral storage service. Define a small driver contract around stable storage behavior:

```go
type VideoStorageDriver interface {
    Store(ctx context.Context, taskID string, source io.Reader, metadata VideoObjectMetadata) (StoredVideo, error)
    Open(ctx context.Context, objectKey string) (VideoReadHandle, error)
    Delete(ctx context.Context, objectKey string) error
}
```

The exact interfaces may be split for writer and reader concerns if that produces clearer dependencies. They must not import provider packages.

The initial local driver:

- runs writes and cleanup only on `ingest_node_name`;
- downloads with the SSRF-protected HTTP client;
- writes to a temporary file in the configured local directory;
- syncs/closes and atomically renames to the final object;
- records actual content type, size, path/key, ready time, and fixed expiry;
- serves range-capable content through the existing content endpoint;
- uses `public_download_base_url` to route users to the ingest node in multi-node deployments.

The task core calls the storage service, not local filesystem functions. A later object-storage driver can implement the same contract without changing provider adapters or task states.

**Alternative considered:** proxy the upstream URL without storing it. Rejected because every result must be transferred locally and upstream URLs must never be part of user delivery.

**Alternative considered:** add a shared filesystem. Rejected because the deployment does not need one and future scaling will use public-cloud object storage.

### 6. Treat all provider result URLs as private secrets

Result extraction will recursively recognize provider response URL fields, including current top-level fields and nested `metadata.url`. Extracted upstream addresses are written only to private task data used by storage and administrator diagnostics.

Sanitization occurs at all outward boundaries:

- submit responses;
- polling/query responses;
- task DTOs;
- user task logs;
- frontend API normalization;
- normal application logs.

The content endpoint never redirects or falls back to `UpstreamResultURL`. If the stored object is not ready, failed, missing, or expired, it returns an appropriate local error.

Administrator diagnostics may display the private upstream address only inside the authorized recovery view. General logs and audit records store identifiers and safe error text, not the media URL itself.

### 7. Fix retention at seven days from storage readiness

`StorageReadyAt` and `StorageExpiresAt = StorageReadyAt + 7*24h` are persisted. The duration is a constant, not an option.

Cleanup:

- claims ready objects whose expiry has passed;
- deletes the stored file/object;
- marks storage `expired`;
- clears local result addresses, upstream result addresses, and playable fields left in private/task data;
- remains idempotent if the object is already missing;
- prevents administrator retry or preview after expiry.

Task and billing metadata remain for audit and accounting, but no playable address remains. Existing completed SilkRoad storage rows continue to be recognized by the generic cleanup path.

### 8. Use common provider settings plus sparse profile overrides

Introduce provider-neutral generic settings and reshape SilkRoad settings:

```text
video_setting
├── enabled / tool groups
└── storage
    ├── driver = local
    ├── local_dir
    ├── max_retry
    ├── ingest_node_name
    └── public_download_base_url

silkroad_setting
├── common capability options
├── default_profile_id
└── profiles[]
    ├── id / label
    ├── exact_models[]
    ├── model_prefixes[]
    └── sparse overrides
```

Retention is not represented as an editable field.

Resolution order is deterministic:

1. exact upstream/public model contract match;
2. longest model-prefix match;
3. selected default profile.

The profile inherits omitted values from SilkRoad common settings. The adapter's code-defined hard limits remain authoritative. The profile may narrow values and provide real protocol differences such as a `seconds` string field versus a numeric `duration` field, but it cannot enable an unsupported operation.

A saved configuration must contain at least one valid profile and a `default_profile_id` referencing an enabled profile. Default fallback use is added to administrator diagnostics because it can indicate missing model classification.

**Alternative considered:** duplicate a complete configuration for every profile. Rejected because most SilkRoad behavior is shared and duplicated settings drift.

### 9. Move to a top-level Video Configuration workspace

Add a first-class System Settings category:

```text
Video Configuration
├── General
├── Storage
└── SilkRoad
```

Ownership:

- General: video tool enablement, allowed API-key groups, and generic task behavior exposed to administrators.
- Storage: local driver path, retry count, ingest node, public base, and a read-only statement of the seven-day rule.
- SilkRoad: common capabilities, profiles, matching, and default profile.
- Channel instances: base URL, key, proxy, models, and model mapping only.

Existing `silkroad_setting.storage` and `video_tool_groups` values are copied or compatibly read into `video_setting`; profiles remain SilkRoad-owned. New explicit keys take precedence over legacy keys. The transition must be safe during a rolling deployment and must not reset valid installations.

### 10. Make the Video Generation UI capability-driven

Rename the extension and route:

```text
Seedance video tool → Video Generation
/extensions/seedance → redirect to /extensions/video
```

The sanitized video tool configuration endpoint returns server capabilities resolved for model profiles. The frontend stops using its duplicated `HARDCODED_GENERATION_TYPES` as a source of truth and renders supported modes, durations, ratios, and media limits from the server.

The backend remains authoritative. A modified client cannot bypass provider capability or billing validation.

The first implementation still supports the existing SilkRoad input behavior, including data URLs accepted by SilkRoad. The neutral media source is intentionally opaque to the framework; no public input staging is added.

All route names, settings, statuses, validation messages, failure guidance, and administrator actions use frontend i18n and are translated into all supported locales through the project i18n workflow.

### 11. Add administrator recovery with explicit authorization and audit

The current content controller resolves tasks by `(user_id, task_id)`, which prevents administrators from previewing other users' tasks even when task logs show them. Keep the owner-scoped query for normal users and add an explicit administrator-only task lookup for video operations.

Authorization behavior:

- ordinary users: owner scope only;
- administrator and super administrator roles: any unexpired stored video;
- all users: no access after expiry;
- upstream URLs: never returned through the content endpoint.

Add administrator APIs and task-log actions for:

1. **Retry storage**: allowed only for non-refunded delivery failures; atomically returns the task to `STORING`.
2. **Confirm upstream result**: re-queries or validates provider diagnostics and shows private details only in the authorized view.
3. **Full refund**: allowed only for a non-refunded terminal delivery failure; refunds the full charged task quota.

The refund operation uses a database transaction, `lockForUpdate(tx)` where supported, and a persisted refund state/idempotency marker. It accepts no amount. A refunded task cannot later retry, complete, or serve content. Storage completion and refund race through mutually exclusive compare-and-set/locked transitions so the final state is either delivered-and-charged or undelivered-and-refunded.

Administrative previews and actions produce request-correlated audit records with actor, target task, action, outcome, and safe reason. Media URLs and credentials are excluded from general audit fields.

### 12. Preserve billing invariants at both provider and delivery boundaries

SilkRoad duration is validated before `EstimateBilling`, bounded by the project maximum, and applied with `PriceData.AddOtherRatio`. Checked quota conversion and saturation audit remain in the generic submit flow.

Billing transitions are:

- provider failure: existing automatic refund policy;
- provider success: settle provider billing, then enter storage;
- storage transient failure: no billing change;
- storage terminal failure: no automatic refund;
- administrator full refund: one idempotent full task refund;
- successful storage: no second charge.

A persisted delivery no-refund marker is necessary because storage failure occurs after the provider polling result and cannot rely only on the transient `TaskInfo.NoRefund` value.

## Risks / Trade-offs

- **[Legacy adapter remains reachable internally]** → Separate submission eligibility from polling adapter lookup and add tests proving NewAPI channels are not selected for new video submissions.
- **[Default profile can hide missing model classification]** → Validate against hard adapter limits and emit administrator diagnostics whenever fallback is used.
- **[Mandatory ingest increases completion latency]** → Expose storage as public `processing`, use a dedicated worker, and record storage progress without repeating provider requests.
- **[Local disk exhaustion blocks every video completion]** → Validate configured paths, write atomically, expose worker errors, monitor free space where available, and provide administrator retry/refund recovery.
- **[Single ingest node is an availability dependency]** → Preserve clear node health diagnostics and make the storage driver replaceable by cloud object storage later.
- **[Provider result URL expires before retries finish]** → Claim storage promptly after provider success, retry within bounded intervals, and surface terminal failure for manual recovery.
- **[Cross-user administrator preview increases privacy exposure]** → Require administrator authentication, keep access task-scoped, serve only local content, and audit every cross-user preview.
- **[Refund and late storage completion race]** → Use transactional state guards and prohibit delivery after the persisted refund marker.
- **[Rolling deployment reads two setting namespaces]** → Define explicit new-key precedence, retain legacy fallback for the transition, and test mixed-version configuration reads.
- **[Existing local files use SilkRoad naming and MP4 assumptions]** → Keep backward-compatible path lookup while new generic metadata records content type and storage key.
- **[Frontend capability migration changes a large component]** → Introduce a versioned sanitized capability response, add behavior tests for rendering and request construction, and preserve current SilkRoad defaults.

## Migration Plan

1. Add provider-neutral types, status/storage fields, configuration readers, and tests without changing routing.
2. Introduce the generic storage service and local driver with backward-compatible reads of existing SilkRoad storage metadata and files.
3. Add the `STORING` lifecycle, mandatory ingest behavior, redaction, fixed expiry, and storage failure no-refund persistence.
4. Add the dedicated SilkRoad channel type and adapter. Keep platform-60 legacy polling support but remove NewAPI video submission eligibility.
5. Add generic video settings with legacy setting fallback and expose the sanitized capability API.
6. Add administrator preview, diagnostics, retry, full refund, and audit endpoints.
7. Move frontend settings into Video Configuration, make the tool server-driven, rename the route, and add the legacy redirect.
8. Deploy while retaining the existing NewAPI SilkRoad channel. The operator manually creates a new SilkRoad channel and validates it.
9. Route new SilkRoad model traffic to the new channel. Keep the old channel until its unfinished tasks are drained; no rows are rewritten automatically.
10. After validation, disable the old channel for new routing but retain it if historical unfinished tasks still require its credentials.

Rollback:

- Disable the new SilkRoad channel and route new traffic back only if the old application still supports that path.
- Keep legacy setting keys and old channel rows untouched.
- Do not roll back while new-platform tasks are unfinished unless the previous release can poll the new platform; drain or pause submissions first.
- Generic storage metadata remains backward-compatible JSON, but stored files created under new keys must retain the compatibility lookup until rollback risk has passed.

## Open Questions

No product decisions remain open for this proposal. Implementation should confirm the existing administrator authorization capability used by task-log management and reuse the project's established audit and billing transaction mechanisms rather than introducing parallel systems.
