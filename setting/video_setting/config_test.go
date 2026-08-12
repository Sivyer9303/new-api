package video_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveEffectiveSettingUsesLegacyValuesWhenNewKeysAreAbsent(t *testing.T) {
	configured := defaultVideoSetting()
	legacy := LegacySetting{
		ToolEnabled:     true,
		VideoToolGroups: []string{" legacy ", "vip", "legacy"},
		StorageEnabled:  true,
		Storage: StorageSetting{
			Driver:                "local",
			LocalDir:              "/legacy/videos",
			MaxRetry:              3,
			IngestNodeName:        "legacy-node",
			PublicDownloadBaseURL: "https://legacy-video.example.com",
		},
	}

	effective := ResolveEffectiveSetting(configured, ExplicitFields{}, legacy)

	assert.True(t, effective.Enabled)
	assert.Equal(t, []string{"legacy", "vip"}, effective.VideoToolGroups)
	assert.True(t, effective.StorageEnabled)
	assert.Equal(t, legacy.Storage, effective.Storage)
}

func TestResolveEffectiveSettingUsesNewValuesPerExplicitField(t *testing.T) {
	configured := VideoSetting{
		Enabled:         false,
		VideoToolGroups: []string{"new"},
		Storage: StorageSetting{
			Driver:                "local",
			LocalDir:              "/new/videos",
			MaxRetry:              7,
			IngestNodeName:        "new-node",
			PublicDownloadBaseURL: "https://new-video.example.com",
		},
	}
	legacy := LegacySetting{
		ToolEnabled:     true,
		VideoToolGroups: []string{"legacy"},
		StorageEnabled:  false,
		Storage:         StorageSetting{LocalDir: "/legacy/videos"},
	}

	effective := ResolveEffectiveSetting(
		configured,
		ExplicitFields{Enabled: true, Storage: true},
		legacy,
	)

	assert.False(t, effective.Enabled)
	assert.Equal(t, []string{"legacy"}, effective.VideoToolGroups)
	assert.True(t, effective.StorageEnabled)
	assert.Equal(t, configured.Storage, effective.Storage)
}

func TestValidateVideoSettingNormalizesGroupsAndEnforcesStorageBounds(t *testing.T) {
	s := defaultVideoSetting()
	s.VideoToolGroups = []string{" default ", "", "vip", "default"}
	require.NoError(t, ValidateVideoSetting(&s))
	assert.Equal(t, []string{"default", "vip"}, s.VideoToolGroups)

	s.Storage.MaxRetry = 0
	require.ErrorContains(t, ValidateVideoSetting(&s), "max_retry")

	s = defaultVideoSetting()
	s.Storage.Driver = "s3"
	require.ErrorContains(t, ValidateVideoSetting(&s), "driver")
}

func TestVideoRetentionIsFixedAtSevenDays(t *testing.T) {
	assert.Equal(t, 7, RetentionDays)
}
