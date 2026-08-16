package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/brioi_setting"
	"github.com/QuantumNous/new-api/setting/compatvideo_setting"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/console_setting"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/setting/silkroad_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/QuantumNous/new-api/setting/video_setting"

	"github.com/gin-gonic/gin"
)

var completionRatioMetaOptionKeys = []string{
	"ModelPrice",
	"ModelRatio",
	"CompletionRatio",
	"CacheRatio",
	"CreateCacheRatio",
	"ImageRatio",
	"AudioRatio",
	"AudioCompletionRatio",
}

var videoProviderGroupUpdateMu sync.Mutex

func isPaymentComplianceOptionKey(key string) bool {
	return strings.HasPrefix(key, "payment_setting.compliance_")
}

func isPositiveOptionValue(value string) bool {
	intValue, err := strconv.Atoi(strings.TrimSpace(value))
	if err == nil {
		return intValue > 0
	}
	floatValue, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return err == nil && floatValue > 0
}

// validateSilkRoadSettingOption applies a single silkroad_setting.* update to a
// copy of the live config and validates only the section being saved.
func validateSilkRoadSettingOption(key, value string) error {
	configKey := strings.TrimPrefix(key, "silkroad_setting.")
	if configKey == key || configKey == "" {
		return fmt.Errorf("invalid silkroad_setting option key")
	}

	current := silkroad_setting.GetSilkRoadSetting()
	raw, err := common.Marshal(current)
	if err != nil {
		return err
	}
	var clone silkroad_setting.SilkRoadSetting
	if err := common.Unmarshal(raw, &clone); err != nil {
		return err
	}
	switch configKey {
	case "common":
		if err := common.UnmarshalJsonStr(value, &clone.Common); err != nil {
			return fmt.Errorf("invalid silkroad common setting: %w", err)
		}
	case "profiles":
		if err := common.UnmarshalJsonStr(value, &clone.Profiles); err != nil {
			return fmt.Errorf("invalid silkroad profiles: %w", err)
		}
	case "default_profile_id":
		clone.DefaultProfileID = value
	case "storage":
		if err := common.UnmarshalJsonStr(value, &clone.Storage); err != nil {
			return fmt.Errorf("invalid silkroad storage setting: %w", err)
		}
	case "video_tool_groups":
		if err := common.UnmarshalJsonStr(value, &clone.VideoToolGroups); err != nil {
			return fmt.Errorf("invalid silkroad video tool groups: %w", err)
		}
	default:
		return fmt.Errorf("unsupported silkroad_setting option key %q", configKey)
	}
	if key != "silkroad_setting.default_profile_id" &&
		!config.GlobalConfig.IsExplicit("silkroad_setting.default_profile_id") &&
		len(clone.Profiles) > 0 {
		clone.DefaultProfileID = clone.Profiles[0].ID
	}
	if configKey == "storage" {
		return silkroad_setting.ValidateSilkRoadStorageSetting(&clone.Storage)
	}
	if err := silkroad_setting.ValidateSilkRoadProviderSetting(&clone); err != nil {
		return err
	}
	return nil
}

func validateBrioiSettingOption(key, value string) error {
	configKey := strings.TrimPrefix(key, "brioi_setting.")
	if configKey == key || configKey == "" {
		return fmt.Errorf("invalid brioi_setting option key")
	}

	current := brioi_setting.GetBrioiSetting()
	raw, err := common.Marshal(current)
	if err != nil {
		return err
	}
	var clone brioi_setting.BrioiSetting
	if err := common.Unmarshal(raw, &clone); err != nil {
		return err
	}
	switch configKey {
	case "profiles":
		if err := common.UnmarshalJsonStr(value, &clone.Profiles); err != nil {
			return fmt.Errorf("invalid brioi profiles: %w", err)
		}
	case "video_tool_groups":
		if err := common.UnmarshalJsonStr(value, &clone.VideoToolGroups); err != nil {
			return fmt.Errorf("invalid brioi video tool groups: %w", err)
		}
	default:
		return fmt.Errorf("unsupported brioi_setting option key %q", configKey)
	}
	if err := brioi_setting.ValidateBrioiSetting(&clone); err != nil {
		return err
	}
	return nil
}

