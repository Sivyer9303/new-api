package setting

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/brioi_setting"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"
	"github.com/QuantumNous/new-api/setting/video_setting"
)

type VideoProvider string

const (
	VideoProviderSilkRoad VideoProvider = "silkroad"
	VideoProviderBrioi    VideoProvider = "brioi"
)

type VideoProviderOwnership struct {
	Provider    VideoProvider
	ChannelType int
}

type VideoProviderGroupConflictError struct {
	Group             string
	ExistingProvider  VideoProvider
	RequestedProvider VideoProvider
}

func (err *VideoProviderGroupConflictError) Error() string {
	return fmt.Sprintf(
		"video group %q is already owned by provider %q and cannot be assigned to provider %q",
		err.Group,
		err.ExistingProvider,
		err.RequestedProvider,
	)
}

type VideoProviderResolutionError struct {
	FirstProvider  VideoProvider
	SecondProvider VideoProvider
}

func IsVideoGenerationToolEnabled() bool {
	return videoGenerationToolEnabled(
		GetEffectiveVideoSetting().Enabled,
		silkroad_setting.GetPublicVideoToolConfig().Enabled,
		brioi_setting.GetPublicVideoToolConfig().Enabled,
	)
}

func videoGenerationToolEnabled(globalEnabled, silkRoadEnabled, brioiEnabled bool) bool {
	return globalEnabled && (silkRoadEnabled || brioiEnabled)
}

func (err *VideoProviderResolutionError) Error() string {
	return fmt.Sprintf(
		"video groups resolve to multiple providers: %q and %q",
		err.FirstProvider,
		err.SecondProvider,
	)
}

func ResolveVideoProviderGroup(group string) (VideoProviderOwnership, bool) {
	group = strings.TrimSpace(group)
	if group == "" {
		return VideoProviderOwnership{}, false
	}
	ownership := currentVideoProviderOwnership()
	owner, ok := ownership[group]
	return owner, ok
}

// ResolveVideoProviderForGroups resolves Auto-token groups only when every
// provider-owned group points at the same provider. Unowned groups are ignored
// and cannot become distribution candidates while the returned constraint is set.
func ResolveVideoProviderForGroups(
	groups []string,
) (VideoProviderOwnership, []string, error) {
	var resolved VideoProviderOwnership
	ownedGroups := make([]string, 0, len(groups))
	for _, group := range video_setting.NormalizeVideoToolGroups(groups) {
		owner, ok := ResolveVideoProviderGroup(group)
		if !ok {
			continue
		}
		if resolved.Provider != "" && resolved.Provider != owner.Provider {
			return VideoProviderOwnership{}, nil, &VideoProviderResolutionError{
				FirstProvider:  resolved.Provider,
				SecondProvider: owner.Provider,
			}
		}
		resolved = owner
		ownedGroups = append(ownedGroups, group)
	}
	return resolved, ownedGroups, nil
}

func GetVideoProviderGroupOwnership() map[string]VideoProviderOwnership {
	ownership := currentVideoProviderOwnership()
	out := make(map[string]VideoProviderOwnership, len(ownership))
	for group, owner := range ownership {
		out[group] = owner
	}
	return out
}

func GetVideoProviderGroups(provider VideoProvider) []string {
	switch provider {
	case VideoProviderSilkRoad:
		return currentSilkRoadVideoProviderGroups()
	case VideoProviderBrioi:
		return brioi_setting.NormalizeVideoToolGroups(
			brioi_setting.GetBrioiSetting().VideoToolGroups,
		)
	default:
		return []string{}
	}
}

func ValidateVideoProviderGroups(provider VideoProvider, candidateGroups []string) error {
	var otherProvider VideoProvider
	switch provider {
	case VideoProviderSilkRoad:
		otherProvider = VideoProviderBrioi
	case VideoProviderBrioi:
		otherProvider = VideoProviderSilkRoad
	default:
		return fmt.Errorf("unknown video provider %q", provider)
	}

	otherGroups := make(map[string]struct{})
	for _, group := range GetVideoProviderGroups(otherProvider) {
		otherGroups[group] = struct{}{}
	}
	for _, group := range video_setting.NormalizeVideoToolGroups(candidateGroups) {
		if _, conflict := otherGroups[group]; conflict {
			return &VideoProviderGroupConflictError{
				Group:             group,
				ExistingProvider:  otherProvider,
				RequestedProvider: provider,
			}
		}
	}
	return nil
}

func VideoProviderSupportsUpstreamModel(provider VideoProvider, upstreamModel string) bool {
	switch provider {
	case VideoProviderSilkRoad:
		_, ok := silkroad_setting.ResolveProfile(upstreamModel)
		return ok
	case VideoProviderBrioi:
		_, ok := brioi_setting.ResolveProfile(upstreamModel)
		return ok
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
	default:
		return "", false
	}
}

func currentSilkRoadVideoProviderGroups() []string {
	if config.GlobalConfig.IsExplicit("silkroad_setting.video_tool_groups") {
		return silkroad_setting.NormalizeVideoToolGroups(
			silkroad_setting.GetSilkRoadSetting().VideoToolGroups,
		)
	}
	return video_setting.NormalizeVideoToolGroups(
		GetEffectiveVideoSetting().VideoToolGroups,
	)
}

func currentVideoProviderOwnership() map[string]VideoProviderOwnership {
	return resolveVideoProviderOwnership(
		currentSilkRoadVideoProviderGroups(),
		brioi_setting.GetBrioiSetting().VideoToolGroups,
	)
}

func resolveVideoProviderOwnership(
	silkRoadGroups []string,
	brioiGroups []string,
) map[string]VideoProviderOwnership {
	ownership := make(map[string]VideoProviderOwnership)
	conflicts := make(map[string]struct{})
	for _, group := range video_setting.NormalizeVideoToolGroups(silkRoadGroups) {
		ownership[group] = VideoProviderOwnership{
			Provider:    VideoProviderSilkRoad,
			ChannelType: constant.ChannelTypeSilkRoad,
		}
	}
	for _, group := range video_setting.NormalizeVideoToolGroups(brioiGroups) {
		if _, conflict := ownership[group]; conflict {
			conflicts[group] = struct{}{}
			continue
		}
		ownership[group] = VideoProviderOwnership{
			Provider:    VideoProviderBrioi,
			ChannelType: constant.ChannelTypeBrioi,
		}
	}
	for group := range conflicts {
		delete(ownership, group)
	}
	return ownership
}
