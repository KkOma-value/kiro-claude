# kiro-claude

`kiro-claude` exposes an Anthropic-compatible HTTP surface for Claude Code and can run in two runtime modes:

- `mock`: local development with no Kiro token required
- `kiro-live`: forward requests to the real Kiro upstream

The current server exposes:

- `POST /v1/messages`
- `GET /v1/models`
- `GET /health`

## Quick start

Start the gateway with the example config:

```powershell
go run ./cmd/server -config .\docs\config.example.yaml
```

The default listen address in the current implementation is `http://127.0.0.1:8000`.

Point Claude Code at the gateway by setting `ANTHROPIC_BASE_URL` in the same shell where you launch Claude Code:

```powershell
$env:ANTHROPIC_BASE_URL = "http://127.0.0.1:8000"
claude
```

For macOS or Linux:

```bash
export ANTHROPIC_BASE_URL=http://127.0.0.1:8000
claude
```

## Runtime modes

### `mock`

`mock` is the default mode. It is intended for local integration work when you do not have a Kiro token yet.

- No Kiro credential is required
- `runtime.mock_scenario: default` returns deterministic text when no tools are provided, and starts a tool loop when tools are present
- `runtime.mock_scenario: tool-use` follows the same tool-first behavior and remains available as an explicit scenario name
- `runtime.mock_scenario: tool-chain` asks for a follow-up tool after the first successful `tool_result`
- `runtime.mock_scenario: tool-result-error` retries a tool after an error-marked `tool_result`

### `kiro-live`

`kiro-live` attempts to load Kiro credentials and forward requests to `runtime.upstream_endpoint`.

Credential sources supported by the current Go implementation:

- cached JSON credentials under `kiro.cache_dir`
- environment variables `KIRO_REFRESH_TOKEN` and `KIRO_REGION`

If `runtime.allow_start_without_token` is `true`, startup falls back to `mock` when live credentials cannot be loaded. If it is `false`, startup fails instead.

## Configuration

The server accepts a YAML config file through `-config`. The default config path in the current binary is:

```text
~/.config/kiro-claude/config.yaml
```

A complete example is in [docs/config.example.yaml](D:\kiro-claude\docs\config.example.yaml) and operational notes are in [docs/usage.md](D:\kiro-claude\docs\usage.md).
