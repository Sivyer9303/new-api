package brioi_setting

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultBrioiProfilesMatchDocumentedHardCapabilities(t *testing.T) {
	setting := defaultBrioiSetting()
	require.Len(t, setting.Profiles, 3)

	byModel := make(map[string]Profile, len(setting.Profiles))
	for _, profile := range setting.Profiles {
		byModel[profile.Model] = profile
	}

	fast := byModel[ModelSeedance20Fast]
	assert.Equal(t, integerRange(4, 15), fast.Durations)
	assert.Equal(t, []string{"480p", "720p"}, fast.Resolutions)
	assert.Equal(t, []string{"21:9", "16:9", "4:3", "1:1", "3:4", "9:16"}, fast.AspectRatios)
	require.Len(t, fast.GenerationModes, 6)
	assert.Equal(t, 9, fast.GenerationModes[2].ImagesMax)
	assert.Equal(t, GenerationReferenceVideos, fast.GenerationModes[5].Value)
	assert.Equal(t, 9, fast.GenerationModes[5].ImagesMax)

	standard := byModel[ModelSeedance20]
	assert.Equal(t, integerRange(4, 15), standard.Durations)
	assert.Equal(t, []string{"480p", "720p", "1080p", "4K"}, standard.Resolutions)
	assert.Equal(t, fast.AspectRatios, standard.AspectRatios)
	assert.Equal(t, 9, standard.GenerationModes[2].ImagesMax)

	latest := byModel[ModelSeedance25]
	assert.Equal(t, integerRange(4, 29), latest.Durations)
	assert.Equal(t, []string{"480p", "720p"}, latest.Resolutions)
	assert.Equal(t, []string{"16:9", "9:16"}, latest.AspectRatios)
	assert.Equal(t, 30, latest.GenerationModes[2].ImagesMax)
	assert.Equal(t, 9, latest.GenerationModes[5].ImagesMax)
}

func TestResolveProfileUsesExactMappedUpstreamNameWithoutFallback(t *testing.T) {
	previous := brioiSetting
	t.Cleanup(func() { brioiSetting = previous })
	brioiSetting = defaultBrioiSetting()

	profile, ok := ResolveProfile(" seedance-2-0 ")
	require.True(t, ok)
	assert.Equal(t, ModelSeedance20, profile.Model)

	_, ok = ResolveProfile("public-seedance-alias")
	assert.False(t, ok)

	brioiSetting.Profiles[1].Enabled = false
	_, ok = ResolveProfile(ModelSeedance20)
	assert.False(t, ok)
}

func TestSoftMergeAddsReferenceVideosMode(t *testing.T) {
	setting := defaultBrioiSetting()
	for index := range setting.Profiles {
		modes := setting.Profiles[index].GenerationModes
		filtered := modes[:0]
		for _, mode := range modes {
			if mode.Value == GenerationReferenceVideos {
				continue
			}
			filtered = append(filtered, mode)
		}
		setting.Profiles[index].GenerationModes = filtered
	}
	require.NoError(t, ValidateBrioiSetting(&setting))
	for _, profile := range setting.Profiles {
		mode, ok := FindGenerationMode(profile, GenerationReferenceVideos)
		require.True(t, ok)
		assert.True(t, mode.Enabled)
		assert.Greater(t, mode.ImagesMax, 0)
	}
}

func TestValidateBrioiSettingRejectsValuesOutsideEveryHardBoundary(t *testing.T) {
	tests := []struct {
		name   string
		change func(*BrioiSetting)
		field  string
	}{
		{
			name: "duration",
			change: func(setting *BrioiSetting) {
				setting.Profiles[0].Durations = []int{3}
			},
			field: "durations",
		},
		{
			name: "resolution",
			change: func(setting *BrioiSetting) {
				setting.Profiles[2].Resolutions = []string{"1080p"}
			},
			field: "resolutions",
		},
		{
			name: "aspect ratio",
			change: func(setting *BrioiSetting) {
				setting.Profiles[2].AspectRatios = []string{"1:1"}
			},
			field: "aspect_ratios",
		},
		{
			name: "generation mode",
			change: func(setting *BrioiSetting) {
				setting.Profiles[0].GenerationModes[0].Value = "video_reference"
			},
			field: "generation_modes",
		},
		{
			name: "image limit",
			change: func(setting *BrioiSetting) {
				setting.Profiles[0].GenerationModes[2].ImagesMax = 10
			},
			field: "images_max",
		},
		{
			name: "unknown model",
			change: func(setting *BrioiSetting) {
				setting.Profiles[0].Model = "seedance-unknown"
			},
			field: "not supported",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setting := defaultBrioiSetting()
			test.change(&setting)
			require.ErrorContains(t, ValidateBrioiSetting(&setting), test.field)
		})
	}
}

