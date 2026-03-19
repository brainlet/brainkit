# WASM Automation Modules

WASM modules are AssemblyScript programs that run inside brainkit. Two execution models:

1. **One-shot (`wasm.run`)** — compile and execute, returns exit code
2. **Shard (`wasm.deploy`)** — event-reactive module with handlers, persists across events

## Import

All imports from `"brainkit"`:

```assemblyscript
import { setMode, on, tool, reply, log, setState, getState, hasState } from "brainkit";
import { bus } from "brainkit";
import { JSONValue, JSONObject, JSONArray } from "brainkit";
```

## One-Shot Execution

Compile, run, done:

```assemblyscript
import { log, setState } from "brainkit";

export function run(): i32 {
  log("Hello from WASM!");
  setState("result", "42");
  return 0;
}
```

```typescript
import { wasm } from "kit";
const mod = await wasm.compile(source, { name: "my-module" });
const result = await wasm.run("my-module");
```

## Shard Model (Event-Reactive)

Shards are deployed modules that react to bus events. `init()` registers handlers:

```assemblyscript
import { setMode, on, reply, log, getState, setState } from "brainkit";

export function init(): void {
  setMode("persistent");
  on("order.new", "handleOrder");
  on("order.status", "handleStatus");
}

export function handleOrder(topic: string, payload: string): void {
  log("received: " + payload);
  reply('{"processed":true}');
}

export function handleStatus(topic: string, payload: string): void {
  var count = getState("orderCount");
  reply('{"count":' + (count.length > 0 ? count : '0') + '}');
}
```

Deploy from .ts:

```typescript
const desc = await wasm.deploy("order-handler");
// desc.mode === "persistent"
// desc.handlers === {"order.new": "handleOrder", "order.status": "handleStatus"}
```

## Two Modes

### Stateless

Ephemeral. Fresh instance per event. State is scratch — discarded after handler returns. Events can process in parallel.

```assemblyscript
export function init(): void {
  setMode("stateless");
  on("process.item", "handle");
}
```

### Persistent

Living instance. WASM memory persists between invocations (smart contract model). `getState`/`setState` persists to Go-side store for durability. Events serialize through the instance.

```assemblyscript
export function init(): void {
  setMode("persistent");
  on("counter.inc", "handleInc");
  on("counter.get", "handleGet");
}

export function handleInc(topic: string, payload: string): void {
  var count: i32 = 0;
  var raw = getState("count");
  if (raw.length > 0) count = I32.parseInt(raw);
  count++;
  setState("count", count.toString());
  reply('{"count":' + count.toString() + '}');
}
```

## Handler Signature

All handlers: `(topic: string, payload: string): void`. Use `reply()` to respond.

## Registering Tools

Shards can provide tools:

```assemblyscript
export function init(): void {
  setMode("stateless");
  tool("double", "handleDouble");
}

export function handleDouble(topic: string, payload: string): void {
  reply('{"result":10}');
}
```

## Async Operations (askAsync)

Call tools, agents, or AI from within a handler:

```assemblyscript
import { bus } from "brainkit";

export function handleRequest(topic: string, payload: string): void {
  bus.askAsyncRaw("ai.generate", '{"model":"gpt-4o","prompt":"hello"}', "onAiResponse");
}

export function onAiResponse(topic: string, payload: string): void {
  setState("result", payload);
}
```

Instance stays alive until all pending askAsync callbacks complete.

## Host Functions (10)

| Function | Purpose |
|----------|---------|
| `send(topic, payload)` | Fire-and-forget bus message |
| `askAsync(topic, payload, callback)` | Async request/response |
| `on(topic, funcName)` | Subscribe to topic (init only) |
| `tool(name, funcName)` | Register a tool (init only) |
| `reply(payload)` | Reply to current message |
| `log(message, level)` | Log (0=debug, 1=info, 2=warn, 3=error) |
| `get_state(key)` | Get state value |
| `set_state(key, value)` | Set state value |
| `has_state(key)` | Check key existence |
| `set_mode(mode)` | Set shard mode (init only) |

## JSON Library

Pure AssemblyScript:

```assemblyscript
var obj = new JSONObject().setString("name", "test").setInt("count", 42);
var parsed = JSONValue.parse('{"name":"Alice"}');
var name = parsed.asObject().getString("name");
```

## Lifecycle

```
wasm.compile(source, {name}) → stored
wasm.deploy(name) → init() → handlers registered → ShardDescriptor returned
  events arrive → handler(topic, payload) → reply()/send()/askAsync()
wasm.undeploy(name) → handlers removed
wasm.remove(name) → module deleted
```
