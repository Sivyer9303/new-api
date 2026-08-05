package newapi

import (
	"io"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRequestBodyImage2VideoSetsImageURL(t *testing.T) {
	a := &TaskAdaptor{}
	c, info := newTestContext(t, `{
		"model":"seedance-2.0-720",
		"prompt":"animate this",
		"generation_type":"image2video",
		"seconds":"10",
		"aspect_ratio":"16:9",
		"images":["https://cdn.example/a.png"]
	}`)

	taskErr := a.ValidateRequestAndSetAction(c, info)
	require.Nil(t, taskErr)

	reader, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	data, err := io.ReadAll(reader)
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, common.Unmarshal(data, &body))

	assert.Equal(t, "seedance-2.0-720", body["model"])
	assert.Equal(t, "animate this", body["prompt"])
	assert.Equal(t, "https://cdn.example/a.png", body["image_url"])
	assert.Equal(t, "10", body["seconds"])
	assert.Equal(t, "16:9", body["aspect_ratio"])
	_, hasGenType := body["generation_type"]
	assert.False(t, hasGenType, "generation_type must not appear in upstream body")
	_, hasImages := body["images"]
	assert.False(t, hasImages, "friendly images must not appear in upstream body")
}

func TestBuildUpstreamBodyDurationAsNumber(t *testing.T) {
	profile, ok := silkroad_setting.MatchProfile("dreamina-seedance-2-0-720")
	require.True(t, ok)

	req := FriendlyRequest{
		Model:          "dreamina-seedance-2-0-720",
		Prompt:         "hi",
		GenerationType: "text2video",
		DurationValue:  "5",
		AspectRatio:    "9:16",
	}
	data, err := buildUpstreamBody(req, profile, "dreamina-seedance-2-0-720")
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, common.Unmarshal(data, &body))
	assert.Equal(t, float64(5), body["duration"])
	assert.Equal(t, "9:16", body["aspect_ratio"])
}

func TestBuildUpstreamBodyMultiImageSetsReferenceURLs(t *testing.T) {
	profile, ok := silkroad_setting.MatchProfile("seedance-2.0-720")
	require.True(t, ok)

	req := FriendlyRequest{
		Model:          "seedance-2.0-720",
		Prompt:         "blend",
		GenerationType: "multi_image",
		DurationValue:  "15",
		AspectRatio:    "1:1",
		Images:         []string{"https://a.png", "https://b.png", "https://c.png"},
	}
	data, err := buildUpstreamBody(req, profile, "seedance-2.0-720")
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, common.Unmarshal(data, &body))

	refs, ok := body["reference_image_urls"].([]any)
	require.True(t, ok)
	require.Len(t, refs, 3)
	assert.Equal(t, "https://a.png", refs[0])

	vc, ok := body["video_config"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "auto", vc["reference_mode"])
}

func TestBuildUpstreamBodyStartEndExtraDoesNotClobberReferenceMode(t *testing.T) {
	profile, ok := silkroad_setting.MatchProfile("seedance-2.0-720")
	require.True(t, ok)

	req := FriendlyRequest{
		Model:          "seedance-2.0-720",
		Prompt:         "blend",
		GenerationType: "start_end",
		DurationValue:  "10",
		AspectRatio:    "16:9",
		Images:         []string{"https://a.png", "https://b.png"},
		Extras: map[string]string{
			"video_config.reference_mode": "auto",
		},
	}
	data, err := buildUpstreamBody(req, profile, "seedance-2.0-720")
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, common.Unmarshal(data, &body))

	vc, ok := body["video_config"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "start_end", vc["reference_mode"], "recipe UpstreamSets must win over ExtraOptions")

	refs, ok := body["reference_image_urls"].([]any)
	require.True(t, ok)
	require.Len(t, refs, 2)
}
