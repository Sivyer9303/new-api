package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"

	"github.com/gin-gonic/gin"
)

// GetSilkRoadVideoToolConfig returns sanitized SilkRoad video profiles for the
// logged-in Seedance-style video tool UI (no storage secrets).
func GetSilkRoadVideoToolConfig(c *gin.Context) {
	cfg := silkroad_setting.GetPublicVideoToolConfig()
	common.ApiSuccess(c, cfg)
}
