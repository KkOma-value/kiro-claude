package account

import (
	"context"
	"testing"

	"github.com/yourusername/kiro-claude/internal/auth"
	"github.com/yourusername/kiro-claude/internal/logger"
)

// mockProvider is a test-only auth.Provider implementation.
type mockProvider struct {
	authType auth.AuthType
}

func (m *mockProvider) Type() auth.AuthType                              { return m.authType }
func (m *mockProvider) GetAccessToken(_ context.Context) (string, error) { return "test-token", nil }
func (m *mockProvider) Refresh(_ context.Context) error                  { return nil }
func (m *mockProvider) IsExpired() bool                                  { return false }
func (m *mockProvider) Close()                                           {}

func newMockManager(n int) *Manager {
	providers := make([]auth.Provider, n)
	for i := 0; i < n; i++ {
		providers[i] = &mockProvider{authType: auth.AuthKiroDesktop}
	}
	return NewManager(providers, logger.NewSimpleLogger(logger.LevelInfo))
}

func TestManagerSingleAccountNoRotation(t *testing.T) {
	mgr := newMockManager(1)
	defer mgr.Close()

	// Fatal error should not rotate
	rotated := mgr.ReportFailure(ErrorFatal)
	if rotated {
		t.Error("fatal error should not rotate with single account")
	}

	// Recoverable error should not rotate (only 1 account)
	rotated = mgr.ReportFailure(ErrorRecoverable)
	if rotated {
		t.Error("recoverable error should not rotate with single account")
	}
}

func TestManagerMultiAccountRotation(t *testing.T) {
	mgr := newMockManager(3)
	defer mgr.Close()

	id1 := mgr.CurrentAccountID()

	// Recoverable error should rotate
	rotated := mgr.ReportFailure(ErrorRecoverable)
	if !rotated {
		t.Error("recoverable error should rotate with multiple accounts")
	}

	id2 := mgr.CurrentAccountID()
	if id1 == id2 {
		t.Errorf("account should have changed: before=%s, after=%s", id1, id2)
	}
}

func TestManagerFatalNoRotation(t *testing.T) {
	mgr := newMockManager(3)
	defer mgr.Close()

	id1 := mgr.CurrentAccountID()

	rotated := mgr.ReportFailure(ErrorFatal)
	if rotated {
		t.Error("fatal error should not rotate")
	}

	id2 := mgr.CurrentAccountID()
	if id1 != id2 {
		t.Errorf("account should not have changed: before=%s, after=%s", id1, id2)
	}
}

func TestManagerReportSuccess(t *testing.T) {
	mgr := newMockManager(2)
	defer mgr.Close()

	// Report some failures then success
	mgr.ReportFailure(ErrorRecoverable)
	mgr.ReportSuccess()

	// Should still have a provider
	if mgr.GetProvider() == nil {
		t.Error("expected a provider after success")
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		code int
		want ErrorType
	}{
		{400, ErrorFatal},
		{403, ErrorFatal},
		{413, ErrorFatal},
		{422, ErrorFatal},
		{429, ErrorRecoverable},
		{402, ErrorRecoverable},
		{500, ErrorRecoverable},
		{502, ErrorRecoverable},
		{503, ErrorRecoverable},
		{504, ErrorRecoverable},
		{401, ErrorFatal},       // Client error default
		{404, ErrorFatal},       // Client error default
		{530, ErrorRecoverable}, // Server error default
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := ClassifyError(tt.code)
			if got != tt.want {
				t.Errorf("ClassifyError(%d) = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}
