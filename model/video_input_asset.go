package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	VideoInputAssetStatusPresigned = "presigned"
	VideoInputAssetStatusReady     = "ready"
	VideoInputAssetStatusFailed    = "failed"
	VideoInputAssetStatusExpired   = "expired"

	VideoInputAssetKindImage = "image"
	VideoInputAssetKindAudio = "audio"
	VideoInputAssetKindVideo = "video"
)

// VideoInputAsset tracks a user-uploaded reference object staged in R2 for
// video generation. Object keys are server-assigned; clients never choose them.
type VideoInputAsset struct {
	Id          int    `json:"id" gorm:"primaryKey"`
	AssetId     string `json:"asset_id" gorm:"type:varchar(64);uniqueIndex;not null"`
	UserId      int    `json:"user_id" gorm:"index;not null"`
	ObjectKey   string `json:"object_key" gorm:"type:varchar(512);not null"`
	Kind        string `json:"kind" gorm:"type:varchar(16);not null"`
	ContentType string `json:"content_type" gorm:"type:varchar(128);not null"`
	Size        int64  `json:"size" gorm:"not null"`
	Status      string `json:"status" gorm:"type:varchar(16);index;not null"`
	ExpiresAt   int64  `json:"expires_at" gorm:"bigint;not null"`
	CreatedAt   int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt   int64  `json:"updated_at" gorm:"bigint"`
}

func (VideoInputAsset) TableName() string {
	return "video_input_assets"
}

func NewVideoInputAssetID() string {
	return "vi_" + common.GetUUID()
}

func CreateVideoInputAsset(asset *VideoInputAsset) error {
	if asset == nil {
		return errors.New("video input asset is required")
	}
	now := time.Now().Unix()
	if asset.CreatedAt == 0 {
		asset.CreatedAt = now
	}
	asset.UpdatedAt = now
	return DB.Create(asset).Error
}

func GetVideoInputAssetByAssetID(userID int, assetID string) (*VideoInputAsset, error) {
	if userID <= 0 || assetID == "" {
		return nil, errors.New("user id and asset id are required")
	}
	var asset VideoInputAsset
	err := DB.Where("user_id = ? AND asset_id = ?", userID, assetID).First(&asset).Error
	if err != nil {
		return nil, err
	}
	return &asset, nil
}

func UpdateVideoInputAsset(asset *VideoInputAsset) error {
	if asset == nil || asset.Id == 0 {
		return errors.New("video input asset is required")
	}
	asset.UpdatedAt = time.Now().Unix()
	return DB.Save(asset).Error
}

func DeleteVideoInputAsset(userID int, assetID string) error {
	if userID <= 0 || assetID == "" {
		return errors.New("user id and asset id are required")
	}
	result := DB.Where("user_id = ? AND asset_id = ?", userID, assetID).Delete(&VideoInputAsset{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("video input asset not found")
	}
	return nil
}

func CountVideoInputAssetsByStatus(userID int, status string) (int64, error) {
	if userID <= 0 || status == "" {
		return 0, errors.New("user id and status are required")
	}
	var count int64
	err := DB.Model(&VideoInputAsset{}).
		Where("user_id = ? AND status = ?", userID, status).
		Count(&count).Error
	return count, err
}
