// AUTO-GENERATED — do not edit. Run scripts/bundle_wasm.sh to regenerate.
// Source files: 16 files from runtime/wasm/

// ════════════════════════════════════════════════════════════
// Source: runtime/wasm/host.ts
// ════════════════════════════════════════════════════════════

// runtime/wasm/host.ts — Raw host function bindings (INTERNAL).
// Developers never import this file directly. Namespace files use these.

@external("host", "send")
export declare function _send(topic: string, payload: string): void

@external("host", "askAsync")
export declare function _askAsync(topic: string, payload: string, callbackFuncName: string): void

@external("host", "on")
export declare function _on(topic: string, funcName: string): void

@external("host", "tool")
export declare function _tool(name: string, funcName: string): void

@external("host", "reply")
export declare function _reply(payload: string): void

@external("host", "log")
export declare function _log(message: string, level: i32): void

@external("host", "get_state")
export declare function _getState(key: string): string

@external("host", "set_state")
export declare function _setState(key: string, value: string): void

@external("host", "has_state")
export declare function _hasState(key: string): i32

@external("host", "set_mode")
export declare function _setMode(mode: string): void

// ════════════════════════════════════════════════════════════
// Source: runtime/wasm/json.ts
// ════════════════════════════════════════════════════════════

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

// ════════════════════════════════════════════════════════════
// Source: runtime/wasm/types.ts
// ════════════════════════════════════════════════════════════

// runtime/wasm/types.ts — Base types for the brainkit WASM module.

/** BusMsg interface — all typed messages must implement this. */
export interface BusMsg {
    topic(): string
    toJSON(): string
}

// ════════════════════════════════════════════════════════════
// Source: runtime/wasm/log.ts
// ════════════════════════════════════════════════════════════

// runtime/wasm/log.ts — Logging functions.


/** Log at info level */
export function log(message: string): void {
    _log(message, 1)
}

/** Log at specific level: 0=debug, 1=info, 2=warn, 3=error */
export function logAt(message: string, level: i32): void {
    _log(message, level)
}

/** Log at debug level */
export function debug(message: string): void {
    _log(message, 0)
}

/** Log at warn level */
export function warn(message: string): void {
    _log(message, 2)
}

/** Log at error level */
export function error(message: string): void {
    _log(message, 3)
}

// ════════════════════════════════════════════════════════════
// Source: runtime/wasm/state.ts
// ════════════════════════════════════════════════════════════

// runtime/wasm/state.ts — State management functions.


/** Get a value from per-execution state. Returns "" if not found. */
export function getState(key: string): string {
    return _getState(key)
}

/** Set a value in per-execution state. */
export function setState(key: string, value: string): void {
    _setState(key, value)
}

/** Check if a key exists in state. */
export function hasState(key: string): bool {
    return _hasState(key) != 0
}

// ════════════════════════════════════════════════════════════
// Source: runtime/wasm/shard.ts
// ════════════════════════════════════════════════════════════

// runtime/wasm/shard.ts — Shard registration functions (init phase only).


/** Subscribe to a topic pattern. Handler function is called when messages match. Init only. */
export function on(topic: string, handlerFuncName: string): void {
    _on(topic, handlerFuncName)
}

/** Register a tool this shard provides. Init only. */
export function tool(name: string, handlerFuncName: string): void {
    _tool(name, handlerFuncName)
}

/** Reply to the current inbound message. */
export function reply(payload: string): void {
    _reply(payload)
}

/** Set shard execution mode: "stateless" or "persistent". Init only. */
export function setMode(mode: string): void {
    _setMode(mode)
}

// ════════════════════════════════════════════════════════════
// Source: runtime/wasm/bus.ts
// ════════════════════════════════════════════════════════════

// runtime/wasm/bus.ts — Raw bus primitives for custom topics/events.


export namespace bus {
    /** Send a typed custom event (fire-and-forget). */
    export function send(msg: BusMsg): void {
        _send(msg.topic(), msg.toJSON())
    }

    /** Async ask with a typed custom message. Callback function called when response arrives. */
    export function askAsync(msg: BusMsg, callbackFuncName: string): void {
        _askAsync(msg.topic(), msg.toJSON(), callbackFuncName)
    }

    /** Send raw topic + payload (fire-and-forget). For advanced use. */
    export function sendRaw(topic: string, payload: string): void {
        _send(topic, payload)
    }

    /** Async ask with raw topic + payload. For advanced use. */
    export function askAsyncRaw(topic: string, payload: string, callbackFuncName: string): void {
        _askAsync(topic, payload, callbackFuncName)
    }
}

// ════════════════════════════════════════════════════════════
// Source: runtime/wasm/ai.ts
// ════════════════════════════════════════════════════════════

// runtime/wasm/ai.ts — AI domain typed messages + namespace functions.


// ── Typed Messages ──

