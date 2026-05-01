package runtime

import (
	"testing"

	"github.com/yourusername/kiro-claude/internal/config"
	"github.com/yourusername/kiro-claude/internal/logger"
)

func TestBuildClientFailsWithoutCredentials(t *testing.T) {
	t.Setenv("KIRO_REFRESH_TOKEN", "")
	t.Setenv("KIRO_REGION", "")

	cfg := config.DefaultConfig()
	cfg.Runtime.Mode = "kiro-live"
	cfg.Kiro.CacheDir = "D:\\nonexistent-kiro-cache"

	result, err := BuildClient(cfg, logger.NewSimpleLogger(logger.LevelError))
	if err == nil {
		result.Client.Close()
		t.Fatalf("expected BuildClient to fail without credentials")
	}
}

func TestBuildClientRejectsUnsupportedMode(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Runtime.Mode = "not-a-real-mode"

	result, err := BuildClient(cfg, logger.NewSimpleLogger(logger.LevelError))
	if err == nil {
		result.Client.Close()
		t.Fatalf("expected BuildClient to reject unsupported mode")
	}
}

func TestBuildClientRejectsMockMode(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Runtime.Mode = "mock"

	result, err := BuildClient(cfg, logger.NewSimpleLogger(logger.LevelError))
	if err == nil {
		result.Client.Close()
		t.Fatalf("expected BuildClient to reject mock mode")
	}
}
