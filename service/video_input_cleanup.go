package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/logger"
)

// maxVideoInputCleanupPages bounds one cleanup pass so a large bucket cannot
// monopolize the worker or the R2 Class A operation budget.
const maxVideoInputCleanupPages = 20

// RunVideoInputCleanupOnce deletes staged reference media past its TTL. Only the
// configured input prefix is touched, so stored results are never affected.
func RunVideoInputCleanupOnce(ctx context.Context) error {
	storage := videoStorageSetting()
	if !storage.IsR2() {
		return nil
	}
	if err := ValidateVideoR2StorageConfigured(); err != nil {
		return nil
	}
	prefix := strings.Trim(strings.TrimSpace(storage.R2.InputPrefix), "/")
	if prefix == "" {
		return nil
	}
	store, err := newR2HTTPObjectStore(storage.R2)
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-storage.R2.InputTTL())
	token := ""
	deleted := 0
	for page := 0; page < maxVideoInputCleanupPages; page++ {
		listed, err := store.ListObjects(ctx, prefix+"/", token)
		if err != nil {
			return err
		}
		for _, object := range listed.Objects {
			if !strings.HasPrefix(object.Key, prefix+"/") {
				continue
			}
			if object.LastModified.IsZero() || !object.LastModified.Before(cutoff) {
				continue
			}
			if err := store.DeleteObject(ctx, object.Key); err != nil {
				logger.LogWarn(ctx, fmt.Sprintf(
					"video input cleanup delete failed key=%s: %s", object.Key, err.Error(),
				))
				continue
			}
			deleted++
		}
		token = listed.NextToken
		if token == "" {
			break
		}
	}
	if deleted > 0 {
		logger.LogInfo(ctx, fmt.Sprintf(
			"video input cleanup removed %d staged objects older than %s",
			deleted, time.Duration(storage.R2.InputTTLHours)*time.Hour,
		))
	}
	return nil
}
