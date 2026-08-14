package model

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	commonRelay "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
)

var ErrAmbiguousTaskID = errors.New("task_id matches multiple tasks")

type TaskStatus string

func (t TaskStatus) ToVideoStatus() string {
	var status string
	switch t {
	case TaskStatusSubmitting, TaskStatusQueued, TaskStatusSubmitted:
		status = dto.VideoStatusQueued
	case TaskStatusInProgress, TaskStatusSettlementProcessing, TaskStatusSettlementRecovering, TaskStatusStoring, TaskStatusStorageProcessing, TaskStatusStorageDeleting:
		status = dto.VideoStatusInProgress
	case TaskStatusSuccess:
		status = dto.VideoStatusCompleted
	case TaskStatusFailure, TaskStatusExpired, TaskStatusRefunded:
		status = dto.VideoStatusFailed
	default:
		status = dto.VideoStatusUnknown // Default fallback
	}
	return status
}

const (
	TaskStatusNotStart             TaskStatus = "NOT_START"
	TaskStatusSubmitting           TaskStatus = "SUBMITTING"
	TaskStatusSubmitted                       = "SUBMITTED"
	TaskStatusQueued                          = "QUEUED"
	TaskStatusInProgress                      = "IN_PROGRESS"
	TaskStatusSettlementProcessing TaskStatus = "SETTLING"
	TaskStatusSettlementRecovering TaskStatus = "SETTLING_RECOVERY"
	TaskStatusStoring              TaskStatus = "STORING"
	TaskStatusStorageProcessing    TaskStatus = "STORAGE_PROCESSING"
	TaskStatusStorageDeleting      TaskStatus = "STORAGE_DELETING"
	TaskStatusFailure                         = "FAILURE"
	TaskStatusSuccess                         = "SUCCESS"
	TaskStatusExpired              TaskStatus = "EXPIRED"
	TaskStatusRefunded             TaskStatus = "REFUNDED"
	TaskStatusUnknown                         = "UNKNOWN"
)

const TaskStorageStatusProviderReview = "provider_review"

// TaskRefundLegacyCutoff separates tasks created before timeout refunds were
// introduced. Those legacy tasks are failed without an automatic refund.
const TaskRefundLegacyCutoff int64 = 1771718400 // 2026-02-22 00:00:00 UTC

type Task struct {
	ID             int64                 `json:"id" gorm:"primary_key;AUTO_INCREMENT"`
	CreatedAt      int64                 `json:"created_at" gorm:"index"`
	UpdatedAt      int64                 `json:"updated_at"`
	TaskID         string                `json:"task_id" gorm:"type:varchar(191);index"` // 第三方id，不一定有/ song id\ Task id
	Platform       constant.TaskPlatform `json:"platform" gorm:"type:varchar(30);index"` // 平台
	UserId         int                   `json:"user_id" gorm:"index"`
	Group          string                `json:"group" gorm:"type:varchar(50)"` // 修正计费用
	ChannelId      int                   `json:"channel_id" gorm:"index"`
	Quota          int                   `json:"quota"`
	RefundPending  bool                  `json:"-" gorm:"index:idx_task_refund_pending"`
	RefundRetryAt  int64                 `json:"-" gorm:"index:idx_task_refund_pending"`
	RefundAttempts int                   `json:"-"`
	Action         string                `json:"action" gorm:"type:varchar(40);index"` // 任务类型, song, lyrics, description-mode
	Status         TaskStatus            `json:"status" gorm:"type:varchar(20);index"` // 任务状态
	FailReason     string                `json:"fail_reason"`
	SubmitTime     int64                 `json:"submit_time" gorm:"index"`
	StartTime      int64                 `json:"start_time" gorm:"index"`
	FinishTime     int64                 `json:"finish_time" gorm:"index"`
	Progress       string                `json:"progress" gorm:"type:varchar(20);index"`
	Properties     Properties            `json:"properties" gorm:"type:json"`
	Username       string                `json:"username,omitempty" gorm:"-"`
	// 禁止返回给用户，内部可能包含key等隐私信息
	PrivateData TaskPrivateData `json:"-" gorm:"column:private_data;type:json"`
	Data        json.RawMessage `json:"data" gorm:"type:json"`
}

func (t *Task) MarkRefundPending(reason string) {
	t.RefundPending = true
	t.RefundRetryAt = 0
	t.RefundAttempts = 0
	t.PrivateData.BillingRefundReason = reason
}

