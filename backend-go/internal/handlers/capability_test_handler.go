package handlers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/BenedictKing/ccx/internal/httpclient"
	"github.com/BenedictKing/ccx/internal/utils"
	"github.com/gin-gonic/gin"
)

// ============== 类型定义 ==============

// CapabilityTestRequest 能力测试请求体
type CapabilityTestRequest struct {
	TargetProtocols []string `json:"targetProtocols"`
	Timeout         int      `json:"timeout"` // 毫秒
}

// ProtocolTestResult 单个协议测试结果
type ProtocolTestResult struct {
	Protocol           string  `json:"protocol"`
	Success            bool    `json:"success"`
	Latency            int64   `json:"latency"` // 毫秒
	StreamingSupported bool    `json:"streamingSupported"`
	TestedModel        string  `json:"testedModel"` // 测试成功的模型名称
	Error              *string `json:"error"`
	TestedAt           string  `json:"testedAt"`
}

// CapabilityTestResponse 能力测试响应体
type CapabilityTestResponse struct {
	ChannelID           int                  `json:"channelId"`
	ChannelName         string               `json:"channelName"`
	SourceType          string               `json:"sourceType"`
	Tests               []ProtocolTestResult `json:"tests"`
	CompatibleProtocols []string             `json:"compatibleProtocols"`
	TotalDuration       int64                `json:"totalDuration"` // 毫秒
}

// ============== 主处理器 ==============

// TestChannelCapability 渠道能力测试处理器
// channelKind 决定从哪个配置获取渠道：messages/responses/gemini/chat
func TestChannelCapability(cfgManager *config.ConfigManager, channelKind string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 解析渠道 ID
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
			return
		}

		// 获取配置并定位渠道
		cfg := cfgManager.GetConfig()
		var channels []config.UpstreamConfig
		switch channelKind {
		case "messages":
			channels = cfg.Upstream
		case "responses":
			channels = cfg.ResponsesUpstream
		case "gemini":
			channels = cfg.GeminiUpstream
		case "chat":
			channels = cfg.ChatUpstream
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel kind"})
			return
		}

		if id < 0 || id >= len(channels) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
			return
		}

		channel := channels[id]

		// 检查 API Key
		if len(channel.APIKeys) == 0 {
			errMsg := "no_api_key"
			c.JSON(http.StatusOK, CapabilityTestResponse{
				ChannelID:           id,
				ChannelName:         channel.Name,
				SourceType:          channel.ServiceType,
				Tests:               []ProtocolTestResult{{Protocol: "all", Error: &errMsg, TestedAt: time.Now().Format(time.RFC3339)}},
				CompatibleProtocols: []string{},
				TotalDuration:       0,
			})
			return
		}

		// 解析请求体
		var req CapabilityTestRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		// 默认超时 15 秒
		timeout := 15 * time.Second
		if req.Timeout > 0 {
			timeout = time.Duration(req.Timeout) * time.Millisecond
		}

		// 默认测试所有协议
		protocols := req.TargetProtocols
		if len(protocols) == 0 {
			protocols = []string{"messages", "chat", "gemini", "responses"}
		}

		log.Printf("[CapabilityTest] 开始测试渠道 %s (ID:%d, 类型:%s) 的协议兼容性: %v", channel.Name, id, channel.ServiceType, protocols)

		// 并发测试
		totalStart := time.Now()
		results := testProtocolCompatibility(c.Request.Context(), &channel, protocols, timeout)
		totalDuration := time.Since(totalStart).Milliseconds()

		// 收集兼容协议
		compatible := []string{}
		for _, r := range results {
			if r.Success {
				compatible = append(compatible, r.Protocol)
			}
		}

		log.Printf("[CapabilityTest] 渠道 %s 测试完成，兼容协议: %v，耗时: %dms", channel.Name, compatible, totalDuration)

		c.JSON(http.StatusOK, CapabilityTestResponse{
			ChannelID:           id,
			ChannelName:         channel.Name,
			SourceType:          channel.ServiceType,
			Tests:               results,
			CompatibleProtocols: compatible,
			TotalDuration:       totalDuration,
		})
	}
}

// ============== 核心测试逻辑 ==============

// testProtocolCompatibility 并发测试多个协议的兼容性
func testProtocolCompatibility(ctx context.Context, channel *config.UpstreamConfig, protocols []string, timeout time.Duration) []ProtocolTestResult {
	results := make([]ProtocolTestResult, len(protocols))
	var wg sync.WaitGroup

	for i, protocol := range protocols {
		wg.Add(1)
		go func(idx int, proto string) {
			defer wg.Done()
			results[idx] = testSingleProtocol(ctx, channel, proto, timeout)
		}(i, protocol)
	}

	wg.Wait()
	return results
}

