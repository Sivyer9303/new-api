package video_setting

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/setting/config"
)

const (
	DriverLocal = "local"
	DriverR2    = "r2"
)

const (
	// RetentionDays is the historical fixed retention for locally stored videos.
	// Kept as the local driver default; use StorageSetting.RetentionDays() for the
	// value that actually applies to the configured driver.
	RetentionDays = 7

	DefaultLocalRetentionDays = RetentionDays
	DefaultR2RetentionDays    = 3
	MinRetentionDays          = 1
	MaxRetentionDays          = 30
)

const (
	DefaultR2Region                  = "auto"
	DefaultR2ResultPrefix            = "videos/"
	DefaultR2InputPrefix             = "video-inputs/"
	DefaultR2ResultPresignTTLSeconds = 900
	DefaultR2InputPresignTTLSeconds  = 21600
	DefaultR2InputTTLHours           = 24

	MinPresignTTLSeconds = 60
	MaxPresignTTLSeconds = 7 * 24 * 3600
	MinInputTTLHours     = 1
	MaxInputTTLHours     = 30 * 24
)

// R2 free-tier guard rails. Deliberately not configurable: the operator
// requirement is to never leave the free storage allowance.
const (
	R2FreeTierBytes  int64   = 10 << 30
	R2SoftLimitRatio float64 = 0.9
)

// R2SoftLimitBytes is the usage level at which new uploads stop.
func R2SoftLimitBytes() int64 {
	return int64(float64(R2FreeTierBytes) * R2SoftLimitRatio)
}

// R2StorageSetting configures the Cloudflare R2 video storage driver.
// AccessKeyID/SecretAccessKey sign S3-compatible object operations, while
// APIToken is only used to read bucket usage for the free-tier guard.
type R2StorageSetting struct {
	AccountID               string `json:"account_id"`
	AccessKeyID             string `json:"access_key_id"`
	SecretAccessKey         string `json:"secret_access_key"`
	APIToken                string `json:"api_token"`
	Bucket                  string `json:"bucket"`
	Endpoint                string `json:"endpoint"`
	Region                  string `json:"region"`
	ResultPrefix            string `json:"result_prefix"`
	InputPrefix             string `json:"input_prefix"`
	RetentionDays           int    `json:"retention_days"`
	ResultPresignTTLSeconds int    `json:"result_presign_ttl_seconds"`
	InputPresignTTLSeconds  int    `json:"input_presign_ttl_seconds"`
	InputTTLHours           int    `json:"input_ttl_hours"`
}

// StorageSetting configures video result storage. The local driver keeps its
// historical flat fields so existing option rows stay readable; R2 settings are
// nested because they are only meaningful for that driver.
type StorageSetting struct {
	Driver                string           `json:"driver"`
	MaxRetry              int              `json:"max_retry"`
	LocalDir              string           `json:"local_dir"`
	IngestNodeName        string           `json:"ingest_node_name"`
	PublicDownloadBaseURL string           `json:"public_download_base_url"`
	LocalRetentionDays    int              `json:"local_retention_days"`
	R2                    R2StorageSetting `json:"r2"`
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
		Storage:         defaultStorageSetting(),
	}
}

func defaultStorageSetting() StorageSetting {
	return StorageSetting{
		Driver:             DriverLocal,
		MaxRetry:           5,
		LocalDir:           "data/videos",
		LocalRetentionDays: DefaultLocalRetentionDays,
		R2:                 defaultR2StorageSetting(),
	}
}

func defaultR2StorageSetting() R2StorageSetting {
	return R2StorageSetting{
		Region:                  DefaultR2Region,
		ResultPrefix:            DefaultR2ResultPrefix,
		InputPrefix:             DefaultR2InputPrefix,
		RetentionDays:           DefaultR2RetentionDays,
		ResultPresignTTLSeconds: DefaultR2ResultPresignTTLSeconds,
		InputPresignTTLSeconds:  DefaultR2InputPresignTTLSeconds,
		InputTTLHours:           DefaultR2InputTTLHours,
	}
}

