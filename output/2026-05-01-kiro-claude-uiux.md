# Kiro Claude Reverse Proxy — UIUX v2

Date: 2026-05-01
Phase: docs
Baseline: 2026-04-28 UIUX + kiro-gateway operator experience analysis

This project remains backend-first. UI/UX is defined as operator and developer experience at the terminal, config file, and HTTP API level.

## Experience Goal

A developer should be able to:
1. Start the proxy in mock mode in **under 60 seconds** from a fresh clone
2. Switch to live Kiro mode with **one config change**
3. Debug a failing request with **clear, structured logs**
4. Trust that the proxy **won't silently break** when Kiro adds new models

## Primary Flows

### Flow 1: First-Time Setup (Mock Mode)

```
$ git clone ... && cd kiro-claude
$ go run ./cmd/server -config docs/config.example.yaml

[INFO] Starting Kiro → Claude Code gateway
[INFO] Runtime mode: mock
[INFO] Active backend: local mock (default scenario)
[INFO] Auth guard: disabled (no proxy_api_key configured)
[INFO] Listening on http://127.0.0.1:8000
[INFO] Claude Code setup: export ANTHROPIC_BASE_URL=http://127.0.0.1:8000
```

### Flow 2: Live Mode with Kiro Desktop Auth

```yaml
# config.yaml
runtime:
  mode: kiro-live
kiro:
  cache_dir: ~/.aws/sso/cache
security:
  proxy_api_key: my-secret-key-123
```

```
$ go run ./cmd/server -config config.yaml

[INFO] Starting Kiro → Claude Code gateway
[INFO] Runtime mode: kiro-live
[INFO] Auth type: Kiro Desktop (auto-detected)
[INFO] Loaded 2 credential(s)
[INFO] Using credential 1/2, expires at 2026-05-01T15:00:00Z
[INFO] Auth guard: enabled
[INFO] Active backend: https://prod.us-east-1.codewhisperer.desktop.kiro.dev
[INFO] Listening on http://127.0.0.1:8000
```

### Flow 3: Live Mode with AWS SSO

```
[INFO] Starting Kiro → Claude Code gateway
[INFO] Runtime mode: kiro-live
[INFO] Auth type: AWS SSO (OIDC, auto-detected)
[INFO] Loaded 1 credential(s) from kiro-cli SSO cache
[INFO] Auth guard: enabled
[INFO] Active backend: https://prod.us-east-1.codewhisperer.desktop.kiro.dev
[INFO] Listening on http://127.0.0.1:8000
```

### Flow 4: Account Failover

```
[WARN] Kiro upstream returned 429 (rate limit) for account 1/2
[INFO] Rotating to account 2/2
[INFO] Retry succeeded on account 2/2
```

### Flow 5: Model Resolution Transparency

```
[DEBUG] Model resolution: 'claude-sonnet-4-5-20250929' → normalized: 'claude-sonnet-4.5' → ID: 'CLAUDE_SONNET_4_5_20250514_V1_0' (source: static)
[DEBUG] Model resolution: 'my-custom-model' → normalized: 'my-custom-model' → ID: 'my-custom-model' (source: passthrough)
```

## Operator UX Requirements

### Startup Logs

Startup logs must clearly print:
- Runtime mode
- Auth type (auto-detected)
- Number of credentials loaded
- Auth guard status (enabled/disabled)
- Active backend endpoint
- Bind address
- Claude Code setup instruction

Startup logs must **never** print:
- Raw access tokens
- Raw refresh tokens
- API key value (mask as `my-s***-123`)

### Error Messages

Errors must be actionable:

| Situation | Message |
|-----------|---------|
| Mock mode, no credentials | `mock mode selected; no credential required` |
| Live mode, no credentials | `kiro-live mode selected but no cache credential or KIRO_REFRESH_TOKEN found. Tip: set allow_start_without_token: true to fall back to mock mode` |
| Auth refresh failed | `upstream auth refresh failed for credential 1/2 (Kiro Desktop): HTTP 401. Rotating to next credential.` |
| All accounts exhausted | `all credentials exhausted after failover. Last error: 429 rate limit on account 2/2` |
| API key missing | `401 authentication_error: Invalid or missing API key. Use x-api-key header or Authorization: Bearer.` |
| Unknown model (passthrough) | `Model 'my-model' not in static map, passing through to Kiro API` |

### Config UX

Configuration should be obvious from one file:

```yaml
server:
  host: 127.0.0.1
  port: 8000

runtime:
  mode: mock
  mock_scenario: default
  upstream_endpoint: https://prod.us-east-1.codewhisperer.desktop.kiro.dev
  allow_start_without_token: true

security:
  proxy_api_key: ""  # empty = no auth required

kiro:
  cache_dir: ~/.aws/sso/cache
  enable_multi_account: true
  refresh_interval: 60

proxy:
  http_proxy: ""
  https_proxy: ""
  socks5_proxy: ""

logging:
  level: info
  format: text
  debug_dump: false
```

## Response UX for Claude Code

Even in mock mode, responses must:
- Match valid Anthropic response schema exactly
- Use valid SSE frame format
- Return stable model identifiers that match what was requested
- Include tool-use fixtures for integration testing
- Return properly normalized model names in responses (not CodeWhisperer IDs)

## API Error Response UX

All API errors follow the Anthropic error envelope:

```json
{
  "type": "error",
  "error": {
    "type": "authentication_error",
    "message": "Invalid or missing API key."
  }
}
```

Error types:
- `authentication_error` — invalid or missing API key
- `invalid_request_error` — malformed request (400)
- `api_error` — upstream failure or internal error

## Debug Dump UX

When `logging.debug_dump: true`:

```
[DEBUG] === Incoming Request ===
[DEBUG] POST /v1/messages
[DEBUG] Model: claude-sonnet-4.5 (resolved from claude-sonnet-4-5-20250929)
[DEBUG] Stream: true
[DEBUG] Messages: 3 (user→assistant→user)
[DEBUG] Tools: 2 (list_files, read_file)

[DEBUG] === Outgoing Kiro Request ===
[DEBUG] Model: CLAUDE_SONNET_4_5_20250514_V1_0
[DEBUG] Endpoint: https://prod.us-east-1.codewhisperer.desktop.kiro.dev/api/v1/messages
[DEBUG] Auth: Kiro Desktop (token expires in 42m)

[DEBUG] === Upstream Response ===
[DEBUG] Status: 200
[DEBUG] Content blocks: 1 (text)
[DEBUG] Stop reason: end_turn
[DEBUG] Tokens: input=32, output=128
```

## Quality Bar

The proxy should feel like a **deliberate, production-grade tool**, not a one-off script.

Key UX signals:
- **Fast startup** — under 1 second for mock, under 3 seconds for live
- **Explicit mode** — never ambiguous about what runtime mode is active
- **Predictable behavior** — same request always produces same result in mock mode
- **Readable errors** — every error message tells the operator what to do next
- **Transparent resolution** — model name transformations are logged at DEBUG level
- **Secure by default intent** — auth guard is easy to enable, documented prominently
