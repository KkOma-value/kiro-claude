# Kiro Claude Reverse Proxy Research

Date: 2026-04-28
Phase: research
Repository: `D:\kiro-claude`

## Objective

Build a local reverse proxy that lets Claude Code speak Anthropic-compatible HTTP while the proxy forwards requests to Kiro's upstream service. The proxy must remain developable on a machine that has no usable Kiro token.

## Current Repository State

The repository already contains a minimal Go service with these modules:

- `cmd/server/main.go`: boots HTTP server and wires config, auth, gateway, and API handler.
- `internal/api/*`: Anthropic request/response types, request conversion, HTTP handlers.
- `internal/auth/*`: Kiro credential loading and refresh support.
- `internal/gateway/*`: upstream HTTP client and SSE reader.
- `internal/config/config.go`: server, proxy, and cache settings.

On 2026-04-28, the codebase was brought to a buildable baseline:

- `go build ./...` passes.
- `go test ./...` passes.
- local `/health` smoke test passes.

## What Works Now

- Anthropic-style `POST /v1/messages` and `GET /v1/models` endpoints exist.
- Request conversion now handles:
  - model mapping
  - string and block message content
  - `system` string vs structured system payload
  - `tool_choice`
  - tool-result error metadata
- Streaming response conversion now normalizes event names and delta names into Anthropic-style snake_case.
- Handler error responses now use Anthropic-style JSON envelopes instead of plain text.
- Upstream client now supports configured proxy transport selection.
- Auth bootstrap can fall back to env-provided refresh token if cache credentials do not exist.

## Main Product Constraint

The user has no local Kiro token. That changes the development target:

1. We cannot treat live upstream connectivity as the development loop.
2. The proxy must support an offline development mode.
3. Live-mode auth must be isolated behind a provider boundary so the rest of the proxy is testable without secrets.

## Gaps Still Open

These are the remaining gaps between "builds" and "usable reverse proxy":

### 1. Auth strategy is not development-friendly enough

- The current startup path still assumes Kiro credentials are needed at runtime.
- There is no explicit `mock` or `disabled-auth` mode.
- There is no fixture-based upstream simulation for message and stream traffic.

### 2. Gateway behavior is too tightly coupled to one upstream path

- Upstream endpoint and auth flow are hardcoded for Kiro production.
- There is no interface boundary for multiple backends:
  - live Kiro
  - mock backend
  - replay backend for recorded sessions

### 3. Anthropic compatibility is still incomplete at the edges

- No explicit `/v1/complete` compatibility path if Claude Code or adjacent tooling falls back to older APIs.
- No robust upstream error classification beyond HTTP status and message.
- No request/response tracing to debug compatibility issues.

### 4. No operator workflow for zero-token setup

- No sample config for offline mode.
- No documented startup flow for "develop now, attach token later".
- No canned integration fixtures that let Claude Code hit the proxy and receive deterministic responses.

## Recommended Direction

The next implementation phase should target a two-mode runtime:

### Mode A: `mock`

Used when there is no Kiro token.

- Starts successfully with no credential bootstrap.
- Returns deterministic Anthropic-compatible responses.
- Supports both non-streaming and SSE streaming.
- Can emulate tool-use flows for Claude Code integration testing.

### Mode B: `kiro-live`

Used when Kiro refresh token or cached credentials are available.

- Reuses the same Anthropic-facing handlers.
- Uses a dedicated auth provider and upstream transport.
- Can be enabled entirely by config or env vars.

## Research Conclusion

The correct next step is not "keep hardcoding Kiro auth until it works." The correct step is to separate the Anthropic-facing proxy surface from the live Kiro transport so the project can be developed and verified without secrets.
