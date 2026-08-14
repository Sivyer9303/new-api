# Brioi Integration Remaining Work

## Handoff State

- Branch: `feat/generic-video-foundation`
- P0 submission durability is implemented on top of `b19b1bb05`.
- Latest core package verification:
  `go test ./model ./service ./relay ./controller`
- `docker-compose.local.yml` remains local and untracked.
- Docker/live-provider verification has not been run in the final hardening state.

## P0: Submission Durability (done)

### 1. Publish provider acceptance only after it is durable

`ProviderAccepted` is now set only after `persistProviderAcceptance` wins its
CAS. A failed write stays `AcceptanceUncertain`, withholds automatic refund,
and best-effort records the upstream ID through a narrower
`UpdatePrivateDataIfStatus` so polling can recover the row.

Retry/create is skipped when a persisted `UpstreamTaskID` already exists.

Covered by:

- `TestRecordProviderAcceptancePublishesFlagOnlyAfterDurableWrite`
- `TestRecordProviderAcceptanceRecoversUpstreamIDAfterFullCASFailure`
- `TestRecordProviderAcceptanceDoesNotClaimDurableIDOnCASLoss`
- `TestPersistProviderAcceptanceLeavesSubmittingRecordRecoverableOnDatabaseFailure`
- `TestPersistProviderAcceptanceCASLossLeavesExistingRowUntouched`
- `TestPersistSubmittingTaskKeepsAcceptedUpstreamID`

### 2. Keep accepted tasks pollable after local submission failure

Polling skips `SUBMITTING` only when `UpstreamTaskID` is empty. Accepted rows
that remain `SUBMITTING` after settle or `SUBMITTED` CAS failure stay pollable
and keep `NoAutomaticRefund`. Finalize/refund is skipped when
`ProviderAccepted` or `HasDurableUpstreamID` is set.

Covered by:

- `TestRunTaskPollingSkipsSubmittingTasksWithoutUpstreamID`
- `TestRunTaskPollingPollsSubmittingTasksWithUpstreamID`
- `TestRunTaskPollingRecoversAcceptedSubmittingTaskIntoStorage`
- `TestFinalizeSubmittingTaskFailureLeavesAcceptedTasksPollable`

## P1: Re-audit Reservation and Refund Recovery

Covered in this session:

1. `TestDecreaseTokenQuotaDirectWritesWhenBatchUpdatesAreEnabled` proves
   `DecreaseTokenQuotaDirect` writes through `BatchUpdateEnabled=true`.
2. `TestReserveReturnsUncertainWhenFundingCompensationFails` and
   `TestEnsureTaskQuotaReservedPropagatesReservationUncertainty` cover
   compensation failure → `ErrBillingReservationUncertain` without retry.
3. `TestRefundTaskQuotaIsAtomicAndIdempotent` covers concurrent workers.
4. `TestRecoverPendingTaskRefundRotatesFailedRows` covers poison-row delay.
5. `TestTaskAutoMigrateAddsRefundRecoveryColumns` covers SQLite AutoMigrate.

Still needed before release:

- MySQL and PostgreSQL AutoMigrate of `refund_pending` / `refund_retry_at` /
  `refund_attempts`.
- A request-level test that a compensation failure after a persisted
  `SUBMITTING` row enters provider review without automatic refund.

## P1: Complete Automated Verification

Run from the repository root:

```text
git diff --check
go test ./...
go test -race ./model ./service ./relay
openspec validate add-brioi-video-channel --strict
```

Run from `web/`:

```text
bun run build:check
bun test src/features/extensions/video-tool src/features/system-settings/video src/features/channels/lib
bunx oxlint -c .oxlintrc.json \
  src/features/extensions/video-tool \
  src/features/system-settings/video \
  src/features/system-settings/extensions/brioi-profile-editor.tsx \
  src/features/system-settings/extensions/brioi-profile-schemas.ts \
  src/features/system-settings/extensions/brioi-settings-section.tsx \
  src/features/system-settings/extensions/silkroad-settings-section.tsx
```

If `relaykit/` or its public API changes during the remaining fixes:

```text
cd relaykit
GOWORK=off go build ./...
```

## P1: Docker and Live Smoke Matrix

Do not mark the change release-ready until the following matrix passes:

1. SilkRoad-only group and Brioi-only group expose the same public model name
   without cross-provider routing.
2. Brioi model-list validation uses the configured base URL and selected key.
3. Brioi text-to-video succeeds end to end.
4. Brioi single/multi-image requests stage every image to R2 and send only
   signed HTTPS URLs upstream.
5. Strict first-frame and strict first/last-frame modes preserve image roles.
6. Polling reaches provider success, billing settlement, R2 result storage,
   and local content delivery.
7. Fixed-price and per-second models charge the configured amount exactly once.
8. Submission 502/504, acceptance-persistence failure, settlement CAS failure,
   missing channel, missing result URL, and failed refund all enter their
   expected recoverable or administrator-review states.

## Local File Decision

`docker-compose.local.yml` was intentionally excluded from the commit. Before
adding it, review it for credentials and decide whether it should remain local,
be added to an ignore rule, or be committed as a sanitized example.
