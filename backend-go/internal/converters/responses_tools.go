package converters

import "github.com/BenedictKing/ccx/internal/types"

func defaultResponsesToolParameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func sanitizeResponsesToolParameters(parameters interface{}) interface{} {
	paramMap, ok := parameters.(map[string]interface{})
	if !ok {
		return defaultResponsesToolParameters()
	}
	if paramType, ok := paramMap["type"].(string); !ok || paramType == "" {
		paramMap["type"] = "object"
	}
	if _, ok := paramMap["properties"].(map[string]interface{}); !ok {
		paramMap["properties"] = map[string]interface{}{}
	}
	return paramMap
}

func extractResponsesToolFields(tool map[string]interface{}) (string, string, interface{}) {
	name, _ := tool["name"].(string)
	description, _ := tool["description"].(string)
	parameters := tool["parameters"]

	if function, ok := tool["function"].(map[string]interface{}); ok {
		if name == "" {
			name, _ = function["name"].(string)
		}
		if description == "" {
			description, _ = function["description"].(string)
		}
		if parameters == nil {
			parameters = function["parameters"]
		}
	}

	if parameters == nil {
		parameters = defaultResponsesToolParameters()
	} else {
		parameters = sanitizeResponsesToolParameters(parameters)
	}

	return name, description, parameters
}

func responsesToolsToOpenAI(tools []map[string]interface{}) []map[string]interface{} {
	openaiTools := make([]map[string]interface{}, 0, len(tools))
	for _, tool := range tools {
		name, description, parameters := extractResponsesToolFields(tool)
		if name == "" {
			continue
		}
		openaiTool := map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":       name,
				"parameters": parameters,
			},
		}
		if description != "" {
			openaiTool["function"].(map[string]interface{})["description"] = description
		}
		openaiTools = append(openaiTools, openaiTool)
	}
	return openaiTools
}

func responsesToolsToClaude(tools []map[string]interface{}) []map[string]interface{} {
	claudeTools := make([]map[string]interface{}, 0, len(tools))
	for _, tool := range tools {
		name, description, parameters := extractResponsesToolFields(tool)
		if name == "" {
			continue
		}
		claudeTool := map[string]interface{}{
			"name":         name,
			"input_schema": parameters,
		}
		if description != "" {
			claudeTool["description"] = description
		}
		claudeTools = append(claudeTools, claudeTool)
	}
	return claudeTools
}

func responsesToolsToGemini(tools []map[string]interface{}) []types.GeminiTool {
	declarations := make([]types.GeminiFunctionDeclaration, 0, len(tools))
	for _, tool := range tools {
		name, description, parameters := extractResponsesToolFields(tool)
		if name == "" {
			continue
		}
		declarations = append(declarations, types.GeminiFunctionDeclaration{
			Name:        name,
			Description: description,
			Parameters:  parameters,
		})
	}
	if len(declarations) == 0 {
		return nil
	}
	return []types.GeminiTool{{FunctionDeclarations: declarations}}
}
