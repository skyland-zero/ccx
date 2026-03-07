package handlers

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/BenedictKing/ccx/internal/config"
)

func TestGetCapabilityProbeModel(t *testing.T) {
	cases := []struct {
		protocol string
		want     string
		wantErr  bool
	}{
		{protocol: "messages", want: "claude-opus-4-6"},
		{protocol: "chat", want: "gpt-5.4"},
		{protocol: "gemini", want: "gemini-3.1-pro-preview"},
		{protocol: "responses", want: "gpt-5.4"},
		{protocol: "unknown", wantErr: true},
	}

	for _, tc := range cases {
		got, err := getCapabilityProbeModel(tc.protocol)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("protocol=%s should return error", tc.protocol)
			}
			continue
		}
		if err != nil {
			t.Fatalf("protocol=%s unexpected error: %v", tc.protocol, err)
		}
		if got != tc.want {
			t.Fatalf("protocol=%s model=%s want=%s", tc.protocol, got, tc.want)
		}
	}
}

func TestBuildTestRequest_UsesCentralizedProbeModels(t *testing.T) {
	channel := &config.UpstreamConfig{
		BaseURL: "https://api.example.com",
		APIKeys: []string{"test-key"},
	}

	cases := []struct {
		protocol      string
		expectedURL   string
		expectedModel string
		modelInURL    bool
	}{
		{
			protocol:      "messages",
			expectedURL:   "https://api.example.com/v1/messages",
			expectedModel: "claude-opus-4-6",
		},
		{
			protocol:      "chat",
			expectedURL:   "https://api.example.com/v1/chat/completions",
			expectedModel: "gpt-5.4",
		},
		{
			protocol:      "gemini",
			expectedURL:   "https://api.example.com/v1beta/models/gemini-3.1-pro-preview:streamGenerateContent?alt=sse",
			expectedModel: "gemini-3.1-pro-preview",
			modelInURL:    true,
		},
		{
			protocol:      "responses",
			expectedURL:   "https://api.example.com/v1/responses",
			expectedModel: "gpt-5.4",
		},
	}

	for _, tc := range cases {
		req, err := buildTestRequest(tc.protocol, channel)
		if err != nil {
			t.Fatalf("protocol=%s build request failed: %v", tc.protocol, err)
		}

		if got := req.URL.String(); got != tc.expectedURL {
			t.Fatalf("protocol=%s url=%s want=%s", tc.protocol, got, tc.expectedURL)
		}

		if tc.modelInURL {
			if !strings.Contains(req.URL.Path, tc.expectedModel) {
				t.Fatalf("protocol=%s url path=%s should contain model=%s", tc.protocol, req.URL.Path, tc.expectedModel)
			}
			continue
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("protocol=%s read body failed: %v", tc.protocol, err)
		}

		var payload map[string]interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("protocol=%s unmarshal body failed: %v, body=%s", tc.protocol, err, string(body))
		}

		model, ok := payload["model"].(string)
		if !ok {
			t.Fatalf("protocol=%s body missing model field", tc.protocol)
		}
		if model != tc.expectedModel {
			t.Fatalf("protocol=%s model=%s want=%s", tc.protocol, model, tc.expectedModel)
		}
	}
}
