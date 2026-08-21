package silkroad

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
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
	assert.Equal(t, constant.TaskActionTextGenerate, info.Action)
	require.NotNil(t, info.RequestSnapshot)
	assert.Equal(t, "text2video", info.RequestSnapshot.GenerationType)
	assert.Empty(t, info.RequestSnapshot.Media)

	reader, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	got, err := io.ReadAll(reader)
	require.NoError(t, err)

	const want = `{"duration":10,"metadata":{"ratio":"16:9"},"model":"seedance-2.0-720","prompt":"a cat walking on the moon","resolution":"720p"}`
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
		"seconds":"5",
		"aspect_ratio":"9:16",
		"images":["data:image/jpeg;base64,abc"]
	}`, "dreamina-seedance-2-0-720-ref", "dreamina-seedance-2-0-720-ref")

	require.Nil(t, a.ValidateRequestAndSetAction(c, info))

	reader, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	got, err := io.ReadAll(reader)
	require.NoError(t, err)

	const want = `{"duration":5,"images":["data:image/jpeg;base64,abc"],"metadata":{"ratio":"9:16"},"model":"dreamina-seedance-2-0-720-ref","prompt":"animate the still photo","resolution":"720p"}`
	assert.JSONEq(t, want, string(got))
	assert.NotContains(t, string(got), "generation_type")
	assert.NotContains(t, string(got), `"image"`)
}

func TestE2EBuildUsesMappedUpstreamModelForProfileAndReferenceValidation(t *testing.T) {
	a := &TaskAdaptor{}
	c, info := newE2EContext(t, `{
		"model":"public-seedance",
		"prompt":"animate the still photo",
		"generation_type":"image2video",
		"seconds":"5",
		"aspect_ratio":"9:16",
		"images":["data:image/jpeg;base64,abc"]
	}`, "public-seedance", "dreamina-seedance-2-0-720-ref")

	require.Nil(t, a.ValidateRequestAndSetAction(c, info))

	reader, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	got, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Contains(t, string(got), `"model":"dreamina-seedance-2-0-720-ref"`)
	assert.NotContains(t, string(got), `"model":"public-seedance"`)
}

func TestE2EBuildOpenAIVideosJSONNormalizesIntoFriendlyRequest(t *testing.T) {
	a := &TaskAdaptor{}
	c, info := newE2EContext(t, `{
		"model":"seedance-2.0-720",
		"prompt":"a lighthouse in a storm",
		"size":"1280x720"
	}`, "seedance-2.0-720", "seedance-2.0-720")
	c.Request.URL.Path = "/v1/videos"

	require.Nil(t, a.ValidateRequestAndSetAction(c, info))
	reader, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	got, err := io.ReadAll(reader)
	require.NoError(t, err)

	assert.JSONEq(t, `{
		"duration":5,
		"metadata":{"ratio":"16:9"},
		"model":"seedance-2.0-720",
		"prompt":"a lighthouse in a storm",
		"resolution":"720p"
	}`, string(got))
}

func TestE2EBuildDreaminaMultiImageRejectsAudio(t *testing.T) {
	a := &TaskAdaptor{}
	c, info := newE2EContext(t, `{
		"model":"dreamina-seedance-2-0-1080p-ref",
		"prompt":"一只橘猫在窗台上伸懒腰",
		"generation_type":"multi_image",
		"seconds":"5",
		"aspect_ratio":"16:9",
		"images":["data:image/jpeg;base64,aaa","data:image/jpeg;base64,bbb"],
		"audio_url":"data:audio/mpeg;base64,ccc"
	}`, "dreamina-seedance-2-0-1080p-ref", "dreamina-seedance-2-0-1080p-ref")

	taskErr := a.ValidateRequestAndSetAction(c, info)
	require.NotNil(t, taskErr)
	assert.Contains(t, strings.ToLower(taskErr.Message), "audio")
}

func TestE2EBuildDreaminaReferenceAudioGolden(t *testing.T) {
	a := &TaskAdaptor{}
	c, info := newE2EContext(t, `{
		"model":"dreamina-seedance-2-0-1080p-ref",
		"prompt":"一只橘猫在窗台上伸懒腰",
		"generation_type":"reference_audio",
		"seconds":"5",
		"aspect_ratio":"16:9",
		"images":["data:image/jpeg;base64,aaa"],
		"audio_url":"data:audio/mpeg;base64,ccc"
	}`, "dreamina-seedance-2-0-1080p-ref", "dreamina-seedance-2-0-1080p-ref")

	require.Nil(t, a.ValidateRequestAndSetAction(c, info))

	reader, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	got, err := io.ReadAll(reader)
	require.NoError(t, err)

	const want = `{
		"duration":5,
		"images":["data:image/jpeg;base64,aaa"],
		"metadata":{
			"audios":["data:audio/mpeg;base64,ccc"],
			"ratio":"16:9"
		},
		"model":"dreamina-seedance-2-0-1080p-ref",
		"prompt":"一只橘猫在窗台上伸懒腰",
		"resolution":"1080p"
	}`
	assert.JSONEq(t, want, string(got))
}

func TestE2EBuildDreaminaReferenceVideosGolden(t *testing.T) {
	a := &TaskAdaptor{}
	c, info := newE2EContext(t, `{
		"model":"dreamina-seedance-2-0-1080p-ref",
		"prompt":"跟随 @Video1 的运镜",
		"generation_type":"reference_videos",
		"seconds":"5",
		"aspect_ratio":"16:9",
		"images":["data:image/jpeg;base64,aaa"],
		"reference_videos":["data:video/mp4;base64,vid"]
	}`, "dreamina-seedance-2-0-1080p-ref", "dreamina-seedance-2-0-1080p-ref")

	require.Nil(t, a.ValidateRequestAndSetAction(c, info))
	assert.Equal(t, constant.TaskActionReferenceGenerate, info.Action)
	require.NotNil(t, info.RequestSnapshot)
	assert.Equal(t, "reference_videos", info.RequestSnapshot.GenerationType)
	assert.Equal(t, []relaycommon.TaskMediaSnapshot{
		{Type: "image", Role: "reference"},
		{Type: "video", Role: "reference"},
	}, info.RequestSnapshot.Media)

	reader, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	got, err := io.ReadAll(reader)
	require.NoError(t, err)

	const want = `{
		"duration":5,
		"images":["data:image/jpeg;base64,aaa"],
		"metadata":{
			"ratio":"16:9",
			"reference_videos":["data:video/mp4;base64,vid"]
		},
		"model":"dreamina-seedance-2-0-1080p-ref",
		"prompt":"跟随 @Video1 的运镜",
		"resolution":"1080p"
	}`
	assert.JSONEq(t, want, string(got))
}

func TestE2EBuildDreaminaStartEndGolden(t *testing.T) {
	a := &TaskAdaptor{}
	c, info := newE2EContext(t, `{
		"model":"dreamina-seedance-2-0-1080p-ref",
		"prompt":"一只橘猫在窗台上伸懒腰",
		"generation_type":"start_end",
		"seconds":"5",
		"aspect_ratio":"16:9",
		"images":["data:image/jpeg;base64,first","data:image/jpeg;base64,last"]
	}`, "dreamina-seedance-2-0-1080p-ref", "dreamina-seedance-2-0-1080p-ref")

	require.Nil(t, a.ValidateRequestAndSetAction(c, info))

	reader, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	got, err := io.ReadAll(reader)
	require.NoError(t, err)

	const want = `{
		"duration":5,
		"metadata":{
			"first_frame":"data:image/jpeg;base64,first",
			"last_frame":"data:image/jpeg;base64,last",
			"ratio":"16:9"
		},
		"model":"dreamina-seedance-2-0-1080p-ref",
		"prompt":"一只橘猫在窗台上伸懒腰",
		"resolution":"1080p"
	}`
	assert.JSONEq(t, want, string(got))
	assert.NotContains(t, string(got), `"images"`)
}