// IsR2 reports whether stored videos live in Cloudflare R2.
func (s StorageSetting) IsR2() bool {
	return strings.TrimSpace(strings.ToLower(s.Driver)) == DriverR2
}

// RetentionDays returns the retention period of the configured driver.
func (s StorageSetting) RetentionDays() int {
	days := s.LocalRetentionDays
	if s.IsR2() {
		days = s.R2.RetentionDays
	}
	if days < MinRetentionDays {
		if s.IsR2() {
			return DefaultR2RetentionDays
		}
		return DefaultLocalRetentionDays
	}
	return days
}

// ResolveEndpoint returns the S3 endpoint, derived from the account when unset.
func (s R2StorageSetting) ResolveEndpoint() string {
	endpoint := strings.TrimRight(strings.TrimSpace(s.Endpoint), "/")
	if endpoint != "" {
		return endpoint
	}
	account := strings.TrimSpace(s.AccountID)
	if account == "" {
		return ""
	}
	return fmt.Sprintf("https://%s.r2.cloudflarestorage.com", account)
}

func (s R2StorageSetting) ResolveRegion() string {
	region := strings.TrimSpace(s.Region)
	if region == "" {
		return DefaultR2Region
	}
	return region
}

func (s R2StorageSetting) ResultPresignTTL() time.Duration {
	return presignTTL(s.ResultPresignTTLSeconds, DefaultR2ResultPresignTTLSeconds)
}

func (s R2StorageSetting) InputPresignTTL() time.Duration {
	return presignTTL(s.InputPresignTTLSeconds, DefaultR2InputPresignTTLSeconds)
}

func (s R2StorageSetting) InputTTL() time.Duration {
	hours := s.InputTTLHours
	if hours < MinInputTTLHours || hours > MaxInputTTLHours {
		hours = DefaultR2InputTTLHours
	}
	return time.Duration(hours) * time.Hour
}

func presignTTL(seconds int, fallback int) time.Duration {
	if seconds < MinPresignTTLSeconds || seconds > MaxPresignTTLSeconds {
		seconds = fallback
	}
	return time.Duration(seconds) * time.Second
}

// ObjectKey joins a prefix with a name using forward slashes (R2 keys are flat).
func ObjectKey(prefix string, name string) string {
	cleaned := strings.Trim(strings.TrimSpace(prefix), "/")
	name = strings.TrimLeft(strings.TrimSpace(name), "/")
	if cleaned == "" {
		return name
	}
	return cleaned + "/" + name
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
	NormalizeStorageSetting(&effective.Storage)
	return effective
}

// NormalizeStorageSetting fills unset fields with defaults so callers never have
// to special-case zero values coming from older option rows.
func NormalizeStorageSetting(s *StorageSetting) {
	if s == nil {
		return
	}
	s.Driver = strings.TrimSpace(strings.ToLower(s.Driver))
	if s.Driver == "" {
		s.Driver = DriverLocal
	}
	if s.MaxRetry < 1 {
		s.MaxRetry = 5
	}
	if strings.TrimSpace(s.LocalDir) == "" {
		s.LocalDir = "data/videos"
	}
	if s.LocalRetentionDays < MinRetentionDays {
		s.LocalRetentionDays = DefaultLocalRetentionDays
	}
	if strings.TrimSpace(s.R2.Region) == "" {
		s.R2.Region = DefaultR2Region
	}
	if strings.TrimSpace(s.R2.ResultPrefix) == "" {
		s.R2.ResultPrefix = DefaultR2ResultPrefix
	}
	if strings.TrimSpace(s.R2.InputPrefix) == "" {
		s.R2.InputPrefix = DefaultR2InputPrefix
	}
	if s.R2.RetentionDays < MinRetentionDays {
		s.R2.RetentionDays = DefaultR2RetentionDays
	}
	if s.R2.ResultPresignTTLSeconds < MinPresignTTLSeconds {
		s.R2.ResultPresignTTLSeconds = DefaultR2ResultPresignTTLSeconds
	}
	if s.R2.InputPresignTTLSeconds < MinPresignTTLSeconds {
		s.R2.InputPresignTTLSeconds = DefaultR2InputPresignTTLSeconds
	}
	if s.R2.InputTTLHours < MinInputTTLHours {
		s.R2.InputTTLHours = DefaultR2InputTTLHours
	}
}

