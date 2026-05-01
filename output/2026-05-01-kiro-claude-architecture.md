# Kiro Claude Reverse Proxy — Architecture v2

Date: 2026-05-01
Phase: docs
Baseline: 2026-04-28 architecture + kiro-gateway analysis

## Architecture Overview

The system retains the two-layer design from v1 (Anthropic-facing proxy + backend adapter) and adds three new cross-cutting concerns:

```text
Claude Code / Anthropic SDK
  → HTTP Anthropic API
  → [Middleware: API Key Guard]
  → [Middleware: Request Logger]
  → Proxy Server
     → API Handler
     → Model Resolver (normalize → static map → passthrough)
     → Anthropic ↔ Internal conversion
     → Backend Adapter interface
        → Mock Backend Adapter
        → Live Backend Adapter
           → Account Manager (multi-credential failover)
           → Auth Provider interface
              → Kiro Desktop Auth Provider
              → AWS SSO (OIDC) Auth Provider
              → Env Auth Provider
           → Upstream HTTP Client
```

## New Components

### 1. `internal/middleware` — HTTP Middleware Layer

#### `auth.go` — Proxy API Key Guard

Responsibilities:
- Check `x-api-key` or `Authorization: Bearer` headers
- Skip auth for health endpoints (`/health`, `/`)
- Return Anthropic-style 401 error envelope on failure
- Use `subtle.ConstantTimeCompare` for timing-attack resistance

```go
type AuthMiddleware struct {
    apiKey string
    next   http.Handler
}
```

#### `logger.go` — Request/Response Debug Logger

Responsibilities:
- Log request body (redact auth headers)
- Log outgoing upstream request
- Log upstream response or error
- Controlled by `logging.debug_dump` config flag

---

### 2. `internal/model` — Smart Model Name Resolution

#### `resolver.go` — Model Name Resolution Pipeline

Responsibilities:
- Normalize model names (dashes→dots, strip dates, handle legacy formats)
- Check static model name mapping
- Passthrough unknown models to Kiro (gateway, not gatekeeper)

```go
type Resolver struct {
    staticMap map[string]string  // normalized name → CodeWhisperer ID
    reverseMap map[string]string // CodeWhisperer ID → normalized name
}

type Resolution struct {
    InternalID    string  // ID to send to Kiro
    ExternalName  string  // Name to return to Claude Code
    Source        string  // "static", "passthrough"
    Original      string  // What the client sent
}

func (r *Resolver) Resolve(clientModel string) Resolution
func (r *Resolver) ReverseResolve(kiroModel string) string
func (r *Resolver) ListModels() []gateway.ModelInfo
```

Normalization rules (from kiro-gateway analysis):
1. `claude-sonnet-4-5` → `claude-sonnet-4.5` (dash-to-dot for minor version)
2. `claude-sonnet-4-5-20250929` → `claude-sonnet-4.5` (strip date suffix)
3. `claude-sonnet-4-5-latest` → `claude-sonnet-4.5` (strip latest suffix)
4. `claude-3-7-sonnet` → `claude-3.7-sonnet` (legacy format)
5. `claude-3-7-sonnet-20250219` → `claude-3.7-sonnet` (legacy + date strip)
6. Already-normalized names pass through unchanged

---

### 3. `internal/auth` — Enhanced Auth Provider System

#### `provider.go` — Auth Provider Interface

```go
type AuthType int

const (
    AuthKiroDesktop AuthType = iota
    AuthAWSSSO
    AuthEnv
)

type Provider interface {
    Type() AuthType
    GetAccessToken(ctx context.Context) (string, error)
    Refresh(ctx context.Context) error
    IsExpired() bool
    Close()
}
```

#### `kiro_desktop.go` — Kiro Desktop Auth (existing, refactored)

- Endpoint: `https://prod.{region}.auth.desktop.kiro.dev/refreshToken`
- Requires: `refreshToken`, `profileArn` (optional)
- No `clientId`/`clientSecret`

#### `aws_sso.go` — AWS SSO (OIDC) Auth (NEW)

- Endpoint: `https://oidc.{region}.amazonaws.com/token`
- Requires: `clientId`, `clientSecret`, `refreshToken`
- Auto-detected when credentials contain `clientId` and `clientSecret`

```go
type AWSSSOProvider struct {
    clientID     string
    clientSecret string
    refreshToken string
    region       string
    accessToken  string
    expiresAt    time.Time
    mu           sync.RWMutex
    client       *http.Client
}
```

#### `detector.go` — Auth Type Auto-Detection (NEW)

```go
// DetectAuthType inspects a credential to determine which auth provider to use.
// If clientId and clientSecret are present → AWS SSO
// Otherwise → Kiro Desktop
func DetectAuthType(cred *Credential) AuthType
```

---

### 4. `internal/account` — Multi-Account Manager (NEW)

#### `manager.go` — Account Failover Manager

