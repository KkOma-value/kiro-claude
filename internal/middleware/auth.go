package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

// AuthGuard is HTTP middleware that requires an API key on non-health endpoints.
// It supports both x-api-key and Authorization: Bearer headers.
// When apiKey is empty, all requests are allowed (disabled mode).
type AuthGuard struct {
	apiKey []byte
	next   http.Handler
}

// NewAuthGuard creates a new auth guard middleware.
// If apiKey is empty, the guard is effectively disabled.
func NewAuthGuard(apiKey string, next http.Handler) http.Handler {
	if apiKey == "" {
		return next // No guard when no key configured
	}
	return &AuthGuard{
		apiKey: []byte(apiKey),
		next:   next,
	}
}

func (a *AuthGuard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Skip auth for health and root endpoints
	path := r.URL.Path
	if path == "/health" || path == "/" {
		a.next.ServeHTTP(w, r)
		return
	}

	// Check x-api-key header (Anthropic native)
	if key := r.Header.Get("x-api-key"); key != "" {
		if subtle.ConstantTimeCompare([]byte(key), a.apiKey) == 1 {
			a.next.ServeHTTP(w, r)
			return
		}
	}

	// Check Authorization: Bearer header
	if auth := r.Header.Get("Authorization"); auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			token := strings.TrimPrefix(auth, "Bearer ")
			if subtle.ConstantTimeCompare([]byte(token), a.apiKey) == 1 {
				a.next.ServeHTTP(w, r)
				return
			}
		}
	}

	// Authentication failed
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"type": "error",
		"error": map[string]string{
			"type":    "authentication_error",
			"message": "Invalid or missing API key. Use x-api-key header or Authorization: Bearer.",
		},
	})
}
