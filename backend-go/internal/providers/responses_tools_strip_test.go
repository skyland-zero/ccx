package providers

import (
	"encoding/json"
	"testing"
)

func TestStripCodexClientOnlyTools(t *testing.T) {
	t.Run("剥离字符串简写并保留 function 对象", func(t *testing.T) {
		req := map[string]interface{}{
			"tools": []interface{}{
				"exec_command",
				"apply_patch",
				map[string]interface{}{
					"type": "function",
					"function": map[string]interface{}{
						"name": "lookup_user",
					},
				},
			},
			"tool_choice":         "auto",
			"parallel_tool_calls": true,
		}
		stripCodexClientOnlyTools(req)

		tools, ok := req["tools"].([]interface{})
		if !ok {
			t.Fatalf("tools 被误删，期望保留 function 条目")
		}
		if len(tools) != 1 {
			t.Fatalf("tools 长度=%d，期望 1", len(tools))
		}
		if _, ok := tools[0].(map[string]interface{}); !ok {
			t.Fatalf("剩余条目类型错误: %T", tools[0])
		}
		if req["tool_choice"] != "auto" {
			t.Fatalf("tool_choice 不应被删除")
		}
	})

	t.Run("全部剥离时同步清理 tool_choice 与 parallel_tool_calls", func(t *testing.T) {
		req := map[string]interface{}{
			"tools": []interface{}{
				"exec_command",
				map[string]interface{}{"type": "web_search"},
				map[string]interface{}{"type": "namespace", "name": "mcp__chrome_devtools__"},
				map[string]interface{}{"type": "custom", "name": "apply_patch"},
			},
			"tool_choice":         "auto",
			"parallel_tool_calls": true,
		}
		stripCodexClientOnlyTools(req)

		if _, ok := req["tools"]; ok {
			t.Fatalf("tools 应当被删除")
		}
		if _, ok := req["tool_choice"]; ok {
			t.Fatalf("tool_choice 应当被删除")
		}
		if _, ok := req["parallel_tool_calls"]; ok {
			t.Fatalf("parallel_tool_calls 应当被删除")
		}
	})

	t.Run("未知对象类型保守保留", func(t *testing.T) {
		req := map[string]interface{}{
			"tools": []interface{}{
				map[string]interface{}{"type": "something_new"},
			},
		}
		stripCodexClientOnlyTools(req)
		tools, ok := req["tools"].([]interface{})
		if !ok || len(tools) != 1 {
			t.Fatalf("未知类型应被保留，当前=%v", req["tools"])
		}
	})

	t.Run("无 tools 字段不报错", func(t *testing.T) {
		req := map[string]interface{}{"model": "gpt-5.5"}
		stripCodexClientOnlyTools(req)
		if _, ok := req["tools"]; ok {
			t.Fatalf("不应注入 tools")
		}
	})

	t.Run("部分剥离时修正指向已删除工具的 tool_choice", func(t *testing.T) {
		req := map[string]interface{}{
			"tools": []interface{}{
				map[string]interface{}{"type": "custom", "name": "apply_patch"},
				map[string]interface{}{"type": "function", "function": map[string]interface{}{"name": "lookup_user"}},
			},
			"tool_choice": map[string]interface{}{"type": "custom", "name": "apply_patch"},
		}
		stripCodexClientOnlyTools(req)

		if req["tool_choice"] != "auto" {
			t.Fatalf("tool_choice=%v，期望 auto", req["tool_choice"])
		}
	})

	t.Run("部分剥离时保留仍有效的 tool_choice", func(t *testing.T) {
		req := map[string]interface{}{
			"tools": []interface{}{
				map[string]interface{}{"type": "custom", "name": "apply_patch"},
				map[string]interface{}{"type": "function", "function": map[string]interface{}{"name": "lookup_user"}},
			},
			"tool_choice": map[string]interface{}{"type": "function", "function": map[string]interface{}{"name": "lookup_user"}},
		}
		stripCodexClientOnlyTools(req)

		choice, ok := req["tool_choice"].(map[string]interface{})
		if !ok {
			t.Fatalf("tool_choice 应保持对象，当前=%v", req["tool_choice"])
		}
		if extractToolChoiceName(choice) != "lookup_user" {
			t.Fatalf("tool_choice 指向错误: %v", choice)
		}
	})
}

func TestStripCodexClientOnlyToolsFromBody(t *testing.T) {
	body := []byte(`{"model":"gpt-5.5","tools":["exec_command",{"type":"namespace","name":"mcp__chrome_devtools__"},{"type":"function","function":{"name":"lookup_user","parameters":{"type":"object","properties":{}}}}],"tool_choice":"auto"}`)
	updated := stripCodexClientOnlyToolsFromBody(body)

	var req map[string]interface{}
	if err := json.Unmarshal(updated, &req); err != nil {
		t.Fatalf("剥离后的 body 应保持合法 JSON: %v", err)
	}
	tools, ok := req["tools"].([]interface{})
	if !ok {
		t.Fatalf("应保留 function 工具，当前=%v", req["tools"])
	}
	if len(tools) != 1 {
		t.Fatalf("tools 长度=%d，期望 1", len(tools))
	}
	if req["tool_choice"] != "auto" {
		t.Fatalf("仍有 function 工具时不应删除 tool_choice")
	}
}
