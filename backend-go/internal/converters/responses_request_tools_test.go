package converters

import (
	"testing"

	"github.com/BenedictKing/ccx/internal/session"
	"github.com/BenedictKing/ccx/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestResponsesToGeminiRequest_PreservesTools(t *testing.T) {
	req := &types.ResponsesRequest{
		Model: "gpt-5",
		Input: "hello",
		Tools: []map[string]interface{}{
			{
				"type":        "function",
				"name":        "get_weather",
				"description": "Get weather",
				"parameters": map[string]interface{}{
					"type": "object",
				},
			},
		},
	}

	geminiReq, err := ResponsesToGeminiRequest(&session.Session{}, req, "gemini-2.5-pro")
	assert.NoError(t, err)
	assert.Len(t, geminiReq.Tools, 1)
	assert.Len(t, geminiReq.Tools[0].FunctionDeclarations, 1)
	assert.Equal(t, "get_weather", geminiReq.Tools[0].FunctionDeclarations[0].Name)
}

func TestOpenAIChatConverter_ToProviderRequest_PreservesTools(t *testing.T) {
	parallelToolCalls := true
	req := &types.ResponsesRequest{
		Model:             "gpt-5",
		Input:             "hello",
		ToolChoice:        "auto",
		ParallelToolCalls: &parallelToolCalls,
		Tools: []map[string]interface{}{
			{
				"type":        "function",
				"name":        "get_weather",
				"description": "Get weather",
				"parameters": map[string]interface{}{
					"type": "object",
				},
			},
		},
	}

	converted, err := (&OpenAIChatConverter{}).ToProviderRequest(&session.Session{}, req)
	assert.NoError(t, err)

	requestMap, ok := converted.(map[string]interface{})
	assert.True(t, ok)
	tools, ok := requestMap["tools"].([]map[string]interface{})
	assert.True(t, ok)
	assert.Len(t, tools, 1)
	assert.Equal(t, "function", tools[0]["type"])
	assert.Equal(t, "auto", requestMap["tool_choice"])
	assert.Equal(t, true, requestMap["parallel_tool_calls"])
}

func TestClaudeConverter_ToProviderRequest_PreservesTools(t *testing.T) {
	req := &types.ResponsesRequest{
		Model: "claude-3-5-sonnet",
		Input: "hello",
		Tools: []map[string]interface{}{
			{
				"type":        "function",
				"name":        "get_weather",
				"description": "Get weather",
				"parameters": map[string]interface{}{
					"type": "object",
				},
			},
		},
	}

	converted, err := (&ClaudeConverter{}).ToProviderRequest(&session.Session{}, req)
	assert.NoError(t, err)

	requestMap, ok := converted.(map[string]interface{})
	assert.True(t, ok)
	tools, ok := requestMap["tools"].([]map[string]interface{})
	assert.True(t, ok)
	assert.Len(t, tools, 1)
	assert.Equal(t, "get_weather", tools[0]["name"])
	assert.NotNil(t, tools[0]["input_schema"])
}

func TestOpenAIChatResponseToResponses_ToolCalls(t *testing.T) {
	openaiResp := map[string]interface{}{
		"model": "gpt-4o",
		"choices": []interface{}{
			map[string]interface{}{
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "I'll call a tool.",
					"tool_calls": []interface{}{
						map[string]interface{}{
							"id":   "call_123",
							"type": "function",
							"function": map[string]interface{}{
								"name":      "get_weather",
								"arguments": `{"location":"NYC"}`,
							},
						},
					},
				},
			},
		},
	}

	resp, err := OpenAIChatResponseToResponses(openaiResp, "sess_test")
	assert.NoError(t, err)
	assert.Len(t, resp.Output, 2)
	assert.Equal(t, "message", resp.Output[0].Type)
	assert.Equal(t, "function_call", resp.Output[1].Type)
	assert.Equal(t, "call_123", resp.Output[1].CallID)
	assert.Equal(t, "get_weather", resp.Output[1].Name)
}
