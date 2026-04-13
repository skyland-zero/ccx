// Package common 提供 handlers 模块的公共功能
package common

import (
	"time"

	"github.com/BenedictKing/ccx/internal/metrics"
	"github.com/BenedictKing/ccx/internal/utils"
)

// RecordChannelLog 统一记录渠道尝试日志。
// 约束：凡是会进入渠道图表统计的尝试，都应该调用此函数或等价逻辑写入日志，保证图表与日志口径一致。
func RecordChannelLog(
	channelLogStore *metrics.ChannelLogStore,
	channelIndex int,
	model, originalModel string,
	statusCode int,
	durationMs int64,
	success bool,
	apiKey, baseURL, errorInfo, interfaceType string,
	isRetry bool,
) {
	if channelLogStore == nil {
		return
	}
	if len(errorInfo) > 200 {
		errorInfo = errorInfo[:200]
	}

	channelLogStore.Record(channelIndex, &metrics.ChannelLog{
		Timestamp:     time.Now(),
		Model:         model,
		OriginalModel: originalModel,
		StatusCode:    statusCode,
		DurationMs:    durationMs,
		Success:       success,
		KeyMask:       utils.MaskAPIKey(apiKey),
		BaseURL:       baseURL,
		ErrorInfo:     errorInfo,
		IsRetry:       isRetry,
		InterfaceType: interfaceType,
	})
}
