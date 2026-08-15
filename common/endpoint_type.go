package common

import (
	"strings"

	"github.com/QuantumNous/new-api/constant"
)

// GetEndpointTypesByChannelType 获取渠道最优先端点类型（所有的渠道都支持 OpenAI 端点）
func GetEndpointTypesByChannelType(channelType int, modelName string) []constant.EndpointType {
	var endpointTypes []constant.EndpointType
	switch channelType {
	case constant.ChannelTypeJina:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeJinaRerank}
	//case constant.ChannelTypeMidjourney, constant.ChannelTypeMidjourneyPlus:
	//	endpointTypes = []constant.EndpointType{constant.EndpointTypeMidjourney}
	//case constant.ChannelTypeSunoAPI:
	//	endpointTypes = []constant.EndpointType{constant.EndpointTypeSuno}
	//case constant.ChannelTypeKling:
	//	endpointTypes = []constant.EndpointType{constant.EndpointTypeKling}
	//case constant.ChannelTypeJimeng:
	//	endpointTypes = []constant.EndpointType{constant.EndpointTypeJimeng}
	case constant.ChannelTypeAws:
		fallthrough
	case constant.ChannelTypeAnthropic:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeAnthropic, constant.EndpointTypeOpenAI}
	case constant.ChannelTypeVertexAi:
		fallthrough
	case constant.ChannelTypeGemini:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeGemini, constant.EndpointTypeOpenAI}
	case constant.ChannelTypeOpenRouter: // OpenRouter 只支持 OpenAI 端点
		endpointTypes = []constant.EndpointType{constant.EndpointTypeOpenAI}
	case constant.ChannelTypeXai:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeOpenAI, constant.EndpointTypeOpenAIResponse}
	case constant.ChannelTypeSora:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeOpenAIVideo}
	case constant.ChannelTypeSilkRoad, constant.ChannelTypeBrioi, constant.ChannelTypeCompatVideo:
		return []constant.EndpointType{constant.EndpointTypeOpenAIVideo}
	case constant.ChannelTypeSub2API, constant.ChannelTypeNewAPI:
		endpointTypes = []constant.EndpointType{
			constant.EndpointTypeOpenAI,
			constant.EndpointTypeOpenAIResponse,
			constant.EndpointTypeOpenAIResponseCompact,
			constant.EndpointTypeAnthropic,
			constant.EndpointTypeGemini,
			constant.EndpointTypeOpenAIAlphaSearch,
		}
	case constant.ChannelTypeCodex:
		endpointTypes = []constant.EndpointType{
			constant.EndpointTypeOpenAIResponse,
			constant.EndpointTypeOpenAIResponseCompact,
			constant.EndpointTypeOpenAIAlphaSearch,
		}
	default:
		if IsOpenAIResponseOnlyModel(modelName) {
			endpointTypes = []constant.EndpointType{constant.EndpointTypeOpenAIResponse}
		} else {
			endpointTypes = []constant.EndpointType{constant.EndpointTypeOpenAI}
		}
	}
	if IsImageGenerationModel(modelName) {
		// add to first
		endpointTypes = append([]constant.EndpointType{constant.EndpointTypeImageGeneration}, endpointTypes...)
	}
	return endpointTypes
}

// IsOpenAIVideoRequestPath recognizes the two public video API families.
func IsOpenAIVideoRequestPath(requestPath string) bool {
	path := strings.TrimSpace(requestPath)
	return path == "/v1/videos" ||
		strings.HasPrefix(path, "/v1/videos/") ||
		IsVideoGenerationRequestPath(path)
}

// IsVideoGenerationRequestPath recognizes the provider-neutral video tool API.
func IsVideoGenerationRequestPath(requestPath string) bool {
	path := strings.TrimSpace(requestPath)
	return path == "/v1/video/generations" ||
		strings.HasPrefix(path, "/v1/video/generations/")
}

// IsVideoTaskRequestPath recognizes every asynchronous video submission route
// that must fail closed when mandatory local result storage is unavailable.
func IsVideoTaskRequestPath(requestPath string) bool {
	path := strings.TrimSpace(requestPath)
	return IsOpenAIVideoRequestPath(path) ||
		strings.HasPrefix(path, "/kling/v1/videos/") ||
		path == "/jimeng" ||
		path == "/jimeng/"
}

// ChannelTypeSupportsRequestPath keeps dedicated video-only channels out of
// non-video routing and prevents legacy NewAPI channels receiving new videos.
func ChannelTypeSupportsRequestPath(channelType int, requestPath string) bool {
	if requestPath == "" {
		return true
	}
	isVideo := IsOpenAIVideoRequestPath(requestPath)
	switch channelType {
	case constant.ChannelTypeSilkRoad, constant.ChannelTypeCompatVideo:
		return isVideo
	case constant.ChannelTypeBrioi:
		return IsVideoGenerationRequestPath(requestPath)
	case constant.ChannelTypeNewAPI:
		return !isVideo
	default:
		return true
	}
}
