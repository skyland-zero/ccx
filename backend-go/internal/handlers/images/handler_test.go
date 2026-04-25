package images

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/gin-gonic/gin"
)

func TestBuildProviderRequest_URLVariants(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"image-default","prompt":"hello"}`))

	upstream := &config.UpstreamConfig{ServiceType: "openai"}
	req, err := buildProviderRequest(c, upstream, "https://api.openai.com", "sk-test", []byte(`{"model":"image-default","prompt":"hello"}`), "image-default")
	if err != nil {
		t.Fatalf("buildProviderRequest() error = %v", err)
	}
	if req.URL.String() != "https://api.openai.com/v1/images/generations" {
		t.Fatalf("unexpected url: %s", req.URL.String())
	}

	req, err = buildProviderRequest(c, upstream, "https://api.openai.com#", "sk-test", []byte(`{"model":"image-default","prompt":"hello"}`), "image-default")
	if err != nil {
		t.Fatalf("buildProviderRequest() error = %v", err)
	}
	if req.URL.String() != "https://api.openai.com/images/generations" {
		t.Fatalf("unexpected # url: %s", req.URL.String())
	}
}

func TestBuildProviderRequest_RejectsUnsupportedServiceType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"image-default","prompt":"hello"}`))

	upstream := &config.UpstreamConfig{ServiceType: "gemini"}
	_, err := buildProviderRequest(c, upstream, "https://api.openai.com", "sk-test", []byte(`{"model":"image-default","prompt":"hello"}`), "image-default")
	if err == nil {
		t.Fatal("expected error for unsupported serviceType")
	}
	if !strings.Contains(err.Error(), "仅支持 openai serviceType") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddUpstream_RejectsUnsupportedServiceType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfgFile := t.TempDir() + "/config.json"
	if err := os.WriteFile(cfgFile, []byte(`{"upstream":[],"imagesUpstream":[]}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfgManager, err := config.NewConfigManager(cfgFile)
	if err != nil {
		t.Fatalf("config manager: %v", err)
	}
	defer cfgManager.Close()

	r := gin.New()
	r.POST("/api/images/channels", AddUpstream(cfgManager))

	body := strings.NewReader(`{"name":"images-gemini","serviceType":"gemini","baseUrl":"https://example.com","apiKeys":["test-key"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/images/channels", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Images 渠道仅支持 openai serviceType") {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestHandler_MissingModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfgFile := t.TempDir() + "/config.json"
	if err := os.WriteFile(cfgFile, []byte(`{"upstream":[]}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfgManager, err := config.NewConfigManager(cfgFile)
	if err != nil {
		t.Fatalf("config manager: %v", err)
	}
	defer cfgManager.Close()

	envCfg := config.NewEnvConfig()
	envCfg.ProxyAccessKey = "test-key"

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"prompt":"hello"}`))
	c.Request.Header.Set("Authorization", "Bearer test-key")
	Handler(envCfg, cfgManager, nil)(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandler_MissingPrompt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfgFile := t.TempDir() + "/config.json"
	if err := os.WriteFile(cfgFile, []byte(`{"upstream":[]}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfgManager, err := config.NewConfigManager(cfgFile)
	if err != nil {
		t.Fatalf("config manager: %v", err)
	}
	defer cfgManager.Close()

	envCfg := config.NewEnvConfig()
	envCfg.ProxyAccessKey = "test-key"

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"gpt-image-1"}`))
	c.Request.Header.Set("Authorization", "Bearer test-key")
	Handler(envCfg, cfgManager, nil)(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
