package controller

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type videoRefundRequest struct {
	Reason string `json:"reason"`
}

func getAdminVideoTask(c *gin.Context) (*model.Task, bool) {
	task, exists, err := model.GetVideoTaskByTaskID(c.Param("task_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return nil, false
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "video task not found"})
		return nil, false
	}
	return task, true
}

func GetAdminVideoDiagnostics(c *gin.Context) {
	task, ok := getAdminVideoTask(c)
	if !ok {
		return
	}
	upstreamResultURL := task.PrivateData.UpstreamResultURL
	if task.PrivateData.StorageExpiresAt > 0 &&
		task.PrivateData.StorageExpiresAt <= time.Now().Unix() {
		upstreamResultURL = ""
	}
	recordManageAuditFor(c, task.UserId, "video.diagnostics", map[string]interface{}{
		"task_id": task.TaskID,
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"task_id":          task.TaskID,
			"upstream_task_id": task.GetUpstreamTaskID(),
			"user_id":          task.UserId,
			"channel_id":       task.ChannelId,
			"platform":         task.Platform,
			"status":           task.Status,
			"progress":         task.Progress,
			"fail_reason":      task.FailReason,
			"quota":            task.Quota,
			"created_at":       task.CreatedAt,
			"updated_at":       task.UpdatedAt,
			"finish_time":      task.FinishTime,
			"storage": gin.H{
				"status":              task.PrivateData.StorageStatus,
				"object_key":          task.PrivateData.StorageObjectKey,
				"path":                task.PrivateData.StoragePath,
				"content_type":        task.PrivateData.StorageContentType,
				"size":                task.PrivateData.StorageSize,
				"ready_at":            task.PrivateData.StorageReadyAt,
				"expires_at":          task.PrivateData.StorageExpiresAt,
				"retry_count":         task.PrivateData.StorageRetryCount,
				"last_error":          task.PrivateData.StorageLastError,
				"upstream_result_url": upstreamResultURL,
				"no_automatic_refund": task.PrivateData.NoAutomaticRefund,
			},
			"manual_refund": gin.H{
				"refunded_at": task.PrivateData.ManualRefundedAt,
				"admin_id":    task.PrivateData.ManualRefundAdmin,
				"reason":      task.PrivateData.ManualRefundReason,
				"quota":       task.PrivateData.ManualRefundQuota,
			},
		},
	})
}

func RetryAdminVideoStorage(c *gin.Context) {
	task, ok := getAdminVideoTask(c)
	if !ok {
		return
	}
	updatedTask, updated, err := service.RetryVideoStorage(task.TaskID)
	params := map[string]interface{}{"task_id": task.TaskID, "updated": updated}
	if err != nil {
		params["error"] = err.Error()
	}
	recordManageAuditFor(c, task.UserId, "video.storage_retry", params)
	if err != nil {
		status := http.StatusConflict
		if errors.Is(err, service.ErrVideoStorageExpired) {
			status = http.StatusGone
		}
		c.JSON(status, gin.H{"success": false, "message": err.Error()})
		return
	}
	if !updated {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": "video task changed while storage retry was requested",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"task_id": updatedTask.TaskID,
		"status":  updatedTask.Status,
	}})
}

func ConfirmAdminVideoProviderResult(c *gin.Context) {
	task, ok := getAdminVideoTask(c)
	if !ok {
		return
	}
	result, err := service.ConfirmVideoProviderResult(task)
	params := map[string]interface{}{"task_id": task.TaskID}
	if err != nil {
		params["error"] = err.Error()
	}
	recordManageAuditFor(c, task.UserId, "video.provider_confirm", params)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, service.ErrVideoStorageExpired) {
			status = http.StatusGone
		} else if errors.Is(err, service.ErrVideoProviderConfirmDenied) {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func RefundAdminVideo(c *gin.Context) {
	task, ok := getAdminVideoTask(c)
	if !ok {
		return
	}
	var request videoRefundRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		recordManageAuditFor(c, task.UserId, "video.refund", map[string]interface{}{
			"task_id": task.TaskID,
			"error":   "invalid refund request",
		})
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid refund request"})
		return
	}
	request.Reason = strings.TrimSpace(request.Reason)
	if request.Reason == "" {
		recordManageAuditFor(c, task.UserId, "video.refund", map[string]interface{}{
			"task_id": task.TaskID,
			"error":   "refund reason is required",
		})
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "refund reason is required"})
		return
	}
	result, err := model.RefundVideoDeliveryFailure(
		task.TaskID,
		c.GetInt("id"),
		request.Reason,
	)
	params := map[string]interface{}{
		"task_id": task.TaskID,
		"reason":  request.Reason,
	}
	if err != nil {
		params["error"] = err.Error()
	} else {
		params["quota"] = result.RefundedQuota
		params["already_refunded"] = result.AlreadyRefunded
	}
	recordManageAuditFor(c, task.UserId, "video.refund", params)
	if err != nil {
		status := http.StatusConflict
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"task_id":          task.TaskID,
		"refunded_quota":   result.RefundedQuota,
		"already_refunded": result.AlreadyRefunded,
	}})
}
