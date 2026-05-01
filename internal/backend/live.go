package backend

import (
	"context"
	"fmt"

	"github.com/yourusername/kiro-claude/internal/auth"
	"github.com/yourusername/kiro-claude/internal/config"
	"github.com/yourusername/kiro-claude/internal/gateway"
	"github.com/yourusername/kiro-claude/internal/logger"
)

// LiveClient forwards requests to the real Kiro upstream.
type LiveClient struct {
	tokenMgr *auth.TokenManager
	client   *gateway.Client
}

// NewLiveClient creates a live Kiro-backed backend.
func NewLiveClient(cfg *config.Config, log logger.Logger) (*LiveClient, error) {
	client, _, err := NewLiveClientWithTokenMgr(cfg, log)
	return client, err
}

// NewLiveClientWithTokenMgr creates a live Kiro-backed backend and returns the token manager.
func NewLiveClientWithTokenMgr(cfg *config.Config, log logger.Logger) (*LiveClient, *auth.TokenManager, error) {
	log.Infof("Loading Kiro credentials from %s", cfg.Kiro.CacheDir)
	tokenMgr, err := auth.NewTokenManager(cfg.Kiro.CacheDir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	status := tokenMgr.Status()
	log.Infof("Loaded %d Kiro credential(s), auth=%s, region=%s", status.TotalAccounts, status.AuthMethod, status.Region)
	log.Infof("Using credential %d/%d, expires at %s", status.CurrentIndex+1, status.TotalAccounts, status.ExpiresAt)

	client := &LiveClient{
		tokenMgr: tokenMgr,
		client:   gateway.NewClient(tokenMgr, cfg.Proxy, log),
	}
	return client, tokenMgr, nil
}

func (c *LiveClient) SendRequest(ctx context.Context, req *gateway.KiroRequest) (string, []gateway.KiroStreamEvent, error) {
	return c.client.SendRequest(ctx, req)
}

func (c *LiveClient) SendStreamRequest(ctx context.Context, req *gateway.KiroRequest) (<-chan gateway.KiroStreamEvent, <-chan error, error) {
	return c.client.SendStreamRequest(ctx, req)
}

func (c *LiveClient) Models() []gateway.ModelInfo {
	return c.client.Models()
}

func (c *LiveClient) Close() error {
	if c.tokenMgr != nil {
		c.tokenMgr.Close()
	}
	return nil
}
