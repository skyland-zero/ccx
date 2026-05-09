package handlers

import "testing"

func TestShouldRunRedirectVerification(t *testing.T) {
	tests := []struct {
		name               string
		protocols          []string
		sourceTab          string
		channelServiceType string
		want               bool
	}{
		{
			name:               "single unrelated protocol does not run redirect verification",
			protocols:          []string{"messages"},
			sourceTab:          "responses",
			channelServiceType: "chat",
			want:               false,
		},
		{
			name:               "source protocol runs redirect verification",
			protocols:          []string{"responses"},
			sourceTab:          "responses",
			channelServiceType: "chat",
			want:               true,
		},
		{
			name:               "explicit virtual protocol runs redirect verification",
			protocols:          []string{"responses->chat"},
			sourceTab:          "responses",
			channelServiceType: "chat",
			want:               true,
		},
		{
			name:               "other virtual protocol does not run redirect verification",
			protocols:          []string{"messages->chat"},
			sourceTab:          "responses",
			channelServiceType: "chat",
			want:               false,
		},
		{
			name:               "all protocols includes source protocol",
			protocols:          []string{"messages", "responses", "chat", "gemini"},
			sourceTab:          "responses",
			channelServiceType: "chat",
			want:               true,
		},
		{
			name:               "same source and channel protocol never creates same-source virtual protocol",
			protocols:          []string{"responses"},
			sourceTab:          "responses",
			channelServiceType: "responses",
			want:               false,
		},
		{
			name:               "empty source tab does not run redirect verification",
			protocols:          []string{"responses"},
			sourceTab:          "",
			channelServiceType: "chat",
			want:               false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRunRedirectVerification(tt.protocols, tt.sourceTab, tt.channelServiceType)
			if got != tt.want {
				t.Fatalf("shouldRunRedirectVerification(%v, %q, %q) = %v, want %v", tt.protocols, tt.sourceTab, tt.channelServiceType, got, tt.want)
			}
		})
	}
}
