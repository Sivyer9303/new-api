package silkroad_setting

import "sort"

// PublicOption is a user-facing selectable option (no internal-only fields beyond what the tool needs).
type PublicOption struct {
	Label       string `json:"label"`
	Value       string `json:"value"`
	UpstreamKey string `json:"upstream_key"`
	Sort        int    `json:"sort"`
}

// PublicGenerationType is a user-facing generation mode (always the hardcoded set).
type PublicGenerationType struct {
	Label           string `json:"label"`
	Value           string `json:"value"`
	Sort            int    `json:"sort"`
	RequireRefModel bool   `json:"require_ref_model"`
	RequireAudio    bool   `json:"require_audio"`
	AllowAudio      bool   `json:"allow_audio"`
	RequireVideo    bool   `json:"require_video"`
	AllowVideo      bool   `json:"allow_video"`
	ImagesMin       int    `json:"images_min"`
	ImagesMax       int    `json:"images_max"`
	VideosMin       int    `json:"videos_min"`
	VideosMax       int    `json:"videos_max"`
}

// PublicProfile is a sanitized profile for the logged-in video tool UI.
type PublicProfile struct {
	ID                    string         `json:"id"`
	Label                 string         `json:"label"`
	ExactModels           []string       `json:"exact_models"`
	ModelPrefixes         []string       `json:"model_prefixes"`
	Durations             []PublicOption `json:"durations"`
	AspectRatios          []PublicOption `json:"aspect_ratios"`
	RequireRefModelSuffix bool           `json:"require_ref_model_suffix"`
	AllowGenerateAudio    bool           `json:"allow_generate_audio"`
}

// PublicVideoToolConfig is returned to logged-in users for the Seedance-style tool page.
type PublicVideoToolConfig struct {
	Version          int                    `json:"version"`
	Enabled          bool                   `json:"enabled"`
	VideoToolGroups  []string               `json:"video_tool_groups"`
	GenerationTypes  []PublicGenerationType `json:"generation_types"`
	Profiles         []PublicProfile        `json:"profiles"`
	DefaultProfileID string                 `json:"default_profile_id"`
}

// GetPublicVideoToolConfig returns enabled profiles/options only (no storage secrets).
func GetPublicVideoToolConfig() PublicVideoToolConfig {
	s := GetSilkRoadSetting()
	out := PublicVideoToolConfig{
		Version:         1,
		Enabled:         true,
		VideoToolGroups: []string{},
		GenerationTypes: publicHardcodedGenerationTypes(),
		Profiles:        make([]PublicProfile, 0),
	}
	if s == nil || len(s.Profiles) == 0 {
		out.Enabled = false
		return out
	}
	out.DefaultProfileID = s.Profiles[effectiveDefaultProfileIndex(s)].ID
	out.VideoToolGroups = NormalizeVideoToolGroups(s.VideoToolGroups)
	for i := range s.Profiles {
		p := buildProfileResolution(s, i, ProfileMatchDefault, s.Profiles[i].ID).Profile
		pub := PublicProfile{
			ID:                    p.ID,
			Label:                 p.Label,
			ExactModels:           append([]string{}, p.ExactModels...),
			ModelPrefixes:         append([]string{}, p.ModelPrefixes...),
			Durations:             publicOptions(p.Durations),
			AspectRatios:          publicOptions(p.AspectRatios),
			RequireRefModelSuffix: p.EnforcesRefModelSuffix(),
			AllowGenerateAudio:    true,
		}
		if len(pub.Durations) == 0 || len(pub.AspectRatios) == 0 {
			continue
		}
		out.Profiles = append(out.Profiles, pub)
	}
	if len(out.Profiles) == 0 {
		out.Enabled = false
	}
	return out
}

func publicOptions(items []OptionItem) []PublicOption {
	out := make([]PublicOption, 0, len(items))
	for _, it := range items {
		if !it.Enabled {
			continue
		}
		out = append(out, PublicOption{
			Label:       it.Label,
			Value:       it.Value,
			UpstreamKey: it.UpstreamKey,
			Sort:        it.Sort,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Sort < out[j].Sort })
	return out
}

func publicHardcodedGenerationTypes() []PublicGenerationType {
	modes := HardcodedGenerationModes()
	out := make([]PublicGenerationType, 0, len(modes))
	for _, m := range modes {
		out = append(out, PublicGenerationType{
			Label:           m.Label,
			Value:           m.Value,
			Sort:            m.Sort,
			RequireRefModel: m.RequireRefModel,
			RequireAudio:    m.RequireAudio,
			AllowAudio:      m.AllowAudio,
			RequireVideo:    m.RequireVideo,
			AllowVideo:      m.AllowVideo,
			ImagesMin:       m.ImagesMin,
			ImagesMax:       m.ImagesMax,
			VideosMin:       m.VideosMin,
			VideosMax:       m.VideosMax,
		})
	}
	return out
}
