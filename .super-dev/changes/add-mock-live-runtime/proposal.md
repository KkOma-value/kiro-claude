# Proposal: Add mock/live runtime split for Kiro Claude reverse proxy

## Why

The proxy currently assumes live Kiro credentials are available at startup. That blocks development and testing on machines that do not have a valid Kiro token.

## Change

Introduce an explicit runtime mode with two backends:

- `mock`: starts without Kiro credentials and returns deterministic Anthropic-compatible responses
- `kiro-live`: uses the existing Kiro credential and upstream gateway path

## Scope

- add runtime config
- add backend abstraction
- add mock backend implementation
- wrap current gateway client in a live backend adapter
- wire mode selection in server bootstrap
- extend tests for mock mode and config behavior

## Expected Outcome

Developers can point Claude Code at the proxy during local development without any Kiro token, then switch to live Kiro behavior later via config only.
