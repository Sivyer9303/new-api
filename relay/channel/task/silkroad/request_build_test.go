package silkroad

import (
	"io"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func metadataOf(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	meta, ok := body["metadata"].(map[string]any)
	require.True(t, ok)
	return meta
}

func TestBuildRequestBodyImage2VideoSetsImages(t *testing.T) {
	a := &TaskAdaptor{}
	c, info := newTestContext(t, `{
		"model":"seedance-2.0-720-ref",
		"prompt":"animate this",
		"generation_type":"image2video",
		"seconds":"10",
		"aspect_ratio":"16:9",
		"images":["data:image/jpeg;base64,abc"]
	}`)
	info.OriginModelName = "seedance-2.0-720-ref"
	info.ChannelMeta.UpstreamModelName = "seedance-2.0-720-ref"

	taskErr := a.ValidateRequestAndSetAction(c, info)
	require.Nil(t, taskErr)

	reader, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	data, err := io.ReadAll(reader)
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, common.Unmarshal(data, &body))

	assert.Equal(t, "seedance-2.0-720-ref", body["model"])
	assert.Equal(t, "animate this", body["prompt"])
	assert.Equal(t, []any{"data:image/jpeg;base64,abc"}, body["images"])
	assert.Equal(t, float64(10), body["duration"])
	assert.Equal(t, "720p", body["resolution"])
	assert.Equal(t, "16:9", metadataOf(t, body)["ratio"])
	_, hasGenType := body["generation_type"]
	assert.False(t, hasGenType, "generation_type must not appear in upstream body")
	_, hasImage := body["image"]
	assert.False(t, hasImage, "singular image must not appear at top level")
	_, hasAspect := body["aspect_ratio"]
	assert.False(t, hasAspect, "aspect_ratio must be nested under metadata.ratio")
}

func TestBuildUpstreamBodyDurationNumberAndMetadataRatio(t *testing.T) {
	profile, ok := silkroad_setting.MatchProfile("seedance-2.0-720")
	require.True(t, ok)

	req := FriendlyRequest{
		Model:          "seedance-2.0-720",
		Prompt:         "hi",
		GenerationType: "text2video",
		DurationValue:  "10",
		AspectRatio:    "9:16",
	}
	data, err := buildUpstreamBody(req, profile, "seedance-2.0-720")
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, common.Unmarshal(data, &body))
	assert.Equal(t, float64(10), body["duration"])
	assert.Equal(t, "720p", body["resolution"])
	assert.Equal(t, "9:16", metadataOf(t, body)["ratio"])
	_, hasSeconds := body["seconds"]
	assert.False(t, hasSeconds)
}

func TestBuildUpstreamBodyMultiImageSetsImages(t *testing.T) {
	profile, ok := silkroad_setting.MatchProfile("dreamina-seedance-2-0-720")
	require.True(t, ok)

	req := FriendlyRequest{
		Model:          "dreamina-seedance-2-0-720-ref",
		Prompt:         "blend",
		GenerationType: "multi_image",
		DurationValue:  "5",
		AspectRatio:    "1:1",
		Images:         []string{"data:image/jpeg;base64,a", "data:image/jpeg;base64,b", "data:image/jpeg;base64,c"},
	}
	data, err := buildUpstreamBody(req, profile, "dreamina-seedance-2-0-720-ref")
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, common.Unmarshal(data, &body))

	refs, ok := body["images"].([]any)
	require.True(t, ok)
	require.Len(t, refs, 3)
	assert.Equal(t, "data:image/jpeg;base64,a", refs[0])
}

