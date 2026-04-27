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

// LoadKiroCredentials scans ~/.aws/sso/cache or custom Kiro cache directory
// for SSO token files and returns a list of credentials
func LoadKiroCredentials(cacheDir string) ([]Credential, error) {
	// Expand ~ to home directory
	if strings.HasPrefix(cacheDir, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		cacheDir = filepath.Join(home, cacheDir[1:])
	}

	// Check if directory exists
	if _, err := os.Stat(cacheDir); err != nil {
		return nil, fmt.Errorf("cache directory not found: %s: %w", cacheDir, err)
	}

	var credentials []Credential

	// Walk through cache directory looking for JSON files
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(cacheDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			// Log and continue; corrupt files shouldn't block loading others
			fmt.Fprintf(os.Stderr, "warning: failed to read %s: %v\n", filePath, err)
			continue
		}

		var cred Credential
		if err := json.Unmarshal(data, &cred); err != nil {
			// Not a credential file, skip
			continue
		}

		// Validate that this looks like a Kiro credential
		if cred.AccessToken != "" && cred.RefreshToken != "" {
			credentials = append(credentials, cred)
		}
	}

	if len(credentials) == 0 {
		return nil, fmt.Errorf("no valid Kiro credentials found in %s", cacheDir)
	}

	// Sort by expiration time (furthest in future first)
	sort.Slice(credentials, func(i, j int) bool {
		return credentials[i].ExpiresAt.After(credentials[j].ExpiresAt)
	})

	return credentials, nil
}

// LoadFromEnv loads a single credential from environment variables
// Used for Docker/CI environments where cache directory is not available
func LoadFromEnv(refreshToken, region string) (*Credential, error) {
	if refreshToken == "" {
		return nil, fmt.Errorf("KIRO_REFRESH_TOKEN not set")
	}

	return &Credential{
		RefreshToken: refreshToken,
		Region:       region,
		ExpiresAt:    time.Now(), // Will trigger refresh on first use
	}, nil
}

// AllowOnMissingCredential merges credentials from both cache and environment variables
// returning the union of both sources, with cache credentials preferred
func AllowOnMissingCredential(cached []Credential, envCred *Credential) []Credential {
	if envCred == nil || envCred.RefreshToken == "" {
		return cached
	}

	// Check if this environment token is already in cached list
	for _, c := range cached {
		if c.RefreshToken == envCred.RefreshToken {
			return cached // Already there
		}
	}

	// Append env credential to the end (lower priority)
	return append(cached, *envCred)
}
