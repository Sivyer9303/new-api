package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	taskdto "github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/samber/lo"
)

// TaskPollingAdaptor 定义轮询所需的最小适配器接口，避免 service -> relay 的循环依赖
type TaskPollingAdaptor interface {
	Init(info *relaycommon.RelayInfo)
	FetchTask(baseURL string, key string, body map[string]any, proxy string) (*http.Response, error)
	ParseTaskResult(body []byte) (*relaycommon.TaskInfo, error)
	// AdjustBillingOnComplete 在任务到达终态（成功/失败）时由轮询循环调用。
	// 返回正数触发差额结算（补扣/退还），返回 0 保持预扣费金额不变。
	AdjustBillingOnComplete(task *model.Task, taskResult *relaycommon.TaskInfo) int
}

// GetTaskAdaptorFunc 由 main 包注入，用于获取指定平台的任务适配器。
// 打破 service -> relay -> relay/channel -> service 的循环依赖。
var GetTaskAdaptorFunc func(platform constant.TaskPlatform) TaskPollingAdaptor

const maxTaskPollingResponseSize = 4 << 20

// sweepTimedOutTasks 在主轮询之前独立清理超时任务。
// 每次最多处理 100 条，剩余的下个周期继续处理。
// 使用 per-task CAS (UpdateWithStatus) 防止覆盖被正常轮询已推进的任务。
func sweepTimedOutTasks(ctx context.Context) {
	if constant.TaskTimeoutMinutes <= 0 {
		return
	}
	cutoff := time.Now().Unix() - int64(constant.TaskTimeoutMinutes)*60
	tasks := model.GetTimedOutUnfinishedTasks(cutoff, 100)
	if len(tasks) == 0 {
		return
	}

	reason := fmt.Sprintf("任务超时（%d分钟）", constant.TaskTimeoutMinutes)
	legacyReason := "任务超时（旧系统遗留任务，不进行退款，请联系管理员）"
	now := time.Now().Unix()
	timedOutCount := 0

	for _, task := range tasks {
		isLegacy := task.SubmitTime > 0 && task.SubmitTime < model.TaskRefundLegacyCutoff
		requiresReview := task.PrivateData.NoAutomaticRefund ||
			task.Status == model.TaskStatusSubmitting ||
			strings.TrimSpace(task.PrivateData.UpstreamTaskID) != ""

		oldStatus := task.Status
		task.Status = model.TaskStatusFailure
		task.Progress = "100%"
		task.FinishTime = now
		if requiresReview {
			task.FailReason = "Task submission outcome is uncertain after timeout; administrator review is required"
			task.PrivateData.NoAutomaticRefund = true
			task.PrivateData.StorageStatus = model.TaskStorageStatusProviderReview
			task.PrivateData.StorageLastError = task.FailReason
		} else if isLegacy {
			task.FailReason = legacyReason
			// 旧系统任务明确不退款，随终态 CAS 一并清掉 quota，
			// 避免留下可再次退款的计费状态。
			task.Quota = 0
		} else {
			task.FailReason = reason
			if task.Quota != 0 {
				task.MarkRefundPending(reason)
			}
		}

		won, err := task.UpdateWithStatus(oldStatus)
		if err != nil {
			logger.LogError(ctx, fmt.Sprintf("sweepTimedOutTasks CAS update error for task %s: %v", task.TaskID, err))
			continue
		}
		if !won {
			logger.LogInfo(ctx, fmt.Sprintf("sweepTimedOutTasks: task %s already transitioned, skip", task.TaskID))
			continue
		}
		timedOutCount++
		if !isLegacy && !requiresReview && task.Quota != 0 {
			RefundTaskQuota(ctx, task, reason)
		}
	}

	if timedOutCount > 0 {
		logger.LogInfo(ctx, fmt.Sprintf("sweepTimedOutTasks: timed out %d tasks", timedOutCount))
	}
}

// TaskPollSummary is the result recorded on an async_task_poll system task row,
// summarizing one polling pass.
type TaskPollSummary struct {
	UnfinishedTasks  int `json:"unfinished_tasks"`
	PlatformsScanned int `json:"platforms_scanned"`
	NullTasksFailed  int `json:"null_tasks_failed"`
}

func HasPendingTaskRefunds() bool {
	var id int64
	now := time.Now().Unix()
	err := model.DB.Model(&model.Task{}).
		Where(
			"status = ? AND quota <> 0 AND refund_pending = ? AND refund_retry_at <= ?",
			model.TaskStatusFailure,
			true,
			now,
		).
		Limit(1).
		Pluck("id", &id).Error
	return err == nil && id != 0
}

