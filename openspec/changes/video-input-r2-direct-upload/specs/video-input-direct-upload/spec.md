## ADDED Requirements

### Requirement: Presigned PUT for video input assets

The system SHALL allow an authenticated user to obtain a short-lived R2 (S3-compatible) Presigned PUT URL for one reference media object without exposing R2 account credentials to the client.

#### Scenario: Successful presign for an image

- **WHEN** an authenticated user requests a presign with a supported image `content_type` and `size` within configured upload limits
- **THEN** the system returns an `asset_id`, a Presigned PUT `upload_url`, any required `upload_headers`, and an expiry time for the upload grant

#### Scenario: Reject oversize or unsupported type on presign

- **WHEN** the declared `size` exceeds the configured limit for that media kind, or `content_type` is not allowed
- **THEN** the system rejects the presign with a client error and does not create an upload grant

#### Scenario: Credentials are not returned

- **WHEN** a client receives a successful presign response
- **THEN** the response MUST NOT include R2 `access_key`, `secret_access_key`, `api_token`, or account management credentials

### Requirement: Complete upload and return short-lived GET URL

After the client uploads bytes to the Presigned PUT URL, the system SHALL verify the object and return a short-lived R2 Presigned GET URL suitable for use as `media[].source` on video generation.

#### Scenario: Successful complete

- **WHEN** the owning user completes an asset whose object exists and matches declared size and allowed type checks
- **THEN** the system returns an HTTPS Presigned GET `url` and `expires_at` aligned with the configured input GET TTL

#### Scenario: Complete fails validation

- **WHEN** complete is called but the object is missing, oversized, or fails type validation
- **THEN** the system rejects complete, and SHOULD delete the invalid object when feasible

### Requirement: User-scoped upload anti-abuse limits

Upload-related APIs SHALL enforce rate and concurrency limits keyed by `user_id` (not by API token id).

#### Scenario: Presign rate limit

- **WHEN** a user exceeds the configured presign rate (default 20 requests per minute)
- **THEN** the system rejects further presign requests with HTTP 429 until the window allows more

#### Scenario: Pending upload cap

- **WHEN** a user already has the maximum number of incomplete (`presigned`) assets (default 5)
- **THEN** the system rejects new presign requests until some assets are completed, deleted, or expire

#### Scenario: R2 soft limit blocks uploads

- **WHEN** video R2 storage uploads are soft-blocked by the existing free-tier guard
- **THEN** the system rejects new presign requests with a clear client error

### Requirement: Video tool uploads before generation

The Video Generation tool SHALL upload selected reference media via the direct-upload flow before submitting generation, and SHALL place only HTTPS URLs into the generation request body for those assets.

#### Scenario: Upload on select with progress

- **WHEN** the user selects a valid reference file in the video tool
- **THEN** the client starts upload promptly and shows upload progress until the asset is ready or failed

#### Scenario: Submit uses HTTPS sources

- **WHEN** the user submits generation with ready uploaded assets
- **THEN** the generation request `media[].source` values for those assets are HTTPS URLs (not `data:` URLs)

#### Scenario: Expired asset before submit

- **WHEN** an uploaded asset’s GET URL is past `expires_at` at submit time
- **THEN** the client prevents submit (or fails fast) and prompts the user to re-select and upload again, without requesting automatic URL renewal

### Requirement: Generation remains compatible with inline and HTTPS media

`/v1/video/generations` SHALL continue to accept existing media encodings so API clients are not broken.

#### Scenario: data URL still accepted

- **WHEN** a client submits generation with a `data:` media source that passes existing validation
- **THEN** the system processes the request using the existing staging path for channels that require R2 delivery

#### Scenario: HTTPS URL pass-through

- **WHEN** a client submits generation with an `https://` media source
- **THEN** adaptors that stage inputs MUST pass the URL through without re-uploading the object bytes as a data URL