func TestProfileUnmarshalAcceptsAdministratorOverridesShape(t *testing.T) {
	var profile Profile
	require.NoError(t, common.Unmarshal([]byte(`{
		"id":"seedance-2-0-fast",
		"label":"Seedance 2.0 Fast",
		"enabled":true,
		"exact_models":["seedance-2-0-fast"],
		"overrides":{
			"generation_types":["text2video","multi_image"],
			"durations":[
				{"value":"4","enabled":true},
				{"value":"5","enabled":false},
				{"value":"15","enabled":true}
			],
			"resolutions":[{"value":"720p","enabled":true}],
			"aspect_ratios":[{"value":"16:9","enabled":true}],
			"media":{"max_items":5}
		}
	}`), &profile))

	assert.Equal(t, ModelSeedance20Fast, profile.Model)
	assert.Equal(t, []int{4, 15}, profile.Durations)
	assert.Equal(t, []string{"720p"}, profile.Resolutions)
	assert.Equal(t, []string{"16:9"}, profile.AspectRatios)
	multiImage, ok := FindGenerationMode(profile, GenerationMultiImage)
	require.True(t, ok)
	assert.Equal(t, 5, multiImage.ImagesMax)
	_, ok = FindGenerationMode(profile, GenerationFirstFrame)
	assert.False(t, ok)

	setting := defaultBrioiSetting()
	setting.Profiles[0] = profile
	require.NoError(t, ValidateBrioiSetting(&setting))
}

func TestProfileUnmarshalRejectsUnsupportedDisabledOptionsAndModes(t *testing.T) {
	var profile Profile
	err := common.Unmarshal([]byte(`{
		"id":"seedance-2-5",
		"label":"Seedance 2.5",
		"exact_models":["seedance-2-5"],
		"overrides":{
			"generation_types":["text2video"],
			"durations":[{"value":"30","enabled":false}],
			"resolutions":[{"value":"720p","enabled":true}],
			"aspect_ratios":[{"value":"16:9","enabled":true}]
		}
	}`), &profile)
	require.ErrorContains(t, err, "outside the Brioi hard capabilities")

	err = common.Unmarshal([]byte(`{
		"id":"seedance-2-5",
		"label":"Seedance 2.5",
		"exact_models":["seedance-2-5"],
		"overrides":{
			"generation_types":["video_reference"],
			"durations":[{"value":"4","enabled":true}],
			"resolutions":[{"value":"720p","enabled":true}],
			"aspect_ratios":[{"value":"16:9","enabled":true}]
		}
	}`), &profile)
	require.ErrorContains(t, err, "generation type")
}

func TestProfileUnmarshalDoesNotDefaultExplicitEmptyGenerationModes(t *testing.T) {
	var profile Profile
	require.NoError(t, common.Unmarshal([]byte(`{
		"model":"seedance-2-0",
		"label":"Seedance 2.0",
		"enabled":true,
		"durations":[4],
		"resolutions":["720p"],
		"aspect_ratios":["16:9"],
		"generation_modes":[]
	}`), &profile))
	assert.Empty(t, profile.GenerationModes)

	setting := defaultBrioiSetting()
	setting.Profiles[1] = profile
	require.ErrorContains(
		t,
		ValidateBrioiSetting(&setting),
		"generation_modes must contain every supported Brioi generation mode",
	)
}

func TestPublicBrioiConfigOmitsDisabledAndAdministrativeFields(t *testing.T) {
	previous := brioiSetting
	t.Cleanup(func() { brioiSetting = previous })
	brioiSetting = defaultBrioiSetting()
	brioiSetting.VideoToolGroups = []string{" brioi ", "brioi"}
	brioiSetting.Profiles[0].Durations = []int{4, 15}
	brioiSetting.Profiles[0].GenerationModes[2].Enabled = false

	public := GetPublicVideoToolConfig()
	require.True(t, public.Enabled)
	assert.Equal(t, []string{"brioi"}, public.VideoToolGroups)
	require.Len(t, public.Profiles, 3)
	assert.Equal(t, []int{4, 15}, public.Profiles[0].Durations)
	assert.Equal(t, ModelSeedance20Fast, public.Profiles[0].ID)
	assert.Equal(t, []string{ModelSeedance20Fast}, public.Profiles[0].ExactModels)
	assert.True(t, public.StrictModelMatching)
	assert.NotEmpty(t, public.GenerationTypes)
	for _, mode := range public.GenerationTypes {
		assert.False(t, mode.RequireRefModel)
	}
	for _, mode := range public.Profiles[0].GenerationModes {
		assert.NotEqual(t, GenerationMultiImage, mode.Value)
		assert.False(t, mode.RequireRefModel)
	}
	var referenceVideos *PublicGenerationMode
	for index := range public.GenerationTypes {
		if public.GenerationTypes[index].Value == GenerationReferenceVideos {
			referenceVideos = &public.GenerationTypes[index]
			break
		}
	}
	require.NotNil(t, referenceVideos)
	assert.True(t, referenceVideos.AllowVideo)
	assert.True(t, referenceVideos.RequireVideo)
	assert.True(t, referenceVideos.AllowAudio)
	assert.False(t, referenceVideos.RequireAudio)
	assert.Equal(t, ReferenceVideosMin, referenceVideos.VideosMin)
	assert.Equal(t, ReferenceVideosMax, referenceVideos.VideosMax)
	assert.True(t, public.Profiles[0].Media.AllowVideo)
	assert.True(t, public.Profiles[0].Media.AllowAudio)
	assert.Contains(t, public.Profiles[0].Media.AcceptedTypes, "video")
	assert.Contains(t, public.Profiles[0].Media.AcceptedTypes, "audio")

	encoded, err := common.Marshal(public)
	require.NoError(t, err)
	payload := strings.ToLower(string(encoded))
	assert.NotContains(t, payload, "api_key")
	assert.NotContains(t, payload, "secret_access_key")
	assert.NotContains(t, payload, "storage")
	assert.NotContains(t, payload, "admin")
}