func recoverPendingTaskRefunds(ctx context.Context, limit int) error {
	if limit <= 0 {
		return nil
	}
	now := time.Now().Unix()
	var tasks []*model.Task
	if err := model.DB.
		Where(
			"status = ? AND quota <> 0 AND refund_pending = ? AND refund_retry_at <= ?",
			model.TaskStatusFailure,
			true,
			now,
		).
		Order("refund_retry_at, id").
		Limit(limit).
		Find(&tasks).Error; err != nil {
		return err
	}
	var firstErr error
	for _, task := range tasks {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		reason := strings.TrimSpace(task.PrivateData.BillingRefundReason)
		if reason == "" {
			reason = "retry pending task refund"
		}
		if RefundTaskQuota(ctx, task, reason) {
			continue
		}
		attempts := task.RefundAttempts + 1
		delaySeconds := int64(15)
		for attempt := 1; attempt < attempts && delaySeconds < 3600; attempt++ {
			delaySeconds *= 2
		}
		if delaySeconds > 3600 {
			delaySeconds = 3600
		}
		err := model.DB.Model(&model.Task{}).
			Where("id = ? AND refund_pending = ? AND quota <> 0", task.ID, true).
			Updates(map[string]any{
				"refund_attempts": attempts,
				"refund_retry_at": now + delaySeconds,
			}).Error
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// RunTaskPollingOnce performs one async-task (Suno/video) polling pass
// synchronously. It honors ctx cancellation (the system-task runner cancels it
// when the lease is lost) and, when report is non-nil, reports progress as
// (processedPlatforms, totalPlatforms). It returns immediately if the task
// adaptor factory has not been wired yet, to avoid a nil call during startup.
func RunTaskPollingOnce(ctx context.Context, report func(processed, total int)) TaskPollSummary {
	summary := TaskPollSummary{}
	if GetTaskAdaptorFunc == nil {
		return summary
	}
	if ctx == nil {
		ctx = context.Background()
	}

	common.SysLog("任务进度轮询开始")
	if err := recoverPendingTaskRefunds(ctx, 100); err != nil {
		logger.LogError(ctx, "recover pending task refunds: "+err.Error())
	}
	if err := recoverStaleVideoSettlements(ctx, videoIngestBatch); err != nil {
		logger.LogError(ctx, "recover stale video settlements: "+err.Error())
	}
	sweepTimedOutTasks(ctx)
	allTasks := model.GetAllUnFinishSyncTasks(constant.TaskQueryLimit)
	summary.UnfinishedTasks = len(allTasks)
	platformTask := make(map[constant.TaskPlatform][]*model.Task)
	for _, t := range allTasks {
		platformTask[t.Platform] = append(platformTask[t.Platform], t)
	}

	totalPlatforms := len(platformTask)
	processedPlatforms := 0
	for platform, tasks := range platformTask {
		if ctx.Err() != nil {
			break
		}
		if report != nil {
			report(processedPlatforms, totalPlatforms)
		}
		processedPlatforms++
		if len(tasks) == 0 {
			continue
		}
		summary.PlatformsScanned++
		taskChannelM := make(map[int][]string)
		taskM := make(map[string]*model.Task)
		nullTaskIds := make([]int64, 0)
		for _, task := range tasks {
			if task.Status == model.TaskStatusSubmitting &&
				!hasPersistedUpstreamTaskID(task) {
				continue
			}
			upstreamID := task.GetUpstreamTaskID()
			if upstreamID == "" {
				// 统计失败的未完成任务
				nullTaskIds = append(nullTaskIds, task.ID)
				continue
			}
			taskM[upstreamID] = task
			taskChannelM[task.ChannelId] = append(taskChannelM[task.ChannelId], upstreamID)
		}
		if len(nullTaskIds) > 0 {
			summary.NullTasksFailed += len(nullTaskIds)
			err := model.TaskBulkUpdateByID(nullTaskIds, map[string]any{
				"status":   "FAILURE",
				"progress": "100%",
			})
			if err != nil {
				logger.LogError(ctx, fmt.Sprintf("Fix null task_id task error: %v", err))
			} else {
				logger.LogInfo(ctx, fmt.Sprintf("Fix null task_id task success: %v", nullTaskIds))
			}
		}
		if len(taskChannelM) == 0 {
			continue
		}

		DispatchPlatformUpdate(ctx, platform, taskChannelM, taskM)
	}
	if report != nil && ctx.Err() == nil {
		report(totalPlatforms, totalPlatforms)
	}
	common.SysLog("任务进度轮询完成")
	return summary
}

func hasPersistedUpstreamTaskID(task *model.Task) bool {
	return task != nil && strings.TrimSpace(task.PrivateData.UpstreamTaskID) != ""
}

// DispatchPlatformUpdate 按平台分发轮询更新
func DispatchPlatformUpdate(ctx context.Context, platform constant.TaskPlatform, taskChannelM map[int][]string, taskM map[string]*model.Task) {
	if ctx == nil {
		ctx = context.Background()
	}
	switch platform {
	case constant.TaskPlatformMidjourney:
		// MJ 轮询由其自身处理，这里预留入口
	case constant.TaskPlatformSuno:
		_ = UpdateSunoTasks(ctx, taskChannelM, taskM)
	default:
		if err := UpdateVideoTasks(ctx, platform, taskChannelM, taskM); err != nil {
			common.SysLog(fmt.Sprintf("UpdateVideoTasks fail: %s", err))
		}
	}
}

// UpdateSunoTasks 按渠道更新所有 Suno 任务
func UpdateSunoTasks(ctx context.Context, taskChannelM map[int][]string, taskM map[string]*model.Task) error {
	for channelId, taskIds := range taskChannelM {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err := updateSunoTasks(ctx, channelId, taskIds, taskM)
		if err != nil {
			logger.LogError(ctx, fmt.Sprintf("渠道 #%d 更新异步任务失败: %s", channelId, err.Error()))
		}
	}
	return nil
}

func updateSunoTasks(ctx context.Context, channelId int, taskIds []string, taskM map[string]*model.Task) error {
	logger.LogInfo(ctx, fmt.Sprintf("渠道 #%d 未完成的任务有: %d", channelId, len(taskIds)))
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if len(taskIds) == 0 {
		return nil
	}
	ch, err := model.CacheGetChannel(channelId)
	if err != nil {
		common.SysLog(fmt.Sprintf("CacheGetChannel: %v", err))
		// Collect DB primary key IDs for bulk update (taskIds are upstream IDs, not task_id column values)
		var failedIDs []int64
		for _, upstreamID := range taskIds {
			if t, ok := taskM[upstreamID]; ok {
				failedIDs = append(failedIDs, t.ID)
			}
		}
		err = model.TaskBulkUpdateByID(failedIDs, map[string]any{
			"fail_reason": fmt.Sprintf("获取渠道信息失败，请联系管理员，渠道ID：%d", channelId),
			"status":      "FAILURE",
			"progress":    "100%",
		})
		if err != nil {
			common.SysLog(fmt.Sprintf("UpdateSunoTask error: %v", err))
		}
		return err
	}
	adaptor := GetTaskAdaptorFunc(constant.TaskPlatformSuno)
	if adaptor == nil {
		return errors.New("adaptor not found")
	}
	proxy := ch.GetSetting().Proxy
	resp, err := adaptor.FetchTask(*ch.BaseURL, ch.Key, map[string]any{
		"ids": taskIds,
	}, proxy)
	if err != nil {
		common.SysLog(fmt.Sprintf("Get Task Do req error: %v", err))
		return err
	}
	if resp.StatusCode != http.StatusOK {
		logger.LogError(ctx, fmt.Sprintf("Get Task status code: %d", resp.StatusCode))
		return fmt.Errorf("Get Task status code: %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		common.SysLog(fmt.Sprintf("Get Suno Task parse body error: %v", err))
		return err
	}
	var responseItems taskdto.TaskResponse[[]taskdto.SunoDataResponse]
	err = common.Unmarshal(responseBody, &responseItems)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("Get Suno Task parse body error2: %v, body: %s", err, string(responseBody)))
		return err
	}
	if !responseItems.IsSuccess() {
		common.SysLog(fmt.Sprintf("渠道 #%d 未完成的任务有: %d, 成功获取到任务数: %s", channelId, len(taskIds), string(responseBody)))
		return err
	}

	for _, responseItem := range responseItems.Data {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		task := taskM[responseItem.TaskID]
		if task == nil {
			logger.LogWarn(ctx, fmt.Sprintf("Suno task response ignored: unknown task_id=%s", responseItem.TaskID))
			continue
		}
		if !taskNeedsUpdate(task, responseItem) {
			continue
		}

		prevStatus := task.Status
		task.Status = lo.If(model.TaskStatus(responseItem.Status) != "", model.TaskStatus(responseItem.Status)).Else(task.Status)
		task.FailReason = lo.If(responseItem.FailReason != "", responseItem.FailReason).Else(task.FailReason)
		task.SubmitTime = lo.If(responseItem.SubmitTime != 0, responseItem.SubmitTime).Else(task.SubmitTime)
		task.StartTime = lo.If(responseItem.StartTime != 0, responseItem.StartTime).Else(task.StartTime)
		task.FinishTime = lo.If(responseItem.FinishTime != 0, responseItem.FinishTime).Else(task.FinishTime)
		isFailure := responseItem.FailReason != "" || task.Status == model.TaskStatusFailure
		if isFailure {
			logger.LogInfo(ctx, task.TaskID+" 构建失败，"+task.FailReason)
			task.Status = model.TaskStatusFailure
			task.Progress = "100%"
			if prevStatus != model.TaskStatusFailure && task.Quota != 0 {
				task.MarkRefundPending(task.FailReason)
			}
		}
		if model.TaskStatus(responseItem.Status) == model.TaskStatusSuccess {
			task.Progress = "100%"
		}
		task.Data = responseItem.Data

		// 持久化走 CAS，防止重叠轮询/sweep/多实例/持久化失败重试导致重复退款或覆盖终态。
		won, err := task.UpdateWithStatus(prevStatus)
		if err != nil {
			logger.LogError(ctx, fmt.Sprintf("UpdateSunoTask task %s error: %v", task.TaskID, err))
		} else if !won {
			logger.LogWarn(ctx, fmt.Sprintf("Task %s CAS lost or no-op update, skip billing", task.TaskID))
		} else if isFailure && prevStatus != model.TaskStatusFailure && task.Quota != 0 {
			RefundTaskQuota(ctx, task, task.FailReason)
		}
	}
	return nil
}

// taskNeedsUpdate 检查 Suno 任务是否需要更新
func taskNeedsUpdate(oldTask *model.Task, newTask taskdto.SunoDataResponse) bool {
	if oldTask.SubmitTime != newTask.SubmitTime {
		return true
	}
	if oldTask.StartTime != newTask.StartTime {
		return true
	}
	if oldTask.FinishTime != newTask.FinishTime {
		return true
	}
	if string(oldTask.Status) != newTask.Status {
		return true
	}
	if oldTask.FailReason != newTask.FailReason {
		return true
	}

	if (oldTask.Status == model.TaskStatusFailure || oldTask.Status == model.TaskStatusSuccess) && oldTask.Progress != "100%" {
		return true
	}

	oldData, _ := common.Marshal(oldTask.Data)
	newData, _ := common.Marshal(newTask.Data)

	sort.Slice(oldData, func(i, j int) bool {
		return oldData[i] < oldData[j]
	})
	sort.Slice(newData, func(i, j int) bool {
		return newData[i] < newData[j]
	})

	if string(oldData) != string(newData) {
		return true
	}
	return false
}

// UpdateVideoTasks 按渠道更新所有视频任务
func UpdateVideoTasks(ctx context.Context, platform constant.TaskPlatform, taskChannelM map[int][]string, taskM map[string]*model.Task) error {
	channelIDs := make([]int, 0, len(taskChannelM))
	for channelID := range taskChannelM {
		channelIDs = append(channelIDs, channelID)
	}
	sort.Ints(channelIDs)

	var wg sync.WaitGroup
	for _, channelId := range channelIDs {
		taskIds := taskChannelM[channelId]
		if len(taskIds) == 0 {
			continue
		}
		taskIds = append([]string(nil), taskIds...)

		wg.Add(1)
		gopool.Go(func() {
			defer wg.Done()
			if err := updateVideoTasks(ctx, platform, channelId, taskIds, taskM); err != nil {
				logger.LogError(ctx, fmt.Sprintf("Channel #%d failed to update video async tasks: %s", channelId, err.Error()))
			}
		})
	}
	wg.Wait()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

func updateVideoTasks(ctx context.Context, platform constant.TaskPlatform, channelId int, taskIds []string, taskM map[string]*model.Task) error {
	logger.LogInfo(ctx, fmt.Sprintf("Channel #%d pending video tasks: %d", channelId, len(taskIds)))
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if len(taskIds) == 0 {
		return nil
	}
	cacheGetChannel, cacheErr := model.CacheGetChannel(channelId)
	if cacheErr != nil || cacheGetChannel == nil {
		var exists bool
		var err error
		cacheGetChannel, exists, err = model.GetChannelByIdIfExists(channelId, true)
		if err != nil {
			return fmt.Errorf("load video polling channel %d: %w", channelId, err)
		}
		if !exists {
			markVideoTasksForMissingChannelReview(ctx, channelId, taskIds, taskM)
			return nil
		}
	}
	adaptor := GetTaskAdaptorFunc(platform)
	if adaptor == nil {
		return fmt.Errorf("video adaptor not found")
	}
	info := &relaycommon.RelayInfo{}
	info.ChannelMeta = &relaycommon.ChannelMeta{
		ChannelBaseUrl: cacheGetChannel.GetBaseURL(),
	}
	info.ApiKey = cacheGetChannel.Key
	adaptor.Init(info)
	disablePollingSleep := cacheGetChannel.GetOtherSettings().DisableTaskPollingSleep
	for i, taskId := range taskIds {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := updateVideoSingleTask(ctx, adaptor, cacheGetChannel, taskId, taskM); err != nil {
			logger.LogError(ctx, fmt.Sprintf("Failed to update video task %s: %s", taskId, err.Error()))
		}
		if disablePollingSleep || i == len(taskIds)-1 {
			continue
		}

		// sleep 1 second between tasks for this channel only.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
	return nil
}

func markVideoTasksForMissingChannelReview(
	ctx context.Context,
	channelID int,
	taskIDs []string,
	taskM map[string]*model.Task,
) {
	reason := fmt.Sprintf(
		"Video polling channel %d no longer exists; administrator review is required",
		channelID,
	)
	for _, upstreamID := range taskIDs {
		task := taskM[upstreamID]
		if task == nil {
			continue
		}
		fromStatus := task.Status
		task.Status = model.TaskStatusFailure
		task.Progress = taskcommon.ProgressComplete
		task.FailReason = reason
		task.PrivateData.NoAutomaticRefund = true
		task.PrivateData.StorageStatus = model.TaskStorageStatusProviderReview
		task.PrivateData.StorageLastError = reason
		won, err := task.UpdateWithStatus(fromStatus)
		if err != nil {
			logger.LogError(ctx, fmt.Sprintf(
				"mark task %s for missing-channel review: %v",
				task.TaskID,
				err,
			))
		} else if !won {
			logger.LogWarn(ctx, fmt.Sprintf(
				"task %s changed before missing-channel review",
				task.TaskID,
			))
		}
	}
}

func readTaskPollingResponse(resp *http.Response) ([]byte, error) {
	if resp == nil || resp.Body == nil {
		return nil, errors.New("upstream returned an empty polling response")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(
		resp.Body,
		maxTaskPollingResponseSize+1,
	))
	if err != nil {
		return nil, err
	}
	if len(body) > maxTaskPollingResponseSize {
		return nil, errors.New("upstream polling response exceeds 4 MiB")
	}
	return body, nil
}

func isRetryableTaskPollingStatus(statusCode int) bool {
	return statusCode == http.StatusRequestTimeout ||
		statusCode == http.StatusTooEarly ||
		statusCode == http.StatusTooManyRequests ||
		statusCode >= http.StatusInternalServerError
}

func closeTaskPollingErrorResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 32<<10))
	_ = resp.Body.Close()
}

