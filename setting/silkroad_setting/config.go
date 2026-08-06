package silkroad_setting

import (
	"github.com/QuantumNous/new-api/setting/config"
)

// OptionItem is a selectable profile option (duration, aspect ratio).
type OptionItem struct {
	Label       string `json:"label"`
	Value       string `json:"value"`
	UpstreamKey string `json:"upstream_key"`
	Enabled     bool   `json:"enabled"`
	Sort        int    `json:"sort"`
}

// Profile is a SilkRoad model family configuration (durations / ratios / prefixes).
// Generation modes are hardcoded in generation_modes.go — not stored per profile.
type Profile struct {
	ID            string       `json:"id"`
	Label         string       `json:"label"`
	ModelPrefixes []string     `json:"model_prefixes"`
	Durations     []OptionItem `json:"durations"`
	AspectRatios  []OptionItem `json:"aspect_ratios"`
}

// StorageSetting configures local video ingest and retention.
type StorageSetting struct {
	Enabled               bool   `json:"enabled"`
	Driver                string `json:"driver"`
	LocalDir              string `json:"local_dir"`
	RetentionDays         int    `json:"retention_days"`
	MaxRetry              int    `json:"max_retry"`
	IngestNodeName        string `json:"ingest_node_name"`
	PublicDownloadBaseURL string `json:"public_download_base_url"`
}

// SilkRoadSetting is the top-level silkroad_setting config module.
type SilkRoadSetting struct {
	Profiles        []Profile      `json:"profiles"`
	Storage         StorageSetting `json:"storage"`
	VideoToolGroups []string       `json:"video_tool_groups"` // groups whose API keys may be used in Seedance tool; empty = none
}

var silkRoadSetting = defaultSilkRoadSetting()

func init() {
	config.GlobalConfig.Register("silkroad_setting", &silkRoadSetting)
}

// GetSilkRoadSetting returns the live silkroad_setting instance.
func GetSilkRoadSetting() *SilkRoadSetting {
	return &silkRoadSetting
}

func defaultSilkRoadSetting() SilkRoadSetting {
	return SilkRoadSetting{
		Profiles: []Profile{
			{
				ID:            "seedance_reverse",
				Label:         "逆向低价",
				ModelPrefixes: []string{"seedance-2.0-"},
				Durations: []OptionItem{
					{Label: "10 秒", Value: "10", UpstreamKey: "seconds", Enabled: true, Sort: 1},
					{Label: "15 秒", Value: "15", UpstreamKey: "seconds", Enabled: true, Sort: 2},
				},
				AspectRatios: defaultAspectRatios(),
			},
			{
				ID:            "dreamina_overseas",
				Label:         "海外满血",
				ModelPrefixes: []string{"dreamina-seedance-2-0-"},
				Durations: []OptionItem{
					{Label: "4 秒", Value: "4", UpstreamKey: "seconds", Enabled: true, Sort: 1},
					{Label: "5 秒", Value: "5", UpstreamKey: "seconds", Enabled: true, Sort: 2},
				},
				AspectRatios: defaultAspectRatios(),
			},
		},
		Storage: StorageSetting{
			// Default off so misconfigured installs never attempt local store
			// or expose incomplete public/ingest wiring unexpectedly.
			Enabled:               false,
			Driver:                "local",
			LocalDir:              "data/silkroad-videos",
			RetentionDays:         7,
			MaxRetry:              5,
			IngestNodeName:        "",
			PublicDownloadBaseURL: "",
		},
		// Empty by default: Seedance tool shows no keys until admins opt in.
		VideoToolGroups: []string{},
	}
}

func defaultAspectRatios() []OptionItem {
	return []OptionItem{
		{Label: "16:9 横屏", Value: "16:9", UpstreamKey: "aspect_ratio", Enabled: true, Sort: 1},
		{Label: "9:16 竖屏", Value: "9:16", UpstreamKey: "aspect_ratio", Enabled: true, Sort: 2},
		{Label: "1:1 方形", Value: "1:1", UpstreamKey: "aspect_ratio", Enabled: true, Sort: 3},
		{Label: "4:3", Value: "4:3", UpstreamKey: "aspect_ratio", Enabled: true, Sort: 4},
		{Label: "3:4", Value: "3:4", UpstreamKey: "aspect_ratio", Enabled: true, Sort: 5},
		{Label: "21:9 超宽", Value: "21:9", UpstreamKey: "aspect_ratio", Enabled: true, Sort: 6},
	}
}
