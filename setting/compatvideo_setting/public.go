package compatvideo_setting

import (
	"strconv"
	"strings"
)

type PublicOption struct {
	Label       string `json:"label"`
	Value       string `json:"value"`
	UpstreamKey string `json:"upstream_key"`
	Sort        int    `json:"sort"`
}

type PublicGenerationMode struct {
	Label           string   `json:"label"`
	Value           string   `json:"value"`
	Sort            int      `json:"sort"`
	RequireRefModel bool     `json:"require_ref_model"`
	RequireAudio    bool     `json:"require_audio"`
	AllowAudio      bool     `json:"allow_audio"`
	RequireVideo    bool     `json:"require_video"`
	AllowVideo      bool     `json:"allow_video"`
	ImagesMin       int      `json:"images_min"`
	ImagesMax       int      `json:"images_max"`
	VideosMin       int      `json:"videos_min"`
	VideosMax       int      `json:"videos_max"`
	ImageRoles      []string `json:"image_roles"`
}

type PublicMediaLimits struct {
	MinItems      int      `json:"min_items"`
	MaxItems      int      `json:"max_items"`
	AcceptedTypes []string `json:"accepted_types"`
	AllowedRoles  []string `json:"allowed_roles"`
	AllowAudio    bool     `json:"allow_audio"`
	AllowVideo    bool     `json:"allow_video"`
}

type PublicProfile struct {
	ID                    string                       `json:"id"`
	Label                 string                       `json:"label"`
	ExactModels           []string                     `json:"exact_models"`
	ModelPrefixes         []string                     `json:"model_prefixes"`
	Durations             []PublicOption               `json:"durations"`
	Resolutions           []PublicOption               `json:"resolutions"`
	AspectRatios          []PublicOption               `json:"aspect_ratios"`
	GenerationTypes       []string                     `json:"generation_types"`
	GenerationModes       []PublicGenerationMode       `json:"generation_modes"`
	RequireRefModelSuffix bool                         `json:"require_ref_model_suffix"`
	AllowGenerateAudio    bool                         `json:"allow_generate_audio"`
	GenerateAudioDefault  bool                         `json:"generate_audio_default"`
	MultiImageMaxDuration int                          `json:"multi_image_max_duration,omitempty"`
	MentionDialect        string                       `json:"mention_dialect"`
	Media                 PublicMediaLimits            `json:"media"`
	MediaLimits           map[string]PublicMediaLimits `json:"media_limits_by_mode"`
}

type PublicVideoToolConfig struct {
	ID                  string                 `json:"id"`
	Label               string                 `json:"label"`
	Version             int                    `json:"version"`
	Enabled             bool                   `json:"enabled"`
	VideoToolGroups     []string               `json:"video_tool_groups"`
	GenerationTypes     []PublicGenerationMode `json:"generation_types"`
	Profiles            []PublicProfile        `json:"profiles"`
	DefaultProfileID    string                 `json:"default_profile_id"`
	StrictModelMatching bool                   `json:"strict_model_matching"`
}

func GetPublicVideoToolConfig() PublicVideoToolConfig {
	profiles := make([]PublicProfile, 0, len(builtInProfiles))
	generationTypes := make([]PublicGenerationMode, 0)
	seenModes := make(map[string]struct{})
	for _, profile := range builtInProfiles {
		public := PublicProfileFor(applyProfileOverrides(cloneProfile(profile)))
		profiles = append(profiles, public)
		for _, mode := range public.GenerationModes {
			if _, exists := seenModes[mode.Value]; exists {
				continue
			}
			seenModes[mode.Value] = struct{}{}
			generationTypes = append(generationTypes, mode)
		}
	}
	return PublicVideoToolConfig{
		ID:                  "compat_video",
		Label:               "",
		Version:             1,
		Enabled:             true,
		VideoToolGroups:     []string{},
		GenerationTypes:     generationTypes,
		Profiles:            profiles,
		DefaultProfileID:    ProfileUnknown,
		StrictModelMatching: false,
	}
}

