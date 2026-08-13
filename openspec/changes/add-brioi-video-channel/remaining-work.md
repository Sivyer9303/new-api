# Brioi Integration Remaining Work

## Handoff State

- Branch: `feat/generic-video-foundation`
- Baseline commit: `b19b1bb05`
- Core package verification completed before handoff:
  `go test ./model ./service ./relay ./controller`
- `docker-compose.local.yml` remains local and untracked.
- Docker/live-provider verification has not been run in the final hardening state.

## P0: Finish Submission Durability

### 1. Publish provider acceptance only after it is durable

`RelayTaskSubmit` currently sets `TaskSubmitResult.ProviderAccepted` before
`persistProviderAcceptance` commits the upstream task ID. If that database
update fails, the controller suppresses refund and failure finalization even
though the durable row has no provider handle.

Required change:

1. Keep the response acceptance state uncertain while parsing and adjusting
   billing.
2. Set `ProviderAccepted=true` only after the acceptance CAS succeeds.
3. If persistence fails after a valid upstream ID was returned, withhold
   automatic refund and persist a recoverable/provider-review marker without
   claiming that acceptance is durable.
4. Add regression tests for database error and CAS-loss boundaries.

Acceptance criteria:

- A failed acceptance write never returns client success.
- A failed acceptance write never automatically refunds.
- `ProviderAccepted` implies the database contains the upstream task ID.
- No retry can create a second upstream task after a valid ID was received.

### 2. Keep accepted tasks pollable after local submission failure

After provider acceptance is persisted, `SettleBilling` or the
`SUBMITTING -> SUBMITTED` update can still fail. The row then remains
`SUBMITTING`, while polling currently skips every `SUBMITTING` row.

Required change:

1. Introduce a recoverable accepted-submission transition, or allow polling of
   `SUBMITTING` rows only when `UpstreamTaskID` is non-empty.
2. Preserve the no-automatic-refund policy for these rows.
3. Ensure polling can still reach provider success, idempotent settlement,
   storage, and content delivery after a process restart.
4. Add tests for settlement failure, submission-ready CAS failure, and restart
   recovery.

Acceptance criteria:

- A durable upstream ID is never left permanently unpolled.
- A local post-accept failure cannot trigger a duplicate create request.
- Recovery settles billing at most once and exposes storage only afterward.

## P1: Re-audit Reservation and Refund Recovery

The final session added:

- synchronous database quota writes for task reservations even when batch
  updates are enabled;
- an explicit reservation-uncertainty error for failed compensation;
- durable `refund_pending`, retry time, and retry-attempt columns;
- atomic persisted-task refunds; and
- exponential refund retry scheduling that prevents poison rows from starving
  later work.

These changes passed the core package tests but still need focused verification:

1. Run with `common.BatchUpdateEnabled=true` and prove task debits are written
   synchronously rather than queued only in memory.
2. Inject token-compensation and funding-compensation failures and verify the
   task enters provider review without automatic refund.
3. Verify concurrent refund workers apply one refund and clear the pending
   marker once.
4. Verify permanently failing refunds are delayed and later task IDs continue
   to receive attempts.
5. Verify `Task` auto-migration for the new refund columns on SQLite, MySQL,
   and PostgreSQL.

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
