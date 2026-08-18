package setting

import (
	"strings"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/brioi_setting"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"
)

type VideoProvider string

const (
	VideoProviderSilkRoad    VideoProvider = "silkroad"
	VideoProviderBrioi       VideoProvider = "brioi"
	VideoProviderCompatVideo VideoProvider = "compat_video"
	VideoProviderAIStarsLab  VideoProvider = "aistarslab"
)

func IsVideoGenerationToolEnabled() bool {
	return GetEffectiveVideoSetting().Enabled
}

// VideoProviderSupportsUpstreamModel reports whether a provider recognizes the
// given upstream model name well enough to expose a model profile for it.
func VideoProviderSupportsUpstreamModel(provider VideoProvider, upstreamModel string) bool {
	switch provider {
	case VideoProviderSilkRoad:
		_, ok := silkroad_setting.ResolveProfile(upstreamModel)
		return ok
	case VideoProviderBrioi:
		_, ok := brioi_setting.ResolveProfile(upstreamModel)
		return ok
	case VideoProviderCompatVideo, VideoProviderAIStarsLab:
		return strings.TrimSpace(upstreamModel) != ""
	default:
		return false
	}
}

func VideoProviderFromChannelType(channelType int) (VideoProvider, bool) {
	switch channelType {
	case constant.ChannelTypeSilkRoad:
		return VideoProviderSilkRoad, true
	case constant.ChannelTypeBrioi:
		return VideoProviderBrioi, true
	case constant.ChannelTypeCompatVideo:
		return VideoProviderCompatVideo, true
	case constant.ChannelTypeAIStarsLab:
		return VideoProviderAIStarsLab, true
	default:
		return "", false
	}
}
