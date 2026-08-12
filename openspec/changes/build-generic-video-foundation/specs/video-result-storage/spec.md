## ADDED Requirements

### Requirement: Mandatory stored delivery
Every successful upstream video result SHALL be copied into the configured video storage before the public task becomes completed. The system SHALL never return, redirect to, or fall back to an upstream video URL for user delivery.

#### Scenario: Upstream generation completes
- **WHEN** a provider reports successful generation with a result address
- **THEN** the system stores the address privately, enters the storage phase, and keeps the public task in `processing`

#### Scenario: Stored file becomes ready
- **WHEN** the storage driver confirms that the complete video file is readable
- **THEN** the system marks storage ready and exposes the task as `completed`

#### Scenario: Stored content request
- **WHEN** an authorized user requests `/v1/videos/{task_id}/content` for a completed unexpired task
- **THEN** the system serves only the stored object through the configured storage driver

### Requirement: Provider-neutral storage driver
The video foundation SHALL define a provider-neutral storage-driver boundary. This change SHALL implement a local-disk driver that writes and reads video results on one designated ingest node and uses a configured public base address for access from other application nodes.

#### Scenario: Local ingest node processes a result
- **WHEN** a pending storage task is claimed by the configured ingest node
- **THEN** the local driver downloads the upstream result with SSRF protections, writes it atomically, verifies readability, and marks it ready

#### Scenario: Non-ingest node runs the worker
- **WHEN** a process is not the configured ingest node
- **THEN** it does not claim or write local video results

#### Scenario: Future object storage driver
- **WHEN** a future cloud storage driver is registered
- **THEN** the task lifecycle can use it without provider-specific changes or a shared local filesystem

### Requirement: Private upstream result data
Upstream result addresses and provider response fields containing playable addresses SHALL be stored only in administrator-private task data while needed for ingest. They SHALL be removed from every user-visible API response, log payload, and frontend data source.

#### Scenario: Provider returns URLs in nested response data
- **WHEN** a submit or poll response contains `url`, `video_url`, `result_url`, `metadata.url`, or equivalent nested provider fields
- **THEN** the system extracts the address privately and redacts it from user-visible task data

#### Scenario: User queries a pending-storage task
- **WHEN** a user queries a task whose upstream generation succeeded but storage is not ready
- **THEN** the response reports `processing` without any upstream result address

### Requirement: Storage retry and terminal delivery failure
The storage service SHALL retry failed result transfers up to the configured retry limit. Exhausting retries after upstream success SHALL produce a delivery-failure state that does not automatically refund the task quota and never exposes the upstream URL.

#### Scenario: Transient transfer error
- **WHEN** a storage attempt fails before reaching the retry limit
- **THEN** the task remains externally `processing`, the retry count and error are recorded privately, and a later worker pass retries it

#### Scenario: Retry limit exhausted
- **WHEN** the final permitted storage attempt fails
- **THEN** the task becomes a non-refundable delivery failure, the user receives instructions to contact an administrator with the task identifier, and the original charge remains

#### Scenario: Upstream generation itself fails
- **WHEN** the provider reports generation failure before a result is available
- **THEN** the normal provider failure policy applies, including automatic refund when allowed by that policy

### Requirement: Fixed seven-day retention
Every stored video SHALL expire exactly seven days after the storage driver marks it ready. The retention duration SHALL not be extendable by configuration or administrator privilege.

#### Scenario: Video remains within retention
- **WHEN** fewer than seven days have elapsed since storage became ready
- **THEN** an authorized user can access the stored video

#### Scenario: Video expires
- **WHEN** seven days have elapsed since storage became ready
- **THEN** the cleanup process deletes the stored file or object, marks the task expired, and clears all playable local and upstream result addresses

#### Scenario: Administrator requests expired video
- **WHEN** an administrator requests content for an expired task
- **THEN** the system refuses access with the same expiry rule applied to ordinary users

### Requirement: Concurrency-safe storage transitions
Storage claiming, ready transitions, expiry, retry exhaustion, and administrative recovery SHALL be concurrency-safe and idempotent across overlapping workers and application instances.

#### Scenario: Two workers attempt the same transfer
- **WHEN** overlapping worker passes attempt to claim one pending result
- **THEN** only one worker performs the active transfer and terminal transition

#### Scenario: Cleanup races with content access
- **WHEN** expiry cleanup and a content request occur concurrently at the retention boundary
- **THEN** the system never serves a partially deleted file and converges to the expired state
