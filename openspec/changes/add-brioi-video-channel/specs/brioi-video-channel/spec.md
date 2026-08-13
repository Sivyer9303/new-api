## ADDED Requirements

### Requirement: Brioi is an independent video-only channel
The system SHALL register Brioi under its own channel type and task adaptor. The Brioi adaptor MUST NOT depend on NewAPI or SilkRoad provider packages, and Brioi channels MUST NOT be eligible for non-video request paths.

#### Scenario: Brioi channel receives a video generation request
- **WHEN** an eligible Video Generation request is distributed to a configured Brioi channel
- **THEN** the system uses the Brioi task adaptor and Brioi protocol

#### Scenario: Non-video request considers a Brioi channel
- **WHEN** a chat, image, audio, embedding, or other non-video request is distributed
- **THEN** the Brioi channel is excluded from eligible channels

### Requirement: Brioi base URL is administrator supplied
The system SHALL require a non-empty base URL for every Brioi channel and MUST NOT provide a default Brioi base URL.

#### Scenario: Administrator saves a Brioi channel without a base URL
- **WHEN** the submitted Brioi channel has a missing or blank base URL
- **THEN** the system rejects the channel configuration with a Brioi-specific validation error

#### Scenario: Brioi request URL is built
- **WHEN** a configured Brioi base URL has an optional trailing slash
- **THEN** the adaptor normalizes that slash and appends the documented Brioi path exactly once

### Requirement: Brioi channel credentials can be tested without generating media
The system SHALL test a Brioi channel by calling `<configured-base-url>/v1/models` with the configured Bearer key and MUST NOT create a billable video task during the test.

#### Scenario: Valid Brioi credentials are tested
- **WHEN** the Brioi models endpoint returns a successful model list
- **THEN** the channel test reports success and the available models

#### Scenario: Invalid Brioi credentials are tested
- **WHEN** the Brioi models endpoint rejects the configured key
- **THEN** the channel test reports the upstream authentication failure without submitting a video task

### Requirement: First-phase Brioi models and generation modes are bounded
The system SHALL support `seedance-2-0-fast`, `seedance-2-0`, and `seedance-2-5` for text-to-video, ordinary single/multi-image reference, strict first-frame, and strict first/last-frame requests from `/v1/video/generations`. The system MUST reject unsupported models, media types, roles, combinations, durations, resolutions, aspect ratios, and item counts before upstream submission.

#### Scenario: Valid Seedance 2.0 request
- **WHEN** a request uses a supported Seedance 2.0 model with duration 4–15 and options allowed by that model
- **THEN** the system accepts the request for Brioi submission

#### Scenario: Valid Seedance 2.5 request
- **WHEN** a request uses `seedance-2-5` with duration 4–29, resolution 480p or 720p, and aspect ratio 16:9 or 9:16
- **THEN** the system accepts the request for Brioi submission

#### Scenario: Unsupported public client route is used
- **WHEN** a first-phase request attempts to select Brioi through the public client `/v1/videos` submission route
- **THEN** the Brioi channel is not eligible for that submission

#### Scenario: Video or audio reference media is supplied
- **WHEN** a first-phase Brioi request contains video or audio reference media
- **THEN** the system rejects the request before upstream submission

### Requirement: Uploaded images are staged to R2 before Brioi submission
The system SHALL upload every image selected in the Video Generation page to the configured R2 input storage before creating a Brioi task. The Brioi request MUST contain signed HTTPS URLs and MUST NOT contain base64 or data URLs.

#### Scenario: Image staging succeeds
- **WHEN** all uploaded images pass validation, upload to R2, and receive signed HTTPS URLs
- **THEN** the system preserves their order and roles and submits only those URLs to Brioi

#### Scenario: Any image staging step fails
- **WHEN** content validation, R2 quota checking, upload, or URL signing fails for any image
- **THEN** the system returns an error and does not call Brioi's create-task endpoint

#### Scenario: Signed input media is recorded
- **WHEN** a Brioi task is created with staged media
- **THEN** the system excludes the signed input URLs from public task data and ordinary application logs

### Requirement: Brioi requests use the documented Seedance payload
The system SHALL submit JSON to `<configured-base-url>/v1/videos` with `model`, `prompt`, integer `duration`, `resolution`, `aspect_ratio`, and an optional `ref` array. Each image reference SHALL use `type=image`; strict frame references SHALL use the corresponding `first_frame` or `last_frame` role.

#### Scenario: Text-to-video payload is built
- **WHEN** a valid Brioi text-to-video request has no images
- **THEN** the upstream body omits `ref`

#### Scenario: Ordinary image-reference payload is built
- **WHEN** a valid Brioi request contains ordinary reference images
- **THEN** each image is emitted as a `ref` item with its staged URL and `type=image` without a strict-frame role

#### Scenario: Strict first and last frames are built
- **WHEN** a valid strict first/last-frame request contains two staged images
- **THEN** the system emits one `first_frame` and one `last_frame` item regardless of their upstream array positions

#### Scenario: Provider-foreign fields are present
- **WHEN** the client request includes internal generation controls that have already been normalized
- **THEN** the Brioi body excludes `generation_type`, base64 media, SilkRoad-only fields, and chat-only fields

### Requirement: Brioi task responses are normalized safely
The system SHALL extract a submit identifier from non-empty `id` or `task_id`, poll `<configured-base-url>/v1/videos/{task_id}`, normalize documented status aliases, and extract completed media from `metadata.url` or `result_url`.

#### Scenario: Brioi task is still running
- **WHEN** polling returns `queued`, `pending`, `processing`, or `in_progress`
- **THEN** the system keeps the same persisted task non-terminal and polls the same upstream identifier again

