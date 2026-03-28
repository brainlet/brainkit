# WASM Shards

WASM shards are AssemblyScript modules compiled to WebAssembly and executed by brainkit's wazero runtime. Two execution models: one-shot (compile → run → done) and shard (event-reactive with persistent handlers).

## One-Shot Execution

Compile AS source, run it, get a result:

```assemblyscript
// Compile and run via Go
import { _log, _setState } from "brainkit";

export function run(): i32 {
    _log("Hello from WASM!");
    _setState("result", "42");
    return 0;  // exit code
}
```

```go
// From Go — pattern from test/e2e/scenarios_test.go
pr, _ := sdk.Publish(rt, ctx, messages.WasmCompileMsg{
    Source: `export function run(): i32 { return 99; }`,
    Options: &messages.WasmCompileOpts{Name: "my-module"},
})
// ... wait for compile response ...

pr2, _ := sdk.Publish(rt, ctx, messages.WasmRunMsg{ModuleID: "my-module"})
// ... wait for run response ...
// resp.ExitCode == 99
```

From .ts code:

```typescript
// fixtures/ts/wasm-compile-and-run/index.ts
const mod = await compile('export function run(): i32 { return 99; }', { name: "test" });
const result = await mod.run({});
output({ exitCode: result.exitCode }); // 99
```

## Shard Model (Event-Reactive)

Shards are deployed modules that react to bus events. `init()` registers handlers and sets the execution mode:

```assemblyscript
import { _busOn, _setMode, _reply, _log, _getState, _setState, _hasState } from "brainkit";

export function init(): void {
    _setMode("persistent");
    _busOn("order.new", "handleOrder");
    _busOn("order.status", "handleStatus");
}

export function handleOrder(topic: usize, payload: usize): void {
    _log("received order");
    _reply('{"processed":true}');
}

export function handleStatus(topic: usize, payload: usize): void {
    var count: i32 = 0;
    if (_hasState("orderCount") != 0) {
        count = parseInt(_getState("orderCount")) as i32;
    }
    count = count + 1;
    _setState("orderCount", count.toString());
    _reply('{"count":' + count.toString() + '}');
}
```

### Deploy lifecycle

```
wasm.compile(source, {name: "order-handler"})
    → AS compiler produces WASM binary
    → Binary stored in WASMService.modules

wasm.deploy(name: "order-handler")
    → Instantiate module, call init()
    → init() calls _setMode("persistent") + _busOn("order.new", "handleOrder")
    → Handlers registered, bus subscriptions created
    → ShardDescriptor returned: {mode: "persistent", handlers: {"order.new": "handleOrder", ...}}

Events arrive on "order.new"
    → WASMService.invokeShardHandler called
    → Fresh wazero instance (stateless) or living instance (persistent)
    → Handler function called with (topicPtr, payloadPtr)
    → Handler calls _reply() → response published to replyTo

wasm.undeploy(name: "order-handler")
    → Bus subscriptions cancelled
    → Shard removed from active shards map
    → State deleted from KitStore (if configured)
```

## Two Modes

### Stateless

Fresh wazero instance per event. State is scratch — discarded after handler returns. Events can process in parallel (each gets its own instance).

```assemblyscript
export function init(): void {
    _setMode("stateless");
    _busOn("process.item", "handle");
}

export function handle(topic: usize, payload: usize): void {
    // State set here is discarded when handler returns
    _setState("temp", "value");
    _reply('{"ok":true}');
}
```

### Persistent

State persists between invocations. If a KitStore is configured, state is also persisted to SQLite. Events serialize through the instance (one at a time — state is locked during handler execution).

```assemblyscript
// Pattern from test/e2e/scenarios_test.go — WasmShardLifecycle
export function init(): void {
    _setMode("persistent");
    _busOn("counter.inc", "handleInc");
}

export function handleInc(topic: usize, payload: usize): void {
    var count: i32 = 0;
    if (_hasState("eventCount") != 0) {
        count = parseInt(_getState("eventCount")) as i32;
    }
    count = count + 1;
    _setState("eventCount", count.toString());
    _reply('{"eventCount":' + count.toString() + '}');
}
```

After 5 events, `_getState("eventCount")` returns `"5"` — the state accumulated across invocations.

## 10 Host Functions

Registered in `kit/wasm_host.go` as wazero host module `"host"`:

