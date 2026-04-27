package images

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/gin-gonic/gin"
)

func TestExtractOperation(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "/v1/images/generations", want: operationGenerations},
		{path: "/prefix/v1/images/generations", want: operationGenerations},
		{path: "/v1/images/edits", want: operationEdits},
		{path: "/prefix/v1/images/edits", want: operationEdits},
		{path: "/v1/images/variations", want: operationVariations},
		{path: "/v1/images/unknown", want: ""},
	}

	for _, tt := range tests {
		if got := extractOperation(tt.path); got != tt.want {
			t.Fatalf("extractOperation(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestBuildOperationRequest_JSONEdits(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := []byte(`{"model":"image-default","prompt":"add sparkles","stream":true}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstream := &config.UpstreamConfig{
		ServiceType: "openai",
		ModelMapping: map[string]string{
			"image-default": "gpt-image-1.5",
		},
	}
	req, err := buildOperationRequest(c, upstream, "https://api.openai.com", "sk-test", body, "image-default", operationEdits, "application/json")
	if err != nil {
		t.Fatalf("buildOperationRequest() error = %v", err)
	}
	if req.URL.String() != "https://api.openai.com/v1/images/edits" {
		t.Fatalf("unexpected url: %s", req.URL.String())
	}
	if req.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("unexpected content type: %s", req.Header.Get("Content-Type"))
	}

	requestBody, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	if !strings.Contains(string(requestBody), `"model":"gpt-image-1.5"`) {
		t.Fatalf("model mapping was not applied: %s", string(requestBody))
	}
}

func TestBuildOperationRequest_MultipartEdits(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("model", "image-default"); err != nil {
		t.Fatalf("write model: %v", err)
	}
	if err := writer.WriteField("prompt", "add sparkles"); err != nil {
		t.Fatalf("write prompt: %v", err)
	}
	part, err := writer.CreateFormFile("image[]", "input.png")
	if err != nil {
		t.Fatalf("create image part: %v", err)
	}
	if _, err := part.Write([]byte("png-data")); err != nil {
		t.Fatalf("write image part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(body.Bytes()))
	contentType := writer.FormDataContentType()
	c.Request.Header.Set("Content-Type", contentType)

	upstream := &config.UpstreamConfig{
		ServiceType: "openai",
		ModelMapping: map[string]string{
			"image-default": "gpt-image-1.5",
		},
	}
	req, err := buildOperationRequest(c, upstream, "https://api.openai.com#", "sk-test", body.Bytes(), "image-default", operationEdits, contentType)
	if err != nil {
		t.Fatalf("buildOperationRequest() error = %v", err)
	}
	if req.URL.String() != "https://api.openai.com/images/edits" {
		t.Fatalf("unexpected url: %s", req.URL.String())
	}
	if !strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/form-data; boundary=") {
		t.Fatalf("unexpected content type: %s", req.Header.Get("Content-Type"))
	}

	requestBody, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	if !strings.Contains(string(requestBody), "gpt-image-1.5") {
		t.Fatalf("model mapping was not applied: %s", string(requestBody))
	}
	if !strings.Contains(string(requestBody), "png-data") {
		t.Fatalf("file part was not preserved: %s", string(requestBody))
	}
}

func TestBuildOperationRequest_VariationsURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := []byte(`{"n":1,"size":"1024x1024"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/variations", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstream := &config.UpstreamConfig{ServiceType: "openai"}
	req, err := buildOperationRequest(c, upstream, "https://api.openai.com", "sk-test", body, "", operationVariations, "application/json")
	if err != nil {
		t.Fatalf("buildOperationRequest() error = %v", err)
	}
	if req.URL.String() != "https://api.openai.com/v1/images/variations" {
		t.Fatalf("unexpected url: %s", req.URL.String())
	}
}

func TestExtractImagesModelAndStream(t *testing.T) {
	jsonBody := []byte(`{"model":"gpt-image-1.5","stream":true}`)
	if got := extractImagesModel(jsonBody, "application/json"); got != "gpt-image-1.5" {
		t.Fatalf("extractImagesModel json = %q", got)
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(jsonBody))
	if !isImagesStreamRequest(c, jsonBody, "application/json") {
		t.Fatal("expected JSON stream request")
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("model", "gpt-image-1")
	_ = writer.WriteField("stream", "true")
	_ = writer.Close()
	contentType := writer.FormDataContentType()
	if got := extractImagesModel(body.Bytes(), contentType); got != "gpt-image-1" {
		t.Fatalf("extractImagesModel multipart = %q", got)
	}
	if !isImagesStreamRequest(c, body.Bytes(), contentType) {
		t.Fatal("expected multipart stream request")
	}
}
