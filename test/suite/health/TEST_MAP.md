# Health Test Map

**Purpose:** Verifies kernel health checks, shutdown/drain lifecycle, metrics reporting, health probes (AI, storage, vector), and degraded health scenarios
**Tests:** 36 functions across 6 files
**Entry point:** `health_test.go` → `Run(t, env)`
**Campaigns:** transport (all 5), fullstack (all 3)

## Files

### checks.go — Core health check operations

| Function | Purpose |
|----------|---------|
| testAliveWhenRunning | Calls kernel.Alive(), asserts true on a running kernel |
| testReadyWhenRunning | Calls kernel.Ready(), asserts true on a running kernel |
| testReadyFalseWhenDraining | Sets draining, calls Ready(), asserts false, clears draining, asserts true again |
| testStatusRunning | Calls kernel.Health(), asserts Status="running" and Healthy=true |
| testTransportProbe | Calls kernel.Health(), verifies the transport check is present and healthy |
| testStorageBridgeCheck | Creates kernel with storage bridges, checks Health() includes storage check |
| testStatusDraining | Sets draining, checks Health(), asserts Status="draining" |
| testDeploymentsCount | Deploys N services, checks Health().Deployments count matches |

### shutdown.go — Kernel shutdown and drain tests

| Function | Purpose |
|----------|---------|
| testDrainsBeforeClose | Deploys a service, calls GracefulShutdown, verifies drain happens before close |
| testDrainTimeoutForcesClose | Sets very short drain timeout, deploys service, calls GracefulShutdown, verifies close completes |
| testCloseStillWorks | Calls kernel.Close() directly (no drain), verifies it succeeds |
| testMessagesDroppedDuringDrain | Sets draining, publishes, verifies messages are dropped (no handler response) |
| testEvalTSWorksDuringDrain | Sets draining, calls EvalTS, verifies it still works (drain only affects bus handlers) |

### metrics.go — Kernel metrics

| Function | Purpose |
|----------|---------|
| testMetricsReflectsState | Deploys services, checks Metrics() reflects deployment count, pump cycles, and schedule count |

### probes.go — Health probes for external services

| Function | Purpose |
|----------|---------|
| testProbeAIProviderRealOpenAI | Runs AI provider probe with real OPENAI_API_KEY, asserts healthy (requires API key) |
| testProbeAIProviderBadKey | Runs AI provider probe with invalid key, asserts unhealthy |
| testProbeAIProviderNotRegistered | Runs AI provider probe when no provider is registered, asserts appropriate status |
| testProbeStorageInMemory | Runs storage probe on in-memory storage, asserts healthy |
| testProbeVectorStoreRealPgVector | Runs vector store probe against real PgVector (requires Podman+Postgres), asserts healthy |
| testProbeAll | Runs all probes, verifies the combined health report structure |
| testProbePeriodicTicker | Starts periodic probe ticker, waits, verifies probes execute on schedule |

### degraded.go — Adversarial degraded health scenarios

| Function | Purpose |
|----------|---------|
| testAliveAfterHeavyLoad | Deploys many services and fires many messages, asserts kernel still alive |
| testReadyToggleDuringDrain | Rapidly toggles draining on/off, asserts Ready() tracks correctly |
| testFullHealthCheckCategories | Checks Health() response includes all expected categories (transport, runtime, etc.) |
| testHealthWithTracingStore | Creates kernel with TraceStore, checks health includes tracing category |
| testHealthWithStorageBridges | Creates kernel with storage bridges, checks health includes storage bridge category |
| testMetricsReflectDeployments | Deploys/tears down, checks Metrics().Deployments tracks correctly |
| testUptimeIncreases | Checks Health().Uptime at two points in time, asserts it increases |
| testHealthAfterClose | Closes kernel, checks Health(), asserts unhealthy or appropriate error state |
| testPersistenceStoreHealth | Creates kernel with SQLite store, checks health includes persistence category |

### shutdown_adv.go — Adversarial shutdown scenarios

| Function | Purpose |
|----------|---------|
| testShutdownGracefulWithActiveDeployments | Deploys multiple services, calls GracefulShutdown, verifies all cleaned up |
| testShutdownWithActiveSchedules | Creates schedules, shuts down, verifies schedules are cleaned |
| testShutdownWithActiveSubscriptions | Creates bus subscriptions, shuts down, verifies subscriptions are cleaned |
| testShutdownDrainTimeoutAdv | Tests drain timeout with a handler that blocks, verifies forced close after timeout |
| testShutdownConcurrentClose | Calls Close() from multiple goroutines simultaneously, asserts no panic |
| testShutdownStorageAccessBeforeClose | Accesses storage during shutdown sequence, verifies no panic |

## Cross-references

- **Campaigns:** `transport/{sqlite,nats,postgres,redis,amqp}_test.go`, `fullstack/{redis_mongodb,amqp_postgres_vector,nats_postgres_rbac}_test.go`
- **Related domains:** bus (drain affects bus handlers), deploy (deployment count in health), gateway (health endpoints)
- **Fixtures:** none
