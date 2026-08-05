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
	s.Storage.Driver = "s3"
	require.Error(t, ValidateSilkRoadSetting(&s))
}

func TestValidateRejectsStorageBadRetention(t *testing.T) {
	s := defaultSilkRoadSetting()
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
