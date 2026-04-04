package common

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNormalizeMetadataUserID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string // expected user_id value after normalization, empty means unchanged
	}{
		{
			name:     "v2.1.78 JSON - only device_id",
			input:    `{"model":"claude-opus-4-6","metadata":{"user_id":"{\"device_id\":\"b854c106939c\",\"account_uuid\":\"\",\"session_id\":\"\"}"},"stream":true}`,
			expected: "user_b854c106939c",
		},
		{
			name:     "v2.1.78 JSON - device_id + session_id",
			input:    `{"model":"claude-opus-4-6","metadata":{"user_id":"{\"device_id\":\"b854c106939c\",\"account_uuid\":\"\",\"session_id\":\"e692f803-4767\"}"}}`,
			expected: "user_b854c106939c_session_e692f803-4767",
		},
		{
			name:     "v2.1.78 JSON - device_id + account_uuid",
			input:    `{"model":"claude-opus-4-6","metadata":{"user_id":"{\"device_id\":\"dev1\",\"account_uuid\":\"acc1\",\"session_id\":\"\"}"}}`,
			expected: "user_dev1_account_acc1",
		},
		{
			name:     "v2.1.78 JSON - all fields populated",
			input:    `{"model":"claude-opus-4-6","metadata":{"user_id":"{\"device_id\":\"abc123\",\"account_uuid\":\"uuid-456\",\"session_id\":\"sess-789\"}"}}`,
			expected: "user_abc123_account_uuid-456_session_sess-789",
		},
		{
			name:     "v2.1.78 JSON - empty device_id, fallback to generic format",
			input:    `{"model":"claude-opus-4-6","metadata":{"user_id":"{\"device_id\":\"\",\"account_uuid\":\"acc1\",\"session_id\":\"sess1\"}"}}`,
			expected: "account_uuid_acc1_session_id_sess1",
		},
		{
			name:     "v2.1.77 flat string user_id - no change",
			input:    `{"model":"claude-opus-4-6","metadata":{"user_id":"user_67bad5_account__session_7581b58b"},"stream":true}`,
			expected: "user_67bad5_account__session_7581b58b",
		},
		{
			name:     "no metadata - no change",
			input:    `{"model":"claude-opus-4-6","stream":true}`,
			expected: "",
		},
		{
			name:     "empty user_id - no change",
			input:    `{"model":"claude-opus-4-6","metadata":{"user_id":""},"stream":true}`,
			expected: "",
		},
		{
			name:     "invalid JSON in user_id - no change",
			input:    `{"model":"claude-opus-4-6","metadata":{"user_id":"{invalid json"}}`,
			expected: "{invalid json",
		},
		{
			name:     "non-claude JSON object - generic key_value format",
			input:    `{"model":"claude-opus-4-6","metadata":{"user_id":"{\"uid\":\"abc123\"}"}}`,
			expected: "uid_abc123",
		},
		{
			name:     "non-claude JSON object - multiple fields sorted",
			input:    `{"model":"claude-opus-4-6","metadata":{"user_id":"{\"session\":\"xyz\",\"uid\":\"abc123\"}"}}`,
			expected: "session_xyz_uid_abc123",
		},
		{
			name:     "preserves other fields",
			input:    `{"model":"claude-opus-4-6","metadata":{"user_id":"{\"device_id\":\"dev1\",\"account_uuid\":\"acc1\",\"session_id\":\"sess1\"}"},"stream":true,"max_tokens":1024}`,
			expected: "user_dev1_account_acc1_session_sess1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeMetadataUserID([]byte(tt.input))

			if tt.expected == "" {
				// Should be unchanged or no user_id
				var data map[string]interface{}
				if err := json.Unmarshal(result, &data); err != nil {
					t.Fatalf("failed to parse result: %v", err)
				}
				metadata, ok := data["metadata"].(map[string]interface{})
				if !ok {
					return // no metadata, as expected
				}
				userID, _ := metadata["user_id"].(string)
				if userID != "" {
					var origData map[string]interface{}
					json.Unmarshal([]byte(tt.input), &origData)
					origMeta, _ := origData["metadata"].(map[string]interface{})
					origUID, _ := origMeta["user_id"].(string)
					if userID != origUID {
						t.Errorf("user_id changed unexpectedly: got %q, want %q", userID, origUID)
					}
				}
				return
			}

			var data map[string]interface{}
			if err := json.Unmarshal(result, &data); err != nil {
				t.Fatalf("failed to parse result: %v", err)
			}
			metadata, ok := data["metadata"].(map[string]interface{})
			if !ok {
				t.Fatal("metadata not found in result")
			}
			userID, ok := metadata["user_id"].(string)
			if !ok {
				t.Fatal("user_id not found in metadata")
			}
			if userID != tt.expected {
				t.Errorf("user_id = %q, want %q", userID, tt.expected)
			}

			// Verify other fields are preserved
			var origData map[string]interface{}
			json.Unmarshal([]byte(tt.input), &origData)
			if origModel, ok := origData["model"].(string); ok {
				if resultModel, ok := data["model"].(string); ok {
					if origModel != resultModel {
						t.Errorf("model changed: got %q, want %q", resultModel, origModel)
					}
				}
			}
		})
	}
}

func TestPassthroughResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	resp := &http.Response{
		StatusCode: http.StatusAccepted,
		Header:     http.Header{"X-Test": []string{"ok"}},
		Body:       io.NopCloser(bytes.NewBufferString(`{"ok":true}`)),
	}

	if err := PassthroughResponse(c, resp); err != nil {
		t.Fatalf("PassthroughResponse() err = %v", err)
	}

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusAccepted)
	}
	if got := w.Header().Get("X-Test"); got != "ok" {
		t.Fatalf("header X-Test = %q, want ok", got)
	}
	if got := w.Body.String(); got != `{"ok":true}` {
		t.Fatalf("body = %q", got)
	}
}

func TestPassthroughJSONResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("decode success", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"X-Test": []string{"ok"}},
			Body:       io.NopCloser(bytes.NewBufferString(`{"usage":{"prompt_tokens":12,"completion_tokens":34}}`)),
		}

		var got map[string]interface{}
		if err := PassthroughJSONResponse(c, resp, &got); err != nil {
			t.Fatalf("PassthroughJSONResponse() err = %v", err)
		}

		if w.Body.String() != `{"usage":{"prompt_tokens":12,"completion_tokens":34}}` {
			t.Fatalf("unexpected body: %q", w.Body.String())
		}
		usage, ok := got["usage"].(map[string]interface{})
		if !ok || usage["prompt_tokens"].(float64) != 12 {
			t.Fatalf("decoded usage = %#v", got["usage"])
		}
	})

	t.Run("decode failure still writes full body", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`{"usage": invalid-json}`)),
		}

		var got map[string]interface{}
		err := PassthroughJSONResponse(c, resp, &got)
		if err == nil {
			t.Fatal("expected decode error")
		}
		var syntaxErr *json.SyntaxError
		if !errors.As(err, &syntaxErr) {
			t.Fatalf("err = %T, want *json.SyntaxError", err)
		}
		if w.Body.String() != `{"usage": invalid-json}` {
			t.Fatalf("unexpected body: %q", w.Body.String())
		}
	})
}

func TestSanitizeMalformedThinkingBlocks(t *testing.T) {
	input := `{
		"messages": [
			{
				"role": "assistant",
				"content": [
					{"type": "thinking"},
					{"type": "text", "text": "ok"}
				]
			},
			{
				"role": "assistant",
				"content": [
					{"type": "thinking", "thinking": {"foo": "bar"}}
				]
			},
			{
				"role": "assistant",
				"content": [
					{"type": "thinking", "thinking": "keep me"}
				]
			},
			{
				"role": "user",
				"content": [
					{"type": "text", "text": "hello"}
				]
			}
		]
	}`

	gotBytes, modified := SanitizeMalformedThinkingBlocks([]byte(input), false, "Messages")
	if !modified {
		t.Fatal("expected modified=true")
	}

	var got map[string]interface{}
	if err := json.Unmarshal(gotBytes, &got); err != nil {
		t.Fatalf("unmarshal result failed: %v", err)
	}

	messages, ok := got["messages"].([]interface{})
	if !ok {
		t.Fatalf("messages type = %T, want []interface{}", got["messages"])
	}

	// 仅含 thinking 的 assistant 消息保留骨架（content 清空），不删除整条消息，共 4 条
	if len(messages) != 4 {
		t.Fatalf("messages len = %d, want 4", len(messages))
	}

	firstMsg, _ := messages[0].(map[string]interface{})
	firstContent, _ := firstMsg["content"].([]interface{})
	if len(firstContent) != 1 {
		t.Fatalf("first message content len = %d, want 1", len(firstContent))
	}
	firstBlock, _ := firstContent[0].(map[string]interface{})
	if firstBlock["type"] != "text" {
		t.Fatalf("first message content[0].type = %v, want text", firstBlock["type"])
	}

	// 第二条：仅含 thinking，保留骨架 content=[]
	secondMsg, _ := messages[1].(map[string]interface{})
	secondContent, _ := secondMsg["content"].([]interface{})
	if len(secondContent) != 0 {
		t.Fatalf("second message content len = %d, want 0 (thinking-only, kept as empty)", len(secondContent))
	}

	// 第三条：同上
	thirdMsg, _ := messages[2].(map[string]interface{})
	thirdContent, _ := thirdMsg["content"].([]interface{})
	if len(thirdContent) != 0 {
		t.Fatalf("third message content len = %d, want 0", len(thirdContent))
	}

	// 最后一条：user 文本消息
	lastMsg, _ := messages[3].(map[string]interface{})
	if lastMsg["role"] != "user" {
		t.Fatalf("last message role = %v, want user", lastMsg["role"])
	}
}

func TestSanitizeMalformedThinkingBlocks_InvalidJSON_NoChange(t *testing.T) {
	input := []byte(`{"messages":[`)
	got, modified := SanitizeMalformedThinkingBlocks(input, false, "Messages")
	if modified {
		t.Fatal("expected modified=false")
	}
	if string(got) != string(input) {
		t.Fatalf("unexpected output change: got %q, want %q", string(got), string(input))
	}
}

func TestSanitizeMalformedThinkingBlocks_ContentObject(t *testing.T) {
	input := `{
		"messages": [
			{
				"role": "assistant",
				"content": {"type": "thinking", "thinking": {}}
			},
			{
				"role": "assistant",
				"content": {"type": "text", "text": "ok", "thinking": {"thinking": "noise"}}
			}
		]
	}`

	gotBytes, modified := SanitizeMalformedThinkingBlocks([]byte(input), false, "Messages")
	if !modified {
		t.Fatal("expected modified=true")
	}

	var got map[string]interface{}
	if err := json.Unmarshal(gotBytes, &got); err != nil {
		t.Fatalf("unmarshal result failed: %v", err)
	}

	messages, _ := got["messages"].([]interface{})
	if len(messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(messages))
	}

	msg, _ := messages[0].(map[string]interface{})
	content, _ := msg["content"].(map[string]interface{})
	if _, exists := content["thinking"]; exists {
		t.Fatalf("unexpected thinking field in non-thinking content: %v", content["thinking"])
	}
	if content["type"] != "text" {
		t.Fatalf("content.type = %v, want text", content["type"])
	}
}
