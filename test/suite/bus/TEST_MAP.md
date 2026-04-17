# Bus Test Map

**Purpose:** Verifies the entire bus subsystem: pub/sub, correlation, reply, streaming, failure/retry, rate limiting, error contracts, surface matrix, transport compliance, and cross-feature interactions
**Tests:** 124 functions across 20 files
**Entry point:** `bus_test.go` → `Run(t, env)`
**Campaigns:** transport (all 5), fullstack (all 3)

## Files

### publish.go — JS bridge pub/sub and deploy+bus.on flow

| Function | Purpose |
|----------|---------|
| testJSPublishReturnsReplyTo | Calls __go_brainkit_bus_publish from JS and asserts the returned object has non-empty replyTo and correlationId |
| testJSEmitFireAndForget | Calls __go_brainkit_bus_emit from JS, subscribes from Go, asserts the message arrives without a replyTo |
| testJSReplyDoneFlag | Sets up a JS subscriber that sends a chunk then a final reply, verifies Go receives both with correct correlationId and done metadata |
| testJSSubscribeReceivesMetadata | Sets up a JS subscriber, publishes from Go, verifies the JS handler receives payload, replyTo, correlationId, and topic |
| testGoToJSRoundTrip | Sets up a JS handler for a topic, publishes a CustomMsg from Go, asserts the JS handler replies with the expected payload |
| testDeployWithBusOn | Deploys .ts with bus.on("greet"), sends via SendToService, verifies the reply contains "hello world" |
| testStreamingChunks | Deploys .ts with msg.send (chunk) + msg.reply (final), verifies chunks arrive and last has done=true |
| testKitRegisterAgentDiscovery | Deploys .ts calling kit.register("agent"), verifies agent appears in AgentListMsg, tears down, verifies agent is removed |

### async.go — Correlation, concurrency, cancellation

| Function | Purpose |
|----------|---------|
| testCorrelationIDFiltering | Publishes ToolListMsg, subscribes to the replyTo, asserts a correlated response arrives |
| testMultipleInFlight | Fires 10 concurrent ToolListMsg publishes, each subscribing to its own replyTo, asserts all 10 get responses |
| testContextCancellation | Publishes with an already-cancelled context, verifies no panic |
| testSubscribeCancellation | Subscribes then immediately unsubs, verifies no messages are received afterward |

### sdk_reply.go — sdk.Reply, sdk.SendChunk, sdk.SendToService

| Function | Purpose |
|----------|---------|
| testSDKReply | Deploys .ts echo service, calls via SendToService, subscribes to replyTo, verifies pong response from TS |
| testSDKReplyGoToGo | Registers a Go subscriber that calls sdk.Reply, publishes from Go, verifies the reply arrives on the replyTo topic |
| testSDKSendChunk | Registers a Go subscriber that sends 3 chunks + final via sdk.SendChunk/sdk.Reply, verifies at least 4 messages received |
| testSDKSendToService | Deploys .ts calc service, calls via SendToService with a=17 b=25, verifies result=42 |

### failure.go — Handler errors, retry, dead letter, exhausted events

| Function | Purpose |
|----------|---------|
| testSyncThrowErrorResponse | Deploys .ts handler that throws synchronously, sends a message, verifies caller gets error containing "sync boom" |
| testAsyncRejectionErrorResponse | Deploys .ts handler that throws in async, sends a message, verifies caller gets error containing "async boom" |
| testHandlerFailedEventEmitted | Subscribes to bus.handler.failed, deploys throwing handler, sends message, asserts the failure event is emitted with the error string |
| testRetryPolicyRetries | Creates kernel with retry policy (max 2), deploys handler that fails first 2 attempts then succeeds, verifies caller gets success response |
| testRetryExhaustedDeadLetter | Creates kernel with retry policy + dead letter topic, deploys always-failing handler, verifies dead letter message and error response arrive |
| testExhaustedEventEmitted | Creates kernel with retry policy, subscribes to bus.handler.exhausted, deploys always-failing handler, verifies exhaustion event with retryCount |
| testRetryPreservesReplyTo | Creates kernel with retry policy, deploys handler that fails once then succeeds, verifies original caller's replyTo receives the success at attempt 2 |

### ratelimit.go — Bus rate limiting

| Function | Purpose |
|----------|---------|

### pump.go — Pump scheduling latency

| Function | Purpose |
|----------|---------|
| testPumpScheduleLatency | Deploys .ts ping handler, measures 10 round-trip latencies, asserts median is under 5ms |
| testPumpResponsiveAfterIdle | Sleeps 500ms then calls EvalTS, asserts the kernel responds (pump is still alive after idle) |

### log.go — Console log handler compartmentalization

