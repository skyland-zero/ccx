package handlers

import (
	"context"
	"testing"
	"time"
)

func TestCapabilityTestDispatcherPool_ReusesSameIdentity(t *testing.T) {
	pool := newCapabilityTestDispatcherPool()

	first := pool.Get("identity-a")
	second := pool.Get("identity-a")

	if first == nil || second == nil {
		t.Fatal("expected non-nil dispatchers")
	}
	if first != second {
		t.Fatal("expected same dispatcher for same identity")
	}
}

func TestCapabilityTestDispatcherPool_IsolatesDifferentIdentities(t *testing.T) {
	pool := newCapabilityTestDispatcherPool()

	first := pool.Get("identity-a")
	second := pool.Get("identity-b")

	if first == nil || second == nil {
		t.Fatal("expected non-nil dispatchers")
	}
	if first == second {
		t.Fatal("expected different dispatchers for different identities")
	}
}

func TestCapabilityTestDispatcherPool_DefaultKey(t *testing.T) {
	pool := newCapabilityTestDispatcherPool()

	first := pool.Get("")
	second := pool.Get("")

	if first == nil || second == nil {
		t.Fatal("expected non-nil dispatchers")
	}
	if first != second {
		t.Fatal("expected empty identity to reuse default dispatcher")
	}
}

func TestCapabilityTestDispatcherPool_GCRemovesIdleDispatcher(t *testing.T) {
	pool := newCapabilityTestDispatcherPool()
	pool.idleTTL = time.Millisecond

	dispatcher := pool.Get("identity-a")
	dispatcher.lastUsed.Store(time.Now().Add(-time.Second).UnixNano())

	pool.gc()

	pool.mu.RLock()
	_, ok := pool.dispatchers["identity-a"]
	pool.mu.RUnlock()
	if ok {
		t.Fatal("expected idle dispatcher to be removed by gc")
	}
}

func TestCapabilityTestDispatcher_AcquireSendSlotOnClosedDispatcherReturnsImmediately(t *testing.T) {
	dispatcher := newCapabilityTestDispatcher()
	dispatcher.mu.Lock()
	dispatcher.closed.Store(true)
	dispatcher.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	started := time.Now()
	err := dispatcher.AcquireSendSlot(ctx, time.Millisecond)
	if err == nil {
		t.Fatal("expected closed dispatcher to return an error")
	}
	if time.Since(started) > 50*time.Millisecond {
		t.Fatalf("AcquireSendSlot blocked too long on closed dispatcher: %s", time.Since(started))
	}
}
