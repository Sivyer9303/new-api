package controller

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func allowBrioiR2Validation(t *testing.T) {
	t.Helper()
	model.RegisterVideoR2StorageValidator(func() error { return nil })
	t.Cleanup(func() {
		model.RegisterVideoR2StorageValidator(service.ValidateVideoR2StorageConfigured)
	})
}

func brioiR2ChannelSetting(t *testing.T) *string {
	t.Helper()
	setting, err := common.Marshal(dto.ChannelSettings{
		VideoInputMediaDelivery: dto.VideoInputMediaR2Presigned,
	})
	require.NoError(t, err)
	value := string(setting)
	return &value
}

func TestValidateChannelProxy(t *testing.T) {
	tests := []struct {
		name    string
		proxy   string
		wantErr bool
	}{
		{name: "empty"},
		{name: "http", proxy: "http://proxy.example:8080"},
		{name: "https", proxy: "https://proxy.example:8443"},
		{name: "socks5", proxy: "socks5://proxy.example"},
		{name: "socks5h", proxy: "socks5h://proxy.example:1080/"},
		{name: "unsupported", proxy: "ftp://proxy.example", wantErr: true},
		{name: "path", proxy: "socks5://proxy.example:1080/path", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setting, err := common.Marshal(dto.ChannelSettings{Proxy: test.proxy})
			require.NoError(t, err)
			channel := &model.Channel{
				Type:    constant.ChannelTypeOpenAI,
				Setting: common.GetPointer(string(setting)),
			}

			err = validateChannel(channel, false)

			if test.wantErr {
				require.ErrorContains(t, err, "invalid channel proxy")
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateChannelRequiresNewAPIBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL *string
		wantErr bool
	}{
		{name: "missing", wantErr: true},
		{name: "blank", baseURL: common.GetPointer("  "), wantErr: true},
		{name: "configured", baseURL: common.GetPointer("https://new-api.example")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			channel := &model.Channel{
				Type:    constant.ChannelTypeNewAPI,
				BaseURL: test.baseURL,
			}

			err := validateChannel(channel, false)

			if test.wantErr {
				require.ErrorContains(t, err, "New API channel base URL cannot be empty")
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateChannelRequiresSilkRoadBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL *string
		wantErr bool
	}{
		{name: "missing", wantErr: true},
		{name: "blank", baseURL: common.GetPointer("  "), wantErr: true},
		{name: "configured", baseURL: common.GetPointer("https://silkroad.example")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			channel := &model.Channel{
				Type:    constant.ChannelTypeSilkRoad,
				BaseURL: test.baseURL,
			}

			err := validateChannel(channel, false)

			if test.wantErr {
				require.ErrorContains(t, err, "SilkRoad channel base URL cannot be empty")
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateChannelRequiresBrioiBaseURL(t *testing.T) {
	allowBrioiR2Validation(t)
	tests := []struct {
		name    string
		baseURL *string
		wantErr bool
	}{
		{name: "missing", wantErr: true},
		{name: "blank", baseURL: common.GetPointer("  "), wantErr: true},
		{name: "configured", baseURL: common.GetPointer("https://brioi.example")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			channel := &model.Channel{
				Type:    constant.ChannelTypeBrioi,
				BaseURL: test.baseURL,
				Setting: brioiR2ChannelSetting(t),
			}

			err := validateChannel(channel, false)

			if test.wantErr {
				require.ErrorContains(t, err, "Brioi channel base URL cannot be empty")
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateChannelRequiresBrioiR2DeliveryAndStorage(t *testing.T) {
	baseURL := "https://brioi.example"
	inlineSetting, err := common.Marshal(dto.ChannelSettings{
		VideoInputMediaDelivery: dto.VideoInputMediaInlineBase64,
	})
	require.NoError(t, err)
	inlineSettingValue := string(inlineSetting)

	allowBrioiR2Validation(t)
	err = validateChannel(&model.Channel{
		Type:    constant.ChannelTypeBrioi,
		BaseURL: &baseURL,
		Setting: &inlineSettingValue,
	}, false)
	require.ErrorContains(t, err, "Brioi requires R2 presigned URL input delivery")

	model.RegisterVideoR2StorageValidator(func() error {
		return errors.New("R2 unavailable")
	})
	err = validateChannel(&model.Channel{
		Type:    constant.ChannelTypeBrioi,
		BaseURL: &baseURL,
		Setting: brioiR2ChannelSetting(t),
	}, false)
	require.ErrorContains(t, err, "video_input_media_delivery requires R2 video storage")
	require.ErrorContains(t, err, "R2 unavailable")
}

func TestNewAPIChannelRegistration(t *testing.T) {
	apiType, ok := common.ChannelType2APIType(constant.ChannelTypeNewAPI)

	require.True(t, ok)
	assert.Equal(t, constant.APITypeNewAPI, apiType)
	assert.Equal(t, "New API", constant.GetChannelTypeName(constant.ChannelTypeNewAPI))
	require.Greater(t, len(constant.ChannelBaseURLs), constant.ChannelTypeNewAPI)
	assert.Empty(t, constant.ChannelBaseURLs[constant.ChannelTypeNewAPI])
}

func TestSilkRoadChannelRegistration(t *testing.T) {
	apiType, ok := common.ChannelType2APIType(constant.ChannelTypeSilkRoad)

	assert.False(t, ok)
	assert.Equal(t, constant.APITypeOpenAI, apiType)
	assert.Equal(t, 61, constant.ChannelTypeSilkRoad)
	assert.Equal(t, constant.ChannelTypeSilkRoad+1, constant.ChannelTypeBrioi)
	assert.Equal(t, "SilkRoad", constant.GetChannelTypeName(constant.ChannelTypeSilkRoad))
	require.Greater(t, len(constant.ChannelBaseURLs), constant.ChannelTypeSilkRoad)
	assert.Empty(t, constant.ChannelBaseURLs[constant.ChannelTypeSilkRoad])
	assert.Equal(
		t,
		[]constant.EndpointType{constant.EndpointTypeOpenAIVideo},
		common.GetEndpointTypesByChannelType(constant.ChannelTypeSilkRoad, "seedance-2.0"),
	)
	assert.Equal(
		t,
		[]constant.EndpointType{constant.EndpointTypeOpenAIVideo},
		common.GetEndpointTypesByChannelType(constant.ChannelTypeSilkRoad, "dall-e-3"),
	)
}

func TestBrioiChannelRegistration(t *testing.T) {
	apiType, ok := common.ChannelType2APIType(constant.ChannelTypeBrioi)

	assert.False(t, ok)
	assert.Equal(t, constant.APITypeOpenAI, apiType)
	assert.Equal(t, 62, constant.ChannelTypeBrioi)
	assert.Equal(t, constant.ChannelTypeBrioi+1, constant.ChannelTypeCompatVideo)
	assert.Equal(t, "Brioi", constant.GetChannelTypeName(constant.ChannelTypeBrioi))
	require.Greater(t, len(constant.ChannelBaseURLs), constant.ChannelTypeBrioi)
	assert.Empty(t, constant.ChannelBaseURLs[constant.ChannelTypeBrioi])
	assert.Equal(
		t,
		[]constant.EndpointType{constant.EndpointTypeOpenAIVideo},
		common.GetEndpointTypesByChannelType(constant.ChannelTypeBrioi, "seedance-2-0"),
	)
	assert.True(t, common.ChannelTypeSupportsRequestPath(
		constant.ChannelTypeBrioi,
		"/v1/video/generations",
	))
	assert.False(t, common.ChannelTypeSupportsRequestPath(
		constant.ChannelTypeBrioi,
		"/v1/videos",
	))
	assert.False(t, common.ChannelTypeSupportsRequestPath(
		constant.ChannelTypeBrioi,
		"/v1/chat/completions",
	))
}

func TestCompatVideoChannelRegistration(t *testing.T) {
	apiType, ok := common.ChannelType2APIType(constant.ChannelTypeCompatVideo)

	assert.False(t, ok)
	assert.Equal(t, constant.APITypeOpenAI, apiType)
	assert.Equal(t, 63, constant.ChannelTypeCompatVideo)
	assert.Equal(t, "xtoken", constant.GetChannelTypeName(constant.ChannelTypeCompatVideo))
	require.Greater(t, len(constant.ChannelBaseURLs), constant.ChannelTypeCompatVideo)
	assert.Empty(t, constant.ChannelBaseURLs[constant.ChannelTypeCompatVideo])
	assert.Equal(
		t,
		[]constant.EndpointType{constant.EndpointTypeOpenAIVideo},
		common.GetEndpointTypesByChannelType(constant.ChannelTypeCompatVideo, "grok-image-video"),
	)
	assert.True(t, common.ChannelTypeSupportsRequestPath(
		constant.ChannelTypeCompatVideo,
		"/v1/video/generations",
	))
	assert.True(t, common.ChannelTypeSupportsRequestPath(
		constant.ChannelTypeCompatVideo,
		"/v1/videos",
	))
	assert.False(t, common.ChannelTypeSupportsRequestPath(
		constant.ChannelTypeCompatVideo,
		"/v1/chat/completions",
	))
}

func TestAIStarsLabChannelRegistration(t *testing.T) {
	apiType, ok := common.ChannelType2APIType(constant.ChannelTypeAIStarsLab)

	assert.False(t, ok)
	assert.Equal(t, constant.APITypeOpenAI, apiType)
	assert.Equal(t, 64, constant.ChannelTypeAIStarsLab)
	assert.Equal(t, constant.ChannelTypeCompatVideo+1, constant.ChannelTypeAIStarsLab)
	assert.Equal(t, constant.ChannelTypeAIStarsLab+1, constant.ChannelTypeDummy)
	assert.Equal(t, "AIStarsLab", constant.GetChannelTypeName(constant.ChannelTypeAIStarsLab))
	require.Greater(t, len(constant.ChannelBaseURLs), constant.ChannelTypeAIStarsLab)
	assert.Equal(t, "https://api.video.aistarslab.com/openai", constant.ChannelBaseURLs[constant.ChannelTypeAIStarsLab])
	assert.Equal(
		t,
		[]constant.EndpointType{constant.EndpointTypeOpenAIVideo},
		common.GetEndpointTypesByChannelType(constant.ChannelTypeAIStarsLab, "test:test-video"),
	)
	assert.True(t, common.ChannelTypeSupportsRequestPath(
		constant.ChannelTypeAIStarsLab,
		"/v1/video/generations",
	))
	assert.True(t, common.ChannelTypeSupportsRequestPath(
		constant.ChannelTypeAIStarsLab,
		"/v1/videos",
	))
	assert.False(t, common.ChannelTypeSupportsRequestPath(
		constant.ChannelTypeAIStarsLab,
		"/v1/chat/completions",
	))
}

func TestBrioiChannelTestUsesNonBillableModelListEndpoint(t *testing.T) {
	allowBrioiR2Validation(t)
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/models", r.URL.Path)
		assert.Equal(t, "Bearer brioi-key", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"data":[{"id":"seedance-2-0"},{"id":"seedance-2-5"}]}`))
	}))
	t.Cleanup(server.Close)

	baseURL := server.URL + "/"
	result := testChannel(context.Background(), &model.Channel{
		Type:    constant.ChannelTypeBrioi,
		Key:     "brioi-key",
		BaseURL: &baseURL,
		Setting: brioiR2ChannelSetting(t),
	}, 0, "seedance-2-0", "", false)

	require.NoError(t, result.localErr)
	assert.Nil(t, result.newAPIError)
	assert.Equal(t, []string{"seedance-2-0", "seedance-2-5"}, result.models)
	assert.Equal(t, 1, requestCount)
}

func TestBrioiChannelTestUsesConfiguredProxy(t *testing.T) {
	allowBrioiR2Validation(t)
	proxyCalls := 0
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyCalls++
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "brioi.invalid", r.URL.Host)
		assert.Equal(t, "/v1/models", r.URL.Path)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	t.Cleanup(proxy.Close)

	settingBytes, err := common.Marshal(dto.ChannelSettings{
		Proxy:                   proxy.URL,
		VideoInputMediaDelivery: dto.VideoInputMediaR2Presigned,
	})
	require.NoError(t, err)
	setting := string(settingBytes)
	baseURL := "http://brioi.invalid"
	result := testChannel(context.Background(), &model.Channel{
		Type:    constant.ChannelTypeBrioi,
		Key:     "brioi-key",
		BaseURL: &baseURL,
		Setting: &setting,
	}, 0, "", "", false)

	require.NoError(t, result.localErr)
	assert.Empty(t, result.models)
	assert.Equal(t, 1, proxyCalls)
}

func TestBrioiChannelTestReportsAuthenticationFailureWithoutSubmittingVideo(t *testing.T) {
	allowBrioiR2Validation(t)
	modelCalls := 0
	videoCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			modelCalls++
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"invalid Brioi key"}}`))
		case "/v1/videos":
			videoCalls++
			w.WriteHeader(http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	baseURL := server.URL
	result := testChannel(context.Background(), &model.Channel{
		Type:    constant.ChannelTypeBrioi,
		Key:     "bad-key",
		BaseURL: &baseURL,
		Setting: brioiR2ChannelSetting(t),
	}, 0, "", "", false)

	require.Error(t, result.localErr)
	require.NotNil(t, result.newAPIError)
	assert.Contains(t, result.localErr.Error(), "invalid Brioi key")
	assert.Equal(t, 1, modelCalls)
	assert.Zero(t, videoCalls)
}

func TestBrioiChannelTestRequiresR2BeforeCallingUpstream(t *testing.T) {
	model.RegisterVideoR2StorageValidator(func() error {
		return errors.New("R2 unavailable")
	})
	t.Cleanup(func() {
		model.RegisterVideoR2StorageValidator(service.ValidateVideoR2StorageConfigured)
	})

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	t.Cleanup(server.Close)

	baseURL := server.URL
	result := testChannel(context.Background(), &model.Channel{
		Type:    constant.ChannelTypeBrioi,
		Key:     "brioi-key",
		BaseURL: &baseURL,
		Setting: brioiR2ChannelSetting(t),
	}, 0, "", "", false)

	require.ErrorContains(t, result.localErr, "R2 unavailable")
	assert.Zero(t, requestCount)
}

func TestSilkRoadChannelSkipsSynchronousChannelTest(t *testing.T) {
	result := testChannel(
		context.Background(),
		&model.Channel{Type: constant.ChannelTypeSilkRoad},
		0,
		"seedance-2.0",
		"",
		false,
	)

	require.ErrorContains(t, result.localErr, "SilkRoad channel test is not supported")
}

func TestResponsesCompactChannelSupport(t *testing.T) {
	tests := []struct {
		name        string
		channelType int
		apiType     int
		want        bool
	}{
		{name: "OpenAI", channelType: constant.ChannelTypeOpenAI, apiType: constant.APITypeOpenAI, want: true},
		{name: "Azure", channelType: constant.ChannelTypeAzure, apiType: constant.APITypeOpenAI, want: true},
		{name: "Codex", channelType: constant.ChannelTypeCodex, apiType: constant.APITypeCodex, want: true},
		{name: "Advanced Custom", channelType: constant.ChannelTypeAdvancedCustom, apiType: constant.APITypeAdvancedCustom, want: true},
		{name: "Sub2API", channelType: constant.ChannelTypeSub2API, apiType: constant.APITypeSub2API, want: true},
		{name: "New API", channelType: constant.ChannelTypeNewAPI, apiType: constant.APITypeNewAPI, want: true},
		{name: "Anthropic", channelType: constant.ChannelTypeAnthropic, apiType: constant.APITypeAnthropic, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, common.SupportsResponsesCompact(test.channelType, test.apiType))
		})
	}
}

func TestMultiprotocolGatewayEndpointTypes(t *testing.T) {
	want := []constant.EndpointType{
		constant.EndpointTypeOpenAI,
		constant.EndpointTypeOpenAIResponse,
		constant.EndpointTypeOpenAIResponseCompact,
		constant.EndpointTypeAnthropic,
		constant.EndpointTypeGemini,
		constant.EndpointTypeOpenAIAlphaSearch,
	}

	assert.Equal(t, want, common.GetEndpointTypesByChannelType(constant.ChannelTypeNewAPI, "gpt-5"))
	assert.Equal(t, want, common.GetEndpointTypesByChannelType(constant.ChannelTypeSub2API, "gpt-5"))
}

func TestCopyChannelRejectsInvalidLegacyProxySettings(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	settingBytes, err := common.Marshal(dto.ChannelSettings{
		Proxy: "socks5://proxy.example/legacy-path",
	})
	require.NoError(t, err)
	setting := string(settingBytes)
	origin := &model.Channel{
		Type:    constant.ChannelTypeOpenAI,
		Name:    "legacy proxy channel",
		Key:     "test-key",
		Models:  "gpt-test",
		Group:   "default",
		Setting: &setting,
	}
	require.NoError(t, db.Create(origin).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", origin.Id)}}
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/channel/copy", nil)

	CopyChannel(ctx)

	assert.Contains(t, recorder.Body.String(), "invalid channel settings")
	var channelCount int64
	require.NoError(t, db.Model(&model.Channel{}).Count(&channelCount).Error)
	assert.Equal(t, int64(1), channelCount)
}

func TestDeleteChannelResetsProxyCacheWhenPreReadFails(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Log{}))
	service.ResetProxyClientCache()
	t.Cleanup(service.ResetProxyClientCache)

	proxyURL := "http://proxy.example:8080"
	beforeDelete, err := service.GetHttpClientWithProxy(proxyURL)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "id", Value: "999999"}}
	ctx.Request = httptest.NewRequest(http.MethodDelete, "/api/channel/999999", nil)

	DeleteChannel(ctx)

	assert.Contains(t, recorder.Body.String(), `"success":true`)
	afterDelete, err := service.GetHttpClientWithProxy(proxyURL)
	require.NoError(t, err)
	assert.NotSame(t, beforeDelete, afterDelete)
}

func TestDeleteChannelBatchReportsAndAuditsActualDeletedCount(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Log{}))
	channel := &model.Channel{Name: "existing", Key: "test-key"}
	require.NoError(t, db.Create(channel).Error)

	requestBody, err := common.Marshal(ChannelBatch{Ids: []int{channel.Id, 999999}})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodDelete, "/api/channel/batch", bytes.NewReader(requestBody))
	ctx.Request.Header.Set("Content-Type", "application/json")

	DeleteChannelBatch(ctx)

	var response struct {
		Success bool  `json:"success"`
		Data    int64 `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Equal(t, int64(1), response.Data)

	var auditLog model.Log
	require.NoError(t, db.Order("id desc").First(&auditLog).Error)
	var auditData struct {
		Operation struct {
			Params map[string]any `json:"params"`
		} `json:"op"`
	}
	require.NoError(t, common.UnmarshalJsonStr(auditLog.Other, &auditData))
	assert.Equal(t, float64(1), auditData.Operation.Params["count"])
}

func TestSettleTestQuotaUsesTieredBilling(t *testing.T) {
	info := &relaycommon.RelayInfo{
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode:   "tiered_expr",
			ExprString:    `param("stream") == true ? tier("stream", p * 3) : tier("base", p * 2)`,
			ExprHash:      billingexpr.ExprHashString(`param("stream") == true ? tier("stream", p * 3) : tier("base", p * 2)`),
			GroupRatio:    1,
			EstimatedTier: "stream",
			QuotaPerUnit:  common.QuotaPerUnit,
			ExprVersion:   1,
		},
		BillingRequestInput: &billingexpr.RequestInput{
			Body: []byte(`{"stream":true}`),
		},
	}

	quota, result := settleTestQuota(info, types.PriceData{
		ModelRatio:      1,
		CompletionRatio: 2,
	}, &dto.Usage{
		PromptTokens: 1000,
	})

	require.Equal(t, 1500, quota)
	require.NotNil(t, result)
	require.Equal(t, "stream", result.MatchedTier)
}

func TestBuildTestLogOtherInjectsTieredInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	info := &relaycommon.RelayInfo{
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode: "tiered_expr",
			ExprString:  `tier("base", p * 2)`,
		},
		ChannelMeta: &relaycommon.ChannelMeta{},
	}
	priceData := types.PriceData{
		GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1},
	}
	usage := &dto.Usage{
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 12,
		},
	}

	requestRules := []billingexpr.RequestRuleTrace{{
		Cond:       `param("service_tier") == "fast"`,
		Multiplier: 2,
		Matched:    true,
	}}
	other := buildTestLogOther(ctx, info, priceData, usage, &billingexpr.TieredResult{
		MatchedTier:  "base",
		RequestRules: requestRules,
	})

	require.Equal(t, "tiered_expr", other["billing_mode"])
	require.Equal(t, "base", other["matched_tier"])
	require.Equal(t, requestRules, other["request_rules"])
	require.NotEmpty(t, other["expr_b64"])
}

func TestResolveChannelTestUserIDUsesRequestUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("id", 2)

	userID, err := resolveChannelTestUserID(ctx)

	require.NoError(t, err)
	require.Equal(t, 2, userID)
}

func TestSelectChannelsForAutomaticTestPassiveRecoveryOnlyUsesAutoDisabled(t *testing.T) {
	channels := []*model.Channel{
		{Id: 1, Status: common.ChannelStatusEnabled},
		{Id: 2, Status: common.ChannelStatusAutoDisabled},
		{Id: 3, Status: common.ChannelStatusManuallyDisabled},
	}

	selected := selectChannelsForAutomaticTest(channels, operation_setting.ChannelTestModePassiveRecovery)

	require.Len(t, selected, 1)
	require.Equal(t, 2, selected[0].Id)
}

func TestSelectChannelsForAutomaticTestScheduledSkipsManualDisabled(t *testing.T) {
	channels := []*model.Channel{
		{Id: 1, Status: common.ChannelStatusEnabled},
		{Id: 2, Status: common.ChannelStatusAutoDisabled},
		{Id: 3, Status: common.ChannelStatusManuallyDisabled},
	}

	selected := selectChannelsForAutomaticTest(channels, operation_setting.ChannelTestModeScheduledAll)

	require.Len(t, selected, 2)
	require.Equal(t, 1, selected[0].Id)
	require.Equal(t, 2, selected[1].Id)
}

func TestSelectChannelsForAutomaticTestAutoBanOnlyUsesEligibleChannels(t *testing.T) {
	autoBanEnabled := 1
	autoBanDisabled := 0
	channels := []*model.Channel{
		{Id: 1, Status: common.ChannelStatusEnabled, AutoBan: &autoBanEnabled},
		{Id: 2, Status: common.ChannelStatusEnabled, AutoBan: &autoBanDisabled},
		{Id: 3, Status: common.ChannelStatusAutoDisabled, AutoBan: &autoBanEnabled},
		{Id: 4, Status: common.ChannelStatusManuallyDisabled, AutoBan: &autoBanEnabled},
		{Id: 5, Status: common.ChannelStatusEnabled},
	}

	selected := selectChannelsForAutomaticTest(channels, operation_setting.ChannelTestModeAutoBanOnly)

	require.Len(t, selected, 2)
	require.Equal(t, 1, selected[0].Id)
	require.Equal(t, 3, selected[1].Id)
}

func TestTestAllChannelsRejectsExistingActiveTask(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.SystemTask{}, &model.SystemTaskLock{}))

	existing, err := model.CreateSystemTask(model.SystemTaskTypeChannelTest, nil, nil)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/channel/test", nil)

	TestAllChannels(ctx)

	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Contains(t, recorder.Body.String(), existing.TaskID)
	require.Contains(t, recorder.Body.String(), "已有通道测试任务正在运行或等待中")
}
