# AS Test Fixtures — NEEDS UPDATE

These AssemblyScript shard fixtures use old host function names:

```ts
import { _on, _send, _invokeAsync } from "brainkit";
```

They need updating to the new names:

```ts
import { _busOn, _busEmit, _busPublish } from "brainkit";
```

Rename mapping:
- `_on()` → `_busOn()`
- `_send()` → `_busEmit()`
- `_invokeAsync()` → `_busPublish()`
- `on()` → stays `on()` (exported from shard.ts, calls `_busOn` internally)
- `reply()` → unchanged
- `setMode()` → unchanged

Some fixtures also reference removed domain types (`AiGenerateMsg`, `AgentRequestMsg`, etc.) that no longer exist in the WASM bundle. Those fixtures need rewriting to use the bus pattern (send messages to .ts services instead of calling catalog commands directly).
