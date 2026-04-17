# Deploy Test Map

**Purpose:** Verifies the .ts deployment lifecycle: deploy, teardown, redeploy, error handling, edge cases, input abuse, state corruption recovery, and TS surface capabilities
**Tests:** 51 functions across 6 files
**Entry point:** `deploy_test.go` → `Run(t, env)`
**Campaigns:** transport (all 5), fullstack (all 3)

## Files

### lifecycle.go — Core deploy/teardown/redeploy lifecycle

| Function | Purpose |
|----------|---------|
| testListEmpty | Publishes KitListMsg on fresh env, asserts Deployments is non-nil (may be empty) |
| testDeployTeardown | Deploys .ts with tool, verifies it appears in list, tears down, verifies removal |
| testRedeploy | Deploys v1, redeploys v2 via KitRedeployMsg, asserts Deployed=true |
| testDeployInvalidCode | Deploys .ts that throws during init, asserts error response with non-empty error |
| testDeployDuplicate | Deploys same source twice without teardown, asserts second returns error |
| testConcurrentDeploySameSource | Deploys same source via kernel.Deploy, deploys again, asserts "already exists" error |

### edge_cases.go — Deploy adversarial and edge case tests

| Function | Purpose |
|----------|---------|
| testTSImportsStripped | Deploys .ts with ES imports from "kit"/"ai"/"agent", verifies imports are stripped and globals work |
| testMultipleDeploymentsCoexist | Deploys 10 services, asserts ListDeployments returns 10, tears all down, asserts 0 |
| testRedeployPreservesOtherDeployments | Deploys 2 services, redeploys one, verifies the other is still listed |
| testLongSourceName | Deploys with 200-char source name, asserts no panic |
| testUnicodeSourceName | Deploys with Unicode source name, verifies it appears in list |
| testJSNotTS | Deploys .js file (not .ts), verifies it executes correctly |
| testEmptyCode | Deploys empty string code, asserts no panic |
| testCodeWithOnlyComments | Deploys code containing only comments, asserts no panic |
| testAsyncInit | Deploys code using top-level await (Promise.resolve), verifies async result |
| testToolWithComplexSchema | Deploys tool with z.object schema (nested, arrays, optionals), calls it with complex input, verifies processed response |
| testMultipleToolsOneDeployment | Deploys single .ts that registers 5 tools in a loop, resolves each one |
| testAgentRegistration | Deploys .ts calling kit.register("agent"), verifies agent in AgentListMsg |
| testWorkflowRegistration | Deploys .ts calling kit.register("workflow"), verifies in ListResources |
| testMemoryRegistration | Deploys .ts calling kit.register("memory"), verifies in ListResources |

### input_abuse.go — Deploy input abuse tests

| Function | Purpose |
|----------|---------|
| testDeployEmptySource | Deploys with empty and whitespace-only source names, asserts error for both |
| testDeployEmptyCode | Deploys with empty code via kernel.Deploy, asserts no panic |
| testDeployHugeCode | Deploys 1MB of mostly comments, asserts no hang |
| testDeploySourcePathTraversal | Deploys with ../escape, null bytes, quotes, backticks, spaces in source name, asserts no panic |
| testDeployThenImmediateTeardown | Deploys then immediately tears down, verifies deployment is gone from list |
| testDeployDuplicateSource | Deploys same source twice, asserts "already exists" error on second |
| testDeployInvalidTSSyntax | Deploys invalid TS syntax, asserts transpile or eval error |
| testDeployNullBytesInSourceName | Deploys with null byte in source name, asserts no panic |
| testDeployThrowsDuringInit | Deploys code that throws during init, verifies deployment is cleaned up from list |
| testDeployPartialCleanup | Deploys code that registers a tool then throws, verifies the tool is cleaned up |
| testDeployRedeployDifferentTools | Deploys with tool-a, redeploys with tool-b only, verifies tool-a is gone |
| testDeployDottedSourceName | Deploys with dots in source name, verifies bus topic resolution works correctly |

### state_corruption.go — Deploy state corruption recovery

| Function | Purpose |
|----------|---------|
| testStateCorruptionBadTranspile | Persists bad+good .ts in store, restarts kernel, verifies good deployment survives and ErrorHandler reports bad one |
| testStateCorruptionDuplicatePersistedSource | Persists duplicate source entries, restarts, verifies only 1 deployment exists (dedup) |
| testStateCorruptionStoreWipedMidlife | Deploys, wipes store behind kernel's back, verifies in-memory deployment still active |
| testStateCorruptionEmptyCode | Persists deployment with empty code, restarts, verifies kernel starts without panic |
| testStateCorruptionZeroDurationSchedule | Persists schedule with 0s duration, restarts, verifies kernel stays alive |
| testStateCorruptionPastScheduleFires | Persists schedule with NextFire in the past, restarts, subscribes to topic, expects it fires soon |

### e2e.go — Deploy lifecycle E2E scenarios

| Function | Purpose |
|----------|---------|
| testDeployLifecycle | Full cycle: deploy v1 with tool, call it, teardown, verify tool gone (NOT_FOUND), redeploy v2, call v2 |
| testE2EDeployWithErrorRecovery | Deploys bad code (throws), then deploys good code to same source, verifies tool works |
| testE2EDeployListRedeployTeardown | Deploys, lists (asserts present), redeploys, tears down, lists (asserts absent) |

### surface.go — TS surface deploy capabilities

| Function | Purpose |
|----------|---------|
| testTSNamespaceIsolation | Deploys 2 services with same handler name, sends to service A, verifies reply comes from A not B |
| testTSModuleImports | Deploys .ts that checks typeof for all endowments (bus, kit, model, tools, fs, mcp, output, registry), asserts all present |
| testTSAgentEndowments | Deploys .ts checking Agent, createTool, createWorkflow, createStep, z availability, asserts all present |
| testTSAISDKEndowments | Deploys .ts checking model, generateText, streamText, generateObject availability, asserts present |
| testTSDeployWithTool | Deploys .ts creating a calc tool, calls from Go with a=10 b=32, asserts sum=42 and source="ts-surface" |
| testTSDeployWithWorkflow | Deploys .ts creating a 2-step Mastra workflow (uppercase then exclaim), runs it, verifies "DEPLOY TEST!!!" |
| testTSDeployWithBusService | Deploys .ts with bus.on("greet"), sends from Go, verifies greeting reply |
| testTSDeployWithStreaming | Deploys .ts sending 3 chunks + final via msg.send/msg.reply, verifies done=true and count=3 |
| testTSFileExtensionHandling | Deploys .ts (transpiled) and .js (direct), verifies both execute correctly |

## Cross-references

- **Campaigns:** `transport/{sqlite,nats,postgres,redis,amqp}_test.go`, `fullstack/{redis_mongodb,amqp_postgres_vector}_test.go`
- **Related domains:** bus (deploy+bus.on flow), agents (agent registration), tools (tool registration), packages (multi-file deploy)
- **Fixtures:** none
