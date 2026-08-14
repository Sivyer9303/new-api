package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/brioi_setting"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"
	"github.com/QuantumNous/new-api/setting/video_setting"

	"github.com/gin-gonic/gin"
)

type publicVideoProviderConfigs struct {
	SilkRoad silkroad_setting.PublicVideoToolConfig `json:"silkroad"`
	Brioi    brioi_setting.PublicVideoToolConfig    `json:"brioi"`
}

type publicVideoToolConfig struct {
	Version         int                               `json:"version"`
	Enabled         bool                              `json:"enabled"`
	ProviderByGroup map[string]setting.VideoProvider  `json:"provider_by_group"`
	Providers       publicVideoProviderConfigs        `json:"providers"`
	UploadLimits    video_setting.UploadLimitsSetting `json:"upload_limits"`
}

// GetVideoToolConfig returns sanitized provider capabilities for the logged-in
// video generation UI. Storage credentials and provider secrets are excluded.
func GetVideoToolConfig(c *gin.Context) {
	silkRoad := silkroad_setting.GetPublicVideoToolConfig()
	silkRoad.VideoToolGroups = setting.GetVideoProviderGroups(setting.VideoProviderSilkRoad)
	brioi := brioi_setting.GetPublicVideoToolConfig()
	brioi.VideoToolGroups = setting.GetVideoProviderGroups(setting.VideoProviderBrioi)

	ownership := setting.GetVideoProviderGroupOwnership()
	publicOwnership := make(map[string]setting.VideoProvider, len(ownership))
	for group, owner := range ownership {
		publicOwnership[group] = owner.Provider
	}

	uploadLimits := setting.GetEffectiveVideoSetting().UploadLimits
	video_setting.NormalizeUploadLimitsSetting(&uploadLimits)

	common.ApiSuccess(c, publicVideoToolConfig{
		Version:         2,
		Enabled:         setting.IsVideoGenerationToolEnabled(),
		ProviderByGroup: publicOwnership,
		Providers: publicVideoProviderConfigs{
			SilkRoad: silkRoad,
			Brioi:    brioi,
		},
		UploadLimits: uploadLimits,
	})
}

func GetSilkRoadVideoToolConfig(c *gin.Context) {
	cfg := silkroad_setting.GetPublicVideoToolConfig()
	effective := setting.GetEffectiveVideoSetting()
	cfg.Enabled = cfg.Enabled && effective.Enabled
	cfg.VideoToolGroups = setting.GetVideoProviderGroups(setting.VideoProviderSilkRoad)
	common.ApiSuccess(c, cfg)
}
