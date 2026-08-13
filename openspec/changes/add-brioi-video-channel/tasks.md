## 1. Channel Registration and Isolation

- [x] 1.1 Add `ChannelTypeBrioi` before `ChannelTypeDummy`, extend channel names/base URL storage with an empty default, and update stable-ID/count tests.
- [x] 1.2 Declare Brioi as video-only in endpoint capability and request-path eligibility helpers while keeping it ineligible for first-phase `/v1/videos` submissions.
- [x] 1.3 Add Brioi channel validation that requires an administrator-supplied non-blank base URL.
- [x] 1.4 Add Brioi channel creation/edit UI metadata, selector entry, icon fallback, and localized channel name; correct any stale frontend mapping that still labels channel type 61 as NewAPI.
- [x] 1.5 Implement a non-billable Brioi channel test against the configured `/v1/models` endpoint with Bearer authentication and proxy support.
- [x] 1.6 Add backend tests proving Brioi registration, blank base URL rejection, video-only routing, `/v1/videos` deferral, and non-billable model-list testing.

## 2. Brioi Provider Settings

- [x] 2.1 Create an independently registered `brioi_setting` module with provider groups and exact model capability profiles.
- [x] 2.2 Define defaults for `seedance-2-0-fast`, `seedance-2-0`, and `seedance-2-5`, including every documented duration, resolution, aspect ratio, image mode, and item limit in first-phase scope.
- [x] 2.3 Implement profile resolution against the model-mapped upstream name with no default-profile fallback.
- [x] 2.4 Implement validation that permits disabling supported values but rejects unsupported values and invalid profile structures.
- [x] 2.5 Add public Brioi capability DTOs that exclude channel credentials, R2 credentials, and administrator-only fields.
- [x] 2.6 Add deterministic settings tests for defaults, exact matching, unknown-model rejection, hard bounds, sparse disabled options, and public redaction.

## 3. Unique Video Provider Group Ownership

- [x] 3.1 Introduce a provider-neutral video group resolver that combines explicit Brioi groups with backward-compatible SilkRoad group ownership.
- [x] 3.2 Reject a provider settings save when any normalized group is owned by another video provider and return the conflicting group/provider in the error.
- [x] 3.3 Ensure saving Brioi or SilkRoad provider settings validates only that provider plus the cross-provider group invariant, not unrelated storage or general settings.
- [x] 3.4 Add tests for unique ownership, overlap rejection, trimming/deduplication, unowned groups, and legacy SilkRoad ownership.
- [x] 3.5 Add a provider constraint to the relay/distributor context for Video Generation submissions and enforce it when selecting channels.
- [x] 3.6 Add distribution regression tests proving identically named SilkRoad and Brioi models cannot cross-route and cannot fail over across provider types.

## 4. Provider-Constrained Video Model Discovery

- [x] 4.1 Add a Video Generation model query that derives the selected key's effective group and provider ownership server-side.
- [x] 4.2 Filter video tool models by provider channel type, video endpoint eligibility, channel status/group/model mapping, token model limits, and billing availability.
- [x] 4.3 Return an empty, user-safe result when a provider-owned group has no eligible provider channel instead of falling back to another provider.
- [x] 4.4 Add authorization checks ensuring users can query models only for their own API keys.
- [x] 4.5 Add backend tests for same-name provider models, non-video channels in the group, disabled channels, model mappings, token limits, missing billing, and missing eligible channels.

## 5. Provider-Neutral Video Request Preparation

- [x] 5.1 Extend `videocommon` request parsing/validation helpers only for behavior shared by SilkRoad and Brioi, without importing either provider package.
- [x] 5.2 Preserve provider-neutral image type and role information for ordinary reference, first-frame, and last-frame requests.
- [x] 5.3 Ensure all optional scalar fields preserve absent versus explicit-zero values until provider validation.
- [x] 5.4 Add common request tests for text-only, ordinary multi-image, first-frame, first/last-frame, conflicting modes, and unsupported media.
- [x] 5.5 Refactor SilkRoad to consume any newly shared helpers without changing its established public or upstream request behavior.
- [x] 5.6 Run focused SilkRoad adaptor tests to prove the common extraction introduces no protocol regression.

## 6. Brioi Task Adaptor