func (t *Task) SetData(data any) {
	b, _ := common.Marshal(data)
	t.Data = json.RawMessage(b)
}

func (t *Task) GetData(v any) error {
	return common.Unmarshal(t.Data, &v)
}

type Properties struct {
	Input             string                           `json:"input,omitempty"`
	UpstreamModelName string                           `json:"upstream_model_name,omitempty"`
	OriginModelName   string                           `json:"origin_model_name,omitempty"`
	Request           *commonRelay.TaskRequestSnapshot `json:"request,omitempty"`
}

func (m *Properties) Scan(val interface{}) error {
	bytesValue, _ := val.([]byte)
	if len(bytesValue) == 0 {
		*m = Properties{}
		return nil
	}
	return common.Unmarshal(bytesValue, m)
}

func (m Properties) Value() (driver.Value, error) {
	if m == (Properties{}) {
		return nil, nil
	}
	return common.Marshal(m)
}

type TaskPrivateData struct {
	Key                string `json:"key,omitempty"`
	UpstreamTaskID     string `json:"upstream_task_id,omitempty"`     // 上游真实 task ID
	ResultURL          string `json:"result_url,omitempty"`           // 任务成功后的结果 URL（视频地址等）
	UpstreamResultURL  string `json:"upstream_result_url,omitempty"`  // 上游原始结果 URL（转存前）
	StorageStatus      string `json:"storage_status,omitempty"`       // pending|processing|ready|failed|provider_review|expired
	StoragePath        string `json:"storage_path,omitempty"`         // Deprecated: existing local path compatibility
	StorageObjectKey   string `json:"storage_object_key,omitempty"`   // Provider-neutral driver object key
	StorageContentType string `json:"storage_content_type,omitempty"` // Stored object content type
	StorageSize        int64  `json:"storage_size,omitempty"`         // Stored object size in bytes
	StorageReadyAt     int64  `json:"storage_ready_at,omitempty"`     // Storage readiness time (Unix seconds)
	StorageExpiresAt   int64  `json:"storage_expires_at,omitempty"`   // Fixed expiry time (Unix seconds)
	StorageRetryCount  int    `json:"storage_retry_count,omitempty"`  // Transfer attempts
	StorageLastError   string `json:"storage_last_error,omitempty"`   // Administrator-only transfer diagnostic
	VideoTask          bool   `json:"video_task,omitempty"`           // Persisted task kind for storage/privacy policy
	NoAutomaticRefund  bool   `json:"no_automatic_refund,omitempty"`  // Provider cost retained after delivery failure
	ManualRefundedAt   int64  `json:"manual_refunded_at,omitempty"`
	ManualRefundAdmin  int    `json:"manual_refund_admin,omitempty"`
	ManualRefundReason string `json:"manual_refund_reason,omitempty"`
	ManualRefundQuota  int    `json:"manual_refund_quota,omitempty"`
	// 计费上下文：用于异步退款/差额结算（轮询阶段读取）
	BillingSource            string              `json:"billing_source,omitempty"`  // "wallet" 或 "subscription"
	SubscriptionId           int                 `json:"subscription_id,omitempty"` // 订阅 ID，用于订阅退款
	TokenId                  int                 `json:"token_id,omitempty"`        // 令牌 ID，用于令牌额度退款
	NodeName                 string              `json:"node_name,omitempty"`       // 发起任务的节点名，轮询结算阶段据此归属日志而非最后查询节点
	BillingContext           *TaskBillingContext `json:"billing_context,omitempty"` // 计费参数快照（用于轮询阶段重新计算）
	BillingReservationTarget int                 `json:"billing_reservation_target,omitempty"`
	BillingRefundReason      string              `json:"billing_refund_reason,omitempty"`
	SettlementTargetQuota    int                 `json:"settlement_target_quota,omitempty"`
	SettlementTargetReady    bool                `json:"settlement_target_ready,omitempty"`
	SettlementTotalTokens    int                 `json:"settlement_total_tokens,omitempty"`
	BillingSettlementApplied bool                `json:"billing_settlement_applied,omitempty"`
}

// TaskBillingContext 记录任务提交时的计费参数，以便轮询阶段可以重新计算额度。
type TaskBillingContext struct {
	ModelPrice      float64            `json:"model_price,omitempty"`       // 模型单价
	GroupRatio      float64            `json:"group_ratio,omitempty"`       // 分组倍率
	ModelRatio      float64            `json:"model_ratio,omitempty"`       // 模型倍率
	OtherRatios     map[string]float64 `json:"other_ratios,omitempty"`      // 附加倍率（时长、分辨率等）
	OriginModelName string             `json:"origin_model_name,omitempty"` // 模型名称，必须为OriginModelName
	PerCallBilling  bool               `json:"per_call_billing,omitempty"`  // 按次计费：跳过轮询阶段的差额结算
}

