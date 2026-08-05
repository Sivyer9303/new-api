package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
)

func TestExtractStoredVideoURLPrefersDataVideoURL(t *testing.T) {
	task := &model.Task{
		Data: []byte(`{"status":"completed","video_url":"https://images.example/a.mp4","url":"https://images.example/a.mp4"}`),
		PrivateData: model.TaskPrivateData{
			ResultURL: "http://localhost:3000/v1/videos/task_abc/content",
		},
	}
	assert.Equal(t, "https://images.example/a.mp4", extractStoredVideoURL(task))
}

func TestExtractStoredVideoURLFromNestedTaskDtoPayload(t *testing.T) {
	task := &model.Task{
		Data: []byte(`{"code":"success","data":{"status":"SUCCESS","result_url":"http://localhost:3000/v1/videos/task_abc/content","data":{"video_url":"https://images.silkroad.example/a.mp4","url":"https://images.silkroad.example/a.mp4"}}}`),
		PrivateData: model.TaskPrivateData{
			ResultURL: "http://localhost:3000/v1/videos/task_abc/content",
		},
	}
	assert.Equal(t, "https://images.silkroad.example/a.mp4", extractStoredVideoURL(task))
}

func TestExtractStoredVideoURLSkipsProxyResultURL(t *testing.T) {
	task := &model.Task{
		PrivateData: model.TaskPrivateData{
			ResultURL: "http://localhost:3000/v1/videos/task_abc/content",
		},
		FailReason: "",
	}
	assert.Empty(t, extractStoredVideoURL(task))
}

func TestExtractStoredVideoURLUsesUpstreamPrivate(t *testing.T) {
	task := &model.Task{
		PrivateData: model.TaskPrivateData{
			UpstreamResultURL: "https://cdn.example/u.mp4",
			ResultURL:         "http://localhost:3000/v1/videos/task_abc/content",
		},
	}
	assert.Equal(t, "https://cdn.example/u.mp4", extractStoredVideoURL(task))
}

func TestIsLocalVideoContentProxyURL(t *testing.T) {
	assert.True(t, isLocalVideoContentProxyURL("http://localhost:3000/v1/videos/task_x/content"))
	assert.False(t, isLocalVideoContentProxyURL("https://images.example/seedance-video/x.mp4"))
}
