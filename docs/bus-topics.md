# Bus topic catalog

Generated from `sdk/*_messages.go` via `go run scripts/gen-bus-topics.go`. Do not edit by hand.

| Topic | Request | Response | Source |
|-------|---------|----------|--------|
| `agents.discover` | `AgentDiscoverMsg` | `AgentDiscoverResp` | `agent_messages.go` |
| `agents.get-status` | `AgentGetStatusMsg` | `AgentGetStatusResp` | `agent_messages.go` |
| `agents.list` | `AgentListMsg` | `AgentListResp` | `agent_messages.go` |
| `agents.set-status` | `AgentSetStatusMsg` | `AgentSetStatusResp` | `agent_messages.go` |
| `audit.prune` | `AuditPruneMsg` | `AuditPruneResp` | `audit_messages.go` |
| `audit.query` | `AuditQueryMsg` | `AuditQueryResp` | `audit_messages.go` |
| `audit.stats` | `AuditStatsMsg` | `AuditStatsResp` | `audit_messages.go` |
| `bus.handler.exhausted` | `HandlerExhaustedEvent` | `(no reply)` | `event_messages.go` |
| `bus.handler.failed` | `HandlerFailedEvent` | `(no reply)` | `event_messages.go` |
| `cluster.peers` | `ClusterPeersMsg` | `ClusterPeersResp` | `kit_messages.go` |
| `gateway.http.route.add` | `GatewayRouteAddMsg` | `GatewayRouteAddResp` | `gateway_messages.go` |
| `gateway.http.route.list` | `GatewayRouteListMsg` | `GatewayRouteListResp` | `gateway_messages.go` |
| `gateway.http.route.remove` | `GatewayRouteRemoveMsg` | `GatewayRouteRemoveResp` | `gateway_messages.go` |
| `gateway.http.status` | `GatewayStatusMsg` | `GatewayStatusResp` | `gateway_messages.go` |
| `kit.deployed` | `KitDeployedEvent` | `(no reply)` | `event_messages.go` |
| `kit.eval` | `KitEvalMsg` | `KitEvalResp` | `kit_messages.go` |
| `kit.health` | `KitHealthMsg` | `KitHealthResp` | `kit_messages.go` |
| `kit.send` | `KitSendMsg` | `KitSendResp` | `kit_messages.go` |
| `kit.set-draining` | `KitSetDrainingMsg` | `KitSetDrainingResp` | `kit_messages.go` |
| `kit.teardown.done` | `KitTeardownedEvent` | `(no reply)` | `event_messages.go` |
| `mcp.callTool` | `McpCallToolMsg` | `McpCallToolResp` | `mcp_messages.go` |
| `mcp.listTools` | `McpListToolsMsg` | `McpListToolsResp` | `mcp_messages.go` |
| `metrics.get` | `MetricsGetMsg` | `MetricsGetResp` | `kit_messages.go` |
| `package.deploy` | `PackageDeployMsg` | `PackageDeployResp` | `package_deploy_messages.go` |
| `package.info` | `PackageDeployInfoMsg` | `PackageDeployInfoResp` | `package_deploy_messages.go` |
| `package.list` | `PackageListDeployedMsg` | `PackageListDeployedResp` | `package_deploy_messages.go` |
| `package.teardown` | `PackageTeardownMsg` | `PackageTeardownResp` | `package_deploy_messages.go` |
| `peers.list` | `PeersListMsg` | `PeersListResp` | `kit_messages.go` |
| `peers.resolve` | `PeersResolveMsg` | `PeersResolveResp` | `kit_messages.go` |
| `plugin.list` | `PluginListRunningMsg` | `PluginListRunningResp` | `package_messages.go` |
| `plugin.manifest` | `PluginManifestMsg` | `PluginManifestResp` | `plugin_messages.go` |
| `plugin.registered` | `PluginRegisteredEvent` | `(no reply)` | `event_messages.go` |
| `plugin.restart` | `PluginRestartMsg` | `PluginRestartResp` | `package_messages.go` |
| `plugin.start` | `PluginStartMsg` | `PluginStartResp` | `package_messages.go` |
| `plugin.started` | `PluginStartedEvent` | `(no reply)` | `event_messages.go` |
| `plugin.status` | `PluginStatusMsg` | `PluginStatusResp` | `package_messages.go` |
| `plugin.stop` | `PluginStopMsg` | `PluginStopResp` | `package_messages.go` |
| `plugin.stopped` | `PluginStoppedEvent` | `(no reply)` | `event_messages.go` |
| `providers.add` | `ProviderAddMsg` | `ProviderAddResp` | `provider_messages.go` |
| `providers.remove` | `ProviderRemoveMsg` | `ProviderRemoveResp` | `provider_messages.go` |
| `registry.has` | `RegistryHasMsg` | `RegistryHasResp` | `registry_messages.go` |
| `registry.list` | `RegistryListMsg` | `RegistryListResp` | `registry_messages.go` |
| `registry.resolve` | `RegistryResolveMsg` | `RegistryResolveResp` | `registry_messages.go` |
| `schedules.cancel` | `ScheduleCancelMsg` | `ScheduleCancelResp` | `schedule_messages.go` |
| `schedules.create` | `ScheduleCreateMsg` | `ScheduleCreateResp` | `schedule_messages.go` |
| `schedules.list` | `ScheduleListMsg` | `ScheduleListResp` | `schedule_messages.go` |
| `secrets.accessed` | `SecretsAccessedEvent` | `(no reply)` | `secret_messages.go` |
| `secrets.delete` | `SecretsDeleteMsg` | `SecretsDeleteResp` | `secret_messages.go` |
| `secrets.deleted` | `SecretsDeletedEvent` | `(no reply)` | `secret_messages.go` |
| `secrets.get` | `SecretsGetMsg` | `SecretsGetResp` | `secret_messages.go` |
| `secrets.list` | `SecretsListMsg` | `SecretsListResp` | `secret_messages.go` |
| `secrets.rotate` | `SecretsRotateMsg` | `SecretsRotateResp` | `secret_messages.go` |
| `secrets.rotated` | `SecretsRotatedEvent` | `(no reply)` | `secret_messages.go` |
| `secrets.set` | `SecretsSetMsg` | `SecretsSetResp` | `secret_messages.go` |
| `secrets.stored` | `SecretsStoredEvent` | `(no reply)` | `secret_messages.go` |
| `storages.add` | `StorageAddMsg` | `StorageAddResp` | `storage_messages.go` |
| `storages.remove` | `StorageRemoveMsg` | `StorageRemoveResp` | `storage_messages.go` |
| `test.run` | `TestRunMsg` | `TestRunResp` | `testing_messages.go` |
| `tools.call` | `ToolCallMsg` | `ToolCallResp` | `tool_messages.go` |
| `tools.list` | `ToolListMsg` | `ToolListResp` | `tool_messages.go` |
| `tools.resolve` | `ToolResolveMsg` | `ToolResolveResp` | `tool_messages.go` |
| `trace.get` | `TraceGetMsg` | `TraceGetResp` | `tracing_messages.go` |
| `trace.list` | `TraceListMsg` | `TraceListResp` | `tracing_messages.go` |
| `vectors.add` | `VectorAddMsg` | `VectorAddResp` | `vector_messages.go` |
| `vectors.remove` | `VectorRemoveMsg` | `VectorRemoveResp` | `vector_messages.go` |
| `workflow.cancel` | `WorkflowCancelMsg` | `WorkflowCancelResp` | `workflow_messages.go` |
| `workflow.list` | `WorkflowListMsg` | `WorkflowListResp` | `workflow_messages.go` |
| `workflow.restart` | `WorkflowRestartMsg` | `WorkflowRestartResp` | `workflow_messages.go` |
| `workflow.resume` | `WorkflowResumeMsg` | `WorkflowResumeResp` | `workflow_messages.go` |
| `workflow.runs` | `WorkflowRunsMsg` | `WorkflowRunsResp` | `workflow_messages.go` |
| `workflow.start` | `WorkflowStartMsg` | `WorkflowStartResp` | `workflow_messages.go` |
| `workflow.startAsync` | `WorkflowStartAsyncMsg` | `WorkflowStartAsyncResp` | `workflow_messages.go` |
| `workflow.status` | `WorkflowStatusMsg` | `WorkflowStatusResp` | `workflow_messages.go` |
