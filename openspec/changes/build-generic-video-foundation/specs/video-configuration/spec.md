## ADDED Requirements

### Requirement: Dedicated Video Configuration area
System Settings SHALL include a top-level Video Configuration area separate from Extensions. It SHALL provide generic video settings and provider-specific child sections.

#### Scenario: Administrator opens video settings
- **WHEN** a super administrator enters System Settings
- **THEN** the navigation includes Video Configuration with General, Storage, and SilkRoad child sections

#### Scenario: Generic setting ownership
- **WHEN** a setting controls behavior shared by all video providers
- **THEN** it is configured in a generic Video Configuration section rather than a provider section

#### Scenario: Provider setting ownership
- **WHEN** a setting affects only SilkRoad protocol or model behavior
- **THEN** it is configured in the SilkRoad child section rather than General or the individual channel instance

### Requirement: Separation of channel instance and channel-type configuration
Individual channel instances SHALL contain only connection and routing data such as base URL, API key, proxy, model list, and model mapping. Provider protocol profiles and defaults SHALL be shared by all instances of that channel type.

#### Scenario: Two SilkRoad channel instances
- **WHEN** two SilkRoad channels are configured with different keys or base URLs
- **THEN** both use the same SilkRoad channel-type capability and profile configuration

#### Scenario: Add a future video channel type
- **WHEN** a future provider requires provider-specific configuration
- **THEN** it adds one provider adapter and one provider configuration child section without adding provider fields to generic settings

### Requirement: SilkRoad common configuration and profile overrides
The SilkRoad section SHALL represent shared provider values once and SHALL allow optional model profiles to override only their differences. It SHALL require an administrator-selected default profile.

#### Scenario: Edit common values
- **WHEN** an administrator changes an allowed common option within SilkRoad hard capability limits
- **THEN** every SilkRoad profile that does not override that option inherits the change

#### Scenario: Edit a profile override
- **WHEN** an administrator configures a duration field or allowed values specific to one SilkRoad model family
- **THEN** only that matching profile uses the override

#### Scenario: Select the default profile
- **WHEN** an administrator selects a different valid SilkRoad profile as default
- **THEN** unmatched SilkRoad models use that profile after the configuration is saved

#### Scenario: Delete the selected default
- **WHEN** an administrator attempts to delete or disable the selected default profile without selecting another
- **THEN** the system rejects the configuration

### Requirement: Backward-compatible settings transition
Existing valid `silkroad_setting` profile, storage, and video-tool group values SHALL remain effective after upgrading. Generic values SHALL transition to provider-neutral video settings without requiring channel or task record migration.

#### Scenario: Upgrade an installation with SilkRoad settings
- **WHEN** the application first loads the new configuration model
- **THEN** it preserves existing profile choices, maps storage and video-tool group values to generic video settings, and selects a deterministic valid default profile

#### Scenario: Read during transition
- **WHEN** legacy setting keys still exist during a rolling deployment
- **THEN** the system uses a defined compatibility precedence and does not silently reset configured storage or profiles

### Requirement: Generic Video Generation extension
The user-facing Seedance extension SHALL be renamed Video Generation and SHALL use the route `/extensions/video`. The old `/extensions/seedance` route SHALL redirect to the new route.

#### Scenario: Open the new extension
- **WHEN** an authorized user opens `/extensions/video`
- **THEN** the generic Video Generation tool is displayed

#### Scenario: Open the legacy extension route
- **WHEN** a user opens `/extensions/seedance`
- **THEN** the router redirects to `/extensions/video` without losing authentication state

### Requirement: Server-driven video form capabilities
The Video Generation frontend SHALL render model options and validation from server-provided capability data rather than a duplicated hardcoded list of generation modes. The server response SHALL contain no storage secrets or provider credentials.

#### Scenario: Render SilkRoad capabilities
- **WHEN** the selected model resolves to a SilkRoad profile
- **THEN** the form displays only the supported generation modes, durations, aspect ratios, and media limits returned by the server

#### Scenario: Capability configuration changes
- **WHEN** an administrator disables an allowed provider option
- **THEN** the frontend reflects the change after refreshing configuration without a frontend code change

#### Scenario: Malicious client bypasses the form
- **WHEN** a client submits a value hidden or disabled by the frontend
- **THEN** backend provider validation still rejects it

### Requirement: Fixed retention presentation
The Storage section SHALL display the mandatory seven-day retention rule but SHALL NOT provide a control that extends or disables retention. It SHALL allow configuration of local driver path, ingest node, public base address, and retry limit.

#### Scenario: Administrator views retention
- **WHEN** an administrator opens Video Configuration Storage
- **THEN** the interface states that all videos expire seven days after storage readiness and provides no retention-duration input

### Requirement: Internationalized video configuration and extension
All newly introduced Video Configuration, SilkRoad, storage, recovery, status, and Video Generation user-facing text SHALL use the project's frontend internationalization system and SHALL be translated for every supported locale.

#### Scenario: Display in a supported language
- **WHEN** a user changes the application language
- **THEN** video settings, tool labels, statuses, validation, and administrative actions render using the selected locale
