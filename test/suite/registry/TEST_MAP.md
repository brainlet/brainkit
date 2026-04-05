# Registry Test Map

**Purpose:** Verifies the provider registry (AI providers, vector stores, storages), JS bridge registry access, packages client search/fetch, runtime storage lifecycle, and input abuse resilience.
**Tests:** 25 functions across 4 files
**Entry point:** `registry_test.go` → `Run(t, env)`
**Campaigns:** transport (amqp, redis, postgres, nats, sqlite), fullstack (nats_postgres_rbac, amqp_postgres_vector, redis_mongodb)

## Files

### providers.go — Go-side + JS bridge registry operations

| Function | Purpose |
|----------|---------|
| testGoSideRegisterAndList | Creates kernel with OpenAI provider + PgVector + InMemory storage, verifies ListAIProviders/ListVectorStores/ListStorages return correct entries with types and capabilities |
| testGoSideRuntimeRegisterUnregister | Starts empty kernel, registers Anthropic provider + Qdrant vector at runtime, verifies they appear in lists, unregisters both, verifies lists are empty |
| testJSBridgeHas | Calls registry.has("provider", "openai") and registry.has("provider", "anthropic") from EvalTS, verifies true/false |
| testJSBridgeList | Creates kernel with 2 AI providers, calls registry.list("provider") from EvalTS, verifies count=2 and both names present |
| testJSBridgeResolve | Calls provider("openai") from EvalTS to resolve a provider instance, verifies resolved=true |
| testWithDeployedTS | Deploys .ts code that creates a tool using registry.has/registry.list, calls the tool, verifies it sees the registered OpenAI provider |

### packages_client.go — Registry client search/fetch tests

| Function | Purpose |
|----------|---------|
| testSearchByName | Starts test HTTP registry server, searches by name "echo", verifies exactly 1 result with correct name |
| testSearchByCapability | Searches by capability "gateway", verifies the telegram-gateway plugin is returned |
| testSearchMultipleCapabilities | Searches by capabilities ["tools", "testing"], verifies the echo plugin matches both |
| testSearchNoResults | Searches for "nonexistent-xyz", verifies empty results |
| testSearchAllPlugins | Searches with empty query and no capabilities, verifies all 2 plugins returned |
| testFetchManifest | Fetches manifest for brainlet/echo with no version, verifies name/owner/capabilities |
| testFetchManifestSpecificVersion | Fetches manifest for brainlet/echo version 1.0.0, verifies exact version match |
| testFetchManifestWrongVersion | Fetches manifest for version 99.0.0, verifies "not found" error |
| testFetchManifestNotFound | Fetches manifest for nonexistent plugin, verifies "not found" error |
| testMultipleRegistries | Creates client with 2 registry sources, searches all, verifies combined results from both |
| testRegistryWithAuth | Starts auth-protected registry, verifies unauthenticated returns empty, authenticated with Bearer token returns the plugin |

### storage_runtime.go — Runtime storage register/unregister

| Function | Purpose |
|----------|---------|
| testStorageRuntimeAddRemove | Adds in-memory storage at runtime, verifies StorageURL is empty, removes it cleanly |
| testStorageRuntimeAddDuplicate | Adds same storage name twice, verifies no panic (may overwrite or error) |
| testStorageRuntimeRemoveNonexistent | Removes nonexistent storage name, verifies idempotent (no error) |
| testStorageRuntimeURLForNonexistent | Gets StorageURL for nonexistent name, verifies empty string |
| testStorageRuntimeSQLiteAdd | Adds SQLite storage at runtime, verifies it gets a non-empty bridge URL, removes it |
| testStorageRuntimeListResources | Lists resources before and after deploying a .ts tool, verifies resource count increases and tool appears in filtered list |
| testStorageRuntimeResourcesFromSource | Deploys .ts that registers a tool + agent, calls ResourcesFrom(source), verifies at least 2 resources |
| testStorageRuntimeScalingPool | Creates an InstanceManager, verifies Pools/PoolInfo/Scale/KillPool all error correctly for nonexistent pools |
| testStorageRuntimeKernelMultipleStorages | Creates kernel with mem + sqlite storages, verifies mem has no URL but sqlite does, both visible from JS registry.has |

### input_abuse.go — Registry input abuse

| Function | Purpose |
|----------|---------|
| testInputAbuseEmptyProviderName | Calls kit.register("tool", "", {}) from EvalTS, verifies "required" error |
| testInputAbuseDuplicateRegister | Registers same tool name twice via kit.register from EvalTS, verifies no panic (overwrite or error) |
| testInputAbuseInvalidConfig | Calls kit.register("banana", ...) with invalid type, verifies "invalid type" error |
| testInputAbuseMissingType | Calls kit.register("", ...) with empty type, verifies error response |

## Cross-references

- **Campaigns:** transport/{amqp,redis,postgres,nats,sqlite}_test.go, fullstack/{nats_postgres_rbac,amqp_postgres_vector,redis_mongodb}_test.go
- **Related domains:** tools (tool registration), deploy (resource tracking)
- **Fixtures:** registry-related TS fixtures
