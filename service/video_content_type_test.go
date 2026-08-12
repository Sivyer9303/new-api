package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeStoredVideoContentTypeRejectsActiveContent(t *testing.T) {
	for _, contentType := range []string{
		"text/html",
		"image/svg+xml",
		"application/javascript",
	} {
		_, err := NormalizeStoredVideoContentType(contentType)
		assert.Error(t, err, contentType)
	}

	normalized, err := NormalizeStoredVideoContentType("video/webm; charset=binary")
	require.NoError(t, err)
	assert.Equal(t, "video/webm", normalized)
}
