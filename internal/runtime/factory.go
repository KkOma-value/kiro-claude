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
<<<<<<< HEAD
=======
	case "mock":
		log.Infof("Runtime mode: mock")
		return backend.NewMockClient(cfg), nil
>>>>>>> 4d4ec89bd026d39331474aa213c9129cc38e884e
	case "kiro-live":
		log.Infof("Runtime mode: kiro-live")
		client, err := backend.NewLiveClient(cfg, log)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize live backend: %w", err)
		}
<<<<<<< HEAD
		return client, nil
=======
		if cfg.Runtime.AllowStartWithoutToken {
			log.Warnf("Live backend unavailable, falling back to mock mode: %v", err)
			return backend.NewMockClient(cfg), nil
		}
		return nil, err
>>>>>>> 4d4ec89bd026d39331474aa213c9129cc38e884e
	default:
		return nil, fmt.Errorf("unsupported runtime mode: %s (only 'kiro-live' is supported)", cfg.Runtime.Mode)
	}
}
