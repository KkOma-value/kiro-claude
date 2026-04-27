# Usage

This document is aligned with the current Go implementation in `cmd/server` and `internal/config`.

## 1. Start in mock mode

`mock` is the default runtime mode and is the right choice when you do not have a Kiro token locally.

Example config:

```yaml
runtime:
  mode: mock
  mock_scenario: default
```

Start the server:

```powershell
go run ./cmd/server -config .\docs\config.example.yaml
```

Smoke test:

```powershell
Invoke-RestMethod http://127.0.0.1:8000/health
Invoke-RestMethod http://127.0.0.1:8000/v1/models
```

Behavior in the current implementation:

- `mock_scenario: default` returns a deterministic text response when no tools are provided, and emits `tool_use` when the incoming request includes `tools`
- `mock_scenario: tool-use` behaves like the default tool-first flow and can be used as an explicit scenario label
- `mock_scenario: tool-chain` asks for a second tool after the first successful `tool_result`
- `mock_scenario: tool-result-error` retries a tool after an error-marked `tool_result`
- if any tool-focused scenario is selected but the request has no tools, the mock backend falls back to a normal text response

## 2. Start Claude Code against the gateway

Set `ANTHROPIC_BASE_URL` in the same shell session where you launch Claude Code.

PowerShell:

```powershell
$env:ANTHROPIC_BASE_URL = "http://127.0.0.1:8000"
claude
```

macOS or Linux:

```bash
export ANTHROPIC_BASE_URL=http://127.0.0.1:8000
claude
```

If you changed `server.host` or `server.port`, update the URL to match.

## 3. Switch to kiro-live

Set `runtime.mode` to `kiro-live` in YAML:

```yaml
runtime:
  mode: kiro-live
  upstream_endpoint: https://prod.us-east-1.codewhisperer.desktop.kiro.dev
  allow_start_without_token: false
```

Or override it from the environment:

```powershell
$env:KIRO_CLAUDE_MODE = "kiro-live"
```

The live backend loads credentials from one of these sources:

- `kiro.cache_dir`, default `~/.aws/sso/cache`
- `KIRO_REFRESH_TOKEN` with optional `KIRO_REGION`

When `allow_start_without_token: true`, the current runtime factory logs a warning and falls back to `mock` if credentials cannot be loaded. When it is `false`, process startup fails.

## 4. Config file and environment variable precedence

The current binary starts with internal defaults, then applies the YAML config file, then applies environment variable overrides.

Default config path used by `cmd/server`:

```text
~/.config/kiro-claude/config.yaml
```

If that file does not exist, startup continues with defaults plus any environment overrides.

Environment variables currently supported by the Go implementation:

- `KIRO_CLAUDE_HOST`
- `KIRO_CLAUDE_PORT`
- `KIRO_CACHE_DIR`
- `KIRO_CLAUDE_MODE`
- `KIRO_CLAUDE_MOCK_SCENARIO`
- `KIRO_UPSTREAM_ENDPOINT`
- `KIRO_ALLOW_START_WITHOUT_TOKEN`
- `LOG_LEVEL`
- `HTTP_PROXY`
- `HTTPS_PROXY`
- `KIRO_REFRESH_TOKEN`
- `KIRO_REGION`

Notes on current implementation:

- `logging.format` exists in YAML but has no environment override
- `proxy.socks5_proxy` exists in YAML and is used by the live HTTP client, but there is no dedicated env override for it
- `kiro.enable_multi_account` and `kiro.refresh_interval` exist in YAML, but current runtime behavior does not expose dedicated env overrides for them

## 5. Example mode switches

Switch to a tool-use mock flow without touching YAML:

```powershell
$env:KIRO_CLAUDE_MODE = "mock"
$env:KIRO_CLAUDE_MOCK_SCENARIO = "tool-use"
go run ./cmd/server
```

Switch to a multi-step tool chain:

```powershell
$env:KIRO_CLAUDE_MODE = "mock"
$env:KIRO_CLAUDE_MOCK_SCENARIO = "tool-chain"
go run ./cmd/server
```

Run live mode but permit fallback to mock during local development:

```powershell
$env:KIRO_CLAUDE_MODE = "kiro-live"
$env:KIRO_ALLOW_START_WITHOUT_TOKEN = "true"
go run ./cmd/server
```

Run live mode and fail fast if credentials are missing:

```powershell
$env:KIRO_CLAUDE_MODE = "kiro-live"
$env:KIRO_ALLOW_START_WITHOUT_TOKEN = "false"
go run ./cmd/server
```
