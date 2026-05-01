<div align="center">

# Kiro Claude Proxy

**A reverse proxy that lets Claude Code talk to Kiro (AWS CodeWhisperer) — with a built-in web management panel**

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev)
[![Next.js](https://img.shields.io/badge/Next.js-16-black?style=flat&logo=next.js)](https://nextjs.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

Expose an [Anthropic-compatible Messages API](https://docs.anthropic.com/en/api/messages) on `localhost`, transparently routing requests to the Kiro upstream service via AWS CodeWhisperer credentials — plus a React dashboard to monitor and configure the proxy in real time.

</div>

---

## How It Works

```
┌──────────────┐        ┌───────────────────┐        ┌──────────────────────┐
│  Claude Code │──API──▶│  Kiro Claude Proxy │──────▶│  Kiro (CodeWhisperer) │
│  (client)    │◀──SSE──│  localhost:8000    │◀──────│  AWS upstream         │
└──────────────┘        └───────────────────┘        └──────────────────────┘
                                  │
                         ┌────────▼────────┐
                         │  Next.js Panel  │
                         │  localhost:3000  │
                         └─────────────────┘
```

The **Go backend** accepts standard Anthropic `/v1/messages` requests, converts them to Kiro's `generateAssistantResponse` format, and streams responses back as Anthropic-compatible SSE events.

The **Next.js frontend** is an optional management panel that proxies requests to the backend and provides a real-time dashboard, configuration editor, model explorer, and interactive API tester — all accessible at `http://localhost:3000`.

## Features

### Proxy (Go Backend)

| Feature | Description |
|:--------|:------------|
| **Model Resolution** | Normalizes model names — `claude-sonnet-4-5-20250929`, `Claude-Sonnet-4-5`, `claude-3-5-sonnet-20241022` all route to the correct upstream model |
| **Multi-Account Failover** | Loads multiple credentials from `~/.aws/sso/cache`; on 429 / 402 / 5xx, automatically rotates to the next account |
| **API Key Guard** | Optional authentication via `x-api-key` or `Authorization: Bearer` on all endpoints (except `/health`) |
| **Streaming & Non-Streaming** | Full support for both `"stream": true` (SSE) and `"stream": false` (JSON) responses |
| **Auto-Detect Auth** | Seamlessly supports Kiro IDE Desktop and AWS SSO (OIDC) credentials without manual configuration |
| **Debug Logging** | Built-in request/response body dump for troubleshooting |

### Management Panel (Next.js Frontend)

| Feature | Description |
|:--------|:------------|
| **Dashboard** | Live status overview — runtime mode, auth guard state, account count, uptime |
| **Account Health** | Lists all loaded credentials and their health status |
| **Model Explorer** | Browse available models and test name resolution interactively |
| **API Tester** | Send test messages to the proxy from the browser and inspect the response |
| **Config Editor** | View and understand the active configuration |
| **Health Monitor** | Real-time `/health` endpoint status |
| **Dark Mode** | System-aware dark/light theme via `next-themes` |

---

## Quick Start

### Prerequisites

- **Go 1.24+** — for the backend proxy
- **Node.js 18+** / **npm** — for the frontend panel
- Valid Kiro credentials from **one** of:
  - [Kiro IDE Desktop](https://kiro.dev) — cached automatically under `~/.aws/sso/cache`
  - AWS SSO (OIDC) credentials
  - Environment variable `KIRO_REFRESH_TOKEN`

### 1. Start the Backend Proxy

```bash
git clone https://github.com/yourusername/kiro-claude.git
cd kiro-claude
go run ./cmd/server
```

The proxy starts on `http://127.0.0.1:8000`.

### 2. Start the Frontend Panel (optional)

```bash
cd frontend
npm install
npm run dev
```

The management panel starts on `http://localhost:3000`. It automatically proxies API requests to the Go backend on port 8000.

### 3. Point Claude Code at the Proxy

<details>
<summary><b>macOS / Linux</b></summary>

```bash
export ANTHROPIC_BASE_URL="http://127.0.0.1:8000"
claude
```
</details>

<details>
<summary><b>Windows (PowerShell)</b></summary>

```powershell
$env:ANTHROPIC_BASE_URL = "http://127.0.0.1:8000"
claude
```
</details>

That's it — Claude Code will now use Kiro as its backend.

### 4. (Optional) Use a Config File

```bash
go run ./cmd/server -config ./docs/config.example.yaml
```

The default config path is `~/.config/kiro-claude/config.yaml`. If the file doesn't exist, the server starts with built-in defaults.

---

## Configuration

### Config File

See [`docs/config.example.yaml`](docs/config.example.yaml) for a fully annotated example.

```yaml
server:
  host: 127.0.0.1
  port: 8000

kiro:
  cache_dir: ~/.aws/sso/cache     # Where to find Kiro credentials
  enable_multi_account: true       # Load all credentials for failover

security:
  proxy_api_key: ""                # Set to protect your proxy

logging:
  level: info                      # debug | info | warn | error
  debug_dump: false                # Log full request/response bodies
```

### Environment Variables

Environment variables override the config file:

| Variable | Description | Default |
|:---------|:------------|:--------|
| `KIRO_CLAUDE_HOST` | Bind address | `127.0.0.1` |
| `KIRO_CLAUDE_PORT` | Bind port | `8000` |
| `KIRO_CACHE_DIR` | SSO cache directory | `~/.aws/sso/cache` |
| `KIRO_UPSTREAM_ENDPOINT` | Kiro upstream URL | `https://prod.us-east-1...` |
| `KIRO_REFRESH_TOKEN` | Manual refresh token | — |
| `KIRO_REGION` | Kiro region | `us-east-1` |
| `KIRO_PROXY_API_KEY` | Proxy API key | — |
| `KIRO_DEBUG_DUMP` | Enable debug logging | `false` |
| `HTTP_PROXY` / `HTTPS_PROXY` | Network proxy | — |

### Securing the Proxy

Set an API key to require authentication:

```yaml
security:
  proxy_api_key: "your-secret-key"
```

Or via environment:

```bash
export KIRO_PROXY_API_KEY="your-secret-key"
```

When set, clients must provide the key via `x-api-key` header or `Authorization: Bearer` header. The `/health` endpoint remains unauthenticated.

---

## API Reference

### Proxy Endpoints (port 8000)

| Method | Endpoint | Description |
|:-------|:---------|:------------|
| `POST` | `/v1/messages` | Create a message (streaming or non-streaming) |
| `GET`  | `/v1/models` | List available models |
| `GET`  | `/health` | Health check |
| `GET`  | `/api/status` | Proxy status (used by the frontend panel) |

All endpoints follow the [Anthropic Messages API](https://docs.anthropic.com/en/api/messages) specification.

### Frontend Panel Routes (port 3000)

| Route | Description |
|:------|:------------|
| `/` | Dashboard — proxy status overview |
| `/health` | Health monitor |
| `/models` | Model explorer and name resolver |
| `/api-test` | Interactive API tester |
| `/config` | Active configuration viewer |
| `/guide` | Setup and usage guide |

---

## Credential Resolution

Credentials are loaded in priority order:

1. **Cache files** — JSON files under `kiro.cache_dir` (default: `~/.aws/sso/cache`)
2. **Environment** — `KIRO_REFRESH_TOKEN` + `KIRO_REGION`

The auth method (Kiro Desktop vs. AWS SSO OIDC) is auto-detected based on the presence of `clientId` / `clientSecret` in the credential payload.

When multiple credentials are found, they are all loaded for **failover** — if one account hits a rate limit (429) or quota (402), the proxy automatically rotates to the next.

---

## Project Structure

```
kiro-claude/
├── cmd/
│   └── server/           # Application entrypoint
├── internal/
│   ├── api/              # Anthropic API handler & format conversion
│   ├── auth/             # Credential loading, token refresh, SSO
│   ├── account/          # Multi-account failover manager
│   ├── backend/          # Backend client interface & live implementation
│   ├── config/           # YAML config + env overrides
│   ├── gateway/          # Kiro upstream HTTP client & SSE parser
│   ├── logger/           # Structured logging
│   ├── middleware/        # Auth guard & debug logger
│   ├── model/            # Model name normalization & mapping
│   └── runtime/          # Backend factory
├── frontend/             # Next.js management panel
│   ├── app/
│   │   ├── page.tsx      # Dashboard
│   │   ├── health/       # Health monitor page
│   │   ├── models/       # Model explorer page
│   │   ├── api-test/     # Interactive API tester page
│   │   ├── config/       # Configuration viewer page
│   │   ├── guide/        # Setup guide page
│   │   └── api/          # Next.js API routes (proxies to Go backend)
│   └── components/
│       ├── layout/       # Sidebar & header
│       ├── health/       # Health status components
│       ├── models/       # Model browser components
│       ├── guide/        # Guide content components
│       └── ui/           # shadcn/ui base components
├── docs/                 # Config example & usage docs
├── tests/                # Integration tests
├── go.mod
└── go.sum
```

---

## Development

### Backend

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Build a binary
go build -o kiro-claude ./cmd/server

# Run with debug logging
KIRO_DEBUG_DUMP=true go run ./cmd/server
```

### Frontend

```bash
cd frontend

# Development server (hot reload)
npm run dev

# Type check
npx tsc --noEmit

# Lint
npm run lint

# Production build
npm run build
```

The frontend expects the Go backend to be running on `http://localhost:8000`. Next.js API routes at `/api/*` transparently proxy requests to the backend, so the browser never needs direct access to port 8000.

---

## Tech Stack

| Layer | Technology |
|:------|:-----------|
| Proxy | Go 1.24, standard library |
| Frontend | Next.js 16, React 19, TypeScript |
| UI Components | shadcn/ui, Base UI, Lucide icons |
| Styling | Tailwind CSS v4 |
| Charts | Recharts |
| Syntax highlight | Shiki |
| Theme | next-themes (dark / light / system) |

---

## License

This project is open source. See the [LICENSE](LICENSE) file for details.