#### Scenario: Brioi task completes with a result
- **WHEN** polling returns `completed` with a non-empty supported result URL
- **THEN** the system records the URL privately and enters the existing mandatory result-storage lifecycle

#### Scenario: Brioi task completes without a result
- **WHEN** polling returns `completed` without `metadata.url` or `result_url`
- **THEN** the system does not expose public success and preserves the anomaly for administrator review

#### Scenario: Brioi returns an unknown status
- **WHEN** polling returns a status outside the known aliases
- **THEN** the task fails with a no-automatic-refund marker and administrator-safe diagnostic information

#### Scenario: Polling receives a temporary transport or server error
- **WHEN** polling times out or receives a retryable 5xx response
- **THEN** the system retries the same upstream task and does not create another task

### Requirement: Generic video storage recognizes Brioi without provider allow-lists
The system SHALL classify new Brioi tasks through provider-neutral video task metadata and route them through the common storage, URL-redaction, cleanup, content-delivery, diagnostics, and administrator recovery lifecycle. Generic lifecycle code MUST NOT require Brioi-specific platform checks.

#### Scenario: Brioi provider reports successful generation
- **WHEN** a marked Brioi video task returns a normalized provider success with a result URL
- **THEN** the common video lifecycle stores the URL privately, redacts it publicly, and starts mandatory result storage

#### Scenario: A historical video task lacks the marker
- **WHEN** an unmarked historical task matches a preexisting video action or platform heuristic, including legacy platform `"60"` or `"61"`
- **THEN** the compatibility fallback continues applying the video storage and redaction lifecycle

#### Scenario: Future video provider uses the common marker
- **WHEN** a new provider creates a task with provider-neutral video metadata
- **THEN** generic lifecycle behavior works without adding its platform ID to a provider allow-list

### Requirement: Brioi billing uses central model pricing
The system SHALL validate the request duration against the selected Brioi model limit and provide that bounded duration as the existing `seconds` billing multiplier. The existing model pricing configuration SHALL remain authoritative for fixed versus per-second billing.

#### Scenario: Per-second Brioi model is billed
- **WHEN** the selected model billing mode is per-second
- **THEN** backend billing applies the validated duration multiplier through the existing quota pipeline

#### Scenario: Fixed-price Brioi model is billed
- **WHEN** the selected model billing mode is fixed per request
- **THEN** backend billing charges the configured fixed model price without duplicating billing mode in Brioi settings

### Requirement: Accepted asynchronous tasks are durable before success is returned
The system SHALL persist a recoverable task row before calling an asynchronous video create-task endpoint. After provider acceptance, it MUST persist the upstream task identifier and public response data before settling billing and before returning client success. Provider adaptors MUST NOT write the client response directly.

#### Scenario: Initial task persistence fails
- **WHEN** the system cannot persist the pre-submission task row
- **THEN** it does not call the provider and returns a local service error

#### Scenario: Provider accepts a task
- **WHEN** the create-task response contains a valid upstream identifier
- **THEN** the system durably updates the existing public task row before settling and returning the public task identifier

#### Scenario: Persistence fails after provider acceptance
- **WHEN** the provider accepted the task but the accepted-state update cannot be committed
- **THEN** the system does not report client success or automatically refund the reserved charge and preserves the pre-submission row for recovery or administrator review

### Requirement: Video success requires a usable provider result
The common video polling lifecycle SHALL require a non-empty supported result URL before any provider success can enter billing settlement or result storage. This invariant applies independently of provider-specific parser defenses.

#### Scenario: SilkRoad reports completion without a result URL
- **WHEN** a SilkRoad polling response normalizes to success but contains no usable result URL
- **THEN** the task enters administrator review with automatic refund withheld and does not enter storage

#### Scenario: Brioi reports completion without a result URL
- **WHEN** a Brioi polling response normalizes to success but contains no usable result URL
- **THEN** the same common administrator-review behavior applies

### Requirement: Video settlement is resumable and idempotent
The system SHALL treat billing settlement as a durable recoverable transition. A task left in `settling` after a database or compare-and-swap failure MUST be discoverable by the watchdog and MUST resume without charging or refunding more than once.

#### Scenario: Storage exposure update fails after settlement starts
- **WHEN** the transition from settlement to pending storage fails
- **THEN** a later watchdog run resumes the transition and exposes the task to storage exactly once

#### Scenario: Two workers observe the same provider success
- **WHEN** concurrent polling workers attempt settlement
- **THEN** only one worker performs the billing transition and both converge on one durable task state

### Requirement: Polling infrastructure failures do not impersonate provider failures
The polling lifecycle SHALL distinguish transient channel cache/database failures from definitive channel removal. It MUST NOT bulk-overwrite affected tasks as ordinary provider failures.

#### Scenario: Channel lookup fails transiently
- **WHEN** polling cannot temporarily load channel configuration
- **THEN** tasks remain non-terminal and are retried on a later polling cycle

#### Scenario: Channel was definitively removed
- **WHEN** polling confirms that the task's channel no longer exists
- **THEN** the task enters administrator review with its upstream identifier and reserved billing context preserved

### Requirement: Task submission HTTP responses are handled safely
The common task submission flow SHALL accept all 2xx provider responses, close every response body, limit and sanitize non-success response bodies, and retry only statuses classified as retryable.

#### Scenario: Provider returns 201 or 202
- **WHEN** the response contains a valid task identifier
- **THEN** the system persists that task and does not send a duplicate create request

#### Scenario: Provider returns a non-success response
- **WHEN** the provider rejects or temporarily fails the submission
- **THEN** the response body is closed, sensitive body content is not exposed, and retry behavior follows the status classification
