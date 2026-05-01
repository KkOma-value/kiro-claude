package runtime

import (
	"fmt"
	"strings"

	"github.com/yourusername/kiro-claude/internal/backend"
	"github.com/yourusername/kiro-claude/internal/config"
	"github.com/yourusername/kiro-claude/internal/logger"
)

// BuildClient creates the configured backend client.
func BuildClient(cfg *config.Config, log logger.Logger) (backend.Client, error) {
	mode := strings.ToLower(strings.TrimSpace(cfg.Runtime.Mode))
	if mode == "" {
		mode = "kiro-live"
	}

	switch mode {
	case "kiro-live":
		log.Infof("Runtime mode: kiro-live")
		log.Infof("Active backend: %s", cfg.Runtime.UpstreamEndpoint)
		client, err := backend.NewLiveClient(cfg, log)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize live backend: %w", err)
		}
		return client, nil
	default:
		return nil, fmt.Errorf("unsupported runtime mode: %s (only 'kiro-live' is supported)", cfg.Runtime.Mode)
	}
}
