package converters

import "testing"

func TestExtractUsageMetrics_ClaudeTTLOnlyFields(t *testing.T) {
	usageRaw := map[string]interface{}{
		"input_tokens":                   100,
		"output_tokens":                  20,
		"cache_read_input_tokens":        30,
		"cache_creation_5m_input_tokens": 7,
		"cache_creation_1h_input_tokens": 3,
	}

	usage := ExtractUsageMetrics(usageRaw)
	if usage.TotalTokens != 160 {
		t.Fatalf("TotalTokens = %d, want 160", usage.TotalTokens)
	}
	if usage.CacheTTL != "mixed" {
		t.Fatalf("CacheTTL = %q, want mixed", usage.CacheTTL)
	}
	if usage.CacheCreation5mInputTokens != 7 || usage.CacheCreation1hInputTokens != 3 {
		t.Fatalf("cache ttl split mismatch: %#v", usage)
	}
}

func TestExtractUsageMetrics_ClaudeCacheCreationUsesAggregateWhenPresent(t *testing.T) {
	usageRaw := map[string]interface{}{
		"input_tokens":                   100,
		"output_tokens":                  20,
		"cache_read_input_tokens":        30,
		"cache_creation_input_tokens":    5,
		"cache_creation_5m_input_tokens": 7,
		"cache_creation_1h_input_tokens": 3,
	}

	usage := ExtractUsageMetrics(usageRaw)
	// 当 aggregate 字段存在时，优先使用 cache_creation_input_tokens，避免与 5m/1h 重复累计。
	if usage.TotalTokens != 155 {
		t.Fatalf("TotalTokens = %d, want 155", usage.TotalTokens)
	}
}
