package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/gin-gonic/gin"
)

func TestCancelCapabilityTestJob_HTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	job := newCapabilityTestJob(0, "channel", "messages", "claude", []string{"messages"}, 10*time.Second)
	job.Status = CapabilityJobStatusRunning
	job.Lifecycle = CapabilityLifecycleActive
	job.Tests[0].ModelResults = []CapabilityModelJobResult{
		{Model: "queued", Status: CapabilityModelStatusQueued, Lifecycle: CapabilityLifecyclePending, Outcome: CapabilityOutcomeUnknown},
		{Model: "running", Status: CapabilityModelStatusRunning, Lifecycle: CapabilityLifecycleActive, Outcome: CapabilityOutcomeUnknown},
	}
	capabilityJobs.create(job)
	capabilityJobs.setCancelFunc(job.JobID, func() {})

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configFile, []byte(`{"upstream":[]}`), 0644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}
	cfgManager, err := config.NewConfigManager(configFile)
	if err != nil {
		t.Fatalf("create config manager failed: %v", err)
	}
	defer cfgManager.Close()

	r := gin.New()
	r.DELETE("/messages/channels/:id/capability-test/:jobId", CancelCapabilityTestJob(cfgManager, "messages"))

	req := httptest.NewRequest(http.MethodDelete, "/messages/channels/0/capability-test/"+job.JobID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want=%d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	stored, ok := capabilityJobs.get(job.JobID)
	if !ok {
		t.Fatalf("job not found after cancel")
	}
	if stored.Lifecycle != CapabilityLifecycleCancelled {
		t.Fatalf("job lifecycle=%s, want cancelled", stored.Lifecycle)
	}
	if stored.Tests[0].ModelResults[1].Outcome != CapabilityOutcomeCancelled {
		t.Fatalf("running model outcome=%s, want cancelled", stored.Tests[0].ModelResults[1].Outcome)
	}
}

func TestGetCapabilityTestJobStatus_HTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	job := newCapabilityTestJob(0, "channel", "messages", "claude", []string{"messages"}, 10*time.Second)
	job.Lifecycle = CapabilityLifecycleDone
	job.Outcome = CapabilityOutcomePartial
	job.Status = CapabilityJobStatusCompleted
	capabilityJobs.create(job)

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configFile, []byte(`{"upstream":[{"name":"channel","service_type":"claude","base_url":"https://example.com","api_keys":["test"]}]}`), 0644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}
	cfgManager, err := config.NewConfigManager(configFile)
	if err != nil {
		t.Fatalf("create config manager failed: %v", err)
	}
	defer cfgManager.Close()

	r := gin.New()
	r.GET("/messages/channels/:id/capability-test/:jobId", GetCapabilityTestJobStatus(cfgManager, "messages"))

	req := httptest.NewRequest(http.MethodGet, "/messages/channels/0/capability-test/"+job.JobID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want=%d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp CapabilityTestJob
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if resp.Outcome != CapabilityOutcomePartial {
		t.Fatalf("outcome=%s, want partial", resp.Outcome)
	}
}

