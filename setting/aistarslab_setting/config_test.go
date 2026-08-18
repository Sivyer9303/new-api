package aistarslab_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicProfileForModelUsesConfiguredPublicName(t *testing.T) {
	setting := GetAIStarsLabSetting()
	previous := *setting
	t.Cleanup(func() { *setting = previous })
	*setting = AIStarsLabSetting{
		Profiles: []ModelOverride{
			{Model: "seedance-2.0-fast", Resolutions: []string{"480p", "720p"}},
		},
	}

	matched := PublicProfileForModel("seedance-2.0-fast")
	assert.Equal(t, "seedance-2.0-fast", matched.ID)
	assert.Equal(t, []string{"seedance-2.0-fast"}, matched.ExactModels)
	require.Len(t, matched.Resolutions, 2)
	assert.Equal(t, "480p", matched.Resolutions[0].Value)
	assert.Equal(t, "720p", matched.Resolutions[1].Value)

	unrelated := PublicProfileForModel("seedance-2-0-fast")
	assert.Equal(t, ProfileDefault, unrelated.ID)
	require.Len(t, unrelated.Resolutions, 3)
	assert.Equal(t, "720p", unrelated.Resolutions[0].Value)
	assert.Equal(t, "1080p", unrelated.Resolutions[1].Value)
	assert.Equal(t, "1K", unrelated.Resolutions[2].Value)

	prefixed := PublicProfileForModel("12:seedance-2.0-fast")
	assert.Equal(t, ProfileDefault, prefixed.ID)
	require.Len(t, prefixed.Resolutions, 3)
}

func TestValidateAIStarsLabSettingRejectsDuplicateModels(t *testing.T) {
	err := ValidateAIStarsLabSetting(&AIStarsLabSetting{
		Profiles: []ModelOverride{
			{Model: " seedance-2.0-fast ", Resolutions: []string{"720p", "720p", "1080p"}},
			{Model: "seedance-2.0-fast", Resolutions: []string{"1K"}},
		},
	})
	require.ErrorContains(t, err, "configured more than once")
}

func TestValidateAIStarsLabSettingNormalizesAndKeepsOrder(t *testing.T) {
	setting := &AIStarsLabSetting{
		Profiles: []ModelOverride{
			{Model: " seedance-2.0-fast ", Resolutions: []string{" 1080p ", "720p", "1080p"}},
		},
	}
	require.NoError(t, ValidateAIStarsLabSetting(setting))
	require.Len(t, setting.Profiles, 1)
	assert.Equal(t, "seedance-2.0-fast", setting.Profiles[0].Model)
	assert.Equal(t, []string{"1080p", "720p"}, setting.Profiles[0].Resolutions)
}

func TestGetPublicVideoToolConfigIncludesExactModelProfiles(t *testing.T) {
	setting := GetAIStarsLabSetting()
	previous := *setting
	t.Cleanup(func() { *setting = previous })
	*setting = AIStarsLabSetting{
		Profiles: []ModelOverride{
			{Model: "seedance-2.0-fast", Resolutions: []string{"720p"}},
		},
	}

	config := GetPublicVideoToolConfig()
	require.GreaterOrEqual(t, len(config.Profiles), 2)
	assert.Equal(t, ProfileDefault, config.Profiles[0].ID)
	assert.Equal(t, "seedance-2.0-fast", config.Profiles[1].ID)
	assert.Equal(t, []string{"seedance-2.0-fast"}, config.Profiles[1].ExactModels)
}
