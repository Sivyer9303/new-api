package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldSilkRoadStore(t *testing.T) {
	s := silkroad_setting.GetSilkRoadSetting()
	prev := s.Storage
	t.Cleanup(func() { s.Storage = prev })

	newAPITask := &model.Task{
		Platform: constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeNewAPI)),
	}
	otherTask := &model.Task{Platform: "openai"}

	s.Storage.Enabled = true
	s.Storage.IngestNodeName = "node-a"
	s.Storage.PublicDownloadBaseURL = "https://video.example.com"
	assert.True(t, shouldSilkRoadStore(newAPITask))
	assert.False(t, shouldSilkRoadStore(otherTask))
	assert.False(t, shouldSilkRoadStore(nil))

	s.Storage.IngestNodeName = ""
	assert.False(t, shouldSilkRoadStore(newAPITask))

	s.Storage.IngestNodeName = "node-a"
	s.Storage.PublicDownloadBaseURL = ""
	assert.False(t, shouldSilkRoadStore(newAPITask))

	s.Storage.PublicDownloadBaseURL = "https://video.example.com"
	s.Storage.Enabled = false
	assert.False(t, shouldSilkRoadStore(newAPITask))
}

func TestMarkSilkRoadPendingStoreKeepsUpstreamPrivate(t *testing.T) {
	withSilkRoadStorage(t, t.TempDir(), "", "https://video.example.com")

	task := &model.Task{TaskID: "task_store_1"}
	upstream := "https://cdn.upstream.example/raw.mp4"
	markSilkRoadPendingStore(task, upstream)

	assert.Equal(t, upstream, task.PrivateData.UpstreamResultURL)
	assert.Equal(t, "pending", task.PrivateData.StorageStatus)
	assert.Equal(t, 0, task.PrivateData.StorageRetryCount)
	assert.Equal(t, "https://video.example.com/v1/videos/task_store_1/content", task.PrivateData.ResultURL)
	assert.NotContains(t, task.PrivateData.ResultURL, "upstream")
	assert.NotEqual(t, upstream, task.PrivateData.ResultURL)
}

func TestRedactSilkRoadUpstreamURLsStripsVideoURL(t *testing.T) {
	withSilkRoadStorage(t, t.TempDir(), "node-a", "https://video.example.com")

	upstream := "https://cdn.upstream.example/raw.mp4"
	body := []byte(`{"id":"cgt-1","status":"completed","progress":100,"video_url":"https://cdn.upstream.example/raw.mp4","data":{"url":"https://cdn.upstream.example/nested.mp4","result_url":"https://cdn.upstream.example/result.mp4"}}`)

	task := &model.Task{TaskID: "task_redact_1", Data: body}
	markSilkRoadPendingStore(task, upstream)
	redacted, err := redactSilkRoadUpstreamURLs(task.Data)
	require.NoError(t, err)
	task.Data = redacted

	assert.NotContains(t, string(task.Data), "cdn.upstream.example")
	assert.NotContains(t, string(task.Data), "video_url")
	assert.NotContains(t, string(task.Data), `"url"`)
	assert.NotContains(t, string(task.Data), "result_url")
	assert.Contains(t, string(task.Data), `"id":"cgt-1"`)
	assert.Equal(t, "https://video.example.com/v1/videos/task_redact_1/content", task.PrivateData.ResultURL)
	assert.Equal(t, upstream, task.PrivateData.UpstreamResultURL)
}

