package account

import (
	"fmt"
	"sync"

	"github.com/yourusername/kiro-claude/internal/auth"
	"github.com/yourusername/kiro-claude/internal/logger"
)

// Account represents a single Kiro credential with its auth provider and health state.
type Account struct {
	ID       string
	Provider auth.Provider
	healthy  bool
	failures int
}

// Manager handles multi-account failover.
// When one account returns a recoverable error, the manager rotates to the next one.
type Manager struct {
	accounts   []*Account
	currentIdx int
	mu         sync.RWMutex
	log        logger.Logger
}

// NewManager creates a manager from a list of auth providers.
func NewManager(providers []auth.Provider, log logger.Logger) *Manager {
	accounts := make([]*Account, len(providers))
	for i, p := range providers {
		authTypeName := "unknown"
		switch p.Type() {
		case auth.AuthKiroDesktop:
			authTypeName = "kiro-desktop"
		case auth.AuthAWSSSO:
			authTypeName = "aws-sso"
		case auth.AuthEnv:
			authTypeName = "env"
		}
		accounts[i] = &Account{
			ID:       fmt.Sprintf("%s-%d", authTypeName, i+1),
			Provider: p,
			healthy:  true,
		}
	}

	return &Manager{
		accounts: accounts,
		log:      log,
	}
}

// GetProvider returns the current healthy account's auth provider.
// Returns nil if no healthy accounts are available.
func (m *Manager) GetProvider() auth.Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.accounts) == 0 {
		return nil
	}

	return m.accounts[m.currentIdx].Provider
}

// CurrentAccountID returns the ID of the currently selected account.
func (m *Manager) CurrentAccountID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.accounts) == 0 {
		return ""
	}
	return m.accounts[m.currentIdx].ID
}

// AccountCount returns the total number of accounts.
func (m *Manager) AccountCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.accounts)
}

// ReportSuccess marks the current account as healthy and resets its failure count.
func (m *Manager) ReportSuccess() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.accounts) == 0 {
		return
	}

	acct := m.accounts[m.currentIdx]
	acct.healthy = true
	acct.failures = 0
}

// ReportFailure records a failure on the current account.
// If the error is recoverable and there are other accounts, it rotates to the next one.
// Returns true if rotation happened, false if not (single account or fatal error).
func (m *Manager) ReportFailure(errType ErrorType) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.accounts) == 0 {
		return false
	}

	acct := m.accounts[m.currentIdx]
	acct.failures++

	if errType == ErrorFatal {
		// Fatal errors don't trigger rotation
		return false
	}

	// Recoverable error: try next account
	if len(m.accounts) <= 1 {
		return false // Only one account, can't rotate
	}

	nextIdx := (m.currentIdx + 1) % len(m.accounts)
	if nextIdx == m.currentIdx {
		return false
	}

	m.log.Warnf("Account %s failed (%d failures), rotating to %s",
		acct.ID, acct.failures, m.accounts[nextIdx].ID)

	m.currentIdx = nextIdx
	return true
}

// Close stops all account providers.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, acct := range m.accounts {
		acct.Provider.Close()
	}
}