// testSingleProtocol 测试单个协议的兼容性（支持多模型依次尝试）
func testSingleProtocol(ctx context.Context, channel *config.UpstreamConfig, protocol string, timeout time.Duration) ProtocolTestResult {
	result := ProtocolTestResult{
		Protocol: protocol,
		TestedAt: time.Now().Format(time.RFC3339),
	}

	log.Printf("[CapabilityTest] 开始测试渠道 %s 的 %s 协议兼容性", channel.Name, protocol)

	// 获取候选模型列表
	models, err := getCapabilityProbeModels(protocol)
	if err != nil {
		errMsg := "no_models_configured"
		result.Error = &errMsg
		log.Printf("[CapabilityTest] 渠道 %s 获取 %s 协议测试模型失败: %v", channel.Name, protocol, err)
		return result
	}

	// 依次尝试每个模型，直到成功
	var lastError error
	var lastStatusCode int
	totalStart := time.Now()

	for i, model := range models {
		log.Printf("[CapabilityTest] 渠道 %s 尝试模型 %s (%d/%d)", channel.Name, model, i+1, len(models))

		// 构建测试请求
		req, err := buildTestRequestWithModel(protocol, channel, model)
		if err != nil {
			lastError = err
			log.Printf("[CapabilityTest] 渠道 %s 构建 %s 测试请求失败 (模型: %s): %v", channel.Name, protocol, model, err)
			continue
		}

		// 创建带超时的上下文
		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		req = req.WithContext(reqCtx)

		// 获取 HTTP 客户端
		client := httpclient.GetManager().GetStandardClient(timeout, channel.InsecureSkipVerify, channel.ProxyURL)

		// 发送请求并检查流式响应
		startTime := time.Now()
		success, streamingSupported, statusCode, sendErr := sendAndCheckStream(reqCtx, client, req)
		latency := time.Since(startTime).Milliseconds()

		cancel() // 释放上下文资源

		if success {
			// 测试成功，记录结果并返回
			result.Success = true
			result.StreamingSupported = streamingSupported
			result.Latency = latency
			result.TestedModel = model
			log.Printf("[CapabilityTest] 渠道 %s 的 %s 协议测试成功 (模型: %s, 流式: %v, 耗时: %dms)",
				channel.Name, protocol, model, streamingSupported, latency)
			return result
		}

		// 测试失败，记录错误并尝试下一个模型
		lastError = sendErr
		lastStatusCode = statusCode
		log.Printf("[CapabilityTest] 渠道 %s 的 %s 协议测试失败 (模型: %s, 耗时: %dms): %v",
			channel.Name, protocol, model, latency, sendErr)

		// 如果不是最后一个模型，等待 2 秒后再尝试下一个（避免触发上游速率限制）
		if i < len(models)-1 {
			log.Printf("[CapabilityTest] 等待 2 秒后尝试下一个模型...")
			time.Sleep(2 * time.Second)
		}
	}

	// 所有模型都失败
	result.Success = false
	result.Latency = time.Since(totalStart).Milliseconds()
	errMsg := classifyError(lastError, lastStatusCode, ctx)
	result.Error = &errMsg
	log.Printf("[CapabilityTest] 渠道 %s 的 %s 协议所有模型测试均失败 (尝试了 %d 个模型, 总耗时: %dms)",
		channel.Name, protocol, len(models), result.Latency)

	return result
}

// ============== 请求构建 ==============

