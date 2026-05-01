package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/yourusername/kiro-claude/internal/logger"
)

// DebugLogger is HTTP middleware that logs request bodies for debugging.
// Only active when enabled is true.
type DebugLogger struct {
	enabled bool
	log     logger.Logger
	next    http.Handler
}

// NewDebugLogger creates a debug logging middleware.
func NewDebugLogger(enabled bool, log logger.Logger, next http.Handler) http.Handler {
	if !enabled {
		return next
	}
	return &DebugLogger{
		enabled: enabled,
		log:     log,
		next:    next,
	}
}

func (d *DebugLogger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only log POST requests to /v1/messages
	if r.Method == "POST" && r.URL.Path == "/v1/messages" {
		body, err := io.ReadAll(r.Body)
		if err == nil {
			d.log.Debugf("=== Incoming Request ===")
			d.log.Debugf("POST %s", r.URL.Path)
			d.log.Debugf("Content-Length: %d", len(body))

			// Truncate body for logging (max 2KB)
			logBody := body
			if len(logBody) > 2048 {
				logBody = append(logBody[:2048], []byte("... (truncated)")...)
			}
			d.log.Debugf("Body: %s", string(logBody))

			// Reconstruct body for downstream handlers
			r.Body = io.NopCloser(bytes.NewReader(body))
		}
	}

	d.next.ServeHTTP(w, r)
}
