package runtime

import (
	"testing"

	"github.com/yourusername/kiro-claude/internal/config"
	"github.com/yourusername/kiro-claude/internal/logger"
)

func TestBuildClientUsesMockModeWithoutCredentials(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Runtime.Mode = "mock"

	client, err := BuildClient(cfg, logger.NewSimpleLogger(logger.LevelError))
	if err != nil {
		t.Fatalf("BuildClient returned error: %v", err)
	}
	defer client.Close()
}

func TestBuildClientFallsBackToMockWhenAllowed(t *testing.T) {
	t.Setenv("KIRO_REFRESH_TOKEN", "")
	t.Setenv("KIRO_REGION", "")

	cfg := config.DefaultConfig()
	cfg.Runtime.Mode = "kiro-live"
	cfg.Runtime.AllowStartWithoutToken = true
	cfg.Kiro.CacheDir = "D:\\nonexistent-kiro-cache"

	client, err := BuildClient(cfg, logger.NewSimpleLogger(logger.LevelError))
	if err != nil {
		t.Fatalf("BuildClient returned error: %v", err)
	}
	defer client.Close()
}

func TestBuildClientFailsWithoutFallback(t *testing.T) {
	t.Setenv("KIRO_REFRESH_TOKEN", "")
	t.Setenv("KIRO_REGION", "")

	cfg := config.DefaultConfig()
	cfg.Runtime.Mode = "kiro-live"
	cfg.Runtime.AllowStartWithoutToken = false
	cfg.Kiro.CacheDir = "D:\\nonexistent-kiro-cache"

	client, err := BuildClient(cfg, logger.NewSimpleLogger(logger.LevelError))
	if err == nil {
		client.Close()
		t.Fatalf("expected BuildClient to fail without fallback")
	}
}

func TestBuildClientRejectsUnsupportedMode(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Runtime.Mode = "not-a-real-mode"

	client, err := BuildClient(cfg, logger.NewSimpleLogger(logger.LevelError))
	if err == nil {
		client.Close()
		t.Fatalf("expected BuildClient to reject unsupported mode")
	}
}
