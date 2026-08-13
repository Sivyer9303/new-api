package brioi

import "github.com/QuantumNous/new-api/setting/brioi_setting"

const (
	ModelSeedance20Fast = brioi_setting.ModelSeedance20Fast
	ModelSeedance20     = brioi_setting.ModelSeedance20
	ModelSeedance25     = brioi_setting.ModelSeedance25

	GenerationTextToVideo  = brioi_setting.GenerationText2Video
	GenerationImageToVideo = brioi_setting.GenerationImage2Video
	GenerationMultiImage   = brioi_setting.GenerationMultiImage
	GenerationFirstFrame   = brioi_setting.GenerationFirstFrame
	GenerationStartEnd     = brioi_setting.GenerationStartEnd
)

var modelList = []string{
	ModelSeedance20Fast,
	ModelSeedance20,
	ModelSeedance25,
}

type modelProfile struct {
	minDuration       int
	maxDuration       int
	resolutions       map[string]struct{}
	aspectRatios      map[string]struct{}
	maxReferenceItems int
}

var modelProfiles = map[string]modelProfile{
	ModelSeedance20Fast: {
		minDuration:       4,
		maxDuration:       15,
		resolutions:       stringSet("480p", "720p"),
		aspectRatios:      stringSet("21:9", "16:9", "4:3", "1:1", "3:4", "9:16"),
		maxReferenceItems: 9,
	},
	ModelSeedance20: {
		minDuration:       4,
		maxDuration:       15,
		resolutions:       stringSet("480p", "720p", "1080p", "4K"),
		aspectRatios:      stringSet("21:9", "16:9", "4:3", "1:1", "3:4", "9:16"),
		maxReferenceItems: 9,
	},
	ModelSeedance25: {
		minDuration:       4,
		maxDuration:       29,
		resolutions:       stringSet("480p", "720p"),
		aspectRatios:      stringSet("16:9", "9:16"),
		maxReferenceItems: 30,
	},
}

func stringSet(values ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}
