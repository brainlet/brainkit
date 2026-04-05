# Stress Test Map

**Purpose:** Verifies kernel stability under extreme concurrency, resource exhaustion, pool scaling, GC cleanup, and multi-surface simultaneous operations.
**Tests:** 55 functions across 7 files
**Entry point:** `stress_test.go` → `Run(t, env)`
**Campaigns:** none (memory-only domain)

## Files

### gc.go — GC and memory pressure (5 tests)

| Function | Purpose |
|----------|---------|
| testGCSingleKernelCleanClose | Creates and closes a single kernel, verifies no error on Close |
| testGCMultipleKernelCleanClose | Creates and closes 5 kernels sequentially, verifies all close cleanly |
| testGCTenKernelCleanClose | Creates and closes 10 kernels sequentially, verifies all close cleanly |
| testGCZeroLeakQuickJSMemory | Creates raw QuickJS runtime, allocates JS objects, closes context + RunGC, logs allocation counts before/after to verify memory freed |
| testGCZeroLeakSESRuntime | Creates full brainkit kernel (SES + Mastra bundle), exercises EvalTS, closes, verifies clean shutdown with no errors |

### scaling.go — Pool scaling and strategy (8 tests)

| Function | Purpose |
|----------|---------|
| testPoolSpawnAndKill | Spawns pool with 2 instances, verifies PoolInfo shows 2, kills pool, verifies NotFoundError on subsequent queries |
| testPoolScaleUpDown | Spawns pool with 1, scales +2 (total 3), scales -2 (total 1), scales -10 (total 0), verifies counts at each step |
| testPoolDuplicateAndNotFound | Verifies AlreadyExistsError on duplicate pool spawn, NotFoundError on scale/kill/info for nonexistent pool |
| testPoolSharedTools | Creates pool with shared registry, registers a tool on the shared registry, spawns 2-instance pool, verifies tool resolvable |
| testStrategyStatic | Tests StaticStrategy.Evaluate: below target returns scale-up, at target returns none, above target returns scale-down |
| testStrategyThreshold | Tests ThresholdStrategy.Evaluate with various pending/current/min/max combinations, verifies correct scale-up/down/none decisions |
| testPoolEvaluateAndScale | Spawns pool with StaticStrategy(3), calls EvaluateAndScale, verifies pool scaled from 1 to 3, second call keeps at 3 |
| testPoolInstancesProcessMessages | Spawns pool with shared "stress-ping" tool, calls the tool via executor, verifies {pong: "ok"} response |

### concurrent.go — Concurrent operation tests (9 tests)

| Function | Purpose |
|----------|---------|
| testParallelDeploy | Deploys 10 services from 10 goroutines simultaneously, verifies all 10 appear in ListDeployments |
| testParallelPublish | Deploys echo handler, sends 10 messages from 10 goroutines, verifies all 10 get responses |
| testParallelEvalTS | Runs 10 EvalTS calls from 10 goroutines, each returning unique JSON, verifies all results match expected |
| testDeployDuringHandler | Triggers a slow (500ms) handler, deploys another service concurrently, verifies no deadlock within 10s |
| testTeardownDuringHandler | Triggers a slow (300ms) handler, tears down the deployment concurrently, verifies no deadlock within 10s |
| testDeployTeardownRaceOnSameSource | Deploys a service, concurrently teardown + redeploy on the same source, verifies both complete without deadlock |
| testStressDeployTeardownCycles | 5 goroutines each do 3 deploy/teardown cycles on unique sources, verifies no deployments remain after |
| testRedeployRace | Deploys v0, 3 goroutines concurrently redeploy v1/v2/v3, verifies exactly 1 deployment survives |
| testDeployDuringDrain | Enables drain, deploys a service, verifies either error or deployed (no panic), restores drain=false |

### concurrency.go — Adversarial concurrency races (12 tests)

