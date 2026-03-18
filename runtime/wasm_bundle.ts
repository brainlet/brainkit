// runtime/wasm/host.ts — raw host function bindings.
// These are NOT exported to developers. api.ts wraps them with typed interfaces.

@external("host", "log")
export declare function _host_log(message: string, level: i32): void;

@external("host", "call_tool")
export declare function _host_call_tool(name: string, argsJSON: string): string;

@external("host", "call_agent")
export declare function _host_call_agent(name: string, prompt: string): string;

@external("host", "get_state")
export declare function _host_get_state(key: string): string;

@external("host", "set_state")
export declare function _host_set_state(key: string, value: string): void;

@external("host", "has_state")
export declare function _host_has_state(key: string): i32;

@external("host", "bus_send")
export declare function _host_bus_send(topic: string, payloadJSON: string): void;

@external("host", "set_mode")
export declare function _host_set_mode(mode: string): void;

@external("host", "set_mode_key")
export declare function _host_set_mode_key(key: string): void;

@external("host", "on_event")
export declare function _host_on_event(topic: string, funcName: string): void;
// runtime/wasm/json.ts — Pure AssemblyScript JSON library.
// Provides typed JSON building, parsing, and serialization.
// No host functions required — everything runs in WASM.

// ═══════════════════════════════════════════════════════════
// JSON Type Enum
// ═══════════════════════════════════════════════════════════

enum JSONType {
  NULL,
  BOOL,
  NUMBER,
  STRING,
  ARRAY,
  OBJECT,
}

// ═══════════════════════════════════════════════════════════
// String Escaping
// ═══════════════════════════════════════════════════════════

function escapeJsonString(s: string): string {
  let result = "";
  for (let i = 0; i < s.length; i++) {
    const ch = s.charCodeAt(i);
    if (ch == 0x22) {        // "
      result += '\\"';
    } else if (ch == 0x5C) { // backslash
      result += "\\\\";
    } else if (ch == 0x08) { // \b
      result += "\\b";
    } else if (ch == 0x0C) { // \f
      result += "\\f";
    } else if (ch == 0x0A) { // \n
      result += "\\n";
    } else if (ch == 0x0D) { // \r
      result += "\\r";
    } else if (ch == 0x09) { // \t
      result += "\\t";
    } else if (ch < 0x20) {  // other control chars
      result += "\\u00";
      const hi: i32 = (ch >> 4) & 0x0F;
      const lo: i32 = ch & 0x0F;
      result += String.fromCharCode(hi < 10 ? hi + 48 : hi + 87);
      result += String.fromCharCode(lo < 10 ? lo + 48 : lo + 87);
    } else {
      result += String.fromCharCode(ch);
    }
  }
  return result;
}

// ═══════════════════════════════════════════════════════════
// JSONValue — Base class for all JSON types
// ═══════════════════════════════════════════════════════════

export class JSONValue {
  _type: JSONType;
  _boolVal: bool;
  _numVal: f64;
  _strVal: string;
  _arrVal: Array<JSONValue>;
  _objKeys: Array<string>;
  _objVals: Array<JSONValue>;

  constructor(type: JSONType) {
    this._type = type;
    this._boolVal = false;
    this._numVal = 0;
    this._strVal = "";
    this._arrVal = new Array<JSONValue>();
    this._objKeys = new Array<string>();
    this._objVals = new Array<JSONValue>();
  }

  // ── Static Constructors ──────────────────────────────

  static Null(): JSONValue {
    return new JSONValue(JSONType.NULL);
  }

  static Bool(value: bool): JSONValue {
    const v = new JSONValue(JSONType.BOOL);
    v._boolVal = value;
    return v;
  }

  static Number(value: f64): JSONValue {
    const v = new JSONValue(JSONType.NUMBER);
    v._numVal = value;
    return v;
  }

  static Str(value: string): JSONValue {
    const v = new JSONValue(JSONType.STRING);
    v._strVal = value;
    return v;
  }

  static Integer(value: i32): JSONValue {
    return JSONValue.Number(f64(value));
  }

  // ── Type Checks ──────────────────────────────────────

  isNull(): bool { return this._type == JSONType.NULL; }
  isBool(): bool { return this._type == JSONType.BOOL; }
  isNumber(): bool { return this._type == JSONType.NUMBER; }
  isString(): bool { return this._type == JSONType.STRING; }
  isArray(): bool { return this._type == JSONType.ARRAY; }
  isObject(): bool { return this._type == JSONType.OBJECT; }

  // ── Type-Safe Accessors ──────────────────────────────

  asBool(): bool {
    assert(this._type == JSONType.BOOL, "JSONValue is not a bool");
    return this._boolVal;
  }

