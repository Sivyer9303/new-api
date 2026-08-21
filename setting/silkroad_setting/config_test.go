package silkroad_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultSilkRoadSettingProfiles(t *testing.T) {
	s := defaultSilkRoadSetting()
	require.Len(t, s.Profiles, 2)
	assert.Equal(t, "seedance_reverse", s.DefaultProfileID)
	require.NotEmpty(t, s.Common.Durations)
	require.Len(t, s.Common.AspectRatios, 6)

	seedance := s.Profiles[0]
	assert.Equal(t, "seedance_reverse", seedance.ID)
	assert.Equal(t, []string{"seedance-2.0-", "seedance-2-0", "seedance-2-5"}, seedance.ModelPrefixes)
	require.Len(t, seedance.Durations, 12)
	assert.Equal(t, "duration", seedance.Durations[0].UpstreamKey)
	assert.Equal(t, "4", seedance.Durations[0].Value)
	assert.Equal(t, "15", seedance.Durations[len(seedance.Durations)-1].Value)
	assert.False(t, seedance.EnforcesRefModelSuffix())
	assert.Empty(t, seedance.AspectRatios)

	dreamina := s.Profiles[1]
	assert.Equal(t, "dreamina_overseas", dreamina.ID)
	assert.Contains(t, dreamina.ModelPrefixes, "dreamina-seedance-2-0-")
	require.Len(t, dreamina.Durations, 12)
	assert.Equal(t, "duration", dreamina.Durations[0].UpstreamKey)
	assert.Equal(t, "4", dreamina.Durations[0].Value)
	assert.False(t, dreamina.EnforcesRefModelSuffix())
	assert.Empty(t, dreamina.AspectRatios)
}

func TestHardcodedGenerationModes(t *testing.T) {
	modes := HardcodedGenerationModes()
	require.Len(t, modes, 6)
	assert.Equal(t, GenerationText2Video, modes[0].Value)
	assert.Equal(t, GenerationImage2Video, modes[1].Value)
	assert.Equal(t, GenerationMultiImage, modes[2].Value)
	assert.Equal(t, GenerationStartEnd, modes[3].Value)
	assert.Equal(t, GenerationReferenceAudio, modes[4].Value)
	assert.Equal(t, GenerationReferenceVideos, modes[5].Value)

	assert.Equal(t, 0, modes[0].ImagesMax)
	assert.Equal(t, 1, modes[1].ImagesMax)
	assert.Equal(t, 9, modes[2].ImagesMax)
	assert.Equal(t, 2, modes[3].ImagesMax)
	assert.True(t, modes[4].AllowAudio)
	assert.True(t, modes[4].RequireAudio)
	assert.False(t, modes[1].AllowAudio)
	assert.True(t, modes[1].RequireRefModel)
	assert.True(t, modes[5].AllowVideo)
	assert.True(t, modes[5].RequireVideo)
	assert.Equal(t, 1, modes[5].VideosMin)
	assert.Equal(t, 3, modes[5].VideosMax)
	assert.Equal(t, 9, modes[5].ImagesMax)
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
	assert.Empty(t, s.VideoToolGroups)
}

func TestEnforcesRefModelSuffixOmittedDefaultsFalse(t *testing.T) {
	assert.False(t, (*Profile)(nil).EnforcesRefModelSuffix())
	omitted := &Profile{}
	assert.False(t, omitted.EnforcesRefModelSuffix())
	enabled := true
	required := &Profile{RequireRefModelSuffix: &enabled}
	assert.True(t, required.EnforcesRefModelSuffix())
}

func TestGetSilkRoadSettingReturnsDefaults(t *testing.T) {
	got := GetSilkRoadSetting()
	require.NotNil(t, got)
	require.NotEmpty(t, got.Profiles)
	assert.Equal(t, "seedance_reverse", got.Profiles[0].ID)
}
