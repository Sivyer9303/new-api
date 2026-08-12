package video_setting

import (
	"testing"
	"time"

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
			LocalRetentionDays:    7,
		},
	}

	effective := ResolveEffectiveSetting(configured, ExplicitFields{}, legacy)

	assert.True(t, effective.Enabled)
	assert.Equal(t, []string{"legacy", "vip"}, effective.VideoToolGroups)
	assert.True(t, effective.StorageEnabled)
	assert.Equal(t, "/legacy/videos", effective.Storage.LocalDir)
	assert.Equal(t, 3, effective.Storage.MaxRetry)
	assert.Equal(t, "legacy-node", effective.Storage.IngestNodeName)
	assert.Equal(t, 7, effective.Storage.RetentionDays())
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
			LocalRetentionDays:    9,
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
	assert.Equal(t, "/new/videos", effective.Storage.LocalDir)
	assert.Equal(t, 9, effective.Storage.RetentionDays())
}

func TestResolveEffectiveSettingFillsDriverDefaults(t *testing.T) {
	effective := ResolveEffectiveSetting(
		VideoSetting{Storage: StorageSetting{Driver: "r2"}},
		ExplicitFields{Storage: true},
		LegacySetting{},
	)

	assert.Equal(t, DriverR2, effective.Storage.Driver)
	assert.Equal(t, DefaultR2RetentionDays, effective.Storage.RetentionDays())
	assert.Equal(t, DefaultR2ResultPrefix, effective.Storage.R2.ResultPrefix)
	assert.Equal(t, DefaultR2InputPrefix, effective.Storage.R2.InputPrefix)
	assert.Equal(t, 900*time.Second, effective.Storage.R2.ResultPresignTTL())
	assert.Equal(t, 6*time.Hour, effective.Storage.R2.InputPresignTTL())
	assert.Equal(t, 24*time.Hour, effective.Storage.R2.InputTTL())
}

func TestValidateVideoSettingNormalizesGroupsAndEnforcesStorageBounds(t *testing.T) {
	s := defaultVideoSetting()
	s.VideoToolGroups = []string{" default ", "", "vip", "default"}
	require.NoError(t, ValidateVideoSetting(&s))
	assert.Equal(t, []string{"default", "vip"}, s.VideoToolGroups)

	s = defaultVideoSetting()
	s.Storage.Driver = "s3"
	require.ErrorContains(t, ValidateVideoSetting(&s), "driver")

	s = defaultVideoSetting()
	s.Storage.LocalRetentionDays = MaxRetentionDays + 1
	require.ErrorContains(t, ValidateVideoSetting(&s), "local_retention_days")
}

func TestValidateVideoSettingRequiresR2CredentialsForR2Driver(t *testing.T) {
	s := defaultVideoSetting()
	s.Storage.Driver = DriverR2
	require.ErrorContains(t, ValidateVideoSetting(&s), "storage.r2.account_id")

	s.Storage.R2 = R2StorageSetting{
		AccountID:       "acct",
		AccessKeyID:     "ak",
		SecretAccessKey: "sk",
		APIToken:        "token",
		Bucket:          "videos",
	}
	require.NoError(t, ValidateVideoSetting(&s))
	assert.Equal(t, "https://acct.r2.cloudflarestorage.com", s.Storage.R2.ResolveEndpoint())
	assert.Equal(t, DefaultR2RetentionDays, s.Storage.R2.RetentionDays)

	s.Storage.R2.RetentionDays = 0
	require.NoError(t, ValidateVideoSetting(&s))
	assert.Equal(t, DefaultR2RetentionDays, s.Storage.R2.RetentionDays)

	s.Storage.R2.InputPrefix = "videos"
	require.ErrorContains(t, ValidateVideoSetting(&s), "must differ")
}

func TestValidateVideoSettingIgnoresIngestNodeForR2(t *testing.T) {
	s := defaultVideoSetting()
	s.Enabled = true
	s.Storage.Driver = DriverR2
	s.Storage.PublicDownloadBaseURL = "https://video.example.com"
	s.Storage.R2 = R2StorageSetting{
		AccountID:       "acct",
		AccessKeyID:     "ak",
		SecretAccessKey: "sk",
		APIToken:        "token",
		Bucket:          "videos",
	}

	require.NoError(t, ValidateVideoSetting(&s))

	s.Storage.PublicDownloadBaseURL = ""
	require.ErrorContains(t, ValidateVideoSetting(&s), "public_download_base_url")
}

func TestValidateVideoSettingRequiresIngestWiringForEnabledLocalDriver(t *testing.T) {
	s := defaultVideoSetting()
	s.Enabled = true

	require.ErrorContains(t, ValidateVideoSetting(&s), "ingest_node_name")

	s.Storage.IngestNodeName = "node-a"
	require.ErrorContains(t, ValidateVideoSetting(&s), "public_download_base_url")
}

func TestR2SoftLimitIsNinetyPercentOfFreeTier(t *testing.T) {
	assert.Equal(t, int64(10<<30), R2FreeTierBytes)
	assert.Equal(t, int64(9<<30), R2SoftLimitBytes())
}

func TestObjectKeyJoinsPrefixWithoutDoubleSlashes(t *testing.T) {
	assert.Equal(t, "videos/task-1", ObjectKey("videos/", "task-1"))
	assert.Equal(t, "videos/task-1", ObjectKey("/videos", "/task-1"))
	assert.Equal(t, "task-1", ObjectKey("", "task-1"))
}

func TestLocalRetentionDefaultStaysSevenDays(t *testing.T) {
	assert.Equal(t, 7, RetentionDays)
	assert.Equal(t, 7, defaultStorageSetting().RetentionDays())
}
