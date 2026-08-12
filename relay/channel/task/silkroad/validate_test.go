package silkroad

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestContext(t *testing.T, body string) (*gin.Context, *relaycommon.RelayInfo) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest(http.MethodPost, "/v1/video/generations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	info := &relaycommon.RelayInfo{
		OriginModelName: "seedance-2.0-720",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "seedance-2.0-720",
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
	}
	return c, info
}

func TestValidateRequiresGenerationTypeAndAspectAndDuration(t *testing.T) {
	a := &TaskAdaptor{}
	c, info := newTestContext(t, `{"model":"seedance-2.0-720","prompt":"hi","seconds":"10","aspect_ratio":"16:9"}`)

	taskErr := a.ValidateRequestAndSetAction(c, info)

	require.NotNil(t, taskErr)
	assert.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
	assert.Contains(t, strings.ToLower(taskErr.Message), "generation_type")
}

func TestValidateRejectsDurationNotInConfig(t *testing.T) {
	a := &TaskAdaptor{}
	c, info := newTestContext(t, `{
		"model":"seedance-2.0-720",
		"prompt":"hi",
		"generation_type":"text2video",
		"seconds":"12",
		"aspect_ratio":"16:9"
	}`)

	taskErr := a.ValidateRequestAndSetAction(c, info)

	require.NotNil(t, taskErr)
	assert.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
}

func TestValidateFriendlyRequestRejectsUnknownTopLevelKey(t *testing.T) {
	profile, ok := silkroad_setting.MatchProfile("seedance-2.0-720")
	require.True(t, ok)

	raw := map[string]any{
		"model":           "seedance-2.0-720",
		"prompt":          "hi",
		"generation_type": "text2video",
		"seconds":         "10",
		"aspect_ratio":    "16:9",
		"video_config":    map[string]any{"reference_mode": "auto"},
	}
	err := validateFriendlyRequest(&FriendlyRequest{}, profile, raw)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "unknown")
}

func TestValidateFriendlyRequestRejectsMissingRefModel(t *testing.T) {
	profile, ok := silkroad_setting.MatchProfile("dreamina-seedance-2-0-720")
	require.True(t, ok)
	mode, ok := silkroad_setting.FindGenerationMode("image2video")
	require.True(t, ok)
	require.True(t, mode.RequireRefModel)

	req := FriendlyRequest{
		Model:          "dreamina-seedance-2-0-720",
		Prompt:         "hi",
		GenerationType: "image2video",
		DurationValue:  "5",
		AspectRatio:    "16:9",
		Images:         []string{"https://example.com/a.png"},
	}
	err := validateFriendlyRequest(&req, profile, map[string]any{
		"model": req.Model, "prompt": req.Prompt, "generation_type": req.GenerationType,
		"duration": 5, "aspect_ratio": req.AspectRatio, "images": req.Images,
	})
	require.NoError(t, err)

	err = checkRequireRefModel(mode, "dreamina-seedance-2-0-720")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "-ref")
}
