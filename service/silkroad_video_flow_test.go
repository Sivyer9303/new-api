package service

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSilkRoadVideoFlowPendingIngestReadyOpen exercises the local store path end-to-end
// with a fake CDN only — no live SilkRoad network calls.
func TestSilkRoadVideoFlowPendingIngestReadyOpen(t *testing.T) {
	dir := t.TempDir()
	withSilkRoadStorage(t, dir, "node-a", "https://video.example.com")
	s := silkroad_setting.GetSilkRoadSetting()
	prevRetention := s.Storage.RetentionDays
	prevEnabled := s.Storage.Enabled
	s.Storage.RetentionDays = 7
	s.Storage.Enabled = true
	t.Cleanup(func() {
		s.Storage.RetentionDays = prevRetention
		s.Storage.Enabled = prevEnabled
	})

	const videoBytes = "fake-cdn-mp4-payload"
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(videoBytes))
	}))
	t.Cleanup(cdn.Close)

	taskID := "task_flow_e2e"
	upstreamURL := cdn.URL + "/generated.mp4"
	task := &model.Task{
		TaskID:   taskID,
		Platform: constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeNewAPI)),
	}
	require.True(t, shouldSilkRoadStore(task))

	// pending: public ResultURL only; upstream kept private
	markSilkRoadPendingStore(task, upstreamURL)
	assert.Equal(t, "pending", task.PrivateData.StorageStatus)
	assert.Equal(t, model.TaskStatusStoring, task.Status)
	assert.Equal(t, upstreamURL, task.PrivateData.UpstreamResultURL)
	assert.Equal(t, "https://video.example.com/v1/videos/"+taskID+"/content", task.PrivateData.ResultURL)
	assert.NotContains(t, task.PrivateData.ResultURL, "127.0.0.1")
	assert.NotContains(t, task.PrivateData.ResultURL, upstreamURL)

	before := time.Now().Unix()
	require.NoError(t, ingestOne(task, func(url string) (io.ReadCloser, error) {
		assert.Equal(t, upstreamURL, url)
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			return nil, io.EOF
		}
		return resp.Body, nil
	}))

	// ready: local file written, public URL unchanged
	assert.Equal(t, "ready", task.PrivateData.StorageStatus)
	assert.Equal(t, model.TaskStatus(model.TaskStatusSuccess), task.Status)
	assert.Equal(t, filepath.Join(dir, taskID), task.PrivateData.StoragePath)
	assert.GreaterOrEqual(t, task.PrivateData.StorageExpiresAt, before+7*86400)
	assert.Equal(t, "https://video.example.com/v1/videos/"+taskID+"/content", task.PrivateData.ResultURL)

	f, err := OpenSilkRoadVideoFile(taskID)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	got, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, []byte(videoBytes), got)

	_, err = os.Stat(SilkRoadVideoLocalPath(taskID))
	require.NoError(t, err)
}