- [x] 6.1 Create `relay/channel/task/brioi` as a sibling package implementing `channel.TaskAdaptor` with no imports from NewAPI or SilkRoad provider packages.
- [x] 6.2 Implement Brioi initialization, explicit-base URL normalization, Bearer headers, submit URL, poll URL, proxy-aware requests, and provider identity methods.
- [x] 6.3 Parse `/v1/video/generations` requests into the provider-neutral request, resolve the mapped Brioi model profile, and enforce prompt/model/mode bounds.
- [x] 6.4 Validate Seedance 2.0/Fast durations 4–15, model-specific resolutions, six documented aspect ratios, and maximum nine ordinary reference images.
- [x] 6.5 Validate Seedance 2.5 durations 4–29, 480p/720p, 16:9/9:16, and maximum thirty ordinary reference images.
- [x] 6.6 Reject video/audio references, last-frame without first-frame, duplicate strict roles, strict/ordinary mixing, and all unsupported top-level fields before upstream submission.
- [x] 6.7 Build Brioi-native bodies with integer `duration`, explicit `resolution`/`aspect_ratio`, optional `ref`, and no client-only or provider-foreign fields.
- [x] 6.8 Add golden request tests for every supported model family and text, ordinary image, multi-image, first-frame, and first/last-frame modes.
- [x] 6.9 Add negative boundary tests for durations, resolutions, ratios, item counts, role/type combinations, unknown models, and forbidden media.

## 7. R2 Input Staging for Brioi

- [x] 7.1 Stage every Brioi image data URL through the existing R2 input service before constructing the upstream `ref` array.
- [x] 7.2 Preserve image order and roles while replacing every inline source with a signed HTTPS URL under the configured input prefix.
- [x] 7.3 Reject Brioi submission when R2 is unavailable, quota-blocked, content validation fails, upload fails, or presigning fails.
- [x] 7.4 Ensure no Brioi create-task request is sent after any partial staging failure and rely on bounded input cleanup for already uploaded objects.
- [x] 7.5 Exclude signed input URLs and base64 payloads from public task data, normal logs, request previews, and error messages.
- [x] 7.6 Add tests asserting successful staging produces only HTTPS URLs, partial failure prevents upstream submission, order/roles are stable, and no sensitive media leaks.

## 8. Brioi Submit, Polling, Storage, and Billing

- [x] 8.1 Register the Brioi task adaptor for new submissions and persisted background polling.
- [x] 8.2 Parse non-empty Brioi submit `id` or `task_id`, replace public response IDs, and never expose provider URLs.
- [x] 8.3 Normalize queued/pending, processing/in_progress, completed, failed, cancelled, and unknown statuses through `videocommon`.
- [x] 8.4 Require `metadata.url` or `result_url` before treating `completed` as provider success, and preserve missing-result diagnostics for administrators.
- [x] 8.5 Ensure temporary poll transport errors and retryable 5xx responses retry the same upstream ID without duplicate creation.
- [x] 8.6 Generalize SilkRoad-named video success-store, ingest, and URL-redaction gates to classify new tasks by `PrivateData.VideoTask` and storage metadata instead of provider platform allow-lists.
- [x] 8.7 Preserve the baseline action/platform fallback for historical unmarked video tasks, including explicit platform 60/61 coverage, and add regression tests for those tasks.
- [x] 8.8 Feed successful Brioi results into the provider-neutral private-URL, settlement, R2 result-storage, verification, seven-day retention, and public content flow.
- [x] 8.9 Apply the existing delivery-failure/no-automatic-refund and administrator retry/confirm/refund policy to Brioi tasks.
- [x] 8.10 Return the validated bounded duration as the `seconds` billing multiplier while leaving fixed/per-second mode in central model pricing.
- [x] 8.11 Add submit/poll parser tests, status/error tests, missing-result tests, no-refund tests, billing-boundary tests, provider-neutral classification tests, and a provider-success-to-stored-delivery integration test.

## 9. Provider-Specific Video Configuration UI

- [x] 9.1 Extend the Video Configuration section registry and routes with an independent Brioi child section while retaining General, Storage, and SilkRoad.
- [x] 9.2 Move provider group ownership controls into the SilkRoad and Brioi sections and display clear unique-group conflict errors.
- [x] 9.3 Build the Brioi profile editor for supported models, durations, resolutions, ratios, generation modes, and image limits without exposing unsupported values.
- [x] 9.4 Keep shared R2 input/result settings only in Storage and shared enablement only in General.
- [x] 9.5 Update settings resolution/save schemas so one section can save without requiring unrelated provider or storage sections to be complete.
- [x] 9.6 Add frontend schema/component tests for separate sections, defaults, hard-bound validation, group conflicts, and independent saves.

