# Writing WASM Automation Modules

WASM automation modules are AssemblyScript programs that run inside brainlet's execution engine. They can call agents, invoke tools, manage state, and publish events — all through a typed API.

## Quick Start

```assemblyscript
import { log, setState, getState } from "wasm";

export function run(): i32 {
  log("Hello from WASM!");
  setState("count", "1");
  const val = getState("count");
  log("Count is: " + val);
  return 0; // success
}
```

Compile and run from a `.ts` orchestration file:

```typescript
import { wasm } from "kit";

const mod = await wasm.compile(asSource, { name: "my-module" });
const result = await wasm.run("my-module");
console.log(result.exitCode); // 0
```

## How It Works

1. You write an AssemblyScript module with `export function run(): i32`
2. The Kit compiles it to WASM via the embedded AS compiler
3. The Kit executes it via wazero with host functions bridging to the platform
4. Your module can call agents, tools, and publish events through the `"wasm"` import

## Importing from "wasm"

Every export is available via `import { ... } from "wasm"`:

```assemblyscript
import {
  // Logging
  log, debug, warn, error, logAt,

  // Agent & Tool calls
  callAgent, callTool, callToolRaw, parseResult,

  // State (per-execution key/value)
  getState, setState, hasState,

  // Bus (Kit event system)
  busSend, busSendRaw,

  // JSON library
  JSONValue, JSONObject, JSONArray,
} from "wasm";
```

The `"wasm"` module is automatically available — no setup needed.

## Logging

```assemblyscript
log("info message");       // level 1 (info)
debug("debug message");    // level 0
warn("warning");           // level 2
error("something broke");  // level 3
logAt("custom", 2);        // explicit level
```

## Calling Tools

Tools registered in the Kit (from Go or .ts) are callable:

```assemblyscript
import { callTool, callToolRaw, parseResult, JSONObject } from "wasm";

// Typed args (recommended)
const args = new JSONObject()
  .setString("query", "SELECT name FROM users")
  .setInt("limit", 10);
const raw = callTool("db_query", args);

// Raw JSON args (advanced)
const raw2 = callToolRaw("db_query", '{"query":"SELECT 1"}');

// Parse result
const result = parseResult(raw);
if (result.asObject().has("error")) {
  error("Tool failed: " + result.asObject().getString("error"));
  return 1;
}
const data = result.asObject().getString("data");
```

## Calling Agents

Agents created in .ts files are callable by name:

```assemblyscript
import { callAgent, parseResult } from "wasm";

const raw = callAgent("researcher", "Find papers on RLHF");
const result = parseResult(raw);

if (result.asObject().has("error")) {
  error("Agent failed");
  return 1;
}
const text = result.asObject().getString("text");
log("Agent said: " + text);
```

The response is always JSON: `{"text":"..."}` on success, `{"error":"..."}` on failure.

## State Management

Per-execution key/value state (fresh for each `wasm.run()`):

```assemblyscript
import { setState, getState, hasState } from "wasm";

setState("counter", "42");
const val = getState("counter");  // "42"
const exists = hasState("counter"); // true
const missing = hasState("nope");   // false
const empty = getState("nope");     // ""
```

State does NOT persist between runs. Use `busSend` to communicate results.

## Publishing Events

```assemblyscript
import { busSend, busSendRaw, JSONObject } from "wasm";

// Typed payload
busSend("automation.complete", new JSONObject()
  .setString("status", "done")
  .setInt("processed", 42));

// Raw JSON
busSendRaw("automation.log", '{"level":"info","msg":"done"}');
```

## JSON Library

Full JSON manipulation in pure AssemblyScript:

### Building

```assemblyscript
const obj = new JSONObject()
  .setString("name", "test")
  .setInt("count", 42)
  .setBool("active", true)
  .setNull("cleared")
  .setArray("tags", new JSONArray().pushString("a").pushString("b"))
  .setObject("meta", new JSONObject().setInt("v", 1));

const json = obj.toString();
// {"name":"test","count":42,"active":true,"cleared":null,"tags":["a","b"],"meta":{"v":1}}
```

### Parsing

```assemblyscript
const v = JSONValue.parse('{"name":"Alice","scores":[95,87,92]}');
if (v.isNull()) {
  error("parse failed");
  return 1;
}

const obj = v.asObject();
const name = obj.getString("name");    // "Alice"
const scores = obj.getArray("scores");
const first = scores.at(0).asInt();    // 95
```

### Error Handling

Parse errors return a null JSONValue (not abort):

```assemblyscript
const bad = JSONValue.parse("not json");
if (bad.isNull()) {
  // Handle gracefully
}
```

Type mismatch in `as*()` methods aborts (programming error):

```assemblyscript
const v = JSONValue.Str("hello");
v.asInt(); // ABORTS — "hello" is not a number
```

## Runtime Selection

Default is `incremental` (automatic GC). For short-lived modules:

```typescript
await wasm.compile(source, { name: "fast", runtime: "stub" });
```

| Runtime | GC | Best For |
|---------|-----|---------|
| `incremental` (default) | Automatic | General use |
| `minimal` | Manual (`__collect()`) | Host-controlled GC |
| `stub` | None | Short-lived, fast |

## Error Handling Pattern

```assemblyscript
import { callTool, parseResult, error, JSONObject } from "wasm";

export function run(): i32 {
  const raw = callTool("maybe_fails", new JSONObject());
  const result = parseResult(raw);

  if (result.isNull()) {
    error("Failed to parse response");
    return 1;
  }

  if (result.asObject().has("error")) {
    error("Tool error: " + result.asObject().getString("error"));
    return 2;
  }

  // Use result.asObject() safely
  return 0;
}
```
