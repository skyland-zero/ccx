package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/BenedictKing/ccx/internal/metrics"
	"github.com/gin-gonic/gin"
)

type CapabilitySnapshot struct {
	IdentityKey         string                        `json:"identityKey"`
	SourceType          string                        `json:"sourceType"`
	Tests               []CapabilityProtocolJobResult `json:"tests"`
	CompatibleProtocols []string                      `json:"compatibleProtocols"`
	TotalDuration       int64                         `json:"totalDuration"`
	Progress            CapabilityTestJobProgress     `json:"progress"`
	Lifecycle           CapabilityLifecycle           `json:"lifecycle"`
	Outcome             CapabilityOutcome             `json:"outcome"`
	UpdatedAt           string                        `json:"updatedAt"`
}

type capabilitySnapshotStore struct {
	sync.RWMutex
	snapshots map[string]*CapabilitySnapshot
	ttl       time.Duration
}

const capabilitySnapshotTTL = 2 * time.Hour

var capabilitySnapshots = newCapabilitySnapshotStoreWithGC()

func newCapabilitySnapshotStore() *capabilitySnapshotStore {
	return &capabilitySnapshotStore{
		snapshots: make(map[string]*CapabilitySnapshot),
		ttl:       capabilitySnapshotTTL,
	}
}

func newCapabilitySnapshotStoreWithGC() *capabilitySnapshotStore {
	store := newCapabilitySnapshotStore()
	go store.gcLoop()
	return store
}

func cloneCapabilitySnapshot(snapshot *CapabilitySnapshot) *CapabilitySnapshot {
	if snapshot == nil {
		return nil
	}
	cloned := *snapshot
	cloned.Tests = make([]CapabilityProtocolJobResult, len(snapshot.Tests))
	for i, test := range snapshot.Tests {
		cloned.Tests[i] = test
		cloned.Tests[i].ModelResults = append([]CapabilityModelJobResult(nil), test.ModelResults...)
	}
	cloned.CompatibleProtocols = append([]string(nil), snapshot.CompatibleProtocols...)
	return &cloned
}

func snapshotFromCapabilityJob(identityKey string, job *CapabilityTestJob) *CapabilitySnapshot {
	if job == nil {
		return nil
	}
	return &CapabilitySnapshot{
		IdentityKey:         identityKey,
		SourceType:          job.SourceType,
		Tests:               append([]CapabilityProtocolJobResult(nil), job.Tests...),
		CompatibleProtocols: append([]string(nil), job.CompatibleProtocols...),
		TotalDuration:       job.TotalDuration,
		Progress:            job.Progress,
		Lifecycle:           job.Lifecycle,
		Outcome:             job.Outcome,
		UpdatedAt:           job.UpdatedAt,
	}
}

func (s *capabilitySnapshotStore) update(identityKey string, updater func(snapshot *CapabilitySnapshot)) *CapabilitySnapshot {
	s.Lock()
	defer s.Unlock()

	snapshot, ok := s.snapshots[identityKey]
	if !ok {
		snapshot = &CapabilitySnapshot{IdentityKey: identityKey}
		s.snapshots[identityKey] = snapshot
	}

	updater(snapshot)
	if snapshot.UpdatedAt == "" {
		snapshot.UpdatedAt = time.Now().Format(time.RFC3339Nano)
	}
	return cloneCapabilitySnapshot(snapshot)
}

func (s *capabilitySnapshotStore) replaceFromJob(identityKey string, job *CapabilityTestJob) *CapabilitySnapshot {
	snapshot := snapshotFromCapabilityJob(identityKey, job)
	if snapshot == nil {
		return nil
	}
	s.Lock()
	s.snapshots[identityKey] = cloneCapabilitySnapshot(snapshot)
	s.Unlock()
	return snapshot
}

func (s *capabilitySnapshotStore) get(identityKey string) (*CapabilitySnapshot, bool) {
	s.RLock()
	defer s.RUnlock()
	snapshot, ok := s.snapshots[identityKey]
	if !ok {
		return nil, false
	}
	return cloneCapabilitySnapshot(snapshot), true
}

func (s *capabilitySnapshotStore) gcLoop() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.gc()
	}
}

func (s *capabilitySnapshotStore) gc() {
	cutoff := time.Now().Add(-s.ttl)
	s.Lock()
	defer s.Unlock()
	for identityKey, snapshot := range s.snapshots {
		if snapshot == nil {
			delete(s.snapshots, identityKey)
			continue
		}
		updatedAt, err := time.Parse(time.RFC3339Nano, snapshot.UpdatedAt)
		if err != nil || updatedAt.Before(cutoff) {
			delete(s.snapshots, identityKey)
		}
	}
}

func resolveCapabilityIdentityKey(channel *config.UpstreamConfig) string {
	if channel == nil {
		return ""
	}
	baseURL := ""
	if len(channel.GetAllBaseURLs()) > 0 {
		baseURL = channel.GetAllBaseURLs()[0]
	}
	apiKey := ""
	if len(channel.APIKeys) > 0 {
		apiKey = channel.APIKeys[0]
	} else if len(channel.DisabledAPIKeys) > 0 {
		apiKey = channel.DisabledAPIKeys[0].Key
	}
	return metrics.GenerateMetricsIdentityKey(baseURL, apiKey, channel.ServiceType)
}

func capabilityJobMatchesChannel(job *CapabilityTestJob, channel *config.UpstreamConfig, channelKind string, channelID int) bool {
	if job == nil {
		return false
	}
	if job.ChannelKind != channelKind {
		return false
	}
	if job.ChannelID == channelID {
		return true
	}
	if channel == nil {
		return false
	}
	identityKey := resolveCapabilityIdentityKey(channel)
	return identityKey != "" && job.IdentityKey == identityKey
}

func GetCapabilitySnapshot(cfgManager *config.ConfigManager, channelKind string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseCapabilityChannelID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
			return
		}

		channel, getErr := getCapabilityTestChannel(cfgManager, channelKind, id)
		if getErr != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
			return
		}

		identityKey := resolveCapabilityIdentityKey(channel)
		snapshot, ok := capabilitySnapshots.get(identityKey)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "Capability snapshot not found"})
			return
		}
		c.JSON(http.StatusOK, snapshot)
	}
}
