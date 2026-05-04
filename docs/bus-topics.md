# Bus topic catalog

Generated from `sdk/**/*_messages.go` and `modules/**/*_messages.go` via `go run scripts/gen-bus-topics.go`. Do not edit by hand.

| Topic | Request | Response | Source |
|-------|---------|----------|--------|
| `agents.discover` | `AgentDiscoverMsg` | `AgentDiscoverResp` | `modules/agents/agentmsg/agent_messages.go` |
| `agents.get-status` | `AgentGetStatusMsg` | `AgentGetStatusResp` | `modules/agents/agentmsg/agent_messages.go` |
| `agents.list` | `AgentListMsg` | `AgentListResp` | `modules/agents/agentmsg/agent_messages.go` |
| `agents.set-status` | `AgentSetStatusMsg` | `AgentSetStatusResp` | `modules/agents/agentmsg/agent_messages.go` |
| `audit.prune` | `AuditPruneMsg` | `AuditPruneResp` | `modules/audit/auditmsg/audit_messages.go` |
| `audit.query` | `AuditQueryMsg` | `AuditQueryResp` | `modules/audit/auditmsg/audit_messages.go` |
| `audit.stats` | `AuditStatsMsg` | `AuditStatsResp` | `modules/audit/auditmsg/audit_messages.go` |
| `bus.handler.exhausted` | `HandlerExhaustedEvent` | `(no reply)` | `sdk/systemmsg/system_messages.go` |
| `bus.handler.failed` | `HandlerFailedEvent` | `(no reply)` | `sdk/systemmsg/system_messages.go` |
| `cluster.peers` | `ClusterPeersMsg` | `ClusterPeersResp` | `modules/control/control_messages.go` |
| `gateway.http.route.add` | `GatewayRouteAddMsg` | `GatewayRouteAddResp` | `modules/gateway/gatewaymsg/gateway_messages.go` |
| `gateway.http.route.list` | `GatewayRouteListMsg` | `GatewayRouteListResp` | `modules/gateway/gatewaymsg/gateway_messages.go` |
| `gateway.http.route.remove` | `GatewayRouteRemoveMsg` | `GatewayRouteRemoveResp` | `modules/gateway/gatewaymsg/gateway_messages.go` |
| `gateway.http.status` | `GatewayStatusMsg` | `GatewayStatusResp` | `modules/gateway/gatewaymsg/gateway_messages.go` |
| `kit.deployed` | `KitDeployedEvent` | `(no reply)` | `sdk/systemmsg/system_messages.go` |
| `kit.eval` | `KitEvalMsg` | `KitEvalResp` | `modules/eval/evalmsg/eval_messages.go` |
| `kit.health` | `KitHealthMsg` | `KitHealthResp` | `modules/health/health_messages.go` |
| `kit.lifecycle` | `KitLifecycleMsg` | `KitLifecycleResp` | `modules/control/control_messages.go` |
| `kit.module.describe` | `KitModuleDescribeMsg` | `KitModuleDescribeResp` | `modules/control/control_messages.go` |
| `kit.module.mount` | `KitModuleMountMsg` | `KitModuleMountResp` | `modules/control/control_messages.go` |
| `kit.module.unmount` | `KitModuleUnmountMsg` | `KitModuleUnmountResp` | `modules/control/control_messages.go` |
| `kit.modules` | `KitModulesMsg` | `KitModulesResp` | `modules/control/control_messages.go` |
| `kit.reference` | `KitReferenceMsg` | `KitReferenceResp` | `modules/reference/referencemsg/reference_messages.go` |
| `kit.reference.list` | `KitReferenceListMsg` | `KitReferenceListResp` | `modules/reference/referencemsg/reference_messages.go` |
| `kit.send` | `KitSendMsg` | `KitSendResp` | `modules/messaging/messaging_messages.go` |
| `kit.set-draining` | `KitSetDrainingMsg` | `KitSetDrainingResp` | `modules/control/control_messages.go` |
| `kit.teardown.done` | `KitTeardownedEvent` | `(no reply)` | `sdk/systemmsg/system_messages.go` |
| `mcp.callTool` | `McpCallToolMsg` | `McpCallToolResp` | `modules/mcp/mcpmsg/mcp_messages.go` |
| `mcp.listTools` | `McpListToolsMsg` | `McpListToolsResp` | `modules/mcp/mcpmsg/mcp_messages.go` |
| `metrics.get` | `MetricsGetMsg` | `MetricsGetResp` | `modules/metrics/metrics_messages.go` |
| `package.deploy` | `PackageDeployMsg` | `PackageDeployResp` | `modules/packages/packagemsg/package_messages.go` |
| `package.info` | `PackageDeployInfoMsg` | `PackageDeployInfoResp` | `modules/packages/packagemsg/package_messages.go` |
| `package.list` | `PackageListDeployedMsg` | `PackageListDeployedResp` | `modules/packages/packagemsg/package_messages.go` |
| `package.teardown` | `PackageTeardownMsg` | `PackageTeardownResp` | `modules/packages/packagemsg/package_messages.go` |
| `peers.list` | `PeersListMsg` | `PeersListResp` | `modules/topology/topology_messages.go` |
| `peers.resolve` | `PeersResolveMsg` | `PeersResolveResp` | `modules/topology/topology_messages.go` |
| `plugin.list` | `PluginListRunningMsg` | `PluginListRunningResp` | `modules/plugins/pluginmsg/plugin_messages.go` |
| `plugin.manifest` | `PluginManifestMsg` | `PluginManifestResp` | `modules/plugins/pluginmsg/plugin_messages.go` |
| `plugin.registered` | `PluginRegisteredEvent` | `(no reply)` | `modules/plugins/pluginmsg/plugin_messages.go` |
| `plugin.restart` | `PluginRestartMsg` | `PluginRestartResp` | `modules/plugins/pluginmsg/plugin_messages.go` |
| `plugin.start` | `PluginStartMsg` | `PluginStartResp` | `modules/plugins/pluginmsg/plugin_messages.go` |
| `plugin.started` | `PluginStartedEvent` | `(no reply)` | `modules/plugins/pluginmsg/plugin_messages.go` |
| `plugin.status` | `PluginStatusMsg` | `PluginStatusResp` | `modules/plugins/pluginmsg/plugin_messages.go` |
| `plugin.stop` | `PluginStopMsg` | `PluginStopResp` | `modules/plugins/pluginmsg/plugin_messages.go` |
| `plugin.stopped` | `PluginStoppedEvent` | `(no reply)` | `modules/plugins/pluginmsg/plugin_messages.go` |
| `providers.add` | `ProviderAddMsg` | `ProviderAddResp` | `modules/registry/registrymsg/registry_messages.go` |
| `providers.remove` | `ProviderRemoveMsg` | `ProviderRemoveResp` | `modules/registry/registrymsg/registry_messages.go` |
| `registry.has` | `RegistryHasMsg` | `RegistryHasResp` | `modules/registry/registrymsg/registry_messages.go` |
| `registry.list` | `RegistryListMsg` | `RegistryListResp` | `modules/registry/registrymsg/registry_messages.go` |
| `registry.resolve` | `RegistryResolveMsg` | `RegistryResolveResp` | `modules/registry/registrymsg/registry_messages.go` |
| `schedules.cancel` | `ScheduleCancelMsg` | `ScheduleCancelResp` | `modules/schedules/schedulemsg/schedule_messages.go` |
| `schedules.create` | `ScheduleCreateMsg` | `ScheduleCreateResp` | `modules/schedules/schedulemsg/schedule_messages.go` |
| `schedules.list` | `ScheduleListMsg` | `ScheduleListResp` | `modules/schedules/schedulemsg/schedule_messages.go` |
| `secrets.accessed` | `SecretsAccessedEvent` | `(no reply)` | `modules/secrets/secretmsg/secret_messages.go` |
| `secrets.delete` | `SecretsDeleteMsg` | `SecretsDeleteResp` | `modules/secrets/secretmsg/secret_messages.go` |
| `secrets.deleted` | `SecretsDeletedEvent` | `(no reply)` | `modules/secrets/secretmsg/secret_messages.go` |
| `secrets.get` | `SecretsGetMsg` | `SecretsGetResp` | `modules/secrets/secretmsg/secret_messages.go` |
| `secrets.list` | `SecretsListMsg` | `SecretsListResp` | `modules/secrets/secretmsg/secret_messages.go` |
| `secrets.rotate` | `SecretsRotateMsg` | `SecretsRotateResp` | `modules/secrets/secretmsg/secret_messages.go` |
| `secrets.rotated` | `SecretsRotatedEvent` | `(no reply)` | `modules/secrets/secretmsg/secret_messages.go` |
| `secrets.set` | `SecretsSetMsg` | `SecretsSetResp` | `modules/secrets/secretmsg/secret_messages.go` |
| `secrets.stored` | `SecretsStoredEvent` | `(no reply)` | `modules/secrets/secretmsg/secret_messages.go` |
| `storages.add` | `StorageAddMsg` | `StorageAddResp` | `modules/registry/registrymsg/registry_messages.go` |
| `storages.remove` | `StorageRemoveMsg` | `StorageRemoveResp` | `modules/registry/registrymsg/registry_messages.go` |
| `test.run` | `TestRunMsg` | `TestRunResp` | `modules/testing/testingmsg/testing_messages.go` |
| `tools.call` | `ToolCallMsg` | `ToolCallResp` | `modules/tools/toolmsg/tool_messages.go` |
| `tools.list` | `ToolListMsg` | `ToolListResp` | `modules/tools/toolmsg/tool_messages.go` |
| `tools.resolve` | `ToolResolveMsg` | `ToolResolveResp` | `modules/tools/toolmsg/tool_messages.go` |
| `trace.get` | `TraceGetMsg` | `TraceGetResp` | `modules/tracing/tracingmsg/tracing_messages.go` |
| `trace.list` | `TraceListMsg` | `TraceListResp` | `modules/tracing/tracingmsg/tracing_messages.go` |
| `vectors.add` | `VectorAddMsg` | `VectorAddResp` | `modules/registry/registrymsg/registry_messages.go` |
| `vectors.remove` | `VectorRemoveMsg` | `VectorRemoveResp` | `modules/registry/registrymsg/registry_messages.go` |
| `workflow.cancel` | `WorkflowCancelMsg` | `WorkflowCancelResp` | `modules/workflow/workflowmsg/workflow_messages.go` |
| `workflow.list` | `WorkflowListMsg` | `WorkflowListResp` | `modules/workflow/workflowmsg/workflow_messages.go` |
| `workflow.restart` | `WorkflowRestartMsg` | `WorkflowRestartResp` | `modules/workflow/workflowmsg/workflow_messages.go` |
| `workflow.resume` | `WorkflowResumeMsg` | `WorkflowResumeResp` | `modules/workflow/workflowmsg/workflow_messages.go` |
| `workflow.runs` | `WorkflowRunsMsg` | `WorkflowRunsResp` | `modules/workflow/workflowmsg/workflow_messages.go` |
| `workflow.start` | `WorkflowStartMsg` | `WorkflowStartResp` | `modules/workflow/workflowmsg/workflow_messages.go` |
| `workflow.startAsync` | `WorkflowStartAsyncMsg` | `WorkflowStartAsyncResp` | `modules/workflow/workflowmsg/workflow_messages.go` |
| `workflow.status` | `WorkflowStatusMsg` | `WorkflowStatusResp` | `modules/workflow/workflowmsg/workflow_messages.go` |
