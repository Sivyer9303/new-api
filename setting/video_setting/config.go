package video_setting

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/setting/config"
)

const RetentionDays = 7

type StorageSetting struct {
	Driver                string `json:"driver"`
	LocalDir              string `json:"local_dir"`
	MaxRetry              int    `json:"max_retry"`
	IngestNodeName        string `json:"ingest_node_name"`
	PublicDownloadBaseURL string `json:"public_download_base_url"`
}

type VideoSetting struct {
	Enabled         bool           `json:"enabled"`
	VideoToolGroups []string       `json:"video_tool_groups"`
	Storage         StorageSetting `json:"storage"`
}

type ExplicitFields struct {
	Enabled         bool
	VideoToolGroups bool
	Storage         bool
}

type LegacySetting struct {
	ToolEnabled     bool
	VideoToolGroups []string
	StorageEnabled  bool
	Storage         StorageSetting
}

type EffectiveSetting struct {
	VideoSetting
	StorageEnabled bool
}

var videoSetting = defaultVideoSetting()

func init() {
	config.GlobalConfig.Register("video_setting", &videoSetting)
}

func GetVideoSetting() *VideoSetting {
	return &videoSetting
}

func defaultVideoSetting() VideoSetting {
	return VideoSetting{
		Enabled:         false,
		VideoToolGroups: []string{},
		Storage: StorageSetting{
			Driver:   "local",
			LocalDir: "data/videos",
			MaxRetry: 5,
		},
	}
}

func ResolveEffectiveSetting(
	configured VideoSetting,
	explicit ExplicitFields,
	legacy LegacySetting,
) EffectiveSetting {
	effective := EffectiveSetting{
		VideoSetting: VideoSetting{
			Enabled:         legacy.ToolEnabled,
			VideoToolGroups: NormalizeVideoToolGroups(legacy.VideoToolGroups),
			Storage:         legacy.Storage,
		},
		StorageEnabled: legacy.StorageEnabled,
	}
	if explicit.Enabled {
		effective.Enabled = configured.Enabled
	}
	if explicit.VideoToolGroups {
		effective.VideoToolGroups = NormalizeVideoToolGroups(configured.VideoToolGroups)
	}
	if explicit.Storage {
		effective.Storage = configured.Storage
		effective.StorageEnabled = true
	}
	return effective
}

func ValidateVideoSetting(s *VideoSetting) error {
	if s == nil {
		return errors.New("video setting is nil")
	}
	if s.Storage.Driver != "local" {
		return fmt.Errorf("storage.driver must be \"local\", got %q", s.Storage.Driver)
	}
	if strings.TrimSpace(s.Storage.LocalDir) == "" {
		return errors.New("storage.local_dir is required")
	}
	if s.Storage.MaxRetry < 1 {
		return errors.New("storage.max_retry must be >= 1")
	}
	if s.Enabled {
		if strings.TrimSpace(s.Storage.IngestNodeName) == "" {
			return errors.New("storage.ingest_node_name is required when video generation is enabled")
		}
		if strings.TrimSpace(s.Storage.PublicDownloadBaseURL) == "" {
			return errors.New("storage.public_download_base_url is required when video generation is enabled")
		}
	}
	s.VideoToolGroups = NormalizeVideoToolGroups(s.VideoToolGroups)
	return nil
}

func NormalizeVideoToolGroups(groups []string) []string {
	if len(groups) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(groups))
	out := make([]string, 0, len(groups))
	for _, group := range groups {
		name := strings.TrimSpace(group)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}