export class AiGenerateMsg {
    model: string
    prompt: string

    constructor(model: string, prompt: string) {
        this.model = model
        this.prompt = prompt
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("model", this.model)
        obj.setString("prompt", this.prompt)
        return obj.toString()
    }
}

export class AiEmbedMsg {
    model: string
    value: string

    constructor(model: string, value: string) {
        this.model = model
        this.value = value
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("model", this.model)
        obj.setString("value", this.value)
        return obj.toString()
    }
}

// ── Typed Responses ──

export class AiGenerateResp {
    text: string
    promptTokens: i32
    completionTokens: i32

    constructor() {
        this.text = ""
        this.promptTokens = 0
        this.completionTokens = 0
    }

    static parse(json: string): AiGenerateResp {
        let resp = new AiGenerateResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.toObject()
            resp.text = obj.getString("text")
            let usage = obj.getObject("usage")
            if (usage != null) {
                resp.promptTokens = usage.getInteger("promptTokens") as i32
                resp.completionTokens = usage.getInteger("completionTokens") as i32
            }
        }
        return resp
    }
}

export class AiEmbedResp {
    embedding: string // JSON array as string — parse externally

    constructor() {
        this.embedding = "[]"
    }

    static parse(json: string): AiEmbedResp {
        let resp = new AiEmbedResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let embVal = val.toObject().get("embedding")
            if (embVal != null) {
                resp.embedding = embVal.toString()
            }
        }
        return resp
    }
}

// ── Namespace Functions ──

export namespace ai {
    export function generate(msg: AiGenerateMsg, callback: string): void {
        _askAsync("ai.generate", msg.toJSON(), callback)
    }

    export function embed(msg: AiEmbedMsg, callback: string): void {
        _askAsync("ai.embed", msg.toJSON(), callback)
    }
}

// ════════════════════════════════════════════════════════════
// Source: runtime/wasm/tools.ts
// ════════════════════════════════════════════════════════════

// runtime/wasm/tools.ts — Tools domain typed messages + namespace functions.


export class ToolCallMsg {
    name: string
    input: string // JSON

    constructor(name: string, input: string) {
        this.name = name
        this.input = input
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        obj.setRaw("input", this.input)
        return obj.toString()
    }
}

export namespace tools {
    export function call(msg: ToolCallMsg, callback: string): void {
        _askAsync("tools.call", msg.toJSON(), callback)
    }
}

// ════════════════════════════════════════════════════════════
// Source: runtime/wasm/agents.ts
// ════════════════════════════════════════════════════════════

// runtime/wasm/agents.ts — Agents domain typed messages + namespace functions.


export class AgentRequestMsg {
    name: string
    prompt: string

    constructor(name: string, prompt: string) {
        this.name = name
        this.prompt = prompt
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        obj.setString("prompt", this.prompt)
        return obj.toString()
    }
}

export class AgentRequestResp {
    text: string

    constructor() { this.text = "" }

    static parse(json: string): AgentRequestResp {
        let resp = new AgentRequestResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            resp.text = val.toObject().getString("text")
        }
        return resp
    }
}

export class AgentMessageMsg {
    target: string
    payload: string // JSON

    constructor(target: string, payload: string) {
        this.target = target
        this.payload = payload
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("target", this.target)
        obj.setRaw("payload", this.payload)
        return obj.toString()
    }
}

export namespace agents {
    export function request(msg: AgentRequestMsg, callback: string): void {
        _askAsync("agents.request", msg.toJSON(), callback)
    }

    export function message(msg: AgentMessageMsg): void {
        _send("agents.message", msg.toJSON())
    }
}

// ════════════════════════════════════════════════════════════
// Source: runtime/wasm/wasm_ops.ts
// ════════════════════════════════════════════════════════════

// runtime/wasm/wasm_ops.ts — WASM operations typed messages + namespace functions.


export class WasmCompileMsg {
    source: string
    name: string

    constructor(source: string, name: string = "") {
        this.source = source
        this.name = name
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("source", this.source)
        if (this.name.length > 0) {
            let opts = new JSONObject()
            opts.setString("name", this.name)
            obj.set("options", opts)
        }
        return obj.toString()
    }
}

export class WasmDeployMsg {
    name: string

    constructor(name: string) {
        this.name = name
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        return obj.toString()
    }
}

export namespace wasm_ops {
    export function compile(msg: WasmCompileMsg, callback: string): void {
        _askAsync("wasm.compile", msg.toJSON(), callback)
    }

    export function deploy(msg: WasmDeployMsg, callback: string): void {
        _askAsync("wasm.deploy", msg.toJSON(), callback)
    }
}

// ════════════════════════════════════════════════════════════
// Source: runtime/wasm/memory.ts
// ════════════════════════════════════════════════════════════

