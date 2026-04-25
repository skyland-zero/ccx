package images

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/BenedictKing/ccx/internal/handlers/common"
	"github.com/BenedictKing/ccx/internal/middleware"
	"github.com/BenedictKing/ccx/internal/scheduler"
	"github.com/BenedictKing/ccx/internal/types"
	"github.com/BenedictKing/ccx/internal/utils"
	"github.com/gin-gonic/gin"
)

// Handler Images API 代理处理器
func Handler(
	envCfg *config.EnvConfig,
	cfgManager *config.ConfigManager,
	channelScheduler *scheduler.ChannelScheduler,
) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		middleware.ProxyAuthMiddleware(envCfg)(c)
		if c.IsAborted() {
			return
		}

		startTime := time.Now()
		maxBodySize := envCfg.MaxRequestBodySize
		bodyBytes, err := common.ReadRequestBody(c, maxBodySize)
		if err != nil {
			return
		}
		c.Set("requestBodyBytes", bodyBytes)

		var reqMap map[string]interface{}
		if len(bodyBytes) > 0 {
			if err := json.Unmarshal(bodyBytes, &reqMap); err != nil {
				imagesErrorResponse(c, 400, fmt.Sprintf("Invalid request body: %v", err), "invalid_request_error", "invalid_json")
				return
			}
		}

		model, _ := reqMap["model"].(string)
		if model == "" {
			imagesErrorResponse(c, 400, "model is required", "invalid_request_error", "missing_parameter")
			return
		}
		prompt, _ := reqMap["prompt"].(string)
		if prompt == "" {
			imagesErrorResponse(c, 400, "prompt is required", "invalid_request_error", "missing_parameter")
			return
		}

		userID := utils.ExtractUnifiedSessionID(c, bodyBytes)
		common.LogOriginalRequest(c, bodyBytes, envCfg, "Images")

		if channelScheduler.IsMultiChannelMode(scheduler.ChannelKindImages) {
			handleMultiChannel(c, envCfg, cfgManager, channelScheduler, bodyBytes, model, userID, startTime)
		} else {
			handleSingleChannel(c, envCfg, cfgManager, channelScheduler, bodyBytes, model, startTime)
		}
	})
}

func handleMultiChannel(
	c *gin.Context,
	envCfg *config.EnvConfig,
	cfgManager *config.ConfigManager,
	channelScheduler *scheduler.ChannelScheduler,
	bodyBytes []byte,
	model string,
	userID string,
	startTime time.Time,
) {
	metricsManager := channelScheduler.GetImagesMetricsManager()
	common.HandleMultiChannelFailover(
		c,
		envCfg,
		channelScheduler,
		scheduler.ChannelKindImages,
		"Images",
		userID,
		model,
		func(selection *scheduler.SelectionResult) common.MultiChannelAttemptResult {
			upstream := selection.Upstream
			channelIndex := selection.ChannelIndex
			if upstream == nil {
				return common.MultiChannelAttemptResult{}
			}

			baseURLs := upstream.GetAllBaseURLs()
			sortedURLResults := channelScheduler.GetSortedURLsForChannel(scheduler.ChannelKindImages, channelIndex, baseURLs)
			handled, successKey, successBaseURLIdx, failoverErr, usage, lastErr := common.TryUpstreamWithAllKeys(
				c,
				envCfg,
				cfgManager,
				channelScheduler,
				scheduler.ChannelKindImages,
				"Images",
				metricsManager,
				upstream,
				sortedURLResults,
				bodyBytes,
				false,
				func(upstream *config.UpstreamConfig, failedKeys map[string]bool) (string, error) {
					return cfgManager.GetNextImagesAPIKey(upstream, failedKeys)
				},
				func(c *gin.Context, upstreamCopy *config.UpstreamConfig, apiKey string) (*http.Request, error) {
					return buildProviderRequest(c, upstreamCopy, upstreamCopy.BaseURL, apiKey, bodyBytes, model)
				},
				func(apiKey string) {
					_ = cfgManager.DeprioritizeAPIKey(apiKey)
				},
				func(url string) {
					channelScheduler.MarkURLFailure(scheduler.ChannelKindImages, channelIndex, url)
				},
				func(url string) {
					channelScheduler.MarkURLSuccess(scheduler.ChannelKindImages, channelIndex, url)
				},
				func(c *gin.Context, resp *http.Response, upstreamCopy *config.UpstreamConfig, apiKey string, actualRequestBody []byte) (*types.Usage, error) {
					return handleSuccess(c, resp, startTime)
				},
				model,
				selection.ChannelIndex,
				channelScheduler.GetChannelLogStore(scheduler.ChannelKindImages),
			)

			return common.MultiChannelAttemptResult{
				Handled:           handled,
				Attempted:         true,
				SuccessKey:        successKey,
				SuccessBaseURLIdx: successBaseURLIdx,
				FailoverError:     failoverErr,
				Usage:             usage,
				LastError:         lastErr,
			}
		},
		nil,
		func(ctx *gin.Context, failoverErr *common.FailoverError, lastError error) {
			handleAllChannelsFailed(ctx, failoverErr, lastError)
		},
	)
}

