package controller

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// serveSilkRoadVideoContent serves a SilkRoad-stored video from local disk.
// It never redirects or proxies the client to UpstreamResultURL.
func serveSilkRoadVideoContent(c *gin.Context, task *model.Task) {
	status := strings.TrimSpace(task.PrivateData.StorageStatus)
	expiresAt := task.PrivateData.StorageExpiresAt
	if status == "expired" || (expiresAt > 0 && time.Now().Unix() >= expiresAt) {
		videoProxyError(c, http.StatusGone, "invalid_request_error", "Video has expired")
		return
	}

	switch status {
	case "pending", "failed":
		videoProxyError(c, http.StatusConflict, "invalid_request_error",
			"Video is still being processed")
		return
	case "ready":
		f, err := service.OpenSilkRoadVideoFile(task.TaskID)
		if err != nil {
			if os.IsNotExist(err) {
				videoProxyError(c, http.StatusNotFound, "invalid_request_error", "Video file not found")
				return
			}
			videoProxyError(c, http.StatusInternalServerError, "server_error", "Failed to open video file")
			return
		}
		defer f.Close()

		stat, err := f.Stat()
		if err != nil {
			videoProxyError(c, http.StatusInternalServerError, "server_error", "Failed to read video file")
			return
		}

		c.Header("Content-Type", "video/mp4")
		c.Header("Cache-Control", "public, max-age=86400")
		http.ServeContent(c.Writer, c.Request, task.TaskID+".mp4", stat.ModTime(), f)
		return
	default:
		videoProxyError(c, http.StatusConflict, "invalid_request_error",
			"Video is still being processed")
	}
}
