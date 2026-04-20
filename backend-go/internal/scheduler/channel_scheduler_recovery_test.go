package scheduler

import (
	"testing"
	"time"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/BenedictKing/ccx/internal/metrics"
)

func TestNextScheduledRecoveryTimeUTC(t *testing.T) {
	tests := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{
			name: "before midnight slot",
			now:  time.Date(2026, 4, 19, 23, 59, 59, 0, time.UTC),
			want: time.Date(2026, 4, 20, 0, 0, 1, 0, time.UTC),
		},
		{
			name: "between midnight and eight",
			now:  time.Date(2026, 4, 20, 0, 0, 2, 0, time.UTC),
			want: time.Date(2026, 4, 20, 8, 0, 1, 0, time.UTC),
		},
		{
			name: "between eight and sixteen",
			now:  time.Date(2026, 4, 20, 8, 0, 2, 0, time.UTC),
			want: time.Date(2026, 4, 20, 16, 0, 1, 0, time.UTC),
		},
		{
			name: "after sixteen",
			now:  time.Date(2026, 4, 20, 16, 0, 2, 0, time.UTC),
			want: time.Date(2026, 4, 21, 0, 0, 1, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NextScheduledRecoveryTimeUTC(tt.now); !got.Equal(tt.want) {
				t.Fatalf("NextScheduledRecoveryTimeUTC() = %s, want %s", got.Format(time.RFC3339), tt.want.Format(time.RFC3339))
			}
		})
	}
}

func TestRunScheduledRecoveries_RestoresEligibleKeysAndActivatesSuspendedChannel(t *testing.T) {
	now := time.Date(2026, 4, 20, 8, 0, 1, 0, time.UTC)
	older := now.Add(-2 * time.Hour).Format(time.RFC3339)
	recent := now.Add(-30 * time.Minute).Format(time.RFC3339)

	cfg := config.Config{
		Upstream: []config.UpstreamConfig{{
			Name:        "msg-channel",
			BaseURLs:    []string{"https://a.example.com", "https://b.example.com"},
			Status:      "suspended",
			APIKeys:     nil,
			ServiceType: "claude",
			DisabledAPIKeys: []config.DisabledKeyInfo{
				{Key: "sk-balance", Reason: "insufficient_balance", DisabledAt: older},
				{Key: "sk-auth", Reason: "authentication_error", DisabledAt: older},
				{Key: "sk-recent", Reason: "insufficient_balance", DisabledAt: recent},
			},
		}},
		ChatUpstream: []config.UpstreamConfig{{
			Name:        "chat-disabled",
			BaseURL:     "https://chat.example.com",
			Status:      "disabled",
			APIKeys:     nil,
			ServiceType: "openai",
			DisabledAPIKeys: []config.DisabledKeyInfo{{
				Key: "sk-chat-balance", Reason: "insufficient_balance", DisabledAt: older,
			}},
		}},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	results, err := scheduler.RunScheduledRecoveries(now)
	if err != nil {
		t.Fatalf("RunScheduledRecoveries() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}
	if !results[0].ActivatedChannel {
		t.Fatal("ActivatedChannel = false, want true")
	}
	if len(results[0].RestoredKeys) != 1 || results[0].RestoredKeys[0] != "sk-balance" {
		t.Fatalf("RestoredKeys = %v, want [sk-balance]", results[0].RestoredKeys)
	}

	updated := scheduler.configManager.GetConfig()
	if updated.Upstream[0].Status != "active" {
		t.Fatalf("status = %s, want active", updated.Upstream[0].Status)
	}
	if len(updated.Upstream[0].APIKeys) != 1 || updated.Upstream[0].APIKeys[0] != "sk-balance" {
		t.Fatalf("APIKeys = %v, want [sk-balance]", updated.Upstream[0].APIKeys)
	}
	if len(updated.Upstream[0].DisabledAPIKeys) != 2 {
		t.Fatalf("DisabledAPIKeys len = %d, want 2", len(updated.Upstream[0].DisabledAPIKeys))
	}
	if updated.ChatUpstream[0].Status != "disabled" {
		t.Fatalf("chat status = %s, want disabled", updated.ChatUpstream[0].Status)
	}

	for _, baseURL := range []string{"https://a.example.com", "https://b.example.com"} {
		if got := scheduler.GetMessagesMetricsManager().GetKeyCircuitState(baseURL, "sk-balance", "claude"); got != metrics.CircuitStateHalfOpen {
			t.Fatalf("circuit state for %s = %v, want %v", baseURL, got, metrics.CircuitStateHalfOpen)
		}
	}
}
