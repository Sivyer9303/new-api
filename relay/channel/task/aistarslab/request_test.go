package aistarslab

import (
	"testing"

	"github.com/QuantumNous/new-api/relay/channel/task/videocommon"
	"github.com/QuantumNous/new-api/setting/aistarslab_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
