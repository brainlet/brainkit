# Security Test Map

**Purpose:** Adversarial security probes: sandbox escape, data leakage, bus forgery, cross-deployment attacks, internal exploits, reply token security, timing attacks, secret exfiltration, gateway injection, state corruption, persistence attacks, and LibSQL validation.
**Tests:** 96 functions across 13 files
**Entry point:** `security_test.go` → `Run(t, env)`
**Campaigns:** none (runs only on memory transport)

## Files

### sandbox.go — SES Compartment sandbox escape attacks (10 tests)

| Function | Purpose |
|----------|---------|
| testSandboxDirectBridgeAccess | Deploys .ts that probes for 14 raw Go bridge functions (__go_brainkit_request etc.), verifies none are accessible inside Compartment |
| testSandboxHijackCompartment | Deploys attacker .ts that probes globalThis.__kit_compartments for victim's Compartment object, verifies not accessible |
| testSandboxRegistryManipulation | Deploys attacker .ts that probes __kit_registry to unregister/replace another deployment's tool, verifies blocked |
| testSandboxBusSubsHijack | Deploys .ts that probes __bus_subs to hijack another deployment's bus handlers, verifies not accessible |
| testSandboxPrototypePollution | Deploys .ts that sets Object/Array/Function prototype properties and hijacks JSON.parse, verifies a second deployment is unaffected |
| testSandboxEndowmentOverwrite | Deploys .ts that overwrites bus.publish/tools.call/secrets.get with interceptors, verifies a second deployment's endowments are intact |
| testSandboxGlobalThisAccess | Deploys .ts that tries 4 techniques to reach real globalThis (indirect eval, Function constructor, etc.), verifies no bridge leak |
| testSandboxFSPathTraversal | Deploys .ts with 8 path traversal patterns (../../../etc/passwd, encoded, backslash), verifies no reads escape the workspace |
| testSandboxFSWriteEscape | Deploys .ts that tries writing to ../../../tmp/ and /tmp/, verifies writes are normalized into workspace |
| testSandboxRuntimeModification | Deploys .ts that modifies __kitEndowments to inject backdoor, verifies a subsequent deployment does not see the backdoor |

### data_leakage.go — Data leakage probes (8 tests)

| Function | Purpose |
|----------|---------|
| testLeakageErrorMessageContent | Triggers tool-not-found, agent-not-found, and bad-deploy errors, scans response strings for sensitive patterns (file paths, goroutine, password) |
| testLeakageSharedGlobalState | Deploys A that writes to globalThis.leaked_secret, deploys B that reads it, verifies B cannot see A's data |
| testLeakageToolStateLeak | Registers a Go tool that stores last caller's input, calls it twice with different data, logs whether caller B can see caller A's data |
| testLeakageMetadataLeak | Deploys .ts handler that returns all msg.* metadata keys, logs which internal fields are visible to handlers |
| testLeakageSecretTimingSideChannel | Measures timing of secrets.get for existing vs nonexistent keys over 20 iterations each, logs timing differential |
| testLeakageDeploymentReconnaissance | Deploys .ts that calls kit.list and tools.list via bridge, logs what deployment/tool information is enumerable |
| testLeakageFilesystemReconnaissance | Creates sensitive files (.env, secrets/), deploys .ts that reads them via fs, logs whether .ts can read workspace files |
| testLeakageProviderReconnaissance | Deploys .ts that calls registry.list/registry.resolve, checks if API keys are exposed in resolved config |

### bus_forgery.go — Bus message forgery attacks (12 tests)

| Function | Purpose |
|----------|---------|
| testForgeryStealReplyTo | Subscribes to another caller's replyTo topic, verifies GoChannel fanout behavior (both attacker and legitimate subscriber receive) |
| testForgeryInjectFakeReply | Deploys slow service, publishes fake response to replyTo before real response arrives, logs which arrives first |
| testForgeryCorrelationIdCollision | Two callers publish to same handler using shared replyTo, logs how many responses are received on the shared channel |
| testForgeryRecursiveBusLoop | Deploys handler that re-emits to its own topic up to 100 times, verifies kernel survives without deadlock |
| testForgeryFloodBus | Publishes 10,000 raw messages to a deployed handler's topic, verifies kernel stays alive |
| testForgerySubscriptionBomb | Deploys .ts that creates 1000 subscriptions, tears it down, verifies kernel survives both creation and cleanup |
| testForgeryScheduleBomb | Deploys .ts that creates 500 schedules firing in 1ms, waits 3s, verifies kernel alive |
| testForgeryCommandTopicBypass | Deploys .ts that tries bus.emit to 12 command topics (tools.call, secrets.set, etc.), logs which are accepted vs blocked |
| testForgeryToolNameCollision | Two deployments register same tool name, calls the tool, logs which version answers |
| testForgeryMetadataInjection | Deploys .ts that tries to set replyTo/correlationId/callerId in bus.publish payload, verifies forged replyTo is not used |
| testForgeryCrossDeploymentResult | Deploys A that sets output, deploys B that tries overwriting globalThis.__module_result, verifies A's output is not visible to B |
| testForgeryMaliciousGoTool | Registers Go tool returning __proto__ and constructor keys in response, deploys .ts that calls it, verifies no prototype pollution |

