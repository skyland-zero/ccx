package responses

import (
	"testing"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/BenedictKing/ccx/internal/types"
)

func TestCalculateTotalTokensWithCache(t *testing.T) {
	if got := calculateTotalTokensWithCache(100, 20, 30, 5, 7, 3); got != 155 {
		t.Fatalf("aggregate cache creation should win: got %d, want 155", got)
	}
	if got := calculateTotalTokensWithCache(100, 20, 30, 0, 7, 3); got != 160 {
		t.Fatalf("ttl split should be summed when aggregate missing: got %d, want 160", got)
	}
}

func TestPatchResponsesUsage_RecalculateTotalIncludesCacheTokens(t *testing.T) {
	resp := &types.ResponsesResponse{
		Usage: types.ResponsesUsage{
			InputTokens:                100,
			OutputTokens:               20,
			TotalTokens:                0,
			CacheReadInputTokens:       30,
			CacheCreation5mInputTokens: 7,
			CacheCreation1hInputTokens: 3,
		},
	}

	patchResponsesUsage(resp, []byte(`{"model":"claude","input":"hello"}`), &config.EnvConfig{})
	if resp.Usage.TotalTokens != 160 {
		t.Fatalf("TotalTokens = %d, want 160", resp.Usage.TotalTokens)
	}
}

func TestPromptTokensTotalFromResponsesInput(t *testing.T) {
	tests := []struct {
		name           string
		inputTokens    int
		upstreamType   string
		hasClaudeCache bool
		want           int
	}{
		{
			name:           "responses total preserved when input is valid",
			inputTokens:    114931,
			upstreamType:   "responses",
			hasClaudeCache: false,
			want:           114931,
		},
		{
			name:           "patched tiny input without claude cache is ignored",
			inputTokens:    1,
			upstreamType:   "responses",
			hasClaudeCache: false,
			want:           0,
		},
		{
			name:           "claude cache backed tiny input keeps total",
			inputTokens:    1,
			upstreamType:   "responses",
			hasClaudeCache: true,
			want:           1,
		},
		{
			name:           "non responses upstream never records total",
			inputTokens:    114931,
			upstreamType:   "chat",
			hasClaudeCache: false,
			want:           0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := promptTokensTotalFromResponsesInput(tt.inputTokens, tt.upstreamType, tt.hasClaudeCache)
			if got != tt.want {
				t.Fatalf("promptTokensTotalFromResponsesInput() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMetricsUsageFromResponsesUsage_UsesCachedTokensFallback(t *testing.T) {
	usage := metricsUsageFromResponsesUsage(types.ResponsesUsage{
		InputTokens:        114931,
		OutputTokens:       100,
		InputTokensDetails: &types.InputTokensDetails{CachedTokens: 112256},
	}, 114931)

	if usage.CacheReadInputTokens != 112256 {
		t.Fatalf("CacheReadInputTokens = %d, want 112256", usage.CacheReadInputTokens)
	}
	if usage.PromptTokensTotal != 114931 {
		t.Fatalf("PromptTokensTotal = %d, want 114931", usage.PromptTokensTotal)
	}
}
