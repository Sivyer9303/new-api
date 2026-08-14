package brioi_setting

import "sort"

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
	ID              string                       `json:"id"`
	Model           string                       `json:"model"`
	Label           string                       `json:"label"`
	ExactModels     []string                     `json:"exact_models"`
	ModelPrefixes   []string                     `json:"model_prefixes"`
	Durations       []int                        `json:"durations"`
	Resolutions     []string                     `json:"resolutions"`
	AspectRatios    []string                     `json:"aspect_ratios"`
	GenerationTypes []string                     `json:"generation_types"`
	GenerationModes []PublicGenerationMode       `json:"generation_modes"`
	Media           PublicMediaLimits            `json:"media"`
	MediaLimits     map[string]PublicMediaLimits `json:"media_limits_by_mode"`
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
	setting := GetBrioiSetting()
	public := PublicVideoToolConfig{
		ID:                  "brioi",
		Label:               "Brioi",
		Version:             1,
		Enabled:             true,
		VideoToolGroups:     []string{},
		GenerationTypes:     []PublicGenerationMode{},
		Profiles:            []PublicProfile{},
		StrictModelMatching: true,
	}
	if setting == nil {
		public.Enabled = false
		return public
	}

	public.VideoToolGroups = NormalizeVideoToolGroups(setting.VideoToolGroups)
	providerModes := make(map[string]PublicGenerationMode)
	for _, profile := range setting.Profiles {
		if !profile.Enabled {
			continue
		}
		hard, _ := hardProfile(profile.Model)
		generationModes := softMergeGenerationModes(profile.GenerationModes, hard.GenerationModes)
		modes := make([]PublicGenerationMode, 0, len(generationModes))
		generationTypes := make([]string, 0, len(generationModes))
		mediaLimits := make(map[string]PublicMediaLimits)
		maxItems := 0
		allowVideo := false
		for _, mode := range generationModes {
			if !mode.Enabled {
				continue
			}
			publicMode := publicGenerationMode(mode)
			modes = append(modes, publicMode)
			generationTypes = append(generationTypes, publicMode.Value)
			mediaLimits[publicMode.Value] = publicModeMediaLimits(publicMode)
			itemCap := publicMode.ImagesMax + publicMode.VideosMax
			if itemCap > maxItems {
				maxItems = itemCap
			}
			if publicMode.AllowVideo {
				allowVideo = true
			}
			current, exists := providerModes[publicMode.Value]
			if !exists || publicMode.ImagesMax > current.ImagesMax || publicMode.VideosMax > current.VideosMax {
				providerModes[publicMode.Value] = publicMode
			}
		}
		sort.SliceStable(modes, func(i, j int) bool {
			return modes[i].Sort < modes[j].Sort
		})
		generationTypes = generationTypes[:0]
		for _, mode := range modes {
			generationTypes = append(generationTypes, mode.Value)
		}
		if len(profile.Durations) == 0 ||
			len(profile.Resolutions) == 0 ||
			len(profile.AspectRatios) == 0 ||
			len(modes) == 0 {
			continue
		}
		acceptedTypes := []string{"image"}
		if allowVideo {
			acceptedTypes = []string{"image", "video"}
		}
		public.Profiles = append(public.Profiles, PublicProfile{
			ID:              profile.Model,
			Model:           profile.Model,
			Label:           profile.Label,
			ExactModels:     []string{profile.Model},
			ModelPrefixes:   []string{},
			Durations:       append([]int(nil), profile.Durations...),
			Resolutions:     append([]string(nil), profile.Resolutions...),
			AspectRatios:    append([]string(nil), profile.AspectRatios...),
			GenerationTypes: generationTypes,
			GenerationModes: modes,
			Media: PublicMediaLimits{
				MinItems:      0,
				MaxItems:      maxItems,
				AcceptedTypes: acceptedTypes,
				AllowedRoles: []string{
					ImageRoleReference,
					ImageRoleFirstFrame,
					ImageRoleLastFrame,
				},
				AllowAudio: false,
				AllowVideo: allowVideo,
			},
			MediaLimits: mediaLimits,
		})
	}
	for _, mode := range providerModes {
		public.GenerationTypes = append(public.GenerationTypes, mode)
	}
	sort.SliceStable(public.GenerationTypes, func(i, j int) bool {
		return public.GenerationTypes[i].Sort < public.GenerationTypes[j].Sort
	})
	if len(public.Profiles) == 0 {
		public.Enabled = false
	}
	return public
}

func publicGenerationMode(mode GenerationModeSetting) PublicGenerationMode {
	public := PublicGenerationMode{
		Label: mode.Value,
		Value: mode.Value,
		Sort:  mode.Sort,
		// Brioi uses the same exact model ID for text and reference modes.
		// This flag is reserved for providers that expose separate *-ref models.
		RequireRefModel: false,
		ImagesMax:       mode.ImagesMax,
		ImageRoles:      []string{},
	}
	switch mode.Value {
	case GenerationImage2Video:
		public.Label = "Image reference"
		public.ImagesMin = 1
		public.ImageRoles = []string{ImageRoleReference}
	case GenerationMultiImage:
		public.Label = "Multi-image reference"
		public.ImagesMin = 2
		public.ImageRoles = []string{ImageRoleReference}
	case GenerationFirstFrame:
		public.Label = "First frame"
		public.ImagesMin = 1
		public.ImageRoles = []string{ImageRoleFirstFrame}
	case GenerationStartEnd:
		public.Label = "First & last frame"
		public.ImagesMin = 2
		public.ImageRoles = []string{ImageRoleFirstFrame, ImageRoleLastFrame}
	case GenerationReferenceVideos:
		public.Label = "Reference video"
		public.RequireVideo = true
		public.AllowVideo = true
		public.ImagesMin = 0
		public.VideosMin = ReferenceVideosMin
		public.VideosMax = ReferenceVideosMax
		public.ImageRoles = []string{ImageRoleReference}
	case GenerationText2Video:
		public.Label = "Text to video"
	}
	return public
}

func publicModeMediaLimits(mode PublicGenerationMode) PublicMediaLimits {
	acceptedTypes := []string{}
	if mode.ImagesMax > 0 {
		acceptedTypes = append(acceptedTypes, "image")
	}
	if mode.AllowVideo || mode.VideosMax > 0 {
		acceptedTypes = append(acceptedTypes, "video")
	}
	minItems := mode.ImagesMin
	if mode.VideosMin > minItems {
		minItems = mode.VideosMin
	}
	maxItems := mode.ImagesMax + mode.VideosMax
	return PublicMediaLimits{
		MinItems:      minItems,
		MaxItems:      maxItems,
		AcceptedTypes: acceptedTypes,
		AllowedRoles:  append([]string(nil), mode.ImageRoles...),
		AllowAudio:    false,
		AllowVideo:    mode.AllowVideo || mode.VideosMax > 0,
	}
}
