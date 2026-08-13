package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/brioi_setting"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"
	"github.com/QuantumNous/new-api/setting/video_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetVideoToolConfigAggregatesSanitizedProviderCapabilities(t *testing.T) {
	brioi := brioi_setting.GetBrioiSetting()
	silkRoad := silkroad_setting.GetSilkRoadSetting()
	video := video_setting.GetVideoSetting()
	previousBrioi := *brioi
	previousSilkRoad := *silkRoad
	previousVideo := *video
	t.Cleanup(func() {
		*brioi = previousBrioi
		*silkRoad = previousSilkRoad
		*video = previousVideo
	})

	*brioi = brioi_setting.DefaultBrioiSetting()
	brioi.VideoToolGroups = []string{" brioi "}
	silkRoad.VideoToolGroups = []string{" silkroad "}
	video.VideoToolGroups = []string{" silkroad "}
	video.Storage.R2.SecretAccessKey = "TOP-SECRET-R2-CREDENTIAL"
	video.Storage.R2.APIToken = "TOP-SECRET-R2-TOKEN"

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/video/tool-config", nil)
	GetVideoToolConfig(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool                  `json:"success"`
		Data    publicVideoToolConfig `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	assert.Equal(t, 2, response.Data.Version)
	assert.Equal(t, setting.VideoProviderSilkRoad, response.Data.ProviderByGroup["silkroad"])
	assert.Equal(t, setting.VideoProviderBrioi, response.Data.ProviderByGroup["brioi"])
	assert.Equal(t, []string{"silkroad"}, response.Data.Providers.SilkRoad.VideoToolGroups)
	assert.Equal(t, []string{"brioi"}, response.Data.Providers.Brioi.VideoToolGroups)
	assert.NotEmpty(t, response.Data.Providers.SilkRoad.Profiles)
	assert.NotEmpty(t, response.Data.Providers.Brioi.Profiles)

	payload := strings.ToLower(recorder.Body.String())
	assert.NotContains(t, payload, "top-secret-r2")
	assert.NotContains(t, payload, "secret_access_key")
	assert.NotContains(t, payload, "api_token")
	assert.NotContains(t, payload, "channel_key")
	assert.NotContains(t, payload, "result_url")
}