func TestRetryCapabilityTestModel_HTTP_RejectsUnknownModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	job := newCapabilityTestJob(0, "channel", "messages", "claude", []string{"messages"}, 10*time.Second)
	job.Status = CapabilityJobStatusCompleted
	job.Lifecycle = CapabilityLifecycleDone
	job.Outcome = CapabilityOutcomeFailed
	job.Tests[0].ModelResults = []CapabilityModelJobResult{{Model: "known", Status: CapabilityModelStatusFailed, Lifecycle: CapabilityLifecycleDone, Outcome: CapabilityOutcomeFailed}}
	capabilityJobs.create(job)

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configFile, []byte(`{"upstream":[{"name":"channel","service_type":"claude","base_url":"https://example.com","api_keys":["test"]}]}`), 0644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}
	cfgManager, err := config.NewConfigManager(configFile)
	if err != nil {
		t.Fatalf("create config manager failed: %v", err)
	}
	defer cfgManager.Close()

	r := gin.New()
	r.POST("/messages/channels/:id/capability-test/:jobId/retry", RetryCapabilityTestModel(cfgManager, "messages"))

	body := `{"protocol":"messages","model":"unknown"}`
	req := httptest.NewRequest(http.MethodPost, "/messages/channels/0/capability-test/"+job.JobID+"/retry", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want=%d, body=%s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestRetryCapabilityTestModel_HTTP_RejectsRunningJob(t *testing.T) {
	gin.SetMode(gin.TestMode)

	job := newCapabilityTestJob(0, "channel", "messages", "claude", []string{"messages"}, 10*time.Second)
	job.Status = CapabilityJobStatusRunning
	job.Lifecycle = CapabilityLifecycleActive
	job.Tests[0].ModelResults = []CapabilityModelJobResult{
		{Model: "known", Status: CapabilityModelStatusFailed, Lifecycle: CapabilityLifecycleDone, Outcome: CapabilityOutcomeFailed},
	}
	capabilityJobs.create(job)

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configFile, []byte(`{"upstream":[{"name":"channel","service_type":"claude","base_url":"https://example.com","api_keys":["test"]}]}`), 0644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}
	cfgManager, err := config.NewConfigManager(configFile)
	if err != nil {
		t.Fatalf("create config manager failed: %v", err)
	}
	defer cfgManager.Close()

	r := gin.New()
	r.POST("/messages/channels/:id/capability-test/:jobId/retry", RetryCapabilityTestModel(cfgManager, "messages"))

	body := `{"protocol":"messages","model":"known"}`
	req := httptest.NewRequest(http.MethodPost, "/messages/channels/0/capability-test/"+job.JobID+"/retry", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("status=%d, want=%d, body=%s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestRetryCapabilityTestModel_HTTP_RejectsNonRetryableModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	job := newCapabilityTestJob(0, "channel", "messages", "claude", []string{"messages"}, 10*time.Second)
	job.Status = CapabilityJobStatusCompleted
	job.Lifecycle = CapabilityLifecycleDone
	job.Outcome = CapabilityOutcomeSuccess
	job.Tests[0].ModelResults = []CapabilityModelJobResult{
		{Model: "known", Status: CapabilityModelStatusSuccess, Lifecycle: CapabilityLifecycleDone, Outcome: CapabilityOutcomeSuccess, Success: true},
	}
	capabilityJobs.create(job)

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configFile, []byte(`{"upstream":[{"name":"channel","service_type":"claude","base_url":"https://example.com","api_keys":["test"]}]}`), 0644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}
	cfgManager, err := config.NewConfigManager(configFile)
	if err != nil {
		t.Fatalf("create config manager failed: %v", err)
	}
	defer cfgManager.Close()

	r := gin.New()
	r.POST("/messages/channels/:id/capability-test/:jobId/retry", RetryCapabilityTestModel(cfgManager, "messages"))

	body := `{"protocol":"messages","model":"known"}`
	req := httptest.NewRequest(http.MethodPost, "/messages/channels/0/capability-test/"+job.JobID+"/retry", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("status=%d, want=%d, body=%s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestExecuteModelTest_RespectsAutoBlacklistBalance(t *testing.T) {
	gin.SetMode(gin.TestMode)

	job := newCapabilityTestJob(0, "channel", "messages", "claude", []string{"messages"}, 10*time.Second)
	capabilityJobs.create(job)

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	initialConfig := `{
		"upstream": [{
			"name": "channel",
			"baseUrl": "REPLACE_ME",
			"apiKeys": ["test-key"],
			"serviceType": "claude",
			"autoBlacklistBalance": false
		}]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"code":"INSUFFICIENT_BALANCE","message":"Insufficient account balance"}`))
	}))
	defer server.Close()

	initialConfig = strings.Replace(initialConfig, "REPLACE_ME", server.URL, 1)
	if err := os.WriteFile(configFile, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	cfgManager, err := config.NewConfigManager(configFile)
	if err != nil {
		t.Fatalf("create config manager failed: %v", err)
	}
	defer cfgManager.Close()

	cfg := cfgManager.GetConfig()
	if len(cfg.Upstream) != 1 {
		t.Fatalf("upstream count=%d, want 1", len(cfg.Upstream))
	}

	result := executeModelTest(context.Background(), &cfg.Upstream[0], "messages", "claude-test", 5*time.Second, job.JobID, cfgManager, 0, "messages", "test-key")
	if result.Success {
		t.Fatalf("result.Success=true, want false")
	}

	updated := cfgManager.GetConfig()
	if len(updated.Upstream[0].DisabledAPIKeys) != 0 {
		t.Fatalf("DisabledAPIKeys=%+v, want empty when autoBlacklistBalance=false", updated.Upstream[0].DisabledAPIKeys)
	}
	if len(updated.Upstream[0].APIKeys) != 1 || updated.Upstream[0].APIKeys[0] != "test-key" {
		t.Fatalf("APIKeys=%v, want original key preserved", updated.Upstream[0].APIKeys)
	}
}