| Function | Purpose |
|----------|---------|
| testConcurrencyDeployTeardownRace | 10 source pairs: concurrent deploy + teardown on same source, logs error counts |
| testConcurrencyPublishUnsubscribeRace | Deploys handler, 20 goroutines publish to it concurrently, verifies no panic |
| testConcurrencySecretSetGetRace | 50 goroutines each do concurrent secrets.set + secrets.get on same key, verifies no panic |
| testConcurrencyMassDeployTeardown | Deploys 10 services simultaneously, tears down all 10 simultaneously, verifies none remain |
| testConcurrencyScheduleUnscheduleRace | 20 goroutines each schedule + immediately unschedule, verifies all cancelled and list is empty |
| testConcurrencyCloseDuringHandlers | Deploys slow handler, sends it a message, calls Close 50ms later, verifies Close succeeds |
| testConcurrencyParallelEvalTS | 5 goroutines run EvalTS simultaneously, verifies all return correct unique results |
| testConcurrencyStorageAddRemoveRace | 20 pairs of concurrent AddStorage + RemoveStorage on same name, verifies no panic |
| testConcurrencyMetricsDuringChurn | Background deploy/teardown loop, foreground reads Metrics() 50 times, verifies PumpCycles >= 0 |
| testConcurrencySharedSQLiteStore | Two kernels sharing same SQLite store, each deploys 5 services concurrently, verifies both alive |
| testConcurrencyDeployDuringRestore | Persists 5 deployments, reopens kernel, immediately deploys 3 new services, verifies kernel alive |
| testConcurrencyRBACAssignCheckRace | 50 goroutines alternate between rbac.assign/revoke and EvalTS bus.publish, verifies no panic |

### concurrency_stress.go — Heavy load stress tests (7 tests)

| Function | Purpose |
|----------|---------|
| test100DeploysSimultaneously | 100 goroutines each deploy a unique .ts, verifies most succeed and kernel stays alive, tears down all |
| test1000BusPublishes | 100 goroutines each publish 10 raw messages (1000 total), verifies majority delivered, kernel alive |
| testSecretRotationDuringReads | Sets secret, 50 goroutines read continuously, 1 goroutine rotates 10 times, verifies no panic and reads > 0 |
| testDeployWhileEvalTS | Concurrent EvalTS x20 + deploy/teardown x20, verifies kernel alive |
| testToolCallsUnderLoad | 100 concurrent tool calls to "echo", verifies majority succeed under load |
| testScheduleStorm | Creates 50 schedules all firing in 200ms, verifies majority fire within 3s |
| testMultiSurfaceSimultaneous | Go SDK tool calls + .ts handler messages + EvalTS all running concurrently (20 each), verifies kernel alive |

### exhaustion.go — Resource exhaustion attacks (16 tests)

| Function | Purpose |
|----------|---------|
| testExhaustionMemoryBomb | Deploys .ts allocating 100x 1MB strings, verifies kernel survives |
| testExhaustionStackOverflow | Deploys .ts with 500-deep recursion, verifies kernel survives |
| testExhaustionPromiseFlood | Deploys .ts creating 10,000 chained promises, verifies kernel survives |
| testExhaustionDeployBomb | 50 goroutines simultaneously deploy services that register tools + handlers, verifies kernel survives |
| testExhaustionFetchBomb | Deploys .ts doing 100 fetch() calls to localhost:1 (fails fast), verifies kernel survives |
| testExhaustionLifecycleChurn | 100 sequential deploy/teardown cycles on same source with different code, verifies no deployments remain |
| testExhaustionOutputBomb | Deploys .ts calling output() with 10MB string, verifies kernel survives |
| testExhaustionConcurrentEvalTS | 100 goroutines run EvalTS simultaneously, verifies kernel survives |
| testExhaustionLargePayloadViaJS | Deploys .ts publishing 5MB JSON via bus.publish, verifies kernel survives |
| testExhaustionTimerBomb | Deploys .ts creating 10,000 setTimeout(fn, 1) calls, verifies kernel survives after 2s |
| testExhaustionSecretValueBomb | Stores a 10MB secret value, verifies kernel survives |
| testExhaustionJSONStringifyHijack | Deploys .ts replacing JSON.stringify with function returning 100MB string, verifies kernel survives |
| testExhaustionFilesystemFill | Deploys .ts writing 100x 1MB files, verifies kernel survives |
| testExhaustionPumpStarvation | Deploys .ts with setTimeout(fn, 0) loop x50000, waits 3s, deploys another service, verifies it still works |
| testExhaustionPersistenceBomb | Deploys 100 services with persistence, closes, reopens, verifies all 100 restored |
| testEvalTSInfiniteLoop | Deploys while(true){}, calls Close from goroutine, verifies Close completes within 15s (JS interrupted) |

### e2e_stress.go — E2E stress scenarios (2 tests)

| Function | Purpose |
|----------|---------|
| testE2EMultipleKernels | Creates 3 independent kernels, each with an echo tool, calls each tool, verifies all respond independently |
| testE2EConcurrentOperations | 3 goroutines call "add" tool concurrently with different inputs, collects all results, verifies correct sums |

## Cross-references

- **Campaigns:** none (memory-only domain)
- **Related domains:** all domains (stress tests exercise tools, secrets, scheduling, deploy, RBAC, persistence)
- **Fixtures:** none (stress tests are Go-driven)
