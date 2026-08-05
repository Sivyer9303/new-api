package silkroad_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultSilkRoadSettingProfiles(t *testing.T) {
	s := defaultSilkRoadSetting()
	require.Len(t, s.Profiles, 2)

	seedance := s.Profiles[0]
	assert.Equal(t, "seedance_reverse", seedance.ID)
	assert.Equal(t, []string{"seedance-2.0-"}, seedance.ModelPrefixes)
	require.Len(t, seedance.Durations, 2)
	assert.Equal(t, "seconds", seedance.Durations[0].UpstreamKey)
	assert.Equal(t, "10", seedance.Durations[0].Value)
	assert.Equal(t, "15", seedance.Durations[1].Value)
	require.Len(t, seedance.AspectRatios, 6)
	require.Len(t, seedance.GenerationTypes, 5)
	assert.Equal(t, "text2video", seedance.GenerationTypes[0].Value)
	assert.Equal(t, "multi_image", seedance.GenerationTypes[2].Value)
	assert.Equal(t, "start_end", seedance.GenerationTypes[4].Value)

	dreamina := s.Profiles[1]
	assert.Equal(t, "dreamina_overseas", dreamina.ID)
	assert.Equal(t, []string{"dreamina-seedance-2-0-"}, dreamina.ModelPrefixes)
	require.Len(t, dreamina.Durations, 2)
	assert.Equal(t, "duration", dreamina.Durations[0].UpstreamKey)
	assert.Equal(t, "4", dreamina.Durations[0].Value)
	assert.Equal(t, "5", dreamina.Durations[1].Value)
	require.Len(t, dreamina.GenerationTypes, 3)
	assert.True(t, dreamina.GenerationTypes[1].RequireRefModel)
	assert.True(t, dreamina.GenerationTypes[2].RequireRefModel)
}

func TestDefaultSilkRoadSettingStorage(t *testing.T) {
	s := defaultSilkRoadSetting()
	assert.False(t, s.Storage.Enabled)
	assert.Equal(t, "local", s.Storage.Driver)
	assert.Equal(t, "data/silkroad-videos", s.Storage.LocalDir)
	assert.Equal(t, 7, s.Storage.RetentionDays)
	assert.Equal(t, 5, s.Storage.MaxRetry)
	assert.Empty(t, s.Storage.IngestNodeName)
	assert.Empty(t, s.Storage.PublicDownloadBaseURL)
}

func TestGetSilkRoadSettingReturnsDefaults(t *testing.T) {
	got := GetSilkRoadSetting()
	require.NotNil(t, got)
	require.NotEmpty(t, got.Profiles)
	assert.Equal(t, "seedance_reverse", got.Profiles[0].ID)
}
