package brioi_setting

import (
	"fmt"

	"github.com/QuantumNous/new-api/setting/config"
)

const (
	ModelSeedance20Fast = "seedance-2-0-fast"
	ModelSeedance20     = "seedance-2-0"
	ModelSeedance25     = "seedance-2-5"
)

const (
	GenerationText2Video      = "text2video"
	GenerationImage2Video     = "image2video"
	GenerationMultiImage      = "multi_image"
	GenerationFirstFrame      = "first_frame"
	GenerationStartEnd        = "start_end"
	GenerationReferenceVideos = "reference_videos"
)

// Mixed-reference bounds from Brioi Seedance 2 docs: images 0–9, videos 1–3,
// audio 0–3, total ref items ≤15. Video may stand alone; audio may not.
const (
	ReferenceVideosMin    = 1
	ReferenceVideosMax    = 3
	ReferenceAudiosMax    = 3
	ReferenceMixImagesMax = 9
	ReferenceMixTotalMax  = 15
)

const (
	ImageRoleReference  = "reference"
	ImageRoleFirstFrame = "first_frame"
	ImageRoleLastFrame  = "last_frame"
)

// GenerationModeSetting lets administrators disable a supported mode and
// reduce its image limit without changing the provider protocol contract.
type GenerationModeSetting struct {
	Value     string `json:"value"`
	Enabled   bool   `json:"enabled"`
	ImagesMax int    `json:"images_max"`
	Sort      int    `json:"sort"`
}

// Profile is the configurable, exact capability set for one Brioi model.
// Values omitted from the option slices are disabled.
type Profile struct {
	Model           string                  `json:"model"`
	Label           string                  `json:"label"`
	Enabled         bool                    `json:"enabled"`
	Durations       []int                   `json:"durations"`
	Resolutions     []string                `json:"resolutions"`
	AspectRatios    []string                `json:"aspect_ratios"`
	GenerationModes []GenerationModeSetting `json:"generation_modes"`
}

type BrioiSetting struct {
	VideoToolGroups []string  `json:"video_tool_groups"`
	Profiles        []Profile `json:"profiles"`
}

var brioiSetting = defaultBrioiSetting()

func init() {
	config.GlobalConfig.Register("brioi_setting", &brioiSetting)
}

func GetBrioiSetting() *BrioiSetting {
	return &brioiSetting
}

func DefaultBrioiSetting() BrioiSetting {
	return defaultBrioiSetting()
}

func defaultBrioiSetting() BrioiSetting {
	return BrioiSetting{
		VideoToolGroups: []string{},
		Profiles: []Profile{
			defaultProfile(ModelSeedance20Fast),
			defaultProfile(ModelSeedance20),
			defaultProfile(ModelSeedance25),
		},
	}
}

func defaultProfile(model string) Profile {
	profile := Profile{
		Model:           model,
		Label:           model,
		Enabled:         true,
		GenerationModes: defaultGenerationModes(model),
	}
	switch model {
	case ModelSeedance20Fast:
		profile.Durations = integerRange(4, 15)
		profile.Resolutions = []string{"480p", "720p"}
		profile.AspectRatios = []string{"21:9", "16:9", "4:3", "1:1", "3:4", "9:16"}
	case ModelSeedance20:
		profile.Durations = integerRange(4, 15)
		profile.Resolutions = []string{"480p", "720p", "1080p", "4K"}
		profile.AspectRatios = []string{"21:9", "16:9", "4:3", "1:1", "3:4", "9:16"}
	case ModelSeedance25:
		profile.Durations = integerRange(4, 29)
		profile.Resolutions = []string{"480p", "720p"}
		profile.AspectRatios = []string{"16:9", "9:16"}
	default:
		panic(fmt.Sprintf("unsupported Brioi model profile %q", model))
	}
	return profile
}

func defaultGenerationModes(model string) []GenerationModeSetting {
	multiImageMax := 9
	if model == ModelSeedance25 {
		multiImageMax = 30
	}
	return []GenerationModeSetting{
		{Value: GenerationText2Video, Enabled: true, ImagesMax: 0, Sort: 1},
		{Value: GenerationImage2Video, Enabled: true, ImagesMax: 1, Sort: 2},
		{Value: GenerationMultiImage, Enabled: true, ImagesMax: multiImageMax, Sort: 3},
		{Value: GenerationFirstFrame, Enabled: true, ImagesMax: 1, Sort: 4},
		{Value: GenerationStartEnd, Enabled: true, ImagesMax: 2, Sort: 5},
		// Mixed refs: companion images optional; video/audio caps are hardcoded.
		{Value: GenerationReferenceVideos, Enabled: true, ImagesMax: ReferenceMixImagesMax, Sort: 6},
	}
}

func integerRange(first, last int) []int {
	values := make([]int, 0, last-first+1)
	for value := first; value <= last; value++ {
		values = append(values, value)
	}
	return values
}