func handleSingleChannel(
	c *gin.Context,
	envCfg *config.EnvConfig,
	cfgManager *config.ConfigManager,
	channelScheduler *scheduler.ChannelScheduler,
	bodyBytes []byte,
	model string,
	startTime time.Time,
) {
	upstream, channelIndex, err := cfgManager.GetCurrentImagesUpstreamWithIndex()
	if err != nil {
		imagesErrorResponse(c, 503, "No Images upstream configured", "service_unavailable", "service_unavailable")
		return
	}
	if len(upstream.APIKeys) == 0 {
		imagesErrorResponse(c, 503, fmt.Sprintf("No API keys configured for upstream \"%s\"", upstream.Name), "service_unavailable", "service_unavailable")
		return
	}

	metricsManager := channelScheduler.GetImagesMetricsManager()
	baseURLs := upstream.GetAllBaseURLs()
	urlResults := common.BuildDefaultURLResults(baseURLs)
	handled, _, _, lastFailoverError, _, lastError := common.TryUpstreamWithAllKeys(
		c,
		envCfg,
		cfgManager,
		channelScheduler,
		scheduler.ChannelKindImages,
		"Images",
		metricsManager,
		upstream,
		urlResults,
		bodyBytes,
		false,
		func(upstream *config.UpstreamConfig, failedKeys map[string]bool) (string, error) {
			return cfgManager.GetNextImagesAPIKey(upstream, failedKeys)
		},
		func(c *gin.Context, upstreamCopy *config.UpstreamConfig, apiKey string) (*http.Request, error) {
			return buildProviderRequest(c, upstreamCopy, upstreamCopy.BaseURL, apiKey, bodyBytes, model)
		},
		func(apiKey string) {
			_ = cfgManager.DeprioritizeAPIKey(apiKey)
		},
		nil,
		nil,
		func(c *gin.Context, resp *http.Response, upstreamCopy *config.UpstreamConfig, apiKey string, actualRequestBody []byte) (*types.Usage, error) {
			return handleSuccess(c, resp, startTime)
		},
		model,
		channelIndex,
		channelScheduler.GetChannelLogStore(scheduler.ChannelKindImages),
	)
	if handled {
		return
	}

	log.Printf("[Images-Error] 所有 API密钥都失败了")
	handleAllKeysFailed(c, lastFailoverError, lastError)
}

func buildProviderRequest(
	c *gin.Context,
	upstream *config.UpstreamConfig,
	baseURL string,
	apiKey string,
	bodyBytes []byte,
	model string,
) (*http.Request, error) {
	serviceType, err := config.NormalizeImagesServiceTypeForProxy(upstream.ServiceType)
	if err != nil {
		return nil, err
	}
	upstream.ServiceType = serviceType
	skipVersionPrefix := strings.HasSuffix(baseURL, "#")
	baseURL = strings.TrimSuffix(strings.TrimRight(baseURL, "/"), "#")

	var reqMap map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &reqMap); err != nil {
		return nil, err
	}
	reqMap["model"] = config.RedirectModel(model, upstream)
	requestBody, err := json.Marshal(reqMap)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/v1/images/generations", strings.TrimRight(baseURL, "/"))
	if skipVersionPrefix {
		url = fmt.Sprintf("%s/images/generations", strings.TrimRight(baseURL, "/"))
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, err
	}
	req.Header = utils.PrepareUpstreamHeaders(c, req.URL.Host)
	req.Header.Set("Content-Type", "application/json")
	utils.SetAuthenticationHeader(req.Header, apiKey)
	utils.ApplyCustomHeaders(req.Header, upstream.CustomHeaders)
	return req, nil
}

func handleSuccess(c *gin.Context, resp *http.Response, startTime time.Time) (*types.Usage, error) {
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		imagesErrorResponse(c, 500, "Failed to read response", "server_error", "server_error")
		return nil, err
	}
	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	var respMap map[string]interface{}
	if err := common.PassthroughJSONResponse(c, resp, &respMap); err != nil {
		return nil, nil
	}
	if u, ok := respMap["usage"].(map[string]interface{}); ok {
		inputTokens, _ := u["input_tokens"].(float64)
		outputTokens, _ := u["output_tokens"].(float64)
		return &types.Usage{InputTokens: int(inputTokens), OutputTokens: int(outputTokens)}, nil
	}
	_ = startTime
	return nil, nil
}

func imagesErrorResponse(c *gin.Context, statusCode int, message, errorType, code string) {
	c.JSON(statusCode, gin.H{
		"error": gin.H{
			"message": message,
			"type":    errorType,
			"code":    code,
		},
	})
}

func handleAllChannelsFailed(c *gin.Context, failoverErr *common.FailoverError, lastError error) {
	if failoverErr != nil {
		c.Data(failoverErr.Status, "application/json", failoverErr.Body)
		return
	}
	errMsg := "All channels failed"
	if lastError != nil {
		errMsg = lastError.Error()
	}
	imagesErrorResponse(c, 503, errMsg, "service_unavailable", "service_unavailable")
}

func handleAllKeysFailed(c *gin.Context, failoverErr *common.FailoverError, lastError error) {
	if failoverErr != nil {
		c.Data(failoverErr.Status, "application/json", failoverErr.Body)
		return
	}
	errMsg := "All API keys failed"
	if lastError != nil {
		errMsg = lastError.Error()
	}
	imagesErrorResponse(c, 503, errMsg, "service_unavailable", "service_unavailable")
}
