package controller

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/assert"
)

func withPresignedVideoDelivery(
	t *testing.T,
	signedURL string,
	presignErr error,
) {
	t.Helper()
	previous := presignStoredVideoURL
	presignStoredVideoURL = func(context.Context, *model.Task) (string, bool, error) {
		return signedURL, true, presignErr
	}
	t.Cleanup(func() { presignStoredVideoURL = previous })
}

func readyStoredVideoTask(taskID string) *model.Task {
	return &model.Task{
		TaskID: taskID,
		Status: model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			StorageStatus:     "ready",
			StorageObjectKey:  "videos/" + taskID,
			UpstreamResultURL: "https://upstream.example/secret.mp4",
			StorageExpiresAt:  time.Now().Unix() + 86400,
		},
	}
}

func TestServeStoredVideoContent_RedirectsToPresignedURL(t *testing.T) {
	signed := "https://acct.r2.cloudflarestorage.com/videos/videos/task_r2?X-Amz-Expires=900&X-Amz-Signature=abc"
	withPresignedVideoDelivery(t, signed, nil)

	w := silkRoadContentRequest(t, readyStoredVideoTask("task_r2"))

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, signed, w.Header().Get("Location"))
	assert.Equal(t, "private, no-store", w.Header().Get("Cache-Control"))
	assert.NotContains(t, w.Header().Get("Location"), "upstream.example")
	assert.NotContains(t, w.Body.String(), "upstream.example")
}

func TestServeStoredVideoContent_PresignFailureDoesNotLeakUpstream(t *testing.T) {
	withPresignedVideoDelivery(t, "", errors.New("signing key rejected"))

	w := silkRoadContentRequest(t, readyStoredVideoTask("task_r2_broken"))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Empty(t, w.Header().Get("Location"))
	assert.NotContains(t, w.Body.String(), "upstream.example")
	assert.NotContains(t, w.Body.String(), "signing key rejected")
}

func TestServeStoredVideoContent_ExpiredTaskIsNeverRedirected(t *testing.T) {
	withPresignedVideoDelivery(t, "https://acct.r2.cloudflarestorage.com/videos/expired", nil)

	task := readyStoredVideoTask("task_r2_expired")
	task.PrivateData.StorageExpiresAt = time.Now().Unix() - 1

	w := silkRoadContentRequest(t, task)

	assert.Equal(t, http.StatusGone, w.Code)
	assert.Empty(t, w.Header().Get("Location"))
}
