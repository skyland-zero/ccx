package handlers

import (
	"testing"
	"time"
)

func TestCapabilitySnapshotStore_ReplaceFromJob(t *testing.T) {
	store := newCapabilitySnapshotStore()
	job := newCapabilityTestJob(1, "channel", "messages", "claude", []string{"messages"}, 0)
	job.IdentityKey = "identity-a"
	job.Tests[0].Outcome = CapabilityOutcomeSuccess
	job.Tests[0].Lifecycle = CapabilityLifecycleDone
	job.CompatibleProtocols = []string{"messages"}
	job.Progress.TotalModels = 2
	job.Progress.CompletedModels = 2
	job.Lifecycle = CapabilityLifecycleDone
	job.Outcome = CapabilityOutcomeSuccess

	store.replaceFromJob(job.IdentityKey, job)

	snapshot, ok := store.get("identity-a")
	if !ok {
		t.Fatal("expected snapshot to exist")
	}
	if snapshot.IdentityKey != "identity-a" {
		t.Fatalf("identityKey=%s, want identity-a", snapshot.IdentityKey)
	}
	if snapshot.Lifecycle != CapabilityLifecycleDone {
		t.Fatalf("lifecycle=%s, want done", snapshot.Lifecycle)
	}
	if len(snapshot.CompatibleProtocols) != 1 || snapshot.CompatibleProtocols[0] != "messages" {
		t.Fatalf("compatibleProtocols=%v, want [messages]", snapshot.CompatibleProtocols)
	}
}

func TestCapabilitySnapshotStore_IsolatesDifferentIdentities(t *testing.T) {
	store := newCapabilitySnapshotStore()

	jobA := newCapabilityTestJob(1, "channel-a", "messages", "claude", []string{"messages"}, 0)
	jobA.IdentityKey = "identity-a"
	jobA.Lifecycle = CapabilityLifecycleDone
	jobA.Outcome = CapabilityOutcomeSuccess
	store.replaceFromJob(jobA.IdentityKey, jobA)

	jobB := newCapabilityTestJob(2, "channel-b", "messages", "claude", []string{"responses"}, 0)
	jobB.IdentityKey = "identity-b"
	jobB.Lifecycle = CapabilityLifecycleCancelled
	jobB.Outcome = CapabilityOutcomeCancelled
	store.replaceFromJob(jobB.IdentityKey, jobB)

	snapshotA, ok := store.get("identity-a")
	if !ok {
		t.Fatal("expected snapshotA to exist")
	}
	snapshotB, ok := store.get("identity-b")
	if !ok {
		t.Fatal("expected snapshotB to exist")
	}
	if snapshotA.IdentityKey == snapshotB.IdentityKey {
		t.Fatal("expected snapshots to be isolated by identity")
	}
	if snapshotA.Outcome != CapabilityOutcomeSuccess {
		t.Fatalf("snapshotA outcome=%s, want success", snapshotA.Outcome)
	}
	if snapshotB.Outcome != CapabilityOutcomeCancelled {
		t.Fatalf("snapshotB outcome=%s, want cancelled", snapshotB.Outcome)
	}
}

func TestCapabilitySnapshotStore_GCRemovesExpiredSnapshots(t *testing.T) {
	store := newCapabilitySnapshotStore()
	store.ttl = time.Hour
	store.snapshots["expired"] = &CapabilitySnapshot{
		IdentityKey: "expired",
		UpdatedAt:   time.Now().Add(-2 * time.Hour).Format(time.RFC3339Nano),
	}
	store.snapshots["fresh"] = &CapabilitySnapshot{
		IdentityKey: "fresh",
		UpdatedAt:   time.Now().Format(time.RFC3339Nano),
	}

	store.gc()

	if _, ok := store.get("expired"); ok {
		t.Fatal("expected expired snapshot to be removed")
	}
	if _, ok := store.get("fresh"); !ok {
		t.Fatal("expected fresh snapshot to remain")
	}
}
