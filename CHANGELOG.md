# Changelog

## Unreleased

### Session 01 — Phase 0 Cleanup

Pure subtraction. No new API, no behavior changes — only removal of orphaned
code from prior feature deletions.

Removed:
- `test/suite/rbac/` domain (RBAC was removed previously; the stranded test
  suite still lived in-tree)
- `internal/engine/scaling.go` — `InstanceManager`, `PoolConfig`, `PoolMode`,
  `PoolSharded`, `PoolReplicated`, `pool`, `StaticStrategy`
- `internal/types/scaling.go` — `ScalingStrategy`, `ScalingDecision`,
  `PoolInfo` types
- Scaling re-exports in root `types.go`: `InstanceManager`, `PoolConfig`,
  `StaticStrategy`, `ScalingDecision`, `ScalingStrategy`, `PoolInfo`,
  `PoolMode`, `PoolSharded`, `PoolReplicated`, `NewInstanceManager`,
  `NewStaticStrategy`
- `Kit.HealthJSON` public method and `Kernel.HealthJSON` — the `kit.health`
  bus command marshals `Kernel.Health(ctx)` inline; `gateway/health.go`
  drops its `healthJSONer` branch and always uses the `alive + ready`
  fallback on `/health`
- `test/suite/stress/scaling.go` and its 7 pool/strategy tests
- `testStorageRuntimeScalingPool` in `test/suite/registry/storage_runtime.go`
- `testHealthJSON` in `test/suite/gateway/routes.go`
- `testConcurrencyRBACAssignCheckRace`, `testTimingRoleChangeWhileHandlerRunning`,
  `testBusRateLimitExceeds`, `testErrorContractBusNotConfiguredRBAC`,
  `testRolePreservedAcrossRestart` — all were RBAC-era stubs that only `t.Skip`
- `secDeployWithRole` helper in `test/suite/security/run.go`
- `role` parameter on `testutil.DeployWithOpts`
- `rbacOnly` field on `test/suite/bus/surface.go` `cmdTest`
- `rbac.assign` / `rbac.revoke` from the forbidden-topic list in
  `test/suite/security/bus_forgery.go`
- `docs/guides/scaling-and-pools.md` guide
- `test/campaigns/fullstack/nats_postgres_rbac_test.go`
- References to the removed symbols across `docs/`, `TEST_MAP.md`,
  `CLAUDE.md` files, and `internal/docs/FEATURES.md`

Changed:
- `MetricsSnapshot` moved from `internal/types/scaling.go` into
  `internal/types/types.go` (still the same struct; only the owning file
  changed)
- `NotConfiguredError` feature strings referencing `"rbac"` now use `"mcp"`
