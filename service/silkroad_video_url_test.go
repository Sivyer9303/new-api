package service

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractUpstreamVideoURLFromJSONNestedTaskDto(t *testing.T) {
	withSilkRoadStorage(t, t.TempDir(), "node-a", "http://localhost:3000")
	body := []byte(`{"code":"success","data":{"status":"SUCCESS","result_url":"http://localhost:3000/v1/videos/task_x/content","data":{"video_url":"https://cdn.upstream.example/a.mp4","url":"https://cdn.upstream.example/a.mp4"}}}`)
	assert.Equal(t, "https://cdn.upstream.example/a.mp4", ExtractUpstreamVideoURLFromJSON(body))
}

func TestExtractUpstreamVideoURLFromJSONSkipsProxyOnly(t *testing.T) {
	withSilkRoadStorage(t, t.TempDir(), "node-a", "http://127.0.0.1:8080")
	body := []byte(`{"result_url":"http://127.0.0.1:8080/v1/videos/task_x/content"}`)
	assert.Empty(t, ExtractUpstreamVideoURLFromJSON(body))
}

func TestExtractUpstreamVideoURLFromJSONKeepsExternalContentShapedURL(t *testing.T) {
	withSilkRoadStorage(t, t.TempDir(), "node-a", "https://video.example.com")
	const upstream = "https://upstream.example/v1/videos/provider-task/content"
	body := []byte(`{"result_url":"` + upstream + `"}`)
	assert.Equal(t, upstream, ExtractUpstreamVideoURLFromJSON(body))
}

func TestApplySilkRoadSuccessStoreQueuesIngestAndRedacts(t *testing.T) {
	withSilkRoadStorage(t, t.TempDir(), "node-a", "https://video.example.com")
	s := silkroad_setting.GetSilkRoadSetting()
	prev := s.Storage.Enabled
	s.Storage.Enabled = true
	t.Cleanup(func() { s.Storage.Enabled = prev })

	task := &model.Task{
		TaskID:   "task_hide_upstream",
		Platform: constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeNewAPI)),
		Data:     []byte(`{"code":"success","data":{"video_url":"https://cdn.upstream.example/leak.mp4","status":"SUCCESS"}}`),
	}
	require.True(t, shouldSilkRoadStore(task))

	applySilkRoadSuccessStore(task, "", []byte(`{"code":"success","data":{"data":{"video_url":"https://cdn.upstream.example/leak.mp4"}}}`))

	assert.Equal(t, "pending", task.PrivateData.StorageStatus)
	assert.Equal(t, "https://cdn.upstream.example/leak.mp4", task.PrivateData.UpstreamResultURL)
	assert.Equal(t, "https://video.example.com/v1/videos/task_hide_upstream/content", task.PrivateData.ResultURL)
	assert.NotContains(t, string(task.Data), "cdn.upstream.example")
	assert.NotContains(t, string(task.Data), "video_url")
}

func TestSanitizeTaskForClientHidesUpstream(t *testing.T) {
	withSilkRoadStorage(t, t.TempDir(), "node-a", "https://video.example.com")
	task := &model.Task{
		TaskID:   "task_sanitize",
		Platform: constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeNewAPI)),
		Data:     []byte(`{"video_url":"https://cdn.upstream.example/leak.mp4"}`),
		PrivateData: model.TaskPrivateData{
			StorageStatus:     "pending",
			ResultURL:         "https://video.example.com/v1/videos/task_sanitize/content",
			UpstreamResultURL: "https://cdn.upstream.example/leak.mp4",
		},
	}
	resultURL, data := SanitizeTaskForClient(task)
	assert.Empty(t, resultURL)
	assert.NotContains(t, string(data), "cdn.upstream.example")
}

func TestSanitizeTaskForClientHidesMarkedFutureProviderVideoWhenStorageIsDisabled(t *testing.T) {
	withSilkRoadStorage(t, t.TempDir(), "", "")
	task := &model.Task{
		TaskID:   "future-provider-video",
		Platform: constant.TaskPlatform("future-provider"),
		Status:   model.TaskStatusSuccess,
		Data:     []byte(`{"video_url":"https://upstream.example/private.mp4"}`),
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://upstream.example/private.mp4",
			VideoTask: true,
		},
	}

	resultURL, data := SanitizeTaskForClient(task)
	assert.Empty(t, resultURL)
	assert.NotContains(t, string(data), "upstream.example")
}

func TestIsVideoTaskUsesMarkerStorageMetadataAndLegacyFallbacks(t *testing.T) {
	tests := []struct {
		name string
		task *model.Task
		want bool
	}{
		{
			name: "provider-neutral marker",
			task: &model.Task{
				Platform:    constant.TaskPlatform("future-provider"),
				PrivateData: model.TaskPrivateData{VideoTask: true},
			},
			want: true,
		},
		{
			name: "persisted storage metadata",
			task: &model.Task{
				Platform: constant.TaskPlatform("future-provider"),
				PrivateData: model.TaskPrivateData{
					StorageStatus: "pending",
				},
			},
			want: true,
		},
		{
			name: "legacy NewAPI platform",
			task: &model.Task{
				Platform: constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeNewAPI)),
			},
			want: true,
		},
		{
			name: "legacy SilkRoad platform",
			task: &model.Task{
				Platform: constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeSilkRoad)),
			},
			want: true,
		},
		{
			name: "legacy named provider",
			task: &model.Task{
				Platform: constant.TaskPlatform("kling"),
			},
			want: true,
		},
		{
			name: "legacy video action",
			task: &model.Task{
				Platform: constant.TaskPlatform("legacy-provider"),
				Action:   constant.TaskActionGenerate,
			},
			want: true,
		},
		{
			name: "unmarked unrelated task",
			task: &model.Task{
				Platform: constant.TaskPlatform("future-provider"),
				Action:   "unrelated",
			},
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, IsVideoTask(test.task))
		})
	}
}

func TestVideoDataRedactionRemovesURLsAndInlineMediaFromNestedMessages(t *testing.T) {
	body := []byte(`{
		"id":"provider-id",
		"error":{"message":"failed to fetch https://signed.example/input.png?signature=secret"},
		"debug":["data:image/png;base64,c2VjcmV0"],
		"status":"failed"
	}`)

	redacted, err := applyVideoDataRedaction(body, "task_public")

	require.NoError(t, err)
	assert.Contains(t, string(redacted), `"id":"task_public"`)
	assert.NotContains(t, string(redacted), "signed.example")
	assert.NotContains(t, string(redacted), "data:image")
	assert.NotContains(t, string(redacted), "c2VjcmV0")
}