// GetUpstreamTaskID 获取上游真实 task ID（用于与 provider 通信）
// 旧数据没有 UpstreamTaskID 时，TaskID 本身就是上游 ID
func (t *Task) GetUpstreamTaskID() string {
	if t.PrivateData.UpstreamTaskID != "" {
		return t.PrivateData.UpstreamTaskID
	}
	return t.TaskID
}

// GetResultURL 获取任务结果 URL（视频地址等）
// 新数据存在 PrivateData.ResultURL 中；旧数据回退到 FailReason（历史兼容）
func (t *Task) GetResultURL() string {
	if t.PrivateData.ResultURL != "" {
		return t.PrivateData.ResultURL
	}
	return t.FailReason
}

// GenerateTaskID 生成对外暴露的 task_xxxx 格式 ID
func GenerateTaskID() string {
	key, _ := common.GenerateRandomCharsKey(32)
	return "task_" + key
}

func (p *TaskPrivateData) Scan(val interface{}) error {
	bytesValue, _ := val.([]byte)
	if len(bytesValue) == 0 {
		*p = TaskPrivateData{}
		return nil
	}
	*p = TaskPrivateData{}
	return common.Unmarshal(bytesValue, p)
}

func (p TaskPrivateData) Value() (driver.Value, error) {
	if (p == TaskPrivateData{}) {
		return nil, nil
	}
	return common.Marshal(p)
}

// SyncTaskQueryParams 用于包含所有搜索条件的结构体，可以根据需求添加更多字段
type SyncTaskQueryParams struct {
	Platform       constant.TaskPlatform
	ChannelID      string
	TaskID         string
	UserID         string
	Action         string
	Status         string
	StartTimestamp int64
	EndTimestamp   int64
	UserIDs        []int
}

func InitTask(platform constant.TaskPlatform, relayInfo *commonRelay.RelayInfo) *Task {
	properties := Properties{}
	privateData := TaskPrivateData{}
	if relayInfo != nil && relayInfo.ChannelMeta != nil {
		if relayInfo.ChannelMeta.ChannelType == constant.ChannelTypeGemini ||
			relayInfo.ChannelMeta.ChannelType == constant.ChannelTypeVertexAi ||
			relayInfo.ChannelMeta.ChannelType == constant.ChannelTypeSilkRoad ||
			relayInfo.ChannelMeta.ChannelType == constant.ChannelTypeBrioi {
			privateData.Key = relayInfo.ChannelMeta.ApiKey
		}
		if relayInfo.UpstreamModelName != "" {
			properties.UpstreamModelName = relayInfo.UpstreamModelName
		}
		if relayInfo.OriginModelName != "" {
			properties.OriginModelName = relayInfo.OriginModelName
		}
	}
	if relayInfo != nil && relayInfo.TaskRelayInfo != nil && relayInfo.TaskRelayInfo.RequestSnapshot != nil {
		snapshot := *relayInfo.TaskRelayInfo.RequestSnapshot
		properties.Request = &snapshot
	}

	// 使用预生成的公开 ID（如果有），否则新生成
	taskID := ""
	if relayInfo.TaskRelayInfo != nil && relayInfo.TaskRelayInfo.PublicTaskID != "" {
		taskID = relayInfo.TaskRelayInfo.PublicTaskID
	} else {
		taskID = GenerateTaskID()
	}

	t := &Task{
		TaskID:      taskID,
		UserId:      relayInfo.UserId,
		Group:       relayInfo.UsingGroup,
		SubmitTime:  time.Now().Unix(),
		Status:      TaskStatusNotStart,
		Progress:    "0%",
		ChannelId:   relayInfo.ChannelId,
		Platform:    platform,
		Properties:  properties,
		PrivateData: privateData,
	}
	return t
}

