# Usage

This document is aligned with the current Go implementation in `cmd/server` and `internal/config`.

## 1. Start Claude Code against the gateway

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

## 2. Configure kiro-live

Set `runtime.mode` to `kiro-live` in YAML:

```yaml
runtime:
  mode: kiro-live
  upstream_endpoint: https://prod.us-east-1.codewhisperer.desktop.kiro.dev
```

Or override it from the environment:

```powershell
$env:KIRO_CLAUDE_MODE = "kiro-live"
```

The live backend loads credentials from one of these sources:

- `kiro.cache_dir`, default `~/.aws/sso/cache`
- `KIRO_REFRESH_TOKEN` with optional `KIRO_REGION`

## 3. Config file and environment variable precedence

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
- `KIRO_UPSTREAM_ENDPOINT`
- `LOG_LEVEL`
- `HTTP_PROXY`
- `HTTPS_PROXY`
- `KIRO_REFRESH_TOKEN`
- `KIRO_REGION`

Notes on current implementation:

- `logging.format` exists in YAML but has no environment override
- `proxy.socks5_proxy` exists in YAML and is used by the live HTTP client, but there is no dedicated env override for it
- `kiro.enable_multi_account` and `kiro.refresh_interval` exist in YAML, but current runtime behavior does not expose dedicated env overrides for them


