package newapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newE2EContext(t *testing.T, body, originModel, upstreamModel string) (*gin.Context, *relaycommon.RelayInfo) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest(http.MethodPost, "/v1/video/generations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	info := &relaycommon.RelayInfo{
		OriginModelName: originModel,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: upstreamModel,
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
	}
	return c, info
}

func TestE2EBuildSeedanceText2VideoGolden(t *testing.T) {
	a := &TaskAdaptor{}
	c, info := newE2EContext(t, `{
		"model":"seedance-2.0-720",
		"prompt":"a cat walking on the moon",
		"generation_type":"text2video",
		"seconds":"10",
		"aspect_ratio":"16:9"
	}`, "seedance-2.0-720", "seedance-2.0-720")

	require.Nil(t, a.ValidateRequestAndSetAction(c, info))

	reader, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	got, err := io.ReadAll(reader)
	require.NoError(t, err)

	const want = `{"aspect_ratio":"16:9","model":"seedance-2.0-720","prompt":"a cat walking on the moon","seconds":"10"}`
	assert.JSONEq(t, want, string(got))
	assert.NotContains(t, string(got), "generation_type")
	assert.NotContains(t, string(got), "images")
}

func TestE2EBuildDreaminaImage2VideoGolden(t *testing.T) {
	a := &TaskAdaptor{}
	c, info := newE2EContext(t, `{
		"model":"dreamina-seedance-2-0-720-ref",
		"prompt":"animate the still photo",
		"generation_type":"image2video",
		"duration":5,
		"aspect_ratio":"9:16",
		"images":["https://cdn.example/ref.png"],
		"generate_audio":true
	}`, "dreamina-seedance-2-0-720-ref", "dreamina-seedance-2-0-720-ref")

	require.Nil(t, a.ValidateRequestAndSetAction(c, info))

	reader, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	got, err := io.ReadAll(reader)
	require.NoError(t, err)

	const want = `{"aspect_ratio":"9:16","duration":5,"generate_audio":true,"image":"https://cdn.example/ref.png","model":"dreamina-seedance-2-0-720-ref","prompt":"animate the still photo"}`
	assert.JSONEq(t, want, string(got))
	assert.NotContains(t, string(got), "generation_type")
	assert.NotContains(t, string(got), `"images"`)
}