  asNumber(): f64 {
    assert(this._type == JSONType.NUMBER, "JSONValue is not a number");
    return this._numVal;
  }

  asInt(): i32 {
    assert(this._type == JSONType.NUMBER, "JSONValue is not a number");
    return i32(this._numVal);
  }

  asString(): string {
    assert(this._type == JSONType.STRING, "JSONValue is not a string");
    return this._strVal;
  }

  asArray(): JSONArray {
    assert(this._type == JSONType.ARRAY, "JSONValue is not an array");
    return changetype<JSONArray>(this);
  }

  asObject(): JSONObject {
    assert(this._type == JSONType.OBJECT, "JSONValue is not an object");
    return changetype<JSONObject>(this);
  }

  // ── Serialization ────────────────────────────────────

  toString(): string {
    switch (this._type) {
      case JSONType.NULL:
        return "null";
      case JSONType.BOOL:
        return this._boolVal ? "true" : "false";
      case JSONType.NUMBER: {
        // Check if integer
        if (this._numVal == f64(i64(this._numVal)) && this._numVal >= -9007199254740991.0 && this._numVal <= 9007199254740991.0) {
          return i64(this._numVal).toString();
        }
        return this._numVal.toString();
      }
      case JSONType.STRING:
        return '"' + escapeJsonString(this._strVal) + '"';
      case JSONType.ARRAY: {
        let result = "[";
        for (let i = 0; i < this._arrVal.length; i++) {
          if (i > 0) result += ",";
          result += this._arrVal[i].toString();
        }
        return result + "]";
      }
      case JSONType.OBJECT: {
        let result = "{";
        for (let i = 0; i < this._objKeys.length; i++) {
          if (i > 0) result += ",";
          result += '"' + escapeJsonString(this._objKeys[i]) + '":' + this._objVals[i].toString();
        }
        return result + "}";
      }
      default:
        return "null";
    }
  }

  // ── Parser Entry Point ───────────────────────────────

  static parse(json: string): JSONValue {
    const parser = new JSONParser(json);
    const result = parser.parseValue();
    if (result === null) {
      return JSONValue.Null();
    }
    return result!;
  }
}

// ═══════════════════════════════════════════════════════════
// JSONObject — Ordered key-value pairs
// ═══════════════════════════════════════════════════════════

export class JSONObject extends JSONValue {
  constructor() {
    super(JSONType.OBJECT);
  }

  // ── Existence ────────────────────────────────────────

  has(key: string): bool {
    for (let i = 0; i < this._objKeys.length; i++) {
      if (this._objKeys[i] == key) return true;
    }
    return false;
  }

  // ── Generic Getter ───────────────────────────────────

  get(key: string): JSONValue {
    for (let i = 0; i < this._objKeys.length; i++) {
      if (this._objKeys[i] == key) return this._objVals[i];
    }
    return JSONValue.Null();
  }

  // ── Typed Getters ────────────────────────────────────

  getString(key: string): string { return this.get(key).asString(); }
  getNumber(key: string): f64 { return this.get(key).asNumber(); }
  getInt(key: string): i32 { return this.get(key).asInt(); }
  getBool(key: string): bool { return this.get(key).asBool(); }
  getObject(key: string): JSONObject { return this.get(key).asObject(); }
  getArray(key: string): JSONArray { return this.get(key).asArray(); }

  // ── Chainable Setters ────────────────────────────────

  private _set(key: string, value: JSONValue): JSONObject {
    // Overwrite if key exists
    for (let i = 0; i < this._objKeys.length; i++) {
      if (this._objKeys[i] == key) {
        this._objVals[i] = value;
        return this;
      }
    }
    this._objKeys.push(key);
    this._objVals.push(value);
    return this;
  }

  set(key: string, value: JSONValue): JSONObject { return this._set(key, value); }
  setString(key: string, value: string): JSONObject { return this._set(key, JSONValue.Str(value)); }
  setNumber(key: string, value: f64): JSONObject { return this._set(key, JSONValue.Number(value)); }
  setInt(key: string, value: i32): JSONObject { return this._set(key, JSONValue.Integer(value)); }
  setBool(key: string, value: bool): JSONObject { return this._set(key, JSONValue.Bool(value)); }
  setNull(key: string): JSONObject { return this._set(key, JSONValue.Null()); }

  setObject(key: string, value: JSONObject): JSONObject {
    return this._set(key, changetype<JSONValue>(value));
  }

  setArray(key: string, value: JSONArray): JSONObject {
    return this._set(key, changetype<JSONValue>(value));
  }

  // ── Removal ──────────────────────────────────────────

