package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
)

type videoResultSource struct {
	Body        io.ReadCloser
	ContentType string
}

func openTaskVideoResultSource(ctx context.Context, task *model.Task) (videoResultSource, error) {
	if task == nil {
		return videoResultSource{}, fmt.Errorf("nil video task")
	}
	source := strings.TrimSpace(task.PrivateData.UpstreamResultURL)
	if strings.HasPrefix(source, "data:") {
		body, contentType, err := openVideoDataURL(source)
		return videoResultSource{Body: body, ContentType: contentType}, err
	}

	channel, channelErr := model.CacheGetChannel(task.ChannelId)
	authenticatedEndpoint := false
	if source == "" && channelErr == nil && channel != nil {
		switch channel.Type {
		case constant.ChannelTypeOpenAI, constant.ChannelTypeSora:
			baseURL := strings.TrimRight(channel.GetBaseURL(), "/")
			if baseURL == "" {
				baseURL = "https://api.openai.com"
			}
			source = fmt.Sprintf(
				"%s/v1/videos/%s/content",
				baseURL,
				task.GetUpstreamTaskID(),
			)
			authenticatedEndpoint = true
		}
	}
	if source == "" {
		return videoResultSource{}, fmt.Errorf("missing upstream video result")
	}
	if err := ValidateSSRFProtectedFetchURL(source); err != nil {
		return videoResultSource{}, fmt.Errorf("upstream video result blocked by fetch policy")
	}

	client := GetSSRFProtectedHTTPClient()
	if channelErr == nil &&
		channel != nil &&
		canUseChannelProxyForVideoResult(source, channel, authenticatedEndpoint) {
		if proxy := strings.TrimSpace(channel.GetSetting().Proxy); proxy != "" {
			var err error
			client, err = GetHttpClientWithProxy(proxy)
			if err != nil {
				return videoResultSource{}, fmt.Errorf("create video result proxy client: %w", err)
			}
		}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return videoResultSource{}, err
	}
	if authenticatedEndpoint && channel != nil {
		request.Header.Set("Authorization", "Bearer "+channel.Key)
	}
	if channel != nil && !authenticatedEndpoint && sourceMatchesChannelBase(source, channel.GetBaseURL()) {
		request.Header.Set("Authorization", "Bearer "+channel.Key)
	}
	if channel != nil && channel.Type == constant.ChannelTypeGemini &&
		(sourceMatchesChannelBase(source, channel.GetBaseURL()) || isGoogleAPIURL(source)) {
		key := task.PrivateData.Key
		if key == "" {
			key = channel.Key
		}
		request.Header.Set("x-goog-api-key", key)
	}

	response, err := client.Do(request)
	if err != nil {
		return videoResultSource{}, fmt.Errorf("download upstream video result: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_ = response.Body.Close()
		return videoResultSource{}, fmt.Errorf(
			"upstream video result returned status %d",
			response.StatusCode,
		)
	}
	return videoResultSource{
		Body:        response.Body,
		ContentType: response.Header.Get("Content-Type"),
	}, nil
}

func canUseChannelProxyForVideoResult(
	source string,
	channel *model.Channel,
	authenticatedEndpoint bool,
) bool {
	if channel == nil {
		return false
	}
	return authenticatedEndpoint ||
		sourceMatchesChannelBase(source, channel.GetBaseURL()) ||
		(channel.Type == constant.ChannelTypeGemini && isGoogleAPIURL(source))
}

func sourceMatchesChannelBase(source, base string) bool {
	sourceURL, sourceErr := url.Parse(source)
	baseURL, baseErr := url.Parse(base)
	if sourceErr != nil || baseErr != nil || sourceURL.Host == "" || baseURL.Host == "" {
		return false
	}
	return strings.EqualFold(sourceURL.Scheme, baseURL.Scheme) &&
		strings.EqualFold(sourceURL.Host, baseURL.Host)
}

func isGoogleAPIURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "googleapis.com" || strings.HasSuffix(host, ".googleapis.com")
}
