# Generic Video Foundation Rollout

## Before enabling video generation

1. Select one ingest instance and give it a stable `NODE_NAME`.
2. In **System Settings → Video Configuration → Storage**, configure:
   - driver: `local`
   - a persistent `local_dir`
   - `ingest_node_name` equal to that instance's `NODE_NAME`
   - the externally reachable application origin as `public_download_base_url`
   - the desired automatic transfer retry count
3. Mount the configured local directory on persistent storage. Do not run two
   ingest instances against separate filesystems with the same node name.
4. Configure allowed API-key groups and keep video generation disabled until
   storage validation succeeds.

Video retention is fixed at seven days. This applies to users and
administrators and cannot be extended by configuration.

## SilkRoad channel cutover

1. Create a new channel manually with type **SilkRoad**. Configure its base URL,
   key, models, groups, and model mapping as needed.
2. Configure SilkRoad common capabilities, profiles, exact/prefix matches, and
   the default fallback profile under **Video Configuration → SilkRoad**.
3. Enable generic video generation, then submit a low-cost test through both
   `/v1/video/generations` and `/v1/videos`.
4. Verify the task remains processing at 99% while storing, then confirm that
   `/v1/videos/{task_id}/content` serves the local file and no public response
   contains the provider URL.
5. Exercise administrator diagnostics, provider confirmation, storage retry,
   and an idempotent full refund on a controlled failed task.

## Compatibility and rollback

- Do not migrate existing channel records automatically.
- Keep the old NewAPI channel enabled for polling unfinished tasks, but stop
  routing new video submissions to it after the dedicated SilkRoad channel is
  verified.
- Remove the old channel only after all of its unfinished tasks have reached a
  terminal state and their seven-day stored-video retention window has passed.
- The legacy SilkRoad settings keys remain read-compatible during rollout.
  Saving the new generic settings makes `video_setting.*` authoritative.
- To stop new work, disable video generation. Keep the ingest instance and old
  channel available until pending storage and cleanup work has drained.

Storage failures after successful provider generation do not trigger automatic
refunds. Users are directed to contact an administrator; administrators should
confirm the provider result, retry storage when possible, and issue the single
idempotent full refund only after confirming delivery failure.
