# Provider Registry

The provider registry maps named configurations to runtime instances. When .ts code calls `model("openai", "gpt-4o-mini")`, the registry resolves the provider config, creates an AI SDK provider instance, and returns a language model. The same pattern works for storage backends and vector stores.

## Three Categories

| Category | Go Registration | JS Resolution | Instance Created |
|----------|----------------|---------------|-----------------|
| AI Provider | Auto-detected from `os.Getenv` | `model(name, id)` / `provider(name)` | AI SDK provider factory (createOpenAI, etc.) |
| Storage | `KernelConfig.Storages` | `storage(name)` | Mastra store (InMemoryStore, PostgresStore, etc.) |
| Vector Store | `KernelConfig.Vectors` | `vectorStore(name)` | Mastra vector (LibSQLVector, PgVector, etc.) |

## Go-Side Registration

Providers are registered at Kernel creation time via typed config structs:

```go
// AI providers are auto-detected from os.Getenv (e.g. OPENAI_API_KEY, ANTHROPIC_API_KEY)
k, err := kit.NewKernel(kit.KernelConfig{
    Vectors: map[string]kit.VectorConfig{
        "main": kit.PgVectorStore("postgres://..."),
    },
    Storages: map[string]kit.StorageConfig{
        "default": kit.InMemoryStorage(),
    },
})
```

Runtime registration is also supported:

```go
k.RegisterAIProvider("groq", registry.AIProviderGroq, registry.GroqProviderConfig{APIKey: "gsk-..."})
k.RegisterVectorStore("docs", registry.VectorStoreLibSQL, registry.LibSQLVectorConfig{URL: "http://..."})
k.RegisterStorage("cache", registry.StorageInMemory, registry.InMemoryStorageConfig{})
```

## 47 Typed Config Structs

### AI Providers (15 types)

| Type Constant | Config Struct | Key Fields |
|---------------|--------------|------------|
| `AIProviderOpenAI` | `OpenAIProviderConfig` | APIKey, BaseURL |
| `AIProviderAnthropic` | `AnthropicProviderConfig` | APIKey, BaseURL |
| `AIProviderGoogle` | `GoogleProviderConfig` | APIKey, BaseURL |
| `AIProviderMistral` | `MistralProviderConfig` | APIKey, BaseURL |
| `AIProviderCohere` | `CohereProviderConfig` | APIKey, BaseURL |
| `AIProviderGroq` | `GroqProviderConfig` | APIKey, BaseURL |
| `AIProviderPerplexity` | `PerplexityProviderConfig` | APIKey, BaseURL |
| `AIProviderDeepSeek` | `DeepSeekProviderConfig` | APIKey, BaseURL |
| `AIProviderFireworks` | `FireworksProviderConfig` | APIKey, BaseURL |
| `AIProviderTogetherAI` | `TogetherAIProviderConfig` | APIKey, BaseURL |
| `AIProviderXAI` | `XAIProviderConfig` | APIKey, BaseURL |
| `AIProviderCerebras` | `CerebrasProviderConfig` | APIKey, BaseURL |
| `AIProviderAzure` | `AzureProviderConfig` | APIKey, BaseURL, ResourceName, DeploymentName |
| `AIProviderHuggingFace` | `HuggingFaceProviderConfig` | APIKey, BaseURL |
| `AIProviderBedrock` | `BedrockProviderConfig` | Region, AccessKeyID, SecretAccessKey |

### Storage Backends (14 types)

| Type Constant | Config Struct | Protocol |
|---------------|--------------|----------|
| `StorageInMemory` | `InMemoryStorageConfig` | N/A |
| `StorageLibSQL` | `LibSQLStorageConfig` | HTTP (Hrana) |
| `StoragePostgres` | `PostgresStorageConfig` | TCP |
| `StorageMongoDB` | `MongoDBStorageConfig` | TCP |
| `StorageUpstash` | `UpstashStorageConfig` | HTTP |
| `StorageClickHouse` | `ClickHouseStorageConfig` | HTTP |
| `StorageCloudflareD1` | `CloudflareD1StorageConfig` | HTTP |
| `StorageConvex` | `ConvexStorageConfig` | HTTP |
| `StorageDynamoDB` | `DynamoDBStorageConfig` | HTTP |
| `StorageLanceDB` | `LanceDBStorageConfig` | Embedded |
| `StorageMSSQL` | `MSSQLStorageConfig` | TCP |
| `StorageDuckDB` | `DuckDBStorageConfig` | Embedded |
| `StorageCouchbase` | `CouchbaseStorageConfig` | TCP |
| `StorageCloudflareKV` | `CloudflareKVStorageConfig` | HTTP |

### Vector Stores (16 types)