func validateVideoSettingOption(key, value string) error {
	configKey := strings.TrimPrefix(key, "video_setting.")
	if configKey == key || configKey == "" {
		return fmt.Errorf("invalid video_setting option key")
	}

	current := setting.GetEffectiveVideoSetting().VideoSetting
	raw, err := common.Marshal(&current)
	if err != nil {
		return err
	}
	var clone video_setting.VideoSetting
	if err := common.Unmarshal(raw, &clone); err != nil {
		return err
	}
	switch configKey {
	case "enabled":
		enabled, err := strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			return err
		}
		clone.Enabled = enabled
	case "storage":
		if err := common.UnmarshalJsonStr(value, &clone.Storage); err != nil {
			return fmt.Errorf("invalid video storage setting: %w", err)
		}
	case "upload_limits":
		if err := common.UnmarshalJsonStr(value, &clone.UploadLimits); err != nil {
			return fmt.Errorf("invalid video upload limits: %w", err)
		}
	case "video_tool_groups":
		if err := common.UnmarshalJsonStr(value, &clone.VideoToolGroups); err != nil {
			return fmt.Errorf("invalid video tool groups: %w", err)
		}
	default:
		return fmt.Errorf("unsupported video_setting option key %q", configKey)
	}
	switch configKey {
	case "enabled":
		return nil
	case "storage":
		return video_setting.ValidateVideoStorageSetting(&clone.Storage)
	case "upload_limits":
		return video_setting.ValidateUploadLimitsSetting(&clone.UploadLimits)
	case "video_tool_groups":
		clone.VideoToolGroups = video_setting.NormalizeVideoToolGroups(clone.VideoToolGroups)
		return nil
	default:
		return fmt.Errorf("unsupported video_setting option key %q", configKey)
	}
}

// perSecondBindingWarning checks the billing_setting.billing_mode JSON map and
// warns about models marked per_second that do not match any video-provider
// profile. Such models never reach an adaptor that supplies the seconds
// multiplier, so they would be charged the unit price only once per call.
// The save still proceeds; the warning is surfaced to administrators.
func perSecondBindingWarning(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	var modes map[string]string
	if err := common.UnmarshalJsonStr(value, &modes); err != nil {
		return "", fmt.Errorf("billing_mode 必须是合法的 JSON 对象: %w", err)
	}
	var fallbackModels []string
	for modelName, mode := range modes {
		if mode != billing_setting.BillingModePerSecond {
			continue
		}
		silkRoadResolution, silkRoadOK := silkroad_setting.ResolveProfile(modelName)
		if silkRoadOK && silkRoadResolution.MatchKind != silkroad_setting.ProfileMatchDefault {
			continue
		}
		if _, brioiOK := brioi_setting.ResolveProfile(modelName); brioiOK {
			continue
		}
		fallbackModels = append(fallbackModels, modelName)
	}
	if len(fallbackModels) == 0 {
		return "", nil
	}
	sort.Strings(fallbackModels)
	return fmt.Sprintf(
		"提示：以下按秒计费模型未命中 SilkRoad 精确模型或模型前缀，将使用管理员选择的默认档案；请确认默认档案的时长和能力配置适用：%s",
		strings.Join(fallbackModels, ", "),
	), nil
}

func collectModelNamesFromOptionValue(raw string, modelNames map[string]struct{}) {
	if strings.TrimSpace(raw) == "" {
		return
	}

	var parsed map[string]any
	if err := common.UnmarshalJsonStr(raw, &parsed); err != nil {
		return
	}

	for modelName := range parsed {
		modelNames[modelName] = struct{}{}
	}
}

func buildCompletionRatioMetaValue(optionValues map[string]string) string {
	modelNames := make(map[string]struct{})
	for _, key := range completionRatioMetaOptionKeys {
		collectModelNamesFromOptionValue(optionValues[key], modelNames)
	}

	meta := make(map[string]ratio_setting.CompletionRatioInfo, len(modelNames))
	for modelName := range modelNames {
		meta[modelName] = ratio_setting.GetCompletionRatioInfo(modelName)
	}

	jsonBytes, err := common.Marshal(meta)
	if err != nil {
		return "{}"
	}
	return string(jsonBytes)
}