  remove(key: string): bool {
    for (let i = 0; i < this._objKeys.length; i++) {
      if (this._objKeys[i] == key) {
        this._objKeys.splice(i, 1);
        this._objVals.splice(i, 1);
        return true;
      }
    }
    return false;
  }

  // ── Introspection ────────────────────────────────────

  keys(): Array<string> {
    return this._objKeys.slice();
  }

  size(): i32 {
    return this._objKeys.length;
  }
}

// ═══════════════════════════════════════════════════════════
// JSONArray — Ordered list of values
// ═══════════════════════════════════════════════════════════

export class JSONArray extends JSONValue {
  constructor() {
    super(JSONType.ARRAY);
  }

  get length(): i32 {
    return this._arrVal.length;
  }

  at(index: i32): JSONValue {
    assert(index >= 0 && index < this._arrVal.length, "JSONArray index out of bounds");
    return this._arrVal[index];
  }

  // ── Chainable Pushers ────────────────────────────────

  push(value: JSONValue): JSONArray {
    this._arrVal.push(value);
    return this;
  }

  pushString(value: string): JSONArray { return this.push(JSONValue.Str(value)); }
  pushNumber(value: f64): JSONArray { return this.push(JSONValue.Number(value)); }
  pushInt(value: i32): JSONArray { return this.push(JSONValue.Integer(value)); }
  pushBool(value: bool): JSONArray { return this.push(JSONValue.Bool(value)); }
  pushNull(): JSONArray { return this.push(JSONValue.Null()); }

  pushObject(value: JSONObject): JSONArray {
    return this.push(changetype<JSONValue>(value));
  }

  pushArray(value: JSONArray): JSONArray {
    return this.push(changetype<JSONValue>(value));
  }
}

// ═══════════════════════════════════════════════════════════
// JSONParser — Recursive descent parser
// ═══════════════════════════════════════════════════════════

class JSONParser {
  src: string;
  pos: i32;
  len: i32;

  constructor(src: string) {
    this.src = src;
    this.pos = 0;
    this.len = src.length;
  }

  private ch(): i32 {
    if (this.pos >= this.len) return -1;
    return this.src.charCodeAt(this.pos);
  }

  private advance(): void {
    this.pos++;
  }

  private skipWhitespace(): void {
    while (this.pos < this.len) {
      const c = this.src.charCodeAt(this.pos);
      if (c == 0x20 || c == 0x09 || c == 0x0A || c == 0x0D) {
        this.pos++;
      } else {
        break;
      }
    }
  }

  parseValue(): JSONValue | null {
    this.skipWhitespace();
    const c = this.ch();
    if (c == -1) return null;

    if (c == 0x7B) return this.parseObject();     // {
    if (c == 0x5B) return this.parseArray();       // [
    if (c == 0x22) return this.parseString();      // "
    if (c == 0x74) return this.parseTrue();        // t
    if (c == 0x66) return this.parseFalse();       // f
    if (c == 0x6E) return this.parseNull();        // n
    if (c == 0x2D || (c >= 0x30 && c <= 0x39)) {  // - or digit
      return this.parseNumber();
    }

    return null; // unexpected char
  }

  private parseObject(): JSONValue | null {
    this.advance(); // skip {
    this.skipWhitespace();

    const obj = new JSONObject();

    if (this.ch() == 0x7D) { // empty object
      this.advance();
      return changetype<JSONValue>(obj);
    }

    while (true) {
      this.skipWhitespace();
      if (this.ch() != 0x22) return null; // expected "

      const keyVal = this.parseString();
      if (keyVal === null) return null;
      const key = keyVal!.asString();

      this.skipWhitespace();
      if (this.ch() != 0x3A) return null; // expected :
      this.advance();

      const value = this.parseValue();
      if (value === null) return null;

      obj._set(key, value!);

      this.skipWhitespace();
      const c = this.ch();
      if (c == 0x7D) { // }
        this.advance();
        return changetype<JSONValue>(obj);
      }
      if (c == 0x2C) { // ,
        this.advance();
        continue;
      }
      return null; // unexpected
    }
  }

  private parseArray(): JSONValue | null {
    this.advance(); // skip [
    this.skipWhitespace();

    const arr = new JSONArray();

    if (this.ch() == 0x5D) { // empty array
      this.advance();
      return changetype<JSONValue>(arr);
    }

    while (true) {
      const value = this.parseValue();
      if (value === null) return null;
      arr._arrVal.push(value!);

      this.skipWhitespace();
      const c = this.ch();
      if (c == 0x5D) { // ]
        this.advance();
        return changetype<JSONValue>(arr);
      }
      if (c == 0x2C) { // ,
        this.advance();
        continue;
      }
      return null; // unexpected
    }
  }

