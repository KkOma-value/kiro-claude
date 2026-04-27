# Kiro Claude Reverse Proxy PRD

Date: 2026-04-28
Phase: research

## Product Goal

Deliver a local reverse proxy that exposes an Anthropic-compatible API for Claude Code while routing requests to either:

- a live Kiro upstream when credentials exist
- a local mock backend when credentials do not exist

## Primary User

A developer who wants Claude Code to use a Kiro-backed model path but cannot rely on live Kiro credentials during day-to-day development.

## Success Criteria

1. The service starts cleanly without Kiro credentials when mock mode is enabled.
2. Claude Code can point `ANTHROPIC_BASE_URL` at the proxy and receive valid responses.
3. Both non-streaming and streaming message flows are supported.
4. The same Anthropic-facing API surface works in mock mode and live mode.
5. Configuration makes the active runtime mode explicit.

## In Scope

- Local HTTP server
- Anthropic-compatible `/v1/messages`
- Anthropic-compatible SSE streaming
- model listing endpoint
- configurable runtime mode
- credential provider abstraction
- mock upstream for no-token development
- focused tests for request conversion, streaming, and startup behavior

## Out of Scope

- Browser UI
- Full Kiro account management UI
- Persistent conversation storage
- Multi-tenant deployment
- Production-grade observability stack

## Functional Requirements

### FR-1 Runtime Modes

The proxy must support:

- `mock`
- `kiro-live`

`mock` must not require any token file, refresh token, or network call to Kiro auth services.

### FR-2 Anthropic Compatibility

The proxy must accept:

- `POST /v1/messages`
- `GET /v1/models`
- `GET /health`

The proxy should preserve:

- `system`
- `tools`
- `tool_choice`
- streaming event order
- tool-use and tool-result payloads

### FR-3 Mock Backend

The mock backend must:

- return valid Anthropic message responses
- support SSE events for stream mode
- optionally emit tool-use flows for integration testing
- allow fixed fixtures for deterministic tests

### FR-4 Live Kiro Backend

The live backend must:

- load credentials from cache or env
- refresh access tokens when needed
- route requests through configured HTTP/HTTPS/SOCKS5 proxy settings
- surface upstream errors in Anthropic-style envelopes

### FR-5 Configuration

Configuration must allow:

- runtime mode selection
- upstream endpoint override
- cache directory override
- refresh token via env
- proxy settings
- log level

## Non-Functional Requirements

- Startup failure must be explicit and actionable.
- Mock mode startup must be under 1 second on a normal developer machine.
- Streaming responses must flush incrementally.
- Test suite must run without external credentials.

## Acceptance Criteria

The first implementation milestone is complete when:

1. `go test ./...` passes with no Kiro token present.
2. `go run ./cmd/server` succeeds in `mock` mode.
3. Hitting `/v1/messages` returns Anthropic-compatible JSON in non-streaming mode.
4. Hitting `/v1/messages` with `stream=true` returns Anthropic-compatible SSE frames.
5. Switching to `kiro-live` mode is done only by config and env changes.
