package metrics

import (
	"math"
	"testing"

	"github.com/BenedictKing/ccx/internal/types"
)

func TestToResponse_TimeWindowsIncludesCacheStats(t *testing.T) {
	m := NewMetricsManagerWithConfig(10, 0.5)

	baseURL := "https://example.com"
	key1 := "k1"
	key2 := "k2"

	m.RecordSuccessWithUsage(baseURL, key1, &types.Usage{
		InputTokens:              100,
		OutputTokens:             10,
		CacheCreationInputTokens: 20,
		CacheReadInputTokens:     50,
	})
	m.RecordSuccessWithUsage(baseURL, key2, &types.Usage{
		InputTokens:  200,
		OutputTokens: 20,
	})

	resp := m.ToResponse(0, baseURL, []string{key1, key2}, 0)
	stats, ok := resp.TimeWindows["15m"]
	if !ok {
		t.Fatalf("expected timeWindows[15m] to exist")
	}

	if stats.InputTokens != 300 {
		t.Fatalf("expected inputTokens=300, got %d", stats.InputTokens)
	}
	if stats.OutputTokens != 30 {
		t.Fatalf("expected outputTokens=30, got %d", stats.OutputTokens)
	}
	if stats.CacheCreationTokens != 20 {
		t.Fatalf("expected cacheCreationTokens=20, got %d", stats.CacheCreationTokens)
	}
	if stats.CacheReadTokens != 50 {
		t.Fatalf("expected cacheReadTokens=50, got %d", stats.CacheReadTokens)
	}

	wantHitRate := float64(50) / float64(50+300) * 100
	if math.Abs(stats.CacheHitRate-wantHitRate) > 0.01 {
		t.Fatalf("expected cacheHitRate=%.4f, got %.4f", wantHitRate, stats.CacheHitRate)
	}
}

func TestRecordSuccessWithUsage_NormalizesResponsesPromptTotalsForCacheHitRate(t *testing.T) {
	m := NewMetricsManagerWithConfig(10, 0.5)

	baseURL := "https://example.com"
	key := "k1"

	m.RecordSuccessWithUsage(baseURL, key, &types.Usage{
		InputTokens:          114931,
		PromptTokensTotal:    114931,
		OutputTokens:         100,
		CacheReadInputTokens: 112256,
	})

	resp := m.ToResponse(0, baseURL, []string{key}, 0)
	stats, ok := resp.TimeWindows["15m"]
	if !ok {
		t.Fatalf("expected timeWindows[15m] to exist")
	}

	wantInput := int64(114931 - 112256)
	if stats.InputTokens != wantInput {
		t.Fatalf("expected normalized inputTokens=%d, got %d", wantInput, stats.InputTokens)
	}
	if stats.CacheReadTokens != 112256 {
		t.Fatalf("expected cacheReadTokens=112256, got %d", stats.CacheReadTokens)
	}

	wantHitRate := float64(112256) / float64(114931) * 100
	if math.Abs(stats.CacheHitRate-wantHitRate) > 0.01 {
		t.Fatalf("expected cacheHitRate=%.4f, got %.4f", wantHitRate, stats.CacheHitRate)
	}
}

func TestRecordSuccessWithUsage_CacheCreationFallbackFromTTLBreakdown(t *testing.T) {
	m := NewMetricsManagerWithConfig(10, 0.5)

	baseURL := "https://example.com"
	key := "k1"

	// 上游有时只返回 TTL 细分字段（5m/1h），不返回 cache_creation_input_tokens。
	m.RecordSuccessWithUsage(baseURL, key, &types.Usage{
		InputTokens:                100,
		OutputTokens:               10,
		CacheCreationInputTokens:   0,
		CacheCreation5mInputTokens: 20,
		CacheCreation1hInputTokens: 30,
		CacheReadInputTokens:       50,
	})

	resp := m.ToResponse(0, baseURL, []string{key}, 0)
	stats, ok := resp.TimeWindows["15m"]
	if !ok {
		t.Fatalf("expected timeWindows[15m] to exist")
	}

	if stats.CacheCreationTokens != 50 {
		t.Fatalf("expected cacheCreationTokens=50, got %d", stats.CacheCreationTokens)
	}
	if stats.CacheReadTokens != 50 {
		t.Fatalf("expected cacheReadTokens=50, got %d", stats.CacheReadTokens)
	}
}
