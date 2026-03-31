/**
 * "brainkit" module — AssemblyScript WASM runtime library.
 *
 * Provides bus messaging, state management, logging, JSON handling,
 * and shard lifecycle for WASM modules compiled with the AS compiler.
 *
 * @example
 * ```ts
 * import { setMode, on, reply, log } from "brainkit";
 *
 * export function init(): void {
 *   setMode("stateless");
 *   on("echo", "handleEcho");
 * }
 *
 * export function handleEcho(topic: string, payload: string): void {
 *   log("received: " + payload);
 *   reply(payload);
 * }
 * ```
 */
declare module "brainkit" {

  // ── Bus Messaging ──────────────────────────────────────────

  /** Publish to bus with replyTo. Callback function receives the reply. */
  export function publish(topic: string, payload: string, callbackFuncName: string): void;

  /** Fire-and-forget bus publish. No replyTo, no callback. */
  export function emit(topic: string, payload: string): void;

  /** Subscribe to a topic pattern. Handler is called when messages match. Init phase only. */
  export function on(topic: string, handlerFuncName: string): void;

  /** Reply to the current inbound message with a payload. */
  export function reply(payload: string): void;

  // ── Shard Lifecycle ────────────────────────────────────────

  /** Set shard execution mode: "stateless" or "persistent". Init phase only. */
  export function setMode(mode: string): void;

  /** Register a tool this shard provides. Init phase only. */
  export function tool(name: string, handlerFuncName: string): void;

  // ── State Management ───────────────────────────────────────

  /** Get a value from per-execution state. Returns "" if not found. */
  export function getState(key: string): string;

  /** Set a value in per-execution state. */
  export function setState(key: string, value: string): void;

  /** Check if a key exists in state. */
  export function hasState(key: string): bool;

  // ── Logging ────────────────────────────────────────────────

  /** Log at info level. */
  export function log(message: string): void;

  /** Log at specific level: 0=debug, 1=info, 2=warn, 3=error. */
  export function logAt(message: string, level: i32): void;

  /** Log at debug level. */
  export function debug(message: string): void;

  /** Log at warn level. */
  export function warn(message: string): void;

  /** Log at error level. */
  export function error(message: string): void;

  // ── JSON Library ───────────────────────────────────────────

  /** Parsed JSON value — can be null, bool, number, string, object, or array. */
  export class JSONValue {
    static parse(s: string): JSONValue;
    static Null(): JSONValue;
    static Bool(value: bool): JSONValue;
    static Integer(value: i32): JSONValue;
    static Number(value: f64): JSONValue;
    static Str(value: string): JSONValue;

    isNull(): bool;
    isBool(): bool;
    isNumber(): bool;
    isString(): bool;
    isObject(): bool;
    isArray(): bool;

    asBool(): bool;
    asInt(): i32;
    asNumber(): f64;
    asString(): string;
    asObject(): JSONObject;
    asArray(): JSONArray;

    toString(): string;
  }

  /** JSON object with string keys. */
  export class JSONObject extends JSONValue {
    constructor();

    getString(key: string): string;
    getNumber(key: string): f64;
    getInt(key: string): i32;
    getBool(key: string): bool;
    getObject(key: string): JSONObject;
    getArray(key: string): JSONArray;
    get(key: string): JSONValue;
    has(key: string): bool;
    keys(): string[];
    size(): i32;

    setString(key: string, value: string): JSONObject;
    setNumber(key: string, value: f64): JSONObject;
    setInt(key: string, value: i32): JSONObject;
    setBool(key: string, value: bool): JSONObject;
    setObject(key: string, value: JSONObject): JSONObject;
    setArray(key: string, value: JSONArray): JSONObject;
    setNull(key: string): JSONObject;
    set(key: string, value: JSONValue): JSONObject;

    toString(): string;
  }

  /** JSON array. */
  export class JSONArray extends JSONValue {
    constructor();

    length: i32;

    getString(index: i32): string;
    getNumber(index: i32): f64;
    getInt(index: i32): i32;
    getBool(index: i32): bool;
    getObject(index: i32): JSONObject;
    getArray(index: i32): JSONArray;
    get(index: i32): JSONValue;
    at(index: i32): JSONValue;

    pushString(value: string): JSONArray;
    pushNumber(value: f64): JSONArray;
    pushInt(value: i32): JSONArray;
    pushBool(value: bool): JSONArray;
    pushObject(value: JSONObject): JSONArray;
    pushArray(value: JSONArray): JSONArray;
    pushNull(): JSONArray;
    push(value: JSONValue): JSONArray;

    toString(): string;
  }

  // ── Typed Message Interface ────────────────────────────────

  /** All typed bus messages implement this. */
  export interface BusMsg {
    topic(): string;
    toJSON(): string;
  }
}
