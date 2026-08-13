## ADDED Requirements

### Requirement: Video Configuration separates generic and provider settings
The system SHALL present General, Storage, SilkRoad, and Brioi as distinct sections under Video Configuration. Generic sections SHALL NOT contain provider request mappings, and provider sections SHALL NOT contain shared storage credentials.

#### Scenario: Administrator opens the Brioi section
- **WHEN** the administrator navigates to Video Configuration > Brioi
- **THEN** the page shows only Brioi provider groups and Brioi model capability profiles

#### Scenario: Administrator opens the SilkRoad section
- **WHEN** the administrator navigates to Video Configuration > SilkRoad
- **THEN** the page shows only SilkRoad provider groups and SilkRoad model capability profiles

#### Scenario: Administrator opens the Storage section
- **WHEN** the administrator navigates to Video Configuration > Storage
- **THEN** the page manages the shared result and input storage configuration without provider profiles

### Requirement: Brioi capabilities are configurable within hard protocol bounds
The Brioi configuration SHALL provide exact profiles for `seedance-2-0-fast`, `seedance-2-0`, and `seedance-2-5`. Administrators MAY disable supported values, but the system MUST reject values outside the provider's documented hard bounds.

#### Scenario: Administrator disables a documented option
- **WHEN** an administrator disables a supported Brioi duration, resolution, aspect ratio, or generation mode
- **THEN** the saved public capability configuration omits that option

#### Scenario: Administrator adds an unsupported option
- **WHEN** a Brioi configuration contains a duration, resolution, aspect ratio, media role, or item limit beyond the model's hard contract
- **THEN** the system rejects the Brioi configuration with a field-specific validation error

#### Scenario: Unknown upstream model is selected
- **WHEN** a model-mapped upstream name matches no enabled Brioi profile
- **THEN** the system rejects generation instead of applying a default Brioi profile

### Requirement: Public video tool configuration is provider specific and sanitized
The video tool configuration endpoint SHALL return sanitized provider configurations and group ownership needed by authenticated users. It MUST exclude channel API keys, R2 credentials, upstream result URLs, and administrator-only metadata.

#### Scenario: User selects a Brioi-owned API key
- **WHEN** the selected key group resolves to Brioi
- **THEN** the Video Generation page uses only the public Brioi profiles and generation capabilities

#### Scenario: User selects a SilkRoad-owned API key
- **WHEN** the selected key group resolves to SilkRoad
- **THEN** the Video Generation page uses only the public SilkRoad profiles and generation capabilities

#### Scenario: User inspects the public configuration response
- **WHEN** the provider configuration is returned to a non-administrator user
- **THEN** no provider secret or storage credential is present

### Requirement: Video Generation renders every provider-specific request option
The Video Generation page SHALL derive duration, resolution, aspect ratio, generation mode, and image limits from the active provider profile. It MUST reset an option that becomes invalid when the selected key, provider, or model changes.

#### Scenario: User selects a Brioi Seedance 2.0 model
- **WHEN** the active Brioi profile supports multiple resolutions
- **THEN** the page presents the enabled resolution selector and includes the selected resolution in the friendly request

#### Scenario: User switches from Seedance 2.0 to Seedance 2.5
- **WHEN** the previously selected resolution or aspect ratio is not enabled for Seedance 2.5
- **THEN** the page replaces it with a valid Seedance 2.5 option before submission

#### Scenario: User switches to a provider profile without a resolution selector
- **WHEN** the newly active provider encodes resolution differently or does not expose that option
- **THEN** the page removes the stale Brioi resolution from its request

### Requirement: Provider settings saves validate only relevant cross-provider invariants
Saving a provider section SHALL validate that provider's fields and the necessary unique group-ownership invariant. It MUST NOT fail because an unrelated provider or generic settings section is incomplete.

#### Scenario: Brioi settings are valid and unrelated storage is incomplete
- **WHEN** an administrator saves a valid Brioi provider profile while an unrelated generic storage section is not ready for activation
- **THEN** the provider settings save is evaluated independently of storage readiness

#### Scenario: Brioi group conflicts with SilkRoad
- **WHEN** an administrator saves Brioi settings containing a SilkRoad-owned group
- **THEN** the save fails with the specific group ownership conflict

### Requirement: Video Generation pricing estimate honors billing mode
The Video Generation page SHALL calculate its estimate from the selected model's configured price, billing mode, duration, and group ratio. It MUST NOT multiply fixed-price models by duration.

#### Scenario: Fixed-price model is selected
- **WHEN** the pricing record marks the selected model as fixed per request
- **THEN** the displayed estimate equals model price multiplied by group ratio

#### Scenario: Per-second model is selected
- **WHEN** the pricing record marks the selected model as per-second
- **THEN** the displayed estimate equals model price multiplied by validated duration and group ratio

#### Scenario: Model has no valid price
- **WHEN** the selected provider model lacks a positive supported pricing configuration
- **THEN** the Video Generation page excludes the model or clearly reports that no priced video model is available

### Requirement: Generic Video Generation visibility includes every enabled provider
The system SHALL derive the public Video Generation enabled state from global video enablement and all registered video provider configurations. `/api/status` and `/api/video/tool-config` MUST use the same computation, and frontend status caches MUST be refreshed after relevant settings change.

#### Scenario: Only Brioi is enabled
- **WHEN** global video generation and Brioi are enabled while SilkRoad is disabled
- **THEN** the sidebar and authenticated Video Generation route remain available

#### Scenario: No provider is enabled
- **WHEN** global video generation is disabled or every video provider is disabled
- **THEN** the generic Video Generation entry is unavailable

### Requirement: Provider settings save atomically
The system SHALL validate and persist one provider's groups and profiles as a single operation. A failed provider save MUST NOT expose a partial revision, and the frontend SHALL emit at most one success or failure notification for that save.

#### Scenario: One field in a provider configuration is invalid
- **WHEN** an administrator submits groups and profiles together and validation fails
- **THEN** neither field is changed

#### Scenario: Provider configuration succeeds
- **WHEN** all provider fields and cross-provider group ownership are valid
- **THEN** the complete revision is committed and public configuration/status caches are refreshed once

### Requirement: Video Generation exposes distinct recoverable UI states
The Video Generation page SHALL distinguish administrator-disabled state from failures loading configuration, API keys, token groups, or models. Recoverable loading failures SHALL provide a retry action.

#### Scenario: Configuration request fails
- **WHEN** the public video configuration endpoint is temporarily unavailable
- **THEN** the page reports a loading failure rather than claiming the feature is disabled

#### Scenario: API key request fails
- **WHEN** the user's API key list cannot be loaded
- **THEN** the page reports that request failure rather than claiming no eligible keys exist

### Requirement: Dependent selections and media controls remain explicit and accessible
The Video Generation page SHALL associate labels with controls, expose generation modes as a single-selection group, and update dependent model/mode/options without silently changing a valid user choice. Multi-image previews SHALL use bounded preview resources.

#### Scenario: Selected mode invalidates the current model
- **WHEN** the user changes to a mode unsupported by the selected model
- **THEN** the page clears the model or explicitly announces the replacement before submission

#### Scenario: User operates generation controls with a keyboard
- **WHEN** focus moves through key, model, mode, duration, resolution, ratio, and media controls
- **THEN** every control has an accessible name and selection state and can be operated without a pointer

#### Scenario: User selects many high-resolution images
- **WHEN** a profile permits up to thirty references
- **THEN** the page uses bounded thumbnails, releases superseded preview resources, and avoids rendering unnecessary empty image elements
