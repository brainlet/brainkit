# Provider Registry

The Kit owns four registries: AI providers, storages, vector stores,
and secrets. Each is a named table that Go code populates and JS code
resolves. When a deployed `.ts` package calls `model("openai",
"gpt-4o-mini")`, `storage("main")`, or `vectorStore("docs")`, the call
routes into the registry, finds the registration, and returns a live
Mastra / AI SDK instance.

## Four Registries

| Accessor           | Kit API                      | Purpose                                   |
| ------------------ | ---------------------------- | ----------------------------------------- |
| AI providers       | `kit.Providers()`            | Model factories for `generateText` et al. |
| Storages           | `kit.Storages()`             | Mastra stores (memory, message history).  |
| Vectors            | `kit.Vectors()`              | Embedding-backed retrieval stores.        |
| Secrets            | `kit.Secrets()`              | Encrypted key/value store.                |

All four accessors are lazily allocated and stable across calls.
Typical pattern:

```go
kit, _ := brainkit.New(brainkit.Config{
    Namespace: "analytics",
    Transport: brainkit.EmbeddedNATS(),
    FSRoot:    ".",
    SecretKey: "hex-32-bytes",
    Providers: []brainkit.ProviderConfig{
        brainkit.OpenAI(os.Getenv("OPENAI_API_KEY")),
        brainkit.Anthropic(os.Getenv("ANTHROPIC_API_KEY")),
    },
})

kit.Storages().Register("main", brainkit.StorageType("libsql"),
    types.LibSQLStorageConfig{URL: "file:./kit.db"})
kit.Vectors().Register("docs", brainkit.VectorStoreType("pgvector"),
    types.PgVectorConfig{URL: os.Getenv("PG_URL")})
_ = kit.Secrets().Set(ctx, "slack-token", os.Getenv("SLACK_TOKEN"))
```

## AI Providers

### 12 first-class providers

Top-level constructors build `ProviderConfig` values you pass to
`Config.Providers`:

```go
Providers: []brainkit.ProviderConfig{
    brainkit.OpenAI(key),
    brainkit.Anthropic(key),
    brainkit.Google(key),
    brainkit.Mistral(key),
    brainkit.Groq(key),
    brainkit.DeepSeek(key),
    brainkit.XAI(key),
    brainkit.Cohere(key),
    brainkit.Perplexity(key),
    brainkit.TogetherAI(key),
    brainkit.Fireworks(key),
    brainkit.Cerebras(key),
}
```

Each constructor accepts `brainkit.WithBaseURL(url)` and
`brainkit.WithHeaders(map[string]string{})` to swap endpoints or
inject custom auth headers.

### Advanced types

The underlying provider type table (`internal/types/providers.go`)
also knows Azure, Bedrock, Vertex, and HuggingFace — register these
by calling the runtime API directly:

```go
kit.Providers().Register("azure-main", brainkit.AIProviderType("azure"),
    types.AzureProviderConfig{...})
```

The JS runtime resolves the type string to a Mastra provider factory.
If a backing factory is not compiled into the current bundle, the
resolution fails fast with a `NOT_CONFIGURED` error — you only pay
for what you use.

### Runtime API

```go
kit.Providers().Register(name string, typ AIProviderType, config any) error
kit.Providers().Unregister(name string)
kit.Providers().List() []ProviderInfo
kit.Providers().Has(name string) bool
kit.Providers().Get(name string) (AIProviderRegistration, bool)
```

### JS resolution

```typescript
const ai = model("openai", "gpt-4o-mini");        // LanguageModel
const claude = model("anthropic", "claude-3-5-sonnet-latest");
const r = await generateText({ model: ai, prompt: "hi" });
```

Under the hood, `model(providerName, modelID)` reads
`globalThis.__kit_providers[providerName]`, finds the factory by type
(`openai` → `createOpenAI` from `@ai-sdk/openai`), instantiates it,
and wires the returned provider to the requested model ID.

Auto-detection from `os.Getenv` is off by default — pass explicit
`brainkit.OpenAI(os.Getenv("OPENAI_API_KEY"))` to wire a provider.
`process.env` inside the JS sandbox is the Go process's real
environment, so Mastra libraries that read keys from `process.env`
still work alongside registry-backed providers.

## Storages

Storages back Mastra's memory + message history + workflow snapshot
tables. A single registered name serves multiple Mastra features.

### Types

`StorageType` is a string alias with constants for every supported
backend:

```
memory, libsql, postgres, mongodb, upstash,
cloudflare-d1, cloudflare-kv, clickhouse, convex, couchbase,
dynamodb, lance, mssql, duckdb
```

