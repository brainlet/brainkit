# WASM Module API Reference

> `import { ... } from "wasm"`

## Logging

| Function | Signature | Description |
|----------|-----------|-------------|
| `log` | `(message: string) => void` | Log at info level |
| `logAt` | `(message: string, level: i32) => void` | Log at specific level (0=debug, 1=info, 2=warn, 3=error) |
| `debug` | `(message: string) => void` | Log at debug level |
| `warn` | `(message: string) => void` | Log at warn level |
| `error` | `(message: string) => void` | Log at error level |

## Agent & Tool Calls

| Function | Signature | Description |
|----------|-----------|-------------|
| `callAgent` | `(name: string, prompt: string) => string` | Call a named agent. Returns JSON `{"text":"..."}` or `{"error":"..."}` |
| `callTool` | `(name: string, args: JSONObject) => string` | Call a tool with typed args. Returns JSON result. |
| `callToolRaw` | `(name: string, argsJSON: string) => string` | Call a tool with raw JSON string |
| `parseResult` | `(jsonString: string) => JSONValue` | Parse JSON string (convenience for `JSONValue.parse`) |

## State

| Function | Signature | Description |
|----------|-----------|-------------|
| `getState` | `(key: string) => string` | Get value (returns "" if missing) |
| `setState` | `(key: string, value: string) => void` | Set value |
| `hasState` | `(key: string) => bool` | Check if key exists (distinguishes missing from empty) |

State is per-execution — fresh for each `wasm.run()`, not persisted.

## Bus

| Function | Signature | Description |
|----------|-----------|-------------|
| `busSend` | `(topic: string, payload: JSONObject) => void` | Publish with typed payload |
| `busSendRaw` | `(topic: string, payloadJSON: string) => void` | Publish with raw JSON |

## JSON Classes

### JSONValue

| Method | Signature | Description |
|--------|-----------|-------------|
| `parse` (static) | `(json: string) => JSONValue` | Parse JSON. Returns null value on malformed input. |
| `Null` (static) | `() => JSONValue` | Create null value |
| `Bool` (static) | `(value: bool) => JSONValue` | Create bool value |
| `Number` (static) | `(value: f64) => JSONValue` | Create number value |
| `Str` (static) | `(value: string) => JSONValue` | Create string value |
| `Integer` (static) | `(value: i32) => JSONValue` | Create integer value |
| `isNull/isBool/isNumber/isString/isArray/isObject` | `() => bool` | Type checks |
| `asBool/asNumber/asInt/asString/asArray/asObject` | `() => T` | Typed accessors (abort on mismatch) |
| `toString` | `() => string` | Serialize to JSON |

### JSONObject

| Method | Signature | Description |
|--------|-----------|-------------|
| `has` | `(key: string) => bool` | Check key existence |
| `get` | `(key: string) => JSONValue` | Get value (null if missing) |
| `getString/getNumber/getInt/getBool/getObject/getArray` | `(key: string) => T` | Typed getters |
| `set` | `(key: string, value: JSONValue) => JSONObject` | Set value (chainable) |
| `setString/setNumber/setInt/setBool/setNull/setObject/setArray` | `(key, val) => JSONObject` | Typed setters (chainable) |
| `remove` | `(key: string) => bool` | Remove key |
| `keys` | `() => string[]` | Get all keys |
| `size` | `() => i32` | Number of entries |
| `toString` | `() => string` | Serialize to JSON |

### JSONArray

| Method | Signature | Description |
|--------|-----------|-------------|
| `length` | `i32` (readonly) | Number of elements |
| `at` | `(index: i32) => JSONValue` | Get element (abort on OOB) |
| `push` | `(value: JSONValue) => JSONArray` | Push value (chainable) |
| `pushString/pushNumber/pushInt/pushBool/pushNull/pushObject/pushArray` | `(val) => JSONArray` | Typed pushers (chainable) |
| `toString` | `() => string` | Serialize to JSON |