func GetOptions(c *gin.Context) {
	var options []*model.Option
	optionValues := make(map[string]string)
	effectiveVideo := setting.GetEffectiveVideoSetting()
	effectiveVideoGroups, _ := common.Marshal(effectiveVideo.VideoToolGroups)
	effectiveVideoStorage, _ := common.Marshal(effectiveVideo.Storage)
	effectiveVideoUploadLimits, _ := common.Marshal(effectiveVideo.UploadLimits)
	common.OptionMapRWMutex.Lock()
	for k, v := range common.OptionMap {
		if k == "theme.frontend" {
			continue
		}
		value := common.Interface2String(v)
		switch {
		case k == "video_setting.enabled" &&
			!config.GlobalConfig.IsExplicit(k):
			value = strconv.FormatBool(effectiveVideo.Enabled)
		case k == "video_setting.video_tool_groups" &&
			!config.GlobalConfig.IsExplicit(k):
			value = string(effectiveVideoGroups)
		case k == "video_setting.storage" &&
			!config.GlobalConfig.IsExplicit(k):
			value = string(effectiveVideoStorage)
		case k == "video_setting.upload_limits" &&
			!config.GlobalConfig.IsExplicit(k):
			value = string(effectiveVideoUploadLimits)
		}
		// Turnstile Site Key is public (embedded in the frontend widget).
		if k == "TurnstileSiteKey" {
			options = append(options, &model.Option{Key: k, Value: value})
			continue
		}
		// Never return the raw Turnstile secret; expose a mask so the admin UI
		// can show that a secret is already configured.
		if k == "TurnstileSecretKey" {
			masked := ""
			if value != "" {
				masked = "********"
			}
			options = append(options, &model.Option{Key: k, Value: masked})
			continue
		}
		isSensitiveKey := strings.HasSuffix(k, "Token") ||
			strings.HasSuffix(k, "Secret") ||
			strings.HasSuffix(k, "Key") ||
			strings.HasSuffix(k, "secret") ||
			strings.HasSuffix(k, "api_key")
		if isSensitiveKey {
			continue
		}
		options = append(options, &model.Option{
			Key:   k,
			Value: value,
		})
		for _, optionKey := range completionRatioMetaOptionKeys {
			if optionKey == k {
				optionValues[k] = value
				break
			}
		}
	}
	common.OptionMapRWMutex.Unlock()
	options = append(options, &model.Option{
		Key:   "CompletionRatioMeta",
		Value: buildCompletionRatioMetaValue(optionValues),
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    options,
	})
}

type OptionUpdateRequest struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

type VideoProviderOptionUpdateRequest struct {
	Provider         setting.VideoProvider           `json:"provider"`
	Common           *silkroad_setting.CommonSetting `json:"common,omitempty"`
	Profiles         json.RawMessage                 `json:"profiles"`
	DefaultProfileID string                          `json:"default_profile_id,omitempty"`
}

