package silkroad_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchProfileSeedance(t *testing.T) {
	p, ok := MatchProfile("seedance-2.0-720")
	require.True(t, ok)
	assert.Equal(t, "seedance_reverse", p.ID)
}

func TestMatchProfileSeedanceHyphenatedAndGlobal(t *testing.T) {
	domestic, ok := MatchProfile("seedance-2-0-1080p")
	require.True(t, ok)
	assert.Equal(t, "seedance_reverse", domestic.ID)

	fast, ok := MatchProfile("seedance-2-0-fast")
	require.True(t, ok)
	assert.Equal(t, "seedance_reverse", fast.ID)

	v25, ok := MatchProfile("seedance-2-5")
	require.True(t, ok)
	assert.Equal(t, "seedance_reverse", v25.ID)

	global, ok := MatchProfile("seedance-2-0-global")
	require.True(t, ok)
	assert.Equal(t, "seedance_reverse", global.ID)

	promax, ok := MatchProfile("seedance-2-5-1080p-promax")
	require.True(t, ok)
	assert.Equal(t, "seedance_reverse", promax.ID)
}

func TestMatchProfileMiss(t *testing.T) {
	p, ok := MatchProfile("unmatched-video-model")
	require.True(t, ok)
	assert.Equal(t, defaultSilkRoadSetting().DefaultProfileID, p.ID)
}

func TestResolveProfilePrefersExactThenLongestPrefixThenDefault(t *testing.T) {
	s := defaultSilkRoadSetting()
	s.Profiles = []Profile{
		{ID: "default", Label: "Default"},
		{ID: "short", Label: "Short", ModelPrefixes: []string{"video-"}},
		{ID: "long", Label: "Long", ModelPrefixes: []string{"video-pro-"}},
		{ID: "exact", Label: "Exact", ExactModels: []string{"video-pro-special"}},
	}
	s.DefaultProfileID = "default"

	resolution, ok := resolveProfileFromSetting(&s, "video-pro-special")
	require.True(t, ok)
	assert.Equal(t, "exact", resolution.Profile.ID)
	assert.Equal(t, ProfileMatchExact, resolution.MatchKind)

	resolution, ok = resolveProfileFromSetting(&s, "video-pro-other")
	require.True(t, ok)
	assert.Equal(t, "long", resolution.Profile.ID)
	assert.Equal(t, ProfileMatchPrefix, resolution.MatchKind)

	resolution, ok = resolveProfileFromSetting(&s, "unknown")
	require.True(t, ok)
	assert.Equal(t, "default", resolution.Profile.ID)
	assert.Equal(t, ProfileMatchDefault, resolution.MatchKind)
}

func TestResolveProfileInheritsCommonOptionsForSparseOverrides(t *testing.T) {
	s := defaultSilkRoadSetting()
	s.Profiles[0].Durations = nil
	s.Profiles[0].AspectRatios = nil

	resolution, ok := resolveProfileFromSetting(&s, "seedance-2.0-720")
	require.True(t, ok)
	assert.Equal(t, s.Common.Durations, resolution.Profile.Durations)
	assert.Equal(t, s.Common.AspectRatios, resolution.Profile.AspectRatios)
}

func TestFindEnabledOptionDisabledSkipped(t *testing.T) {
	items := []OptionItem{{Label: "10s", Value: "10", UpstreamKey: "seconds", Enabled: false}}
	_, ok := FindEnabledOption(items, "10")
	assert.False(t, ok)
}

func TestFindEnabledOptionFindsEnabled(t *testing.T) {
	items := []OptionItem{
		{Label: "10s", Value: "10", UpstreamKey: "seconds", Enabled: false},
		{Label: "15s", Value: "15", UpstreamKey: "seconds", Enabled: true},
	}
	opt, ok := FindEnabledOption(items, "15")
	require.True(t, ok)
	assert.Equal(t, "15", opt.Value)
	assert.Equal(t, "seconds", opt.UpstreamKey)
}

func TestFindGenerationMode(t *testing.T) {
	gt, ok := FindGenerationMode("image2video")
	require.True(t, ok)
	assert.Equal(t, "image2video", gt.Value)
	assert.Equal(t, 1, gt.ImagesMin)
	assert.True(t, gt.RequireRefModel)
}

func TestFindGenerationModeUnknown(t *testing.T) {
	_, ok := FindGenerationMode("start_frame")
	assert.False(t, ok)
}
