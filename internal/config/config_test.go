package config

import "testing"

func TestLoadConfigAppliesRuntimeEnvOverrides(t *testing.T) {
	t.Setenv("KIRO_CLAUDE_MODE", "kiro-live")
	t.Setenv("KIRO_UPSTREAM_ENDPOINT", "https://example.test")

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if cfg.Runtime.Mode != "kiro-live" {
		t.Fatalf("unexpected mode: %s", cfg.Runtime.Mode)
	}
	if cfg.Runtime.UpstreamEndpoint != "https://example.test" {
		t.Fatalf("unexpected endpoint: %s", cfg.Runtime.UpstreamEndpoint)
	}
}

func TestDefaultConfigUsesKiroLive(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Runtime.Mode != "kiro-live" {
		t.Fatalf("expected default mode to be kiro-live, got: %s", cfg.Runtime.Mode)
	}
}
