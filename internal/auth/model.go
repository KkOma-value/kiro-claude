package auth

import (
	"time"
)

// Credential represents a Kiro IDE credential (supports both Social and IDC auth)
type Credential struct {
	AccessToken            string    `json:"accessToken"`
	RefreshToken           string    `json:"refreshToken"`
	ExpiresAt              time.Time `json:"expiresAt"`
	Region                 string    `json:"region"`
	IDCRegion              string    `json:"idcRegion,omitempty"`
	ClientID               string    `json:"clientId,omitempty"`
	ClientSecret           string    `json:"clientSecret,omitempty"`
	ProfileArn             string    `json:"profileArn,omitempty"`
	AuthMethod             string    `json:"authMethod,omitempty"` // "social" or "IdC" / "builder-id"
	StartURL               string    `json:"startUrl,omitempty"`
	RegistrationExpiresAt  string    `json:"registrationExpiresAt,omitempty"`
}

func (c *Credential) IsExpired() bool {
	return time.Now().After(c.ExpiresAt.Add(-2 * time.Minute))
}

func (c *Credential) IsIDC() bool {
	return c.AuthMethod == "IdC" || c.AuthMethod == "builder-id"
}

func (c *Credential) EffectiveRegion() string {
	if c.Region != "" {
		return c.Region
	}
	return "us-east-1"
}

func (c *Credential) EffectiveIDCRegion() string {
	if c.IDCRegion != "" {
		return c.IDCRegion
	}
	return c.EffectiveRegion()
}

type TokenStatus struct {
	CurrentIndex  int
	TotalAccounts int
	AccessToken   string
	ExpiresAt     time.Time
	IsExpired     bool
	LastRefresh   time.Time
	AuthMethod    string
	Region        string
}
