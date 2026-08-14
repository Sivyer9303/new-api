package video_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUploadLimitsMaxBytesForContentType(t *testing.T) {
	limits := UploadLimitsSetting{
		MaxImageMB: 2,
		MaxAudioMB: 3,
		MaxVideoMB: 4,
	}
	assert.Equal(t, int64(2)<<20, limits.MaxBytesForContentType("image/png"))
	assert.Equal(t, int64(3)<<20, limits.MaxBytesForContentType("audio/mpeg"))
	assert.Equal(t, int64(4)<<20, limits.MaxBytesForContentType("video/mp4"))
}

func TestValidateUploadLimitsSettingBounds(t *testing.T) {
	require.NoError(t, ValidateUploadLimitsSetting(&UploadLimitsSetting{
		MaxImageMB: 10,
		MaxAudioMB: 24,
		MaxVideoMB: 50,
	}))
	require.Error(t, ValidateUploadLimitsSetting(&UploadLimitsSetting{
		MaxImageMB: 0,
		MaxAudioMB: 24,
		MaxVideoMB: 50,
	}))
}
