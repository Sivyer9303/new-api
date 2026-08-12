package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/video_setting"

	"github.com/gin-gonic/gin"
)

// GetVideoStorageStatus reports the active driver plus the R2 free-tier usage the
// Storage settings page displays. Credentials are never echoed back.
func GetVideoStorageStatus(c *gin.Context) {
	effective := setting.GetEffectiveVideoSetting()
	storage := effective.Storage
	usage := service.GetR2UsageSnapshot()

	readyError := ""
	if err := service.ValidateVideoStorageReady(); err != nil {
		readyError = err.Error()
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"driver":          storage.Driver,
			"storage_enabled": effective.StorageEnabled,
			"retention_days":  storage.RetentionDays(),
			"ready":           readyError == "",
			"ready_error":     readyError,
			"r2": gin.H{
				"configured":       storage.IsR2() && service.ValidateVideoR2StorageConfigured() == nil,
				"bucket":           storage.R2.Bucket,
				"input_ttl_hours":  storage.R2.InputTTLHours,
				"usage_bytes":      usage.UsageBytes,
				"quota_bytes":      usage.QuotaBytes,
				"soft_limit_bytes": usage.SoftLimitBytes,
				"soft_limit_ratio": video_setting.R2SoftLimitRatio,
				"upload_blocked":   usage.Blocked,
				"checked_at":       usage.CheckedAt,
				"last_error":       usage.LastError,
			},
		},
	})
}

// RefreshVideoStorageUsage re-reads R2 usage on demand so administrators do not
// have to wait for the hourly poll after freeing space.
func RefreshVideoStorageUsage(c *gin.Context) {
	if err := service.RefreshR2Usage(c.Request.Context()); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": service.GetR2UsageSnapshot()})
}
