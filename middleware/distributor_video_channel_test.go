package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	appi18n "github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestChannelSupportsRequestPathKeepsSilkRoadVideoOnly(t *testing.T) {
	silkRoad := &model.Channel{Type: constant.ChannelTypeSilkRoad}
	tests := []struct {
		path string
		want bool
	}{
		{path: "/v1/videos", want: true},
		{path: "/v1/videos/task-id", want: true},
		{path: "/v1/video/generations", want: true},
		{path: "/v1/chat/completions", want: false},
		{path: "/v1/images/generations", want: false},
		{path: "/v1/audio/speech", want: false},
		{path: "/v1/embeddings", want: false},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			assert.Equal(t, test.want, channelSupportsRequestPath(silkRoad, test.path, "seedance-2.0"))
		})
	}
}

func TestChannelSupportsRequestPathKeepsBrioiOnVideoToolRouteOnly(t *testing.T) {
	brioi := &model.Channel{Type: constant.ChannelTypeBrioi}
	tests := []struct {
		path string
		want bool
	}{
		{path: "/v1/video/generations", want: true},
		{path: "/v1/video/generations/task-id", want: true},
		{path: "/v1/videos", want: false},
		{path: "/v1/videos/task-id", want: false},
		{path: "/v1/chat/completions", want: false},
		{path: "/v1/images/generations", want: false},
		{path: "/v1/audio/speech", want: false},
		{path: "/v1/embeddings", want: false},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			assert.Equal(t, test.want, channelSupportsRequestPath(brioi, test.path, "seedance-2-0"))
		})
	}
}

func TestChannelSupportsRequestPathKeepsCompatVideoOnVideoRoutes(t *testing.T) {
	channel := &model.Channel{Type: constant.ChannelTypeCompatVideo}
	tests := []struct {
		path string
		want bool
	}{
		{path: "/v1/videos", want: true},
		{path: "/v1/videos/task-id", want: true},
		{path: "/v1/video/generations", want: true},
		{path: "/v1/video/generations/task-id", want: true},
		{path: "/v1/chat/completions", want: false},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			assert.Equal(
				t,
				test.want,
				channelSupportsRequestPath(channel, test.path, "grok-image-video"),
			)
		})
	}
}

func TestChannelSupportsRequestPathKeepsAIStarsLabOnVideoRoutes(t *testing.T) {
	channel := &model.Channel{Type: constant.ChannelTypeAIStarsLab}
	tests := []struct {
		path string
		want bool
	}{
		{path: "/v1/videos", want: true},
		{path: "/v1/videos/task-id", want: true},
		{path: "/v1/video/generations", want: true},
		{path: "/v1/video/generations/task-id", want: true},
		{path: "/v1/chat/completions", want: false},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			assert.Equal(
				t,
				test.want,
				channelSupportsRequestPath(channel, test.path, "test:test-video"),
			)
		})
	}
}

func TestChannelSupportsRequestPathRejectsNewAPIVideoSubmissions(t *testing.T) {
	newAPI := &model.Channel{Type: constant.ChannelTypeNewAPI}

	assert.False(t, channelSupportsRequestPath(newAPI, "/v1/videos", "seedance-2.0"))
	assert.False(t, channelSupportsRequestPath(newAPI, "/v1/video/generations", "seedance-2.0"))
	assert.True(t, channelSupportsRequestPath(newAPI, "/v1/chat/completions", "gpt-5"))
}

func TestDistributeRejectsExplicitLegacyNewAPIVideoChannel(t *testing.T) {
	require.NoError(t, appi18n.Init())
	previousDB := model.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Channel{}))
	model.DB = db
	t.Cleanup(func() { model.DB = previousDB })

	baseURL := "https://new-api.example"
	channel := &model.Channel{
		Type:    constant.ChannelTypeNewAPI,
		Name:    "legacy video",
		Key:     "test-key",
		BaseURL: &baseURL,
		Models:  "seedance-2.0",
		Group:   "default",
		Status:  common.ChannelStatusEnabled,
	}
	require.NoError(t, db.Create(channel).Error)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		common.SetContextKey(
			c,
			constant.ContextKeyTokenSpecificChannelId,
			strconv.Itoa(channel.Id),
		)
		c.Next()
	})
	router.Use(Distribute())
	router.POST("/v1/videos", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	router.GET("/v1/videos/:task_id", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/videos",
		bytes.NewBufferString(`{"model":"seedance-2.0"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	queryRecorder := httptest.NewRecorder()
	queryRequest := httptest.NewRequest(
		http.MethodGet,
		"/v1/videos/task_public_legacy",
		nil,
	)
	router.ServeHTTP(queryRecorder, queryRequest)

	assert.Equal(t, http.StatusNoContent, queryRecorder.Code)
}
