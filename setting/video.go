package setting

import (
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"
	"github.com/QuantumNous/new-api/setting/video_setting"
)

func GetEffectiveVideoSetting() video_setting.EffectiveSetting {
	configured := *video_setting.GetVideoSetting()
	legacy := silkroad_setting.GetSilkRoadSetting()
	legacyPublic := silkroad_setting.GetPublicVideoToolConfig()

	return video_setting.ResolveEffectiveSetting(
		configured,
		video_setting.ExplicitFields{
			Enabled:         config.GlobalConfig.IsExplicit("video_setting.enabled"),
			VideoToolGroups: config.GlobalConfig.IsExplicit("video_setting.video_tool_groups"),
			Storage:         config.GlobalConfig.IsExplicit("video_setting.storage"),
		},
		video_setting.LegacySetting{
			ToolEnabled:     legacyPublic.Enabled,
			VideoToolGroups: legacy.VideoToolGroups,
			StorageEnabled:  legacy.Storage.Enabled,
			Storage: video_setting.StorageSetting{
				Driver:                legacy.Storage.Driver,
				LocalDir:              legacy.Storage.LocalDir,
				MaxRetry:              legacy.Storage.MaxRetry,
				IngestNodeName:        legacy.Storage.IngestNodeName,
				PublicDownloadBaseURL: legacy.Storage.PublicDownloadBaseURL,
				LocalRetentionDays:    legacy.Storage.RetentionDays,
			},
		},
	)
}
