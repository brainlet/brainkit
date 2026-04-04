# Test Reorganization Migration Progress

**Branch:** `feature/test-reorg-foundation`
**Last updated:** 2026-04-04 (session 2)

## Summary

**580 / 629 tests migrated (92%)** across 20 suite domains. All compile and pass.

## Phase 2: Suite Domain Extraction — Status

| Domain | Tests | Status | Notes |
|--------|-------|--------|-------|
| agents | 9 | COMPLETE | +4 AI tests (skip without OPENAI_API_KEY) |
| bus | 77 | COMPLETE | +20 adversarial gap tests |
| cli | 11 | COMPLETE | |
| cross | 35 | COMPLETE | New domain (cross-kit, plugins, discovery, node commands) |
| deploy | 38 | COMPLETE | +18 adversarial gap tests |
| fs | 11 | COMPLETE | |
| gateway | 56 | COMPLETE | New domain (routes, stream, advanced, errors, attacks) |
| health | 36 | COMPLETE | +22 new tests (probes, degraded, shutdown_adv) |
| mcp | 3 | COMPLETE | Multi-backend belongs in campaigns |
| packages | 3 | COMPLETE | |
| persistence | 16 | COMPLETE | New domain (store, schedule persistence) |
| rbac | 35 | COMPLETE | New domain (enforcement, bridge, matrix) |
| registry | 30 | COMPLETE | +13 adversarial gap tests |
| scheduling | 6 | COMPLETE | Persistence schedule tests moved to persistence domain |
| secrets | 21 | COMPLETE | +12 adversarial gap tests |
| security | 96 | COMPLETE | New domain (sandbox, data leakage, bus forgery, etc.) |
| stress | 54 | COMPLETE | New domain (GC, scaling, concurrent, exhaustion) |
| tools | 11 | COMPLETE | +5 adversarial gap tests |
| tracing | 8 | COMPLETE | |
| workflows | 24 | COMPLETE | New domain (commands, storage, concurrent, developer) |

## Remaining ~49 tests

The remaining ~49 tests (629 - 580) fall into categories that belong in **campaigns** (Phase 4) or are accounted for in other ways:

- `test/transport/` (2 files) → campaigns/transport (multi-backend matrix)
- `test/auth/` (1 file, 10 tests) → campaigns/auth (auth × backend matrix)
- `test/adversarial/backend_advanced_test.go` (4) → campaigns/transport
- `test/adversarial/backend_matrix_test.go` (6) → campaigns/transport
- `test/adversarial/persistence_matrix_test.go` (4) → campaigns/transport
- `test/adversarial/rbac_backend_test.go` (2) → campaigns/transport
- `test/adversarial/surface_matrix_test.go` (4) → campaigns/transport
- `test/adversarial/crosskit_matrix_test.go` (2) → campaigns/crosskit
- `test/adversarial/failure_cascade_test.go` (10) → split bus/health (partially migrated)
- `test/bench/bench_test.go` (0 tests, benchmarks only) → bench structure
- `test/fixtures/` (4 files) → Phase 3 fixture runner rewrite
- `test/e2e/scenarios_test.go` (remaining subtests) → already partially migrated

## Phase Status

| Phase | Status |
|-------|--------|
| Phase 1: Foundation (TestEnv) | COMPLETE |
| Phase 2: Suite Domain Extraction | **92% COMPLETE** (580/629) |
| Phase 3: Fixture Runner Rewrite | NOT STARTED |
| Phase 4: Campaign Infrastructure | NOT STARTED |
| Phase 5: Cleanup + Bench | NOT STARTED |

## Git State

```
Branch: feature/test-reorg-foundation
Key commits:
  922693b feat: fill PARTIAL domain gaps — bus (+20), secrets (+12), registry (+13), tools (+5), deploy (+18), agents (+4)
  d616e28 feat: extract suite/security (96 tests), suite/stress (56 tests), suite/cross (35 tests)
  67b8926 feat: extract suite/persistence domain (16 tests) + complete suite/health gaps (22 new tests)
  2a17c77 feat: extract suite/gateway domain (56 tests from infra + adversarial)
  1e27e8f feat: extract suite/workflows domain (24 tests from infra/workflow_bus_test.go)
  b44e7eb feat: extract suite/rbac domain (35 tests from infra + adversarial)
  ... (earlier Phase 2 commits from session 1)
  6f35b4c feat: add TestEnv foundation for suite-based testing
```

All existing tests still pass. New suite tests coexist with old tests. No old files deleted.