| Function | Purpose |
|----------|---------|
| testLogHandlerTSCompartment | Creates kernel with LogHandler, deploys .ts calling console.log/warn/error, asserts log entries are tagged with the source file and correct levels |
| testLogHandlerMultipleFiles | Deploys two .ts files with different console.log messages, asserts logs are correctly attributed to each source |
| testLogHandlerConcurrent | Deploys and tears down 5 times concurrently with LogHandler, asserts no panic and log entries are received |
| testLogHandlerNilDefault | Creates kernel without LogHandler, deploys .ts calling console.log, asserts no panic |

### error_contract.go — Bus error response format

| Function | Purpose |
|----------|---------|
| testBusErrorResponseCarriesCode | Calls a nonexistent tool via bus, asserts response JSON has error, code="NOT_FOUND", and details.name |
| testResultMetaIncludesCode | Marshals a ResultMeta struct with error+code+details, asserts the JSON roundtrip preserves all fields |

### test_framework.go — JS built-in test framework

| Function | Purpose |
|----------|---------|
| testFrameworkPassingTests | Runs a JS test file with 3 passing tests (math, string, truthiness), asserts all pass |
| testFrameworkFailingTest | Runs a JS test file with 1 passing and 1 failing test, asserts correct pass/fail states and error message |
| testFrameworkAsyncTests | Runs a JS test with async/await + sleep, asserts it passes |
| testFrameworkDeployAndTest | Runs a JS test that deploys a .ts service and calls it via bus, asserts the test passes |
| testFrameworkHooks | Runs a JS test with beforeAll/afterAll hooks setting a counter, asserts hook-dependent test passes |
| testFrameworkNotAssertions | Runs a JS test using expect().not.toBe() and .not.toContain(), asserts it passes |

### errors.go — Bus error paths (command topic blocking, metadata, subscribe/unsubscribe)

| Function | Purpose |
|----------|---------|
| testPublishToCommandTopic | Calls __go_brainkit_bus_send on a command topic ("tools.call") from JS, asserts an error is thrown |
| testEmitToCommandTopic | Calls bus.emit on a command topic from JS, asserts it throws an error |
| testSubscribeReceivesMetadataAdv | Deploys handler that replies with topic/replyTo/correlationId presence flags, verifies all are true |
| testReplyWithoutReplyTo | Deploys handler that calls msg.reply, sends via emit (no replyTo), asserts kernel stays alive |
| testSendToNonexistentService | Calls bus.sendTo for a nonexistent service from JS, asserts it returns a replyTo (fire and forget) |
| testCorrelationIDPreserved | Deploys handler that echoes correlationId, verifies it matches the original publish's correlationId |
| testMultipleReplies | Deploys handler sending 2 chunks + final, asserts at least 2 messages received including done=true |
| testSubscribeUnsubscribe | Subscribes from JS, emits, unsubscribes, emits again, asserts no panic |
| testDeploymentNamespace | Deploys .ts that outputs kit.source/namespace/callerId, verifies output contains the source name |
| testScheduleWithPayload | Creates a schedule with "in 200ms" and a JSON payload, subscribes to the topic, asserts payload arrives |

### integration.go — Multi-service chain

| Function | Purpose |
|----------|---------|
| testTwoServiceInteraction | Deploys service B (bus.on process), then service A (bus.on ask, forwards to B via sendTo), sends to A, verifies response |

### async_diag.go — Async operation levels inside bus.on handlers

| Function | Purpose |
|----------|---------|
| testDiagBusOnAwaitPromiseResolve | Deploys handler that awaits Promise.resolve, verifies resolved value in reply |
| testDiagBusOnAwaitSetTimeout | Deploys handler that awaits setTimeout(50ms), verifies "delayed" in reply |
| testDiagBusOnAwaitToolsCall | Deploys handler that awaits tools.call("echo"), verifies tool result in reply |
| testDiagBusOnAwaitFetch | Deploys handler that awaits fetch("https://httpbin.org/get"), verifies status 200 |
| testDiagBusOnAwaitGenerateText | Deploys handler that awaits generateText (requires OPENAI_API_KEY), verifies non-empty text |

### surface.go — Bus command matrix (valid/empty/garbage payloads)

| Function | Purpose |
|----------|---------|
| testBusMatrixValidInput | Sends valid input to every bus command topic (28 commands), asserts none hang or panic |
| testBusMatrixEmptyInput | Sends empty/missing-field input to commands with expected error codes, asserts correct error codes |
| testBusMatrixGarbagePayload | Sends 6 types of garbage JSON to 25 command topics, asserts kernel stays alive after each |

### cross_feature.go — Cross-feature adversarial interactions

