package service

import (
	"bytes"
	"context"
	"os"
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

func TestExpireOneSilkRoadVideoDeletesFileAndMarksExpired(t *testing.T) {
	dir := t.TempDir()
	withSilkRoadStorage(t, dir, "node-a", "https://video.example.com")

	taskID := "task_expire_1"
	_, _, err := WriteSilkRoadVideoFile(taskID, bytes.NewReader([]byte("video")))
	require.NoError(t, err)

	task := &model.Task{
		TaskID: taskID,
		PrivateData: model.TaskPrivateData{
			StorageStatus:    "ready",
			StorageExpiresAt: time.Now().Unix() - 60,
			StoragePath:        SilkRoadVideoLocalPath(taskID),
		},
	}
	expireOneSilkRoadVideo(task)

	assert.Equal(t, "expired", task.PrivateData.StorageStatus)
	_, statErr := os.Stat(SilkRoadVideoLocalPath(taskID))
	assert.True(t, os.IsNotExist(statErr))
}

func TestRunSilkRoadVideoCleanupOnceExpiresReadyPastRetention(t *testing.T) {
	truncate(t)
	dir := t.TempDir()
	withSilkRoadStorage(t, dir, "node-a", "https://video.example.com")
	silkroad_setting.GetSilkRoadSetting().Storage.Enabled = true

	prevNode := common.NodeName
	common.NodeName = "node-a"
	t.Cleanup(func() { common.NodeName = prevNode })

	expiredID := "task_cleanup_expired"
	freshID := "task_cleanup_fresh"
	for _, spec := range []struct {
		taskID    string
		expiresAt int64
	}{
		{expiredID, time.Now().Unix() - 120},
		{freshID, time.Now().Unix() + 3600},
	} {
		_, _, err := WriteSilkRoadVideoFile(spec.taskID, bytes.NewReader([]byte("bytes")))
		require.NoError(t, err)
		task := &model.Task{
			TaskID:    spec.taskID,
			Platform:  constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeNewAPI)),
			UserId:    1,
			Status:    model.TaskStatusSuccess,
			CreatedAt: time.Now().Unix(),
			UpdatedAt: time.Now().Unix(),
			PrivateData: model.TaskPrivateData{
				StorageStatus:    "ready",
				StorageExpiresAt: spec.expiresAt,
				StoragePath:        SilkRoadVideoLocalPath(spec.taskID),
				ResultURL:          BuildSilkRoadPublicURL(spec.taskID),
			},
		}
		require.NoError(t, model.DB.Create(task).Error)
	}

	require.NoError(t, RunSilkRoadVideoCleanupOnce(context.Background()))

	var expiredTask model.Task
	require.NoError(t, model.DB.Where("task_id = ?", expiredID).First(&expiredTask).Error)
	assert.Equal(t, "expired", expiredTask.PrivateData.StorageStatus)
	_, statErr := os.Stat(SilkRoadVideoLocalPath(expiredID))
	assert.True(t, os.IsNotExist(statErr))

	var freshTask model.Task
	require.NoError(t, model.DB.Where("task_id = ?", freshID).First(&freshTask).Error)
	assert.Equal(t, "ready", freshTask.PrivateData.StorageStatus)
	_, err := os.Stat(SilkRoadVideoLocalPath(freshID))
	require.NoError(t, err)
}

func TestRunSilkRoadVideoCleanupOnceSkipsNonIngestNode(t *testing.T) {
	truncate(t)
	dir := t.TempDir()
	withSilkRoadStorage(t, dir, "node-a", "https://video.example.com")
	silkroad_setting.GetSilkRoadSetting().Storage.Enabled = true

	prevNode := common.NodeName
	common.NodeName = "node-b"
	t.Cleanup(func() { common.NodeName = prevNode })

	taskID := "task_cleanup_skip"
	_, _, err := WriteSilkRoadVideoFile(taskID, bytes.NewReader([]byte("x")))
	require.NoError(t, err)
	task := &model.Task{
		TaskID:    taskID,
		Platform:  constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeNewAPI)),
		UserId:    1,
		Status:    model.TaskStatusSuccess,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		PrivateData: model.TaskPrivateData{
			StorageStatus:    "ready",
			StorageExpiresAt: time.Now().Unix() - 10,
		},
	}
	require.NoError(t, model.DB.Create(task).Error)

	require.NoError(t, RunSilkRoadVideoCleanupOnce(context.Background()))

	var reloaded model.Task
	require.NoError(t, model.DB.Where("task_id = ?", taskID).First(&reloaded).Error)
	assert.Equal(t, "ready", reloaded.PrivateData.StorageStatus)
	_, statErr := os.Stat(SilkRoadVideoLocalPath(taskID))
	require.NoError(t, statErr)
}
