# Workflows Test Map

**Purpose:** Verifies the Mastra workflow engine: sequential/parallel execution, suspend/resume, cancel, tool calls inside steps, bus events from steps, conditional branching, step state, storage persistence, crash recovery, concurrent starts, and long-running integration.
**Tests:** 24 functions across 4 files
**Entry point:** `workflows_test.go` → `Run(t, env)`
**Campaigns:** transport (amqp, redis, postgres, nats, sqlite), fullstack (nats_postgres_rbac, amqp_postgres_vector, redis_mongodb)

## Files

### commands.go — Happy path + error paths

| Function | Purpose |
|----------|---------|
| testStartSequential | Deploys 2-step workflow (upper -> exclaim), starts with "hello", verifies status "success" |
| testStartParallel | Deploys 2-branch parallel workflow (doubled, tripled), starts with x=5, verifies status "success" |
| testList | Deploys workflow, publishes WorkflowListMsg, verifies the workflow appears in the list with non-empty source |
| testSuspendResume | Deploys workflow with suspend step (needs approval), starts it (suspends), resumes with approved=true, verifies success |
| testCancel | Deploys suspending workflow, starts (suspends), cancels by runID, verifies status "canceled" from storage |
| testWithToolCall | Deploys workflow that calls tools.call("echo") inside a step, starts it, verifies status "success" |
| testNotFound | Starts a nonexistent workflow, verifies "not found" error |
| testResumeNonexistentRun | Resumes a fake runID, verifies "not found" error |
| testStatusNonexistentRun | Queries status for fake runID, verifies "not found" error |
| testCancelNonexistentRun | Cancels a fake runID, verifies "not found" error |
| testStepWithError | Deploys workflow with a step that throws, starts it, verifies status "failed" |

### storage.go — Persistence + storage tests

| Function | Purpose |
|----------|---------|
| testStorageUpgrade | Deploys suspending workflow with SQLite storage, starts (suspends), verifies snapshot is persisted by reading store internals via EvalTS |
| testStatusFromStorage | Deploys fast workflow with SQLite storage, starts (success), queries status, verifies "success" from storage |
| testRuns | Deploys workflow that suspends conditionally, starts 2 runs (one succeeds, one suspends), queries WorkflowRunsMsg, verifies total=2 and filtered suspended=1 |
| testStartAsyncEvent | Starts workflow via WorkflowStartAsyncMsg (non-blocking), subscribes to workflow.completed.{runID} event, verifies completion event fires with correct status |
| testCrashRecoverySuspended | Starts suspending workflow, closes kernel, reopens with same stores, verifies status still "suspended", resumes, verifies "success" |

### concurrent.go — Concurrency + stress

| Function | Purpose |
|----------|---------|
| testConcurrentStarts | 5 goroutines start the same workflow concurrently with different inputs, verifies all return non-empty runIDs and no errors |
| testMultiWorkflowStress | Deploys 4 workflow types (fast-seq, suspend, parallel, multi-suspend), runs 10 fast + 3 suspend + 2 parallel + 2 multi-suspend, resumes all suspended, verifies all complete correctly |
| testLongRunningIntegration | Deploys 5 workflow types, runs 5 waves: (1) 10 fast + 5 sleepers + 3 suspenders + 2 multi-suspend, (2) parallel + fast while sleepers sleep, (3) resume suspenders, (4) resume multi-suspend, (5) cancel, then waits for sleepers, verifies all outputs from storage |

### developer.go — Developer scenario tests

| Function | Purpose |
|----------|---------|
| testToolCallInsideStep | Deploys 2-step workflow where step 1 calls tools.call("echo") then step 2 transforms, verifies echoed query appears uppercased in final output |
| testBusEmitFromStep | Deploys workflow that calls bus.emit("order.processing") from a step, subscribes to the event, verifies orderId and stage in event payload |
| testConditionalBranch | Deploys workflow with classify step + branch (premium >= 100, standard < 100), tests both paths, verifies correct branch output from storage |
| testStepState | Deploys 3-step workflow using setState to accumulate counter + log, verifies final counterValue=11 and logLength=2 |
| testSuspendWithContextData | Deploys doc-review workflow with rich suspend/resume schemas, starts (suspends with context), verifies suspend payload in storage, resumes with reviewer info, verifies final output |

## Cross-references

- **Campaigns:** transport/{amqp,redis,postgres,nats,sqlite}_test.go, fullstack/{nats_postgres_rbac,amqp_postgres_vector,redis_mongodb}_test.go
- **Related domains:** tools (tools.call inside workflows), persistence (workflow state persistence), bus (bus.emit from steps)
- **Fixtures:** workflow TS fixtures
