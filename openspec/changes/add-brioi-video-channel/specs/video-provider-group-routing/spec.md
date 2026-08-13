## ADDED Requirements

### Requirement: Each Video Generation group has one provider owner
The system SHALL assign every group exposed in the Video Generation tool to exactly one video provider type. A group MUST NOT be simultaneously assigned to SilkRoad and Brioi.

#### Scenario: Administrator assigns an unused group to Brioi
- **WHEN** a Brioi provider configuration includes a group that is not owned by another video provider
- **THEN** the system saves the Brioi group assignment

#### Scenario: Administrator assigns an overlapping group
- **WHEN** a provider configuration includes a group already assigned to another video provider type
- **THEN** the system rejects only that provider configuration update with a message identifying the conflicting group and provider

#### Scenario: Group has no provider owner
- **WHEN** an API key belongs to a group not assigned to any video provider
- **THEN** the Video Generation page does not offer that key for video generation

### Requirement: Provider resolution is based on the selected API key group
The system SHALL resolve the Video Generation provider from the selected API key's effective token group and MUST NOT infer the provider from model name alone.

#### Scenario: SilkRoad and Brioi expose the same model name
- **WHEN** two keys belong to provider-owned groups that both expose the same public model ID
- **THEN** each key resolves to its own provider type and provider capability configuration

#### Scenario: Client supplies a model name associated with another provider
- **WHEN** the selected key group resolves to Brioi but the client attempts to force SilkRoad provider behavior for an identically named model
- **THEN** the server continues using the Brioi provider constraint or rejects the request

### Requirement: Video model discovery is provider constrained
The Video Generation model query SHALL return only models eligible for the selected key's group, resolved provider channel type, video endpoint, token model limits, and billing configuration.

#### Scenario: Selected group contains unrelated channel types
- **WHEN** a token group contains Brioi and non-video channels
- **THEN** the video model query returns only models served by eligible Brioi video channels

#### Scenario: Selected group has no eligible provider channel
- **WHEN** the group is assigned to Brioi but has no enabled Brioi channel serving a configured video model
- **THEN** the video model query returns no models and the page reports that no eligible video models are available

#### Scenario: Token model limits exclude a model
- **WHEN** the selected API key restricts access to a subset of models
- **THEN** the video model query excludes every model outside that token limit

### Requirement: Task distribution repeats the provider constraint
The backend SHALL constrain Video Generation task distribution to the provider type resolved from the authenticated token group, even if another video provider in the group exposes the same model.

#### Scenario: Identically named model exists on multiple provider types
- **WHEN** a Brioi-owned group submits a model name also available on a SilkRoad channel
- **THEN** only eligible Brioi channels can receive the task

#### Scenario: No matching provider channel is available
- **WHEN** no enabled channel matches the resolved provider type, group, model, and video endpoint
- **THEN** the system returns a no-eligible-channel error without routing to another provider type

### Requirement: Existing SilkRoad groups remain compatible
The system SHALL preserve current SilkRoad Video Generation group visibility as SilkRoad-owned during migration unless an administrator explicitly changes the provider-specific group configuration.

#### Scenario: Existing deployment upgrades without Brioi groups
- **WHEN** an installation has existing Video Generation groups and no explicit Brioi group assignment
- **THEN** those groups continue resolving to SilkRoad with their existing behavior
