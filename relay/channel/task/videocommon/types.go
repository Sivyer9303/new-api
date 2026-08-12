package videocommon

// VideoMediaType identifies the kind of media attached to a video request.
type VideoMediaType string

const (
	VideoMediaImage VideoMediaType = "image"
	VideoMediaVideo VideoMediaType = "video"
	VideoMediaAudio VideoMediaType = "audio"
)

// VideoMediaRole describes how a provider should use an attached media item.
type VideoMediaRole string

const (
	VideoMediaRoleReference  VideoMediaRole = "reference"
	VideoMediaRoleFirstFrame VideoMediaRole = "first_frame"
	VideoMediaRoleLastFrame  VideoMediaRole = "last_frame"
)

// VideoMedia is a provider-neutral media attachment. Source stays opaque to
// the task framework; provider adapters decide which source forms they accept.
type VideoMedia struct {
	Type   VideoMediaType `json:"type"`
	Role   VideoMediaRole `json:"role,omitempty"`
	Source string         `json:"source"`
}

// VideoGenerateRequest is the normalized request shared by public video APIs.
// Pointer scalars preserve the difference between an absent value and zero.
type VideoGenerateRequest struct {
	Model          string       `json:"model"`
	Prompt         string       `json:"prompt"`
	GenerationType string       `json:"generation_type,omitempty"`
	Duration       *int         `json:"duration,omitempty"`
	Resolution     string       `json:"resolution,omitempty"`
	AspectRatio    string       `json:"aspect_ratio,omitempty"`
	Media          []VideoMedia `json:"media,omitempty"`
}

// ProviderTaskStatus is the normalized status returned by video providers.
type ProviderTaskStatus string

const (
	ProviderTaskSubmitted ProviderTaskStatus = "submitted"
	ProviderTaskQueued    ProviderTaskStatus = "queued"
	ProviderTaskRunning   ProviderTaskStatus = "running"
	ProviderTaskSucceeded ProviderTaskStatus = "succeeded"
	ProviderTaskFailed    ProviderTaskStatus = "failed"
)

// VideoProviderResult contains only provider-neutral polling information.
type VideoProviderResult struct {
	UpstreamTaskID string
	Status         ProviderTaskStatus
	Progress       int
	ResultURL      string
	FailureReason  string
	NoRefund       bool
	RawStatus      string
}

// Option is a provider capability value and its upstream protocol field.
type Option struct {
	Label       string `json:"label,omitempty"`
	Value       string `json:"value"`
	UpstreamKey string `json:"upstream_key,omitempty"`
	Enabled     bool   `json:"enabled,omitempty"`
	Sort        int    `json:"sort,omitempty"`
}

// MediaCapabilities bounds media accepted by a provider or profile.
type MediaCapabilities struct {
	MinItems      int              `json:"min_items,omitempty"`
	MaxItems      int              `json:"max_items,omitempty"`
	AcceptedTypes []VideoMediaType `json:"accepted_types,omitempty"`
	AllowedRoles  []VideoMediaRole `json:"allowed_roles,omitempty"`
	AllowAudio    bool             `json:"allow_audio,omitempty"`
}

// MediaCapabilityOverrides is sparse: nil fields inherit provider common
// values, while present fields intentionally replace them.
type MediaCapabilityOverrides struct {
	MinItems      *int             `json:"min_items,omitempty"`
	MaxItems      *int             `json:"max_items,omitempty"`
	AcceptedTypes []VideoMediaType `json:"accepted_types,omitempty"`
	AllowedRoles  []VideoMediaRole `json:"allowed_roles,omitempty"`
	AllowAudio    *bool            `json:"allow_audio,omitempty"`
}

// Capabilities is the resolved capability set used for request validation.
type Capabilities struct {
	GenerationTypes []string          `json:"generation_types,omitempty"`
	Durations       []Option          `json:"durations,omitempty"`
	AspectRatios    []Option          `json:"aspect_ratios,omitempty"`
	Media           MediaCapabilities `json:"media,omitempty"`
}

// CapabilityOverrides uses nil slices to mean inheritance from common values.
type CapabilityOverrides struct {
	GenerationTypes []string                  `json:"generation_types,omitempty"`
	Durations       []Option                  `json:"durations,omitempty"`
	AspectRatios    []Option                  `json:"aspect_ratios,omitempty"`
	Media           *MediaCapabilityOverrides `json:"media,omitempty"`
}

// Profile specializes common capabilities for a model family.
type Profile struct {
	ID            string              `json:"id"`
	Label         string              `json:"label,omitempty"`
	Enabled       *bool               `json:"enabled,omitempty"`
	ExactModels   []string            `json:"exact_models,omitempty"`
	ModelPrefixes []string            `json:"model_prefixes,omitempty"`
	Overrides     CapabilityOverrides `json:"overrides,omitempty"`
}

type ProfileMatchKind string

const (
	ProfileMatchExact   ProfileMatchKind = "exact"
	ProfileMatchPrefix  ProfileMatchKind = "prefix"
	ProfileMatchDefault ProfileMatchKind = "default"
)

// ProfileResolution records both resolved values and fallback diagnostics.
type ProfileResolution struct {
	ProfileID    string
	ProfileLabel string
	MatchKind    ProfileMatchKind
	UsedDefault  bool
	Capabilities Capabilities
}