  private parseString(): JSONValue | null {
    this.advance(); // skip opening "
    let result = "";

    while (this.pos < this.len) {
      const c = this.src.charCodeAt(this.pos);

      if (c == 0x22) { // closing "
        this.advance();
        return JSONValue.Str(result);
      }

      if (c == 0x5C) { // backslash
        this.advance();
        if (this.pos >= this.len) return null;
        const esc = this.src.charCodeAt(this.pos);
        this.advance();

        if (esc == 0x22) result += '"';
        else if (esc == 0x5C) result += "\\";
        else if (esc == 0x2F) result += "/";
        else if (esc == 0x62) result += "\b";
        else if (esc == 0x66) result += "\f";
        else if (esc == 0x6E) result += "\n";
        else if (esc == 0x72) result += "\r";
        else if (esc == 0x74) result += "\t";
        else if (esc == 0x75) { // \uXXXX
          if (this.pos + 4 > this.len) return null;
          const hex = this.src.substring(this.pos, this.pos + 4);
          const code = I32.parseInt(hex, 16);
          result += String.fromCharCode(code);
          this.pos += 4;
        } else {
          result += String.fromCharCode(esc);
        }
      } else {
        result += String.fromCharCode(c);
        this.advance();
      }
    }

    return null; // unterminated string
  }

  private parseNumber(): JSONValue | null {
    const start = this.pos;

    // Optional minus
    if (this.ch() == 0x2D) this.advance();

    // Integer part
    if (this.pos >= this.len) return null;
    const firstDigit = this.ch();
    if (firstDigit < 0x30 || firstDigit > 0x39) return null;

    if (firstDigit == 0x30) {
      this.advance(); // single zero
    } else {
      while (this.pos < this.len) {
        const d = this.src.charCodeAt(this.pos);
        if (d < 0x30 || d > 0x39) break;
        this.advance();
      }
    }

    // Fraction
    if (this.pos < this.len && this.src.charCodeAt(this.pos) == 0x2E) {
      this.advance();
      if (this.pos >= this.len) return null;
      const fd = this.src.charCodeAt(this.pos);
      if (fd < 0x30 || fd > 0x39) return null;
      while (this.pos < this.len) {
        const d = this.src.charCodeAt(this.pos);
        if (d < 0x30 || d > 0x39) break;
        this.advance();
      }
    }

    // Exponent
    if (this.pos < this.len) {
      const ec = this.src.charCodeAt(this.pos);
      if (ec == 0x65 || ec == 0x45) { // e or E
        this.advance();
        if (this.pos < this.len) {
          const sign = this.src.charCodeAt(this.pos);
          if (sign == 0x2B || sign == 0x2D) this.advance(); // + or -
        }
        if (this.pos >= this.len) return null;
        const ed = this.src.charCodeAt(this.pos);
        if (ed < 0x30 || ed > 0x39) return null;
        while (this.pos < this.len) {
          const d = this.src.charCodeAt(this.pos);
          if (d < 0x30 || d > 0x39) break;
          this.advance();
        }
      }
    }

    const numStr = this.src.substring(start, this.pos);
    const value = parseFloat(numStr);
    return JSONValue.Number(value);
  }

  private parseTrue(): JSONValue | null {
    if (this.pos + 4 > this.len) return null;
    if (this.src.substring(this.pos, this.pos + 4) == "true") {
      this.pos += 4;
      return JSONValue.Bool(true);
    }
    return null;
  }

  private parseFalse(): JSONValue | null {
    if (this.pos + 5 > this.len) return null;
    if (this.src.substring(this.pos, this.pos + 5) == "false") {
      this.pos += 5;
      return JSONValue.Bool(false);
    }
    return null;
  }

  private parseNull(): JSONValue | null {
    if (this.pos + 4 > this.len) return null;
    if (this.src.substring(this.pos, this.pos + 4) == "null") {
      this.pos += 4;
      return JSONValue.Null();
    }
    return null;
  }
}
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

// ── Shard Registration (init phase only) ─────────────────────

/** Set shard state mode: "stateless" or "shared" */
export function setMode(mode: string): void {
  _host_set_mode(mode);
}

/** Set keyed mode with the payload field name used as state key */
export function setModeKeyed(keyField: string): void {
  _host_set_mode_key(keyField);
}

/** Register an event handler: topic pattern -> exported function name */
export function onEvent(topic: string, handlerName: string): void {
  _host_on_event(topic, handlerName);
}

// runtime/wasm/index.ts — public API surface.
// In the concatenated bundle, all exports come from json.ts and api.ts directly.
// This file exists as the logical entry point for the 4-file development layout.
// When building the bundle, this file is included but adds no new exports
// (json.ts and api.ts already export everything developers need).
