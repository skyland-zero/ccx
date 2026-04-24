package chat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/gin-gonic/gin"
)

func TestBuildProviderRequest_InjectsReasoningBeforeModelRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil).WithContext(context.Background())

	bodyBytes := []byte(`{"model":"gpt-5.1-codex","messages":[{"role":"user","content":"hi"}]}`)
	upstream := &config.UpstreamConfig{
		ServiceType: "openai",
		ModelMapping: map[string]string{
			"gpt-5.1-codex": "gpt-5.2-codex",
		},
		ReasoningMapping: map[string]string{
			"gpt-5.1-codex": "xhigh",
		},
		TextVerbosity: "low",
		FastMode:      true,
	}

	req, err := buildProviderRequest(c, upstream, "https://api.example.com", "sk-test", bodyBytes, "gpt-5.1-codex", false)
	if err != nil {
		t.Fatalf("buildProviderRequest() err = %v", err)
	}

	var got map[string]interface{}
	if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
		t.Fatalf("decode request body: %v", err)
	}

	if got["model"] != "gpt-5.2-codex" {
		t.Fatalf("model = %v, want gpt-5.2-codex", got["model"])
	}

	reasoning, ok := got["reasoning"].(map[string]interface{})
	if !ok || reasoning["effort"] != "xhigh" {
		t.Fatalf("reasoning = %#v, want effort=xhigh", got["reasoning"])
	}

	text, ok := got["text"].(map[string]interface{})
	if !ok || text["verbosity"] != "low" {
		t.Fatalf("text = %#v, want verbosity=low", got["text"])
	}

	if got["service_tier"] != "priority" {
		t.Fatalf("service_tier = %v, want priority", got["service_tier"])
	}
}

func TestBuildProviderRequest_PreservesMultimodalContentArray(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		serviceType string
		upstream    *config.UpstreamConfig
		model       string
		wantModel   string
	}{
		{
			name:        "openai_passthrough_keeps_image_url",
			serviceType: "openai",
			upstream: &config.UpstreamConfig{
				ServiceType: "openai",
			},
			model:     "gpt-4o-image",
			wantModel: "gpt-4o-image",
		},
		{
			name:        "responses_passthrough_keeps_image_url",
			serviceType: "responses",
			upstream: &config.UpstreamConfig{
				ServiceType: "responses",
			},
			model:     "gpt-4o-image",
			wantModel: "gpt-4o-image",
		},
		{
			name:        "gemini_passthrough_keeps_image_url_without_remarshal",
			serviceType: "gemini",
			upstream: &config.UpstreamConfig{
				ServiceType: "gemini",
			},
			model:     "gpt-4o-image",
			wantModel: "gpt-4o-image",
		},
		{
			name:        "gemini_passthrough_keeps_image_url_with_remarshal",
			serviceType: "gemini",
			upstream: &config.UpstreamConfig{
				ServiceType: "gemini",
				ModelMapping: map[string]string{
					"gpt-4o-image": "gemini-2.5-flash-image-preview",
				},
			},
			model:     "gpt-4o-image",
			wantModel: "gemini-2.5-flash-image-preview",
		},
	}

	bodyBytes := []byte(`{"model":"gpt-4o-image","messages":[{"role":"user","content":[{"type":"text","text":"修改这个图片"},{"type":"image_url","image_url":{"url":"https://example.com/image.png"}}]}]}`)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil).WithContext(context.Background())

			req, err := buildProviderRequest(c, tt.upstream, "https://api.example.com", "sk-test", bodyBytes, tt.model, false)
			if err != nil {
				t.Fatalf("buildProviderRequest() err = %v", err)
			}

			var got map[string]interface{}
			if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
				t.Fatalf("decode request body: %v", err)
			}

			if got["model"] != tt.wantModel {
				t.Fatalf("model = %v, want %v", got["model"], tt.wantModel)
			}

			messages, ok := got["messages"].([]interface{})
			if !ok || len(messages) != 1 {
				t.Fatalf("messages = %#v, want single message", got["messages"])
			}

			msg, ok := messages[0].(map[string]interface{})
			if !ok {
				t.Fatalf("message[0] = %#v, want object", messages[0])
			}

			content, ok := msg["content"].([]interface{})
			if !ok || len(content) != 2 {
				t.Fatalf("content = %#v, want 2-part array", msg["content"])
			}

			textPart, ok := content[0].(map[string]interface{})
			if !ok || textPart["type"] != "text" || textPart["text"] != "修改这个图片" {
				t.Fatalf("text part = %#v, want text block", content[0])
			}

			imagePart, ok := content[1].(map[string]interface{})
			if !ok || imagePart["type"] != "image_url" {
				t.Fatalf("image part = %#v, want image_url block", content[1])
			}

			imageURL, ok := imagePart["image_url"].(map[string]interface{})
			if !ok || imageURL["url"] != "https://example.com/image.png" {
				t.Fatalf("image_url = %#v, want original url", imagePart["image_url"])
			}
		})
	}
}
