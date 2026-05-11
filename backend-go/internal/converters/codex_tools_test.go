package converters

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/BenedictKing/ccx/internal/types"
)

func TestBuildApplyPatchInput_AddFileDoesNotAddExtraBlankLine(t *testing.T) {
	got := BuildApplyPatchInput([]ApplyPatchOperation{{
		Type:    "add_file",
		Path:    "docs/test.md",
		Content: "# Test\n",
	}})
	want := "*** Begin Patch\n*** Add File: docs/test.md\n+# Test\n*** End Patch"
	if got != want {
		t.Fatalf("patch mismatch\n got: %q\nwant: %q", got, want)
	}
}

func TestRemapCustomToolCallsInResponseWritesInputField(t *testing.T) {
	ctx := BuildCodexToolContext([]map[string]interface{}{{
		"type": "custom",
		"name": "apply_patch",
	}})
	resp := &types.ResponsesResponse{
		Output: []types.ResponsesItem{{
			Type:      "function_call",
			CallID:    "call_1",
			Name:      "apply_patch_add_file",
			Arguments: `{"path":"docs/test.md","content":"# Test\n"}`,
		}},
	}

	ctx.RemapCustomToolCallsInResponse(resp)

	if len(resp.Output) != 1 {
		t.Fatalf("output len = %d", len(resp.Output))
	}
	item := resp.Output[0]
	if item.Type != "custom_tool_call" || item.Name != "apply_patch" {
		t.Fatalf("item = %#v", item)
	}
	if item.Input != "*** Begin Patch\n*** Add File: docs/test.md\n+# Test\n*** End Patch" {
		t.Fatalf("input = %q", item.Input)
	}
	if item.Output != nil {
		t.Fatalf("custom tool call should not use output field: %#v", item.Output)
	}

	b, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"input"`) || strings.Contains(string(b), `"output"`) {
		t.Fatalf("marshaled item = %s", b)
	}
}