func TaskGetAllUserTask(userId int, startIdx int, num int, queryParams SyncTaskQueryParams) []*Task {
	var tasks []*Task
	var err error

	// 初始化查询构建器
	query := DB.Where("user_id = ?", userId)

	if queryParams.TaskID != "" {
		query = query.Where("task_id = ?", queryParams.TaskID)
	}
	if queryParams.Action != "" {
		query = query.Where("action = ?", queryParams.Action)
	}
	if queryParams.Status != "" {
		query = query.Where("status = ?", queryParams.Status)
	}
	if queryParams.Platform != "" {
		query = query.Where("platform = ?", queryParams.Platform)
	}
	if queryParams.StartTimestamp != 0 {
		// 假设您已将前端传来的时间戳转换为数据库所需的时间格式，并处理了时间戳的验证和解析
		query = query.Where("submit_time >= ?", queryParams.StartTimestamp)
	}
	if queryParams.EndTimestamp != 0 {
		query = query.Where("submit_time <= ?", queryParams.EndTimestamp)
	}

	// 获取数据
	err = query.Omit("channel_id").Order("id desc").Limit(num).Offset(startIdx).Find(&tasks).Error
	if err != nil {
		return nil
	}

	return tasks
}

func TaskGetAllTasks(startIdx int, num int, queryParams SyncTaskQueryParams) []*Task {
	var tasks []*Task
	var err error

	// 初始化查询构建器
	query := DB

	// 添加过滤条件
	if queryParams.ChannelID != "" {
		query = query.Where("channel_id = ?", queryParams.ChannelID)
	}
	if queryParams.Platform != "" {
		query = query.Where("platform = ?", queryParams.Platform)
	}
	if queryParams.UserID != "" {
		query = query.Where("user_id = ?", queryParams.UserID)
	}
	if len(queryParams.UserIDs) != 0 {
		query = query.Where("user_id in (?)", queryParams.UserIDs)
	}
	if queryParams.TaskID != "" {
		query = query.Where("task_id = ?", queryParams.TaskID)
	}
	if queryParams.Action != "" {
		query = query.Where("action = ?", queryParams.Action)
	}
	if queryParams.Status != "" {
		query = query.Where("status = ?", queryParams.Status)
	}
	if queryParams.StartTimestamp != 0 {
		query = query.Where("submit_time >= ?", queryParams.StartTimestamp)
	}
	if queryParams.EndTimestamp != 0 {
		query = query.Where("submit_time <= ?", queryParams.EndTimestamp)
	}

	// 获取数据
	err = query.Order("id desc").Limit(num).Offset(startIdx).Find(&tasks).Error
	if err != nil {
		return nil
	}

	return tasks
}

func GetTimedOutUnfinishedTasks(cutoffUnix int64, limit int) []*Task {
	var tasks []*Task
	err := DB.Where("progress != ?", "100%").
		Where("status NOT IN ?", videoStorageOwnedTaskStatuses()).
		Where("submit_time < ?", cutoffUnix).
		Order("submit_time").
		Limit(limit).
		Find(&tasks).Error
	if err != nil {
		return nil
	}
	return tasks
}

func GetAllUnFinishSyncTasks(limit int) []*Task {
	var tasks []*Task
	var err error
	// get all tasks progress is not 100%
	err = DB.Where("progress != ?", "100%").
		Where("status NOT IN ?", videoStorageOwnedTaskStatuses()).
		Limit(limit).
		Order("id").
		Find(&tasks).Error
	if err != nil {
		return nil
	}
	return tasks
}

// HasUnfinishedSyncTasks reports whether at least one async (Suno/video) task is
// still in progress. It is a cheap existence check (LIMIT 1) used to decide
// whether the async_task_poll system task needs to run; when no task is pending
// the scheduler skips creating a row entirely.
func HasUnfinishedSyncTasks() bool {
	var id int64
	err := DB.Model(&Task{}).
		Where("progress != ?", "100%").
		Where("status NOT IN ?", videoStorageOwnedTaskStatuses()).
		Limit(1).
		Pluck("id", &id).Error
	return err == nil && id != 0
}

func videoStorageOwnedTaskStatuses() []TaskStatus {
	return []TaskStatus{
		TaskStatusFailure,
		TaskStatusSuccess,
		TaskStatusSettlementProcessing,
		TaskStatusSettlementRecovering,
		TaskStatusStoring,
		TaskStatusStorageProcessing,
		TaskStatusStorageDeleting,
		TaskStatusExpired,
		TaskStatusRefunded,
	}
}

func GetByTaskId(userId int, taskId string) (*Task, bool, error) {
	if taskId == "" {
		return nil, false, nil
	}
	var task *Task
	var err error
	err = DB.Where("user_id = ? and task_id = ?", userId, taskId).
		First(&task).Error
	exist, err := RecordExist(err)
	if err != nil {
		return nil, false, err
	}
	return task, exist, err
}

