package aistarslab_setting

import (
	"fmt"
	"strings"
)

const minimaxPublicModelPrefix = "minimax"

// publicModelUsesMinimaxLimits reports whether the local/public AIStarsLab
// model alias belongs to the MiniMax family, which rejects reference videos.
func publicModelUsesMinimaxLimits(publicModel string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(publicModel)), minimaxPublicModelPrefix)
}

func applyModelSpecificGenerationLimits(publicModel string, profile PublicProfile) PublicProfile {
	if !publicModelUsesMinimaxLimits(publicModel) {
		return profile
	}
	modes := make([]PublicGenerationMode, len(profile.GenerationModes))
	for index, mode := range profile.GenerationModes {
		modes[index] = mode
		if mode.Value != GenerationImage2Video {
			continue
		}
		modes[index].VideosMax = 0
		modes[index].VideosMin = 0
		modes[index].AllowVideo = false
		modes[index].RequireVideo = false
	}
	profile.GenerationModes = modes
	profile.MediaLimits = mediaLimitsForModes(modes)
	return profile
}

func mediaLimitsForModes(modes []PublicGenerationMode) map[string]PublicMediaLimits {
	limits := make(map[string]PublicMediaLimits, len(modes))
	for _, mode := range modes {
		limits[mode.Value] = mediaLimitsForMode(mode)
	}
	return limits
}

// ValidateReferenceVideoCount rejects image2video requests that exceed the
// per-model reference-video limit for AIStarsLab public model aliases.
func ValidateReferenceVideoCount(publicModel, generationType string, videoCount int) error {
	if videoCount <= 0 || generationType != GenerationImage2Video {
		return nil
	}
	max := referenceVideosMaxForPublicModel(publicModel)
	if videoCount <= max {
		return nil
	}
	publicModel = strings.TrimSpace(publicModel)
	if max == 0 {
		return fmt.Errorf("model %q does not accept reference videos", publicModel)
	}
	return fmt.Errorf("model %q accepts at most %d reference video(s)", publicModel, max)
}

func referenceVideosMaxForPublicModel(publicModel string) int {
	profile := PublicProfileForModel(publicModel)
	for _, mode := range profile.GenerationModes {
		if mode.Value == GenerationImage2Video {
			return mode.VideosMax
		}
	}
	return 0
}
