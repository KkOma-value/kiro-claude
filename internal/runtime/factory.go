package runtime

import (
	"fmt"
	"strings"

	"github.com/yourusername/kiro-claude/internal/auth"
	"github.com/yourusername/kiro-claude/internal/backend"
	"github.com/yourusername/kiro-claude/internal/config"
	"github.com/yourusername/kiro-claude/internal/logger"
)

// BuildResult holds the backend client and supporting components.
type BuildResult struct {
	Client    backend.Client
	TokenMgr  *auth.TokenManager // nil when running without credentials
}

// BuildClient creates the configured backend client and returns supporting components.
func BuildClient(cfg *config.Config, log logger.Logger) (*BuildResult, error) {
	mode := strings.ToLower(strings.TrimSpace(cfg.Runtime.Mode))
	if mode == "" {
		mode = "kiro-live"
	}

	switch mode {
	case "kiro-live":
		log.Infof("Runtime mode: kiro-live")
		client, tokenMgr, err := backend.NewLiveClientWithTokenMgr(cfg, log)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize live backend: %w", err)
		}
		return &BuildResult{Client: client, TokenMgr: tokenMgr}, nil
	default:
		return nil, fmt.Errorf("unsupported runtime mode: %s (only 'kiro-live' is supported)", cfg.Runtime.Mode)
	}
}