| Type Constant | Config Struct | Protocol |
|---------------|--------------|----------|
| `VectorStoreLibSQL` | `LibSQLVectorConfig` | HTTP |
| `VectorStorePg` | `PgVectorConfig` | TCP |
| `VectorStoreMongoDB` | `MongoDBVectorConfig` | TCP |
| `VectorStorePinecone` | `PineconeVectorConfig` | HTTP |
| `VectorStoreQdrant` | `QdrantVectorConfig` | HTTP |
| `VectorStoreChroma` | `ChromaVectorConfig` | HTTP |
| `VectorStoreUpstash` | `UpstashVectorConfig` | HTTP |
| `VectorStoreAstra` | `AstraVectorConfig` | HTTP |
| `VectorStoreElasticsearch` | `ElasticsearchVectorConfig` | HTTP |
| `VectorStoreOpenSearch` | `OpenSearchVectorConfig` | HTTP |
| `VectorStoreTurbopuffer` | `TurbopufferVectorConfig` | HTTP |
| `VectorStoreCloudflare` | `CloudflareVectorConfig` | HTTP |
| `VectorStoreDuckDB` | `DuckDBVectorConfig` | Embedded |
| `VectorStoreLanceDB` | `LanceDBVectorConfig` | Embedded |
| `VectorStoreConvex` | `ConvexVectorConfig` | HTTP |
| `VectorStoreS3` | `S3VectorConfig` | HTTP |

Most of these are type definitions waiting for bundle integration. Currently bundled and tested: InMemory, LibSQL, Postgres, MongoDB, Upstash (storage); LibSQL, Pg, MongoDB (vectors); OpenAI, Anthropic, Google, Mistral, Groq, DeepSeek, xAI, Cerebras, Perplexity, TogetherAI, Fireworks, Cohere (AI providers).

## JS-Side Resolution

When .ts code calls `model("openai", "gpt-4o-mini")`, the flow is:

1. `kit_runtime.js` `resolveModel(providerName, modelId)` reads from `globalThis.__kit_providers` (injected during Kernel init)
2. Looks up the provider factory name: `openai` → `createOpenAI`
3. Calls `embed.createOpenAI({ apiKey: pc.APIKey, baseURL: pc.BaseURL })(modelId)`
4. Returns a real AI SDK `LanguageModel` instance

For `storage("name")` and `vectorStore("name")`:

1. `kit_runtime.js` calls `__go_registry_resolve(category, name)` — a Go bridge function
2. Go looks up the registration in `ProviderRegistry`, returns JSON with type + config
3. JS switches on type and instantiates the real Mastra class: `new PostgresStore({ connectionString })`, `new LibSQLVector({ connectionUrl })`, etc.
4. Instances are cached per name — subsequent calls return the same instance (IIFE closure cache, not `this`-based, because endowment functions are detached from their object)

## AI Provider Auto-Detection

AI providers are auto-detected from `os.Getenv`. If `OPENAI_API_KEY` is set in the host environment, the OpenAI provider is available automatically in .ts code via `model("openai", "gpt-4o-mini")`. No explicit `AIProviders` map is needed.

The JS `process.env` proxy reads directly from the Go-backed environment, so Mastra libraries that read API keys from `process.env` (e.g., `process.env.OPENAI_API_KEY`) just work without explicit configuration.

## Health Probing

The registry supports live health probes for registered providers:

```go
result := k.ProbeAIProvider("openai")
// result.Available: true/false
// result.Latency: time.Duration
// result.Capabilities: KnownAICapabilities for this provider type
// result.Error: "" or error message
```

AI provider probes make an HTTP GET to `/models` with the provider's API key. Vector store and storage probes instantiate the store in JS and call a simple operation (`listIndexes()` for vectors, `listThreads({})` for storage).

Periodic probing runs automatically if `KernelConfig.Probe.PeriodicInterval` is set:

```go
kit.NewKernel(kit.KernelConfig{
    Probe: registry.ProbeConfig{
        PeriodicInterval: 5 * time.Minute,
        Timeout:          10 * time.Second,
    },
})
```

## Dynamic Registration from .ts

The `registry` object in .ts code supports runtime registration:

```typescript
// Register a new provider at runtime
registry.register("provider", "custom-openai", {
    type: "openai",
    apiKey: "sk-...",
    baseURL: "https://my-proxy.com/v1",
});

// Check availability
registry.has("provider", "custom-openai"); // true

// List all providers
registry.list("provider"); // [{name, type, healthy, lastProbed}]

// Unregister
registry.unregister("provider", "custom-openai");
```

These call `__go_brainkit_control("registry.register", ...)` which routes to the Go ProviderRegistry.
