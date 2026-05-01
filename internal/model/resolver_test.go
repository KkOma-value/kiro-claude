package model

import "testing"

func TestNormalizeModelName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Standard: claude-{family}-{major}-{minor}
		{"claude-sonnet-4-5", "claude-sonnet-4.5"},
		{"claude-haiku-4-5", "claude-haiku-4.5"},
		{"claude-opus-4-5", "claude-opus-4.5"},
		{"claude-opus-4-7", "claude-opus-4.7"},

		// Standard with date suffix
		{"claude-sonnet-4-5-20250929", "claude-sonnet-4.5"},
		{"claude-haiku-4-5-20251001", "claude-haiku-4.5"},
		{"claude-opus-4-7-20250219", "claude-opus-4.7"},

		// Standard with latest suffix
		{"claude-sonnet-4-5-latest", "claude-sonnet-4.5"},

		// No minor version
		{"claude-sonnet-4", "claude-sonnet-4"},
		{"claude-opus-4", "claude-opus-4"},

		// No minor with date
		{"claude-sonnet-4-20250514", "claude-sonnet-4"},

		// Legacy: claude-{major}-{minor}-{family}
		{"claude-3-7-sonnet", "claude-3.7-sonnet"},
		{"claude-3-5-sonnet", "claude-3.5-sonnet"},
		{"claude-3-5-haiku", "claude-3.5-haiku"},

		// Legacy with date
		{"claude-3-7-sonnet-20250219", "claude-3.7-sonnet"},
		{"claude-3-5-sonnet-20241022", "claude-3.5-sonnet"},
		{"claude-3-5-haiku-20241022", "claude-3.5-haiku"},

		// Already normalized with dot + date
		{"claude-sonnet-4.5-20250514", "claude-sonnet-4.5"},

		// Inverted with suffix
		{"claude-4.5-opus-high", "claude-opus-4.5"},
		{"claude-4.5-sonnet-low", "claude-sonnet-4.5"},

		// Already correct (passthrough)
		{"claude-sonnet-4.5", "claude-sonnet-4.5"},
		{"claude-opus-4.7", "claude-opus-4.7"},
		{"claude-3.7-sonnet", "claude-3.7-sonnet"},

		// Non-claude models (passthrough)
		{"gpt-4", "gpt-4"},
		{"auto", "auto"},

		// Empty
		{"", ""},

		// Case insensitive
		{"Claude-Sonnet-4-5", "claude-sonnet-4.5"},
		{"CLAUDE-OPUS-4-7", "claude-opus-4.7"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeModelName(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeModelName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolverStaticLookup(t *testing.T) {
	r := NewResolver()

	tests := []struct {
		input      string
		wantSource string
		wantID     string
	}{
		{"claude-sonnet-4.5", "static", "CLAUDE_SONNET_4_5_20250514_V1_0"},
		{"claude-sonnet-4-5", "static", "CLAUDE_SONNET_4_5_20250514_V1_0"},
		{"claude-sonnet-4-5-20250929", "static", "CLAUDE_SONNET_4_5_20250514_V1_0"},
		{"claude-3-5-sonnet-20241022", "static", "CLAUDE_3_5_SONNET_20241022_V1_0"},
		{"claude-3-7-sonnet", "static", "CLAUDE_3_7_SONNET_20250219_V1_0"},
		{"claude-opus-4.7", "static", "CLAUDE_OPUS_4_7_20250219_V1_0"},
		{"claude-opus-4-7", "static", "CLAUDE_OPUS_4_7_20250219_V1_0"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			res := r.Resolve(tt.input)
			if res.Source != tt.wantSource {
				t.Errorf("Resolve(%q).Source = %q, want %q", tt.input, res.Source, tt.wantSource)
			}
			if res.InternalID != tt.wantID {
				t.Errorf("Resolve(%q).InternalID = %q, want %q", tt.input, res.InternalID, tt.wantID)
			}
		})
	}
}

func TestResolverPassthrough(t *testing.T) {
	r := NewResolver()

	res := r.Resolve("my-custom-model")
	if res.Source != "passthrough" {
		t.Errorf("Resolve(unknown).Source = %q, want %q", res.Source, "passthrough")
	}
	if res.InternalID != "my-custom-model" {
		t.Errorf("Resolve(unknown).InternalID = %q, want %q", res.InternalID, "my-custom-model")
	}
}

func TestResolverReverseResolve(t *testing.T) {
	r := NewResolver()

	tests := []struct {
		kiroModel string
		want      string
	}{
		{"CLAUDE_SONNET_4_5_20250514_V1_0", "claude-sonnet-4.5"},
		{"CLAUDE_OPUS_4_7_20250219_V1_0", "claude-opus-4.7"},
		{"UNKNOWN_MODEL", "UNKNOWN_MODEL"},
	}

	for _, tt := range tests {
		t.Run(tt.kiroModel, func(t *testing.T) {
			got := r.ReverseResolve(tt.kiroModel)
			if got != tt.want {
				t.Errorf("ReverseResolve(%q) = %q, want %q", tt.kiroModel, got, tt.want)
			}
		})
	}
}
