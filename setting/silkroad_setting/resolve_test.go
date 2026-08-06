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

func TestMatchProfileDreamina(t *testing.T) {
	p, ok := MatchProfile("dreamina-seedance-2-0-720")
	require.True(t, ok)
	assert.Equal(t, "dreamina_overseas", p.ID)
}

func TestMatchProfileMiss(t *testing.T) {
	_, ok := MatchProfile("gpt-4o")
	assert.False(t, ok)
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