func PublicProfileFor(profile Profile) PublicProfile {
	modes := make([]PublicGenerationMode, 0, len(profile.GenerationModes))
	generationTypes := make([]string, 0, len(profile.GenerationModes))
	mediaLimits := make(map[string]PublicMediaLimits, len(profile.GenerationModes))
	maxImages := 0
	allowAudio := false
	allowVideo := false
	accepted := map[string]struct{}{}
	roles := map[string]struct{}{}
	for _, mode := range profile.GenerationModes {
		publicMode := publicGenerationMode(mode)
		modes = append(modes, publicMode)
		generationTypes = append(generationTypes, mode.Value)
		limits := mediaLimitsForMode(mode)
		mediaLimits[mode.Value] = limits
		if mode.ImagesMax > maxImages {
			maxImages = mode.ImagesMax
		}
		allowAudio = allowAudio || mode.AllowAudio
		allowVideo = allowVideo || mode.AllowVideo
		for _, mediaType := range limits.AcceptedTypes {
			accepted[mediaType] = struct{}{}
		}
		for _, role := range limits.AllowedRoles {
			roles[role] = struct{}{}
		}
	}
	return PublicProfile{
		ID:                    profile.ID,
		Label:                 publicProfileLabel(profile.Label),
		ExactModels:           append([]string(nil), profile.ExactModels...),
		ModelPrefixes:         append([]string(nil), profile.ModelPrefixes...),
		Durations:             intOptions(profile.Durations, "duration"),
		Resolutions:           stringOptions(profile.Resolutions, "resolution"),
		AspectRatios:          stringOptions(profile.AspectRatios, "aspect_ratio"),
		GenerationTypes:       generationTypes,
		GenerationModes:       modes,
		RequireRefModelSuffix: false,
		AllowGenerateAudio:    profile.AllowGenerateAudio,
		GenerateAudioDefault:  profile.GenerateAudioDefault,
		MultiImageMaxDuration: profile.MultiImageMaxDuration,
		MentionDialect:        "latin",
		Media: PublicMediaLimits{
			MinItems:      0,
			MaxItems:      maxImages,
			AcceptedTypes: setValues(accepted),
			AllowedRoles:  setValues(roles),
			AllowAudio:    allowAudio,
			AllowVideo:    allowVideo,
		},
		MediaLimits: mediaLimits,
	}
}

func publicGenerationMode(mode GenerationMode) PublicGenerationMode {
	roles := append([]string(nil), mode.ImageRoles...)
	if len(roles) == 0 && mode.ImagesMax > 0 {
		roles = []string{"reference"}
	}
	return PublicGenerationMode{
		Label:        mode.Label,
		Value:        mode.Value,
		Sort:         mode.Sort,
		RequireAudio: mode.RequireAudio,
		AllowAudio:   mode.AllowAudio,
		RequireVideo: mode.RequireVideo,
		AllowVideo:   mode.AllowVideo,
		ImagesMin:    mode.ImagesMin,
		ImagesMax:    mode.ImagesMax,
		VideosMin:    mode.VideosMin,
		VideosMax:    mode.VideosMax,
		ImageRoles:   roles,
	}
}

func mediaLimitsForMode(mode GenerationMode) PublicMediaLimits {
	accepted := make([]string, 0, 3)
	if mode.ImagesMax > 0 {
		accepted = append(accepted, "image")
	}
	if mode.AllowVideo || mode.VideosMax > 0 {
		accepted = append(accepted, "video")
	}
	if mode.AllowAudio {
		accepted = append(accepted, "audio")
	}
	return PublicMediaLimits{
		MinItems:      mode.ImagesMin,
		MaxItems:      mode.ImagesMax,
		AcceptedTypes: accepted,
		AllowedRoles:  append([]string(nil), mode.ImageRoles...),
		AllowAudio:    mode.AllowAudio,
		AllowVideo:    mode.AllowVideo || mode.VideosMax > 0,
	}
}

func intOptions(values []int, upstreamKey string) []PublicOption {
	out := make([]PublicOption, 0, len(values))
	for index, value := range values {
		label := strconv.Itoa(value)
		out = append(out, PublicOption{
			Label:       label,
			Value:       label,
			UpstreamKey: upstreamKey,
			Sort:        index + 1,
		})
	}
	return out
}

func stringOptions(values []string, upstreamKey string) []PublicOption {
	out := make([]PublicOption, 0, len(values))
	for index, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, PublicOption{
			Label:       value,
			Value:       value,
			UpstreamKey: upstreamKey,
			Sort:        index + 1,
		})
	}
	return out
}

func publicProfileLabel(label string) string {
	normalized := strings.ToLower(strings.TrimSpace(label))
	if normalized == "" {
		return ""
	}
	if strings.Contains(normalized, "grok") ||
		strings.Contains(normalized, "compatible video") ||
		strings.Contains(normalized, "brioi") ||
		strings.Contains(normalized, "silkroad") ||
		strings.Contains(normalized, "silk road") {
		return "Video"
	}
	return label
}

func setValues(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	return out
}
