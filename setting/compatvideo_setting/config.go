package compatvideo_setting

import (
	"github.com/QuantumNous/new-api/setting/config"
)

// CompatVideoSetting holds administrator overrides for the built-in Compatible
// Video profiles. For each profile only the fields the administrator actually
// fills in override the built-in defaults; empty values leave the built-in
// capability untouched. Keep this surface minimal on purpose — extend it with
// new fields as new capabilities are needed.
type CompatVideoSetting struct {
	// Profiles keys each override set by the built-in profile ID
	// (seedance2 / grok-image-video / grok-video-1.5 / unknown).
	Profiles []Profile `json:"profiles"`
}

var compatVideoSetting = defaultCompatVideoSetting()

func init() {
	config.GlobalConfig.Register("compatvideo_setting", &compatVideoSetting)
}

func GetCompatVideoSetting() *CompatVideoSetting {
	return &compatVideoSetting
}

func DefaultCompatVideoSetting() CompatVideoSetting {
	return defaultCompatVideoSetting()
}

func defaultCompatVideoSetting() CompatVideoSetting {
	return CompatVideoSetting{
		Profiles: []Profile{},
	}
}
