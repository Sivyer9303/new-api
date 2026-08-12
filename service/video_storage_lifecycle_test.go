package service

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupVideoLifecycleTestDB(t *testing.T) {
	t.Helper()
	originalDB := model.DB
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Task{}, &model.Channel{}))
	model.DB = db
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		model.DB = originalDB
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
	})
}

type videoConfirmationAdaptor struct {
	initialized bool
}

func (a *videoConfirmationAdaptor) Init(_ *relaycommon.RelayInfo) {
	a.initialized = true
}

func (a *videoConfirmationAdaptor) FetchTask(
	_ string,
	_ string,
	_ map[string]any,
	_ string,
) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(
			`{"status":"SUCCESS","progress":"100%","video_url":"https://upstream.example/video.mp4"}`,
		)),
	}, nil
}

func (a *videoConfirmationAdaptor) ParseTaskResult(_ []byte) (*relaycommon.TaskInfo, error) {
	return &relaycommon.TaskInfo{
		Status:   "SUCCESS",
		Progress: "100%",
		Url:      "https://upstream.example/video.mp4",
	}, nil
}

func (a *videoConfirmationAdaptor) AdjustBillingOnComplete(
	_ *model.Task,
	_ *relaycommon.TaskInfo,
) int {
	return 0
}

func TestRetryVideoStorageQueuesOnlyUnexpiredDeliveryFailures(t *testing.T) {
	setupVideoLifecycleTestDB(t)
	task := model.Task{
		TaskID:    "task_retry_delivery",
		CreatedAt: time.Now().Add(-time.Hour).Unix(),
		Status:    model.TaskStatusFailure,
		Progress:  "100%",
		Quota:     20,
		PrivateData: model.TaskPrivateData{
			StorageStatus:     "failed",
			StorageRetryCount: 3,
			StorageLastError:  "disk full",
			NoAutomaticRefund: true,
		},
	}
	require.NoError(t, model.DB.Create(&task).Error)

	queued, updated, err := RetryVideoStorage(task.TaskID)
	require.NoError(t, err)
	assert.True(t, updated)
	assert.Equal(t, model.TaskStatusStoring, queued.Status)
	assert.Equal(t, "pending", queued.PrivateData.StorageStatus)
	assert.Zero(t, queued.PrivateData.StorageRetryCount)
	assert.False(t, queued.PrivateData.NoAutomaticRefund)

	_, updated, err = RetryVideoStorage(task.TaskID)
	assert.ErrorIs(t, err, ErrVideoStorageRetryNotAllowed)
	assert.False(t, updated)
}

func TestRetryVideoStorageRejectsExpiredAndRefundedTasks(t *testing.T) {
	setupVideoLifecycleTestDB(t)
	expired := model.Task{
		TaskID:    "task_expired_delivery",
		CreatedAt: time.Now().Add(-8 * 24 * time.Hour).Unix(),
		Status:    model.TaskStatusFailure,
		PrivateData: model.TaskPrivateData{
			StorageStatus:     "failed",
			StorageExpiresAt:  time.Now().Add(-time.Hour).Unix(),
			NoAutomaticRefund: true,
		},
	}
	refunded := model.Task{
		TaskID:    "task_refunded_delivery",
		CreatedAt: time.Now().Unix(),
		Status:    model.TaskStatusRefunded,
		PrivateData: model.TaskPrivateData{
			StorageStatus:    "refunded",
			ManualRefundedAt: time.Now().Unix(),
		},
	}
	require.NoError(t, model.DB.Create(&expired).Error)
	require.NoError(t, model.DB.Create(&refunded).Error)

	_, updated, err := RetryVideoStorage(expired.TaskID)
	assert.ErrorIs(t, err, ErrVideoStorageExpired)
	assert.False(t, updated)
	_, updated, err = RetryVideoStorage(refunded.TaskID)
	assert.ErrorIs(t, err, ErrVideoStorageRetryNotAllowed)
	assert.False(t, updated)
}

func TestRetryVideoStorageDoesNotInventExpiryFromTaskCreation(t *testing.T) {
	setupVideoLifecycleTestDB(t)
	task := model.Task{
		TaskID:    "task_long_provider_run",
		CreatedAt: time.Now().Add(-30 * 24 * time.Hour).Unix(),
		Status:    model.TaskStatusFailure,
		PrivateData: model.TaskPrivateData{
			StorageStatus:     "failed",
			NoAutomaticRefund: true,
		},
	}
	require.NoError(t, model.DB.Create(&task).Error)

	_, updated, err := RetryVideoStorage(task.TaskID)
	require.NoError(t, err)
	assert.True(t, updated)
}

func TestConfirmVideoProviderResultDoesNotCompleteOrRefundTask(t *testing.T) {
	setupVideoLifecycleTestDB(t)
	baseURL := "https://provider.example"
	channel := model.Channel{
		Type:    constant.ChannelTypeSilkRoad,
		Key:     "provider-key",
		Name:    "video-provider",
		BaseURL: &baseURL,
	}
	require.NoError(t, model.DB.Create(&channel).Error)
	task := model.Task{
		TaskID:    "task_confirm_provider",
		ChannelId: channel.Id,
		Platform:  constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeSilkRoad)),
		Status:    model.TaskStatusFailure,
		Quota:     80,
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID:    "upstream-task",
			StorageStatus:     "failed",
			NoAutomaticRefund: true,
		},
	}
	require.NoError(t, model.DB.Create(&task).Error)

	adaptor := &videoConfirmationAdaptor{}
	originalFactory := GetTaskAdaptorFunc
	GetTaskAdaptorFunc = func(_ constant.TaskPlatform) TaskPollingAdaptor {
		return adaptor
	}
	t.Cleanup(func() { GetTaskAdaptorFunc = originalFactory })

	confirmation, err := ConfirmVideoProviderResult(&task)
	require.NoError(t, err)
	assert.True(t, adaptor.initialized)
	assert.Equal(t, "SUCCESS", confirmation.Status)
	assert.Equal(t, "https://upstream.example/video.mp4", confirmation.ResultURL)
	assert.Equal(t, "upstream-task", confirmation.UpstreamTaskID)

	var stored model.Task
	require.NoError(t, model.DB.Where("task_id = ?", task.TaskID).First(&stored).Error)
	assert.Equal(t, model.TaskStatus(model.TaskStatusFailure), stored.Status)
	assert.Equal(t, 80, stored.Quota)
	assert.True(t, stored.PrivateData.NoAutomaticRefund)
}

func TestConfirmVideoProviderResultRejectsExpiredTaskBeforeProviderFetch(t *testing.T) {
	task := &model.Task{
		Status: model.TaskStatusFailure,
		PrivateData: model.TaskPrivateData{
			StorageStatus:     "failed",
			StorageExpiresAt:  time.Now().Add(-time.Second).Unix(),
			NoAutomaticRefund: true,
		},
	}

	_, err := ConfirmVideoProviderResult(task)
	assert.ErrorIs(t, err, ErrVideoStorageExpired)
}
