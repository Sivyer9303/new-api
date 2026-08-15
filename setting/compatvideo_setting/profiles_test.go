package compatvideo_setting

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchProfileUsesExactThenLongestPrefixThenUnknown(t *testing.T) {
	seedance := MatchProfile("seedance-2-0")
	assert.Equal(t, ProfileSeedance2, seedance.ID)
	assert.Equal(t, DialectOpenAIVideos, seedance.Dialect)

	grokImage := MatchProfile("grok-imagine-video")
	assert.Equal(t, ProfileGrokImageVideo, grokImage.ID)
	assert.Equal(t, DialectNewAPIGenerations, grokImage.Dialect)

	grokVideo := MatchProfile("grok-video-1.5")
	assert.Equal(t, ProfileGrokVideo15, grokVideo.ID)

	unknown := MatchProfile("veo-3-preview")
	assert.Equal(t, ProfileUnknown, unknown.ID)
	assert.Equal(t, DialectNewAPIGenerations, unknown.Dialect)
}

func TestDurationAllowedCapsMultiImage(t *testing.T) {
	profile := MatchProfile("grok-image-video")
	mode, ok := FindGenerationMode(profile, GenerationMultiImage)
	require.True(t, ok)
	assert.True(t, DurationAllowed(profile, 10, mode))
	assert.False(t, DurationAllowed(profile, 12, mode))

	textMode, ok := FindGenerationMode(profile, GenerationText2Video)
	require.True(t, ok)
	assert.True(t, DurationAllowed(profile, 12, textMode))
}

func TestGetPublicVideoToolConfigIncludesBuiltInProfiles(t *testing.T) {
	cfg := GetPublicVideoToolConfig()
	require.True(t, cfg.Enabled)
	assert.Equal(t, "compat_video", cfg.ID)
	assert.Empty(t, cfg.Label)
	require.NotEmpty(t, cfg.Profiles)
	ids := make([]string, 0, len(cfg.Profiles))
	for _, profile := range cfg.Profiles {
		ids = append(ids, profile.ID)
		assert.NotEmpty(t, profile.Durations)
		assert.NotEmpty(t, profile.AspectRatios)
	}
	assert.Contains(t, ids, ProfileSeedance2)
	assert.Contains(t, ids, ProfileGrokImageVideo)
	assert.Contains(t, ids, ProfileGrokVideo15)
	assert.Contains(t, ids, ProfileUnknown)
	for _, profile := range cfg.Profiles {
		assert.NotContains(t, strings.ToLower(profile.Label), "grok")
		assert.NotContains(t, strings.ToLower(profile.Label), "compatible video")
	}
}
