package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withSilkRoadLocalDir(t *testing.T, dir string) {
	t.Helper()
	s := silkroad_setting.GetSilkRoadSetting()
	prev := s.Storage.LocalDir
	s.Storage.LocalDir = dir
	t.Cleanup(func() { s.Storage.LocalDir = prev })
}

func silkRoadContentRequest(t *testing.T, task *model.Task) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/"+task.TaskID+"/content", nil)
	serveSilkRoadVideoContent(c, task)
	return w
}

func TestServeSilkRoadVideoContent_ReadyServesLocalFile(t *testing.T) {
	dir := t.TempDir()
	withSilkRoadLocalDir(t, dir)

	taskID := "task_ready_serve"
	payload := []byte("fake-mp4-bytes")
	_, _, err := service.WriteSilkRoadVideoFile(taskID, bytes.NewReader(payload))
	require.NoError(t, err)

	task := &model.Task{
		TaskID: taskID,
		Status: model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			StorageStatus:     "ready",
			UpstreamResultURL: "https://upstream.example/secret.mp4",
			ResultURL:         "https://video.example.com/v1/videos/" + taskID + "/content",
			StorageExpiresAt:  time.Now().Unix() + 86400,
		},
	}

	w := silkRoadContentRequest(t, task)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "video/mp4", w.Header().Get("Content-Type"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, payload, w.Body.Bytes())
	assert.NotContains(t, w.Body.String(), "upstream.example")
	assert.NotContains(t, w.Header().Get("Location"), "upstream.example")
}

func TestServeSilkRoadVideoContent_RejectsActiveContentType(t *testing.T) {
	dir := t.TempDir()
	withSilkRoadLocalDir(t, dir)
	taskID := "task_active_content"
	_, _, err := service.WriteSilkRoadVideoFile(
		taskID,
		bytes.NewReader([]byte("<script>alert(1)</script>")),
	)
	require.NoError(t, err)

	task := &model.Task{
		TaskID: taskID,
		Status: model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			StorageStatus:      "ready",
			StorageContentType: "text/html; charset=utf-8",
			StorageExpiresAt:   time.Now().Add(time.Hour).Unix(),
		},
	}

	w := silkRoadContentRequest(t, task)
	assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)
	assert.NotEqual(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestServeSilkRoadVideoContent_PendingConflict(t *testing.T) {
	withSilkRoadLocalDir(t, t.TempDir())
	task := &model.Task{
		TaskID: "task_pending",
		Status: model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			StorageStatus:     "pending",
			UpstreamResultURL: "https://upstream.example/secret.mp4",
		},
	}
	w := silkRoadContentRequest(t, task)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "still being processed")
	assert.NotContains(t, w.Body.String(), "upstream.example")
}

func TestServeSilkRoadVideoContent_ExpiredGone(t *testing.T) {
	withSilkRoadLocalDir(t, t.TempDir())
	task := &model.Task{
		TaskID: "task_expired",
		Status: model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			StorageStatus:    "expired",
			StorageExpiresAt: time.Now().Unix() - 10,
		},
	}
	w := silkRoadContentRequest(t, task)
	assert.Equal(t, http.StatusGone, w.Code)
	assert.Contains(t, w.Body.String(), "expired")
}

func TestServeSilkRoadVideoContent_ReadyPastExpiresAtGone(t *testing.T) {
	dir := t.TempDir()
	withSilkRoadLocalDir(t, dir)
	taskID := "task_ttl_expired"
	_, _, err := service.WriteSilkRoadVideoFile(taskID, bytes.NewReader([]byte("x")))
	require.NoError(t, err)

	task := &model.Task{
		TaskID: taskID,
		Status: model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			StorageStatus:    "ready",
			StorageExpiresAt: time.Now().Unix() - 1,
		},
	}
	w := silkRoadContentRequest(t, task)
	assert.Equal(t, http.StatusGone, w.Code)
}

func TestServeSilkRoadVideoContent_ReadyMissingFileNotFound(t *testing.T) {
	withSilkRoadLocalDir(t, t.TempDir())
	task := &model.Task{
		TaskID: "task_missing_file",
		Status: model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			StorageStatus:    "ready",
			StorageExpiresAt: time.Now().Unix() + 3600,
		},
	}
	w := silkRoadContentRequest(t, task)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestServeSilkRoadVideoContent_FailedNeverProxiesUpstream(t *testing.T) {
	withSilkRoadLocalDir(t, t.TempDir())
	task := &model.Task{
		TaskID: "task_failed_store",
		Status: model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			StorageStatus:     "failed",
			UpstreamResultURL: "https://upstream.example/secret.mp4",
		},
	}
	w := silkRoadContentRequest(t, task)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.NotContains(t, w.Body.String(), "upstream.example")
	assert.Empty(t, w.Header().Get("Location"))
}