| Function | Purpose |
|----------|---------|
| testCrossDeployCallsGoTool | Deploys .ts that calls Go "echo" tool during init, verifies result via globalThis |
| testCrossTSToolCallsAnotherTSTool | Service A registers tool "doubler", service B calls it, verifies doubled result |
| testCrossHandlerCallsTool | Bus handler calls tools.call("echo") during message processing, verifies response |
| testCrossHandlerReadsSecret | Sets a secret, deploys handler that reads it, verifies secret prefix in reply |
| testCrossHandlerWritesFS | Deploys handler that writes a file and reads it back, verifies content in reply |
| testCrossGoToolEmitsBusEvent | Registers a Go tool that emits a bus event as side effect, calls it, verifies event was emitted |
| testCrossTracedToolCall | Creates kernel with TraceStore, calls a tool, verifies trace spans are recorded |
| testCrossHealthDuringDeployChurn | Rapidly deploys/tears down 10 times, asserts kernel stays healthy throughout |
| testCrossMetricsTrackSchedules | Creates schedules, verifies Metrics().ActiveSchedules increments/decrements correctly |
| testCrossDeployWithPersistenceAndRestart | Deploys handler with SQLite store, closes kernel, restarts, calls handler, verifies it still works |

### error_contract_adv.go — Bus error contract adversarial tests

| Function | Purpose |
|----------|---------|
| testErrorContractBusNotFound | Calls nonexistent tool, asserts code="NOT_FOUND" with details.name |
| testErrorContractBusValidationError | Publishes SecretsSetMsg with empty name, asserts code="VALIDATION_ERROR" |
| testErrorContractBusAlreadyExists | Deploys then deploys again same source, asserts code="ALREADY_EXISTS" |
| testErrorContractBusDeployErrorBadSyntax | Deploys invalid TS syntax, asserts code="DEPLOY_ERROR" with details.source |
| testErrorContractErrorsAsAllTypes | Wraps every BrainkitError type twice, asserts errors.As finds it and Code() matches |
| testErrorContractJSBridgeValidationErrorMissingArgs | Calls __go_brainkit_bus_schedule with missing args from JS, asserts VALIDATION_ERROR code |
| testErrorContractJSBridgeRateLimited | Creates kernel with 1 req/s rate limit, deploys .ts that publishes 10 times, asserts RATE_LIMITED code appears |
| testErrorContractJSBridgeNotConfiguredSecrets | Calls secrets.get for nonexistent key on bare kernel, asserts empty string returned (not panic) |
| testErrorContractErrorHandlerPersistenceError | Closes the store, schedules an operation, asserts ErrorHandler receives PersistenceError |
| testErrorContractErrorHandlerDeployError | Persists corrupt code, restarts kernel, asserts ErrorHandler receives DeployError for the corrupt source |

### input_abuse.go — Bus input abuse (empty topics, large payloads, deep nesting)

| Function | Purpose |
|----------|---------|
| testInputAbuseBusEmptyTopic | Calls bus.publish("") from JS, asserts an error is thrown (not a panic) |
| testInputAbuseBusLargePayload | Publishes a 100KB payload from JS, asserts it succeeds and returns a replyTo |
| testInputAbuseBusDeeplyNestedJSON | Publishes a 50-level nested JSON object from JS, asserts it succeeds |
| testInputAbuseBusSubscribeEmptyTopic | Calls bus.subscribe("") from JS, asserts no panic |

### e2e.go — Multi-service chain E2E scenarios

| Function | Purpose |
|----------|---------|
| testE2EMultiServiceChain | Deploys service B (calls echo tool), service A (forwards to B), calls A, verifies chain response |
| testE2EStreamingResponse | Deploys handler using msg.stream.text/progress/end, verifies SSE-style stream events arrive |
| testE2EMultiDomain | Writes a file via FS polyfill, reads it, processes with echo tool, writes output, reads and verifies |

### failure_cascade.go — Failure cascade and edge cases

| Function | Purpose |
|----------|---------|
| testCascadeDeployWithBrokenStore | Closes SQLite store, deploys in memory, asserts ErrorHandler called for persistence failure |
| testCascadeCorruptedStore | Writes garbage to store file, asserts NewSQLiteStore returns error |
| testCascadePublishDuringDrain | Sets kernel draining, publishes from JS, asserts publish still works |
| testCascadeEvalTSDuringClose | Closes kernel, calls EvalTS, asserts error (not panic) |
| testCascadeSecretRotatePluginFails | Sets a secret, rotates with restart=true when no plugins running, asserts rotation succeeds |
| testCascadeRetryExhausted | Creates kernel with retry policy, deploys always-failing handler, verifies exhaustion event fires |
| testCascadeHandlerThrowNoReplyTo | Deploys handler that throws, sends via emit (no replyTo), asserts bus.handler.failed event or no panic |
| testCascadeTeardownCleansSubscriptions | Deploys handler, tears down, publishes to same topic, asserts no response (subscription cleaned) |
| testCascadeScheduleNoHandler | Schedules to a topic nobody listens on, waits for fire, asserts kernel stays alive |
| testCascadeConcurrentErrorHandler | Closes store, fires 10 concurrent schedule operations, asserts ErrorHandler called without panic |

