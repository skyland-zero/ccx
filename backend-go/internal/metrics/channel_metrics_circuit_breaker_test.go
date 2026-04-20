package metrics

import (
	"testing"
	"time"
)

func TestMoveKeyToHalfOpenCreatesMetricsAndSwitchesState(t *testing.T) {
	m := NewMetricsManager()
	defer m.Stop()

	m.MoveKeyToHalfOpen("https://example.com", "sk-test", "claude")

	if got := m.GetKeyCircuitState("https://example.com", "sk-test", "claude"); got != CircuitStateHalfOpen {
		t.Fatalf("circuit state = %v, want %v", got, CircuitStateHalfOpen)
	}

	metricsKey := GenerateMetricsIdentityKey("https://example.com", "sk-test", "claude")
	m.mu.RLock()
	metrics := m.keyMetrics[metricsKey]
	m.mu.RUnlock()
	if metrics == nil {
		t.Fatal("metrics should be created")
	}
	if metrics.NextRetryAt != nil {
		t.Fatalf("NextRetryAt = %v, want nil", metrics.NextRetryAt)
	}
	if metrics.HalfOpenAt == nil {
		t.Fatal("HalfOpenAt should be set")
	}
}

func TestMoveKeyToHalfOpenKeepsBreakerHistory(t *testing.T) {
	m := NewMetricsManager()
	defer m.Stop()

	m.mu.Lock()
	metrics := m.getOrCreateKey("https://example.com", "sk-test", "claude")
	metrics.breakerResults = []bool{false, false, true}
	metrics.BackoffLevel = 2
	nextRetryAt := time.Now().Add(time.Minute)
	metrics.NextRetryAt = &nextRetryAt
	metrics.CircuitState = CircuitStateOpen
	m.mu.Unlock()

	m.MoveKeyToHalfOpen("https://example.com", "sk-test", "claude")

	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(metrics.breakerResults) != 3 {
		t.Fatalf("breakerResults len = %d, want 3", len(metrics.breakerResults))
	}
	if metrics.BackoffLevel != 2 {
		t.Fatalf("BackoffLevel = %d, want 2", metrics.BackoffLevel)
	}
	if metrics.CircuitState != CircuitStateHalfOpen {
		t.Fatalf("CircuitState = %v, want %v", metrics.CircuitState, CircuitStateHalfOpen)
	}
}
