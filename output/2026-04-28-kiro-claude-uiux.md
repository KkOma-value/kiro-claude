# Kiro Claude Reverse Proxy UIUX

Date: 2026-04-28
Phase: research

This project is backend-first. There is no browser UI requirement right now, so UI/UX work is defined as operator and developer experience.

## Experience Goal

A developer should be able to understand and use the proxy in under five minutes, even with no Kiro credentials.

## Primary Flows

### Flow 1: Offline development

1. Set runtime mode to `mock`.
2. Start the server.
3. Point Claude Code at the local base URL.
4. Receive deterministic responses without any Kiro token.

### Flow 2: Switch to live Kiro

1. Provide cache credentials or env refresh token.
2. Change runtime mode to `kiro-live`.
3. Restart the server.
4. Observe clear logs showing live backend activation.

## Operator UX Requirements

### Startup Logs

Startup logs must clearly print:

- runtime mode
- bind address
- active backend
- credential source
- whether streaming is enabled

Startup logs must never print raw access tokens.

### Failure Messages

Errors must be actionable:

- `mock mode selected; no credential required`
- `kiro-live mode selected but no cache credential or KIRO_REFRESH_TOKEN found`
- `upstream auth refresh failed`

Avoid generic messages like `gateway error` when a more specific cause exists.

### Config UX

Configuration should be obvious from one file:

```yaml
server:
  host: 127.0.0.1
  port: 8000

runtime:
  mode: mock

kiro:
  cache_dir: ~/.aws/sso/cache

logging:
  level: info
```

## Response UX for Claude Code

Even in mock mode, responses should feel realistic enough to validate integration:

- valid Anthropic response schema
- valid SSE frames
- stable model identifiers
- optional tool-use fixture for command/tool testing

## Quality Bar

The proxy should feel deliberate and debuggable, not like a fragile one-off bridge script. The main UX requirement is trust: fast startup, explicit mode, predictable behavior, and readable errors.