func markTaskPollingProviderReview(
	ctx context.Context,
	task *model.Task,
	statusCode int,
) error {
	fromStatus := task.Status
	reason := fmt.Sprintf(
		"Provider polling returned HTTP status %d; administrator review is required",
		statusCode,
	)
	task.Status = model.TaskStatusFailure
	task.Progress = taskcommon.ProgressComplete
	task.FinishTime = time.Now().Unix()
	task.FailReason = reason
	task.PrivateData.NoAutomaticRefund = true
	task.PrivateData.StorageStatus = model.TaskStorageStatusProviderReview
	task.PrivateData.StorageLastError = reason
	won, err := task.UpdateWithStatus(fromStatus)
	if err != nil {
		return err
	}
	if !won {
		logger.LogWarn(ctx, fmt.Sprintf(
			"task %s changed before polling HTTP review",
			task.TaskID,
		))
	}
	return nil
}

func sanitizeTaskFailureReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "Provider task failed"
	}
	lowerReason := strings.ToLower(reason)
	firstURL := -1
	for _, prefix := range []string{
		"https://",
		"http://",
		"data:image/",
		"data:video/",
		"data:audio/",
		"data:application/",
	} {
		if index := strings.Index(lowerReason, prefix); index >= 0 &&
			(firstURL < 0 || index < firstURL) {
			firstURL = index
		}
	}
	if firstURL >= 0 {
		reason = strings.TrimSpace(reason[:firstURL]) + " [provider URL redacted]"
	}
	const maxRunes = 512
	runes := []rune(reason)
	if len(runes) > maxRunes {
		reason = string(runes[:maxRunes]) + "..."
	}
	return strings.TrimSpace(reason)
}

