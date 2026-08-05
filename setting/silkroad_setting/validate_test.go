package silkroad_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRejectsEmptyDurations(t *testing.T) {
	s := defaultSilkRoadSetting()
	s.Profiles[0].Durations = nil
	require.Error(t, ValidateSilkRoadSetting(&s))
}

func TestValidateRejectsEmptyProfileID(t *testing.T) {
	s := defaultSilkRoadSetting()
	s.Profiles[0].ID = ""
	require.Error(t, ValidateSilkRoadSetting(&s))
}

func TestValidateRejectsDisabledOnlyAspectRatios(t *testing.T) {
	s := defaultSilkRoadSetting()
	for i := range s.Profiles[0].AspectRatios {
		s.Profiles[0].AspectRatios[i].Enabled = false
	}
	require.Error(t, ValidateSilkRoadSetting(&s))
}

func TestValidateRejectsEnabledOptionMissingUpstreamKey(t *testing.T) {
	s := defaultSilkRoadSetting()
	s.Profiles[0].Durations[0].UpstreamKey = ""
	require.Error(t, ValidateSilkRoadSetting(&s))
}

func TestValidateRejectsStorageNonLocalDriver(t *testing.T) {
	s := defaultSilkRoadSetting()
	s.Storage.Enabled = true
	s.Storage.Driver = "s3"
	s.Storage.IngestNodeName = "node-a"
	s.Storage.PublicDownloadBaseURL = "https://video.example.com"
	require.Error(t, ValidateSilkRoadSetting(&s))
}

func TestValidateRejectsStorageBadRetention(t *testing.T) {
	s := defaultSilkRoadSetting()
	s.Storage.Enabled = true
	s.Storage.IngestNodeName = "node-a"
	s.Storage.PublicDownloadBaseURL = "https://video.example.com"
	s.Storage.RetentionDays = 0
	require.Error(t, ValidateSilkRoadSetting(&s))
}

func TestValidateAcceptsDefaults(t *testing.T) {
	s := defaultSilkRoadSetting()
	require.NoError(t, ValidateSilkRoadSetting(&s))
}

func TestValidateNilSetting(t *testing.T) {
	err := ValidateSilkRoadSetting(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestValidateRejectsStorageEnabledMissingIngest(t *testing.T) {
	s := defaultSilkRoadSetting()
	s.Storage.Enabled = true
	s.Storage.IngestNodeName = ""
	s.Storage.PublicDownloadBaseURL = "https://video.example.com"
	err := ValidateSilkRoadSetting(&s)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ingest_node_name")
}

func TestValidateRejectsStorageEnabledMissingPublicBase(t *testing.T) {
	s := defaultSilkRoadSetting()
	s.Storage.Enabled = true
	s.Storage.IngestNodeName = "node-a"
	s.Storage.PublicDownloadBaseURL = ""
	err := ValidateSilkRoadSetting(&s)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "public_download_base_url")
}

func TestValidateAcceptsStorageEnabledComplete(t *testing.T) {
	s := defaultSilkRoadSetting()
	s.Storage.Enabled = true
	s.Storage.IngestNodeName = "node-a"
	s.Storage.PublicDownloadBaseURL = "https://video.example.com"
	require.NoError(t, ValidateSilkRoadSetting(&s))
}

func TestValidateNormalizesVideoToolGroups(t *testing.T) {
	s := defaultSilkRoadSetting()
	s.VideoToolGroups = []string{" default ", "", "vip", "default"}
	require.NoError(t, ValidateSilkRoadSetting(&s))
	assert.Equal(t, []string{"default", "vip"}, s.VideoToolGroups)
}

func TestNormalizeVideoToolGroupsEmptyMeansNone(t *testing.T) {
	assert.Empty(t, NormalizeVideoToolGroups(nil))
	assert.Empty(t, NormalizeVideoToolGroups([]string{}))
	assert.Empty(t, NormalizeVideoToolGroups([]string{"", "  "}))
}