Pass the corresponding `types.*StorageConfig` value when registering.
Only a subset is bundled at any given time — `libsql` and `postgres`
are the main-line backends, others are opt-in per release.

### Runtime API

```go
kit.Storages().Register(name string, typ StorageType, config any) error
kit.Storages().List() []StorageInfo
kit.Storages().Has(name string) bool
kit.Storages().Get(name string) (StorageRegistration, bool)
kit.Storages().Unregister(name string)
```

### JS resolution

```typescript
const s = storage("main");                          // Mastra store
const memory = new Memory({ storage: s });          // Mastra memory
```

`storage(name)` calls a Go bridge (`__go_registry_resolve`) that
returns `{type, config}`. The JS side switches on `type` to
instantiate the real Mastra class (`new LibSQLStore({...})`,
`new PostgresStore({...})`, etc.) and caches the instance for
subsequent calls.

## Vectors

Vector registries follow the same pattern:

```go
kit.Vectors().Register("qdrant", brainkit.VectorStoreType("qdrant"),
    types.QdrantVectorConfig{URL: "...", APIKey: "..."})
```

`VectorStoreType` constants cover `libsql`, `pgvector`, `mongodb`,
`pinecone`, `qdrant`, `chroma`, `upstash`, `astra`, `elasticsearch`,
`opensearch`, `turbopuffer`, `cloudflare`, `duckdb`, `lance`,
`convex`, `couchbase`, and `s3vectors`. Bundled backends are a
subset — the same NOT_CONFIGURED contract applies when you resolve
something the current bundle does not know how to build.

JS side:

```typescript
const v = vectorStore("qdrant");
await v.upsert({ indexName: "knowledge", vectors: [...], metadata: [...] });
```

## Secrets

`kit.Secrets()` is the only accessor that talks to a persistent
store. When `Config.SecretKey` is set, secrets are encrypted at rest
with AES-GCM and persisted via the configured `Config.SecretStore`
(or the Kit's default SQLite store when none is given).

```go
_ = kit.Secrets().Set(ctx, "slack-token", "xoxp-…")
val, _ := kit.Secrets().Get(ctx, "slack-token")
_ = kit.Secrets().Delete(ctx, "slack-token")
list, _ := kit.Secrets().List(ctx)   // metadata only, no values
_ = kit.Secrets().Rotate(ctx, "slack-token", "xoxp-new-…")
```

Without a `SecretKey`, the accessor falls back to an env-backed
read-only store: `Get` returns the value of `SECRETS_<NAME>` (or
`NAME` lower-cased) from the process environment, while `Set`,
`Delete`, and `Rotate` return a `NOT_CONFIGURED` error.

Secrets have a rotation hook — when the `plugins` module is wired,
rotating a secret referenced by a plugin's env restarts the plugin
to pick up the new value. This is what makes `kit.Secrets().Rotate`
safe to call on a running fleet.

## Health Probing

The `probes` module runs periodic health checks against every
registered provider/storage/vector. Each backend exposes a probe
endpoint:

- **AI provider** — HTTP GET to `/v1/models` (or the provider's
  equivalent) with the configured API key. Latency + availability.
- **Storage** — instantiate in JS and call `listThreads({})` once.
- **Vector** — instantiate in JS and call `listIndexes()` once.

Results are surfaced through the probes module's bus commands
(`probes.list`, `probes.run`) and, when the tracing module is loaded,
are emitted as OpenTelemetry attributes.

## Runtime Registration from JS

`.ts` packages can add providers at runtime through the `registry`
endowment:

```typescript
registry.register("provider", "custom", {
    type: "openai",
    apiKey: secret("openai-proxy"),
    baseURL: "https://proxy.internal/v1",
});
const ai = model("custom", "gpt-4o-mini");
```

This bridges to `kit.Providers().Register(...)` on the Go side. The
same pattern works for `"storage"` and `"vector"` categories.

## Summary

- Four typed registries: providers, storages, vectors, secrets.
- 12 shipped AI providers with top-level constructors; more types are
  registerable via `kit.Providers().Register`.
- Storages and vectors expose the full Mastra catalog as typed
  constants; bundle only what you use.
- Secrets are encrypted with `Config.SecretKey`, rotate-aware when
  plugins are wired.
- Every registration is visible to deployed `.ts` code through
  `model`, `storage`, `vectorStore`, `secret`, and `registry`
  endowments.

## See Also

- `providers.go`, `accessors.go` — Kit registration surface.
- `internal/types/providers.go` — the canonical type constants.
- [deployment-pipeline.md](deployment-pipeline.md) — how `.ts`
  packages see the registry through endowments.
- [error-handling.md](error-handling.md) — `NOT_CONFIGURED` and
  `NOT_FOUND` semantics for the registry.
