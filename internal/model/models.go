package model

// ModelInfo represents model metadata.
type ModelInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

// SupportedModels returns the public Anthropic model ids available through the proxy.
func SupportedModels() []ModelInfo {
	return []ModelInfo{
		{ID: "claude-opus-4.7", Name: "Claude Opus 4.7", Provider: "anthropic"},
		{ID: "claude-sonnet-4.5", Name: "Claude Sonnet 4.5", Provider: "anthropic"},
		{ID: "claude-sonnet-4", Name: "Claude Sonnet 4", Provider: "anthropic"},
		{ID: "claude-haiku-4.5", Name: "Claude Haiku 4.5", Provider: "anthropic"},
		{ID: "claude-3.7-sonnet", Name: "Claude 3.7 Sonnet", Provider: "anthropic"},
		{ID: "claude-3.5-sonnet", Name: "Claude 3.5 Sonnet", Provider: "anthropic"},
		{ID: "claude-3.5-haiku", Name: "Claude 3.5 Haiku", Provider: "anthropic"},
		{ID: "claude-3-haiku", Name: "Claude 3 Haiku", Provider: "anthropic"},
	}
}

// ForwardModelMap maps public Anthropic model names (normalized, with dots)
// to CodeWhisperer internal IDs for the Kiro upstream API.
var ForwardModelMap = map[string]string{
	"claude-opus-4.7":   "CLAUDE_OPUS_4_7_20250219_V1_0",
	"claude-opus-4.5":   "CLAUDE_OPUS_4_5_20250805_V1_0",
	"claude-opus-4":     "CLAUDE_OPUS_4_20250805_V1_0",
	"claude-sonnet-4.5": "CLAUDE_SONNET_4_5_20250514_V1_0",
	"claude-sonnet-4":   "CLAUDE_SONNET_4_20250514_V1_0",
	"claude-haiku-4.5":  "CLAUDE_HAIKU_4_5_20251001_V1_0",
	"claude-3.7-sonnet": "CLAUDE_3_7_SONNET_20250219_V1_0",
	"claude-3.5-sonnet": "CLAUDE_3_5_SONNET_20241022_V1_0",
	"claude-3.5-haiku":  "CLAUDE_3_5_HAIKU_20241022_V1_0",
	"claude-3-sonnet":   "CLAUDE_3_SONNET_20240229_V1_0",
	"claude-3-haiku":    "CLAUDE_3_HAIKU_20240307_V1_0",
	"claude-instant-1.2": "CLAUDE_INSTANT_1_2_V1_0",

	// Legacy versioned names (direct match before normalization)
	"claude-opus-4-20250805":     "CLAUDE_OPUS_4_20250805_V1_0",
	"claude-sonnet-4-20250514":   "CLAUDE_SONNET_4_20250514_V1_0",
	"claude-sonnet-4-20250102":   "CLAUDE_SONNET_4_20250102_V1_0",
	"claude-sonnet-4-20240229":   "CLAUDE_SONNET_4_20240229_V1_0",
	"claude-3-5-sonnet-20241022": "CLAUDE_3_5_SONNET_20241022_V1_0",
	"claude-3-5-sonnet-20240620": "CLAUDE_3_5_SONNET_20240620_V1_0",
	"claude-3-5-haiku-20241022":  "CLAUDE_3_5_HAIKU_20241022_V1_0",
	"claude-3-haiku-20240307":    "CLAUDE_3_HAIKU_20240307_V1_0",
	"claude-sonnet-4.5-20250514": "CLAUDE_SONNET_4_5_20250514_V1_0",
}

// ReverseModelMap maps CodeWhisperer IDs back to public Anthropic names.
var ReverseModelMap = map[string]string{
	"CLAUDE_OPUS_4_7_20250219_V1_0":   "claude-opus-4.7",
	"CLAUDE_OPUS_4_5_20250805_V1_0":   "claude-opus-4.5",
	"CLAUDE_OPUS_4_20250805_V1_0":     "claude-opus-4",
	"CLAUDE_SONNET_4_5_20250514_V1_0": "claude-sonnet-4.5",
	"CLAUDE_SONNET_4_20250514_V1_0":   "claude-sonnet-4",
	"CLAUDE_SONNET_4_20250102_V1_0":   "claude-sonnet-4",
	"CLAUDE_SONNET_4_20240229_V1_0":   "claude-sonnet-4",
	"CLAUDE_3_7_SONNET_20250219_V1_0": "claude-3.7-sonnet",
	"CLAUDE_3_5_SONNET_20241022_V1_0": "claude-3.5-sonnet",
	"CLAUDE_3_5_SONNET_20240620_V1_0": "claude-3.5-sonnet",
	"CLAUDE_3_SONNET_20240229_V1_0":   "claude-3-sonnet",
	"CLAUDE_3_5_HAIKU_20241022_V1_0":  "claude-3.5-haiku",
	"CLAUDE_3_HAIKU_20240307_V1_0":    "claude-3-haiku",
	"CLAUDE_INSTANT_1_2_V1_0":         "claude-instant-1.2",
}
