package compatvideo_setting

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func saveCompatVideoSetting(t *testing.T) *CompatVideoSetting {
	t.Helper()
	previous := compatVideoSetting.Profiles
	t.Cleanup(func() { compatVideoSetting.Profiles = previous })
	compatVideoSetting.Profiles = nil
	return &compatVideoSetting
}

func TestMatchProfileKeepsBuiltInWithoutOverrides(t *testing.T) {
	saveCompatVideoSetting(t)

	profile := MatchProfile("seedance-2-0")
	assert.Equal(t, ProfileSeedance2, profile.ID)
	assert.Equal(t, DialectOpenAIVideos, profile.Dialect)
	assert.Equal(t, duration4to15, profile.Durations)
	assert.Equal(t, []string{"480p", "720p"}, profile.Resolutions)
}

func TestMatchProfileAppliesPerProfileOverrides(t *testing.T) {
	cfg := saveCompatVideoSetting(t)
	cfg.Profiles = []Profile{
		{
			ID:           ProfileSeedance2,
			Durations:    []int{5, 10},
			Resolutions:  []string{"1080p"},
			AspectRatios: []string{"16:9", "9:16"},
			Dialect:      DialectNewAPIGenerations,
		},
	}

	profile := MatchProfile("seedance-2-0")
	assert.Equal(t, ProfileSeedance2, profile.ID)
	assert.Equal(t, []int{5, 10}, profile.Durations)
	assert.Equal(t, []string{"1080p"}, profile.Resolutions)
	assert.Equal(t, []string{"16:9", "9:16"}, profile.AspectRatios)
	assert.Equal(t, DialectNewAPIGenerations, profile.Dialect)
	// Unrelated profile untouched.
	other := MatchProfile("grok-image-video")
	assert.Equal(t, DialectNewAPIGenerations, other.Dialect)
}

func TestMatchProfilePartialOverrideKeepsUnconfiguredFields(t *testing.T) {
	cfg := saveCompatVideoSetting(t)
	cfg.Profiles = []Profile{{ID: ProfileSeedance2, Dialect: DialectNewAPIGenerations}}

	profile := MatchProfile("seedance-2-0")
	assert.Equal(t, DialectNewAPIGenerations, profile.Dialect)
	assert.Equal(t, duration4to15, profile.Durations)
	assert.Equal(t, []string{"480p", "720p"}, profile.Resolutions)
}

func TestMatchProfileOverrideAppliesToUnknownFallback(t *testing.T) {
	cfg := saveCompatVideoSetting(t)
	cfg.Profiles = []Profile{{ID: ProfileUnknown, Dialect: DialectOpenAIVideos}}

	profile := MatchProfile("some-unknown-model")
	assert.Equal(t, ProfileUnknown, profile.ID)
	assert.Equal(t, DialectOpenAIVideos, profile.Dialect)
}

func TestGetPublicVideoToolConfigReflectsOverrides(t *testing.T) {
	cfg := saveCompatVideoSetting(t)
	cfg.Profiles = []Profile{
		{ID: ProfileSeedance2, Resolutions: []string{"4K"}},
	}

	public := GetPublicVideoToolConfig()
	var seedance *PublicProfile
	for i := range public.Profiles {
		if public.Profiles[i].ID == ProfileSeedance2 {
			seedance = &public.Profiles[i]
			break
		}
	}
	require.NotNil(t, seedance)
	require.Len(t, seedance.Resolutions, 1)
	assert.Equal(t, "4K", seedance.Resolutions[0].Value)
}

func TestValidateCompatVideoSettingRejectsUnknownProfile(t *testing.T) {
	err := ValidateCompatVideoSetting(&CompatVideoSetting{
		Profiles: []Profile{{ID: "not-a-profile"}},
	})
	require.ErrorContains(t, err, "unknown compat_video profile")
}

func TestValidateCompatVideoSettingRejectsDuplicateProfile(t *testing.T) {
	err := ValidateCompatVideoSetting(&CompatVideoSetting{
		Profiles: []Profile{{ID: ProfileSeedance2}, {ID: ProfileSeedance2}},
	})
	require.ErrorContains(t, err, "more than once")
}

func TestValidateCompatVideoSettingAcceptsEmptyProfiles(t *testing.T) {
	require.NoError(t, ValidateCompatVideoSetting(&CompatVideoSetting{Profiles: []Profile{}}))
	profile := MatchProfile("seedance-2-0")
	assert.NotEmpty(t, profile.Durations)
}

func TestValidateCompatVideoSettingNormalizesOverridesBeforePersistence(t *testing.T) {
	setting := &CompatVideoSetting{Profiles: []Profile{{
		ID:           "  seedance2  ",
		Durations:    []int{10, 4, 10},
		Resolutions:  []string{" 720p ", "480p", "720p"},
		AspectRatios: []string{" 16:9 ", "9:16", "16:9"},
		Dialect:      " openai_videos ",
	}}}

	require.NoError(t, ValidateCompatVideoSetting(setting))
	require.Len(t, setting.Profiles, 1)
	assert.Equal(t, ProfileSeedance2, setting.Profiles[0].ID)
	assert.Equal(t, []int{4, 10}, setting.Profiles[0].Durations)
	assert.Equal(t, []string{"480p", "720p"}, setting.Profiles[0].Resolutions)
	assert.Equal(t, []string{"16:9", "9:16"}, setting.Profiles[0].AspectRatios)
	assert.Equal(t, DialectOpenAIVideos, setting.Profiles[0].Dialect)
}

func TestValidateCompatVideoSettingRejectsDurationAboveTaskLimit(t *testing.T) {
	err := ValidateCompatVideoSetting(&CompatVideoSetting{Profiles: []Profile{{
		ID:        ProfileSeedance2,
		Durations: []int{relaycommon.MaxTaskDurationSeconds + 1},
	}}})
	require.ErrorContains(t, err, "invalid duration")
}

func TestPublicProfileMediaOptionsAreDeterministic(t *testing.T) {
	profile := PublicProfileFor(Profile{
		ID: "deterministic",
		GenerationModes: []GenerationMode{
			{Value: "mode", ImagesMax: 1, AllowAudio: true, AllowVideo: true, ImageRoles: []string{"reference"}},
		},
	})
	assert.Equal(t, []string{"audio", "image", "video"}, profile.Media.AcceptedTypes)
	assert.Equal(t, []string{"reference"}, profile.Media.AllowedRoles)
}