func ValidateVideoSetting(s *VideoSetting) error {
	if s == nil {
		return errors.New("video setting is nil")
	}
	NormalizeStorageSetting(&s.Storage)
	switch s.Storage.Driver {
	case DriverLocal:
		if err := validateLocalStorage(s.Storage, s.Enabled); err != nil {
			return err
		}
	case DriverR2:
		if err := validateR2Storage(s.Storage.R2); err != nil {
			return err
		}
		if s.Enabled && strings.TrimSpace(s.Storage.PublicDownloadBaseURL) == "" {
			return errors.New("storage.public_download_base_url is required when video generation is enabled")
		}
	default:
		return fmt.Errorf("storage.driver must be %q or %q, got %q",
			DriverLocal, DriverR2, s.Storage.Driver)
	}
	if s.Storage.MaxRetry < 1 {
		return errors.New("storage.max_retry must be >= 1")
	}
	s.VideoToolGroups = NormalizeVideoToolGroups(s.VideoToolGroups)
	return nil
}

func validateLocalStorage(storage StorageSetting, videoEnabled bool) error {
	if strings.TrimSpace(storage.LocalDir) == "" {
		return errors.New("storage.local_dir is required")
	}
	if err := validateRetentionDays("storage.local_retention_days", storage.LocalRetentionDays); err != nil {
		return err
	}
	if !videoEnabled {
		return nil
	}
	if strings.TrimSpace(storage.IngestNodeName) == "" {
		return errors.New("storage.ingest_node_name is required when video generation is enabled")
	}
	if strings.TrimSpace(storage.PublicDownloadBaseURL) == "" {
		return errors.New("storage.public_download_base_url is required when video generation is enabled")
	}
	return nil
}

func validateR2Storage(r2 R2StorageSetting) error {
	required := []struct {
		field string
		value string
	}{
		{"storage.r2.account_id", r2.AccountID},
		{"storage.r2.access_key_id", r2.AccessKeyID},
		{"storage.r2.secret_access_key", r2.SecretAccessKey},
		{"storage.r2.api_token", r2.APIToken},
		{"storage.r2.bucket", r2.Bucket},
	}
	for _, item := range required {
		if strings.TrimSpace(item.value) == "" {
			return fmt.Errorf("%s is required when storage.driver is %q", item.field, DriverR2)
		}
	}
	if r2.ResolveEndpoint() == "" {
		return errors.New("storage.r2.endpoint is required")
	}
	if err := validateRetentionDays("storage.r2.retention_days", r2.RetentionDays); err != nil {
		return err
	}
	if strings.Trim(strings.TrimSpace(r2.ResultPrefix), "/") ==
		strings.Trim(strings.TrimSpace(r2.InputPrefix), "/") {
		return errors.New("storage.r2.result_prefix and storage.r2.input_prefix must differ")
	}
	for _, item := range []struct {
		field string
		value int
	}{
		{"storage.r2.result_presign_ttl_seconds", r2.ResultPresignTTLSeconds},
		{"storage.r2.input_presign_ttl_seconds", r2.InputPresignTTLSeconds},
	} {
		if item.value < MinPresignTTLSeconds || item.value > MaxPresignTTLSeconds {
			return fmt.Errorf("%s must be between %d and %d",
				item.field, MinPresignTTLSeconds, MaxPresignTTLSeconds)
		}
	}
	if r2.InputTTLHours < MinInputTTLHours || r2.InputTTLHours > MaxInputTTLHours {
		return fmt.Errorf("storage.r2.input_ttl_hours must be between %d and %d",
			MinInputTTLHours, MaxInputTTLHours)
	}
	return nil
}

func validateRetentionDays(field string, days int) error {
	if days < MinRetentionDays || days > MaxRetentionDays {
		return fmt.Errorf("%s must be between %d and %d", field, MinRetentionDays, MaxRetentionDays)
	}
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
