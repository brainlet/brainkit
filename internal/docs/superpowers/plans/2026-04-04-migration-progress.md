# Test Reorganization Migration Progress

**Branch:** `feature/test-reorg-foundation`
**Last session:** 2026-04-04

## What Was Done

### Phase 1: Foundation — COMPLETE
- `test/suite/env.go` — TestEnv, EnvConfig, 17 EnvOption functions, Full/Minimal/NewEnv presets, shared helpers (Deploy, EvalTS, PublishAndWait, SendAndReceive, ResponseCode, ResponseHasError, RequireAI, RequirePodman)
- `test/suite/env_test.go` — 7 smoke tests
- Deleted 5 dead helpers from testutil (NewTestKernelWithTransport, NewTestKernelWithStorageAndBackend, RestartKernelWithStore, NewTestKernelPair, NewTestKernelWithStorage)

### Phase 2: Suite Domain Extraction — PARTIALLY COMPLETE

**Domains marked "complete" below only migrated from infra/ sources. Adversarial sources were DEFERRED in most cases. This is the main gap.**

| Domain | Tests migrated | Status | What's missing |
|--------|---------------|--------|----------------|
| bus | 58 | PARTIAL | Missing: adversarial/cross_feature_test.go (10 tests), adversarial/error_contract_test.go (bus subset ~5), adversarial/bus_command_matrix_test.go RBAC subtest (1) |
| deploy | 21 | PARTIAL | Missing: adversarial/input_abuse_test.go deploy subset (8), adversarial/state_corruption_test.go deploy subset (~3), adversarial/e2e_scenarios_test.go:DeployLifecycle (1), surface/ts_test.go deploy subset (~3) |
| scheduling | 7 | COMPLETE for non-persistence | Persistence tests (SurvivesRestart, MissedCatchUp, ExpiredOneFires) belong in persistence domain |
| fs | 12 | COMPLETE | All infra + adversarial fs tests migrated |
| tools | 7 | PARTIAL | Missing: adversarial/input_abuse_test.go tools subset (4), adversarial/e2e_scenarios_test.go:ToolPipeline (1) |
| agents | 6 | PARTIAL | Missing: Deploy_Agent_Then_List (needs AI key), surface/ts_test.go AI tests |
| tracing | 9 | COMPLETE | All infra + adversarial tracing tests migrated |
| mcp | 4 | PARTIAL | Original iterated AllBackends — suite only runs on memory. Multi-backend belongs in campaigns |
| packages | 4 | COMPLETE | All infra/package_deploy_test.go migrated |
| secrets | 10 | PARTIAL | Missing: adversarial/secrets_matrix_test.go (7), adversarial/input_abuse_test.go secrets subset (4), infra/integration_test.go:SecretsRotation (1) |
| cli | 12 | COMPLETE | All e2e/cli_cobra + cli_commands migrated |
| registry | 18 | PARTIAL | Missing: adversarial/storage_runtime_test.go (9), adversarial/input_abuse_test.go registry subset (4) |
| health | 15 | PARTIAL | Missing: infra/probe_test.go (7), adversarial/health_degraded_test.go (9), adversarial/shutdown_test.go (6) |
| rbac | 0 | NOT STARTED | Directory exists but empty. Source: infra/rbac_test.go (12), adversarial/rbac_enforcement_test.go (14), adversarial/rbac_matrix_test.go (9) |
| gateway | 0 | NOT STARTED | Source: infra/gateway_test.go (23), infra/gateway_ratelimit_test.go (1), infra/stream_test.go (8), adversarial/gateway_advanced_test.go (7), adversarial/gateway_errors_test.go (9), adversarial/gateway_attack_test.go (8 gateway-specific) |
| workflows | 0 | NOT STARTED | Source: infra/workflow_bus_test.go (24) |
| persistence | 0 | NOT STARTED | Source: infra/persistence_test.go (11), schedule persistence (4), workflow crash recovery (2), adversarial/persistence_attack_test.go (2 edge cases) |
| security | 0 | NOT STARTED | Source: 78 cross-domain probes from adversarial/ (sandbox_escape, data_leakage, bus_forgery, cross_deployment, internal_exploit, rbac_escape, reply_token, timing_attack, secret_exfiltration, gateway_attack subset, persistence_attack subset, state_corruption subset) |
| stress | 0 | NOT STARTED | Source: infra/gc_debug_test.go (5), infra/scaling_test.go (8), concurrent/concurrent_test.go (9), adversarial/concurrency_test.go (12), adversarial/concurrency_stress_test.go (7), adversarial/resource_exhaustion_test.go (15) |
| cross | 0 | NOT STARTED | Source: cross/ts_go_test.go (1), cross/plugin_go_test.go (1), cross/ts_plugin_test.go (1), plugin/inprocess_test.go (1), plugin/subprocess_test.go (1), adversarial/crosskit_matrix_test.go (2→campaigns), adversarial/node_commands_test.go (8), adversarial/plugin_surface_test.go (6), adversarial/discovery_test.go (5), infra/discovery_test.go (1) |

