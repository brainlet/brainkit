/**
 * WASM Module Type Definitions
 *
 * These types define the developer-facing API for AssemblyScript automation
 * modules running on the brainlet platform. Everything is imported from "wasm".
 *
 * @example
 * ```assemblyscript
 * import { log, callTool, JSONObject, parseResult } from "wasm";
 *
 * export function run(): i32 {
 *   const args = new JSONObject().setString("query", "SELECT 1");
 *   const raw = callTool("db", args);
 *   const result = parseResult(raw);
 *   log("Got: " + result.asObject().getString("data"));
 *   return 0;
 * }
 * ```
 *
 * @see brainkit-maps/brainkit/specs/2026-03-18-phase4-as-developer-experience.md
 */
declare module "wasm" {

  // ═══════════════════════════════════════════════════════════
  // Logging
  // ═══════════════════════════════════════════════════════════

  /** Log at info level (most common). */
  export function log(message: string): void;

  /** Log at specific level: 0=debug, 1=info, 2=warn, 3=error. */
  export function logAt(message: string, level: i32): void;

  /** Log at debug level. */
  export function debug(message: string): void;

  /** Log at warn level. */
  export function warn(message: string): void;

  /** Log at error level. */
  export function error(message: string): void;

  // ═══════════════════════════════════════════════════════════
  // Agent & Tool Calls
  // ═══════════════════════════════════════════════════════════

  /**
   * Call a named agent and get its response.
   * Returns JSON: `{"text":"..."}` on success, `{"error":"..."}` on failure.
   * The agent must be created in a .ts file before calling from WASM.
   *
   * @example
   * ```assemblyscript
   * const raw = callAgent("helper", "explain RLHF");
   * const result = parseResult(raw);
   * if (result.asObject().has("error")) {
   *   error("Agent failed: " + result.asObject().getString("error"));
   *   return 1;
   * }
   * const text = result.asObject().getString("text");
   * ```
   */
  export function callAgent(name: string, prompt: string): string;

  /**
   * Call a registered tool with typed JSONObject args.
   * Returns JSON result string. On error: `{"error":"..."}`.
   *
   * @example
   * ```assemblyscript
   * const args = new JSONObject().setString("query", "SELECT 1");
   * const raw = callTool("db_query", args);
   * ```
   */
  export function callTool(name: string, args: JSONObject): string;

  /** Call a registered tool with a raw JSON string (advanced escape hatch). */
  export function callToolRaw(name: string, argsJSON: string): string;

  /**
   * Parse a JSON string into a typed JSONValue.
   * Returns a null JSONValue (isNull() == true) on malformed input.
   * Pure AS convenience — equivalent to `JSONValue.parse(jsonString)`.
   */
  export function parseResult(jsonString: string): JSONValue;

  // ═══════════════════════════════════════════════════════════
  // State (per-execution key/value)
  // ═══════════════════════════════════════════════════════════

  /** Get a value from per-execution state. Returns "" if not found. */
  export function getState(key: string): string;

  /** Set a value in per-execution state. */
  export function setState(key: string, value: string): void;

  /** Check if a key exists in state (distinguishes "not set" from "set to empty"). */
  export function hasState(key: string): bool;

  // ═══════════════════════════════════════════════════════════
  // Bus (Kit event system)
  // ═══════════════════════════════════════════════════════════

  /** Publish a message on the Kit bus with a typed JSONObject payload. */
  export function busSend(topic: string, payload: JSONObject): void;

  /** Publish a message on the Kit bus with a raw JSON string (advanced). */
  export function busSendRaw(topic: string, payloadJSON: string): void;

  // ═══════════════════════════════════════════════════════════
  // JSON Library
  // ═══════════════════════════════════════════════════════════

  /**
   * Represents any JSON value (string, number, bool, null, object, array).
   * Use type check methods (isString, isObject, etc.) before accessing typed values.
   * Type-safe accessors (asString, asObject, etc.) abort on type mismatch.
   */
  export class JSONValue {
    /** Parse a JSON string. Returns null JSONValue on malformed input. */
    static parse(json: string): JSONValue;

    /** Static constructors */
    static Null(): JSONValue;
    static Bool(value: bool): JSONValue;
    static Number(value: f64): JSONValue;
    static Str(value: string): JSONValue;
    static Integer(value: i32): JSONValue;

    /** Type checks */
    isNull(): bool;
    isBool(): bool;
    isNumber(): bool;
    isString(): bool;
    isArray(): bool;
    isObject(): bool;

    /** Type-safe accessors (abort on type mismatch) */
    asBool(): bool;
    asNumber(): f64;
    asInt(): i32;
    asString(): string;
    asArray(): JSONArray;
    asObject(): JSONObject;

    /** Serialize to JSON string */
    toString(): string;
  }

  /**
   * A JSON object — ordered key-value pairs.
   * Setters are chainable: `new JSONObject().setString("a", "b").setInt("c", 1)`
   */
  export class JSONObject extends JSONValue {
    constructor();

    /** Check if key exists */
    has(key: string): bool;

    /** Get value by key (returns null JSONValue if missing) */
    get(key: string): JSONValue;

    /** Typed getters (abort if key missing or wrong type) */
    getString(key: string): string;
    getNumber(key: string): f64;
    getInt(key: string): i32;
    getBool(key: string): bool;
    getObject(key: string): JSONObject;
    getArray(key: string): JSONArray;

    /** Chainable typed setters */
    set(key: string, value: JSONValue): JSONObject;
    setString(key: string, value: string): JSONObject;
    setNumber(key: string, value: f64): JSONObject;
    setInt(key: string, value: i32): JSONObject;
    setBool(key: string, value: bool): JSONObject;
    setNull(key: string): JSONObject;
    setObject(key: string, value: JSONObject): JSONObject;
    setArray(key: string, value: JSONArray): JSONObject;

    /** Remove a key. Returns true if key existed. */
    remove(key: string): bool;

    /** Get all keys */
    keys(): string[];

    /** Number of key-value pairs */
    size(): i32;

    /** Serialize to JSON string */
    toString(): string;
  }

  /**
   * A JSON array — ordered list of values.
   * Pushers are chainable: `new JSONArray().pushString("a").pushInt(1)`
   */
  export class JSONArray extends JSONValue {
    constructor();

    /** Number of elements */
    readonly length: i32;

    /** Get element at index (abort on out of bounds) */
    at(index: i32): JSONValue;

    /** Chainable typed pushers */
    push(value: JSONValue): JSONArray;
    pushString(value: string): JSONArray;
    pushNumber(value: f64): JSONArray;
    pushInt(value: i32): JSONArray;
    pushBool(value: bool): JSONArray;
    pushNull(): JSONArray;
    pushObject(value: JSONObject): JSONArray;
    pushArray(value: JSONArray): JSONArray;

    /** Serialize to JSON string */
    toString(): string;
  }
}
