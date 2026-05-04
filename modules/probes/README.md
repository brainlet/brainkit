# modules/probes — beta

Periodically exercises registered AI providers, vector stores, and
storage backends so `Kit.Health` carries live healthy/unhealthy
state. Probe results feed `ProviderInfo` / `StorageInfo` /
`VectorStoreInfo`.

## Usage

```go
import (
    "github.com/brainlet/brainkit"
    bkmodule "github.com/brainlet/brainkit/module"
    "github.com/brainlet/brainkit/modules/probes"
)

brainkit.New(brainkit.Config{
    Modules: []bkmodule.Module{probes.New(probes.Config{})},
})
```

Without the module, provider info surfaces as `healthy: true` with
`lastProbed: zero` — a "never checked" default rather than a lie.
Core keeps explicit `Kit.ProbeAll` / `ProbeAllContext` available, but it does
not schedule background provider probes on startup or JS runtime activation.

## Capabilities

- Requires: `brainkit.core.probe_all` as the typed, context-aware
  `module.ProbeRunner` capability.
- Uses when present: `brainkit.core.lifecycle_debug_registry`.
- Provides: none.

## Runtime resources

Owns `probes.loop`, the module-scoped provider/storage/vector probe loop. The
first sweep runs on mount by default (`ProbeOnRegister: true`), then optional
periodic sweeps run when `Interval > 0`. Each sweep receives the loop context
so unmount can cancel in-flight provider checks instead of leaving a stale
callback running after the module is gone. Lifecycle debug reports `closing`,
`closed`, loop state, and active sweep count.

## Hot unmount

Unmounting cancels the loop, waits for the active sweep to return, and drops
the probe runner after the loop has actually stopped. If a probe implementation
ignores context cancellation, unmount returns the caller context error rather
than blocking indefinitely; lifecycle debug keeps the runner attached and
`closed: false` until a later close retry completes.
