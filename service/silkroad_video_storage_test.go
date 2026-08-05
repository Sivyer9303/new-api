package service

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withSilkRoadStorage(t *testing.T, localDir, ingestNode, publicBase string) {
	t.Helper()
	s := silkroad_setting.GetSilkRoadSetting()
	prev := s.Storage
	s.Storage.LocalDir = localDir
	s.Storage.IngestNodeName = ingestNode
	s.Storage.PublicDownloadBaseURL = publicBase
	t.Cleanup(func() { s.Storage = prev })
}

func TestSilkRoadVideoIsIngestNode(t *testing.T) {
	prevNode := common.NodeName
	t.Cleanup(func() { common.NodeName = prevNode })

	withSilkRoadStorage(t, t.TempDir(), "", "")
	common.NodeName = "node-a"
	assert.False(t, IsSilkRoadIngestNode())

	withSilkRoadStorage(t, t.TempDir(), "node-a", "")
	common.NodeName = "node-a"
	assert.True(t, IsSilkRoadIngestNode())

	common.NodeName = "node-b"
	assert.False(t, IsSilkRoadIngestNode())
}

func TestSilkRoadVideoWriteOpenDelete(t *testing.T) {
	dir := t.TempDir()
	withSilkRoadStorage(t, dir, "", "")

	taskID := "task_test_123"
	payload := []byte("fake-video-bytes")

	absPath, size, err := WriteSilkRoadVideoFile(taskID, bytes.NewReader(payload))
	require.NoError(t, err)
	assert.Equal(t, int64(len(payload)), size)
	assert.Equal(t, filepath.Join(dir, taskID), SilkRoadVideoLocalPath(taskID))
	require.Equal(t, filepath.Join(dir, taskID), absPath)

	f, err := OpenSilkRoadVideoFile(taskID)
	require.NoError(t, err)
	defer f.Close()
	got, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, payload, got)

	require.NoError(t, DeleteSilkRoadVideoFile(taskID))
	_, err = os.Stat(SilkRoadVideoLocalPath(taskID))
	assert.True(t, os.IsNotExist(err))
}

func TestSilkRoadVideoPublicURL(t *testing.T) {
	withSilkRoadStorage(t, t.TempDir(), "", "https://video.example.com/")
	assert.Equal(t,
		"https://video.example.com/v1/videos/task_abc/content",
		BuildSilkRoadPublicURL("task_abc"),
	)

	withSilkRoadStorage(t, t.TempDir(), "", "https://video.example.com")
	assert.Equal(t,
		"https://video.example.com/v1/videos/task_abc/content",
		BuildSilkRoadPublicURL("task_abc"),
	)
}
