package controller

import (
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/brioi_setting"
	"github.com/QuantumNous/new-api/setting/compatvideo_setting"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"
)

func attachVideoToolCapabilities(
	channelType int,
	upstreamModel string,
) (setting.VideoProvider, any, any, bool) {
	provider, ok := setting.VideoProviderFromChannelType(channelType)
	if !ok || !setting.VideoProviderSupportsUpstreamModel(provider, upstreamModel) {
		return "", nil, nil, false
	}
	switch channelType {
	case constant.ChannelTypeSilkRoad:
		profile, genTypes, found := silkRoadAttachedProfile(upstreamModel)
		return provider, profile, genTypes, found
	case constant.ChannelTypeBrioi:
		profile, genTypes, found := brioiAttachedProfile(upstreamModel)
		return provider, profile, genTypes, found
	case constant.ChannelTypeCompatVideo:
		public := compatvideo_setting.PublicProfileFor(compatvideo_setting.MatchProfile(upstreamModel))
		return provider, public, public.GenerationModes, true
	default:
		return provider, nil, nil, false
	}
}

func silkRoadAttachedProfile(upstreamModel string) (any, any, bool) {
	cfg := silkroad_setting.GetPublicVideoToolConfig()
	matched, ok := silkroad_setting.MatchProfile(upstreamModel)
	if !ok {
		return nil, nil, false
	}
	for _, profile := range cfg.Profiles {
		if profile.ID == matched.ID {
			return profile, cfg.GenerationTypes, true
		}
	}
	return nil, cfg.GenerationTypes, true
}

func brioiAttachedProfile(upstreamModel string) (any, any, bool) {
	cfg := brioi_setting.GetPublicVideoToolConfig()
	for _, profile := range cfg.Profiles {
		for _, exact := range profile.ExactModels {
			if exact == upstreamModel {
				return profile, profile.GenerationModes, true
			}
		}
		if profile.Model == upstreamModel {
			return profile, profile.GenerationModes, true
		}
	}
	resolved, ok := brioi_setting.ResolveProfile(upstreamModel)
	if !ok {
		return nil, nil, false
	}
	for _, profile := range cfg.Profiles {
		if profile.Model == resolved.Model {
			return profile, profile.GenerationModes, true
		}
	}
	return nil, cfg.GenerationTypes, true
}