func updateVideoSingleTask(ctx context.Context, adaptor TaskPollingAdaptor, ch *model.Channel, taskId string, taskM map[string]*model.Task) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	baseURL := constant.ChannelBaseURLs[ch.Type]
	if ch.GetBaseURL() != "" {
		baseURL = ch.GetBaseURL()
	}
	proxy := ch.GetSetting().Proxy

	task := taskM[taskId]
	if task == nil {
		logger.LogError(ctx, fmt.Sprintf("Task %s not found in taskM", taskId))
		return fmt.Errorf("task %s not found", taskId)
	}
	key := ch.Key

	privateData := task.PrivateData
	if privateData.Key != "" {
		key = privateData.Key
	}
	resp, err := adaptor.FetchTask(baseURL, key, map[string]any{
		"task_id": task.GetUpstreamTaskID(),
		"action":  task.Action,
	}, proxy)
	if err != nil {
		return fmt.Errorf("fetchTask failed for task %s: %w", taskId, err)
	}
	if resp == nil || resp.Body == nil {
		return fmt.Errorf("poll response for task %s is empty", taskId)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		statusCode := resp.StatusCode
		closeTaskPollingErrorResponse(resp)
		if isRetryableTaskPollingStatus(statusCode) {
			return fmt.Errorf(
				"provider polling returned retryable status %d for task %s",
				statusCode,
				taskId,
			)
		}
		return markTaskPollingProviderReview(ctx, task, statusCode)
	}
	responseBody, err := readTaskPollingResponse(resp)
	if err != nil {
		return fmt.Errorf("read poll response for task %s: %w", taskId, err)
	}

	logger.LogDebug(ctx, "updateVideoSingleTask received response for task %s", task.TaskID)

	snap := task.Snapshot()

	taskResult := &relaycommon.TaskInfo{}
	preferDirectParser := false
	if preference, ok := adaptor.(interface{ PreferDirectTaskResultParsing() bool }); ok {
		preferDirectParser = preference.PreferDirectTaskResultParsing()
	}
	if preferDirectParser {
		if taskResult, err = adaptor.ParseTaskResult(responseBody); err != nil {
			return fmt.Errorf("parseTaskResult failed for task %s: %w", taskId, err)
		}
	} else {
		// Some upstream gateways return this application's TaskResponse envelope.
		var responseItems taskdto.TaskResponse[model.Task]
		if err = common.Unmarshal(responseBody, &responseItems); err == nil && responseItems.IsSuccess() {
			logger.LogDebug(ctx, "updateVideoSingleTask parsed legacy response for task %s", task.TaskID)
			t := responseItems.Data
			taskResult.TaskID = t.TaskID
			taskResult.Status = string(t.Status)
			taskResult.Url = t.GetResultURL()
			if taskResult.Url == "" || isVideoContentProxyURL(taskResult.Url) {
				taskResult.Url = ExtractUpstreamVideoURLFromJSON(responseBody)
			}
			taskResult.Progress = t.Progress
			taskResult.Reason = t.FailReason
			task.Data = t.Data
		} else if taskResult, err = adaptor.ParseTaskResult(responseBody); err != nil {
			return fmt.Errorf("parseTaskResult failed for task %s: %w", taskId, err)
		}
	}

	task.Data = redactVideoResponseBody(responseBody)
	cleaned, redactErr := applyVideoDataRedaction(task.Data, task.TaskID)
	if redactErr != nil {
		logger.LogWarn(ctx, fmt.Sprintf(
			"video response redaction failed task=%s; clearing task.Data",
			task.TaskID,
		))
		task.Data = []byte("{}")
	} else {
		task.Data = cleaned
	}

	logger.LogDebug(
		ctx,
		"updateVideoSingleTask normalized task=%s status=%s progress=%s",
		task.TaskID,
		taskResult.Status,
		taskResult.Progress,
	)

	now := time.Now().Unix()
	if taskResult.Status == "" {
		//taskResult = relaycommon.FailTaskInfo("upstream returned empty status")
		errorResult := &dto.GeneralErrorResponse{}
		if err = common.Unmarshal(responseBody, &errorResult); err == nil {
			openaiError := errorResult.TryToOpenAIError()
			if openaiError != nil {
				// 返回规范的 OpenAI 错误格式，提取错误信息，判断错误是否为任务失败
				if openaiError.Code == "429" {
					// 429 错误通常表示请求过多或速率限制，暂时不认为是任务失败，保持原状态等待下一轮轮询
					return nil
				}

				// 其他错误认为是任务失败，记录错误信息并更新任务状态
				taskResult = relaycommon.FailTaskInfo("upstream returned error")
			} else {
				logger.LogError(ctx, fmt.Sprintf("Task %s returned empty status with unrecognized error format", taskId))
				taskResult = relaycommon.FailTaskInfo("upstream returned unrecognized message")
			}
		}
	}

	shouldRefund := false
	shouldSettle := false
	quota := task.Quota

	task.Status = model.TaskStatus(taskResult.Status)
	switch model.TaskStatus(taskResult.Status) {
	case model.TaskStatusSubmitted:
		task.Progress = taskcommon.ProgressSubmitted
	case model.TaskStatusQueued:
		task.Progress = taskcommon.ProgressQueued
	case model.TaskStatusInProgress:
		task.Progress = taskcommon.ProgressInProgress
		if task.StartTime == 0 {
			task.StartTime = now
		}
	case model.TaskStatusSuccess:
		resultURL := taskResult.Url
		if resultURL == "" {
			resultURL = taskResult.RemoteUrl
		}
		if !hasUsableVideoResultSource(ch, task, resultURL) {
			task.Status = model.TaskStatusFailure
			task.Progress = taskcommon.ProgressComplete
			task.FinishTime = now
			task.FailReason = "Provider completed the task without a usable result; administrator review is required"
			task.PrivateData.NoAutomaticRefund = true
			task.PrivateData.StorageStatus = model.TaskStorageStatusProviderReview
			task.PrivateData.StorageLastError = task.FailReason
			break
		}
		task.Status = model.TaskStatusSettlementProcessing
		task.Progress = "99%"
		applyVideoSuccessStore(task, resultURL, responseBody)
		if taskResult.TotalTokens > 0 {
			task.PrivateData.SettlementTotalTokens = taskResult.TotalTokens
		}
		task.PrivateData.SettlementTargetQuota = resolveTaskSettlementTargetQuota(
			adaptor,
			task,
			taskResult,
		)
		task.PrivateData.SettlementTargetReady = true
		task.Status = model.TaskStatusSettlementProcessing
		task.PrivateData.StorageStatus = "settling"
		shouldSettle = true
	case model.TaskStatusFailure:
		task.Status = model.TaskStatusFailure
		task.Progress = taskcommon.ProgressComplete
		if task.FinishTime == 0 {
			task.FinishTime = now
		}
		task.FailReason = sanitizeTaskFailureReason(taskResult.Reason)
		logger.LogInfo(ctx, fmt.Sprintf("Task %s reported a provider failure", task.TaskID))
		taskResult.Progress = taskcommon.ProgressComplete
		if taskResult.NoRefund {
			// 适配器判定需人工介入（如上游返回未知终态）：保留预扣额度，不自动退款。
			task.PrivateData.NoAutomaticRefund = true
			task.PrivateData.StorageStatus = model.TaskStorageStatusProviderReview
			task.PrivateData.StorageLastError = task.FailReason
			if quota != 0 {
				logger.LogWarn(ctx, fmt.Sprintf(
					"Task %s failed but refund is withheld pending manual review, quota %d retained",
					task.TaskID, quota,
				))
			}
		} else if quota != 0 {
			shouldRefund = true
			task.MarkRefundPending(task.FailReason)
		}
	default:
		return fmt.Errorf("unknown task status %s for task %s", taskResult.Status, task.TaskID)
	}
	if taskResult.Progress != "" {
		task.Progress = taskResult.Progress
	}
	if task.Status == model.TaskStatusStoring {
		task.Progress = "99%"
	}

	isDone := task.Status == model.TaskStatusSettlementProcessing ||
		task.Status == model.TaskStatusFailure
	if isDone && snap.Status != task.Status {
		won, err := task.UpdateWithStatus(snap.Status)
		if err != nil {
			logger.LogError(ctx, fmt.Sprintf("UpdateWithStatus failed for task %s: %s", task.TaskID, err.Error()))
			shouldRefund = false
			shouldSettle = false
		} else if !won {
			logger.LogWarn(ctx, fmt.Sprintf("Task %s CAS lost or no-op update, skip billing", task.TaskID))
			shouldRefund = false
			shouldSettle = false
		}
	} else if !snap.Equal(task.Snapshot()) {
		if _, err := task.UpdateWithStatus(snap.Status); err != nil {
			logger.LogError(ctx, fmt.Sprintf("Failed to update task %s: %s", task.TaskID, err.Error()))
		}
	} else {
		// No changes, skip update
		logger.LogDebug(ctx, "No update needed for task %s", task.TaskID)
	}

	if shouldSettle {
		logger.LogInfo(ctx, fmt.Sprintf(
			"task_lifecycle stage=provider_success task_id=%s channel_id=%d",
			task.TaskID,
			task.ChannelId,
		))
		if err := settleTaskBillingOnComplete(ctx, adaptor, task, taskResult); err != nil {
			task.Status = model.TaskStatusFailure
			task.Progress = taskcommon.ProgressComplete
			task.FinishTime = time.Now().Unix()
			task.FailReason = "Video billing settlement requires administrator review"
			task.PrivateData.NoAutomaticRefund = true
			task.PrivateData.StorageStatus = model.TaskStorageStatusProviderReview
			task.PrivateData.StorageLastError = err.Error()
			won, updateErr := task.UpdateWithStatus(model.TaskStatusSettlementProcessing)
			if updateErr != nil {
				return fmt.Errorf("persist video settlement failure: %w", updateErr)
			}
			if !won {
				return fmt.Errorf("video task %s changed while settlement failed", task.TaskID)
			}
			return nil
		}
		task.Status = model.TaskStatusStoring
		task.PrivateData.StorageStatus = "pending"
		won, err := task.UpdateWithStatus(model.TaskStatusSettlementProcessing)
		if err != nil {
			return fmt.Errorf("expose settled video task for storage: %w", err)
		}
		if !won {
			return fmt.Errorf(
				"video task %s changed while billing settlement completed",
				task.TaskID,
			)
		}
	}
	if shouldRefund {
		RefundTaskQuota(ctx, task, task.FailReason)
	}

	return nil
}