```go
type ErrorType int

const (
    ErrorFatal       ErrorType = iota  // 400 validation, 403 forbidden
    ErrorRecoverable ErrorType = iota  // 429 rate limit, 402 quota
)

type Manager struct {
    accounts    []Account
    currentIdx  int
    mu          sync.RWMutex
}

type Account struct {
    ID       string
    Provider auth.Provider
    Healthy  bool
    Failures int
}

func (m *Manager) GetProvider(ctx context.Context) (auth.Provider, error)
func (m *Manager) ReportSuccess(accountID string)
func (m *Manager) ReportFailure(accountID string, errType ErrorType)
func (m *Manager) NextAccount() bool
```

Classification rules (from kiro-gateway):
- **Fatal** (return immediately): 400, 403, 413, 422
- **Recoverable** (try next account): 429, 402, 500, 502, 503, 504

---

## Modified Components

### `internal/config/config.go` — Enhanced Configuration

New fields:

```yaml
security:
  proxy_api_key: ""

logging:
  debug_dump: false
```

New env overrides:
- `KIRO_PROXY_API_KEY`
- `KIRO_DEBUG_DUMP`

### `internal/runtime/factory.go` — Enhanced Startup

Changes:
- Create auth providers with auto-detection
- Build account manager with all loaded credentials
- Wire model resolver into API handler
- Apply middleware (auth guard, debug logger)

### `internal/api/handler.go` — Enhanced Handler

Changes:
- Use model resolver instead of direct static map lookup
- Accept model resolver as dependency
- Improved error envelope handling for upstream failures

### `internal/gateway/types.go` — Enhanced Model Data

Changes:
- Move model map to model resolver package
- Add normalization tests
- Keep `SupportedModels()` as fallback

### `cmd/server/main.go` — Enhanced Bootstrap

Changes:
- Wire auth middleware
- Wire debug logger middleware
- Log startup banner with runtime info (mode, auth type, bind address)
- Mask API key in logs

---

## Package Layout

```text
cmd/
  server/
    main.go                    # [MODIFY] Add middleware wiring
  cli/
    (future)

internal/
  api/
    handler.go                 # [MODIFY] Use model resolver
    handler_test.go            # [MODIFY] Add auth tests
    converter.go               # [MODIFY] Use resolver for model mapping
    converter_test.go          # [MODIFY] Add normalization test cases
    types.go                   # no change

  auth/
    model.go                   # [MODIFY] Add ClientID/ClientSecret fields
    sso.go                     # [MODIFY] Refactor to provider pattern
    token.go                   # [MODIFY] Refactor to provider pattern
    provider.go                # [NEW] Auth provider interface
    kiro_desktop.go            # [NEW] Kiro Desktop auth implementation
    aws_sso.go                 # [NEW] AWS SSO OIDC auth implementation
    detector.go                # [NEW] Auth type auto-detection

  backend/
    client.go                  # no change
    mock.go                    # no change
    mock_test.go               # no change
    live.go                    # [MODIFY] Use account manager

  config/
    config.go                  # [MODIFY] Add security + debug config
    config_test.go             # [MODIFY] Add new field tests

  gateway/
    client.go                  # [MODIFY] Accept auth.Provider instead of TokenManager
    types.go                   # [MODIFY] Move model maps to model package

  logger/
    logger.go                  # no change

  middleware/                  # [NEW] Package
    auth.go                    # [NEW] Proxy API key guard
    logger.go                  # [NEW] Debug request/response logger

  model/                       # [NEW] Package
    resolver.go                # [NEW] Model name resolution pipeline
    resolver_test.go           # [NEW] Normalization test cases
    models.go                  # [NEW] Model maps + info (moved from gateway/types.go)

  account/                     # [NEW] Package
    manager.go                 # [NEW] Multi-account failover
    errors.go                  # [NEW] Error classification
    manager_test.go            # [NEW] Failover test cases

  runtime/
    factory.go                 # [MODIFY] Wire new components
    factory_test.go            # [MODIFY] Add middleware tests
```

## Testing Strategy

### Unit Tests
- Model name normalization (20+ test cases covering all patterns)
- Auth type auto-detection
- Error classification
- API key guard middleware
- Config parsing with new fields

### Integration Tests
- Boot server in mock mode with API key → verify auth
- Boot server in mock mode → verify model name normalization in requests
- Hit `/v1/messages` with various model name formats
- Verify health endpoint is unauthenticated

### Compatibility Tests
- Verify all existing tests still pass with no modification
- Verify mock scenarios work unchanged

## Migration Path

1. Create new packages (`middleware`, `model`, `account`) without modifying existing code
2. Refactor auth into provider pattern (backward-compatible)
3. Wire model resolver into handler (replaces direct map lookup)
4. Wire middleware into server bootstrap
5. Add multi-account failover to live backend
6. Add integration tests
7. Update documentation

## Key Design Decisions

1. **Gateway, not gatekeeper**: Unknown model names are passed through to Kiro. The proxy never rejects a model it doesn't recognize.

2. **Auth detection by credential shape**: No explicit "auth type" config flag. The system inspects credential content to decide which auth flow to use.

3. **Middleware over handler logic**: API key guard and debug logging are implemented as HTTP middleware, not embedded in the API handler. This keeps handler code focused on conversion.

4. **Error classification at boundary**: Error classification happens where the upstream response is received, not in the handler. The handler gets pre-classified errors.

5. **Backward-compatible defaults**: `proxy_api_key: ""` means no auth required (same as current behavior). All new features are opt-in.
