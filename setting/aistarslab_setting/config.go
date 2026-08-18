package aistarslab_setting

import (
	"github.com/QuantumNous/new-api/setting/config"
)

// AIStarsLabSetting holds administrator overrides for public video-tool
// capabilities. Each row is keyed by the New API model name configured on the
// channel, not the upstream {channelCode}:{model} identifier.
type AIStarsLabSetting struct {
	Profiles []ModelOverride `json:"profiles"`
}

type ModelOverride struct {
	Model       string   `json:"model"`
	Resolutions []string `json:"resolutions"`
}

var aiStarsLabSetting = defaultAIStarsLabSetting()

func init() {
	config.GlobalConfig.Register("aistarslab_setting", &aiStarsLabSetting)
}

func GetAIStarsLabSetting() *AIStarsLabSetting {
	return &aiStarsLabSetting
}

func DefaultAIStarsLabSetting() AIStarsLabSetting {
	return defaultAIStarsLabSetting()
}

func defaultAIStarsLabSetting() AIStarsLabSetting {
	return AIStarsLabSetting{
		Profiles: []ModelOverride{},
	}
}
