# RBAC Test Map

**Purpose:** Verifies role-based access control enforcement at kernel level, bridge level (JS Compartment), and permission matrix correctness for all 4 standard roles.
**Tests:** 36 functions across 5 files
**Entry point:** `rbac_test.go` → `Run(t, env)`
**Campaigns:** fullstack/nats_postgres_rbac_test.go

## Files

### enforcement.go — Kernel-level RBAC enforcement via SDK bus messages

| Function | Purpose |
|----------|---------|
| testRestrictedCannotPublishForbidden | Deploys a service with restricted role, sends it a message that tries bus.publish to secrets.set, verifies the reply contains "permission denied" |
| testRestrictedCannotRegisterTools | Deploys with restricted role, handler tries kit.register("tool"), verifies "permission denied" in reply |
| testOwnMailboxAlwaysAllowed | Deploys with restricted role, handler uses bus.on for its own mailbox topic, verifies msg.reply works (own mailbox is always accessible) |
| testAdminCanDoEverything | Creates a kernel with admin default role, deploys code that emits events, verifies no errors |
| testAssignRevokeViaBus | Publishes RBACAssignMsg, RBACListMsg, RBACRevokeMsg, RBACListMsg sequence, verifies assign/list/revoke/empty-list lifecycle |
| testPermissionDeniedEventEmitted | Subscribes to bus.permission.denied, triggers a denied publish, verifies the event contains source/action/role |
| testWithRoleOnDeploy | Deploys with WithRole("observer"), handler tries bus.publish to events.*, verifies "permission denied" |
| testRolePersistenceAcrossRestart | Deploys with observer role, closes kernel, reopens, verifies deployment restored and functional |
| testSecretBridgeEnforcement | Deploys with restricted role, handler tries secrets.get, verifies denied response |
| testGatewayRouteEnforcement | Deploys with restricted role, handler tries publishing to gateway.http.route.add, verifies error |
| testCommandMatrix | Table-driven test checking AllowsCommand for 13 command/role combinations across service/observer/admin/gateway |
| testMultiDeploymentIsolation | Deploys admin and observer services on same kernel, admin can emit events, observer is denied |
| testRBACDeniedFromTS | Deploys observer .ts that tries bus.publish and bus.subscribe, verifies publish denied but subscribe allowed |
| testInputAbuseRBACEmptySource | Sends RBACAssignMsg with empty Source, verifies VALIDATION_ERROR response |
| testInputAbuseRBACNonexistentRole | Sends RBACAssignMsg with nonexistent role name, verifies error response |
| testRBACEnforcementOnTransport | Deploys observer .ts that tries bus.publish to forbidden topic, verifies DENIED via EvalTS output |
| testRBACToolCallOnTransport | Deploys service .ts that calls tools.call("echo"), verifies ALLOWED via EvalTS output |

### bridge.go — Bridge-level RBAC enforcement via deploy+output()+EvalTS

| Function | Purpose |
|----------|---------|
| testBridgeServiceCanPublishIncoming | Service role can publish to incoming.* topics |
| testBridgeServiceCannotPublishRandom | Service role cannot publish to random.forbidden topic |
| testBridgeServiceCanEmitEvents | Service role can emit to events.* topics |
| testBridgeServiceCannotEmitGateway | Service role cannot emit to gateway.* topics |
| testBridgeServiceCanRegisterTool | Service role can register tools via kit.register |
| testBridgeServiceCannotRegisterAgent | Service role cannot register agents |
| testBridgeGatewayCanPublishGateway | Gateway role can publish to gateway.* topics |
| testBridgeGatewayCannotPublishEvents | Gateway role cannot publish to events.* topics |
| testBridgeGatewayCanEmitGateway | Gateway role can emit to gateway.* topics |
| testBridgeObserverCannotPublish | Observer role cannot publish to any topic |
| testBridgeObserverCannotEmit | Observer role cannot emit to any topic |
| testBridgeObserverCanSubscribe | Observer role can subscribe to events.anything (subscribe:* allowed) |
| testBridgeAdminCanDoEverything | Admin role can publish, emit, subscribe, register tools and agents across 6 sub-tests |
| testBridgeOwnMailboxAlwaysAllowed | All 4 roles can use bus.on for their own mailbox |

### matrix.go — RBAC permission matrix unit tests

| Function | Purpose |
|----------|---------|
| testMatrixCommandPermissions | Tests AllowsCommand for 40+ commands across all 4 roles with explicit allowlists |
| testMatrixBusPublish | Tests Bus.Publish.Allows for 8 topic patterns across all 4 roles |
| testMatrixBusSubscribe | Tests Bus.Subscribe.Allows for 4 topic patterns across all 4 roles |
| testMatrixBusEmit | Tests Bus.Emit.Allows for 4 topic patterns across all 4 roles |
| testMatrixRegistration | Tests Registration.Tools and Registration.Agents booleans for all 4 roles |
| testMatrixOwnMailbox | Tests IsOwnMailbox with 7 source/topic cases including edge cases |
| testMatrixIntegrationObserverDeniedPublish | Integration: observer .ts deploy tries bus.publish, verifies DENIED |
| testMatrixIntegrationServiceAllowedToolCall | Integration: service .ts deploy calls tools.call, verifies ALLOWED |
| testMatrixIntegrationGatewayDeniedEverything | Integration: gateway .ts deploy tests 5 operations, verifies correct allow/deny per gateway rules |

## Cross-references

- **Campaigns:** fullstack/nats_postgres_rbac_test.go
- **Related domains:** security (RBAC escape tests), persistence (role persistence)
- **Fixtures:** RBAC-related TS fixtures
