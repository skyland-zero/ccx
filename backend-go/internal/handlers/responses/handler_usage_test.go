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

func TestMetricsUsageFromResponsesUsage_UsesCachedTokensFallback(t *testing.T) {
	usage := metricsUsageFromResponsesUsage(types.ResponsesUsage{
		InputTokens:        114931,
		OutputTokens:       100,
		InputTokensDetails: &types.InputTokensDetails{CachedTokens: 112256},
	}, "responses")

	if usage.CacheReadInputTokens != 112256 {
		t.Fatalf("CacheReadInputTokens = %d, want 112256", usage.CacheReadInputTokens)
	}
	if usage.PromptTokensTotal != 114931 {
		t.Fatalf("PromptTokensTotal = %d, want 114931", usage.PromptTokensTotal)
	}
}
