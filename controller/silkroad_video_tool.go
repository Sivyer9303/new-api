package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/aistarslab_setting"
	"github.com/QuantumNous/new-api/setting/brioi_setting"
	"github.com/QuantumNous/new-api/setting/compatvideo_setting"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"
	"github.com/QuantumNous/new-api/setting/video_setting"

	"github.com/gin-gonic/gin"
)

type publicVideoProviderConfigs struct {
	SilkRoad    silkroad_setting.PublicVideoToolConfig    `json:"silkroad"`
	Brioi       brioi_setting.PublicVideoToolConfig       `json:"brioi"`
	CompatVideo compatvideo_setting.PublicVideoToolConfig `json:"compat_video"`
	AIStarsLab  aistarslab_setting.PublicVideoToolConfig  `json:"aistarslab"`
}

type publicVideoToolConfig struct {
	Version         int                               `json:"version"`
	Enabled         bool                              `json:"enabled"`
	ProviderByGroup map[string]setting.VideoProvider  `json:"provider_by_group"`
	VideoToolGroups []string                          `json:"video_tool_groups"`
	Providers       publicVideoProviderConfigs        `json:"providers"`
	UploadLimits    video_setting.UploadLimitsSetting `json:"upload_limits"`
}

// GetVideoToolConfig returns sanitized provider capabilities for the logged-in
// video generation UI. Storage credentials and provider secrets are excluded.
func GetVideoToolConfig(c *gin.Context) {
	silkRoad := silkroad_setting.GetPublicVideoToolConfig()
	silkRoad.VideoToolGroups = []string{}
	brioi := brioi_setting.GetPublicVideoToolConfig()
	brioi.VideoToolGroups = []string{}
	compatVideo := compatvideo_setting.GetPublicVideoToolConfig()
	aiStarsLab := aistarslab_setting.GetPublicVideoToolConfig()

	groups, err := model.ListEnabledVideoToolGroups()
	if err != nil {
		groups = []string{}
	}

	uploadLimits := setting.GetEffectiveVideoSetting().UploadLimits
	video_setting.NormalizeUploadLimitsSetting(&uploadLimits)

	common.ApiSuccess(c, publicVideoToolConfig{
		Version:         3,
		Enabled:         setting.IsVideoGenerationToolEnabled(),
		ProviderByGroup: map[string]setting.VideoProvider{},
		VideoToolGroups: groups,
		Providers: publicVideoProviderConfigs{
			SilkRoad:    silkRoad,
			Brioi:       brioi,
			CompatVideo: compatVideo,
			AIStarsLab:  aiStarsLab,
		},
		UploadLimits: uploadLimits,
	})
}

func GetSilkRoadVideoToolConfig(c *gin.Context) {
	cfg := silkroad_setting.GetPublicVideoToolConfig()
	effective := setting.GetEffectiveVideoSetting()
	cfg.Enabled = cfg.Enabled && effective.Enabled
	cfg.VideoToolGroups = []string{}
	common.ApiSuccess(c, cfg)
}
