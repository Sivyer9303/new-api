## 1. Provider-Neutral Video Domain

- [x] 1.1 Add normalized video request, media, provider result, capability, and profile-resolution types without introducing provider imports into the common task layer
- [x] 1.2 Implement exact-match, longest-prefix, and selected-default profile resolution with inheritance from provider common settings
- [x] 1.3 Add deterministic tests for profile precedence, default fallback diagnostics, inheritance, and hard-capability rejection
- [x] 1.4 Extract shared submit ID, status, error, progress, and nested result URL parsing helpers from the current NewAPI task adapter
- [x] 1.5 Add provider-neutral parser tests covering flat and nested envelopes, `metadata.url`, empty status, known status synonyms, and unknown no-refund status

## 2. Dedicated SilkRoad Channel

- [x] 2.1 Add the stable SilkRoad channel constant, name, base URL slot, frontend metadata, display ordering, and registration tests while keeping `ChannelTypeDummy` last
- [x] 2.2 Declare SilkRoad eligible only for the OpenAI video endpoint type and add routing tests proving it is excluded from chat, image, audio, and other endpoints
- [x] 2.3 Move SilkRoad request validation, request conversion, polling, billing estimation, and response parsing from `task/newapi` into a dedicated SilkRoad task adapter using the neutral helpers
- [x] 2.4 Register the dedicated adapter for new SilkRoad tasks and preserve model mapping, bounded per-second billing, public task IDs, and upstream URL redaction
- [x] 2.5 Separate new-submission eligibility from historical platform adapter lookup so NewAPI channels cannot receive new video tasks while platform-60 legacy tasks remain pollable
- [x] 2.6 Add adapter and routing regression tests for dedicated SilkRoad submission, legacy NewAPI polling, unsupported endpoint rejection, and unknown model fallback

## 3. Generic Video Settings Backend

- [x] 3.1 Introduce provider-neutral video settings for tool enablement, allowed API-key groups, local storage path, retry count, ingest node, and public download base
- [x] 3.2 Reshape SilkRoad settings into common values, sparse profile overrides, exact models, model prefixes, and a required administrator-selected default profile
- [x] 3.3 Enforce adapter hard limits and reject invalid, missing-default, deleted-default, ambiguous, or unsupported SilkRoad configurations
- [x] 3.4 Add compatibility reads and deterministic precedence for existing `silkroad_setting.storage`, `profiles`, and `video_tool_groups` values without migrating channel or task rows
- [x] 3.5 Replace the SilkRoad-specific per-second binding warning with provider/profile-aware validation and administrator diagnostics for default-profile fallback
- [x] 3.6 Add backend setting validation and compatibility tests across default, legacy-only, new-only, and mixed rolling-deployment configurations

## 4. Mandatory Video Result Storage

- [x] 4.1 Define the provider-neutral video storage driver contract and generic stored-object metadata
- [x] 4.2 Implement the local driver with designated ingest-node gating, SSRF-protected source download, temporary files, atomic rename, content metadata, open, and idempotent delete
- [x] 4.3 Generalize SilkRoad ingest, cleanup, storage path, and URL code into provider-neutral video services while retaining backward-compatible lookup for existing files and private data
- [x] 4.4 Add the persisted storing/delivery state and transition provider success to storing without exposing public completion or repeating provider polling
- [x] 4.5 Transition to success only after the local driver verifies the stored object is readable; make the content endpoint serve only storage-driver content with no upstream fallback
- [x] 4.6 Implement bounded retry claiming, retry counts, safe errors, and terminal non-refundable delivery failure with user contact guidance
- [x] 4.7 Fix expiry at exactly seven days from storage readiness, remove retention editing, and clear local and upstream playable addresses during idempotent cleanup
- [x] 4.8 Add concurrency and lifecycle tests for overlapping pollers, storage claims, atomic writes, retry exhaustion, success transition, cleanup races, missing files, and expiry without administrator bypass

## 5. Public Video API Compatibility

- [x] 5.1 Route both `/v1/video/generations` and `/v1/videos` submissions through the normalized video request and the same persisted task lifecycle
- [x] 5.2 Preserve the friendly JSON parser and legacy task response envelope for `/v1/video/generations`
- [x] 5.3 Implement flat OpenAI-style submit and query conversion for `/v1/videos`, including SilkRoad tasks and storing/delivery-failure states
- [x] 5.4 Ensure every submit, query, task DTO, and content response uses the public task ID and contains no upstream result address
- [x] 5.5 Add API contract tests for both route families across submit, queued, generating, storing, completed, provider failure, delivery failure, and expired states

## 6. Administrator Access and Recovery

