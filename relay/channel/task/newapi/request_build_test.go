package newapi

import (
	"io"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRequestBodyImage2VideoSetsImage(t *testing.T) {
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
	assert.Equal(t, "data:image/jpeg;base64,abc", body["image"])
	assert.Equal(t, "10", body["seconds"])
	assert.Equal(t, "16:9", body["aspect_ratio"])
	_, hasGenType := body["generation_type"]
	assert.False(t, hasGenType, "generation_type must not appear in upstream body")
	_, hasImages := body["images"]
	assert.False(t, hasImages, "friendly images array must not appear for single-image mode")
}

func TestBuildUpstreamBodySecondsString(t *testing.T) {
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
	assert.Equal(t, "10", body["seconds"])
	assert.Equal(t, "9:16", body["aspect_ratio"])
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

func TestBuildUpstreamBodyStartEndSetsFrames(t *testing.T) {
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
	assert.Equal(t, "data:image/jpeg;base64,first", body["first_frame"])
	assert.Equal(t, "data:image/jpeg;base64,last", body["last_frame"])
	_, hasImages := body["images"]
	assert.False(t, hasImages)
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
	assert.Equal(t, "data:image/jpeg;base64,pic", body["image"])
	assert.Equal(t, "data:audio/mpeg;base64,aud", body["audio_url"])
}
