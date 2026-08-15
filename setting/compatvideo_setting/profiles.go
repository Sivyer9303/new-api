package compatvideo_setting

import (
	"strings"
)

type Dialect string

const (
	DialectNewAPIGenerations Dialect = "newapi_generations"
	DialectOpenAIVideos      Dialect = "openai_videos"

	ProfileSeedance2      = "seedance2"
	ProfileGrokImageVideo = "grok-image-video"
	ProfileGrokVideo15    = "grok-video-1.5"
	ProfileUnknown        = "unknown"

	GenerationText2Video  = "text2video"
	GenerationImage2Video = "image2video"
	GenerationMultiImage  = "multi_image"
)

type GenerationMode struct {
	Label        string
	Value        string
	Sort         int
	ImagesMin    int
	ImagesMax    int
	VideosMin    int
	VideosMax    int
	RequireAudio bool
	AllowAudio   bool
	RequireVideo bool
	AllowVideo   bool
	ImageRoles   []string
	MaxDuration  int
}

type Profile struct {
	ID                    string
	Label                 string
	ExactModels           []string
	ModelPrefixes         []string
	Dialect               Dialect
	Durations             []int
	Resolutions           []string
	AspectRatios          []string
	GenerationModes       []GenerationMode
	AllowGenerateAudio    bool
	GenerateAudioDefault  bool
	MultiImageMaxDuration int
}

func MatchProfile(upstreamModel string) Profile {
	model := strings.ToLower(strings.TrimSpace(upstreamModel))
	if model == "" {
		return unknownProfile()
	}

	var exact *Profile
	for i := range builtInProfiles {
		profile := &builtInProfiles[i]
		for _, exactModel := range profile.ExactModels {
			if strings.ToLower(strings.TrimSpace(exactModel)) != model {
				continue
			}
			exact = profile
		}
	}
	if exact != nil {
		return cloneProfile(*exact)
	}

	var prefixProfile *Profile
	longest := -1
	for i := range builtInProfiles {
		profile := &builtInProfiles[i]
		for _, rawPrefix := range profile.ModelPrefixes {
			prefix := strings.ToLower(strings.TrimSpace(rawPrefix))
			if prefix == "" || !strings.HasPrefix(model, prefix) {
				continue
			}
			if len(prefix) > longest {
				prefixProfile = profile
				longest = len(prefix)
			}
		}
	}
	if prefixProfile != nil {
		return cloneProfile(*prefixProfile)
	}
	return unknownProfile()
}

func FindGenerationMode(profile Profile, value string) (GenerationMode, bool) {
	value = strings.TrimSpace(value)
	for _, mode := range profile.GenerationModes {
		if mode.Value == value {
			return mode, true
		}
	}
	return GenerationMode{}, false
}

func DurationAllowed(profile Profile, seconds int, mode GenerationMode) bool {
	if seconds <= 0 {
		return false
	}
	allowed := false
	for _, option := range profile.Durations {
		if option == seconds {
			allowed = true
			break
		}
	}
	if !allowed {
		return false
	}
	maxDuration := mode.MaxDuration
	if maxDuration <= 0 {
		maxDuration = profile.MultiImageMaxDuration
	}
	if mode.Value == GenerationMultiImage && maxDuration > 0 && seconds > maxDuration {
		return false
	}
	return true
}

func cloneProfile(profile Profile) Profile {
	profile.ExactModels = append([]string(nil), profile.ExactModels...)
	profile.ModelPrefixes = append([]string(nil), profile.ModelPrefixes...)
	profile.Durations = append([]int(nil), profile.Durations...)
	profile.Resolutions = append([]string(nil), profile.Resolutions...)
	profile.AspectRatios = append([]string(nil), profile.AspectRatios...)
	profile.GenerationModes = append([]GenerationMode(nil), profile.GenerationModes...)
	for i := range profile.GenerationModes {
		profile.GenerationModes[i].ImageRoles = append(
			[]string(nil),
			profile.GenerationModes[i].ImageRoles...,
		)
	}
	return profile
}

func unknownProfile() Profile {
	return cloneProfile(builtInProfiles[len(builtInProfiles)-1])
}

var duration4to15 = []int{4, 6, 8, 10, 12, 15}

var builtInProfiles = []Profile{
	{
		ID:            ProfileSeedance2,
		Label:         "Seedance 2",
		ExactModels:   []string{"seedance2", "seedance-2", "seedance-2.0", "seedance-2-0"},
		ModelPrefixes: []string{"seedance2", "seedance-2"},
		Dialect:       DialectOpenAIVideos,
		Durations:     duration4to15,
		Resolutions:   []string{"480p", "720p"},
		AspectRatios:  []string{"auto", "21:9", "16:9", "4:3", "1:1", "3:4", "9:16"},
		GenerationModes: []GenerationMode{
			{
				Label: "Text to video", Value: GenerationText2Video, Sort: 1,
				AllowAudio: true, AllowVideo: true, VideosMax: 1,
			},
			{
				Label: "Image to video", Value: GenerationImage2Video, Sort: 2,
				ImagesMin: 1, ImagesMax: 2, ImageRoles: []string{"reference"},
				AllowAudio: true, AllowVideo: true, VideosMax: 1,
			},
		},
		AllowGenerateAudio:   true,
		GenerateAudioDefault: true,
	},
	{
		ID:            ProfileGrokImageVideo,
		Label:         "Grok Image Video",
		ExactModels:   []string{"grok-image-video"},
		ModelPrefixes: []string{"grok-image", "grok-imagine"},
		Dialect:       DialectNewAPIGenerations,
		Durations:     duration4to15,
		Resolutions:   []string{"480p", "720p"},
		AspectRatios:  []string{"1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3"},
		GenerationModes: []GenerationMode{
			{Label: "Text to video", Value: GenerationText2Video, Sort: 1},
			{
				Label: "Image to video", Value: GenerationImage2Video, Sort: 2,
				ImagesMin: 1, ImagesMax: 1, ImageRoles: []string{"reference"},
			},
			{
				Label: "Multi-image", Value: GenerationMultiImage, Sort: 3,
				ImagesMin: 2, ImagesMax: 7, ImageRoles: []string{"reference"},
				MaxDuration: 10,
			},
		},
		MultiImageMaxDuration: 10,
	},
	{
		ID:            ProfileGrokVideo15,
		Label:         "Grok Video 1.5",
		ExactModels:   []string{"grok-video-1.5", "grok-video-1.5-preview"},
		ModelPrefixes: []string{"grok-video-1.5", "grok-video"},
		Dialect:       DialectNewAPIGenerations,
		Durations:     duration4to15,
		Resolutions:   []string{"480p", "720p"},
		AspectRatios:  []string{"16:9", "9:16"},
		GenerationModes: []GenerationMode{
			{
				Label: "Image to video", Value: GenerationImage2Video, Sort: 1,
				ImagesMin: 1, ImagesMax: 1, ImageRoles: []string{"reference"},
			},
		},
	},
	{
		ID:           ProfileUnknown,
		Label:        "Compatible Video",
		ExactModels:  nil,
		Dialect:      DialectNewAPIGenerations,
		Durations:    duration4to15,
		Resolutions:  []string{"480p", "720p"},
		AspectRatios: []string{"16:9", "9:16", "1:1"},
		GenerationModes: []GenerationMode{
			{Label: "Text to video", Value: GenerationText2Video, Sort: 1},
			{
				Label: "Image to video", Value: GenerationImage2Video, Sort: 2,
				ImagesMin: 1, ImagesMax: 2, ImageRoles: []string{"reference"},
			},
		},
	},
}