## 10. Provider-Aware Video Generation Page

- [x] 10.1 Change the public video tool configuration shape to return sanitized provider configurations and provider group ownership.
- [x] 10.2 Filter the API key selector to keys whose groups have exactly one configured video provider owner.
- [x] 10.3 Resolve the active provider configuration from the selected key group before selecting models or capabilities.
- [x] 10.4 Replace unrestricted `/v1/models` consumption with the provider-constrained video model query and preserve stale-request cancellation when keys change.
- [x] 10.5 Resolve same-name models against the active provider's profiles and reset incompatible model/mode/options when switching keys.
- [x] 10.6 Render Brioi text, ordinary image, multi-image, first-frame, and first/last-frame controls plus provider-specific duration, resolution, and aspect-ratio selectors while keeping video/audio references unavailable.
- [x] 10.7 Reset stale duration/resolution/aspect/mode selections when the key, provider, or model changes, and omit unsupported provider fields after a switch.
- [x] 10.8 Build the existing friendly `/v1/video/generations` request with selected resolution and enough normalized image-role information for the Brioi adaptor.
- [x] 10.9 Update price estimation to use model `billing_mode`: fixed models omit the seconds multiplier and per-second models include it.
- [x] 10.10 Add frontend tests for key-group provider selection, same-name models across groups, no eligible models, provider switching, resolution changes, request bodies, and fixed/per-second estimates.

## 11. Internationalization, Security, and Verification

- [x] 11.1 Add all new Brioi channel, settings, validation, generation, and error text through frontend i18n and synchronize every supported locale.
- [x] 11.2 Audit Brioi request/task logging and API responses to verify API keys, signed input URLs, base64 payloads, and upstream result URLs are never exposed.
- [x] 11.3 Run targeted Go tests for settings, routing, distribution, R2 staging, the Brioi adaptor, polling, billing, and storage lifecycle.
- [x] 11.4 Run the full relevant Go test/build suite and verify all database-facing changes remain SQLite, MySQL, and PostgreSQL compatible.
- [x] 11.5 Run frontend unit tests, lint/format checks, i18n checks, TypeScript checks, and the production build with Bun.
- [ ] 11.6 Perform a manual Docker smoke test with separate SilkRoad and Brioi groups sharing the same public model name, including Brioi model-list validation, text generation, image staging, polling, R2 delivery, and group-isolated routing.
- [x] 11.7 Confirm `relaykit/` remains independently buildable with `GOWORK=off go build ./...` if any relaykit file or public API changes.

## 12. Post-Audit Reliability Hardening

- [x] 12.1 Centralize generic Video Generation enablement across status/config endpoints, refresh frontend status caches after relevant saves, and test Brioi-only/SilkRoad-only/global-off matrices.
- [x] 12.2 Refactor task adaptors to return parsed public responses without writing HTTP output; accept all successful 2xx responses, close all bodies, sanitize bounded errors, and test 201/202/non-success behavior.
- [x] 12.3 Persist a recoverable pre-submission task before the upstream create call, durably record provider acceptance before settlement/client success, and add database-failure regression tests for every boundary.
- [x] 12.4 Enforce the usable-result-URL invariant in the provider-neutral polling layer and add SilkRoad/Brioi missing-result administrator-review tests.
- [x] 12.5 Make `settling` watchdog-recoverable with idempotent billing/storage exposure and add forced CAS/database-failure plus concurrent-worker tests.
- [x] 12.6 Replace bulk terminal failure on channel lookup errors with transient retry or definitive provider review and cover cache miss, database failure, and removed-channel cases.
- [x] 12.7 Add atomic provider-level settings save operations for SilkRoad and Brioi, invalidate public config/status once, and prevent partial saves or duplicate toasts.
- [x] 12.8 Split disabled and recoverable Video Generation error states, provide retry actions, and prevent silent model replacement when modes change.
- [x] 12.9 Associate labels and single-selection semantics with generation controls, generate bounded image thumbnails, and add keyboard/resource lifecycle regressions.
- [x] 12.10 Decompose `VideoToolPage` into focused bootstrap, form-state, polling, media-lifecycle, and presentation units without changing request or billing behavior.
- [x] 12.11 Add lifecycle diagnostics for provider acceptance, task persistence, provider success, settlement, storage readiness, and content delivery; verify the stage-to-stage failure signals.
- [ ] 12.12 Re-run full backend/frontend/relaykit verification and complete the Docker fault/same-model smoke matrix before release.
