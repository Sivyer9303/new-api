package setting

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
)

func TestVideoProviderFromChannelTypeRoundTrip(t *testing.T) {
	cases := []struct {
		channelType int
		provider    VideoProvider
		wantOK      bool
	}{
		{constant.ChannelTypeSilkRoad, VideoProviderSilkRoad, true},
		{constant.ChannelTypeBrioi, VideoProviderBrioi, true},
		{constant.ChannelTypeCompatVideo, VideoProviderCompatVideo, true},
		{constant.ChannelTypeAIStarsLab, VideoProviderAIStarsLab, true},
		{constant.ChannelTypeOpenAI, "", false},
		{0, "", false},
	}
	for _, tc := range cases {
		provider, ok := VideoProviderFromChannelType(tc.channelType)
		require.Equal(t, tc.wantOK, ok, "channelType %d", tc.channelType)
		if tc.wantOK {
			require.Equal(t, tc.provider, provider)
		}
	}
}

func TestVideoProviderSupportsUpstreamModelRejectsEmpty(t *testing.T) {
	require.False(t, VideoProviderSupportsUpstreamModel(VideoProviderCompatVideo, "  "))
	require.True(t, VideoProviderSupportsUpstreamModel(VideoProviderCompatVideo, "seedance-2-0"))
}
