package auth

import (
	"time"
)

// Credential represents a Kiro IDE SSO credential
type Credential struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
	Region       string    `json:"region"`
	ClientID     string    `json:"clientId"`
	ClientSecret string    `json:"clientSecret"`
	ProfileArn   string    `json:"profileArn,omitempty"`
}

// IsExpired checks if the credential is expired or about to expire (within 10 minutes)
func (c *Credential) IsExpired() bool {
	return time.Now().After(c.ExpiresAt.Add(-10 * time.Minute))
}

// TokenStatus represents the current token state
type TokenStatus struct {
	CurrentIndex  int
	TotalAccounts int
	AccessToken   string
	ExpiresAt     time.Time
	IsExpired     bool
	LastRefresh   time.Time
}