func TestBuildUpstreamBodyStartEndSetsMetadataFrames(t *testing.T) {
	profile, ok := silkroad_setting.MatchProfile("dreamina-seedance-2-0-720")
	require.True(t, ok)

	req := FriendlyRequest{
		Model:          "dreamina-seedance-2-0-720-ref",
		Prompt:         "blend",
		GenerationType: "start_end",
		DurationValue:  "5",
		AspectRatio:    "16:9",
		Images:         []string{"data:image/jpeg;base64,first", "data:image/jpeg;base64,last"},
	}
	data, err := buildUpstreamBody(req, profile, "dreamina-seedance-2-0-720-ref")
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, common.Unmarshal(data, &body))
	meta := metadataOf(t, body)
	assert.Equal(t, "data:image/jpeg;base64,first", meta["first_frame"])
	assert.Equal(t, "data:image/jpeg;base64,last", meta["last_frame"])
	_, hasImages := body["images"]
	assert.False(t, hasImages)
	_, hasTopFrame := body["first_frame"]
	assert.False(t, hasTopFrame)
}

func TestBuildUpstreamBodyReferenceAudio(t *testing.T) {
	profile, ok := silkroad_setting.MatchProfile("dreamina-seedance-2-0-720")
	require.True(t, ok)

	req := FriendlyRequest{
		Model:          "dreamina-seedance-2-0-720-ref",
		Prompt:         "with audio",
		GenerationType: "reference_audio",
		DurationValue:  "5",
		AspectRatio:    "16:9",
		Images:         []string{"data:image/jpeg;base64,pic"},
		AudioURL:       "data:audio/mpeg;base64,aud",
	}
	data, err := buildUpstreamBody(req, profile, "dreamina-seedance-2-0-720-ref")
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, common.Unmarshal(data, &body))
	assert.Equal(t, []any{"data:image/jpeg;base64,pic"}, body["images"])
	assert.Equal(t, []any{"data:audio/mpeg;base64,aud"}, metadataOf(t, body)["audios"])
	_, hasAudioURL := body["audio_url"]
	assert.False(t, hasAudioURL)
}

func TestBuildUpstreamBodyReferenceVideos(t *testing.T) {
	profile, ok := silkroad_setting.MatchProfile("dreamina-seedance-2-0-720")
	require.True(t, ok)

	req := FriendlyRequest{
		Model:           "dreamina-seedance-2-0-720-ref",
		Prompt:          "follow @Video1 camera motion",
		GenerationType:  "reference_videos",
		DurationValue:   "5",
		AspectRatio:     "16:9",
		Images:          []string{"data:image/jpeg;base64,pic"},
		ReferenceVideos: []string{"data:video/mp4;base64,vid"},
	}
	data, err := buildUpstreamBody(req, profile, "dreamina-seedance-2-0-720-ref")
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, common.Unmarshal(data, &body))
	assert.Equal(t, []any{"data:image/jpeg;base64,pic"}, body["images"])
	assert.Equal(t, []any{"data:video/mp4;base64,vid"}, metadataOf(t, body)["reference_videos"])
	_, hasTopVideos := body["reference_videos"]
	assert.False(t, hasTopVideos)
}

func TestBuildUpstreamBodySeedanceControlsAndExplicitResolution(t *testing.T) {
	profile, ok := silkroad_setting.MatchProfile("seedance-2-0")
	require.True(t, ok)

	generateAudio := true
	cameraFixed := true
	seed := 42
	req := FriendlyRequest{
		Model:          "seedance-2-0",
		Prompt:         "a cat on a windowsill",
		GenerationType: "text2video",
		DurationValue:  "5",
		AspectRatio:    "16:9",
		Resolution:     "4K",
		GenerateAudio:  &generateAudio,
		CameraFixed:    &cameraFixed,
		Seed:           &seed,
	}
	data, err := buildUpstreamBody(req, profile, "seedance-2-0")
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, common.Unmarshal(data, &body))
	assert.Equal(t, "4k", body["resolution"])
	meta := metadataOf(t, body)
	assert.Equal(t, "16:9", meta["ratio"])
	assert.Equal(t, true, meta["generate_audio"])
	assert.Equal(t, true, meta["camera_fixed"])
	assert.Equal(t, float64(42), meta["seed"])
}
