## Why

The video foundation currently supports SilkRoad but cannot route two providers that expose the same public model names without applying the wrong provider capabilities. Brioi must be added as a standalone video-only channel, with user-uploaded images staged through R2 because its upstream rejects base64 media.

## What Changes

- Add a dedicated Brioi video-only channel type and task adaptor; Brioi will not share protocol branches or settings with NewAPI or SilkRoad.
- Support Brioi Seedance 2.0, Seedance 2.0 Fast, and Seedance 2.5 from the Video Generation page through the existing `/v1/video/generations` task flow.
- Stage every image uploaded through the Video Generation page to R2 before submitting its signed HTTPS URL to Brioi.
- Add a provider-specific Brioi configuration page under Video Configuration, separate from the SilkRoad page.
- Assign each Video Generation group to exactly one video provider type, then apply that provider constraint to model discovery, capability selection, and task distribution.
- Keep model pricing and fixed/per-second billing mode in the existing global model pricing system; correct the Video Generation estimate to honor the selected model's billing mode.
- Reuse the provider-neutral polling, billing, result storage, URL redaction, content delivery, and administrator recovery lifecycle.
- Defer public client support for `/v1/videos`, Brioi `/v1/realperson` assets, and video/audio reference media.

## Capabilities

### New Capabilities

- `brioi-video-channel`: Dedicated Brioi channel registration, Seedance request conversion, R2 image staging, polling, and result normalization.
- `video-provider-group-routing`: Unique provider ownership for Video Generation groups and provider-constrained model discovery and task distribution.
- `provider-specific-video-configuration`: Separate sanitized SilkRoad and Brioi capability configurations selected by the active API key's group.

### Modified Capabilities

None. There are no synchronized main OpenSpec capabilities yet; the existing generic video foundation remains the implementation base.

## Impact

- Backend channel constants, endpoint eligibility, channel validation/testing, task adaptor registration, provider settings, video tool configuration, model discovery, distribution, billing estimates, and R2 input staging.
- Frontend Video Configuration navigation and forms, Video Generation key/model/capability resolution, request construction, price estimates, i18n, and tests.
- No new external dependency is required. The existing R2 storage and asynchronous video task infrastructure is reused.
