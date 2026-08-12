package controller

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// presignStoredVideoURL resolves the redirect target for object-storage drivers.
// It is a variable so content delivery can be tested without live credentials.
var presignStoredVideoURL = service.PresignStoredVideoURL

// serveSilkRoadVideoContent delivers a stored video: local files stream through
// this handler, object storage answers with a short-lived signed redirect.
// It never redirects or proxies the client to UpstreamResultURL.
func serveSilkRoadVideoContent(c *gin.Context, task *model.Task) {
	status := strings.TrimSpace(task.PrivateData.StorageStatus)
	expiresAt := task.PrivateData.StorageExpiresAt
	if status == "expired" || (expiresAt > 0 && time.Now().Unix() >= expiresAt) {
		videoProxyError(c, http.StatusGone, "invalid_request_error", "Video has expired")
		return
	}

	switch status {
	case "pending", "processing":
		videoProxyError(c, http.StatusConflict, "invalid_request_error",
			"Video is still being processed")
		return
	case "failed":
		videoProxyError(c, http.StatusConflict, "invalid_request_error",
			"Video delivery failed; contact an administrator with the task ID")
		return
	case "ready":
		// Object-storage drivers deliver the file directly: authorize here, then
		// hand the client a short-lived signed URL instead of proxying bytes.
		signedURL, redirects, err := presignStoredVideoURL(c.Request.Context(), task)
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf(
				"failed to sign stored video url task=%s: %s", task.TaskID, err.Error(),
			))
			videoProxyError(c, http.StatusInternalServerError, "server_error", "Failed to open video file")
			return
		}
		if redirects {
			c.Header("Cache-Control", "private, no-store")
			c.Redirect(http.StatusFound, signedURL)
			return
		}

		f, err := service.OpenStoredVideo(c.Request.Context(), task)
		if err != nil {
			if os.IsNotExist(err) {
				if task.ID > 0 {
					_, _ = service.MarkStoredVideoMissing(task, err)
				}
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

		contentType, err := service.NormalizeStoredVideoContentType(
			task.PrivateData.StorageContentType,
		)
		if err != nil {
			if task.ID > 0 {
				_, _ = service.MarkStoredVideoMissing(task, err)
			}
			videoProxyError(c, http.StatusUnsupportedMediaType, "invalid_request_error",
				"Stored content is not a supported video")
			return
		}
		c.Header("Content-Type", contentType)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Cache-Control", "public, max-age=86400")
		http.ServeContent(c.Writer, c.Request, task.TaskID+".mp4", stat.ModTime(), f)
		return
	default:
		videoProxyError(c, http.StatusConflict, "invalid_request_error",
			"Video is still being processed")
	}
}