| Function | Signature | Phase | Purpose |
|----------|-----------|-------|---------|
| `_log` | `(msg: string, level: i32)` | Any | Log message. Level: 0=debug, 1=info, 2=warn, 3=error |
| `_busEmit` | `(topic: string, payload: string)` | Any | Fire-and-forget bus publish |
| `_busPublish` | `(topic: string, payload: string, callbackFuncName: string)` | Any | Publish with replyTo. Callback receives the response |
| `_busOn` | `(topic: string, funcName: string)` | Init only | Subscribe to topic pattern |
| `_tool` | `(name: string, funcName: string)` | Init only | Register a tool this shard provides |
| `_reply` | `(payload: string)` | Handler | Reply to current inbound message |
| `_getState` | `(key: string) → string` | Any | Get state value (empty string if not found) |
| `_setState` | `(key: string, value: string)` | Any | Set state value |
| `_hasState` | `(key: string) → i32` | Any | 1 if key exists, 0 otherwise |
| `_setMode` | `(mode: string)` | Init only | Set shard mode: "stateless" or "persistent" |

Plus `abort` in the `"env"` module (AssemblyScript runtime requirement).

### Handler Signature

All handlers must be exported with signature `(topic: usize, payload: usize): void`. The parameters are AssemblyScript string pointers (read via `readASString` in Go). Reply via `_reply()`.

### _busPublish — Async Callbacks

`_busPublish` is the async pattern for WASM. The callback is the name of an exported function:

```assemblyscript
export function handleRequest(topic: usize, payload: usize): void {
    _busPublish("tools.call", '{"name":"echo","input":{"message":"hi"}}', "onToolResult");
}

export function onToolResult(topic: usize, payload: usize): void {
    // payload contains the tool call result
    _reply(payload);  // forward to original caller
}
```

For catalog commands (tools.call, fs.read, etc.), `_busPublish` uses the LocalInvoker — instant, in-process, no transport round-trip. For non-catalog topics, it publishes to the bus with replyTo and subscribes for the response.

The WASM instance stays alive until all pending `_busPublish` callbacks complete (tracked via `sync.WaitGroup`).

## Importing from "brainkit"

The AS source imports from `"brainkit"` — this resolves to `kit/runtime/wasm_bundle.ts`, which is auto-injected during compilation:

```assemblyscript
import { _busOn, _setMode, _reply, _log, _getState, _setState, _hasState } from "brainkit";
import { _busPublish, _busEmit } from "brainkit";
import { JSONValue, JSONObject, JSONArray } from "brainkit";
```

Note the underscore prefix: `_busOn`, `_busPublish`, etc. These are wrapper functions that call the actual host function imports.

## JSON Library

Pure AssemblyScript JSON parser/builder (no external dependencies):

```assemblyscript
import { JSONValue, JSONObject, JSONArray } from "brainkit";

// Parse
var parsed = JSONValue.parse('{"name":"Alice","age":30}');
var obj = parsed.asObject();
var name = obj.getString("name");  // "Alice"
var age = obj.getInt("age");       // 30

// Build
var result = new JSONObject()
    .setString("greeting", "Hello, " + name)
    .setInt("doubled", age * 2);
_reply(result.toString());
// → '{"greeting":"Hello, Alice","doubled":60}'

// Arrays
var arr = new JSONArray();
arr.pushString("one");
arr.pushInt(2);
arr.pushBool(true);
// → '["one",2,true]'
```

## Tool Registration from Shards

Shards can provide tools:

```assemblyscript
export function init(): void {
    _setMode("stateless");
    _tool("double", "handleDouble");
}

export function handleDouble(topic: usize, payload: usize): void {
    var input = JSONValue.parse(payload).asObject();
    var n = input.getInt("n");
    _reply('{"result":' + (n * 2).toString() + '}');
}
```

The tool is registered in the shared tool registry and callable from Go, .ts, plugins, and other WASM shards.

## KitStore Persistence

When `KernelConfig.Store` is set (e.g., `NewSQLiteStore("./data.db")`), WASM modules, shard descriptors, and persistent shard state are saved to SQLite:

- Compiled WASM binaries survive Kit restarts
- Shard descriptors (mode, handlers) are restored on startup
- Persistent shard state is loaded on first access and saved after each handler invocation

On `Node.Start()`, `restoreTransportSubscriptions` re-binds bus subscriptions for all restored shards.

## Lifecycle Commands

| Command | What it does |
|---------|-------------|
| `wasm.compile` | Compile AS source to WASM binary. Stores in WASMService.modules |
| `wasm.run` | Execute a compiled module's `run()` export. Returns exit code |
| `wasm.deploy` | Call `init()`, register handlers, subscribe to bus topics |
| `wasm.undeploy` | Cancel subscriptions, remove shard, delete state |
| `wasm.list` | List all compiled modules (metadata only) |
| `wasm.get` | Get a specific module's metadata |
| `wasm.remove` | Delete a compiled module (fails if shard is deployed) |
| `wasm.describe` | Get a deployed shard's descriptor (mode, handlers) |

## File Naming Gotcha

Go interprets `_wasm.go` and `_js.go` suffixes as platform build constraints. Never name Go files `handlers_wasm.go` — use `handlers_wasmdom.go` or `wasmmod.go` instead. This applies to any Go platform suffix: `_linux`, `_darwin`, `_windows`, `_amd64`.