- [x] 6.1 Add an explicit administrator-only unscoped video task lookup while preserving owner-scoped lookups for normal task and content APIs
- [x] 6.2 Permit administrators and super administrators to preview any user's unexpired stored video and deny cross-user ordinary access and all expired access
- [x] 6.3 Add administrator diagnostics that safely expose upstream task status, private result address, storage attempts, and last transfer error without leaking those fields to general logs
- [x] 6.4 Add a concurrency-safe retry-storage operation restricted to non-refunded delivery failures
- [x] 6.5 Add an upstream-confirmation operation that refreshes provider diagnostics without completing or refunding the task implicitly
- [x] 6.6 Add a transactional, idempotent, full-only manual refund operation using the shared billing path and persisted refund state
- [x] 6.7 Prevent retry, completion, and content delivery after refund and resolve storage-completion/refund races into delivered-and-charged or undelivered-and-refunded
- [x] 6.8 Record request-correlated audit events for cross-user preview, diagnostics, retry, confirmation, refund success, and rejected actions
- [x] 6.9 Add authorization, privacy, idempotency, billing, and race regression tests for all administrator operations

## 7. Video Configuration Frontend

- [x] 7.1 Add a top-level Video Configuration system-settings category with General, Storage, and SilkRoad routes and sidebar navigation
- [x] 7.2 Move generic tool-group and storage controls out of Extensions, display the fixed seven-day policy without an editable retention input, and preserve current configured values
- [x] 7.3 Refactor the SilkRoad settings form into common capability values, sparse profile overrides, matching rules, and an explicit default-profile selector
- [x] 7.4 Display backend validation errors and default-fallback diagnostics without allowing options outside adapter hard limits
- [x] 7.5 Remove the old SilkRoad section from Extensions and add route compatibility if administrators follow an old settings link
- [x] 7.6 Add frontend behavior tests for settings navigation, default-profile validation, inheritance editing, legacy values, and fixed retention presentation

## 8. Generic Video Generation Extension

- [x] 8.1 Rename the Seedance extension to Video Generation, add `/extensions/video`, and redirect `/extensions/seedance` while preserving the existing API-key label changes in the modified page
- [x] 8.2 Replace SilkRoad-specific feature flags, config endpoint names, React Query keys, sidebar labels, disabled-state copy, and route guards with provider-neutral video equivalents
- [x] 8.3 Version and consume the sanitized server capability response instead of frontend `HARDCODED_GENERATION_TYPES`
- [x] 8.4 Render model generation modes, durations, aspect ratios, and media constraints from resolved server capabilities while retaining backend validation as authoritative
- [x] 8.5 Update submit, polling, storing-state progress, delivery-failure guidance, local preview, and expired-state behavior for the common task lifecycle
- [x] 8.6 Add frontend tests for capability rendering, model/profile resolution, request construction, both API response normalizers, storing state, no-URL leakage, preview authorization errors, and route redirect

## 9. Task Log Recovery UI and Internationalization

- [x] 9.1 Add administrator task-log controls for cross-user preview, retry storage, confirm upstream result, and one full refund with confirmation and terminal-state gating
- [x] 9.2 Show safe delivery-failure guidance to users and detailed private diagnostics only to authorized administrators
- [x] 9.3 Add accessible loading, disabled, success, duplicate-refund, and error states for recovery actions
- [x] 9.4 Add all new static video, settings, storage, status, audit, and recovery text through `useTranslation()` and register any dynamic static keys
- [x] 9.5 Run the project i18n synchronization workflow and complete translations for en, zh, zh-TW, fr, ja, ru, and vi
- [x] 9.6 Add task-log interaction and authorization-visibility tests from ordinary-user, administrator, and super-administrator perspectives

## 10. Verification and Deployment Readiness

- [x] 10.1 Run focused Go tests for channel routing, provider adapters, settings, task polling, billing, storage, proxying, cleanup, authorization, audit, and administrator recovery
- [x] 10.2 Run cross-database-relevant model and service tests and verify all new GORM queries and locking use SQLite, MySQL, and PostgreSQL-compatible patterns
- [x] 10.3 Run frontend unit tests, type checking, affected-file linting, i18n checks, and the production build with Bun
- [x] 10.4 Verify no user-visible API, task log, browser response, or normal application log exposes an upstream video address
- [x] 10.5 Perform an end-to-end mock flow for provider submit, provider poll, storing, local playback, expiry, storage failure, administrator retry, and idempotent full refund
- [x] 10.6 Document the operator rollout: configure the ingest node, create a new SilkRoad channel manually, test it, retain the old NewAPI channel until unfinished tasks drain, and avoid automatic record migration
