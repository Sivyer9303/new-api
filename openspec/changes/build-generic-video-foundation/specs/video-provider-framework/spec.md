## ADDED Requirements

### Requirement: Provider-neutral video task lifecycle
The system SHALL process video generation through a provider-neutral lifecycle that owns request normalization, channel selection, task persistence, polling orchestration, billing hooks, and public task identifiers. Adding a provider SHALL NOT require changes to the lifecycle orchestration or task persistence model.

#### Scenario: Submit through a registered provider
- **WHEN** a valid video request selects a registered video-only channel
- **THEN** the system normalizes the request, invokes that provider's adapter, stores the upstream task identifier privately, and returns a public task identifier

#### Scenario: Add a future provider
- **WHEN** a developer registers a new video channel type, provider adapter, capabilities, and provider configuration
- **THEN** the provider can use the existing submission, polling, billing, storage, and client response flow without provider-specific branches in the task core

### Requirement: Dual public video API compatibility
The system SHALL retain both `/v1/video/generations` and `/v1/videos` submission and query APIs. Both APIs SHALL enter the same internal video task flow while retaining their documented request parsing and response representation.

#### Scenario: Legacy video API request
- **WHEN** a client submits a supported request to `/v1/video/generations`
- **THEN** the system converts the friendly JSON request into the normalized video request and returns legacy-compatible task responses

#### Scenario: OpenAI-style video API request
- **WHEN** a client submits a supported request to `/v1/videos`
- **THEN** the system converts the request into the normalized video request and returns a flat OpenAI-style video task response

#### Scenario: Query the same task from its API family
- **WHEN** a client queries a public task identifier through the corresponding video query endpoint
- **THEN** the system reads the same persisted task and renders the route-specific response without reselecting a channel

### Requirement: Dedicated SilkRoad video channel
The system SHALL provide a dedicated SilkRoad channel type that supports video endpoints only. It SHALL use a SilkRoad task adapter and SHALL NOT advertise or handle chat, image-generation, embedding, audio, or text-model endpoints.

#### Scenario: Create a SilkRoad channel
- **WHEN** an administrator creates a SilkRoad channel instance
- **THEN** the administrator configures its base URL, API key, models, model mappings, proxy, and connection settings through the normal channel interface

#### Scenario: Reject a non-video endpoint
- **WHEN** routing considers a SilkRoad channel for a non-video request
- **THEN** the channel is ineligible for that request

### Requirement: Provider-owned protocol conversion
Each video provider adapter SHALL define its upstream submit and poll requests, authentication, response parsing, status mapping, result extraction, and billing estimation within adapter hard limits. Provider-specific protocol fields SHALL NOT leak into the provider-neutral task core.

#### Scenario: SilkRoad request conversion
- **WHEN** a normalized request is submitted through a SilkRoad channel
- **THEN** the SilkRoad adapter converts it to the SilkRoad `/v1/video/generations` contract and validates all provider-specific constraints before sending it

#### Scenario: Unknown upstream status
- **WHEN** a provider returns a status that its adapter does not recognize
- **THEN** the adapter marks the task for administrator review, prevents automatic refund, and preserves diagnostic details privately

### Requirement: Capability and profile resolution
Each provider SHALL publish hard capability limits in code and MAY define common configurable values plus model profiles that only narrow or specialize those limits. Profile resolution SHALL use exact model match, then longest matching prefix, then the administrator-selected default profile.

#### Scenario: Exact model profile
- **WHEN** a model exactly matches a configured profile
- **THEN** the system applies that profile and inherits any omitted values from the provider common configuration

#### Scenario: Prefix model profile
- **WHEN** no exact profile matches and multiple prefixes match the model
- **THEN** the system applies the profile with the longest matching prefix

#### Scenario: Default profile fallback
- **WHEN** neither an exact model nor a prefix matches
- **THEN** the system applies the administrator-selected default profile and records the fallback in administrator diagnostics

#### Scenario: Configuration exceeds provider limits
- **WHEN** an administrator configuration or client request attempts to enable a value outside the provider adapter's hard limits
- **THEN** the system rejects the configuration or request rather than sending an unsupported upstream request

### Requirement: Billing safety across providers
The video task foundation SHALL invoke provider billing hooks before pre-consume and at terminal settlement, SHALL bound every user-controlled billing multiplier, and SHALL use the centralized checked quota conversion helpers.

#### Scenario: Per-second SilkRoad request
- **WHEN** a valid SilkRoad profile uses per-second billing
- **THEN** the validated duration contributes a bounded `seconds` multiplier before quota pre-consume

#### Scenario: Invalid billing multiplier
- **WHEN** a request supplies an out-of-range, non-finite, or otherwise invalid multiplier
- **THEN** the system rejects the request before quota calculation or upstream submission

### Requirement: Legacy NewAPI task compatibility without data migration
The system SHALL NOT automatically migrate existing NewAPI channels or task records. It SHALL retain enough legacy task-adapter behavior to query and finish existing SilkRoad video tasks stored with the NewAPI platform while all newly configured SilkRoad video traffic uses the dedicated SilkRoad channel type.

#### Scenario: Poll an unfinished legacy task
- **WHEN** an unfinished historical SilkRoad task has the NewAPI platform and its original channel still exists
- **THEN** the polling service continues querying it with the legacy SilkRoad protocol until it reaches a terminal state

#### Scenario: Submit through a new SilkRoad channel
- **WHEN** an operator manually creates and enables the dedicated SilkRoad channel
- **THEN** new matching video tasks persist the SilkRoad platform and do not depend on the NewAPI task adapter