### Total: ~183 tests migrated out of 629 (29%)

## Source Files NOT Yet Migrated

Every file below still needs to be migrated per the Appendix A migration matrix in the plan. The "Destination" column shows which suite domain or campaign each file maps to.

### test/adversarial/ — 44 files, ~364 test functions
**NONE of these have been migrated yet** (except the ones specifically listed as migrated above: bus_error_paths_test.go, deploy_edge_cases_test.go, fs_matrix_test.go, tracing_test.go, logging_test.go partial).

Files that HAVE been migrated from adversarial/:
- bus_error_paths_test.go (10) → suite/bus/errors.go ✓
- deploy_edge_cases_test.go (14) → suite/deploy/edge_cases.go ✓
- fs_matrix_test.go (8) → suite/fs/operations.go ✓
- tracing_test.go (5) → suite/tracing/spans.go ✓
- logging_test.go (3 partial) → suite/bus/log.go ✓
- bus_command_matrix_test.go (3 of 4 — RBAC test missing) → suite/bus/surface.go ✓

Files that have NOT been migrated from adversarial/:
- backend_advanced_test.go (4) → campaigns/transport
- backend_matrix_test.go (6) → campaigns/transport
- bus_forgery_test.go (12) → suite/security
- concurrency_stress_test.go (7) → suite/stress
- concurrency_test.go (12) → suite/stress
- cross_deployment_attack_test.go (10) → suite/security
- cross_feature_test.go (10) → suite/bus
- crosskit_matrix_test.go (2) → campaigns/crosskit
- data_leakage_test.go (8) → suite/security
- discovery_test.go (5) → suite/cross
- e2e_scenarios_test.go (7) → split across bus/deploy/tools/stress
- error_contract_test.go (13) → suite/bus
- failure_cascade_test.go (10) → split bus/health
- gateway_advanced_test.go (7) → suite/gateway
- gateway_attack_test.go (12) → split gateway (8) / security (4)
- gateway_errors_test.go (9) → suite/gateway
- health_degraded_test.go (9) → suite/health
- helpers_test.go (0) → already handled via suite/env.go SendAndReceive
- input_abuse_test.go (24) → split deploy(8)/secrets(4)/tools(4)/registry(4)/bus(4)
- internal_exploit_test.go (13) → suite/security
- node_commands_test.go (8) → suite/cross
- persistence_attack_test.go (6) → split security(4)/persistence(2)
- persistence_matrix_test.go (4) → campaigns/transport
- plugin_surface_test.go (6) → suite/cross
- rbac_backend_test.go (2) → campaigns/transport
- rbac_enforcement_test.go (14) → suite/rbac
- rbac_escape_test.go (9) → suite/security
- rbac_matrix_test.go (9) → suite/rbac
- reply_token_test.go (7) → suite/security
- resource_exhaustion_test.go (15) → suite/stress
- sandbox_escape_test.go (10) → suite/security
- secret_exfiltration_test.go (7) → suite/security
- secrets_matrix_test.go (7) → suite/secrets
- shutdown_test.go (6) → suite/health
- state_corruption_test.go (7) → split deploy/security
- storage_runtime_test.go (9) → suite/registry
- surface_matrix_test.go (4) → campaigns/transport
- timing_attack_test.go (10) → suite/security

