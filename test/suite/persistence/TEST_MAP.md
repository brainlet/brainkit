# Persistence Test Map

**Purpose:** Verifies that deployments, schedules, metadata, and secrets survive kernel restarts via SQLite store, including corrupt store recovery.
**Tests:** 20 functions across 4 files
**Entry point:** `persistence_test.go` → `Run(t, env)`
**Campaigns:** transport (amqp, redis, postgres, nats, sqlite)

## Files

### store.go — Deploy persistence, metadata preservation, corrupt store recovery

| Function | Purpose |
|----------|---------|
| testDeploySurvivesRestart | Deploys a service, closes kernel, reopens with same store, verifies the service auto-redeploys and responds to messages |
| testTeardownRemovesFromStore | Deploys then tears down a service, reopens kernel, verifies torn-down deployment is not restored |
| testOrderPreserved | Deploys 3 services in order, verifies LoadDeployments returns them sorted by deploy_order |
| testFailedRedeployDoesNotBlock | Injects a broken deployment into the store alongside a working one, verifies kernel starts and the working service still responds |
| testPackageNameSurvivesRestart | Deploys with WithPackageName, closes kernel, verifies packageName persisted in store |
| testRedeployPreservesMetadata | Deploys with packageName + role, redeploys with new code, verifies both metadata fields survive in the store |
| testWithRestoringSkipsPersist | Deploys with WithRestoring flag, verifies the store remains empty (restore-path deployments skip persistence) |
| testRolePreservedAcrossRestart | Deploys with WithRole("admin"), closes kernel, reopens, verifies the stored role field is "admin" |
| testScheduleCatchUpOnRestart | Creates a one-time schedule, closes before it fires, waits past fire time, reopens kernel, verifies the schedule was fired and deleted |
| testRecurringScheduleRestartsCorrectly | Creates a recurring "every 1h" schedule, closes kernel, reopens, verifies schedule restored with correct expression and topic |
| testDeployOrderPreservedExactly | Deploys alpha/beta/gamma, closes kernel, loads from store, verifies exact order and monotonic deploy_order values |
| testCorruptDeploymentTable | Injects corrupt deployments (throwing code, binary garbage, 1MB code) into SQLite, reopens kernel, verifies it survives and valid deployments still work |
| testCorruptScheduleTable | Injects corrupt schedules (invalid expression, empty topic, negative duration) into SQLite, reopens kernel, verifies it starts without panic |

### schedule.go — Schedule-specific persistence across restart

| Function | Purpose |
|----------|---------|
| testScheduleSurvivesRestart | Creates a recurring schedule, closes kernel, reopens, verifies schedule restored with correct topic |
| testMissedRecurringCatchUp | Seeds a schedule with NextFire 2 hours in the past, opens kernel, verifies NextFire was advanced to the future |
| testExpiredOneTimeFires | Seeds an expired one-time schedule, opens kernel, waits briefly, verifies it was fired then deleted from both memory and store |

### backend_matrix.go — Ported adversarial persistence tests

| Function | Purpose |
|----------|---------|
| testDeployPersistRestart | Deploys via kernel API (not SDK), closes, reopens with same store, verifies deployment source appears in ListDeployments |
| testSecretsSurviveRestart | Sets a secret via SDK bus message, closes kernel, reopens with same key, verifies the secret value is retrievable |
| testMultiDeployOrderAndMetadata | Deploys 3 services with different metadata (role, packageName), closes, reopens, verifies all 3 sources present |
| testMultipleSchedulesSurvive | Creates 3 schedules (hourly, 5min, 24h one-time), closes, reopens, verifies at least 2 survive (one-time may have fired) |
| testDeployWithBusHandlerSurvivesRestart | Deploys a service with bus.on handler, closes, reopens, sends a message to the handler, verifies it replies |

## Cross-references

- **Campaigns:** transport/{amqp,redis,postgres,nats,sqlite}_test.go
- **Related domains:** secrets (encrypted persistence), scheduling (schedule persistence)
- **Fixtures:** persistence-related TS fixtures
