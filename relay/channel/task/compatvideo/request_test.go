package compatvideo

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/relay/channel/task/videocommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/compatvideo_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRequestAcceptsNormalizedGrokPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	info := &relaycommon.RelayInfo{
		ChannelMeta:   &relaycommon.ChannelMeta{UpstreamModelName: "grok-image-video"},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
	}

	request, profile, err := parseRequest([]byte(`{
		"model":"grok-image-video",
		"prompt":"A cat walks",
		"generation_type":"image2video",
		"duration":8,
		"aspect_ratio":"16:9",
		"resolution":"720p",
		"media":[{"type":"image","role":"reference","source":"data:image/png;base64,abc"}]
	}`), info)
	require.NoError(t, err)
	assert.Equal(t, compatvideo_setting.ProfileGrokImageVideo, profile.ID)
	require.NotNil(t, request.Duration)
	assert.Equal(t, 8, *request.Duration)
	require.Len(t, request.Media, 1)
}

func TestParseRequestRejectsGrokTextOnlyOnVideo15(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta:   &relaycommon.ChannelMeta{UpstreamModelName: "grok-video-1.5"},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
	}
	_, _, err := parseRequest([]byte(`{
		"model":"grok-video-1.5",
		"prompt":"A cat walks",
		"generation_type":"text2video",
		"duration":8,
		"aspect_ratio":"16:9"
	}`), info)
	require.Error(t, err)
}

func TestSerializeGrokUsesNumericSecondsAndImageURLs(t *testing.T) {
	duration := 8
	profile := compatvideo_setting.MatchProfile("grok-image-video")
	payload, err := serializeUpstreamRequest(profile, videocommon.VideoGenerateRequest{
		Model:       "grok-image-video",
		Prompt:      "hello",
		Duration:    &duration,
		AspectRatio: "16:9",
		Resolution:  "720p",
	}, []videocommon.VideoMedia{{
		Type:   videocommon.VideoMediaImage,
		Source: "https://cdn.example/a.png",
	}}, []string{"https://cdn.example/a.png"})
	require.NoError(t, err)
	assert.Equal(t, 8, payload["seconds"])
	assert.Equal(t, []string{"https://cdn.example/a.png"}, payload["image_urls"])
	_, hasInput := payload["input_reference"]
	_, hasRefs := payload["reference_images"]
	assert.False(t, hasInput)
	assert.False(t, hasRefs)
}

func TestSerializeSeedanceUsesOpenAIVideosFields(t *testing.T) {
	duration := 6
	generateAudio := true
	profile := compatvideo_setting.MatchProfile("seedance-2-0")
	payload, err := serializeUpstreamRequest(profile, videocommon.VideoGenerateRequest{
		Model:         "seedance-2-0",
		Prompt:        "hello",
		Duration:      &duration,
		AspectRatio:   "16:9",
		Resolution:    "720p",
		GenerateAudio: &generateAudio,
	}, []videocommon.VideoMedia{{
		Type:   videocommon.VideoMediaImage,
		Source: "https://cdn.example/a.png",
	}}, []string{"https://cdn.example/a.png"})
	require.NoError(t, err)
	assert.Equal(t, "6", payload["seconds"])
	assert.Equal(t, true, payload["generate_audio"])
	content, ok := payload["content"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, content, 1)
	assert.Equal(t, "https://cdn.example/a.png", content[0]["image_url"])
}

func TestValidateRequestAndSetActionStoresDialect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	body := bytes.NewBufferString(`{
		"model":"seedance-2-0",
		"prompt":"hello",
		"generation_type":"text2video",
		"duration":8,
		"aspect_ratio":"16:9",
		"resolution":"720p"
	}`)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", body)
	context.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		ChannelMeta:   &relaycommon.ChannelMeta{UpstreamModelName: "seedance-2-0"},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{PublicTaskID: "task_test"},
	}

	adaptor := &TaskAdaptor{}
	err := adaptor.ValidateRequestAndSetAction(context, info)
	require.Nil(t, err)
	assert.Equal(t, compatvideo_setting.DialectOpenAIVideos, adaptor.dialect)
}
