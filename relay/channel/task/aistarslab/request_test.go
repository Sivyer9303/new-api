package aistarslab

import (
	"bytes"
	stdcontext "context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/relay/channel/task/videocommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/setting/aistarslab_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func aiStarsLabTestImageDataURL(t *testing.T) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	var payload bytes.Buffer
	require.NoError(t, png.Encode(&payload, img))
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(payload.Bytes())
}

func TestSerializeUpstreamRequestUsesAIStarsLabMetadata(t *testing.T) {
	duration := 4
	payload, err := serializeUpstreamRequest(videocommon.VideoGenerateRequest{
		Model:          "test:test-video",
		Prompt:         "海边日落，镜头缓慢向前推进",
		GenerationType: aistarslab_setting.GenerationImage2Video,
		Duration:       &duration,
		AspectRatio:    "16:9",
		Resolution:     "720p",
	}, []videocommon.VideoMedia{
		{Type: videocommon.VideoMediaImage, Role: videocommon.VideoMediaRoleReference, Source: "https://cdn.example/a.png"},
		{Type: videocommon.VideoMediaAudio, Source: "https://cdn.example/a.mp3"},
	}, []string{"https://cdn.example/a.png", "https://cdn.example/a.mp3"})
	require.NoError(t, err)
	assert.Equal(t, "test:test-video", payload["model"])
	assert.Equal(t, "4", payload["seconds"])
	assert.Equal(t, "16:9", payload["size"])
	assert.Equal(t, 1, payload["n"])
	metadata, ok := payload["metadata"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "720p", metadata["resolution"])
	assert.Equal(t, aistarslab_setting.GenerationImage2Video, metadata["mode_type"])
	assert.Equal(t, []string{"https://cdn.example/a.png"}, metadata["images"])
	assert.Equal(t, []string{"https://cdn.example/a.mp3"}, metadata["audios"])
}

func TestSerializeUpstreamRequestOrdersFirstLastFrames(t *testing.T) {
	duration := 5
	payload, err := serializeUpstreamRequest(videocommon.VideoGenerateRequest{
		Model:          "12:example-video-model",
		Prompt:         "from start to end",
		GenerationType: aistarslab_setting.GenerationFrames2Video,
		Duration:       &duration,
		AspectRatio:    "9:16",
	}, []videocommon.VideoMedia{
		{Type: videocommon.VideoMediaImage, Role: videocommon.VideoMediaRoleLastFrame, Source: "https://cdn.example/last.png"},
		{Type: videocommon.VideoMediaImage, Role: videocommon.VideoMediaRoleFirstFrame, Source: "https://cdn.example/first.png"},
	}, []string{"https://cdn.example/last.png", "https://cdn.example/first.png"})
	require.NoError(t, err)
	metadata, ok := payload["metadata"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, aistarslab_setting.GenerationFrames2Video, metadata["mode_type"])
	assert.Equal(t, []string{"https://cdn.example/first.png", "https://cdn.example/last.png"}, metadata["images"])
}

func TestParseRequestAcceptsVideoToolAndOpenAIVideosPayloads(t *testing.T) {
	request, err := parseRequestForPath("/v1/video/generations", []byte(`{
		"model":"test:test-video",
		"prompt":"海边日落，镜头缓慢向前推进",
		"generation_type":"text2video",
		"duration":4,
		"aspect_ratio":"16:9",
		"resolution":"720p"
	}`), nil)
	require.NoError(t, err)
	require.NotNil(t, request.Duration)
	assert.Equal(t, 4, *request.Duration)
	assert.Equal(t, "16:9", request.AspectRatio)
	assert.Equal(t, aistarslab_setting.GenerationText2Video, request.GenerationType)

	request, err = parseRequestForPath("/v1/videos", []byte(`{
		"model":"test:test-video",
		"prompt":"测试视频生成接口",
		"seconds":"5",
		"size":"16:9",
		"n":1,
		"metadata":{"mode_type":"image2video","images":["https://example.com/reference.jpg"]}
	}`), nil)
	require.NoError(t, err)
	require.NotNil(t, request.Duration)
	assert.Equal(t, 5, *request.Duration)
	assert.Equal(t, "16:9", request.AspectRatio)
	assert.Equal(t, aistarslab_setting.GenerationImage2Video, request.GenerationType)
	require.Len(t, request.Media, 1)
	assert.Equal(t, "https://example.com/reference.jpg", request.Media[0].Source)
}

func TestParseRequestRejectsText2VideoWithImages(t *testing.T) {
	_, err := parseRequestForPath("/v1/video/generations", []byte(`{
		"model":"test:test-video",
		"prompt":"no images",
		"generation_type":"text2video",
		"duration":4,
		"media":[{"type":"image","source":"https://example.com/a.png"}]
	}`), nil)
	require.Error(t, err)
}

func TestBuildRequestBodyStagesInlineMediaWithoutChannelR2Setting(t *testing.T) {
	imageDataURL := aiStarsLabTestImageDataURL(t)
	videoDataURL := "data:video/mp4;base64," + base64.StdEncoding.EncodeToString([]byte("fake-mp4"))
	gin.SetMode(gin.TestMode)
	request := httptest.NewRequest(http.MethodPost, "/v1/video/generations", strings.NewReader(fmt.Sprintf(`{
		"model":"48:seedance-2.0-fast",
		"prompt":"让橘猫像是被吓一跳一样跳起来",
		"generation_type":"image2video",
		"duration":4,
		"resolution":"480p",
		"aspect_ratio":"16:9",
		"media":[
			{"type":"image","role":"reference","source":"%s"},
			{"type":"video","role":"reference","source":"%s"}
		]
	}`, imageDataURL, videoDataURL)))
	request.Header.Set("Content-Type", "application/json")
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = request
	info := &relaycommon.RelayInfo{
		OriginModelName: "seedance-2-0-fast",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:         6,
			ChannelType:       64,
			ChannelBaseUrl:    "https://api.video.aistarslab.com/openai",
			ApiKey:            "secret-key",
			UpstreamModelName: "48:seedance-2.0-fast",
			ChannelSetting: dto.ChannelSettings{
				VideoInputMediaDelivery: dto.VideoInputMediaInlineBase64,
			},
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{PublicTaskID: "task_public"},
	}

	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(context, info))

	staged := make([]string, 0, 2)
	adaptor.stageInput = func(_ stdcontext.Context, channelID int, media string) (string, error) {
		assert.Equal(t, 6, channelID)
		assert.True(t, strings.HasPrefix(media, "data:"))
		url := fmt.Sprintf("https://r2.example/%d.bin", len(staged))
		staged = append(staged, url)
		return url, nil
	}

	reader, err := adaptor.BuildRequestBody(context, info)
	require.NoError(t, err)
	got, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Len(t, staged, 2)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(got, &payload))
	metadata, ok := payload["metadata"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []any{"https://r2.example/0.bin"}, metadata["images"])
	assert.Equal(t, []any{"https://r2.example/1.bin"}, metadata["videos"])
	assert.NotContains(t, string(got), "data:")
}
