package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsVideoTaskRequestPathIncludesAllTaskVideoRoutes(t *testing.T) {
	for _, path := range []string{
		"/v1/video/generations",
		"/v1/videos",
		"/v1/videos/task-id/remix",
		"/kling/v1/videos/text2video",
		"/kling/v1/videos/image2video",
		"/jimeng",
		"/jimeng/",
	} {
		assert.True(t, IsVideoTaskRequestPath(path), path)
	}

	for _, path := range []string{
		"/v1/chat/completions",
		"/v1/images/generations",
		"/suno/submit/music",
	} {
		assert.False(t, IsVideoTaskRequestPath(path), path)
	}
}
