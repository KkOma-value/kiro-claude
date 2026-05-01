# Kiro Claude Reverse Proxy — PRD v2

Date: 2026-05-01
Phase: docs
Baseline: 2026-04-28 PRD + evolve research

## Product Goal

Deliver a production-grade local reverse proxy that:
1. Exposes an **Anthropic-compatible API** for Claude Code
2. Routes requests to **Kiro upstream** (CodeWhisperer) via multiple auth methods
3. Provides **multi-account failover** with error classification
4. Supports **smart model resolution** that doesn't break on new models
5. Protects the proxy with an **API key guard**

## Primary Users

1. **Developer with Kiro IDE credentials**: Has `~/.aws/sso/cache/kiro-auth-token.json`, wants Claude Code to work through Kiro.
2. **Developer with kiro-cli (AWS SSO)**: Has OIDC credentials from `kiro-cli login`, uses Builder ID or corporate SSO.
3. **Developer without credentials**: Uses mock mode for integration testing.

## Success Criteria

1. `go test ./...` passes with zero Kiro credentials.
2. `go run ./cmd/server` starts in mock mode within 1 second.
3. Claude Code connects via `ANTHROPIC_BASE_URL` and receives valid responses in both streaming and non-streaming modes.
4. In `kiro-live` mode with valid credentials, requests reach Kiro upstream and return real responses.
5. Model names like `claude-sonnet-4-5-20250929`, `claude-sonnet-4.5`, `claude-3-7-sonnet` all resolve correctly.
6. The proxy requires an API key when `proxy_api_key` is configured.
7. When one credential fails with a recoverable error (429/402), the proxy rotates to the next credential.

## In Scope

- Anthropic-compatible `/v1/messages` (streaming + non-streaming)
- Anthropic-compatible `GET /v1/models`
- Health check endpoint
- **Proxy API key guard** (`x-api-key` and `Authorization: Bearer`)
- **Smart model name resolution** (normalize, passthrough, fuzzy matching)
- **Auth provider abstraction** with auto-detection:
  - Kiro Desktop auth (`prod.{region}.auth.desktop.kiro.dev/refreshToken`)
  - AWS SSO OIDC auth (`oidc.{region}.amazonaws.com/token`)
  - Environment variable fallback (`KIRO_REFRESH_TOKEN`)
- **Multi-account failover** with error classification (fatal vs recoverable)
- **Debug request/response logging** (dump to files when `LOG_LEVEL=debug`)
- Mock backend (existing) for local development
- YAML + env var configuration
- Unit and integration tests

## Out of Scope (unchanged)

- Browser UI
- OpenAI-compatible API (`/v1/chat/completions`) — deferred to P2
- Docker deployment — deferred to P2
- Persistent conversation storage
- Multi-tenant deployment
- Web search / MCP integration

## Functional Requirements

### FR-1 Runtime Modes (existing, no change)

- `mock` — local development, no credentials required
- `kiro-live` — forward to Kiro upstream

### FR-2 Anthropic Compatibility (existing, enhanced)

Endpoints:
- `POST /v1/messages` — messages with streaming
- `GET /v1/models` — model listing
- `GET /health` — health check

Request fields preserved:
- `system` (string or block array)
- `tools`, `tool_choice`
- Streaming event order
- `tool_use` and `tool_result` payloads

**NEW**: Model names are normalized through a resolution pipeline before mapping.

### FR-3 Smart Model Resolution (NEW)

Model name resolution pipeline:
1. **Normalize**: `claude-sonnet-4-5` → `claude-sonnet-4.5`, strip date suffixes, handle legacy formats
2. **Static map**: Check known model name → CodeWhisperer ID mapping
3. **Passthrough**: Unknown models are sent to Kiro as-is (gateway, not gatekeeper)

### FR-4 Auth Provider Abstraction (ENHANCED)

Auto-detect auth type from credentials file:
- If `clientId` and `clientSecret` are present → AWS SSO (OIDC)
- Otherwise → Kiro Desktop auth

Credential sources (priority order):
1. JSON credentials file (`kiro.cache_dir`)
2. Environment variables (`KIRO_REFRESH_TOKEN`)

### FR-5 Multi-Account Failover (NEW)

- Support multiple credentials (from multiple JSON files or configs)
- On recoverable API errors (429 rate limit, 402 quota), rotate to next credential
- On fatal errors (400 validation, 403 forbidden), return error immediately
- Track per-account health status
- Persist account state across restarts (optional, config-driven)

### FR-6 Proxy API Key Guard (NEW)

- When `proxy_api_key` is configured, require authentication on all API endpoints
- Accept `x-api-key: <key>` header (Anthropic native)
- Accept `Authorization: Bearer <key>` header
- Return Anthropic-style 401 error on auth failure
- Health endpoint remains unauthenticated

### FR-7 Debug Logging (NEW)

When debug mode is enabled:
- Dump incoming Anthropic request body to log
- Dump outgoing Kiro request body to log
- Dump upstream response (or error) to log
- Log all auth refresh operations
- All dumps use structured JSON format

### FR-8 Configuration (ENHANCED)

New configuration fields:

```yaml
runtime:
  mode: mock
  mock_scenario: default
  upstream_endpoint: https://prod.us-east-1.codewhisperer.desktop.kiro.dev

security:
  proxy_api_key: ""  # empty = no auth required

kiro:
  cache_dir: ~/.aws/sso/cache
  enable_multi_account: true
  refresh_interval: 60

logging:
  level: info
  debug_dump: false  # dump request/response to files
```

New environment variables:
- `KIRO_PROXY_API_KEY`
- `KIRO_DEBUG_DUMP`

## Non-Functional Requirements

- Startup must fail fast with actionable error messages.
- Mock mode startup under 1 second.
- Streaming responses flush incrementally.
- Test suite runs without external credentials.
- Auth refresh must not block request processing for more than 10 seconds.
- API key comparison must be constant-time to prevent timing attacks.

## Acceptance Criteria (v2)

1. All existing tests pass.
2. `go run ./cmd/server` succeeds in mock mode with proxy API key guard enabled.
3. Model name normalization handles at least: `claude-sonnet-4-5`, `claude-sonnet-4.5`, `claude-sonnet-4-5-20250929`, `claude-3-7-sonnet`, `claude-3-7-sonnet-20250219`.
4. Auth provider auto-detects Kiro Desktop vs AWS SSO.
5. Multi-account failover rotates on 429/402 errors.
6. Unauthenticated requests return 401 when API key is configured.
7. Health endpoint returns 200 without auth.
