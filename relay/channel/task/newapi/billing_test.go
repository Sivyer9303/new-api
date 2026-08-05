package newapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEstimateBillingUsesSeconds(t *testing.T) {
	a := &TaskAdaptor{}
	c, info := newTestContext(t, `{
		"model":"seedance-2.0-720",
		"prompt":"a cat running",
		"generation_type":"text2video",
		"seconds":"10",
		"aspect_ratio":"16:9"
	}`)

	taskErr := a.ValidateRequestAndSetAction(c, info)
	require.Nil(t, taskErr)

	ratios := a.EstimateBilling(c, info)
	require.NotNil(t, ratios)
	assert.Equal(t, map[string]float64{"seconds": 10}, ratios)
}

func TestEstimateBillingUsesDurationAsSeconds(t *testing.T) {
	a := &TaskAdaptor{}
	ginBody := `{
		"model":"dreamina-seedance-2-0-720",
		"prompt":"a cat running",
		"generation_type":"text2video",
		"duration":5,
		"aspect_ratio":"16:9"
	}`
	c, info := newTestContext(t, ginBody)
	info.OriginModelName = "dreamina-seedance-2-0-720"
	info.ChannelMeta.UpstreamModelName = "dreamina-seedance-2-0-720"

	taskErr := a.ValidateRequestAndSetAction(c, info)
	require.Nil(t, taskErr)

	ratios := a.EstimateBilling(c, info)
	require.NotNil(t, ratios)
	assert.Equal(t, map[string]float64{"seconds": 5}, ratios)
}
