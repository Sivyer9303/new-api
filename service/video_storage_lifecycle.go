package service

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

var (
	ErrVideoStorageRetryNotAllowed = errors.New("video storage retry is not allowed")
	ErrVideoStorageExpired         = errors.New("video retention period has expired")
	ErrVideoProviderConfirmDenied  = errors.New("video provider confirmation is not allowed")
)

func VideoDeliveryFailureMessage(taskID string) string {
	return fmt.Sprintf(
		"Video delivery failed after generation. Contact an administrator with task ID %s for review.",
		taskID,
	)
}

func MarkStoredVideoMissing(task *model.Task, cause error) (bool, error) {
	if task == nil || task.Status != model.TaskStatusSuccess {
		return false, nil
	}
	task.Status = model.TaskStatusFailure
	task.Progress = "100%"
	task.FinishTime = time.Now().Unix()
	task.FailReason = VideoDeliveryFailureMessage(task.TaskID)
	task.PrivateData.StorageStatus = "failed"
	task.PrivateData.NoAutomaticRefund = true
	if cause != nil {
		task.PrivateData.StorageLastError = cause.Error()
	}
	return task.UpdateWithStatus(model.TaskStatusSuccess)
}

func RetryVideoStorage(taskID string) (*model.Task, bool, error) {
	task, exists, err := model.GetVideoTaskByTaskID(taskID)
	if err != nil || !exists {
		return task, false, err
	}
	if task.Status != model.TaskStatusFailure ||
		task.PrivateData.StorageStatus != "failed" ||
		!task.PrivateData.NoAutomaticRefund ||
		task.PrivateData.ManualRefundedAt > 0 {
		return task, false, ErrVideoStorageRetryNotAllowed
	}
	if videoRecoveryExpired(task) {
		return task, false, ErrVideoStorageExpired
	}

	task.Status = model.TaskStatusStoring
	task.Progress = "99%"
	task.FailReason = ""
	task.PrivateData.StorageStatus = "pending"
	task.PrivateData.StorageRetryCount = 0
	task.PrivateData.StorageLastError = ""
	task.PrivateData.NoAutomaticRefund = false
	updated, err := task.UpdateWithStatus(model.TaskStatusFailure)
	return task, updated, err
}

type VideoProviderConfirmation struct {
	Status         string `json:"status"`
	Progress       string `json:"progress,omitempty"`
	FailureReason  string `json:"failure_reason,omitempty"`
	ResultURL      string `json:"result_url,omitempty"`
	UpstreamTaskID string `json:"upstream_task_id,omitempty"`
	CheckedAt      int64  `json:"checked_at"`
}

func ConfirmVideoProviderResult(task *model.Task) (*VideoProviderConfirmation, error) {
	if task == nil {
		return nil, errors.New("nil video task")
	}
	if videoRecoveryExpired(task) {
		return nil, ErrVideoStorageExpired
	}
	if task.Status != model.TaskStatusFailure ||
		task.PrivateData.StorageStatus != "failed" ||
		!task.PrivateData.NoAutomaticRefund ||
		task.PrivateData.ManualRefundedAt > 0 {
		return nil, ErrVideoProviderConfirmDenied
	}
	if GetTaskAdaptorFunc == nil {
		return nil, errors.New("video task adaptor is unavailable")
	}
	channel, err := model.CacheGetChannel(task.ChannelId)
	if err != nil {
		return nil, fmt.Errorf("get task channel: %w", err)
	}
	adaptor := GetTaskAdaptorFunc(task.Platform)
	if adaptor == nil {
		return nil, errors.New("video task adaptor not found")
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: channel.GetBaseURL(),
			ApiKey:         channel.Key,
		},
	}
	adaptor.Init(info)

	baseURL := ""
	if channel.Type >= 0 && channel.Type < len(constant.ChannelBaseURLs) {
		baseURL = constant.ChannelBaseURLs[channel.Type]
	}
	if configured := strings.TrimSpace(channel.GetBaseURL()); configured != "" {
		baseURL = configured
	}
	key := channel.Key
	if task.PrivateData.Key != "" {
		key = task.PrivateData.Key
	}
	response, err := adaptor.FetchTask(baseURL, key, map[string]any{
		"task_id": task.GetUpstreamTaskID(),
		"action":  task.Action,
	}, channel.GetSetting().Proxy)
	if err != nil {
		return nil, fmt.Errorf("fetch provider task: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("read provider task: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("provider task returned status %d", response.StatusCode)
	}
	taskInfo, err := adaptor.ParseTaskResult(body)
	if err != nil {
		return nil, fmt.Errorf("parse provider task: %w", err)
	}
	resultURL := taskInfo.Url
	if resultURL == "" {
		resultURL = taskInfo.RemoteUrl
	}
	return &VideoProviderConfirmation{
		Status:         taskInfo.Status,
		Progress:       taskInfo.Progress,
		FailureReason:  taskInfo.Reason,
		ResultURL:      resultURL,
		UpstreamTaskID: task.GetUpstreamTaskID(),
		CheckedAt:      time.Now().Unix(),
	}, nil
}

func videoRecoveryExpired(task *model.Task) bool {
	if task == nil {
		return false
	}
	if task.Status == model.TaskStatusExpired ||
		task.PrivateData.StorageStatus == "expired" {
		return true
	}
	expiresAt := task.PrivateData.StorageExpiresAt
	return expiresAt > 0 && time.Now().Unix() >= expiresAt
}
