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
	require.NoError(t, ApplyGenerationMedia(body, mode, []string{"data:image/jpeg;base64,a"}, "", nil))
	assert.Equal(t, []string{"data:image/jpeg;base64,a"}, body["images"])
}

func TestApplyGenerationMediaReferenceAudioRequiresAudio(t *testing.T) {
	mode, ok := FindGenerationMode(GenerationReferenceAudio)
	require.True(t, ok)
	err := ApplyGenerationMedia(map[string]any{}, mode, nil, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "audio_url")
}

func TestApplyGenerationMediaReferenceAudioRequiresImage(t *testing.T) {
	mode, ok := FindGenerationMode(GenerationReferenceAudio)
	require.True(t, ok)
	err := ApplyGenerationMedia(map[string]any{}, mode, nil, "data:audio/mpeg;base64,b", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least 1 image")
}

func TestApplyGenerationMediaReferenceAudioWithImage(t *testing.T) {
	mode, ok := FindGenerationMode(GenerationReferenceAudio)
	require.True(t, ok)
	body := map[string]any{}
	require.NoError(t, ApplyGenerationMedia(body, mode, []string{"data:image/jpeg;base64,a"}, "data:audio/mpeg;base64,b", nil))
	assert.Equal(t, []string{"data:image/jpeg;base64,a"}, body["images"])
	meta, ok := body["metadata"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []string{"data:audio/mpeg;base64,b"}, meta["audios"])
}

func TestApplyGenerationMediaReferenceVideos(t *testing.T) {
	mode, ok := FindGenerationMode(GenerationReferenceVideos)
	require.True(t, ok)
	body := map[string]any{}
	require.NoError(t, ApplyGenerationMedia(
		body,
		mode,
		[]string{"data:image/jpeg;base64,a"},
		"",
		[]string{"data:video/mp4;base64,v1"},
	))
	assert.Equal(t, []string{"data:image/jpeg;base64,a"}, body["images"])
	meta, ok := body["metadata"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []string{"data:video/mp4;base64,v1"}, meta["reference_videos"])
}

func TestApplyGenerationMediaReferenceVideosRequiresVideo(t *testing.T) {
	mode, ok := FindGenerationMode(GenerationReferenceVideos)
	require.True(t, ok)
	err := ApplyGenerationMedia(map[string]any{}, mode, nil, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reference_videos")
}

func TestApplyGenerationMediaStartEndUsesMetadataFrames(t *testing.T) {
	mode, ok := FindGenerationMode(GenerationStartEnd)
	require.True(t, ok)
	body := map[string]any{}
	require.NoError(t, ApplyGenerationMedia(
		body,
		mode,
		[]string{"data:image/jpeg;base64,first", "data:image/jpeg;base64,last"},
		"",
		nil,
	))
	_, hasImages := body["images"]
	assert.False(t, hasImages)
	meta, ok := body["metadata"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "data:image/jpeg;base64,first", meta["first_frame"])
	assert.Equal(t, "data:image/jpeg;base64,last", meta["last_frame"])
}