func GetVideoTaskByTaskID(taskID string) (*Task, bool, error) {
	if taskID == "" {
		return nil, false, nil
	}
	var tasks []Task
	if err := DB.Where("task_id = ?", taskID).Limit(2).Find(&tasks).Error; err != nil {
		return nil, false, err
	}
	if len(tasks) == 0 {
		return nil, false, nil
	}
	if len(tasks) > 1 {
		return nil, false, ErrAmbiguousTaskID
	}
	return &tasks[0], true, nil
}

func GetByTaskIds(userId int, taskIds []any) ([]*Task, error) {
	if len(taskIds) == 0 {
		return nil, nil
	}
	var task []*Task
	var err error
	err = DB.Where("user_id = ? and task_id in (?)", userId, taskIds).
		Find(&task).Error
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (Task *Task) Insert() error {
	var err error
	err = DB.Create(Task).Error
	return err
}

type taskSnapshot struct {
	Status     TaskStatus
	Progress   string
	StartTime  int64
	FinishTime int64
	FailReason string
	ResultURL  string
	Data       json.RawMessage
}

func (s taskSnapshot) Equal(other taskSnapshot) bool {
	return s.Status == other.Status &&
		s.Progress == other.Progress &&
		s.StartTime == other.StartTime &&
		s.FinishTime == other.FinishTime &&
		s.FailReason == other.FailReason &&
		s.ResultURL == other.ResultURL &&
		bytes.Equal(s.Data, other.Data)
}

func (t *Task) Snapshot() taskSnapshot {
	return taskSnapshot{
		Status:     t.Status,
		Progress:   t.Progress,
		StartTime:  t.StartTime,
		FinishTime: t.FinishTime,
		FailReason: t.FailReason,
		ResultURL:  t.PrivateData.ResultURL,
		Data:       t.Data,
	}
}

func (Task *Task) Update() error {
	var err error
	err = DB.Save(Task).Error
	return err
}

func (t *Task) UpdateQuota() error {
	return DB.Model(t).Update("quota", t.Quota).Error
}

// UpdateWithStatus performs a conditional UPDATE guarded by fromStatus (CAS).
// Returns (true, nil) if this caller won the update, (false, nil) if
// another process already moved the task out of fromStatus. MySQL commonly
// reports changed rows rather than matched rows, so a same-value no-op update
// can also return false even when the status predicate still matched.
//
// Uses Model().Select("*").Updates() instead of Save() because GORM's Save
// falls back to INSERT ON CONFLICT when the WHERE-guarded UPDATE matches
// zero rows, which silently bypasses the CAS guard.
func (t *Task) UpdateWithStatus(fromStatus TaskStatus) (bool, error) {
	result := DB.Model(t).Where("status = ?", fromStatus).Select("*").Updates(t)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// UpdatePrivateDataIfStatus writes private_data and data only while the row is
// still in fromStatus. Used when a full CAS may have failed after the provider
// already returned a task ID.
func (t *Task) UpdatePrivateDataIfStatus(fromStatus TaskStatus) (bool, error) {
	if t == nil {
		return false, errors.New("task is nil")
	}
	result := DB.Model(t).
		Where("id = ? AND status = ?", t.ID, fromStatus).
		Updates(map[string]any{
			"private_data": t.PrivateData,
			"data":         t.Data,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// ClaimWithStatusAndUpdatedAt leases a task revision by guarding both its
// status and update timestamp. It also works when fromStatus equals toStatus,
// which lets watchdogs renew a stale recovery row without two workers owning
// the same revision.
func (t *Task) ClaimWithStatusAndUpdatedAt(
	fromStatus TaskStatus,
	toStatus TaskStatus,
) (bool, error) {
	expectedUpdatedAt := t.UpdatedAt
	nextUpdatedAt := time.Now().Unix()
	if nextUpdatedAt <= expectedUpdatedAt {
		nextUpdatedAt = expectedUpdatedAt + 1
	}
	result := DB.Model(&Task{}).
		Where("id = ? AND status = ? AND updated_at = ?", t.ID, fromStatus, expectedUpdatedAt).
		Updates(map[string]any{
			"status":     toStatus,
			"updated_at": nextUpdatedAt,
		})
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 0 {
		return false, nil
	}
	t.Status = toStatus
	t.UpdatedAt = nextUpdatedAt
	return true, nil
}

// UpdateWithStatusAndUpdatedAt commits only the recovery lease revision held
// by this task. A worker whose lease was renewed by another process cannot
// overwrite that process's settlement result.
func (t *Task) UpdateWithStatusAndUpdatedAt(
	fromStatus TaskStatus,
	expectedUpdatedAt int64,
) (bool, error) {
	nextUpdatedAt := time.Now().Unix()
	if nextUpdatedAt <= expectedUpdatedAt {
		nextUpdatedAt = expectedUpdatedAt + 1
	}
	t.UpdatedAt = nextUpdatedAt
	result := DB.Model(&Task{}).
		Where("id = ? AND status = ? AND updated_at = ?", t.ID, fromStatus, expectedUpdatedAt).
		Select("*").
		Updates(t)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// TaskBulkUpdateByID performs an unconditional bulk UPDATE by primary key IDs.
// WARNING: This function has NO CAS (Compare-And-Swap) guard — it will overwrite
// any concurrent status changes. DO NOT use in billing/quota lifecycle flows
// (e.g., timeout, success, failure transitions that trigger refunds or settlements).
// For status transitions that involve billing, use Task.UpdateWithStatus() instead.
func TaskBulkUpdateByID(ids []int64, params map[string]any) error {
	if len(ids) == 0 {
		return nil
	}
	return DB.Model(&Task{}).
		Where("id in (?)", ids).
		Updates(params).Error
}

type TaskQuotaUsage struct {
	Mode  string  `json:"mode"`
	Count float64 `json:"count"`
}

// TaskCountAllTasks returns total tasks that match the given query params (admin usage)
func TaskCountAllTasks(queryParams SyncTaskQueryParams) int64 {
	var total int64
	query := DB.Model(&Task{})
	if queryParams.ChannelID != "" {
		query = query.Where("channel_id = ?", queryParams.ChannelID)
	}
	if queryParams.Platform != "" {
		query = query.Where("platform = ?", queryParams.Platform)
	}
	if queryParams.UserID != "" {
		query = query.Where("user_id = ?", queryParams.UserID)
	}
	if len(queryParams.UserIDs) != 0 {
		query = query.Where("user_id in (?)", queryParams.UserIDs)
	}
	if queryParams.TaskID != "" {
		query = query.Where("task_id = ?", queryParams.TaskID)
	}
	if queryParams.Action != "" {
		query = query.Where("action = ?", queryParams.Action)
	}
	if queryParams.Status != "" {
		query = query.Where("status = ?", queryParams.Status)
	}
	if queryParams.StartTimestamp != 0 {
		query = query.Where("submit_time >= ?", queryParams.StartTimestamp)
	}
	if queryParams.EndTimestamp != 0 {
		query = query.Where("submit_time <= ?", queryParams.EndTimestamp)
	}
	_ = query.Count(&total).Error
	return total
}

// TaskCountAllUserTask returns total tasks for given user
func TaskCountAllUserTask(userId int, queryParams SyncTaskQueryParams) int64 {
	var total int64
	query := DB.Model(&Task{}).Where("user_id = ?", userId)
	if queryParams.TaskID != "" {
		query = query.Where("task_id = ?", queryParams.TaskID)
	}
	if queryParams.Action != "" {
		query = query.Where("action = ?", queryParams.Action)
	}
	if queryParams.Status != "" {
		query = query.Where("status = ?", queryParams.Status)
	}
	if queryParams.Platform != "" {
		query = query.Where("platform = ?", queryParams.Platform)
	}
	if queryParams.StartTimestamp != 0 {
		query = query.Where("submit_time >= ?", queryParams.StartTimestamp)
	}
	if queryParams.EndTimestamp != 0 {
		query = query.Where("submit_time <= ?", queryParams.EndTimestamp)
	}
	_ = query.Count(&total).Error
	return total
}
func (t *Task) ToOpenAIVideo() *dto.OpenAIVideo {
	openAIVideo := dto.NewOpenAIVideo()
	openAIVideo.ID = t.TaskID
	openAIVideo.Status = t.Status.ToVideoStatus()
	openAIVideo.Model = t.Properties.OriginModelName
	openAIVideo.SetProgressStr(t.Progress)
	openAIVideo.CreatedAt = t.CreatedAt
	openAIVideo.CompletedAt = t.UpdatedAt
	openAIVideo.SetMetadata("url", t.GetResultURL())
	return openAIVideo
}
