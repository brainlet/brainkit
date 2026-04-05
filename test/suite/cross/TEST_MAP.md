# Cross Test Map

**Purpose:** Verifies cross-surface interactions (TS<->Go, plugin<->Go, TS<->plugin), Node commands, discovery, and cross-Kit pub/sub across transport backends
**Tests:** 37 functions across 6 files
**Entry point:** `cross_test.go` → `Run(t, env)`
**Campaigns:** plugins (nats, postgres, redis), crosskit (nats, postgres, redis)

## Files

### crosskit.go — TS<->Go cross-kit and plugin<->Go cross-kit

| Function | Purpose |
|----------|---------|
| testTSDeploysToolGoCallsIt | Deploys .ts that creates a tool via createTool+kit.register, Go calls it via ToolCallMsg, verifies result across all backends |
| testGoRegistersToolTSCallsViaDeploy | Deploys .ts that creates an "echo-wrapper" tool calling the Go "echo" tool, Go calls the wrapper, verifies nested result across all backends |
| testPluginToolCalledFromGo | Builds subprocess plugin, starts Node with NATS, calls plugin's "echo" tool from Go, verifies echoed message and plugin name (requires Podman) |
| testGoToolVisibleInList | Builds subprocess plugin, starts Node, lists tools, verifies both plugin tools (echo, concat) and host tool (host-multiply) appear |
| testTSCallsPluginTool | Builds subprocess plugin, deploys .ts that calls the plugin's "concat" tool via tools.call, verifies concatenated result (requires Podman+NATS) |
| testTSDeployedToolVisibleAlongsidePlugin | Builds subprocess plugin, deploys .ts tool, lists tools, verifies plugin tools and TS tool all appear together |

### plugins.go — In-process and subprocess plugin tests

| Function | Purpose |
|----------|---------|
| testPluginInProcessListTools | Creates a Node, lists tools, asserts the "echo" tool is visible from the plugin surface |
| testPluginInProcessCallTool | Creates a Node, calls "add" tool with a=100 b=200, asserts sum=300 |
| testPluginInProcessFSWriteRead | Creates kernel, writes/reads a file via EvalTS, asserts roundtrip match |
| testPluginInProcessDeployTeardown | Creates a Node, deploys .ts with tool, asserts Deployed=true, then tears down |
| testPluginInProcessAsyncSubscribe | Creates a Node, publishes ToolListMsg, subscribes to replyTo, asserts response arrives |
| testPluginSubprocessEcho | Builds subprocess plugin, starts Node with NATS, calls "echo" tool, verifies echoed message and plugin name (requires Podman) |
| testPluginSubprocessConcat | Builds subprocess plugin, calls "concat" tool with a="foo" b="bar", verifies "foobar" result |
| testPluginSubprocessHostToolStillWorks | Builds subprocess plugin, calls host-side "host-add" tool, verifies sum=30 |
| testPluginSubprocessToolsListShowsBoth | Builds subprocess plugin, lists tools, verifies echo, concat (plugin) and host-add (host) all appear |

### node_commands.go — Node-level bus commands (requires Podman+NATS)

| Function | Purpose |
|----------|---------|
| testNodeCommandsPluginList | Creates a Node, publishes PluginListRunningMsg, verifies response contains "plugins" |
| testNodeCommandsPluginStopNonexistent | Publishes PluginStopMsg for nonexistent plugin, asserts error response |
| testNodeCommandsPluginRestartNonexistent | Publishes PluginRestartMsg for nonexistent plugin, asserts error response |
| testNodeCommandsPluginStatusNonexistent | Publishes PluginStatusMsg for nonexistent plugin, asserts error response |
| testNodeCommandsPluginStateGetSet | Publishes PluginStateSetMsg then PluginStateGetMsg, verifies key-value roundtrip |
| testNodeCommandsPackageListEmpty | Publishes PackageListDeployedMsg, verifies response contains "packages" |
| testNodeCommandsDeployOnNode | Deploys .ts on a Node, sends a message, verifies reply, tears down |
| testNodeCommandsNodeShutdownClean | Starts a Node, deploys a service, closes cleanly, asserts no error |

### plugin_surface.go — Plugin surface operations (requires Podman+NATS)

| Function | Purpose |
|----------|---------|
| testPluginSurfaceGoToolFromPlugin | Registers a Go tool on a Node, calls it, verifies result with source="host" |
| testPluginSurfaceTSFromPlugin | Deploys .ts handler on a Node, sends message via bus, verifies reply from TS |
| testPluginSurfaceToolsList | Registers a tool on a Node, lists tools, verifies the tool appears |
| testPluginSurfaceErrorCodeFromNode | Calls nonexistent tool on a Node, asserts code="NOT_FOUND" |
| testPluginSurfaceSecretsFromNode | Sets and gets a secret on a Node, verifies value roundtrip |
| testPluginSurfaceDeployFromNode | Deploys .ts via bus on a Node, verifies tool is resolvable |

### discovery.go — Discovery provider tests

| Function | Purpose |
|----------|---------|
| testDiscoveryStaticPeers | Creates static provider with 2 peers, resolves both by name, verifies namespace mapping, asserts unknown name errors |
| testDiscoveryBrowse | Creates static provider with 3 peers, browses, asserts 3 peers with correct names |
| testDiscoveryRegister | Creates empty static provider, registers a peer, browses, asserts 1 peer |
| testDiscoveryResolveNonexistent | Creates empty static provider, resolves unknown name, asserts error |
| testDiscoveryClose | Creates static provider, closes, asserts no error |
| testDiscoveryStaticPeersBus | Creates a Node with static peers config, publishes PeersListMsg, verifies peer-a and peer-b appear |

### backend_matrix.go — Cross-Kit publish/reply and error propagation

| Function | Purpose |
|----------|---------|
| testCrossKitPublishReply | Creates 2 Nodes (A and B), deploys handler on B, A publishes to B's namespace, verifies reply (requires Podman) |
| testCrossKitErrorPropagation | Creates 2 Nodes, A calls nonexistent tool on B, verifies NOT_FOUND error code survives cross-Kit (requires Podman) |

## Cross-references

- **Campaigns:** `plugins/{nats,postgres,redis}_test.go`, `crosskit/{nats,postgres,redis}_test.go`
- **Related domains:** bus (bus operations), deploy (deploy on nodes), tools (tool calls across surfaces)
- **Fixtures:** testplugin binary (built by testutil.BuildTestPlugin)
