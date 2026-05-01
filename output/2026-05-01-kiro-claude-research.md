# Kiro Claude Reverse Proxy — Research Update

Date: 2026-05-01
Phase: research (evolve)
Work Mode: `evolve` — the repository has a working baseline; this phase identifies gaps vs mature implementations

## Reference Project: jwadow/kiro-gateway

- Repository: https://github.com/jwadow/kiro-gateway
- Stars: 1.1k | Forks: 274 | 195 commits | Python (FastAPI) | AGPL-3.0
- Latest release: v2.3 (2026-02-03)
- Supports: Claude Code, OpenCode, Cursor, Cline, Roo Code, Kilo Code, Obsidian, LangChain, OpenAI SDK

### Key Features in kiro-gateway That Our Project Lacks

| Feature | kiro-gateway | kiro-claude (current) | Gap Severity |
|---------|-------------|----------------------|-------------|
| **Smart Model Resolution** (fuzzy name normalization: dashes→dots, strip dates, legacy formats) | ✅ 4-layer pipeline | ❌ Static map only | **HIGH** |
| **OpenAI-compatible API** (`/v1/chat/completions`) | ✅ Full | ❌ None | **MEDIUM** |
| **Multi-account failover** | ✅ With auto-recovery, state persistence | ⚠️ TokenManager has rotation but no failover on API errors | **HIGH** |
| **AWS SSO (OIDC) auth** | ✅ Auto-detect Kiro Desktop vs AWS SSO | ⚠️ Kiro Desktop only | **HIGH** |
| **kiro-cli SQLite DB credentials** | ✅ Reads from `data.sqlite3` | ❌ None | **MEDIUM** |
| **Proxy API key (auth guard)** | ✅ `PROXY_API_KEY` with `x-api-key` + `Authorization: Bearer` | ❌ No proxy auth | **MEDIUM** |
| **Streaming truncation recovery** | ✅ Synthetic messages | ❌ None | **LOW** |
| **Debug logging middleware** | ✅ Request/response dump to files | ⚠️ SimpleLogger only | **MEDIUM** |
| **Error classification** (fatal vs recoverable) | ✅ `classify_error()` + error type enum | ⚠️ Basic HTTP status passthrough | **HIGH** |
| **Anthropic API key header** (`x-api-key`) | ✅ Both `x-api-key` and `Authorization: Bearer` | ❌ No auth check at all | **MEDIUM** |
| **Model info caching** (ListAvailableModels) | ✅ Dynamic from Kiro API | ❌ Hardcoded list | **MEDIUM** |
| **VPN/Proxy support** (SOCKS5, HTTP, HTTPS) | ✅ With `NO_PROXY` localhost exclusion | ⚠️ Basic proxy config, no NO_PROXY | **LOW** |
| **Docker deployment** | ✅ Dockerfile + docker-compose | ❌ None | **LOW** |
| **Connection pooling** | ✅ httpx shared async client | ⚠️ Per-request http.Client | **MEDIUM** |
| **Graceful shutdown** | ✅ State save + async cleanup | ⚠️ Basic signal handler | **LOW** |
| **Web search / MCP integration** | ✅ Native + emulated | ❌ Out of scope | N/A |

### Architecture Insights from kiro-gateway

1. **Auth auto-detection**: Checks for `clientId`/`clientSecret` in credentials JSON to distinguish Kiro Desktop vs AWS SSO (OIDC). Desktop auth uses `prod.{region}.auth.desktop.kiro.dev/refreshToken`. SSO uses `oidc.{region}.amazonaws.com/token`.

2. **Model resolution pipeline**: 4 layers — (0) alias → (1) normalize → (2) dynamic cache → (3) hidden models → (4) passthrough. Never rejects unknown models; lets Kiro decide.

3. **Account failover loop**: On recoverable errors (429, 402), automatically tries next account. On fatal errors (400 validation), returns immediately. Persists account state to JSON for cross-restart recovery.

4. **Streaming architecture**: Uses per-request HTTP clients for streaming (prevents CLOSE_WAIT on VPN disconnect) and shared pooled clients for non-streaming requests.

5. **Kiro API endpoint**: `{api_host}/generateAssistantResponse` — the upstream is NOT a plain Anthropic-compatible endpoint.

6. **Region auto-detection**: API region can differ from SSO region. Configurable via `KIRO_API_REGION`.

## Current Repository Baseline Audit

### What Works Well

1. **Clean Go architecture**: Backend adapter interface (`Client`) with mock and live implementations.
2. **Runtime factory**: Clean mode selection (mock / kiro-live) with fallback behavior.
3. **Request/response conversion**: Bidirectional Anthropic ↔ CodeWhisperer format conversion.
4. **Mock backend**: Comprehensive tool-use scenarios (default, tool-chain, tool-result-error).
5. **SSE streaming**: Both mock and live backends support streaming with proper event normalization.
6. **Test coverage**: Unit tests for converter, handler, mock backend, runtime factory, and config.
7. **Configuration**: YAML file + env var overrides with clear precedence.

### Critical Gaps to Close

1. **Auth robustness**: AWS SSO (OIDC) auth path missing; only Kiro Desktop auth supported.
2. **Model resolution**: Static mapping will break when Kiro adds new models. Need dynamic + fuzzy matching.
3. **Error handling at API boundary**: No account failover; no error classification.
4. **Missing proxy auth**: Anyone on the network can hit the proxy — need API key guard.
5. **Request/response logging**: No debug dump capability for troubleshooting live traffic issues.
6. **Upstream endpoint path**: Hard-coded `/api/v1/messages` path. kiro-gateway uses `/generateAssistantResponse`.

## Recommended Scope for This Iteration

### P0 — Must Have

1. **AWS SSO (OIDC) auth support** — auto-detect auth type from credentials file
2. **Smart model name resolution** — fuzzy normalization + passthrough
3. **Proxy API key guard** — `x-api-key` and `Authorization: Bearer` header support
4. **Error classification** — fatal vs recoverable error handling
5. **Multi-account failover** — rotate on recoverable API errors
6. **Verify upstream endpoint path** — confirm `/api/v1/messages` vs `/generateAssistantResponse`

### P1 — Should Have

7. **Request/response debug logging** — dump to files for troubleshooting
8. **kiro-cli SQLite credential source** — read tokens from kiro-cli DB
9. **Dynamic model listing** — cache from `ListAvailableModels` API
10. **Connection pooling improvement** — reuse HTTP clients across requests

### P2 — Nice to Have

11. **OpenAI-compatible API** (`/v1/chat/completions`)
12. **Docker support**
13. **NO_PROXY localhost exclusion**
