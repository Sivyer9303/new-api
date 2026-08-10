package silkroad_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyGenerationMediaImage2Video(t *testing.T) {
	mode, ok := FindGenerationMode(GenerationImage2Video)
	require.True(t, ok)
	body := map[string]any{}
	require.NoError(t, ApplyGenerationMedia(body, mode, []string{"data:image/jpeg;base64,a"}, ""))
	assert.Equal(t, "data:image/jpeg;base64,a", body["image"])
}

func TestApplyGenerationMediaReferenceAudioRequiresAudio(t *testing.T) {
	mode, ok := FindGenerationMode(GenerationReferenceAudio)
	require.True(t, ok)
	err := ApplyGenerationMedia(map[string]any{}, mode, nil, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "audio_url")
}

func TestApplyGenerationMediaReferenceAudioWithImage(t *testing.T) {
	mode, ok := FindGenerationMode(GenerationReferenceAudio)
	require.True(t, ok)
	body := map[string]any{}
	require.NoError(t, ApplyGenerationMedia(body, mode, []string{"data:image/jpeg;base64,a"}, "data:audio/mpeg;base64,b"))
	assert.Equal(t, "data:image/jpeg;base64,a", body["image"])
	assert.Equal(t, "data:audio/mpeg;base64,b", body["audio_url"])
}
