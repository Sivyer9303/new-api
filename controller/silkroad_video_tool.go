package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"

	"github.com/gin-gonic/gin"
)

// GetVideoToolConfig returns sanitized provider capabilities for the logged-in
// video generation UI. Storage credentials and provider secrets are excluded.
func GetVideoToolConfig(c *gin.Context) {
	cfg := silkroad_setting.GetPublicVideoToolConfig()
	effective := setting.GetEffectiveVideoSetting()
	cfg.Enabled = cfg.Enabled && effective.Enabled
	cfg.VideoToolGroups = effective.VideoToolGroups
	common.ApiSuccess(c, cfg)
}

func GetSilkRoadVideoToolConfig(c *gin.Context) {
	GetVideoToolConfig(c)
}
