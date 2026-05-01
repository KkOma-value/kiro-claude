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

// AWSSSOProvider implements Provider for AWS SSO (OIDC) auth.
// Endpoint: https://oidc.{region}.amazonaws.com/token
// Used when credentials contain clientId and clientSecret.
type AWSSSOProvider struct {
	clientID     string
	clientSecret string
	refreshToken string
	region       string
	accessToken  string
	expiresAt    time.Time
	mu           sync.RWMutex
	client       *http.Client
	stopChan     chan struct{}
}

// NewAWSSSOProvider creates a provider from an AWS SSO credential.
func NewAWSSSOProvider(cred *Credential) *AWSSSOProvider {
	region := cred.Region
	if region == "" {
		region = "us-east-1"
	}

	p := &AWSSSOProvider{
		clientID:     cred.ClientID,
		clientSecret: cred.ClientSecret,
		refreshToken: cred.RefreshToken,
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

func (p *AWSSSOProvider) Type() AuthType { return AuthAWSSSO }

func (p *AWSSSOProvider) GetAccessToken(ctx context.Context) (string, error) {
	p.mu.RLock()
	token := p.accessToken
	expired := p.isExpiredLocked()
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

func (p *AWSSSOProvider) Refresh(ctx context.Context) error {
	endpoint := fmt.Sprintf("https://oidc.%s.amazonaws.com/token", p.region)

	reqBody := map[string]interface{}{
		"grant_type":    "refresh_token",
		"client_id":     p.clientID,
		"client_secret": p.clientSecret,
		"refresh_token": p.refreshToken,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal OIDC refresh request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create OIDC refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("OIDC refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OIDC refresh failed with status %d", resp.StatusCode)
	}

	var result struct {
		AccessToken  string `json:"accessToken"`
		ExpiresIn    int64  `json:"expiresIn"`
		RefreshToken string `json:"refreshToken"`
		TokenType    string `json:"tokenType"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse OIDC refresh response: %w", err)
	}

	p.mu.Lock()
	p.accessToken = result.AccessToken
	p.expiresAt = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	// AWS SSO can rotate refresh tokens
	if result.RefreshToken != "" {
		p.refreshToken = result.RefreshToken
	}
	p.mu.Unlock()

	return nil
}

func (p *AWSSSOProvider) IsExpired() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isExpiredLocked()
}

func (p *AWSSSOProvider) isExpiredLocked() bool {
	return time.Now().After(p.expiresAt.Add(-10 * time.Minute))
}

func (p *AWSSSOProvider) Close() {
	close(p.stopChan)
}

func (p *AWSSSOProvider) backgroundRefresh() {
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
