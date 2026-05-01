package auth

import "context"

// AuthType identifies the authentication method used.
type AuthType int

const (
	AuthKiroDesktop AuthType = iota // Kiro IDE Desktop auth
	AuthAWSSSO                      // AWS SSO (OIDC) auth
	AuthEnv                         // Environment variable (refresh token only)
)

// Provider abstracts over different Kiro authentication methods.
// All auth providers must implement this interface.
type Provider interface {
	// Type returns the authentication method used.
	Type() AuthType

	// GetAccessToken returns a valid access token, refreshing if needed.
	GetAccessToken(ctx context.Context) (string, error)

	// Refresh explicitly refreshes the access token.
	Refresh(ctx context.Context) error

	// IsExpired returns true if the current token is expired or about to expire.
	IsExpired() bool

	// Close releases any resources held by the provider.
	Close()
}

// DetectAuthType inspects a credential to determine which auth provider to use.
// If clientId and clientSecret are present → AWS SSO
// If only refreshToken is present → Kiro Desktop or Env
func DetectAuthType(cred *Credential) AuthType {
	if cred.ClientID != "" && cred.ClientSecret != "" {
		return AuthAWSSSO
	}
	return AuthKiroDesktop
}
