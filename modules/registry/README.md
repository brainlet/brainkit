# modules/registry - stable

Runtime admin surface for provider, storage, and vector registries. Startup
configuration still belongs in `brainkit.Config`; this module is the explicit
live mutation path.

## Bus commands

- `registry.list`, `registry.has`, `registry.resolve`
- `providers.add`, `providers.remove`
- `storages.add`, `storages.remove`
- `vectors.add`, `vectors.remove`

## Capabilities

- Requires: `brainkit.core.provider_registry`,
  `brainkit.core.registry_mutation`.
- Provides: none.

## Runtime resources

Registry responses redact credentials when returning resolved configuration.
The module owns only the admin bus command surface. Live provider, storage, and
vector mutations delegate to the typed `brainkit.core.registry_mutation`
capability so the runtime hosts own provider registry updates, storage/vector
bridge lifecycle, and active JS runtime cache invalidation as one explicit
workflow. When a JS runtime is mounted, successful bus-side mutations
synchronously invalidate the runtime cache for the changed entry; invalidation
failures are returned to the caller instead of being hidden, while the mutation
remains visible for explicit retry/repair.

## Hot unmount

Unmounting unregisters registry/provider/storage/vector command handlers and
drops registry/mutation capability references. Live storage/vector bridge
teardown remains owned by the runtime storage host.
