# Kiro Claude Reverse Proxy Architecture

Date: 2026-04-28
Phase: research

## Architectural Decision

Split the system into two layers:

1. Anthropic-facing proxy layer
2. backend adapter layer

The Anthropic-facing layer must not know whether responses come from live Kiro or a local mock implementation.

## Proposed Runtime Shape

```text
Claude Code
  -> HTTP Anthropic API
  -> Proxy Server
     -> API Handler
     -> Anthropic <-> Internal conversion
     -> Backend Adapter interface
        -> Mock Backend Adapter
        -> Kiro Live Backend Adapter
           -> Credential Provider
           -> Upstream HTTP Client
```

## Core Interfaces

### `BackendAdapter`

Responsibilities:

- execute non-streaming completion
- execute streaming completion
- expose supported models

Suggested methods:

```go
type BackendAdapter interface {
    SendMessage(ctx context.Context, req *gateway.CodeWhispererRequest) (*gateway.CodeWhispererResponse, error)
    StreamMessage(ctx context.Context, req *gateway.CodeWhispererRequest) (<-chan *gateway.CodeWhispererStreamChunk, <-chan error, error)
    Models(ctx context.Context) []api.ModelData
}
```

### `CredentialProvider`

Responsibilities:

- load refresh-token or cached credential material
- refresh access tokens
- report whether live auth is available

Suggested implementations:

- `CacheCredentialProvider`
- `EnvCredentialProvider`
- `NoopCredentialProvider`

## Module Plan

### `internal/runtime`

New package for mode selection and dependency wiring.

Responsibilities:

- parse runtime mode
- select mock or live adapter
- validate configuration at startup

### `internal/backend/mock`

New package for deterministic local development backend.

Responsibilities:

- generate fixed Anthropic-compatible assistant output
- optionally emit SSE frames
- support fixture-based tool-use scenarios

### `internal/backend/kiro`

New package that wraps the current gateway client.

Responsibilities:

- own live upstream endpoint selection
- own credential refresh path
- translate upstream failures into typed proxy errors

## Configuration Changes

Add a required runtime switch:

```yaml
runtime:
  mode: mock # or kiro-live
  allow_start_without_token: true
```

Recommended optional fields:

```yaml
runtime:
  mode: mock
  mock_scenario: default
  upstream_endpoint: https://prod.us-east-1.codewhisperer.desktop.kiro.dev
```

## Testing Strategy

### Unit

- request conversion
- stream event normalization
- typed error mapping
- config parsing

### Adapter

- mock backend non-streaming
- mock backend streaming
- live adapter startup validation with missing credentials

### Integration

- boot server in mock mode
- hit `/v1/messages`
- hit `/v1/messages` with stream enabled
- assert Anthropic-compatible body and SSE frames

## Migration Strategy

1. Introduce runtime mode and backend adapter interface.
2. Move current gateway client behind the live adapter.
3. Add mock adapter and no-token startup path.
4. Wire server bootstrap to choose adapter from config.
5. Add integration tests for mock mode.
6. Only after mock mode is stable, expand live Kiro behavior.

## Key Decision

Do not make Claude-facing handlers conditional on token presence. Put that decision one layer lower, in runtime assembly and backend selection.