### backend_advanced.go — Backend stress and transport compliance

| Function | Purpose |
|----------|---------|
| testConcurrentPublish50 | Fires 50 concurrent PublishRaw operations, asserts at least 1 message received |
| testLargePayload100KB | Deploys handler, sends 100KB message, asserts handler replies with the payload size |
| testDottedTopicNames | Deploys handler with dotted source name, publishes to dotted topic, verifies reply |
| testDeployHandlerCall | Deploys handler that calls tools.call, publishes, verifies tool result in reply |
| testPublishReply | Deploys handler that replies, publishes, verifies reply content |
| testErrorCodeOnBus | Calls nonexistent tool, asserts error code "NOT_FOUND" survives transport |
| testTransportCompliancePublishSubscribe | Subscribes to a topic, publishes raw JSON, asserts exact JSON match on receive |
| testTransportComplianceCorrelationID | Publishes raw, asserts correlationId is present in received message metadata |
| testTransportComplianceDottedTopics | Subscribes/publishes on a dotted topic, asserts message arrives |

### surface_matrix.go — Surface consistency (Go SDK, TS deployed, EvalTS, error consistency)

| Function | Purpose |
|----------|---------|
| testSurfaceGoSDK | Exercises tools.list, tools.call, secrets set+get, fs write+read, bus publish+reply, schedule, metrics, registry from Go SDK |
| testSurfaceTSDeployed | Deploys .ts code for each operation (tools, secrets, fs, bus, registry, schedule, metrics), verifies output |
| testSurfaceEvalTS | Runs each operation via EvalTS in global scope, verifies correct returns |
| testSurfaceErrorConsistency | Verifies NOT_FOUND and VALIDATION_ERROR produce identical error codes from Go, TS deployed, and EvalTS surfaces |

### transport_matrix.go — Transport matrix operations (ported from transport compliance tests)

| Function | Purpose |
|----------|---------|
| testTransportMatrixToolsCall | Calls tools.call with add(10,32) via sdk.Runtime, asserts sum=42 |
| testTransportMatrixToolsList | Calls tools.list, asserts non-empty tools array |
| testTransportMatrixToolsResolve | Calls tools.resolve for "echo", asserts ShortName="echo" |
| testTransportMatrixFSWriteRead | Writes and reads a file via EvalTS, asserts content match |
| testTransportMatrixFSMkdirListStatDelete | Creates dir, writes file, lists, stats, deletes via EvalTS, asserts correct counts and types |
| testTransportMatrixAgentsListEmpty | Calls agents.list, asserts non-nil Agents slice |
| testTransportMatrixKitDeployTeardown | Deploys .ts with tool, calls the tool, tears down, verifies full lifecycle |
| testTransportMatrixKitRedeploy | Deploys then redeploys same source, verifies redeploy succeeds |
| testTransportMatrixRegistryHasList | Checks registry.has for nonexistent provider (found=false), then lists, asserts non-nil Items |
| testTransportMatrixAsyncCorrelation | Publishes ToolListMsg, asserts non-empty correlation result |

## Cross-references

### audit.go — Centralized audit log bus commands

| Function | Purpose |
|----------|---------|
| testAuditQueryAfterDeploy | Deploys .ts, queries audit.query with category=deploy, verifies deploy event appears |
| testAuditStatsResponse | Queries audit.stats, asserts response has EventsByCategory map |
| testAuditPruneWorks | Calls audit.prune with 1h threshold, asserts no error |
| testAuditToolCallRecorded | Calls echo tool, queries audit for tools category, verifies tool call event |
| testAuditMetricsGetIncludesBus | Generates traffic, queries metrics.get, verifies Bus per-topic breakdown present |

## Cross-references

- **Campaigns:** `transport/{embedded,nats,redis,amqp}_test.go`, `fullstack/{redis_mongodb,amqp_postgres_vector}_test.go`
- **Related domains:** deploy (deploy lifecycle), agents (agent discovery), fs (cross-feature FS), health (cross-feature health), secrets (cross-feature secrets), tools (tool call), registry (registry operations)
- **Fixtures:** echo tool, add tool (registered by test helpers)
