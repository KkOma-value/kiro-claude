package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// LoadKiroCredentials scans credential directories for Kiro token files.
// Supports: ~/.aws/sso/cache, configs/kiro/, and custom paths.
func LoadKiroCredentials(cacheDir string) ([]Credential, error) {
	cacheDir = expandHome(cacheDir)

	if _, err := os.Stat(cacheDir); err != nil {
		return nil, fmt.Errorf("cache directory not found: %s: %w", cacheDir, err)
	}

	var credentials []Credential

	err := filepath.WalkDir(cacheDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}

		cred, parseErr := parseCredentialFile(path)
		if parseErr != nil || cred == nil {
			return nil
		}
		credentials = append(credentials, *cred)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk cache directory: %w", err)
	}

	if len(credentials) == 0 {
		return nil, fmt.Errorf("no valid Kiro credentials found in %s", cacheDir)
	}

	sort.Slice(credentials, func(i, j int) bool {
		return credentials[i].ExpiresAt.After(credentials[j].ExpiresAt)
	})

	return credentials, nil
}

func parseCredentialFile(filePath string) (*Credential, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Try standard JSON parse first
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	// Must have at least refreshToken or accessToken
	_, hasRefresh := raw["refreshToken"]
	_, hasAccess := raw["accessToken"]
	if !hasRefresh && !hasAccess {
		return nil, nil
	}

	cred := &Credential{}

	if v, ok := raw["accessToken"].(string); ok {
		cred.AccessToken = v
	}
	if v, ok := raw["refreshToken"].(string); ok {
		cred.RefreshToken = v
	}
	if v, ok := raw["clientId"].(string); ok {
		cred.ClientID = v
	}
	if v, ok := raw["clientSecret"].(string); ok {
		cred.ClientSecret = v
	}
	if v, ok := raw["profileArn"].(string); ok {
		cred.ProfileArn = v
	}
	if v, ok := raw["authMethod"].(string); ok {
		cred.AuthMethod = v
	}
	if v, ok := raw["region"].(string); ok {
		cred.Region = v
	}
	if v, ok := raw["idcRegion"].(string); ok {
		cred.IDCRegion = v
	}
	if v, ok := raw["startUrl"].(string); ok {
		cred.StartURL = v
	}
	if v, ok := raw["registrationExpiresAt"].(string); ok {
		cred.RegistrationExpiresAt = v
	}

	// Parse expiresAt
	if v, ok := raw["expiresAt"].(string); ok {
		for _, layout := range []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02T15:04:05.000Z",
			"2006-01-02T15:04:05Z",
		} {
			if t, err := time.Parse(layout, v); err == nil {
				cred.ExpiresAt = t
				break
			}
		}
	}

	// Validate: must have refreshToken to be useful
	if cred.RefreshToken == "" {
		return nil, nil
	}

	// Default region
	if cred.Region == "" {
		cred.Region = "us-east-1"
	}

	return cred, nil
}

// LoadFromEnv loads a single credential from environment variables.
func LoadFromEnv() (*Credential, error) {
	refreshToken := os.Getenv("KIRO_REFRESH_TOKEN")
	if refreshToken == "" {
		return nil, fmt.Errorf("KIRO_REFRESH_TOKEN not set")
	}

	region := os.Getenv("KIRO_REGION")
	if region == "" {
		region = "us-east-1"
	}

	cred := &Credential{
		RefreshToken: refreshToken,
		Region:       region,
		ExpiresAt:    time.Now(), // Will trigger refresh on first use
	}

	// IDC credentials from env
	if clientID := os.Getenv("KIRO_CLIENT_ID"); clientID != "" {
		cred.ClientID = clientID
		cred.ClientSecret = os.Getenv("KIRO_CLIENT_SECRET")
		cred.AuthMethod = "IdC"
		if idcRegion := os.Getenv("KIRO_IDC_REGION"); idcRegion != "" {
			cred.IDCRegion = idcRegion
		}
	} else {
		cred.AuthMethod = "social"
		cred.ProfileArn = os.Getenv("KIRO_PROFILE_ARN")
	}

	return cred, nil
}

// MergeCredentials merges credentials from cache and environment, deduplicating by refreshToken.
func MergeCredentials(cached []Credential, envCred *Credential) []Credential {
	if envCred == nil || envCred.RefreshToken == "" {
		return cached
	}
	for _, c := range cached {
		if c.RefreshToken == envCred.RefreshToken {
			return cached
		}
	}
	return append(cached, *envCred)
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return p
		}
		return filepath.Join(home, p[1:])
	}
	return p
}