### cross_deploy.go — Cross-deployment attack vectors (10 tests)

| Function | Purpose |
|----------|---------|
| testXDeployTeardownAnother | Deployment B tries to teardown deployment A via bridge kit.teardown command, verifies A's handler still responds |
| testXDeployReplyImpersonation | Deployment B subscribes to A's mailbox and tries msg.reply, logs whether impersonation reply or legitimate reply arrives first |
| testXDeployUnregisterAlienTool | Deployment B tries kit.unregister on A's tool, then calls the tool, verifies A's tool still works |
| testXDeployStealOutput | Deployment A outputs sensitive data, deployment B reads globalThis.__module_result, verifies B cannot see A's output |
| testXDeployMailboxEavesdrop | Go subscriber listens on A's mailbox topic, sends a message to A, logs how many messages the eavesdropper intercepts |
| testXDeployAgentRegistrationRace | Two deployments register same agent name, lists agents, verifies no duplicate registrations |
| testXDeployCreateToolMonkeyPatch | Deployment A monkey-patches createTool to intercept tool inputs, deployment B creates a tool, calls it, verifies no interception across Compartments |
| testXDeploySendToCrafted | Deployment sends bus.sendTo with crafted payloads (__proto__, constructor), verifies kernel survives |
| testXDeploySelfRedeploy | Deployment tries kit.redeploy on itself via bridge, verifies kernel stays alive |
| testXDeployWorkflowEscalation | Deployment tries kit.register("memory") and kit.register("workflow"), logs whether these invalid types are blocked |

### internal_exploit.go — Internal runtime exploit attempts (13 tests)

| Function | Purpose |
|----------|---------|
| testExploitReplyToRedirect | Handler modifies msg.replyTo before calling msg.reply, verifies whether response goes to attacker topic or legitimate replyTo |
| testExploitSendToNamespaceConfusion | Tests bus.sendTo with crafted service names (../../admin.ts, empty, deep paths), logs resulting topic resolution |
| testExploitScheduleFiresCommandTopic | Attempts to schedule message to "secrets.set" command topic, verifies error "command topic" is returned |
| testExploitAPIKeyJSInjection | Creates kernels with API keys containing JS injection payloads, verifies no code execution via escaped key values |
| testExploitDeployFileEscape | Sends kit.deploy.file messages with paths like /etc/hosts and ../../../etc/passwd, verifies error responses |
| testExploitHardenBypass | Deploys .ts that tries adding properties to bus/kit/tools/secrets objects and replacing secrets.get, logs mutability findings |
| testExploitDeployOrderingAttack | Seeds store with first-runner .ts that patches __kitEndowments, adds legit second .ts, verifies the patch does not cross Compartment boundary |
| testExploitReentrantSourceTracking | Deployment A calls B's tool, which accesses kit.source, verifies source tracking during reentrant tool calls |
| testExploitPluginStateKeyCollision | Documents that plugin names with dots/slashes/dashes sanitize to the same key (known limitation) |
| testExploitLibSQLCacheExhaustion | Deploys .ts that writes 100 files in a loop, verifies kernel stays alive |
| testExploitRegistryResolveLeak | Deploys .ts that calls registry.resolve("provider", "openai"), checks if API key appears in the result |
| testExploitProviderGlobalLeak | Deploys .ts that probes __kit_providers on globalThis, checks if provider configs with API keys are visible |


| Function | Purpose |
|----------|---------|

### reply_token.go — Reply token security (7 tests)

| Function | Purpose |
|----------|---------|
| testTokenOwnMailboxGetsToken | Deploys service .ts handler, sends message, verifies handler receives non-empty replyToken in msg |
| testTokenLegitHandlerCanReply | Deploys service handler that calls msg.reply, verifies the reply arrives at the legitimate replyTo |
| testTokenObserverCannotReply | Deploys service + observer, observer subscribes to service's mailbox and tries msg.reply, verifies only legitimate handler's reply arrives |
| testTokenStreamingWithToken | Deploys service that sends stream chunks (text, progress, end), verifies all chunks arrive with valid token |
| testTokenAuditEventEmitted | Observer tries to reply on service's mailbox, subscribes to bus.reply.denied, verifies audit event with "invalid reply token" |
| testTokenCrossDeploymentScoped | Service A and admin B both subscribe to A's mailbox, B tries to reply, verifies only A's reply arrives |

### timing.go — Timing and race condition attacks (10 tests)