// runtime/wasm/memory.ts — Memory domain typed messages + namespace functions.


export class MemoryRecallMsg {
    threadId: string
    query: string

    constructor(threadId: string, query: string) {
        this.threadId = threadId
        this.query = query
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("threadId", this.threadId)
        obj.setString("query", this.query)
        return obj.toString()
    }
}

export class MemorySaveMsg {
    threadId: string
    messagesJSON: string

    constructor(threadId: string, messagesJSON: string) {
        this.threadId = threadId
        this.messagesJSON = messagesJSON
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("threadId", this.threadId)
        obj.setRaw("messages", this.messagesJSON)
        return obj.toString()
    }
}

export namespace memory {
    export function recall(msg: MemoryRecallMsg, callback: string): void {
        _askAsync("memory.recall", msg.toJSON(), callback)
    }

    export function save(msg: MemorySaveMsg, callback: string): void {
        _askAsync("memory.save", msg.toJSON(), callback)
    }
}

// ════════════════════════════════════════════════════════════
// Source: runtime/wasm/workflows.ts
// ════════════════════════════════════════════════════════════

// runtime/wasm/workflows.ts — Workflows domain typed messages + namespace functions.


export class WorkflowRunMsg {
    name: string
    inputJSON: string

    constructor(name: string, inputJSON: string) {
        this.name = name
        this.inputJSON = inputJSON
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        obj.setRaw("input", this.inputJSON)
        return obj.toString()
    }
}

export namespace workflows {
    export function run(msg: WorkflowRunMsg, callback: string): void {
        _askAsync("workflows.run", msg.toJSON(), callback)
    }
}

// ════════════════════════════════════════════════════════════
// Source: runtime/wasm/vectors.ts
// ════════════════════════════════════════════════════════════

// runtime/wasm/vectors.ts — Vectors domain typed messages + namespace functions.


export class VectorQueryMsg {
    index: string
    embeddingJSON: string
    topK: i32

    constructor(index: string, embeddingJSON: string, topK: i32) {
        this.index = index
        this.embeddingJSON = embeddingJSON
        this.topK = topK
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("index", this.index)
        obj.setRaw("embedding", this.embeddingJSON)
        obj.setInteger("topK", this.topK as i64)
        return obj.toString()
    }
}

export class VectorUpsertMsg {
    index: string
    vectorsJSON: string

    constructor(index: string, vectorsJSON: string) {
        this.index = index
        this.vectorsJSON = vectorsJSON
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("index", this.index)
        obj.setRaw("vectors", this.vectorsJSON)
        return obj.toString()
    }
}

export namespace vectors {
    export function query(msg: VectorQueryMsg, callback: string): void {
        _askAsync("vectors.query", msg.toJSON(), callback)
    }

    export function upsert(msg: VectorUpsertMsg, callback: string): void {
        _askAsync("vectors.upsert", msg.toJSON(), callback)
    }
}

// ════════════════════════════════════════════════════════════
// Source: runtime/wasm/fs.ts
// ════════════════════════════════════════════════════════════

// runtime/wasm/fs.ts — Filesystem domain typed messages + namespace functions.


export class FsReadMsg {
    path: string

    constructor(path: string) {
        this.path = path
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("path", this.path)
        return obj.toString()
    }
}

export class FsWriteMsg {
    path: string
    data: string

    constructor(path: string, data: string) {
        this.path = path
        this.data = data
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("path", this.path)
        obj.setString("data", this.data)
        return obj.toString()
    }
}

export class FsListMsg {
    path: string
    pattern: string

    constructor(path: string, pattern: string = "") {
        this.path = path
        this.pattern = pattern
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("path", this.path)
        if (this.pattern.length > 0) {
            obj.setString("pattern", this.pattern)
        }
        return obj.toString()
    }
}

export namespace fs_ops {
    export function read(msg: FsReadMsg, callback: string): void {
        _askAsync("fs.read", msg.toJSON(), callback)
    }

    export function write(msg: FsWriteMsg, callback: string): void {
        _askAsync("fs.write", msg.toJSON(), callback)
    }

    export function list(msg: FsListMsg, callback: string): void {
        _askAsync("fs.list", msg.toJSON(), callback)
    }
}

// ════════════════════════════════════════════════════════════
// Source: runtime/wasm/index.ts
// ════════════════════════════════════════════════════════════

// runtime/wasm/index.ts — Re-exports everything as the "brainkit" module.
// This file is concatenated LAST in the bundle.
// All exports from the namespace files are already in scope due to concatenation.

// Domain namespaces: ai, tools, agents, wasm_ops, memory, workflows, vectors, fs_ops, bus
// Shard functions: on, tool, reply, setMode, setModeKey
// State functions: getState, setState, hasState
// Log functions: log, logAt, debug, warn, error
// JSON library: JSONValue, JSONObject, JSONArray

