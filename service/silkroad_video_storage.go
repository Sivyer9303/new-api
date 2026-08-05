package service

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"
)

// IsSilkRoadIngestNode reports whether this process is the configured SilkRoad
// video ingest node. Empty IngestNodeName returns false to avoid dual-writer races.
func IsSilkRoadIngestNode() bool {
	ingest := silkroad_setting.GetSilkRoadSetting().Storage.IngestNodeName
	if ingest == "" {
		return false
	}
	return common.NodeName == ingest
}

// SilkRoadVideoLocalPath returns the local filesystem path for a task video.
func SilkRoadVideoLocalPath(taskID string) string {
	return filepath.Join(silkroad_setting.GetSilkRoadSetting().Storage.LocalDir, taskID)
}

// WriteSilkRoadVideoFile writes video bytes for taskID under LocalDir.
func WriteSilkRoadVideoFile(taskID string, r io.Reader) (absPath string, size int64, err error) {
	path := SilkRoadVideoLocalPath(taskID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", 0, err
	}
	f, err := os.Create(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	n, err := io.Copy(f, r)
	if err != nil {
		return "", 0, err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", 0, err
	}
	return abs, n, nil
}

// OpenSilkRoadVideoFile opens the local video file for reading.
func OpenSilkRoadVideoFile(taskID string) (*os.File, error) {
	return os.Open(SilkRoadVideoLocalPath(taskID))
}

// DeleteSilkRoadVideoFile removes the local video file for taskID.
func DeleteSilkRoadVideoFile(taskID string) error {
	return os.Remove(SilkRoadVideoLocalPath(taskID))
}

// BuildSilkRoadPublicURL builds the public content URL for a stored video.
func BuildSilkRoadPublicURL(taskID string) string {
	base := strings.TrimRight(silkroad_setting.GetSilkRoadSetting().Storage.PublicDownloadBaseURL, "/")
	return base + "/v1/videos/" + taskID + "/content"
}
