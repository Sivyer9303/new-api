package silkroad_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func findPublicProfile(profiles []PublicProfile, id string) (PublicProfile, bool) {
	for _, p := range profiles {
		if p.ID == id {
			return p, true
		}
	}
	return PublicProfile{}, false
}

func TestGetPublicVideoToolConfigFiltersDisabled(t *testing.T) {
	prev := silkRoadSetting
	t.Cleanup(func() { silkRoadSetting = prev })

	silkRoadSetting = defaultSilkRoadSetting()
	require.NotEmpty(t, silkRoadSetting.Profiles)
	silkRoadSetting.Profiles[0].Durations[0].Enabled = false

	cfg := GetPublicVideoToolConfig()
	require.True(t, cfg.Enabled)
	assert.Equal(t, 1, cfg.Version)
	assert.Equal(t, silkRoadSetting.DefaultProfileID, cfg.DefaultProfileID)
	seedance, ok := findPublicProfile(cfg.Profiles, "seedance_reverse")
	require.True(t, ok)
	for _, d := range seedance.Durations {
		assert.NotEqual(t, "10", d.Value)
	}
	assert.Len(t, cfg.GenerationTypes, 6)
	assert.Equal(t, GenerationReferenceVideos, cfg.GenerationTypes[5].Value)
	assert.True(t, cfg.GenerationTypes[5].AllowVideo)
	assert.Contains(t, seedance.ModelPrefixes, "seedance-2.0-")
	assert.True(t, seedance.RequireRefModelSuffix)
}

func TestGetPublicVideoToolConfigIncludesPrefixes(t *testing.T) {
	prev := silkRoadSetting
	t.Cleanup(func() { silkRoadSetting = prev })
	silkRoadSetting = defaultSilkRoadSetting()

	cfg := GetPublicVideoToolConfig()
	require.True(t, cfg.Enabled)
	p, ok := findPublicProfile(cfg.Profiles, "dreamina_overseas")
	require.True(t, ok)
	assert.Contains(t, p.ModelPrefixes, "dreamina-seedance-2-0-")
	assert.NotEmpty(t, p.AspectRatios)
	assert.NotEmpty(t, p.Durations)
	assert.Empty(t, cfg.VideoToolGroups)
}

func TestGetPublicVideoToolConfigExposesNormalizedGroups(t *testing.T) {
	prev := silkRoadSetting
	t.Cleanup(func() { silkRoadSetting = prev })
	silkRoadSetting = defaultSilkRoadSetting()
	silkRoadSetting.VideoToolGroups = []string{" default ", "", "default", "silkroad"}

	cfg := GetPublicVideoToolConfig()
	assert.Equal(t, []string{"default", "silkroad"}, cfg.VideoToolGroups)
}

func TestGetPublicVideoToolConfigDisablesRefSuffixWhenConfigured(t *testing.T) {
	prev := silkRoadSetting
	t.Cleanup(func() { silkRoadSetting = prev })

	silkRoadSetting = defaultSilkRoadSetting()
	disabled := false
	silkRoadSetting.Profiles = append(silkRoadSetting.Profiles, Profile{
		ID:                    "grok",
		Label:                 "Grok",
		ExactModels:           []string{"grok-image-video"},
		RequireRefModelSuffix: &disabled,
		Durations: []OptionItem{
			{Label: "5s", Value: "5", UpstreamKey: "seconds", Enabled: true, Sort: 1},
		},
		AspectRatios: defaultAspectRatios(),
	})

	cfg := GetPublicVideoToolConfig()
	grok, ok := findPublicProfile(cfg.Profiles, "grok")
	require.True(t, ok)
	assert.False(t, grok.RequireRefModelSuffix)
}
