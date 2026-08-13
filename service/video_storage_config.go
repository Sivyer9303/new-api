package service

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/video_setting"
)

func init() {
	// Channel save must reject R2 input media staging before R2 is configured,
	// but model cannot import service, so the check is injected here.
	model.RegisterVideoR2StorageValidator(ValidateVideoR2StorageConfigured)
}

// videoStorageSetting resolves the storage configuration currently in effect.
// It is a variable so storage-driver behavior can be exercised without mutating
// the global option registry.
var videoStorageSetting = func() video_setting.StorageSetting {
	return setting.GetEffectiveVideoSetting().Storage
}

var videoGenerationEnabled = func() bool {
	return setting.GetEffectiveVideoSetting().Enabled
}

func ValidateVideoGenerationReady() error {
	if !videoGenerationEnabled() {
		return errors.New("video generation is not enabled")
	}
	return ValidateVideoStorageReady()
}

func ValidateVideoStorageReady() error {
	effective := setting.GetEffectiveVideoSetting()
	if !effective.StorageEnabled {
		return errors.New("video result storage is not enabled")
	}
	storage := effective.Storage
	if storage.MaxRetry < 1 {
		return errors.New("video result storage retry limit is invalid")
	}
	if strings.TrimSpace(storage.PublicDownloadBaseURL) == "" {
		return errors.New("video result storage public download base is not configured")
	}
	switch storage.Driver {
	case video_setting.DriverLocal:
		if strings.TrimSpace(storage.LocalDir) == "" {
			return errors.New("video result storage local directory is not configured")
		}
		if strings.TrimSpace(storage.IngestNodeName) == "" {
			return errors.New("video result storage ingest node is not configured")
		}
		return nil
	case video_setting.DriverR2:
		return ValidateVideoR2StorageConfigured()
	default:
		return errors.New("video result storage driver is not supported")
	}
}

// ValidateVideoR2StorageConfigured reports whether R2 is the active driver and
// fully configured. Channels that stage input media on R2 require this.
func ValidateVideoR2StorageConfigured() error {
	storage := videoStorageSetting()
	if !storage.IsR2() {
		return errors.New("video result storage driver is not r2")
	}
	for _, item := range []struct {
		name  string
		value string
	}{
		{"account id", storage.R2.AccountID},
		{"access key id", storage.R2.AccessKeyID},
		{"secret access key", storage.R2.SecretAccessKey},
		{"api token", storage.R2.APIToken},
		{"bucket", storage.R2.Bucket},
	} {
		if strings.TrimSpace(item.value) == "" {
			return errors.New("video r2 storage " + item.name + " is not configured")
		}
	}
	if storage.R2.ResolveEndpoint() == "" {
		return errors.New("video r2 storage endpoint is not configured")
	}
	return nil
}
