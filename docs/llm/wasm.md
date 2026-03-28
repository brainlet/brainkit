# WASM / AssemblyScript — API Reference for brainkit

> `import { ... } from "brainkit";`
> Types from `kit/runtime/wasm_bundle.ts` and `kit/wasm_host.go`.

## Host Functions

All registered in wazero `"host"` module. AssemblyScript wrappers in `wasm_bundle.ts` use underscore prefix.

### Bus Messaging

```typescript
/** Publish with replyTo. Callback is an exported function name.
 *  For catalog commands (tools.call, fs.read, etc.), uses LocalInvoker — instant.
 *  For other topics, publishes to bus and subscribes for reply. */
export function _busPublish(topic: string, payload: string, callbackFuncName: string): void;

/** Fire-and-forget. No replyTo. */
export function _busEmit(topic: string, payload: string): void;

/** Subscribe to topic. INIT PHASE ONLY (inside init()). */
export function _busOn(topic: string, handlerFuncName: string): void;

/** Reply to current inbound message. */
export function _reply(payload: string): void;
```

### Shard Lifecycle

```typescript
/** Set execution mode. INIT PHASE ONLY.
 *  "stateless" = fresh instance per event, state discarded.
 *  "persistent" = state persists between events, saved to KitStore. */
export function _setMode(mode: string): void;

/** Register a tool. INIT PHASE ONLY.
 *  Handler must be exported: export function myHandler(topic: usize, payload: usize): void */
export function _tool(name: string, handlerFuncName: string): void;
```

### State

```typescript
/** Get value. Returns "" if not found. */
export function _getState(key: string): string;

/** Set value. */
export function _setState(key: string, value: string): void;

/** Check existence. Returns 1 if key exists, 0 otherwise.
 *  Distinguishes "not set" from "set to empty string". */
export function _hasState(key: string): bool;
```

State behavior by mode:
- **stateless**: scratch — discarded after handler returns
- **persistent**: persists between invocations. If KitStore configured, saved to SQLite.

### Logging

```typescript
/** Log at info level (level=1). */
export function _log(message: string): void;

// Raw host function: log(msgPtr: u32, level: u32)
// Level: 0=debug, 1=info, 2=warn, 3=error
```

## Handler Signature

All event handlers must be exported with:

```typescript
export function handleMyEvent(topic: usize, payload: usize): void {
    // topic and payload are AS string pointers
    // Use _reply() to respond
}
```

## Module Entry Points

### One-shot

```typescript
export function run(): i32 {
    // Return exit code
    return 0;
}
```

### Shard

```typescript
export function init(): void {
    _setMode("stateless"); // or "persistent"
    _busOn("my.topic", "handleMyTopic");
    _tool("my-tool", "handleMyTool");
}
```

`init()` is called once during `wasm.deploy`. All `_busOn`, `_tool`, and `_setMode` calls must happen here.

## JSON Library

Pure AssemblyScript — no external dependencies.

### JSONValue (base class)

```typescript
class JSONValue {
    static parse(json: string): JSONValue;
    toString(): string;
    isNull(): bool;
    isString(): bool;
    isNumber(): bool;
    isBool(): bool;
    isObject(): bool;
    isArray(): bool;
    asString(): string;
    asInt(): i32;
    asFloat(): f64;
    asBool(): bool;
    asObject(): JSONObject;
    asArray(): JSONArray;
}
```

### JSONObject

```typescript
class JSONObject extends JSONValue {
    constructor();
    getString(key: string): string;
    getInt(key: string): i32;
    getFloat(key: string): f64;
    getBool(key: string): bool;
    getObject(key: string): JSONObject;
    getArray(key: string): JSONArray;
    getValue(key: string): JSONValue;
    has(key: string): bool;
    keys(): string[];
    setString(key: string, value: string): JSONObject;
    setInt(key: string, value: i32): JSONObject;
    setFloat(key: string, value: f64): JSONObject;
    setBool(key: string, value: bool): JSONObject;
    setNull(key: string): JSONObject;
    setObject(key: string, value: JSONObject): JSONObject;
    setArray(key: string, value: JSONArray): JSONObject;
    toString(): string;
}
```

### JSONArray

```typescript
class JSONArray extends JSONValue {
    constructor();
    length: i32;
    getString(index: i32): string;
    getInt(index: i32): i32;
    getObject(index: i32): JSONObject;
    getValue(index: i32): JSONValue;
    pushString(value: string): JSONArray;
    pushInt(value: i32): JSONArray;
    pushFloat(value: f64): JSONArray;
    pushBool(value: bool): JSONArray;
    pushNull(): JSONArray;
    pushObject(value: JSONObject): JSONArray;
    pushArray(value: JSONArray): JSONArray;
    toString(): string;
}
```

## Complete Shard Example

```assemblyscript
import { _busOn, _setMode, _reply, _log, _getState, _setState, _hasState, _busPublish } from "brainkit";
import { JSONValue, JSONObject } from "brainkit";

export function init(): void {
    _setMode("persistent");
    _busOn("counter.inc", "handleInc");
    _busOn("counter.get", "handleGet");
    _busOn("counter.call-tool", "handleCallTool");
}

export function handleInc(topic: usize, payload: usize): void {
    var count: i32 = 0;
    if (_hasState("count") != 0) {
        count = parseInt(_getState("count")) as i32;
    }
    count++;
    _setState("count", count.toString());
    _reply(new JSONObject().setInt("count", count).toString());
}

export function handleGet(topic: usize, payload: usize): void {
    var count = _getState("count");
    _reply(new JSONObject().setString("count", count.length > 0 ? count : "0").toString());
}

export function handleCallTool(topic: usize, payload: usize): void {
    // Async: call a tool and get the result via callback
    _busPublish("tools.call", '{"name":"echo","input":{"message":"from-wasm"}}', "onToolResult");
}

export function onToolResult(topic: usize, payload: usize): void {
    _log("tool returned: " + payload);
    _reply(payload);
}
```

## Compilation Requirements

- ExportRuntime must be true (for `__new`, `__pin`, `__unpin` — needed for host string interop)
- The `"brainkit"` import resolves to `kit/runtime/wasm_bundle.ts` (auto-injected during compile)
- AS target: default (not `--runtime stub`)
