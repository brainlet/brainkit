# modules/probes — beta

Periodically exercises registered AI providers, vector stores, and
storage backends so `Kit.Health` carries live healthy/unhealthy
state. Probe results feed `ProviderInfo` / `StorageInfo` /
`VectorStoreInfo`.

## Usage

```go
import (
    "github.com/brainlet/brainkit"
    "github.com/brainlet/brainkit/modules/probes"
)

brainkit.New(brainkit.Config{
    Modules: []brainkit.Module{probes.New(probes.Config{})},
})
```

Without the module, provider info surfaces as `healthy: true` with
`lastProbed: zero` — a "never checked" default rather than a lie.
