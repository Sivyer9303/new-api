package model

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskPrivateData_SilkroadStorageFieldsRoundTrip(t *testing.T) {
	orig := TaskPrivateData{
		Key:               "sk-test",
		UpstreamTaskID:    "upstream-1",
		ResultURL:         "https://cdn.example/result.mp4",
		UpstreamResultURL: "https://upstream.example/raw.mp4",
		StorageStatus:     "ready",
		StoragePath:       "s3://bucket/path/video.mp4",
		StorageExpiresAt:  1735689600,
		StorageRetryCount: 2,
	}

	data, err := common.Marshal(orig)
	require.NoError(t, err)

	raw := string(data)
	for _, key := range []string{
		"upstream_result_url",
		"storage_status",
		"storage_path",
		"storage_expires_at",
		"storage_retry_count",
	} {
		assert.True(t, strings.Contains(raw, `"`+key+`"`), "missing json key %q in %s", key, raw)
	}

	var decoded TaskPrivateData
	require.NoError(t, common.Unmarshal(data, &decoded))
	assert.Equal(t, orig, decoded)
}

func TestTaskPrivateData_SilkroadStorageFieldsValueScan(t *testing.T) {
	orig := TaskPrivateData{
		UpstreamResultURL: "https://upstream.example/raw.mp4",
		StorageStatus:     "pending",
		StorageRetryCount: 0,
	}

	val, err := orig.Value()
	require.NoError(t, err)

	var scanned TaskPrivateData
	require.NoError(t, scanned.Scan(val))
	assert.Equal(t, orig, scanned)
}