func UpdateVideoProviderOption(c *gin.Context) {
	var request VideoProviderOptionUpdateRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorMsg(c, "invalid video provider settings")
		return
	}

	videoProviderGroupUpdateMu.Lock()
	defer videoProviderGroupUpdateMu.Unlock()

	values, err := videoProviderOptionValues(request)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if err := model.UpdateOptionsBulk(values); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "option.video_provider.update", map[string]interface{}{
		"provider": request.Provider,
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

func videoProviderOptionValues(request VideoProviderOptionUpdateRequest) (map[string]string, error) {
	if len(request.Profiles) == 0 {
		// Compatible Video may save an empty override list (all built-in defaults).
		if request.Provider != setting.VideoProviderCompatVideo {
			return nil, fmt.Errorf("video provider profiles are required")
		}
	}

	switch request.Provider {
	case setting.VideoProviderSilkRoad:
		if request.Common == nil {
			return nil, fmt.Errorf("SilkRoad common settings are required")
		}
		var profiles []silkroad_setting.Profile
		if err := common.Unmarshal(request.Profiles, &profiles); err != nil {
			return nil, fmt.Errorf("invalid SilkRoad profiles: %w", err)
		}
		candidate := silkroad_setting.SilkRoadSetting{
			Common:           *request.Common,
			Profiles:         profiles,
			DefaultProfileID: request.DefaultProfileID,
			Storage:          silkroad_setting.GetSilkRoadSetting().Storage,
		}
		if err := silkroad_setting.ValidateSilkRoadProviderSetting(&candidate); err != nil {
			return nil, err
		}
		commonValue, err := common.Marshal(candidate.Common)
		if err != nil {
			return nil, err
		}
		profilesValue, err := common.Marshal(candidate.Profiles)
		if err != nil {
			return nil, err
		}
		return map[string]string{
			"silkroad_setting.common":             string(commonValue),
			"silkroad_setting.profiles":           string(profilesValue),
			"silkroad_setting.default_profile_id": candidate.DefaultProfileID,
		}, nil
	case setting.VideoProviderBrioi:
		var profiles []brioi_setting.Profile
		if err := common.Unmarshal(request.Profiles, &profiles); err != nil {
			return nil, fmt.Errorf("invalid Brioi profiles: %w", err)
		}
		candidate := brioi_setting.BrioiSetting{
			Profiles: profiles,
		}
		if err := brioi_setting.ValidateBrioiSetting(&candidate); err != nil {
			return nil, err
		}
		profilesValue, err := common.Marshal(candidate.Profiles)
		if err != nil {
			return nil, err
		}
		return map[string]string{
			"brioi_setting.profiles": string(profilesValue),
		}, nil
	case setting.VideoProviderCompatVideo:
		var profiles []compatvideo_setting.Profile
		if len(request.Profiles) > 0 {
			if err := common.Unmarshal(request.Profiles, &profiles); err != nil {
				return nil, fmt.Errorf("invalid Compatible Video profiles: %w", err)
			}
		}
		candidate := compatvideo_setting.CompatVideoSetting{
			Profiles: profiles,
		}
		if err := compatvideo_setting.ValidateCompatVideoSetting(&candidate); err != nil {
			return nil, err
		}
		profilesValue, err := common.Marshal(profiles)
		if err != nil {
			return nil, err
		}
		return map[string]string{
			"compatvideo_setting.profiles": string(profilesValue),
		}, nil

	default:
		return nil, fmt.Errorf("unsupported video provider %q", request.Provider)
	}
}

func UpdateOption(c *gin.Context) {
	var option OptionUpdateRequest
	err := common.DecodeJson(c.Request.Body, &option)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的参数",
		})
		return
	}
	switch option.Value.(type) {
	case bool:
		option.Value = common.Interface2String(option.Value.(bool))
	case float64:
		option.Value = common.Interface2String(option.Value.(float64))
	case int:
		option.Value = common.Interface2String(option.Value.(int))
	default:
		option.Value = fmt.Sprintf("%v", option.Value)
	}
	// Masked / empty secret means "keep existing" — never wipe TurnstileSecretKey
	if option.Key == "TurnstileSecretKey" {
		secret := option.Value.(string)
		if secret == "" || secret == "********" {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "",
			})
			return
		}
	}
	switch option.Key {
	case "QuotaForInviter", "QuotaForInvitee":
		if isPositiveOptionValue(option.Value.(string)) && !operation_setting.IsPaymentComplianceConfirmed() {
			common.ApiErrorI18n(c, i18n.MsgPaymentComplianceRequired)
			return
		}
	default:
		if isPaymentComplianceOptionKey(option.Key) {
			common.ApiErrorMsg(c, "合规确认字段不允许通过通用设置接口修改")
			return
		}
	}
	saveWarning := ""
	switch option.Key {
	case "video_setting.video_tool_groups",
		"silkroad_setting.video_tool_groups",
		"brioi_setting.video_tool_groups":
		videoProviderGroupUpdateMu.Lock()
		defer videoProviderGroupUpdateMu.Unlock()
	}
	switch option.Key {
	case "GitHubOAuthEnabled":
		if option.Value == "true" && common.GitHubClientId == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 GitHub OAuth，请先填入 GitHub Client Id 以及 GitHub Client Secret！",
			})
			return
		}
	case "discord.enabled":
		if option.Value == "true" && system_setting.GetDiscordSettings().ClientId == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 Discord OAuth，请先填入 Discord Client Id 以及 Discord Client Secret！",
			})
			return
		}
	case "oidc.enabled":
		if option.Value == "true" && system_setting.GetOIDCSettings().ClientId == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 OIDC 登录，请先填入 OIDC Client Id 以及 OIDC Client Secret！",
			})
			return
		}
	case "LinuxDOOAuthEnabled":
		if option.Value == "true" && common.LinuxDOClientId == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 LinuxDO OAuth，请先填入 LinuxDO Client Id 以及 LinuxDO Client Secret！",
			})
			return
		}
	case "EmailDomainRestrictionEnabled":
		if option.Value == "true" && len(common.EmailDomainWhitelist) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用邮箱域名限制，请先填入限制的邮箱域名！",
			})
			return
		}
	case "WeChatAuthEnabled":
		if option.Value == "true" && common.WeChatServerAddress == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用微信登录，请先填入微信登录相关配置信息！",
			})
			return
		}
	case "TurnstileCheckEnabled":
		if option.Value == "true" && common.TurnstileSiteKey == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 Turnstile 校验，请先填入 Turnstile 校验相关配置信息！",
			})

			return
		}
	case "TelegramOAuthEnabled":
		if option.Value == "true" && common.TelegramBotToken == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 Telegram OAuth，请先填入 Telegram Bot Token！",
			})
			return
		}
	case "theme.frontend":
		if option.Value != "default" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "Classic 前端已移除，主题只能设置为 default",
			})
			return
		}
	case "GroupRatio":
		err = ratio_setting.CheckGroupRatio(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "gemini.safety_settings":
		err = model_setting.ValidateGeminiSafetySettings(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "claude.default_max_tokens":
		err = model_setting.ValidateClaudeDefaultMaxTokens(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case operation_setting.ToolPriceOptionKey:
		err = operation_setting.ValidateToolPricesJSON(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "ImageRatio":
		err = ratio_setting.UpdateImageRatioByJSONString(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "图片倍率设置失败: " + err.Error(),
			})
			return
		}
	case "AudioRatio":
		err = ratio_setting.UpdateAudioRatioByJSONString(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "音频倍率设置失败: " + err.Error(),
			})
			return
		}
	case "AudioCompletionRatio":
		err = ratio_setting.UpdateAudioCompletionRatioByJSONString(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "音频补全倍率设置失败: " + err.Error(),
			})
			return
		}
	case "CreateCacheRatio":
		err = ratio_setting.UpdateCreateCacheRatioByJSONString(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "缓存创建倍率设置失败: " + err.Error(),
			})
			return
		}
	case "ModelRequestRateLimitGroup":
		err = setting.CheckModelRequestRateLimitGroup(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "AutomaticDisableStatusCodes":
		_, err = operation_setting.ParseHTTPStatusCodeRanges(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "AutomaticRetryStatusCodes":
		_, err = operation_setting.ParseHTTPStatusCodeRanges(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "console_setting.api_info":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "ApiInfo")
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "console_setting.announcements":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "Announcements")
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "console_setting.faq":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "FAQ")
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "console_setting.custom_pages":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "CustomPages")
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "console_setting.availability_monitor_visibility":
		err = console_setting.ValidateAvailabilityMonitorVisibility(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "console_setting.availability_monitor_refresh_interval":
		err = console_setting.ValidateAvailabilityMonitorRefreshInterval(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "console_setting.uptime_kuma_groups":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "UptimeKumaGroups")
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "silkroad_setting.common", "silkroad_setting.profiles", "silkroad_setting.default_profile_id",
		"silkroad_setting.storage", "silkroad_setting.video_tool_groups":
		err = validateSilkRoadSettingOption(option.Key, option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "brioi_setting.profiles", "brioi_setting.video_tool_groups":
		err = validateBrioiSettingOption(option.Key, option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "billing_setting." + billing_setting.BillingModeField:
		saveWarning, err = perSecondBindingWarning(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "video_setting.enabled", "video_setting.storage", "video_setting.upload_limits", "video_setting.video_tool_groups":
		err = validateVideoSettingOption(option.Key, option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	}
	err = model.UpdateOption(option.Key, option.Value.(string))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	// 出于安全考虑只记录被修改的配置项名称，不记录配置值（可能含密钥等敏感信息）。
	recordManageAudit(c, "option.update", map[string]interface{}{
		"key": option.Key,
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": saveWarning,
	})
}
