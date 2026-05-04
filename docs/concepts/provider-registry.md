# Provider Registry

Brainkit keeps named registries for AI providers, storages, vector stores, and
secrets. Startup configuration seeds those registries; optional modules expose
runtime administration over the bus.

Deployed `.ts` packages resolve the same tables with `model("openai", ...)`,
`storage("main")`, `vectorStore("docs")`, and `secret("NAME")`.

## Startup Configuration

AI providers are configured with top-level builders:

```go
import _ "github.com/brainlet/brainkit/storagebridges/sqlite"

kit, _ := brainkit.New(brainkit.Config{
    Namespace: "analytics",
    Transport: brainkit.Memory(),
    FSRoot:    ".",
    Providers: []brainkit.ProviderConfig{
        brainkit.OpenAI(os.Getenv("OPENAI_API_KEY")),
        brainkit.Anthropic(os.Getenv("ANTHROPIC_API_KEY")),
    },
    Storages: map[string]brainkit.StorageConfig{
        "main": brainkit.SQLiteStorage("./kit.db"),
    },
    Vectors: map[string]brainkit.VectorConfig{
        "docs": brainkit.SQLiteVector("./vectors.db"),
    },
})
```

The SQLite storage/vector builders remain root config constructors.
The SQLite runtime bridge is optional and linked explicitly with
`storagebridges/sqlite`.

`Config.Providers == nil` auto-detects common provider API keys from the
process environment. An explicitly empty provider slice disables auto-detect.

## Runtime Registry Module

Mount `modules/registry` when a Kit should accept runtime provider, storage, or
vector admin commands:

```go
kit, _ := brainkit.New(brainkit.Config{
    Transport: brainkit.Memory(),
    Modules: []module.Module{
        registrymod.New(),
    },
})

providerConfig, _ := json.Marshal(map[string]any{
    "APIKey":  os.Getenv("OPENAI_API_KEY"),
    "BaseURL": "https://proxy.internal/v1",
})
_, _ = registrymsg.CallProviderAdd(kit, ctx, registrymsg.ProviderAddMsg{
    Name:   "proxy-openai",
    Type:   "openai",
    Config: providerConfig,
})

list, _ := registrymsg.CallRegistryList(kit, ctx,
    registrymsg.RegistryListMsg{Category: "provider"})
```

Registry categories are `"provider"`, `"storage"`, and `"vectorStore"`.
Available commands:

| Command | Message |
|---|---|
| `registry.list` | `registrymsg.RegistryListMsg` |
| `registry.has` | `registrymsg.RegistryHasMsg` |
| `registry.resolve` | `registrymsg.RegistryResolveMsg` |
| `providers.add` / `providers.remove` | `registrymsg.ProviderAddMsg`, `ProviderRemoveMsg` |
| `storages.add` / `storages.remove` | `registrymsg.StorageAddMsg`, `StorageRemoveMsg` |
| `vectors.add` / `vectors.remove` | `registrymsg.VectorAddMsg`, `VectorRemoveMsg` |

The root `brainkit` package does not expose registry mutator accessors. Runtime
mutation is explicit module behavior. Internally, the registry command module
delegates live mutations through the typed `brainkit.core.registry_mutation`
capability, so provider registry updates, storage/vector bridge ownership, and
active JS runtime cache invalidation stay in the runtime hosts instead of the
bus command handlers.

## Probing Ownership

The registry stores probe results, but it does not schedule application
background work by default. Core exposes explicit `Kit.ProbeAll` /
`ProbeAllContext` helpers and the typed `brainkit.core.probe_all` capability.
Mount `modules/probes` when a Kit should run an initial sweep on mount or
periodic sweeps over providers, storages, and vector stores. Without that
module, health output marks registered providers as "not yet probed" until an
explicit probe runs.

## Secrets Module

Mount `modules/secrets` when a Kit should expose secret management commands:

```go
kit, _ := brainkit.New(brainkit.Config{
    Transport: brainkit.Memory(),
    Store:     store,
    SecretKey: "32-byte-or-longer-application-key",
    Modules: []module.Module{
        secretsmod.New(),
    },
})

_, _ = secretmsg.CallSecretsSet(kit, ctx,
    secretmsg.SecretsSetMsg{Name: "slack-token", Value: "xoxp-..."})
got, _ := secretmsg.CallSecretsGet(kit, ctx,
    secretmsg.SecretsGetMsg{Name: "slack-token"})
```

Available commands: `secrets.set`, `secrets.get`, `secrets.delete`,
`secrets.list`, and `secrets.rotate`. Rotation can restart plugins that
reference the secret when `modules/plugins` is also mounted.

Without `modules/secrets`, the secret store can still be used internally for
`$secret:NAME` resolution, but no `secrets.*` bus admin commands are registered.

## JS Resolution

```typescript
const ai = model("openai", "gpt-4o-mini");
const store = storage("main");
const vectors = vectorStore("docs");
const token = secret("slack-token");
```

The JS runtime reads the provider/storage/vector tables from the Kit registry
and instantiates the matching AI SDK or Mastra backend. If a backing factory is
not compiled into the current binary, resolution fails fast with a
`NOT_CONFIGURED` error.

`.ts` code can also register entries through the `registry` endowment:

```typescript
registry.register("provider", "custom", {
  type: "openai",
  apiKey: secret("openai-proxy"),
  baseURL: "https://proxy.internal/v1",
});
```

That bridge writes to the same registry table exposed by `modules/registry`.

## See Also

- `providers.go` and `storage.go` — startup configuration constructors.
- `modules/registry` — runtime provider/storage/vector admin module.
- `modules/secrets` — runtime secret admin module.
- [deployment-pipeline.md](deployment-pipeline.md) — how deployed packages see
  registry-backed endowments.
