package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

// TokenManager handles lifecycle of Kiro tokens: refresh, rotation, multi-account failover
type TokenManager struct {
	credentials   []Credential
	currentIdx    int
	mu            sync.RWMutex
	refreshTicker *time.Ticker
	stopChan      chan struct{}
	lastRefresh   time.Time
	client        *http.Client
}

// NewTokenManager creates and initializes a token manager from Kiro IDE cache
func NewTokenManager(cacheDir string) (*TokenManager, error) {
	credentials, err := LoadKiroCredentials(cacheDir)
	envCred, envErr := LoadFromEnv(os.Getenv("KIRO_REFRESH_TOKEN"), os.Getenv("KIRO_REGION"))
	if envErr == nil {
		credentials = AllowOnMissingCredential(credentials, envCred)
	}
	if err != nil && len(credentials) == 0 {
		if envErr != nil {
			return nil, fmt.Errorf("failed to load credentials: %w", err)
		}
		credentials = []Credential{*envCred}
	}
	if len(credentials) == 0 {
		return nil, fmt.Errorf("no valid credentials available")
	}

	tm := &TokenManager{
		credentials: credentials,
		currentIdx:  0,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		stopChan: make(chan struct{}),
	}

	// Start background token refresh goroutine
	go tm.backgroundRefresh()

	return tm, nil
}

// GetAccessToken returns the current valid access token
// Will block briefly if a refresh is in progress
func (tm *TokenManager) GetAccessToken(ctx context.Context) (string, error) {
	tm.mu.RLock()
	if tm.currentIdx >= len(tm.credentials) {
		tm.mu.RUnlock()
		return "", fmt.Errorf("no valid credentials available")
	}

	cred := tm.credentials[tm.currentIdx]
	tm.mu.RUnlock()

	// Check if token needs refresh
	if cred.IsExpired() {
		if err := tm.RefreshToken(ctx); err != nil {
			return "", fmt.Errorf("token refresh failed: %w", err)
		}
		tm.mu.RLock()
		cred = tm.credentials[tm.currentIdx]
		tm.mu.RUnlock()
	}

	return cred.AccessToken, nil
}

// RefreshToken manually refreshes the current credential's access token
func (tm *TokenManager) RefreshToken(ctx context.Context) error {
	tm.mu.Lock()
	if tm.currentIdx >= len(tm.credentials) {
		tm.mu.Unlock()
		return fmt.Errorf("no valid credentials to refresh")
	}

	cred := tm.credentials[tm.currentIdx]
	tm.mu.Unlock()

	// Call Kiro refresh endpoint (similar to AWS SSO)
	// Reference: prod.us-east-1.auth.desktop.kiro.dev/refreshToken
	newToken, expiresIn, err := tm.refreshAccessToken(ctx, &cred)
	if err != nil {
		return fmt.Errorf("refresh failed for credential %d: %w", tm.currentIdx, err)
	}

	tm.mu.Lock()
	tm.credentials[tm.currentIdx].AccessToken = newToken
	tm.credentials[tm.currentIdx].ExpiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
	tm.lastRefresh = time.Now()
	tm.mu.Unlock()

	return nil
}

// NextAccount rotates to the next credential (for multi-account failover)
// Returns true if rotation was successful, false if no more accounts available
func (tm *TokenManager) NextAccount() bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if len(tm.credentials) <= 1 {
		return false // Can't rotate with only one account
	}

	nextIdx := (tm.currentIdx + 1) % len(tm.credentials)
	if nextIdx == tm.currentIdx {
		return false // Cycled through all, nothing new
	}

	tm.currentIdx = nextIdx
	return true
}

// Status returns the current token and credential status
func (tm *TokenManager) Status() TokenStatus {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	status := TokenStatus{
		CurrentIndex:  tm.currentIdx,
		TotalAccounts: len(tm.credentials),
		LastRefresh:   tm.lastRefresh,
	}

	if tm.currentIdx < len(tm.credentials) {
		status.AccessToken = tm.credentials[tm.currentIdx].AccessToken
		status.ExpiresAt = tm.credentials[tm.currentIdx].ExpiresAt
		status.IsExpired = tm.credentials[tm.currentIdx].IsExpired()
	}

	return status
}

// Close stops the background refresh goroutine
func (tm *TokenManager) Close() {
	close(tm.stopChan)
	if tm.refreshTicker != nil {
		tm.refreshTicker.Stop()
	}
}

// --- Private helper methods ---

// backgroundRefresh runs periodically to refresh tokens before expiration
func (tm *TokenManager) backgroundRefresh() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tm.mu.RLock()
			if tm.currentIdx < len(tm.credentials) && tm.credentials[tm.currentIdx].IsExpired() {
				tm.mu.RUnlock()
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				_ = tm.RefreshToken(ctx)
				cancel()
			} else {
				tm.mu.RUnlock()
			}
		case <-tm.stopChan:
			return
		}
	}
}

// refreshAccessToken calls Kiro's token refresh endpoint
// This mimics AWS SSO's /refreshToken behavior
func (tm *TokenManager) refreshAccessToken(ctx context.Context, cred *Credential) (newToken string, expiresIn int64, err error) {
	// Kiro refresh endpoint (reference from kiro-gateway implementation)
	endpoint := "https://prod.us-east-1.auth.desktop.kiro.dev/refreshToken"

	reqBody := map[string]interface{}{
		"refreshToken": cred.RefreshToken,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := tm.client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("refresh failed with status %d", resp.StatusCode)
	}

	var respBody struct {
		AccessToken string `json:"accessToken"`
		ExpiresIn   int64  `json:"expiresIn"`
		TokenType   string `json:"tokenType"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return "", 0, fmt.Errorf("failed to parse response: %w", err)
	}

	return respBody.AccessToken, respBody.ExpiresIn, nil
}