### test/infra/ — files NOT yet migrated
- probe_test.go (7) → suite/health
- integration_test.go (remaining 3: ScheduleDuringDrain→scheduling, SecretsRotation→secrets, DeployOrderRestart→persistence)
- persistence_test.go (11) → suite/persistence
- gateway_test.go (23) → suite/gateway
- gateway_ratelimit_test.go (1) → suite/gateway
- stream_test.go (8) → suite/gateway
- rbac_test.go (12) → suite/rbac
- workflow_bus_test.go (24) → suite/workflows
- scaling_test.go (8) → suite/stress
- gc_debug_test.go (5) → suite/stress

### Other test/ directories NOT yet migrated
- test/transport/ (2 files) → campaigns/transport
- test/auth/ (1 file, 10 tests) → campaigns/auth
- test/cross/ (3 files) → suite/cross
- test/plugin/ (2 files) → suite/cross
- test/concurrent/ (1 file, 9 tests) → suite/stress
- test/surface/ts_test.go (11 tests) → split across deploy/agents/bus/security
- test/e2e/scenarios_test.go (4 tests) → split across tools/deploy/bus/stress
- test/bench/bench_test.go (0 test functions, benchmarks only) → bench structure

### Phases 3-5: NOT STARTED
- Phase 3 (Fixture runner rewrite): Not started
- Phase 4 (Campaigns infrastructure): Not started
- Phase 5 (Cleanup + bench): Not started

## How to Continue

1. Read this file and the migration matrix in `internal/docs/superpowers/plans/2026-04-04-test-reorganization.md` (Appendix A)
2. Read the spec at `internal/docs/superpowers/specs/2026-04-03-test-reorganization-design.md`
3. For each domain marked PARTIAL above, read the source files listed in "What's missing" and migrate them
4. For each domain marked NOT STARTED, follow the same Task 3 pattern: create package, migrate tests, write entry point, verify
5. Key learning from this session: **tests that check global state (ListSchedules, bus.handler.failed events) need fresh kernels via suite.Full(t) to avoid cross-test pollution on the shared kernel**
6. Key learning: **source names in Deploy calls must be unique across all tests sharing a kernel — use suffixes like `-edge`, `-adv`, etc.**

## Git State

```
Branch: feature/test-reorg-foundation
Commits from main:
  9dbba42 feat: extract suite/health domain (14 tests)
  c6f4821 feat: extract suite/registry domain (17 tests)
  8ba93f9 feat: extract suite/secrets domain (9 tests)
  1bddfab feat: extract suite/packages domain (3 tests)
  0e22e5a feat: extract suite/cli domain (11 tests)
  f357666 feat: extract suite/tracing domain (8 tests)
  ee1f700 feat: extract suite/mcp domain (3 tests)
  2d43c19 feat: extract suite/agents domain (5 tests)
  f6fad1d feat: extract suite/tools domain (6 tests)
  ae5ce32 feat: extract suite/fs domain (11 tests)
  7980a1d feat: extract suite/scheduling domain (6 tests)
  d85ae03 feat: extract suite/deploy domain (20 tests)
  dbf577d feat: complete suite/bus domain core (57 tests)
  ... (earlier bus commits)
  6f35b4c feat: add TestEnv foundation for suite-based testing
```

All existing tests still pass. New suite tests coexist with old tests. No old files deleted.
