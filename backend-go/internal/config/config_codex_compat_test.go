package config

import (
	"testing"
)

func codexCompatBoolPtr(b bool) *bool {
	return &b
}

func TestCodexToolsCompatPriority(t *testing.T) {
	tests := []struct {
		name            string
		codexToolsCompat *bool
		codexToolCompat  *bool
		want            bool
	}{
		{
			name:            "only codexToolsCompat true",
			codexToolsCompat: codexCompatBoolPtr(true),
			codexToolCompat:  nil,
			want:            true,
		},
		{
			name:            "only codexToolsCompat false",
			codexToolsCompat: codexCompatBoolPtr(false),
			codexToolCompat:  nil,
			want:            false,
		},
		{
			name:            "only codexToolCompat true (backward compat)",
			codexToolsCompat: nil,
			codexToolCompat:  codexCompatBoolPtr(true),
			want:            true,
		},
		{
			name:            "only codexToolCompat false (backward compat)",
			codexToolsCompat: nil,
			codexToolCompat:  codexCompatBoolPtr(false),
			want:            false,
		},
		{
			name:            "both set, codexToolsCompat takes priority when true",
			codexToolsCompat: codexCompatBoolPtr(true),
			codexToolCompat:  codexCompatBoolPtr(false),
			want:            true,
		},
		{
			name:            "both set, codexToolsCompat takes priority when false",
			codexToolsCompat: codexCompatBoolPtr(false),
			codexToolCompat:  codexCompatBoolPtr(true),
			want:            false,
		},
		{
			name:            "both nil, defaults to false",
			codexToolsCompat: nil,
			codexToolCompat:  nil,
			want:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			up := &UpstreamConfig{
				CodexToolsCompat: tt.codexToolsCompat,
				CodexToolCompat:  tt.codexToolCompat,
			}
			if got := up.IsCodexToolsCompatEnabled(); got != tt.want {
				t.Errorf("IsCodexToolsCompatEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}
