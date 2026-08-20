package controller

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

type videoInputPresignRequest struct {
	Kind        string `json:"kind"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

// PresignVideoInputAsset issues a short-lived R2 PUT URL for reference media.
func PresignVideoInputAsset(c *gin.Context) {
	userID := c.GetInt("id")
	var req videoInputPresignRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request body",
		})
		return
	}
	result, err := service.CreateVideoInputAssetPresign(
		c.Request.Context(),
		userID,
		req.Kind,
		req.ContentType,
		req.Size,
	)
	if err != nil {
		respondVideoInputUploadError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// CompleteVideoInputAsset verifies the uploaded object and returns a GET URL.
func CompleteVideoInputAsset(c *gin.Context) {
	userID := c.GetInt("id")
	assetID := strings.TrimSpace(c.Param("id"))
	if assetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "asset id is required",
		})
		return
	}
	result, err := service.CompleteVideoInputAsset(c.Request.Context(), userID, assetID)
	if err != nil {
		respondVideoInputUploadError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// DeleteVideoInputAsset abandons an upload and deletes the R2 object when possible.
func DeleteVideoInputAsset(c *gin.Context) {
	userID := c.GetInt("id")
	assetID := strings.TrimSpace(c.Param("id"))
	if assetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "asset id is required",
		})
		return
	}
	if err := service.DeleteVideoInputAssetUpload(c.Request.Context(), userID, assetID); err != nil {
		respondVideoInputUploadError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func respondVideoInputUploadError(c *gin.Context, err error) {
	status := service.VideoInputUploadHTTPStatus(err)
	c.JSON(status, gin.H{
		"success": false,
		"message": err.Error(),
	})
}