| Function | Purpose |
|----------|---------|
| testTimingPreemptiveReplySubscribe | Subscribes to a broad reply pattern before sending a message, logs whether the attacker receives messages via fanout |
| testTimingDeployTeardownRace | Runs 50 concurrent deploy+teardown pairs on the same source, verifies kernel survives and no deadlock |
| testTimingMessageDuringRestore | Persists a deployment, reopens kernel, immediately sends 10 messages from goroutines, verifies kernel survives and some respond |
| testTimingConcurrentRedeploy | Runs 20 concurrent Redeploy calls on same source, verifies exactly 1 deployment survives |
| testTimingToolCallDuringDeploy | Deploys .ts that registers a tool then immediately calls it (reentrant), logs the result |
| testTimingScheduleFiresBeforeHandlerReady | Schedules a message 200ms out, deploys the handler 100ms later, verifies kernel stays alive regardless of delivery timing |
| testTimingCloseWhileToolCallInProgress | Deploys slow (2s) tool, starts calling it, calls kernel.Close after 100ms, verifies Close succeeds cleanly |
| testTimingRoleChangeWhileHandlerRunning | Admin handler does bus.publish, mid-execution reassign role to observer, logs whether the second publish is denied |
| testTimingScheduleUnscheduleRace | 50 goroutines each schedule+immediately unschedule, verifies kernel alive and logs how many schedules still fired |
| testTimingStorageRaceWithDeploy | 3 goroutines: AddStorage x20, RemoveStorage x20, Deploy/Teardown x10 concurrently, verifies kernel survives |

### secrets.go — Secret exfiltration attacks (7 tests)

| Function | Purpose |
|----------|---------|
| testSecretPublishToBus | Service reads secret then publishes it to incoming.* and events.* topics, registers a leak-tool, subscribes to exfiltration topics, logs detection count |
| testSecretObserverReadsSecret | Observer role tries secrets.get, verifies the secret value is not returned |
| testSecretEnvVarDump | Deploys .ts that probes process.env for sensitive keys (OPENAI_API_KEY etc.), logs which are accessible |
| testSecretEnumeration | Deploys .ts that calls secrets.list via bridge, logs whether secret names are enumerable |
| testSecretAuditEventSnooping | Subscribes to secrets.accessed event as an eavesdropper, triggers a legitimate get, logs intercepted audit events |
| testSecretRotateDOS | Another caller rotates a secret that a deployed handler cached at startup, verifies the handler detects the changed value |
| testSecretDecryptionOracle | Sets encrypted secret with correct key, reopens with 4 wrong keys, verifies none can decrypt to the original value |

### gateway.go — Gateway HTTP security (4 tests)

| Function | Purpose |
|----------|---------|
| testGatewayHeaderInjection | Sends HTTP request with forged X-Caller-ID/X-Reply-To headers, verifies they do not set callerID in the handler's msg |
| testGatewayProtoPollutionViaHTTP | Sends JSON body with __proto__ and constructor keys, verifies no prototype pollution in the handler |
| testGatewayPathTraversalParams | Sends 5 path traversal URLs (../, encoded, null byte, XSS, SQL injection), verifies kernel survives all |
| testGatewayWebSocketInjection | Sends HTTP GET to WebSocket endpoint (without upgrade), verifies kernel survives |

### state.go — State corruption security (2 tests)

| Function | Purpose |
|----------|---------|
| testStateNonexistentRoleOnDeploy | Seeds store with deployment having nonexistent role, reopens kernel, verifies it starts without crash |
| testStateStoreWipedMidlife | Deploys a service, deletes it from store behind kernel's back, verifies in-memory deployment survives |

### persistence.go — Persistence attack vectors (4 tests)

| Function | Purpose |
|----------|---------|
| testPersistSQLInjectionInSource | Deploys with SQL injection source names ('; DROP TABLE, OR 1=1, null byte), closes, reopens, verifies kernel recovers |
| testPersistCodeMutatesStoreDuringRestore | Seeds store with .ts that tries kit.teardown on another deployment during restore, verifies both deployments survive |
| testPersistEvilPluginPaths | Seeds running_plugins table with evil binary paths (curl to evil.com, ../../../bin/sh), verifies kernel starts without executing them |
| testPersistConcurrentStoreWrites | Two kernels sharing same SQLite store file, concurrent deploy/teardown from both, verifies both stay alive |

### libsql_validation.go — LibSQL file: URL blocking (2 tests)

| Function | Purpose |
|----------|---------|
| testLibSQLFileURLBlocked | Deploys .ts that creates LibSQLStore with file: URL, verifies VALIDATION_ERROR with "file:" in message |
| testLibSQLHttpURLNotBlocked | Deploys .ts that creates LibSQLStore with http: URL, verifies it does not trigger the file: URL validation |

## Cross-references

- **Campaigns:** none (memory-only domain)
- **Fixtures:** security-related TS fixtures
