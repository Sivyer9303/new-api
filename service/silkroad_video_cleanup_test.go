package service

import (
	"bytes"
	"context"
	"fmt"
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

func TestClaimSilkRoadExpiredVideoTasksSkipsNonReadySuccessRows(t *testing.T) {
	truncate(t)
	platform := constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeNewAPI))
	now := time.Now().Unix()
	expiredReadyID := "task_claim_expired_ready"
	ts := time.Now().Unix()

	// Old unfiltered Limit*10 scan could miss an expired-ready row behind many non-ready successes.
	for i := 0; i < 205; i++ {
		task := &model.Task{
			TaskID:    fmt.Sprintf("task_claim_pending_%d", i),
			Platform:  platform,
			UserId:    1,
			Status:    model.TaskStatusSuccess,
			CreatedAt: ts,
			UpdatedAt: ts,
			PrivateData: model.TaskPrivateData{
				StorageStatus: "pending",
			},
		}
		require.NoError(t, model.DB.Create(task).Error)
	}
	require.NoError(t, model.DB.Create(&model.Task{
		TaskID:    expiredReadyID,
		Platform:  platform,
		UserId:    1,
		Status:    model.TaskStatusSuccess,
		CreatedAt: ts,
		UpdatedAt: ts,
		PrivateData: model.TaskPrivateData{
			StorageStatus:    "ready",
			StorageExpiresAt: now - 60,
		},
	}).Error)

	claimed, err := claimSilkRoadExpiredVideoTasks(5, now)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	assert.Equal(t, expiredReadyID, claimed[0].TaskID)
}

func TestExpireOneSilkRoadVideoDeletesFileAndMarksExpired(t *testing.T) {
	dir := t.TempDir()
	withSilkRoadStorage(t, dir, "node-a", "https://video.example.com")

	taskID := "task_expire_1"
	_, _, err := WriteSilkRoadVideoFile(taskID, bytes.NewReader([]byte("video")))
	require.NoError(t, err)

	task := &model.Task{
		TaskID: taskID,
		PrivateData: model.TaskPrivateData{
			StorageStatus:     "ready",
			StorageExpiresAt:  time.Now().Unix() - 60,
			StoragePath:       SilkRoadVideoLocalPath(taskID),
			StorageObjectKey:  taskID,
			ResultURL:         BuildSilkRoadPublicURL(taskID),
			UpstreamResultURL: "https://cdn.example/private.mp4",
		},
	}
	require.NoError(t, expireOneSilkRoadVideo(task))

	assert.Equal(t, "expired", task.PrivateData.StorageStatus)
	assert.Equal(t, model.TaskStatusExpired, task.Status)
	assert.Empty(t, task.PrivateData.ResultURL)
	assert.Empty(t, task.PrivateData.UpstreamResultURL)
	assert.Empty(t, task.PrivateData.StorageObjectKey)
	_, statErr := os.Stat(SilkRoadVideoLocalPath(taskID))
	assert.True(t, os.IsNotExist(statErr))
}

func TestExpireOneSilkRoadVideoPreservesMetadataWhenDeleteFails(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "child"), []byte("x"), 0o600))
	task := &model.Task{
		TaskID: "task_cleanup_delete_failure",
		Status: model.TaskStatusStorageDeleting,
		PrivateData: model.TaskPrivateData{
			StorageStatus:    "ready",
			StoragePath:      dir,
			StorageObjectKey: "",
			ResultURL:        "/v1/videos/task_cleanup_delete_failure/content",
		},
	}

	err := expireOneSilkRoadVideo(task)
	require.Error(t, err)
	assert.Equal(t, model.TaskStatusStorageDeleting, task.Status)
	assert.Equal(t, "ready", task.PrivateData.StorageStatus)
	assert.Equal(t, dir, task.PrivateData.StoragePath)
	assert.NotEmpty(t, task.PrivateData.ResultURL)
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
				StoragePath:      SilkRoadVideoLocalPath(spec.taskID),
				ResultURL:        BuildSilkRoadPublicURL(spec.taskID),
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

func TestCleanupClaimCASPreventsOverlappingDelete(t *testing.T) {
	truncate(t)
	now := time.Now().Unix()
	task := &model.Task{
		TaskID:   "task_cleanup_cas",
		Platform: constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeSilkRoad)),
		Status:   model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			StorageStatus:    "ready",
			StorageExpiresAt: now - 1,
		},
	}
	require.NoError(t, model.DB.Create(task).Error)

	var first, staleSecond model.Task
	require.NoError(t, model.DB.First(&first, task.ID).Error)
	require.NoError(t, model.DB.First(&staleSecond, task.ID).Error)

	first.Status = model.TaskStatusStorageDeleting
	won, err := first.UpdateWithStatus(model.TaskStatusSuccess)
	require.NoError(t, err)
	assert.True(t, won)

	staleSecond.Status = model.TaskStatusStorageDeleting
	won, err = staleSecond.UpdateWithStatus(model.TaskStatusSuccess)
	require.NoError(t, err)
	assert.False(t, won)
}