func hasUsableVideoResultSource(ch *model.Channel, task *model.Task, resultURL string) bool {
	if strings.TrimSpace(resultURL) != "" {
		return true
	}
	if ch == nil || task == nil || strings.TrimSpace(task.GetUpstreamTaskID()) == "" {
		return false
	}
	return ch.Type == constant.ChannelTypeOpenAI || ch.Type == constant.ChannelTypeSora
}

func redactVideoResponseBody(body []byte) []byte {
	var m map[string]any
	if err := common.Unmarshal(body, &m); err != nil {
		return body
	}
	resp, _ := m["response"].(map[string]any)
	if resp != nil {
		delete(resp, "bytesBase64Encoded")
		if v, ok := resp["video"].(string); ok {
			resp["video"] = truncateBase64(v)
		}
		if vs, ok := resp["videos"].([]any); ok {
			for i := range vs {
				if vm, ok := vs[i].(map[string]any); ok {
					delete(vm, "bytesBase64Encoded")
				}
			}
		}
	}
	b, err := common.Marshal(m)
	if err != nil {
		return body
	}
	return b
}

func truncateBase64(s string) string {
	const maxKeep = 256
	if len(s) <= maxKeep {
		return s
	}
	return s[:maxKeep] + "..."
}

// settleTaskBillingOnComplete 任务完成时的统一计费调整。
// 优先级：1. adaptor.AdjustBillingOnComplete 返回正数 → 使用 adaptor 计算的额度
//
//  2. taskResult.TotalTokens > 0 → 按 token 重算
//  3. 都不满足 → 保持预扣额度不变
func settleTaskBillingOnComplete(ctx context.Context, adaptor TaskPollingAdaptor, task *model.Task, taskResult *relaycommon.TaskInfo) error {
	actualQuota := task.PrivateData.SettlementTargetQuota
	if !task.PrivateData.SettlementTargetReady && actualQuota <= 0 {
		// Compatibility for tasks claimed before settlement targets were
		// persisted. The pre-consumed quota is the safe fallback.
		actualQuota = resolveTaskSettlementTargetQuota(adaptor, task, taskResult)
	}
	return RecalculateTaskQuota(ctx, task, actualQuota, "video completion settlement")
}

func resolveTaskSettlementTargetQuota(
	adaptor TaskPollingAdaptor,
	task *model.Task,
	taskResult *relaycommon.TaskInfo,
) int {
	targetQuota := task.Quota
	if billing := task.PrivateData.BillingContext; billing != nil && billing.PerCallBilling {
		return targetQuota
	}
	if adaptor != nil && taskResult != nil {
		if adjustedQuota := adaptor.AdjustBillingOnComplete(task, taskResult); adjustedQuota > 0 {
			return adjustedQuota
		}
	}
	totalTokens := task.PrivateData.SettlementTotalTokens
	if totalTokens <= 0 && taskResult != nil {
		totalTokens = taskResult.TotalTokens
	}
	if actualQuota, _, ok := calculateTaskQuotaByTokens(task, totalTokens); ok {
		return actualQuota
	}
	return targetQuota
}
