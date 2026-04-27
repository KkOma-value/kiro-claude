package config

import "testing"

func TestLoadConfigAppliesRuntimeEnvOverrides(t *testing.T) {
	t.Setenv("KIRO_CLAUDE_MODE", "kiro-live")
	t.Setenv("KIRO_CLAUDE_MOCK_SCENARIO", "tool-use")
	t.Setenv("KIRO_UPSTREAM_ENDPOINT", "https://example.test")
	t.Setenv("KIRO_ALLOW_START_WITHOUT_TOKEN", "false")

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if cfg.Runtime.Mode != "kiro-live" {
		t.Fatalf("unexpected mode: %s", cfg.Runtime.Mode)
	}
	if cfg.Runtime.MockScenario != "tool-use" {
		t.Fatalf("unexpected mock scenario: %s", cfg.Runtime.MockScenario)
	}
	if cfg.Runtime.UpstreamEndpoint != "https://example.test" {
		t.Fatalf("unexpected endpoint: %s", cfg.Runtime.UpstreamEndpoint)
	}
	if cfg.Runtime.AllowStartWithoutToken {
		t.Fatalf("expected allow_start_without_token to be false")
	}
}
