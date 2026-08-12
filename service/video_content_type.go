package service

import (
	"fmt"
	"mime"
	"strings"
)

func NormalizeStoredVideoContentType(raw string) (string, error) {
	contentType := strings.TrimSpace(raw)
	if contentType == "" || strings.EqualFold(contentType, "application/octet-stream") {
		return "video/mp4", nil
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", fmt.Errorf("invalid stored video content type")
	}
	mediaType = strings.ToLower(mediaType)
	if mediaType == "application/mp4" {
		return "video/mp4", nil
	}
	if !strings.HasPrefix(mediaType, "video/") {
		return "", fmt.Errorf("stored content is not a video")
	}
	return mediaType, nil
}
