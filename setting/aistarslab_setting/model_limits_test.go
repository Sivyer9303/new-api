package aistarslab_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicProfileForModelDisablesReferenceVideosForMinimaxModels(t *testing.T) {
	for _, model := range []string{
		"minimax-h3-480p-ref",
		"minimax-h3-480p",
		"minimax-h3-720p",
		"minimax-h3-1080p",
		"minimax-h3-2k",
	} {
		t.Run(model, func(t *testing.T) {
			profile := PublicProfileForModel(model)
			imageMode := findGenerationMode(profile.GenerationModes, GenerationImage2Video)
			require.NotNil(t, imageMode)
			assert.Equal(t, 0, imageMode.VideosMax)
			assert.False(t, imageMode.AllowVideo)

			limits := profile.MediaLimits[GenerationImage2Video]
			assert.False(t, limits.AllowVideo)
			assert.NotContains(t, limits.AcceptedTypes, "video")
		})
	}
}

func TestPublicProfileForModelKeepsReferenceVideosForOtherModels(t *testing.T) {
	profile := PublicProfileForModel("seedance-2-0-fast-ref")
	imageMode := findGenerationMode(profile.GenerationModes, GenerationImage2Video)
	require.NotNil(t, imageMode)
	assert.Equal(t, 1, imageMode.VideosMax)
	assert.True(t, imageMode.AllowVideo)
}

func TestValidateReferenceVideoCountRejectsMinimaxReferenceVideos(t *testing.T) {
	for _, model := range []string{"minimax-h3-480p-ref", "minimax-h3-720p"} {
		t.Run(model, func(t *testing.T) {
			err := ValidateReferenceVideoCount(model, GenerationImage2Video, 1)
			require.ErrorContains(t, err, `does not accept reference videos`)
			require.NoError(t, ValidateReferenceVideoCount(model, GenerationImage2Video, 0))
		})
	}
}

func TestValidateReferenceVideoCountAllowsSeedanceReferenceVideos(t *testing.T) {
	require.NoError(t, ValidateReferenceVideoCount("seedance-2-0-fast-ref", GenerationImage2Video, 1))
	err := ValidateReferenceVideoCount("seedance-2-0-fast-ref", GenerationImage2Video, 2)
	require.ErrorContains(t, err, "accepts at most 1 reference video")
}

func findGenerationMode(
	modes []PublicGenerationMode,
	value string,
) *PublicGenerationMode {
	for index := range modes {
		if modes[index].Value == value {
			return &modes[index]
		}
	}
	return nil
}
