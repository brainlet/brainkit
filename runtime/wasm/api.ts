// runtime/wasm/api.ts — developer-facing API wrappers.
// Wraps raw @external host functions with typed, documented interfaces.
// In the bundle, host.ts is concatenated BEFORE this file, so _host_* functions are in scope.

// ── Logging ──────────────────────────────────────────────────

/** Log at info level */
export function log(message: string): void {
  _host_log(message, 1);
}

/** Log at specific level: 0=debug, 1=info, 2=warn, 3=error */
export function logAt(message: string, level: i32): void {
  _host_log(message, level);
}

/** Log at debug level */
export function debug(message: string): void {
  _host_log(message, 0);
}

/** Log at warn level */
export function warn(message: string): void {
  _host_log(message, 2);
}

/** Log at error level */
export function error(message: string): void {
  _host_log(message, 3);
}

// ── Agent & Tool Calls ───────────────────────────────────────

/** Call a named agent. Returns JSON: {"text":"..."} or {"error":"..."}. */
export function callAgent(name: string, prompt: string): string {
  return _host_call_agent(name, prompt);
}

/** Call a registered tool with typed JSONObject args. Returns JSON result. */
export function callTool(name: string, args: JSONObject): string {
  return _host_call_tool(name, args.toString());
}

/** Call a registered tool with raw JSON string. */
export function callToolRaw(name: string, argsJSON: string): string {
  return _host_call_tool(name, argsJSON);
}

/** Parse a JSON string into a typed JSONValue (convenience for JSONValue.parse). */
export function parseResult(jsonString: string): JSONValue {
  return JSONValue.parse(jsonString);
}

// ── State ────────────────────────────────────────────────────

/** Get a value from per-execution state. Returns "" if not found. */
export function getState(key: string): string {
  return _host_get_state(key);
}

/** Set a value in per-execution state. */
export function setState(key: string, value: string): void {
  _host_set_state(key, value);
}

/** Check if a key exists in state (distinguishes missing from empty). */
export function hasState(key: string): bool {
  return _host_has_state(key) != 0;
}

// ── Bus ──────────────────────────────────────────────────────

/** Publish a message on the Kit bus with typed payload. */
export function busSend(topic: string, payload: JSONObject): void {
  _host_bus_send(topic, payload.toString());
}

/** Publish a message on the Kit bus with raw JSON string. */
export function busSendRaw(topic: string, payloadJSON: string): void {
  _host_bus_send(topic, payloadJSON);
}
