package aistarslab_setting

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	ProfileDefault = "aistarslab"

	GenerationText2Video   = "text2video"
	GenerationImage2Video  = "image2video"
	GenerationFrames2Video = "frames2video"
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
	defaultProfile := DefaultPublicProfile()
	profiles := []PublicProfile{defaultProfile}
	for _, override := range GetAIStarsLabSetting().Profiles {
		profiles = append(profiles, publicProfileForOverride(override))
	}
	return PublicVideoToolConfig{
		ID:                  "aistarslab",
		Label:               "",
		Version:             1,
		Enabled:             true,
		VideoToolGroups:     []string{},
		GenerationTypes:     defaultProfile.GenerationModes,
		Profiles:            profiles,
		DefaultProfileID:    ProfileDefault,
		StrictModelMatching: false,
	}
}

func PublicProfileForModel(publicModel string) PublicProfile {
	if override, ok := findModelOverride(publicModel); ok {
		return publicProfileForOverride(override)
	}
	return DefaultPublicProfile()
}

func findModelOverride(publicModel string) (ModelOverride, bool) {
	model := strings.TrimSpace(publicModel)
	if model == "" {
		return ModelOverride{}, false
	}
	for _, override := range GetAIStarsLabSetting().Profiles {
		if override.Model == model {
			return override, true
		}
	}
	return ModelOverride{}, false
}

func publicProfileForOverride(override ModelOverride) PublicProfile {
	profile := DefaultPublicProfile()
	profile.ID = override.Model
	profile.Label = override.Model
	profile.ExactModels = []string{override.Model}
	if len(override.Resolutions) > 0 {
		profile.Resolutions = stringOptions(override.Resolutions, "resolution")
	}
	return profile
}

func DefaultPublicProfile() PublicProfile {
	modes := []PublicGenerationMode{
		{
			Label: "Text to video", Value: GenerationText2Video, Sort: 1,
		},
		{
			Label: "Image to video", Value: GenerationImage2Video, Sort: 2,
			RequireRefModel: true,
			ImagesMin:       1, ImagesMax: 8, ImageRoles: []string{"reference"},
			AllowAudio: true, AllowVideo: true, VideosMax: 1,
		},
		{
			Label: "First and last frame", Value: GenerationFrames2Video, Sort: 3,
			RequireRefModel: true,
			ImagesMin:       2, ImagesMax: 2, ImageRoles: []string{"first_frame", "last_frame"},
		},
	}
	generationTypes := make([]string, 0, len(modes))
	mediaLimits := make(map[string]PublicMediaLimits, len(modes))
	for _, mode := range modes {
		generationTypes = append(generationTypes, mode.Value)
		mediaLimits[mode.Value] = mediaLimitsForMode(mode)
	}
	return PublicProfile{
		ID:                    ProfileDefault,
		Label:                 "Video",
		Durations:             intOptions([]int{5, 10, 15}, "seconds"),
		Resolutions:           stringOptions([]string{"720p", "1080p", "1K"}, "resolution"),
		AspectRatios:          stringOptions([]string{"16:9", "9:16", "1:1"}, "size"),
		GenerationTypes:       generationTypes,
		GenerationModes:       modes,
		RequireRefModelSuffix: true,
		MentionDialect:        "latin",
		Media: PublicMediaLimits{
			MinItems:      0,
			MaxItems:      8,
			AcceptedTypes: []string{"image", "video", "audio"},
			AllowedRoles:  []string{"reference", "first_frame", "last_frame"},
			AllowAudio:    true,
			AllowVideo:    true,
		},
		MediaLimits: mediaLimits,
	}
}

// PublicModelHasRef reports whether a local/public model name is the paid
// reference variant. Upstream identifiers are unified and must not be used.
func PublicModelHasRef(publicModel string) bool {
	return strings.Contains(publicModel, "-ref")
}

// ValidateGenerationTypeForPublicModel keeps cheaper non-ref models on
// text2video and -ref models on image/frame modes.
func ValidateGenerationTypeForPublicModel(generationType, publicModel string) error {
	generationType = strings.TrimSpace(generationType)
	publicModel = strings.TrimSpace(publicModel)
	hasRef := PublicModelHasRef(publicModel)
	switch generationType {
	case GenerationText2Video:
		if hasRef {
			return fmt.Errorf("generation_type %q requires a model whose name does not contain -ref, got %q", generationType, publicModel)
		}
	case GenerationImage2Video, GenerationFrames2Video:
		if !hasRef {
			return fmt.Errorf("generation_type %q requires a model whose name contains -ref, got %q", generationType, publicModel)
		}
	}
	return nil
}

func mediaLimitsForMode(mode PublicGenerationMode) PublicMediaLimits {
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
		out = append(out, PublicOption{
			Label:       value,
			Value:       value,
			UpstreamKey: upstreamKey,
			Sort:        index + 1,
		})
	}
	return out
}
