# Kiro Claude Reverse Proxy — Execution Plan

Date: 2026-05-01
Phase: spec

## Execution Order

Implementation follows dependency order: foundation packages first, then integration.

### Sprint 1: Model Resolution (no external deps)
1. Create `internal/model/resolver.go` — normalization pipeline + static map
2. Create `internal/model/models.go` — model maps moved from `gateway/types.go`
3. Create `internal/model/resolver_test.go` — 20+ normalization test cases
4. Update `gateway/types.go` — remove model maps, import from model package
5. Update `api/converter.go` — use model resolver instead of direct map

### Sprint 2: Auth Provider Abstraction
6. Create `internal/auth/provider.go` — Provider interface + AuthType enum
7. Create `internal/auth/kiro_desktop.go` — refactor existing token refresh into provider
8. Create `internal/auth/aws_sso.go` — new OIDC auth provider
9. Create `internal/auth/detector.go` — auto-detect auth type from credential shape
10. Update `internal/auth/model.go` — add ClientID/ClientSecret fields
11. Update `internal/auth/sso.go` — use provider interface for credential loading

### Sprint 3: Middleware Layer
12. Create `internal/middleware/auth.go` — proxy API key guard
13. Create `internal/middleware/logger.go` — debug request/response logger
14. Update `internal/config/config.go` — add Security and debug config fields

### Sprint 4: Multi-Account Manager
15. Create `internal/account/errors.go` — error classification (fatal vs recoverable)
16. Create `internal/account/manager.go` — multi-account failover logic
17. Create `internal/account/manager_test.go` — failover test cases

### Sprint 5: Integration & Wiring
18. Update `internal/backend/live.go` — use Provider interface + account manager
19. Update `internal/gateway/client.go` — accept Provider instead of TokenManager
20. Update `internal/runtime/factory.go` — wire all new components
21. Update `cmd/server/main.go` — apply middleware, enhanced startup banner
22. Update `docs/config.example.yaml` — document new config fields
23. Update `docs/usage.md` — document new features

### Sprint 6: Testing & Verification
24. Run `go build ./...` — verify compilation
25. Run `go test ./...` — verify all tests pass
26. Manual smoke test with mock mode
27. Update README.md
