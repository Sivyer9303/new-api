package brioi

import (
	"bytes"
	stdcontext "context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/brioi_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func brioiTestImageDataURL(t *testing.T) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	var payload bytes.Buffer
	require.NoError(t, png.Encode(&payload, img))
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(payload.Bytes())
}

func newBrioiContext(
	t *testing.T,
	path string,
	body string,
	originModel string,
	upstreamModel string,
) (*gin.Context, *relaycommon.RelayInfo) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = request
	info := &relaycommon.RelayInfo{
		OriginModelName: originModel,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:         42,
			ChannelBaseUrl:    "https://brioi.example/",
			ApiKey:            "secret-key",
			UpstreamModelName: upstreamModel,
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{PublicTaskID: "task_public"},
	}
	return context, info
}

func TestBuildRequestBodyGoldenModesAndModels(t *testing.T) {
	imageDataURL := brioiTestImageDataURL(t)
	videoDataURL := "data:video/mp4;base64," + base64.StdEncoding.EncodeToString([]byte("fake-mp4"))
	audioDataURL := "data:audio/mpeg;base64," + base64.StdEncoding.EncodeToString([]byte("fake-mp3"))
	movDataURL := "data:video/quicktime;base64," + base64.StdEncoding.EncodeToString([]byte("fake-mov"))
	wavDataURL := "data:audio/wav;base64," + base64.StdEncoding.EncodeToString([]byte("fake-wav"))
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "Seedance 2.0 Fast text",
			body: `{
				"model":"seedance-2-0-fast",
				"prompt":"a comet above the sea",
				"generation_type":"text2video",
				"duration":4,
				"resolution":"480p",
				"aspect_ratio":"21:9"
			}`,
			want: `{
				"model":"seedance-2-0-fast",
				"prompt":"a comet above the sea",
				"duration":4,
				"resolution":"480p",
				"aspect_ratio":"21:9"
			}`,
		},
		{
			name: "Seedance 2.0 ordinary multi-image",
			body: fmt.Sprintf(`{
				"model":"seedance-2-0",
				"prompt":"blend the references",
				"generation_type":"multi_image",
				"duration":15,
				"resolution":"4K",
				"aspect_ratio":"1:1",
				"images":[
					"%s",
					"%s"
				]
			}`, imageDataURL, imageDataURL),
			want: `{
				"model":"seedance-2-0",
				"prompt":"blend the references",
				"duration":15,
				"resolution":"4K",
				"aspect_ratio":"1:1",
				"ref":[
					{"url":"https://r2.example/0.png?signature=s0","type":"image"},
					{"url":"https://r2.example/1.png?signature=s1","type":"image"}
				]
			}`,
		},
		{
			name: "Seedance 2.5 strict first and last frames",
			body: fmt.Sprintf(`{
				"model":"seedance-2-5",
				"prompt":"move from night to dawn",
				"generation_type":"start_end",
				"seconds":"29",
				"resolution":"720p",
				"aspect_ratio":"9:16",
				"media":[
					{"type":"image","role":"last_frame","source":"%s"},
					{"type":"image","role":"first_frame","source":"%s"}
				]
			}`, imageDataURL, imageDataURL),
			want: `{
				"model":"seedance-2-5",
				"prompt":"move from night to dawn",
				"duration":29,
				"resolution":"720p",
				"aspect_ratio":"9:16",
				"ref":[
					{"url":"https://r2.example/0.png?signature=s0","type":"image","role":"last_frame"},
					{"url":"https://r2.example/1.png?signature=s1","type":"image","role":"first_frame"}
				]
			}`,
		},
		{
			name: "Seedance 2.0 strict first frame",
			body: fmt.Sprintf(`{
				"model":"seedance-2-0",
				"prompt":"animate from this frame",
				"generation_type":"first_frame",
				"duration":8,
				"resolution":"1080p",
				"aspect_ratio":"16:9",
				"images":["%s"]
			}`, imageDataURL),
			want: `{
				"model":"seedance-2-0",
				"prompt":"animate from this frame",
				"duration":8,
				"resolution":"1080p",
				"aspect_ratio":"16:9",
				"ref":[
					{"url":"https://r2.example/0.png?signature=s0","type":"image","role":"first_frame"}
				]
			}`,
		},
		{
			name: "Seedance 2.0 mixed image, video, and audio refs",
			body: fmt.Sprintf(`{
				"model":"seedance-2-0",
				"prompt":"保持 @图片1 中人物，衔接 @视频1 的动作，声音参考 @音频1",
				"generation_type":"reference_videos",
				"duration":10,
				"resolution":"720p",
				"aspect_ratio":"16:9",
				"media":[
					{"type":"image","role":"reference","source":"%s"},
					{"type":"image","role":"reference","source":"%s"},
					{"type":"video","role":"reference","source":"%s"},
					{"type":"video","role":"reference","source":"%s"},
					{"type":"audio","role":"reference","source":"%s"},
					{"type":"audio","role":"reference","source":"%s"}
				]
			}`, imageDataURL, imageDataURL, videoDataURL, videoDataURL, audioDataURL, audioDataURL),
			want: `{
				"model":"seedance-2-0",
				"prompt":"保持 @图片1 中人物，衔接 @视频1 的动作，声音参考 @音频1",
				"duration":10,
				"resolution":"720p",
				"aspect_ratio":"16:9",
				"ref":[
					{"url":"https://r2.example/0.png?signature=s0","type":"image"},
					{"url":"https://r2.example/1.png?signature=s1","type":"image"},
					{"url":"https://r2.example/2.mp4?signature=s2","type":"video"},
					{"url":"https://r2.example/3.mp4?signature=s3","type":"video"},
					{"url":"https://r2.example/4.mp3?signature=s4","type":"audio"},
					{"url":"https://r2.example/5.mp3?signature=s5","type":"audio"}
				]
			}`,
		},
		{
			name: "Seedance 2.0 video-only reference",
			body: fmt.Sprintf(`{
				"model":"seedance-2-0",
				"prompt":"参考 @视频1 的动作与镜头节奏生成新视频",
				"generation_type":"reference_videos",
				"duration":4,
				"resolution":"720p",
				"aspect_ratio":"16:9",
				"reference_videos":["%s"]
			}`, videoDataURL),
			want: `{
				"model":"seedance-2-0",
				"prompt":"参考 @视频1 的动作与镜头节奏生成新视频",
				"duration":4,
				"resolution":"720p",
				"aspect_ratio":"16:9",
				"ref":[
					{"url":"https://r2.example/0.mp4?signature=s0","type":"video"}
				]
			}`,
		},
		{
			name: "Seedance 2.0 MOV and WAV mixed refs",
			body: fmt.Sprintf(`{
				"model":"seedance-2-0",
				"prompt":"人物先参考 @视频1，说话声音参考 @音频1",
				"generation_type":"reference_videos",
				"duration":8,
				"resolution":"720p",
				"aspect_ratio":"16:9",
				"media":[
					{"type":"video","role":"reference","source":"%s"},
					{"type":"audio","role":"reference","source":"%s"}
				]
			}`, movDataURL, wavDataURL),
			want: `{
				"model":"seedance-2-0",
				"prompt":"人物先参考 @视频1，说话声音参考 @音频1",
				"duration":8,
				"resolution":"720p",
				"aspect_ratio":"16:9",
				"ref":[
					{"url":"https://r2.example/0.mov?signature=s0","type":"video"},
					{"url":"https://r2.example/1.wav?signature=s1","type":"audio"}
				]
			}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			context, info := newBrioiContext(
				t,
				"/v1/video/generations",
				test.body,
				"",
				"",
			)
			adaptor := &TaskAdaptor{}
			staged := 0
			adaptor.stageInput = func(
				_ stdcontext.Context,
				channelID int,
				source string,
			) (string, error) {
				assert.Equal(t, 42, channelID)
				assert.True(
					t,
					strings.HasPrefix(source, "data:image/") ||
						strings.HasPrefix(source, "data:video/") ||
						strings.HasPrefix(source, "data:audio/"),
				)
				ext := ".png"
				switch {
				case strings.HasPrefix(source, "data:video/quicktime"):
					ext = ".mov"
				case strings.HasPrefix(source, "data:video/"):
					ext = ".mp4"
				case strings.HasPrefix(source, "data:audio/wav"),
					strings.HasPrefix(source, "data:audio/x-wav"),
					strings.HasPrefix(source, "data:audio/wave"):
					ext = ".wav"
				case strings.HasPrefix(source, "data:audio/"):
					ext = ".mp3"
				}
				url := fmt.Sprintf(
					"https://r2.example/%d%s?signature=s%d",
					staged,
					ext,
					staged,
				)
				staged++
				return url, nil
			}

			require.Nil(t, adaptor.ValidateRequestAndSetAction(context, info))
			reader, err := adaptor.BuildRequestBody(context, info)
			require.NoError(t, err)
			body, err := io.ReadAll(reader)
			require.NoError(t, err)

			assert.JSONEq(t, test.want, string(body))
			assert.NotContains(t, string(body), "generation_type")
			assert.NotContains(t, string(body), "data:image")
			assert.NotContains(t, string(body), "data:video")
			assert.NotContains(t, string(body), "data:audio")
			assert.NotContains(t, string(body), `"images"`)
		})
	}
}

func TestRequestValidationUsesMappedModelAndBoundedBillingSeconds(t *testing.T) {
	context, info := newBrioiContext(t, "/v1/video/generations", `{
		"model":"public-seedance",
		"prompt":"mapped model",
		"generation_type":"text2video",
		"duration":15,
		"resolution":"720p",
		"aspect_ratio":"16:9"
	}`, "public-seedance", ModelSeedance20)
	adaptor := &TaskAdaptor{}

	require.Nil(t, adaptor.ValidateRequestAndSetAction(context, info))

	stored, ok := getNormalizedRequest(context)
	require.True(t, ok)
	assert.Equal(t, ModelSeedance20, stored.request.Model)
	assert.Equal(t, constant.TaskActionTextGenerate, info.Action)
	require.NotNil(t, info.RequestSnapshot)
	assert.Equal(t, "text2video", info.RequestSnapshot.GenerationType)
	assert.Equal(t, map[string]float64{"seconds": 15}, adaptor.EstimateBilling(context, info))
}

func TestValidateRequestStoresSanitizedMixedReferenceSnapshot(t *testing.T) {
	imageDataURL := brioiTestImageDataURL(t)
	videoDataURL := "data:video/mp4;base64," + base64.StdEncoding.EncodeToString([]byte("fake-mp4"))
	audioDataURL := "data:audio/mpeg;base64," + base64.StdEncoding.EncodeToString([]byte("fake-mp3"))
	body := fmt.Sprintf(`{
		"model":"seedance-2-0",
		"prompt":"保持 @图片1，衔接 @视频1",
		"generation_type":"reference_videos",
		"duration":10,
		"resolution":"720p",
		"aspect_ratio":"16:9",
		"media":[
			{"type":"image","role":"reference","source":"%s"},
			{"type":"video","role":"reference","source":"%s"},
			{"type":"audio","role":"reference","source":"%s"}
		]
	}`, imageDataURL, videoDataURL, audioDataURL)
	context, info := newBrioiContext(t, "/v1/video/generations", body, ModelSeedance20, ModelSeedance20)

	require.Nil(t, (&TaskAdaptor{}).ValidateRequestAndSetAction(context, info))

	assert.Equal(t, constant.TaskActionReferenceGenerate, info.Action)
	require.NotNil(t, info.RequestSnapshot)
	assert.Equal(t, "reference_videos", info.RequestSnapshot.GenerationType)
	assert.Equal(t, "保持 @图片1，衔接 @视频1", info.RequestSnapshot.Prompt)
	assert.NotContains(t, info.RequestSnapshot.Prompt, "data:")
	assert.Equal(t, []relaycommon.TaskMediaSnapshot{
		{Type: "image", Role: "reference"},
		{Type: "video", Role: "reference"},
		{Type: "audio", Role: "reference"},
	}, info.RequestSnapshot.Media)
}

func TestRequestDerivesResolutionFromMappedOriginModel(t *testing.T) {
	tests := []struct {
		name         string
		originModel  string
		body         string
		wantRes      string
		wantContains string
	}{
		{
			name:        "derives 480p from local alias",
			originModel: "seedance-2-0-480p",
			body: `{
				"model":"seedance-2-0-480p",
				"prompt":"auto resolution",
				"generation_type":"text2video",
				"duration":4,
				"aspect_ratio":"16:9"
			}`,
			wantRes: "480p",
		},
		{
			name:        "derives 4K from alias and keeps upstream model",
			originModel: "dreamina-seedance-2-0-4K",
			body: `{
				"model":"dreamina-seedance-2-0-4K",
				"prompt":"auto resolution",
				"generation_type":"text2video",
				"duration":4,
				"aspect_ratio":"16:9"
			}`,
			wantRes: "4K",
		},
		{
			name:        "accepts matching explicit resolution",
			originModel: "seedance-2-0-720p",
			body: `{
				"model":"seedance-2-0-720p",
				"prompt":"auto resolution",
				"generation_type":"text2video",
				"duration":4,
				"resolution":"720p",
				"aspect_ratio":"16:9"
			}`,
			wantRes: "720p",
		},
		{
			name:        "rejects conflicting explicit resolution",
			originModel: "seedance-2-0-480p",
			body: `{
				"model":"seedance-2-0-480p",
				"prompt":"auto resolution",
				"generation_type":"text2video",
				"duration":4,
				"resolution":"720p",
				"aspect_ratio":"16:9"
			}`,
			wantContains: "conflicts",
		},
		{
			name:        "still requires resolution when model has no suffix",
			originModel: "seedance-2-0",
			body: `{
				"model":"seedance-2-0",
				"prompt":"missing resolution",
				"generation_type":"text2video",
				"duration":4,
				"aspect_ratio":"16:9"
			}`,
			wantContains: "resolution is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			context, info := newBrioiContext(
				t,
				"/v1/video/generations",
				test.body,
				test.originModel,
				ModelSeedance20,
			)
			adaptor := &TaskAdaptor{}
			taskErr := adaptor.ValidateRequestAndSetAction(context, info)
			if test.wantContains != "" {
				require.NotNil(t, taskErr)
				assert.Contains(t, taskErr.Message, test.wantContains)
				return
			}
			require.Nil(t, taskErr)
			stored, ok := getNormalizedRequest(context)
			require.True(t, ok)
			assert.Equal(t, ModelSeedance20, stored.request.Model)
			assert.Equal(t, test.wantRes, stored.request.Resolution)

			reader, err := adaptor.BuildRequestBody(context, info)
			require.NoError(t, err)
			upstreamBody, err := io.ReadAll(reader)
			require.NoError(t, err)
			assert.Contains(t, string(upstreamBody), `"resolution":"`+test.wantRes+`"`)
			assert.Contains(t, string(upstreamBody), `"model":"`+ModelSeedance20+`"`)
		})
	}
}

func TestResolutionFromModelName(t *testing.T) {
	assert.Equal(t, "480p", resolutionFromModelName("seedance-2-0-480p"))
	assert.Equal(t, "720p", resolutionFromModelName("dreamina-seedance-2-0-720p-ref"))
	assert.Equal(t, "1080p", resolutionFromModelName("seedance-2-0-1080p"))
	assert.Equal(t, "4K", resolutionFromModelName("seedance-2-0-4k"))
	assert.Equal(t, "", resolutionFromModelName("seedance-2-0"))
}

func TestRequestValidationRejectsProtocolBoundaries(t *testing.T) {
	validImage := brioiTestImageDataURL(t)
	base := map[string]any{
		"model":           ModelSeedance20,
		"prompt":          "valid prompt",
		"generation_type": GenerationTextToVideo,
		"duration":        4,
		"resolution":      "720p",
		"aspect_ratio":    "16:9",
	}
	tests := []struct {
		name     string
		mutate   func(map[string]any)
		upstream string
		contains string
	}{
		{
			name:     "unknown mapped model",
			upstream: "seedance-unknown",
			contains: "not supported",
		},
		{
			name:     "explicit zero duration",
			mutate:   func(body map[string]any) { body["duration"] = 0 },
			contains: "between 4 and 15",
		},
		{
			name:     "prompt above provider character limit",
			mutate:   func(body map[string]any) { body["prompt"] = strings.Repeat("画", 5001) },
			contains: "5000 characters",
		},
		{
			name:     "duration above Seedance 2.0 maximum",
			mutate:   func(body map[string]any) { body["duration"] = 16 },
			contains: "between 4 and 15",
		},
		{
			name:     "duration above global billing bound",
			mutate:   func(body map[string]any) { body["duration"] = 999999999999 },
			contains: "between 0 and",
		},
		{
			name:     "unsupported resolution",
			mutate:   func(body map[string]any) { body["resolution"] = "8K" },
			contains: "resolution",
		},
		{
			name:     "unsupported aspect ratio",
			mutate:   func(body map[string]any) { body["aspect_ratio"] = "2:1" },
			contains: "aspect_ratio",
		},
		{
			name: "audio reference",
			mutate: func(body map[string]any) {
				body["audio_url"] = "data:audio/mpeg;base64,YXVkaW8="
			},
			contains: "only supported",
		},
		{
			name: "video_url singular rejected",
			mutate: func(body map[string]any) {
				body["video_url"] = "data:video/mp4;base64,dmlkZW8="
			},
			contains: "video_url is not supported",
		},
		{
			name:     "unknown top-level field",
			mutate:   func(body map[string]any) { body["temperature"] = 0 },
			contains: "unknown field",
		},
		{
			name: "remote input bypasses mandatory staging",
			mutate: func(body map[string]any) {
				body["generation_type"] = GenerationImageToVideo
				body["images"] = []string{"https://example.com/input.png"}
			},
			contains: "inline image data URL",
		},
		{
			name: "malformed image base64",
			mutate: func(body map[string]any) {
				body["generation_type"] = GenerationImageToVideo
				body["images"] = []string{"data:image/png;base64,%%%"}
			},
			contains: "invalid video data URL payload",
		},
		{
			name: "spoofed image payload",
			mutate: func(body map[string]any) {
				body["generation_type"] = GenerationImageToVideo
				body["images"] = []string{"data:image/png;base64,dGV4dA=="}
			},
			contains: "does not match declared type",
		},
		{
			name: "valid image reaches generation rules",
			mutate: func(body map[string]any) {
				body["generation_type"] = GenerationImageToVideo
				body["images"] = []string{validImage, validImage}
			},
			contains: "exactly one",
		},
		{
			name:     "Seedance 2.5 rejects 4K",
			upstream: ModelSeedance25,
			mutate: func(body map[string]any) {
				body["model"] = ModelSeedance25
				body["resolution"] = "4K"
			},
			contains: "resolution",
		},
		{
			name:     "Seedance 2.5 rejects unsupported ratio",
			upstream: ModelSeedance25,
			mutate: func(body map[string]any) {
				body["model"] = ModelSeedance25
				body["aspect_ratio"] = "1:1"
			},
			contains: "aspect_ratio",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := make(map[string]any, len(base))
			for key, value := range base {
				body[key] = value
			}
			if test.mutate != nil {
				test.mutate(body)
			}
			encoded, err := common.Marshal(body)
			require.NoError(t, err)
			upstream := test.upstream
			if upstream == "" {
				upstream = ModelSeedance20
			}
			context, info := newBrioiContext(
				t,
				"/v1/video/generations",
				string(encoded),
				ModelSeedance20,
				upstream,
			)

			taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(context, info)

			require.NotNil(t, taskErr)
			assert.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
			assert.Contains(t, taskErr.Message, test.contains)
		})
	}
}

func TestRequestValidationEnforcesReferenceItemHardLimits(t *testing.T) {
	tests := []struct {
		name       string
		model      string
		count      int
		duration   int
		resolution string
		contains   string
	}{
		{
			name:       "Seedance 2.0 rejects tenth reference",
			model:      ModelSeedance20,
			count:      10,
			duration:   8,
			resolution: "720p",
			contains:   "between 2 and 9",
		},
		{
			name:       "Seedance 2.5 rejects thirty-first reference",
			model:      ModelSeedance25,
			count:      31,
			duration:   20,
			resolution: "720p",
			contains:   "between 2 and 30",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			images := make([]string, test.count)
			for index := range images {
				images[index] = brioiTestImageDataURL(t)
			}
			body, err := common.Marshal(map[string]any{
				"model":           test.model,
				"prompt":          "many references",
				"generation_type": GenerationMultiImage,
				"duration":        test.duration,
				"resolution":      test.resolution,
				"aspect_ratio":    "16:9",
				"images":          images,
			})
			require.NoError(t, err)
			context, info := newBrioiContext(
				t,
				"/v1/video/generations",
				string(body),
				test.model,
				test.model,
			)

			taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(context, info)

			require.NotNil(t, taskErr)
			assert.Contains(t, taskErr.Message, test.contains)
		})
	}
}

func TestRequestValidationHonorsDisabledConfiguredOptions(t *testing.T) {
	setting := brioi_setting.GetBrioiSetting()
	backup, err := common.Marshal(setting)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, common.Unmarshal(backup, setting))
	})
	for index := range setting.Profiles {
		if setting.Profiles[index].Model != ModelSeedance20 {
			continue
		}
		setting.Profiles[index].Durations = []int{8}
		setting.Profiles[index].Resolutions = []string{"720p"}
		setting.Profiles[index].AspectRatios = []string{"16:9"}
		for modeIndex := range setting.Profiles[index].GenerationModes {
			if setting.Profiles[index].GenerationModes[modeIndex].Value == GenerationFirstFrame {
				setting.Profiles[index].GenerationModes[modeIndex].Enabled = false
			}
		}
	}

	context, info := newBrioiContext(t, "/v1/video/generations", fmt.Sprintf(`{
		"model":"seedance-2-0",
		"prompt":"disabled option",
		"generation_type":"first_frame",
		"duration":8,
		"resolution":"720p",
		"aspect_ratio":"16:9",
		"images":["%s"]
	}`, brioiTestImageDataURL(t)), ModelSeedance20, ModelSeedance20)

	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(context, info)

	require.NotNil(t, taskErr)
	assert.Contains(t, taskErr.Message, "generation_type")
	assert.Contains(t, taskErr.Message, "not enabled")
}

func TestRequestValidationRejectsInvalidRoleCombinations(t *testing.T) {
	imageDataURL := brioiTestImageDataURL(t)
	tests := []struct {
		name       string
		generation string
		media      []map[string]string
		contains   string
	}{
		{
			name:       "last frame without first frame",
			generation: GenerationStartEnd,
			media: []map[string]string{{
				"type": "image", "role": "last_frame", "source": imageDataURL,
			}},
			contains: "requires first_frame",
		},
		{
			name:       "duplicate first frame",
			generation: GenerationStartEnd,
			media: []map[string]string{
				{"type": "image", "role": "first_frame", "source": imageDataURL},
				{"type": "image", "role": "first_frame", "source": imageDataURL},
			},
			contains: "duplicate first_frame",
		},
		{
			name:       "strict and ordinary mixing",
			generation: GenerationStartEnd,
			media: []map[string]string{
				{"type": "image", "role": "first_frame", "source": imageDataURL},
				{"type": "image", "role": "reference", "source": imageDataURL},
			},
			contains: "mixed",
		},
		{
			name:       "audio media type",
			generation: GenerationImageToVideo,
			media: []map[string]string{{
				"type": "audio", "role": "reference", "source": "data:audio/mpeg;base64,YQ==",
			}},
			contains: "only supported",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body, err := common.Marshal(map[string]any{
				"model":           ModelSeedance20,
				"prompt":          "roles",
				"generation_type": test.generation,
				"duration":        8,
				"resolution":      "720p",
				"aspect_ratio":    "16:9",
				"media":           test.media,
			})
			require.NoError(t, err)
			context, info := newBrioiContext(
				t,
				"/v1/video/generations",
				string(body),
				ModelSeedance20,
				ModelSeedance20,
			)

			taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(context, info)

			require.NotNil(t, taskErr)
			assert.Contains(t, taskErr.Message, test.contains)
		})
	}
}

func TestReferenceVideosRejectsAudioOnlyAndOverCap(t *testing.T) {
	video := "data:video/mp4;base64," + base64.StdEncoding.EncodeToString([]byte("fake-mp4"))
	audio := "data:audio/mpeg;base64," + base64.StdEncoding.EncodeToString([]byte("fake-mp3"))
	tests := []struct {
		name     string
		media    []map[string]string
		contains string
	}{
		{
			name: "audio without video",
			media: []map[string]string{{
				"type": "audio", "role": "reference", "source": audio,
			}},
			contains: "between 1 and 3 reference videos",
		},
		{
			name: "four reference videos",
			media: []map[string]string{
				{"type": "video", "role": "reference", "source": video},
				{"type": "video", "role": "reference", "source": video},
				{"type": "video", "role": "reference", "source": video},
				{"type": "video", "role": "reference", "source": video},
			},
			contains: "between 1 and 3 reference videos",
		},
		{
			name: "four companion audios",
			media: []map[string]string{
				{"type": "video", "role": "reference", "source": video},
				{"type": "audio", "role": "reference", "source": audio},
				{"type": "audio", "role": "reference", "source": audio},
				{"type": "audio", "role": "reference", "source": audio},
				{"type": "audio", "role": "reference", "source": audio},
			},
			contains: "up to 3 companion reference audios",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body, err := common.Marshal(map[string]any{
				"model":           ModelSeedance20,
				"prompt":          "refs",
				"generation_type": GenerationReferenceVideos,
				"duration":        8,
				"resolution":      "720p",
				"aspect_ratio":    "16:9",
				"media":           test.media,
			})
			require.NoError(t, err)
			context, info := newBrioiContext(
				t,
				"/v1/video/generations",
				string(body),
				ModelSeedance20,
				ModelSeedance20,
			)

			taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(context, info)

			require.NotNil(t, taskErr)
			assert.Contains(t, taskErr.Message, test.contains)
		})
	}
}

func TestBuildRequestBodyStopsOnPartialStagingFailureWithoutLeakingMedia(t *testing.T) {
	secondImage := brioiTestImageDataURL(t)
	context, info := newBrioiContext(t, "/v1/video/generations", fmt.Sprintf(`{
		"model":"seedance-2-0",
		"prompt":"staging",
		"generation_type":"multi_image",
		"duration":8,
		"resolution":"720p",
		"aspect_ratio":"16:9",
		"images":["%s","%s"]
	}`, brioiTestImageDataURL(t), secondImage), ModelSeedance20, ModelSeedance20)
	adaptor := &TaskAdaptor{}
	stageCalls := 0
	adaptor.stageInput = func(_ stdcontext.Context, _ int, _ string) (string, error) {
		stageCalls++
		if stageCalls == 2 {
			return "", errors.New("R2 upload failed")
		}
		return "https://r2.example/first.png?signature=secret", nil
	}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(context, info))

	reader, err := adaptor.BuildRequestBody(context, info)

	require.Error(t, err)
	assert.Nil(t, reader)
	assert.Equal(t, 2, stageCalls)
	assert.NotContains(t, err.Error(), secondImage)
	assert.NotContains(t, err.Error(), "signature=secret")
}

func TestBuildRequestBodyRejectsNonHTTPSStagingResult(t *testing.T) {
	context, info := newBrioiContext(t, "/v1/video/generations", fmt.Sprintf(`{
		"model":"seedance-2-0",
		"prompt":"secure staging",
		"generation_type":"image2video",
		"duration":8,
		"resolution":"720p",
		"aspect_ratio":"16:9",
		"media":[
			{"type":"image","role":"reference","source":"%s"}
		]
	}`, brioiTestImageDataURL(t)), ModelSeedance20, ModelSeedance20)
	adaptor := &TaskAdaptor{
		stageInput: func(_ stdcontext.Context, _ int, _ string) (string, error) {
			return "http://r2.example/input.png?signature=secret", nil
		},
	}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(context, info))

	reader, err := adaptor.BuildRequestBody(context, info)

	require.ErrorContains(t, err, "did not return an HTTPS URL")
	assert.Nil(t, reader)
	assert.NotContains(t, err.Error(), "signature=secret")
}

func TestValidateRequestRejectsDeferredVideosRoute(t *testing.T) {
	context, info := newBrioiContext(t, "/v1/videos", `{
		"model":"seedance-2-0",
		"prompt":"deferred route",
		"generation_type":"text2video",
		"duration":8,
		"resolution":"720p",
		"aspect_ratio":"16:9"
	}`, ModelSeedance20, ModelSeedance20)

	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(context, info)

	require.NotNil(t, taskErr)
	assert.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
	assert.Contains(t, taskErr.Message, "/v1/video/generations")
	assert.NotContains(t, taskErr.Message, "Brioi")
}
