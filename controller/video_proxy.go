package controller

import (
	"fmt"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

func videoProxyError(c *gin.Context, status int, errType, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"message": message,
			"type":    errType,
		},
	})
}

func VideoProxy(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		videoProxyError(c, http.StatusBadRequest, "invalid_request_error", "task_id is required")
		return
	}

	userID := c.GetInt("id")
	isAdmin := c.GetInt("role") >= common.RoleAdminUser || model.IsAdmin(userID)
	var task *model.Task
	var exists bool
	var err error
	if isAdmin {
		task, exists, err = model.GetVideoTaskByTaskID(taskID)
	} else {
		task, exists, err = model.GetByTaskId(userID, taskID)
	}
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to query task %s: %s", taskID, err.Error()))
		videoProxyError(c, http.StatusInternalServerError, "server_error", "Failed to query task")
		return
	}
	if !exists || task == nil {
		videoProxyError(c, http.StatusNotFound, "invalid_request_error", "Task not found")
		return
	}
	if isAdmin && task.UserId != userID {
		recordManageAuditFor(c, task.UserId, "video.preview", map[string]interface{}{
			"task_id":     task.TaskID,
			"task_status": task.Status,
		})
	}
	if task.Status != model.TaskStatusSuccess {
		videoProxyError(c, http.StatusBadRequest, "invalid_request_error",
			fmt.Sprintf("Task is not completed yet, current status: %s", task.Status))
		return
	}

	// Video delivery is storage-only. Never redirect or proxy a client to an
	// upstream result, including legacy tasks without stored-object metadata.
	serveSilkRoadVideoContent(c, task)
}
