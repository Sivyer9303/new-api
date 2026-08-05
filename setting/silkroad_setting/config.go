package silkroad_setting

import (
	"github.com/QuantumNous/new-api/setting/config"
)

// OptionItem is a selectable profile option (duration, aspect ratio, extra).
type OptionItem struct {
	Label       string `json:"label"`
	Value       string `json:"value"`
	UpstreamKey string `json:"upstream_key"`
	Enabled     bool   `json:"enabled"`
	Sort        int    `json:"sort"`
}

// UpstreamSet maps a friendly field or fixed value onto an upstream JSON key.
type UpstreamSet struct {
	UpstreamKey string `json:"upstream_key"`
	Value       string `json:"value,omitempty"`
	From        string `json:"from,omitempty"`
}

// MediaRequirements constrains image counts for a generation type.
type MediaRequirements struct {
	ImagesMin int `json:"images_min"`
	ImagesMax int `json:"images_max"`
}

// GenerationType describes a generation mode recipe for a profile.
type GenerationType struct {
	Label             string            `json:"label"`
	Value             string            `json:"value"`
	Enabled           bool              `json:"enabled"`
	Sort              int               `json:"sort"`
	RequireRefModel   bool              `json:"require_ref_model"`
	UpstreamSets      []UpstreamSet     `json:"upstream_sets"`
	MediaRequirements MediaRequirements `json:"media_requirements"`
}

// Profile is a SilkRoad model family configuration.
type Profile struct {
	ID              string           `json:"id"`
	Label           string           `json:"label"`
	ModelPrefixes   []string         `json:"model_prefixes"`
	Durations       []OptionItem     `json:"durations"`
	AspectRatios    []OptionItem     `json:"aspect_ratios"`
	GenerationTypes []GenerationType `json:"generation_types"`
	ExtraOptions    []OptionItem     `json:"extra_options"`
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
	Profiles []Profile      `json:"profiles"`
	Storage  StorageSetting `json:"storage"`
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
				GenerationTypes: []GenerationType{
					{
						Label:             "文生视频",
						Value:             "text2video",
						Enabled:           true,
						Sort:              1,
						RequireRefModel:   false,
						UpstreamSets:      []UpstreamSet{},
						MediaRequirements: MediaRequirements{ImagesMin: 0, ImagesMax: 0},
					},
					{
						Label:           "图生视频（单图）",
						Value:           "image2video",
						Enabled:         true,
						Sort:            2,
						RequireRefModel: false,
						UpstreamSets: []UpstreamSet{
							{UpstreamKey: "image_url", From: "images[0]"},
						},
						MediaRequirements: MediaRequirements{ImagesMin: 1, ImagesMax: 1},
					},
					{
						Label:           "多图生视频",
						Value:           "multi_image",
						Enabled:         true,
						Sort:            3,
						RequireRefModel: false,
						UpstreamSets: []UpstreamSet{
							{UpstreamKey: "reference_image_urls", From: "images"},
							{UpstreamKey: "video_config.reference_mode", Value: "auto"},
						},
						MediaRequirements: MediaRequirements{ImagesMin: 2, ImagesMax: 9},
					},
					{
						Label:           "首帧",
						Value:           "start_frame",
						Enabled:         true,
						Sort:            4,
						RequireRefModel: false,
						UpstreamSets: []UpstreamSet{
							{UpstreamKey: "image_url", From: "images[0]"},
							{UpstreamKey: "video_config.reference_mode", Value: "start_frame"},
						},
						MediaRequirements: MediaRequirements{ImagesMin: 1, ImagesMax: 1},
					},
					{
						Label:           "首尾帧",
						Value:           "start_end",
						Enabled:         true,
						Sort:            5,
						RequireRefModel: false,
						UpstreamSets: []UpstreamSet{
							{UpstreamKey: "reference_image_urls", From: "images"},
							{UpstreamKey: "video_config.reference_mode", Value: "start_end"},
						},
						MediaRequirements: MediaRequirements{ImagesMin: 2, ImagesMax: 2},
					},
				},
				ExtraOptions: []OptionItem{
					{
						Label:       "参考模式-自动",
						Value:       "auto",
						UpstreamKey: "video_config.reference_mode",
						Enabled:     true,
						Sort:        1,
					},
				},
			},
			{
				ID:            "dreamina_overseas",
				Label:         "海外满血",
				ModelPrefixes: []string{"dreamina-seedance-2-0-"},
				Durations: []OptionItem{
					{Label: "4 秒", Value: "4", UpstreamKey: "duration", Enabled: true, Sort: 1},
					{Label: "5 秒", Value: "5", UpstreamKey: "duration", Enabled: true, Sort: 2},
				},
				AspectRatios: defaultAspectRatios(),
				GenerationTypes: []GenerationType{
					{
						Label:             "文生视频",
						Value:             "text2video",
						Enabled:           true,
						Sort:              1,
						RequireRefModel:   false,
						UpstreamSets:      []UpstreamSet{},
						MediaRequirements: MediaRequirements{ImagesMin: 0, ImagesMax: 0},
					},
					{
						Label:           "图生 / 参考生",
						Value:           "image2video",
						Enabled:         true,
						Sort:            2,
						RequireRefModel: true,
						UpstreamSets: []UpstreamSet{
							{UpstreamKey: "image", From: "images[0]"},
						},
						MediaRequirements: MediaRequirements{ImagesMin: 1, ImagesMax: 9},
					},
					{
						Label:           "首尾帧",
						Value:           "start_end",
						Enabled:         true,
						Sort:            3,
						RequireRefModel: true,
						UpstreamSets: []UpstreamSet{
							{UpstreamKey: "first_frame", From: "images[0]"},
							{UpstreamKey: "last_frame", From: "images[1]"},
						},
						MediaRequirements: MediaRequirements{ImagesMin: 2, ImagesMax: 2},
					},
				},
				ExtraOptions: []OptionItem{
					{
						Label:       "生成声音-开",
						Value:       "true",
						UpstreamKey: "generate_audio",
						Enabled:     true,
						Sort:        1,
					},
					{
						Label:       "生成声音-关",
						Value:       "false",
						UpstreamKey: "generate_audio",
						Enabled:     true,
						Sort:        2,
					},
				},
			},
		},
		Storage: StorageSetting{
			Enabled:               true,
			Driver:                "local",
			LocalDir:              "data/silkroad-videos",
			RetentionDays:         7,
			MaxRetry:              5,
			IngestNodeName:        "",
			PublicDownloadBaseURL: "",
		},
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
