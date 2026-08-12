package service

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/setting"
)

func ValidateVideoStorageReady() error {
	effective := setting.GetEffectiveVideoSetting()
	if !effective.StorageEnabled {
		return errors.New("video result storage is not enabled")
	}
	if effective.Storage.Driver != "local" {
		return errors.New("video result storage driver is not supported")
	}
	if strings.TrimSpace(effective.Storage.LocalDir) == "" {
		return errors.New("video result storage local directory is not configured")
	}
	if strings.TrimSpace(effective.Storage.IngestNodeName) == "" {
		return errors.New("video result storage ingest node is not configured")
	}
	if strings.TrimSpace(effective.Storage.PublicDownloadBaseURL) == "" {
		return errors.New("video result storage public download base is not configured")
	}
	if effective.Storage.MaxRetry < 1 {
		return errors.New("video result storage retry limit is invalid")
	}
	return nil
}
