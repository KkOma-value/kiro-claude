package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// TokenManager handles lifecycle of Kiro tokens: refresh, rotation, multi-account failover
type TokenManager struct {
	credentials   []Credential
	currentIdx    int
	mu            sync.RWMutex
	stopChan      chan struct{}
	lastRefresh   time.Time
	client        *http.Client
}

func NewTokenManager(cacheDir string) (*TokenManager, error) {
	credentials, cacheErr := LoadKiroCredentials(cacheDir)
	envCred, envErr := LoadFromEnv()
	if envErr == nil {
		credentials = MergeCredentials(credentials, envCred)
	}
	if cacheErr != nil && len(credentials) == 0 {
		if envErr != nil {
			return nil, fmt.Errorf("failed to load credentials: %w", cacheErr)
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
			Timeout: 15 * time.Second,
		},
		stopChan: make(chan struct{}),
	}

	go tm.backgroundRefresh()

	return tm, nil
}

func (tm *TokenManager) GetAccessToken(ctx context.Context) (string, error) {
	tm.mu.RLock()
	if tm.currentIdx >= len(tm.credentials) {
		tm.mu.RUnlock()
		return "", fmt.Errorf("no valid credentials available")
	}
	cred := tm.credentials[tm.currentIdx]
	tm.mu.RUnlock()

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

// CurrentCredential returns a copy of the active credential (for headers, profileArn, etc.)
func (tm *TokenManager) CurrentCredential() Credential {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	if tm.currentIdx < len(tm.credentials) {
		return tm.credentials[tm.currentIdx]
	}
	return Credential{}
}

func (tm *TokenManager) RefreshToken(ctx context.Context) error {
	tm.mu.Lock()
	if tm.currentIdx >= len(tm.credentials) {
		tm.mu.Unlock()
		return fmt.Errorf("no valid credentials to refresh")
	}
	cred := tm.credentials[tm.currentIdx]
	tm.mu.Unlock()

	var newToken string
	var newRefreshToken string
	var expiresIn int64
	var err error

	if cred.IsIDC() {
		newToken, newRefreshToken, expiresIn, err = tm.refreshIDCToken(ctx, &cred)
	} else {
		newToken, newRefreshToken, expiresIn, err = tm.refreshSocialToken(ctx, &cred)
	}

	if err != nil {
		return fmt.Errorf("refresh failed for credential %d (%s): %w", tm.currentIdx, cred.AuthMethod, err)
	}

	tm.mu.Lock()
	tm.credentials[tm.currentIdx].AccessToken = newToken
	if newRefreshToken != "" {
		tm.credentials[tm.currentIdx].RefreshToken = newRefreshToken
	}
	tm.credentials[tm.currentIdx].ExpiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
	tm.lastRefresh = time.Now()
	tm.mu.Unlock()

	return nil
}

func (tm *TokenManager) NextAccount() bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if len(tm.credentials) <= 1 {
		return false
	}
	nextIdx := (tm.currentIdx + 1) % len(tm.credentials)
	if nextIdx == tm.currentIdx {
		return false
	}
	tm.currentIdx = nextIdx
	return true
}

func (tm *TokenManager) Status() TokenStatus {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	status := TokenStatus{
		CurrentIndex:  tm.currentIdx,
		TotalAccounts: len(tm.credentials),
		LastRefresh:   tm.lastRefresh,
	}

	if tm.currentIdx < len(tm.credentials) {
		c := tm.credentials[tm.currentIdx]
		status.AccessToken = c.AccessToken
		status.ExpiresAt = c.ExpiresAt
		status.IsExpired = c.IsExpired()
		status.AuthMethod = c.AuthMethod
		status.Region = c.EffectiveRegion()
	}

	return status
}

func (tm *TokenManager) Close() {
	close(tm.stopChan)
}

func (tm *TokenManager) backgroundRefresh() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tm.mu.RLock()
			needsRefresh := tm.currentIdx < len(tm.credentials) && tm.credentials[tm.currentIdx].IsExpired()
			tm.mu.RUnlock()
			if needsRefresh {
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				_ = tm.RefreshToken(ctx)
				cancel()
			}
		case <-tm.stopChan:
			return
		}
	}
}

// refreshSocialToken calls Kiro's social auth refresh endpoint
func (tm *TokenManager) refreshSocialToken(ctx context.Context, cred *Credential) (accessToken string, refreshToken string, expiresIn int64, err error) {
	region := cred.EffectiveRegion()
	endpoint := strings.Replace(
		"https://prod.{{region}}.auth.desktop.kiro.dev/refreshToken",
		"{{region}}", region, 1,
	)

	reqBody := map[string]interface{}{
		"refreshToken": cred.RefreshToken,
	}

	return tm.doRefreshRequest(ctx, endpoint, reqBody, nil)
}

// refreshIDCToken calls AWS OIDC token endpoint for Builder ID / IdC auth
func (tm *TokenManager) refreshIDCToken(ctx context.Context, cred *Credential) (accessToken string, refreshToken string, expiresIn int64, err error) {
	idcRegion := cred.EffectiveIDCRegion()
	endpoint := strings.Replace(
		"https://oidc.{{region}}.amazonaws.com/token",
		"{{region}}", idcRegion, 1,
	)

	reqBody := map[string]interface{}{
		"grantType":    "refresh_token",
		"refreshToken": cred.RefreshToken,
		"clientId":     cred.ClientID,
		"clientSecret": cred.ClientSecret,
	}

	headers := map[string]string{
		"User-Agent": "KiroIDE",
	}

	return tm.doRefreshRequest(ctx, endpoint, reqBody, headers)
}

func (tm *TokenManager) doRefreshRequest(ctx context.Context, endpoint string, reqBody map[string]interface{}, extraHeaders map[string]string) (string, string, int64, error) {
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := tm.client.Do(req)
	if err != nil {
		return "", "", 0, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", 0, fmt.Errorf("refresh failed with status %d", resp.StatusCode)
	}

	var respBody struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		ExpiresIn    int64  `json:"expiresIn"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return "", "", 0, fmt.Errorf("failed to parse response: %w", err)
	}

	if respBody.ExpiresIn == 0 {
		respBody.ExpiresIn = 3600
	}

	return respBody.AccessToken, respBody.RefreshToken, respBody.ExpiresIn, nil
}
