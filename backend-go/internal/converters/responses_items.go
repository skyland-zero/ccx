package converters

// responses_items.go 收敛 Responses canonical item 的共享解析与 provider 转换辅助逻辑。
// 约定内部优先使用 function_call / function_call_output；legacy tool_* 仅通过兼容路径进入。

import (
	"encoding/json"
	"fmt"

	"github.com/BenedictKing/ccx/internal/types"
)

func resolveFunctionCallItem(item types.ResponsesItem) (string, string, string, error) {
	callID := item.CallID
	name := item.Name
	arguments := item.Arguments

	if contentMap, ok := item.Content.(map[string]interface{}); ok {
		if callID == "" {
			callID, _ = contentMap["call_id"].(string)
		}
		if name == "" {
			name, _ = contentMap["name"].(string)
		}
		if arguments == "" {
			arguments, _ = contentMap["arguments"].(string)
		}
		if nestedContent, ok := contentMap["content"].(map[string]interface{}); ok {
			if callID == "" {
				callID, _ = nestedContent["call_id"].(string)
			}
			if name == "" {
				name, _ = nestedContent["name"].(string)
			}
			if arguments == "" {
				arguments, _ = nestedContent["arguments"].(string)
			}
		}
	}

	if name == "" {
		return "", "", "", fmt.Errorf("function_call 缺少 name")
	}
	if callID == "" {
		callID = name
	}

	return callID, name, arguments, nil
}

func resolveFunctionCallOutputItem(item types.ResponsesItem) (string, interface{}, error) {
	callID := item.CallID
	if callID == "" {
		callID = item.Name
	}
	output := item.Output

	if contentMap, ok := item.Content.(map[string]interface{}); ok {
		if callID == "" {
			callID, _ = contentMap["call_id"].(string)
		}
		if callID == "" {
			callID, _ = contentMap["name"].(string)
		}
		if output == nil {
			output = contentMap["output"]
		}
		if nestedContent, ok := contentMap["content"].(map[string]interface{}); ok {
			if callID == "" {
				callID, _ = nestedContent["call_id"].(string)
			}
			if callID == "" {
				callID, _ = nestedContent["name"].(string)
			}
			if output == nil {
				output = nestedContent["output"]
			}
		}
	}

	if callID == "" {
		return "", nil, fmt.Errorf("function_call_output 缺少 call_id")
	}

	return callID, output, nil
}

func parseFunctionCallArguments(arguments string) interface{} {
	input := interface{}(map[string]interface{}{})
	if arguments == "" {
		return input
	}

	var parsed interface{}
	if err := json.Unmarshal([]byte(arguments), &parsed); err == nil {
		return parsed
	}

	return input
}

func resolveResponsesTextItem(item types.ResponsesItem) (string, string) {
	role := item.Role
	if role == "" {
		role = "user"
	}
	return role, extractTextFromContent(item.Content)
}

func normalizeGeminiRole(role string) string {
	if role == "assistant" {
		return "model"
	}
	return role
}

func parseGeminiFunctionCallArgs(arguments string) map[string]interface{} {
	if arguments == "" {
		return nil
	}

	var args map[string]interface{}
	_ = JSONUnmarshal([]byte(arguments), &args)
	return args
}

func buildGeminiFunctionResponsePayload(output interface{}) map[string]interface{} {
	switch value := output.(type) {
	case string:
		return map[string]interface{}{"result": value}
	case map[string]interface{}:
		return value
	default:
		return map[string]interface{}{"result": fmt.Sprintf("%v", output)}
	}
}
