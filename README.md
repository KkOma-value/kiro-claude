<div align="center">
  <h1>Kiro Claude Proxy</h1>
  <p>A production-grade reverse proxy exposing an Anthropic-compatible API for Claude Code, routing to AWS CodeWhisperer (Kiro).</p>
</div>

---

## ✨ Features

- 🧠 **Smart Model Resolution**: Automatically normalizes model names (e.g., `claude-sonnet-4-5-20250929` → `claude-sonnet-4.5`) while passing unknown models through to Kiro.
- 🛡️ **API Key Guard**: Secure your local proxy using standard `x-api-key` or `Authorization: Bearer` headers.
- 🔑 **Auto-Detect Auth**: Seamlessly supports both Kiro IDE Desktop (`~/.aws/sso/cache`) and AWS SSO (OIDC) credentials without manual configuration.
- 🔄 **Multi-Account Failover**: Automatically rotates credentials on recoverable errors (e.g., 429 Rate Limit, 402 Quota Exceeded).
- 🐛 **Debug Logging**: Built-in request/response dumping for easy troubleshooting.

## 🚀 Quick Start

### 1. Prerequisites

You need valid Kiro credentials from one of the following sources:
- Kiro IDE Desktop (cached under `~/.aws/sso/cache`)
- AWS SSO (OIDC) credentials via `kiro-cli login`
- Environment variable `KIRO_REFRESH_TOKEN`

### 2. Start the Gateway

Run the server using the provided example configuration:

```bash
go run ./cmd/server -config ./docs/config.example.yaml
```

*By default, the server listens on `http://127.0.0.1:8000`.*

### 3. Connect Claude Code

In a new terminal window, configure Claude Code to use your local proxy:

**macOS / Linux:**
```bash
export ANTHROPIC_BASE_URL="http://127.0.0.1:8000"
claude
```

**Windows (PowerShell):**
```powershell
$env:ANTHROPIC_BASE_URL = "http://127.0.0.1:8000"
claude
```

## ⚙️ Configuration

The server accepts a YAML configuration file via the `-config` flag. The default path is `~/.config/kiro-claude/config.yaml`.

See [`docs/config.example.yaml`](docs/config.example.yaml) for a complete example.

### Security

To protect your proxy from unauthorized access on your local network, set an API key:

```yaml
# config.yaml
security:
  proxy_api_key: "your-secret-key"
```

When set, all API endpoints (except `/health`) require authentication. Claude Code will automatically send this key if you configure it, or you can pass it manually via `x-api-key` or `Authorization: Bearer`.

### Environment Variables

You can override configuration using the following environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `KIRO_CLAUDE_HOST` | Bind host | `127.0.0.1` |
| `KIRO_CLAUDE_PORT` | Bind port | `8000` |
| `KIRO_UPSTREAM_ENDPOINT`| Kiro upstream URL | `https://prod.us-east-1...` |
| `KIRO_CACHE_DIR` | SSO cache directory | `~/.aws/sso/cache` |
| `KIRO_REFRESH_TOKEN` | Manual refresh token | - |
| `KIRO_REGION` | Kiro region | `us-east-1` |
| `KIRO_PROXY_API_KEY` | Proxy API key | - |
| `KIRO_DEBUG_DUMP` | Enable debug logging (`true`/`false`) | `false` |
| `HTTP_PROXY` / `HTTPS_PROXY`| Proxy settings | - |

## 📡 API Endpoints

The proxy exposes the following Anthropic-compatible endpoints:

- `POST /v1/messages` — Message generation (supports both streaming and non-streaming)
- `GET /v1/models` — List available models
- `GET /health` — Health check

## 📖 Credential Resolution

The gateway loads credentials in the following priority order:
1. Cached JSON credentials under `kiro.cache_dir`
2. Environment variables (`KIRO_REFRESH_TOKEN` + `KIRO_REGION`)

The authentication provider (Desktop vs. AWS SSO) is automatically determined based on the presence of `clientId` and `clientSecret` in the payload.
