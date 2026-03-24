# Test Fixtures — NEEDS UPDATE

These `.ts` fixtures use the OLD single-module import pattern:

```js
import { agent, createTool, z, output } from "kit";
```

They need updating to the new four-module system:

```js
import { generateText, streamText, z } from "ai";
import { Agent, createTool, Memory } from "agent";
import { bus, kit, model, tools, fs, output } from "kit";
```

Key changes needed:
- `agent()` wrapper is gone → use `new Agent()` + `kit.register("agent", name, ref)`
- `createTool()` no longer auto-registers → add `kit.register("tool", name, ref)` after creation
- `ai.generate()` / `ai.stream()` wrappers are gone → use `generateText()` / `streamText()` from `"ai"` directly
- `createMemory()` wrapper is gone → use `new Memory()` from `"agent"` directly
- `bus.send()` / `bus.publish()` → use `bus.emit()` / `bus.publish()` from `"kit"`
- All CallSettings (temperature, maxTokens, etc.) are now passed directly to AI SDK functions

These fixtures are not currently referenced by any Go test code — they were used by the deleted surface tests. They will be needed when we rewrite the surface tests.