func TestIngestOneSuccessWritesLocalFile(t *testing.T) {
	dir := t.TempDir()
	withSilkRoadStorage(t, dir, "node-a", "https://video.example.com")
	s := silkroad_setting.GetSilkRoadSetting()
	prevRetention := s.Storage.RetentionDays
	s.Storage.RetentionDays = 7
	t.Cleanup(func() { s.Storage.RetentionDays = prevRetention })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("fake-mp4-bytes"))
	}))
	t.Cleanup(srv.Close)

	task := &model.Task{
		TaskID: "task_ingest_ok",
		PrivateData: model.TaskPrivateData{
			UpstreamResultURL: srv.URL + "/video.mp4",
			StorageStatus:     "pending",
			ResultURL:         BuildSilkRoadPublicURL("task_ingest_ok"),
		},
	}

	before := time.Now().Unix()
	require.NoError(t, ingestOne(task, func(url string) (io.ReadCloser, error) {
		assert.Equal(t, task.PrivateData.UpstreamResultURL, url)
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		return resp.Body, nil
	}))

	assert.Equal(t, "ready", task.PrivateData.StorageStatus)
	assert.Equal(t, filepath.Join(dir, "task_ingest_ok"), task.PrivateData.StoragePath)
	assert.GreaterOrEqual(t, task.PrivateData.StorageExpiresAt, before+7*86400)
	assert.LessOrEqual(t, task.PrivateData.StorageExpiresAt, time.Now().Unix()+7*86400)
	assert.Equal(t, "https://video.example.com/v1/videos/task_ingest_ok/content", task.PrivateData.ResultURL)

	got, err := os.ReadFile(SilkRoadVideoLocalPath("task_ingest_ok"))
	require.NoError(t, err)
	assert.Equal(t, []byte("fake-mp4-bytes"), got)
}

func TestIngestOneFailureIncrementsRetryNeverRefunds(t *testing.T) {
	dir := t.TempDir()
	withSilkRoadStorage(t, dir, "node-a", "https://video.example.com")
	s := silkroad_setting.GetSilkRoadSetting()
	prevMax := s.Storage.MaxRetry
	s.Storage.MaxRetry = 3
	t.Cleanup(func() { s.Storage.MaxRetry = prevMax })

	task := &model.Task{
		Quota:  12345,
		TaskID: "task_ingest_fail",
		PrivateData: model.TaskPrivateData{
			UpstreamResultURL: "https://cdn.example/missing.mp4",
			StorageStatus:     "pending",
			StorageRetryCount: 0,
			ResultURL:         BuildSilkRoadPublicURL("task_ingest_fail"),
		},
	}
	quotaBefore := task.Quota

	err := ingestOne(task, func(string) (io.ReadCloser, error) {
		return nil, errors.New("download refused")
	})
	require.Error(t, err)
	assert.Equal(t, 1, task.PrivateData.StorageRetryCount)
	assert.Equal(t, "pending", task.PrivateData.StorageStatus)
	assert.Equal(t, quotaBefore, task.Quota)
	assert.Equal(t, "https://video.example.com/v1/videos/task_ingest_fail/content", task.PrivateData.ResultURL)
	_, statErr := os.Stat(SilkRoadVideoLocalPath("task_ingest_fail"))
	assert.True(t, os.IsNotExist(statErr))

	task.PrivateData.StorageRetryCount = 2
	err = ingestOne(task, func(string) (io.ReadCloser, error) {
		return nil, errors.New("still failing")
	})
	require.Error(t, err)
	assert.Equal(t, 3, task.PrivateData.StorageRetryCount)
	assert.Equal(t, "failed", task.PrivateData.StorageStatus)
	assert.Equal(t, quotaBefore, task.Quota)
}

func TestRunSilkRoadVideoIngestOnceSkipsNonIngestNode(t *testing.T) {
	prevNode := common.NodeName
	t.Cleanup(func() { common.NodeName = prevNode })

	withSilkRoadStorage(t, t.TempDir(), "node-a", "https://video.example.com")
	common.NodeName = "node-b"

	require.NoError(t, RunSilkRoadVideoIngestOnce(context.Background()))
}

func TestIngestOneUsesInjectedBodyReader(t *testing.T) {
	dir := t.TempDir()
	withSilkRoadStorage(t, dir, "", "https://video.example.com")

	task := &model.Task{
		TaskID: "task_reader",
		PrivateData: model.TaskPrivateData{
			UpstreamResultURL: "https://cdn.example/v.mp4",
			StorageStatus:     "pending",
		},
	}
	require.NoError(t, ingestOne(task, func(string) (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader([]byte("abc"))), nil
	}))
	assert.Equal(t, "ready", task.PrivateData.StorageStatus)
	got, err := os.ReadFile(SilkRoadVideoLocalPath("task_reader"))
	require.NoError(t, err)
	assert.Equal(t, []byte("abc"), got)
}
