package scheduler

import (
	"testing"
	"time"

	"github.com/BenedictKing/ccx/internal/config"
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

func TestRunScheduledRecoveries_UsesRecoverAtWhenPresent(t *testing.T) {
	now := time.Date(2026, 4, 20, 8, 0, 1, 0, time.UTC)
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{{
			Name:        "msg-channel",
			BaseURL:     "https://a.example.com",
			Status:      "suspended",
			APIKeys:     nil,
			ServiceType: "claude",
			DisabledAPIKeys: []config.DisabledKeyInfo{
				{Key: "sk-ready", Reason: "insufficient_balance", DisabledAt: now.Add(-30 * time.Minute).Format(time.RFC3339), RecoverAt: now.Add(-time.Minute).Format(time.RFC3339)},
				{Key: "sk-wait", Reason: "insufficient_balance", DisabledAt: now.Add(-3 * time.Hour).Format(time.RFC3339), RecoverAt: now.Add(time.Hour).Format(time.RFC3339)},
			},
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
	if len(results[0].RestoredKeys) != 1 || results[0].RestoredKeys[0] != "sk-ready" {
		t.Fatalf("RestoredKeys = %v, want [sk-ready]", results[0].RestoredKeys)
	}

	updated := scheduler.configManager.GetConfig()
	if len(updated.Upstream[0].DisabledAPIKeys) != 1 || updated.Upstream[0].DisabledAPIKeys[0].Key != "sk-wait" {
		t.Fatalf("DisabledAPIKeys = %+v, want only sk-wait left", updated.Upstream[0].DisabledAPIKeys)
	}
	if len(updated.Upstream[0].APIKeys) != 1 || updated.Upstream[0].APIKeys[0] != "sk-ready" {
		t.Fatalf("APIKeys = %v, want [sk-ready]", updated.Upstream[0].APIKeys)
	}
}