// buildTestRequestWithModel 构建最小化测试请求（指定模型）
func buildTestRequestWithModel(protocol string, channel *config.UpstreamConfig, model string) (*http.Request, error) {
	// 获取 BaseURL
	urls := channel.GetAllBaseURLs()
	if len(urls) == 0 {
		return nil, fmt.Errorf("no base URL configured")
	}
	baseURL := urls[0]

	// 如果末尾有 #，去掉 # 后不添加版本前缀
	noVersionPrefix := false
	if strings.HasSuffix(baseURL, "#") {
		baseURL = strings.TrimSuffix(baseURL, "#")
		noVersionPrefix = true
	}

	// 处理 BaseURL：去除末尾 /
	baseURL = strings.TrimSuffix(baseURL, "/")

	apiKey := channel.APIKeys[0]

	var (
		requestURL string
		body       []byte
		err        error
		isGemini   bool
	)

	switch protocol {
	case "messages":
		if noVersionPrefix {
			requestURL = baseURL + "/messages"
		} else {
			requestURL = baseURL + "/v1/messages"
		}
		body, err = json.Marshal(map[string]interface{}{
			"model": model,
			"system": []map[string]interface{}{
				{
					"type": "text",
					"text": "x-anthropic-billing-header: cc_version=2.1.71.2f9; cc_entrypoint=cli;",
				},
				{
					"type": "text",
					"text": "You are Claude Code, Anthropic's official CLI for Claude.",
					"cache_control": map[string]string{
						"type": "ephemeral",
					},
				},
			},
			"messages":   []map[string]string{{"role": "user", "content": "What are you best at: code generation, creative writing, or math problem solving?"}},
			"max_tokens": 20,
			"stream":     true,
		})

	case "chat":
		if noVersionPrefix {
			requestURL = baseURL + "/chat/completions"
		} else {
			requestURL = baseURL + "/v1/chat/completions"
		}
		body, err = json.Marshal(map[string]interface{}{
			"model": model,
			"messages": []map[string]string{
				{"role": "system", "content": "You are a helpful assistant."},
				{"role": "user", "content": "What are you best at: code generation, creative writing, or math problem solving?"},
			},
			"max_tokens": 20,
			"stream":     true,
		})

	case "gemini":
		if noVersionPrefix {
			requestURL = baseURL + "/models/" + model + ":streamGenerateContent?alt=sse"
		} else {
			requestURL = baseURL + "/v1beta/models/" + model + ":streamGenerateContent?alt=sse"
		}
		body, err = json.Marshal(map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"role":  "user",
					"parts": []map[string]string{{"text": "What are you best at: code generation, creative writing, or math problem solving?"}},
				},
			},
			"systemInstruction": map[string]interface{}{
				"parts": []map[string]string{{"text": "You are Gemini CLI, an interactive CLI agent specializing in software engineering tasks."}},
			},
			"generationConfig": map[string]interface{}{
				"maxOutputTokens": 20,
			},
		})
		isGemini = true

	case "responses":
		if noVersionPrefix {
			requestURL = baseURL + "/responses"
		} else {
			requestURL = baseURL + "/v1/responses"
		}
		body, err = json.Marshal(map[string]interface{}{
			"model":        model,
			"input":        "What are you best at: code generation, creative writing, or math problem solving?",
			"instructions": "You are Codex, a coding agent based on GPT-5.",
			"max_tokens":   20,
			"stream":       true,
		})

	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}

	if err != nil {
		return nil, fmt.Errorf("marshal request body failed: %w", err)
	}

	req, err := http.NewRequest("POST", requestURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	// 设置通用头部
	req.Header.Set("Content-Type", "application/json")

	// 设置认证头部
	if isGemini {
		utils.SetGeminiAuthenticationHeader(req.Header, apiKey)
	} else {
		utils.SetAuthenticationHeader(req.Header, apiKey)
		// Messages 协议需要 anthropic-version、anthropic-beta、User-Agent 和 X-App 头部
		if protocol == "messages" {
			req.Header.Set("anthropic-version", "2023-06-01")
			req.Header.Set("anthropic-beta", "claude-code-20250219,adaptive-thinking-2026-01-28,prompt-caching-scope-2026-01-05,effort-2025-11-24")
			req.Header.Set("User-Agent", "claude-cli/2.1.71 (external, cli)")
			req.Header.Set("X-App", "cli")
		}
		// Responses 协议需要 Originator 和 User-Agent 头部
		if protocol == "responses" {
			req.Header.Set("Originator", "codex_cli_rs")
			req.Header.Set("User-Agent", "codex_cli_rs/0.111.0 (Mac OS 26.3.0; arm64) iTerm.app/3.6.6")
		}
	}

	// 应用自定义请求头
	if channel.CustomHeaders != nil {
		for key, value := range channel.CustomHeaders {
			req.Header.Set(key, value)
		}
	}

	return req, nil
}

// buildTestRequest 构建最小化测试请求（使用首选模型，兼容旧接口）
func buildTestRequest(protocol string, channel *config.UpstreamConfig) (*http.Request, error) {
	model, err := getCapabilityProbeModel(protocol)
	if err != nil {
		return nil, err
	}
	return buildTestRequestWithModel(protocol, channel, model)
}

// ============== 流式响应检测 ==============

// sendAndCheckStream 发送请求并检查流式响应能力
// 返回: success（HTTP 2xx）, streamingSupported（能解析 SSE chunk）, statusCode, error
func sendAndCheckStream(ctx context.Context, client *http.Client, req *http.Request) (bool, bool, int, error) {
	resp, err := client.Do(req)
	if err != nil {
		return false, false, 0, err
	}
	defer resp.Body.Close()

	// 非 2xx 视为不兼容
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, false, resp.StatusCode, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// HTTP 2xx，尝试读取第一个 SSE chunk 检测流式支持
	streamingSupported := false

	// 使用 5 秒读取超时
	readCtx, readCancel := context.WithTimeout(ctx, 5*time.Second)
	defer readCancel()

	// 在 goroutine 中扫描以支持超时取消
	doneCh := make(chan bool, 1)
	go func() {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				// 跳过 [DONE] 标记
				if data == "[DONE]" {
					continue
				}
				// 尝试 JSON 解析
				var jsonObj map[string]interface{}
				if json.Unmarshal([]byte(data), &jsonObj) == nil {
					doneCh <- true
					return
				}
			}
		}
		doneCh <- false
	}()

	select {
	case result := <-doneCh:
		streamingSupported = result
	case <-readCtx.Done():
		// 读取超时，但 HTTP 2xx 所以 success 仍为 true
		streamingSupported = false
	}

	return true, streamingSupported, resp.StatusCode, nil
}

// ============== 错误分类 ==============

// classifyError 对错误进行分类
func classifyError(err error, statusCode int, ctx context.Context) string {
	if ctx.Err() == context.DeadlineExceeded {
		return "timeout"
	}

	if statusCode == 429 {
		return "rate_limited"
	}

	if statusCode > 0 {
		return fmt.Sprintf("http_error_%d", statusCode)
	}

	errStr := err.Error()
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
		return "timeout"
	}

	return "request_failed: " + errStr
}
