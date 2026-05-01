package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// KiroDesktopProvider implements Provider for Kiro IDE Desktop auth.
// Endpoint: https://prod.{region}.auth.desktop.kiro.dev/refreshToken
type KiroDesktopProvider struct {
	refreshToken string
	profileArn   string
	region       string
	accessToken  string
	expiresAt    time.Time
	mu           sync.RWMutex
	client       *http.Client
	stopChan     chan struct{}
}

// NewKiroDesktopProvider creates a provider from a Kiro Desktop credential.
func NewKiroDesktopProvider(cred *Credential) *KiroDesktopProvider {
	region := cred.Region
	if region == "" {
		region = "us-east-1"
	}

	p := &KiroDesktopProvider{
		refreshToken: cred.RefreshToken,
		profileArn:   cred.ProfileArn,
		region:       region,
		accessToken:  cred.AccessToken,
		expiresAt:    cred.ExpiresAt,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		stopChan: make(chan struct{}),
	}

	go p.backgroundRefresh()
	return p
}

func (p *KiroDesktopProvider) Type() AuthType { return AuthKiroDesktop }

func (p *KiroDesktopProvider) GetAccessToken(ctx context.Context) (string, error) {
	p.mu.RLock()
	token := p.accessToken
	expired := p.IsExpiredLocked()
	p.mu.RUnlock()

	if expired {
		if err := p.Refresh(ctx); err != nil {
			return "", err
		}
		p.mu.RLock()
		token = p.accessToken
		p.mu.RUnlock()
	}
	return token, nil
}

func (p *KiroDesktopProvider) Refresh(ctx context.Context) error {
	endpoint := fmt.Sprintf("https://prod.%s.auth.desktop.kiro.dev/refreshToken", p.region)

	reqBody := map[string]interface{}{
		"refreshToken": p.refreshToken,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal refresh request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("refresh failed with status %d", resp.StatusCode)
	}

	var result struct {
		AccessToken string `json:"accessToken"`
		ExpiresIn   int64  `json:"expiresIn"`
		TokenType   string `json:"tokenType"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse refresh response: %w", err)
	}

	p.mu.Lock()
	p.accessToken = result.AccessToken
	p.expiresAt = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	p.mu.Unlock()

	return nil
}

func (p *KiroDesktopProvider) IsExpired() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.IsExpiredLocked()
}

// IsExpiredLocked checks expiry without locking (caller must hold at least RLock).
func (p *KiroDesktopProvider) IsExpiredLocked() bool {
	return time.Now().After(p.expiresAt.Add(-10 * time.Minute))
}

func (p *KiroDesktopProvider) Close() {
	close(p.stopChan)
}

func (p *KiroDesktopProvider) backgroundRefresh() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if p.IsExpired() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				_ = p.Refresh(ctx)
				cancel()
			}
		case <-p.stopChan:
			return
		}
	}
}
