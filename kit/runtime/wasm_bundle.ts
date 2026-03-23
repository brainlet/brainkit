// AUTO-GENERATED — do not edit. Run scripts/bundle_wasm.sh to regenerate.
// Source files: 19 files from kit/runtime/wasm/

// ════════════════════════════════════════════════════════════
// Source: kit/runtime/wasm/host.ts
// ════════════════════════════════════════════════════════════

// runtime/wasm/host.ts — Raw host function bindings (INTERNAL).
// Developers never import this file directly. Namespace files use these.

@external("host", "send")
export declare function _send(topic: string, payload: string): void

@external("host", "invokeAsync")
export declare function _invokeAsync(topic: string, payload: string, callbackFuncName: string): void

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
// Source: kit/runtime/wasm/json.ts
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
// Source: kit/runtime/wasm/types.ts
// ════════════════════════════════════════════════════════════

// runtime/wasm/types.ts — Base types for the brainkit WASM module.

/** BusMsg interface — all typed messages must implement this. */
export interface BusMsg {
    topic(): string
    toJSON(): string
}

// ════════════════════════════════════════════════════════════
// Source: kit/runtime/wasm/log.ts
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
// Source: kit/runtime/wasm/state.ts
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
// Source: kit/runtime/wasm/shard.ts
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
// Source: kit/runtime/wasm/generated/ai.ts
// ════════════════════════════════════════════════════════════

// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: ai

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

export class AiEmbedManyMsg {
    model: string
    values: string

    constructor(model: string, values: string) {
        this.model = model
        this.values = values
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("model", this.model)
        if (this.values.length > 0) obj.set("values", JSONValue.parse(this.values))
        return obj.toString()
    }
}

export class AiGenerateMsg {
    model: string
    prompt: string
    messages: string
    tools: string
    schema: string

    constructor(model: string, prompt: string, messages: string, tools: string, schema: string) {
        this.model = model
        this.prompt = prompt
        this.messages = messages
        this.tools = tools
        this.schema = schema
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("model", this.model)
        obj.setString("prompt", this.prompt)
        if (this.messages.length > 0) obj.set("messages", JSONValue.parse(this.messages))
        if (this.tools.length > 0) obj.set("tools", JSONValue.parse(this.tools))
        if (this.schema.length > 0) obj.set("schema", JSONValue.parse(this.schema))
        return obj.toString()
    }
}

export class AiGenerateObjectMsg {
    model: string
    prompt: string
    schema: string

    constructor(model: string, prompt: string, schema: string) {
        this.model = model
        this.prompt = prompt
        this.schema = schema
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("model", this.model)
        obj.setString("prompt", this.prompt)
        if (this.schema.length > 0) obj.set("schema", JSONValue.parse(this.schema))
        return obj.toString()
    }
}

export class AiEmbedResp {
    embedding: string
    error: string

    constructor() {
        this.embedding = ""
        this.error = ""
    }

    static parse(json: string): AiEmbedResp {
        let resp = new AiEmbedResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("embedding")) resp.embedding = obj.get("embedding").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class AiEmbedManyResp {
    embeddings: string
    error: string

    constructor() {
        this.embeddings = ""
        this.error = ""
    }

    static parse(json: string): AiEmbedManyResp {
        let resp = new AiEmbedManyResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("embeddings")) resp.embeddings = obj.get("embeddings").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class AiGenerateResp {
    text: string
    toolCalls: string
    usage: string
    error: string

    constructor() {
        this.text = ""
        this.toolCalls = ""
        this.usage = ""
        this.error = ""
    }

    static parse(json: string): AiGenerateResp {
        let resp = new AiGenerateResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("text")) resp.text = obj.getString("text")
            if (obj.has("toolCalls")) resp.toolCalls = obj.get("toolCalls").toString()
            if (obj.has("usage")) resp.usage = obj.get("usage").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class AiGenerateObjectResp {
    object: string
    error: string

    constructor() {
        this.object = ""
        this.error = ""
    }

    static parse(json: string): AiGenerateObjectResp {
        let resp = new AiGenerateObjectResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("object")) resp.object = obj.get("object").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export namespace ai {
    export function embed(msg: AiEmbedMsg, callback: string): void {
        _invokeAsync("ai.embed", msg.toJSON(), callback)
    }

    export function embedMany(msg: AiEmbedManyMsg, callback: string): void {
        _invokeAsync("ai.embedMany", msg.toJSON(), callback)
    }

    export function generate(msg: AiGenerateMsg, callback: string): void {
        _invokeAsync("ai.generate", msg.toJSON(), callback)
    }

    export function generateObject(msg: AiGenerateObjectMsg, callback: string): void {
        _invokeAsync("ai.generateObject", msg.toJSON(), callback)
    }

}

// Events
export class AiStreamMsg {
    model: string
    prompt: string
    messages: string
    streamTo: string

    constructor(model: string, prompt: string, messages: string, streamTo: string) {
        this.model = model
        this.prompt = prompt
        this.messages = messages
        this.streamTo = streamTo
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("model", this.model)
        obj.setString("prompt", this.prompt)
        if (this.messages.length > 0) obj.set("messages", JSONValue.parse(this.messages))
        obj.setString("streamTo", this.streamTo)
        return obj.toString()
    }
}


// ════════════════════════════════════════════════════════════
// Source: kit/runtime/wasm/generated/agents.ts
// ════════════════════════════════════════════════════════════

// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: agents

export class AgentDiscoverMsg {
    capability: string
    model: string
    status: string

    constructor(capability: string, model: string, status: string) {
        this.capability = capability
        this.model = model
        this.status = status
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("capability", this.capability)
        obj.setString("model", this.model)
        obj.setString("status", this.status)
        return obj.toString()
    }
}

export class AgentGetStatusMsg {
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

export class AgentListMsg {
    filter: string

    constructor(filter: string) {
        this.filter = filter
    }

    toJSON(): string {
        let obj = new JSONObject()
        if (this.filter.length > 0) obj.set("filter", JSONValue.parse(this.filter))
        return obj.toString()
    }
}

export class AgentMessageMsg {
    target: string
    payload: string

    constructor(target: string, payload: string) {
        this.target = target
        this.payload = payload
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("target", this.target)
        if (this.payload.length > 0) obj.set("payload", JSONValue.parse(this.payload))
        return obj.toString()
    }
}

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

export class AgentSetStatusMsg {
    name: string
    status: string

    constructor(name: string, status: string) {
        this.name = name
        this.status = status
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        obj.setString("status", this.status)
        return obj.toString()
    }
}

export class AgentDiscoverResp {
    agents: string
    error: string

    constructor() {
        this.agents = ""
        this.error = ""
    }

    static parse(json: string): AgentDiscoverResp {
        let resp = new AgentDiscoverResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("agents")) resp.agents = obj.get("agents").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class AgentGetStatusResp {
    name: string
    status: string
    error: string

    constructor() {
        this.name = ""
        this.status = ""
        this.error = ""
    }

    static parse(json: string): AgentGetStatusResp {
        let resp = new AgentGetStatusResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("name")) resp.name = obj.getString("name")
            if (obj.has("status")) resp.status = obj.getString("status")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class AgentListResp {
    agents: string
    error: string

    constructor() {
        this.agents = ""
        this.error = ""
    }

    static parse(json: string): AgentListResp {
        let resp = new AgentListResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("agents")) resp.agents = obj.get("agents").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class AgentMessageResp {
    delivered: bool
    error: string

    constructor() {
        this.delivered = false
        this.error = ""
    }

    static parse(json: string): AgentMessageResp {
        let resp = new AgentMessageResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("delivered")) resp.delivered = obj.getBool("delivered")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class AgentRequestResp {
    text: string
    error: string

    constructor() {
        this.text = ""
        this.error = ""
    }

    static parse(json: string): AgentRequestResp {
        let resp = new AgentRequestResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("text")) resp.text = obj.getString("text")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class AgentSetStatusResp {
    ok: bool
    error: string

    constructor() {
        this.ok = false
        this.error = ""
    }

    static parse(json: string): AgentSetStatusResp {
        let resp = new AgentSetStatusResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("ok")) resp.ok = obj.getBool("ok")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export namespace agents {
    export function discover(msg: AgentDiscoverMsg, callback: string): void {
        _invokeAsync("agents.discover", msg.toJSON(), callback)
    }

    export function getStatus(msg: AgentGetStatusMsg, callback: string): void {
        _invokeAsync("agents.get-status", msg.toJSON(), callback)
    }

    export function list(msg: AgentListMsg, callback: string): void {
        _invokeAsync("agents.list", msg.toJSON(), callback)
    }

    export function message(msg: AgentMessageMsg, callback: string): void {
        _invokeAsync("agents.message", msg.toJSON(), callback)
    }

    export function request(msg: AgentRequestMsg, callback: string): void {
        _invokeAsync("agents.request", msg.toJSON(), callback)
    }

    export function setStatus(msg: AgentSetStatusMsg, callback: string): void {
        _invokeAsync("agents.set-status", msg.toJSON(), callback)
    }

}

// Events
export class AgentStreamMsg {
    name: string
    prompt: string
    streamTo: string

    constructor(name: string, prompt: string, streamTo: string) {
        this.name = name
        this.prompt = prompt
        this.streamTo = streamTo
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        obj.setString("prompt", this.prompt)
        obj.setString("streamTo", this.streamTo)
        return obj.toString()
    }
}


// ════════════════════════════════════════════════════════════
// Source: kit/runtime/wasm/generated/fs.ts
// ════════════════════════════════════════════════════════════

// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: fs

export class FsDeleteMsg {
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

export class FsListMsg {
    path: string
    pattern: string

    constructor(path: string, pattern: string) {
        this.path = path
        this.pattern = pattern
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("path", this.path)
        obj.setString("pattern", this.pattern)
        return obj.toString()
    }
}

export class FsMkdirMsg {
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

export class FsStatMsg {
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

export class FsDeleteResp {
    ok: bool
    error: string

    constructor() {
        this.ok = false
        this.error = ""
    }

    static parse(json: string): FsDeleteResp {
        let resp = new FsDeleteResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("ok")) resp.ok = obj.getBool("ok")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class FsListResp {
    files: string
    error: string

    constructor() {
        this.files = ""
        this.error = ""
    }

    static parse(json: string): FsListResp {
        let resp = new FsListResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("files")) resp.files = obj.get("files").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class FsMkdirResp {
    ok: bool
    error: string

    constructor() {
        this.ok = false
        this.error = ""
    }

    static parse(json: string): FsMkdirResp {
        let resp = new FsMkdirResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("ok")) resp.ok = obj.getBool("ok")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class FsReadResp {
    data: string
    error: string

    constructor() {
        this.data = ""
        this.error = ""
    }

    static parse(json: string): FsReadResp {
        let resp = new FsReadResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("data")) resp.data = obj.getString("data")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class FsStatResp {
    size: i32
    isDir: bool
    modTime: string
    error: string

    constructor() {
        this.size = 0
        this.isDir = false
        this.modTime = ""
        this.error = ""
    }

    static parse(json: string): FsStatResp {
        let resp = new FsStatResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("size")) resp.size = obj.getInt("size")
            if (obj.has("isDir")) resp.isDir = obj.getBool("isDir")
            if (obj.has("modTime")) resp.modTime = obj.getString("modTime")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class FsWriteResp {
    ok: bool
    error: string

    constructor() {
        this.ok = false
        this.error = ""
    }

    static parse(json: string): FsWriteResp {
        let resp = new FsWriteResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("ok")) resp.ok = obj.getBool("ok")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export namespace fs_ops {
    export function delete(msg: FsDeleteMsg, callback: string): void {
        _invokeAsync("fs.delete", msg.toJSON(), callback)
    }

    export function list(msg: FsListMsg, callback: string): void {
        _invokeAsync("fs.list", msg.toJSON(), callback)
    }

    export function mkdir(msg: FsMkdirMsg, callback: string): void {
        _invokeAsync("fs.mkdir", msg.toJSON(), callback)
    }

    export function read(msg: FsReadMsg, callback: string): void {
        _invokeAsync("fs.read", msg.toJSON(), callback)
    }

    export function stat(msg: FsStatMsg, callback: string): void {
        _invokeAsync("fs.stat", msg.toJSON(), callback)
    }

    export function write(msg: FsWriteMsg, callback: string): void {
        _invokeAsync("fs.write", msg.toJSON(), callback)
    }

}

// ════════════════════════════════════════════════════════════
// Source: kit/runtime/wasm/generated/kit.ts
// ════════════════════════════════════════════════════════════

// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: kit

export class KitDeployMsg {
    source: string
    code: string

    constructor(source: string, code: string) {
        this.source = source
        this.code = code
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("source", this.source)
        obj.setString("code", this.code)
        return obj.toString()
    }
}

export class KitListMsg {

    toJSON(): string {
        let obj = new JSONObject()
        return obj.toString()
    }
}

export class KitRedeployMsg {
    source: string
    code: string

    constructor(source: string, code: string) {
        this.source = source
        this.code = code
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("source", this.source)
        obj.setString("code", this.code)
        return obj.toString()
    }
}

export class KitTeardownMsg {
    source: string

    constructor(source: string) {
        this.source = source
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("source", this.source)
        return obj.toString()
    }
}

export class KitDeployResp {
    deployed: bool
    resources: string
    error: string

    constructor() {
        this.deployed = false
        this.resources = ""
        this.error = ""
    }

    static parse(json: string): KitDeployResp {
        let resp = new KitDeployResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("deployed")) resp.deployed = obj.getBool("deployed")
            if (obj.has("resources")) resp.resources = obj.get("resources").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class KitListResp {
    deployments: string
    error: string

    constructor() {
        this.deployments = ""
        this.error = ""
    }

    static parse(json: string): KitListResp {
        let resp = new KitListResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("deployments")) resp.deployments = obj.get("deployments").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class KitRedeployResp {
    deployed: bool
    resources: string
    error: string

    constructor() {
        this.deployed = false
        this.resources = ""
        this.error = ""
    }

    static parse(json: string): KitRedeployResp {
        let resp = new KitRedeployResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("deployed")) resp.deployed = obj.getBool("deployed")
            if (obj.has("resources")) resp.resources = obj.get("resources").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class KitTeardownResp {
    removed: i32
    error: string

    constructor() {
        this.removed = 0
        this.error = ""
    }

    static parse(json: string): KitTeardownResp {
        let resp = new KitTeardownResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("removed")) resp.removed = obj.getInt("removed")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export namespace kit {
    export function deploy(msg: KitDeployMsg, callback: string): void {
        _invokeAsync("kit.deploy", msg.toJSON(), callback)
    }

    export function list(msg: KitListMsg, callback: string): void {
        _invokeAsync("kit.list", msg.toJSON(), callback)
    }

    export function redeploy(msg: KitRedeployMsg, callback: string): void {
        _invokeAsync("kit.redeploy", msg.toJSON(), callback)
    }

    export function teardown(msg: KitTeardownMsg, callback: string): void {
        _invokeAsync("kit.teardown", msg.toJSON(), callback)
    }

}

// Events
export class KitDeployedEvent {
    source: string
    resources: string

    constructor(source: string, resources: string) {
        this.source = source
        this.resources = resources
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("source", this.source)
        if (this.resources.length > 0) obj.set("resources", JSONValue.parse(this.resources))
        return obj.toString()
    }
}

export class KitTeardownedEvent {
    source: string
    removed: i32

    constructor(source: string, removed: i32) {
        this.source = source
        this.removed = removed
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("source", this.source)
        obj.setInt("removed", this.removed)
        return obj.toString()
    }
}


// ════════════════════════════════════════════════════════════
// Source: kit/runtime/wasm/generated/mcp.ts
// ════════════════════════════════════════════════════════════

// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: mcp

export class McpCallToolMsg {
    server: string
    tool: string
    args: string

    constructor(server: string, tool: string, args: string) {
        this.server = server
        this.tool = tool
        this.args = args
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("server", this.server)
        obj.setString("tool", this.tool)
        if (this.args.length > 0) obj.set("args", JSONValue.parse(this.args))
        return obj.toString()
    }
}

export class McpListToolsMsg {
    server: string

    constructor(server: string) {
        this.server = server
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("server", this.server)
        return obj.toString()
    }
}

export class McpCallToolResp {
    result: string
    error: string

    constructor() {
        this.result = ""
        this.error = ""
    }

    static parse(json: string): McpCallToolResp {
        let resp = new McpCallToolResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("result")) resp.result = obj.get("result").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class McpListToolsResp {
    tools: string
    error: string

    constructor() {
        this.tools = ""
        this.error = ""
    }

    static parse(json: string): McpListToolsResp {
        let resp = new McpListToolsResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("tools")) resp.tools = obj.get("tools").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export namespace mcp {
    export function callTool(msg: McpCallToolMsg, callback: string): void {
        _invokeAsync("mcp.callTool", msg.toJSON(), callback)
    }

    export function listTools(msg: McpListToolsMsg, callback: string): void {
        _invokeAsync("mcp.listTools", msg.toJSON(), callback)
    }

}

// ════════════════════════════════════════════════════════════
// Source: kit/runtime/wasm/generated/memory.ts
// ════════════════════════════════════════════════════════════

// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: memory

export class MemoryCreateThreadMsg {
    opts: string

    constructor(opts: string) {
        this.opts = opts
    }

    toJSON(): string {
        let obj = new JSONObject()
        if (this.opts.length > 0) obj.set("opts", JSONValue.parse(this.opts))
        return obj.toString()
    }
}

export class MemoryDeleteThreadMsg {
    threadId: string

    constructor(threadId: string) {
        this.threadId = threadId
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("threadId", this.threadId)
        return obj.toString()
    }
}

export class MemoryGetThreadMsg {
    threadId: string

    constructor(threadId: string) {
        this.threadId = threadId
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("threadId", this.threadId)
        return obj.toString()
    }
}

export class MemoryListThreadsMsg {
    filter: string

    constructor(filter: string) {
        this.filter = filter
    }

    toJSON(): string {
        let obj = new JSONObject()
        if (this.filter.length > 0) obj.set("filter", JSONValue.parse(this.filter))
        return obj.toString()
    }
}

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
    messages: string

    constructor(threadId: string, messages: string) {
        this.threadId = threadId
        this.messages = messages
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("threadId", this.threadId)
        if (this.messages.length > 0) obj.set("messages", JSONValue.parse(this.messages))
        return obj.toString()
    }
}

export class MemoryCreateThreadResp {
    threadId: string
    error: string

    constructor() {
        this.threadId = ""
        this.error = ""
    }

    static parse(json: string): MemoryCreateThreadResp {
        let resp = new MemoryCreateThreadResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("threadId")) resp.threadId = obj.getString("threadId")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class MemoryDeleteThreadResp {
    ok: bool
    error: string

    constructor() {
        this.ok = false
        this.error = ""
    }

    static parse(json: string): MemoryDeleteThreadResp {
        let resp = new MemoryDeleteThreadResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("ok")) resp.ok = obj.getBool("ok")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class MemoryGetThreadResp {
    thread: string
    error: string

    constructor() {
        this.thread = ""
        this.error = ""
    }

    static parse(json: string): MemoryGetThreadResp {
        let resp = new MemoryGetThreadResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("thread")) resp.thread = obj.get("thread").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class MemoryListThreadsResp {
    threads: string
    error: string

    constructor() {
        this.threads = ""
        this.error = ""
    }

    static parse(json: string): MemoryListThreadsResp {
        let resp = new MemoryListThreadsResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("threads")) resp.threads = obj.get("threads").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class MemoryRecallResp {
    messages: string
    error: string

    constructor() {
        this.messages = ""
        this.error = ""
    }

    static parse(json: string): MemoryRecallResp {
        let resp = new MemoryRecallResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("messages")) resp.messages = obj.get("messages").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class MemorySaveResp {
    ok: bool
    error: string

    constructor() {
        this.ok = false
        this.error = ""
    }

    static parse(json: string): MemorySaveResp {
        let resp = new MemorySaveResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("ok")) resp.ok = obj.getBool("ok")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export namespace memory {
    export function createThread(msg: MemoryCreateThreadMsg, callback: string): void {
        _invokeAsync("memory.createThread", msg.toJSON(), callback)
    }

    export function deleteThread(msg: MemoryDeleteThreadMsg, callback: string): void {
        _invokeAsync("memory.deleteThread", msg.toJSON(), callback)
    }

    export function getThread(msg: MemoryGetThreadMsg, callback: string): void {
        _invokeAsync("memory.getThread", msg.toJSON(), callback)
    }

    export function listThreads(msg: MemoryListThreadsMsg, callback: string): void {
        _invokeAsync("memory.listThreads", msg.toJSON(), callback)
    }

    export function recall(msg: MemoryRecallMsg, callback: string): void {
        _invokeAsync("memory.recall", msg.toJSON(), callback)
    }

    export function save(msg: MemorySaveMsg, callback: string): void {
        _invokeAsync("memory.save", msg.toJSON(), callback)
    }

}

// ════════════════════════════════════════════════════════════
// Source: kit/runtime/wasm/generated/plugin.ts
// ════════════════════════════════════════════════════════════

// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: plugin

export class PluginManifestMsg {
    owner: string
    name: string
    version: string
    description: string
    tools: string
    subscriptions: string
    events: string

    constructor(owner: string, name: string, version: string, description: string, tools: string, subscriptions: string, events: string) {
        this.owner = owner
        this.name = name
        this.version = version
        this.description = description
        this.tools = tools
        this.subscriptions = subscriptions
        this.events = events
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("owner", this.owner)
        obj.setString("name", this.name)
        obj.setString("version", this.version)
        obj.setString("description", this.description)
        if (this.tools.length > 0) obj.set("tools", JSONValue.parse(this.tools))
        if (this.subscriptions.length > 0) obj.set("subscriptions", JSONValue.parse(this.subscriptions))
        if (this.events.length > 0) obj.set("events", JSONValue.parse(this.events))
        return obj.toString()
    }
}

export class PluginStateGetMsg {
    key: string

    constructor(key: string) {
        this.key = key
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("key", this.key)
        return obj.toString()
    }
}

export class PluginStateSetMsg {
    key: string
    value: string

    constructor(key: string, value: string) {
        this.key = key
        this.value = value
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("key", this.key)
        obj.setString("value", this.value)
        return obj.toString()
    }
}

export class PluginManifestResp {
    registered: bool
    error: string

    constructor() {
        this.registered = false
        this.error = ""
    }

    static parse(json: string): PluginManifestResp {
        let resp = new PluginManifestResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("registered")) resp.registered = obj.getBool("registered")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class PluginStateGetResp {
    value: string
    error: string

    constructor() {
        this.value = ""
        this.error = ""
    }

    static parse(json: string): PluginStateGetResp {
        let resp = new PluginStateGetResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("value")) resp.value = obj.getString("value")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class PluginStateSetResp {
    ok: bool
    error: string

    constructor() {
        this.ok = false
        this.error = ""
    }

    static parse(json: string): PluginStateSetResp {
        let resp = new PluginStateSetResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("ok")) resp.ok = obj.getBool("ok")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export namespace plugin {
    export function manifest(msg: PluginManifestMsg, callback: string): void {
        _invokeAsync("plugin.manifest", msg.toJSON(), callback)
    }

    export function stateGet(msg: PluginStateGetMsg, callback: string): void {
        _invokeAsync("plugin.state.get", msg.toJSON(), callback)
    }

    export function stateSet(msg: PluginStateSetMsg, callback: string): void {
        _invokeAsync("plugin.state.set", msg.toJSON(), callback)
    }

}

// Events
export class PluginRegisteredEvent {
    owner: string
    name: string
    version: string
    tools: i32

    constructor(owner: string, name: string, version: string, tools: i32) {
        this.owner = owner
        this.name = name
        this.version = version
        this.tools = tools
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("owner", this.owner)
        obj.setString("name", this.name)
        obj.setString("version", this.version)
        obj.setInt("tools", this.tools)
        return obj.toString()
    }
}


// ════════════════════════════════════════════════════════════
// Source: kit/runtime/wasm/generated/stream.ts
// ════════════════════════════════════════════════════════════

// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: stream

export namespace stream {
}

// Events
export class StreamChunk {
    streamId: string
    seq: i32
    delta: string
    done: bool
    final: string

    constructor(streamId: string, seq: i32, delta: string, done: bool, final: string) {
        this.streamId = streamId
        this.seq = seq
        this.delta = delta
        this.done = done
        this.final = final
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("streamId", this.streamId)
        obj.setInt("seq", this.seq)
        obj.setString("delta", this.delta)
        obj.setBool("done", this.done)
        if (this.final.length > 0) obj.set("final", JSONValue.parse(this.final))
        return obj.toString()
    }
}


// ════════════════════════════════════════════════════════════
// Source: kit/runtime/wasm/generated/tools.ts
// ════════════════════════════════════════════════════════════

// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: tools

export class ToolCallMsg {
    name: string
    input: string

    constructor(name: string, input: string) {
        this.name = name
        this.input = input
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        if (this.input.length > 0) obj.set("input", JSONValue.parse(this.input))
        return obj.toString()
    }
}

export class ToolListMsg {
    namespace: string

    constructor(namespace: string) {
        this.namespace = namespace
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("namespace", this.namespace)
        return obj.toString()
    }
}

export class ToolResolveMsg {
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

export class ToolCallResp {
    result: string
    error: string

    constructor() {
        this.result = ""
        this.error = ""
    }

    static parse(json: string): ToolCallResp {
        let resp = new ToolCallResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("result")) resp.result = obj.get("result").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class ToolListResp {
    tools: string
    error: string

    constructor() {
        this.tools = ""
        this.error = ""
    }

    static parse(json: string): ToolListResp {
        let resp = new ToolListResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("tools")) resp.tools = obj.get("tools").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class ToolResolveResp {
    name: string
    shortName: string
    description: string
    inputSchema: string
    error: string

    constructor() {
        this.name = ""
        this.shortName = ""
        this.description = ""
        this.inputSchema = ""
        this.error = ""
    }

    static parse(json: string): ToolResolveResp {
        let resp = new ToolResolveResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("name")) resp.name = obj.getString("name")
            if (obj.has("shortName")) resp.shortName = obj.getString("shortName")
            if (obj.has("description")) resp.description = obj.getString("description")
            if (obj.has("inputSchema")) resp.inputSchema = obj.get("inputSchema").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export namespace tools {
    export function call(msg: ToolCallMsg, callback: string): void {
        _invokeAsync("tools.call", msg.toJSON(), callback)
    }

    export function list(msg: ToolListMsg, callback: string): void {
        _invokeAsync("tools.list", msg.toJSON(), callback)
    }

    export function resolve(msg: ToolResolveMsg, callback: string): void {
        _invokeAsync("tools.resolve", msg.toJSON(), callback)
    }

}

// ════════════════════════════════════════════════════════════
// Source: kit/runtime/wasm/generated/vectors.ts
// ════════════════════════════════════════════════════════════

// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: vectors

export class VectorCreateIndexMsg {
    name: string
    dimension: i32
    metric: string

    constructor(name: string, dimension: i32, metric: string) {
        this.name = name
        this.dimension = dimension
        this.metric = metric
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        obj.setInt("dimension", this.dimension)
        obj.setString("metric", this.metric)
        return obj.toString()
    }
}

export class VectorDeleteIndexMsg {
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

export class VectorListIndexesMsg {

    toJSON(): string {
        let obj = new JSONObject()
        return obj.toString()
    }
}

export class VectorQueryMsg {
    index: string
    embedding: string
    topK: i32
    filter: string

    constructor(index: string, embedding: string, topK: i32, filter: string) {
        this.index = index
        this.embedding = embedding
        this.topK = topK
        this.filter = filter
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("index", this.index)
        if (this.embedding.length > 0) obj.set("embedding", JSONValue.parse(this.embedding))
        obj.setInt("topK", this.topK)
        if (this.filter.length > 0) obj.set("filter", JSONValue.parse(this.filter))
        return obj.toString()
    }
}

export class VectorUpsertMsg {
    index: string
    vectors: string

    constructor(index: string, vectors: string) {
        this.index = index
        this.vectors = vectors
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("index", this.index)
        if (this.vectors.length > 0) obj.set("vectors", JSONValue.parse(this.vectors))
        return obj.toString()
    }
}

export class VectorCreateIndexResp {
    ok: bool
    error: string

    constructor() {
        this.ok = false
        this.error = ""
    }

    static parse(json: string): VectorCreateIndexResp {
        let resp = new VectorCreateIndexResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("ok")) resp.ok = obj.getBool("ok")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class VectorDeleteIndexResp {
    ok: bool
    error: string

    constructor() {
        this.ok = false
        this.error = ""
    }

    static parse(json: string): VectorDeleteIndexResp {
        let resp = new VectorDeleteIndexResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("ok")) resp.ok = obj.getBool("ok")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class VectorListIndexesResp {
    indexes: string
    error: string

    constructor() {
        this.indexes = ""
        this.error = ""
    }

    static parse(json: string): VectorListIndexesResp {
        let resp = new VectorListIndexesResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("indexes")) resp.indexes = obj.get("indexes").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class VectorQueryResp {
    matches: string
    error: string

    constructor() {
        this.matches = ""
        this.error = ""
    }

    static parse(json: string): VectorQueryResp {
        let resp = new VectorQueryResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("matches")) resp.matches = obj.get("matches").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class VectorUpsertResp {
    ok: bool
    error: string

    constructor() {
        this.ok = false
        this.error = ""
    }

    static parse(json: string): VectorUpsertResp {
        let resp = new VectorUpsertResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("ok")) resp.ok = obj.getBool("ok")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export namespace vectors {
    export function createIndex(msg: VectorCreateIndexMsg, callback: string): void {
        _invokeAsync("vectors.createIndex", msg.toJSON(), callback)
    }

    export function deleteIndex(msg: VectorDeleteIndexMsg, callback: string): void {
        _invokeAsync("vectors.deleteIndex", msg.toJSON(), callback)
    }

    export function listIndexes(msg: VectorListIndexesMsg, callback: string): void {
        _invokeAsync("vectors.listIndexes", msg.toJSON(), callback)
    }

    export function query(msg: VectorQueryMsg, callback: string): void {
        _invokeAsync("vectors.query", msg.toJSON(), callback)
    }

    export function upsert(msg: VectorUpsertMsg, callback: string): void {
        _invokeAsync("vectors.upsert", msg.toJSON(), callback)
    }

}

// ════════════════════════════════════════════════════════════
// Source: kit/runtime/wasm/generated/wasm.ts
// ════════════════════════════════════════════════════════════

// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: wasm

export class WasmCompileMsg {
    source: string
    options: string

    constructor(source: string, options: string) {
        this.source = source
        this.options = options
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("source", this.source)
        if (this.options.length > 0) obj.set("options", JSONValue.parse(this.options))
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

export class WasmDescribeMsg {
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

export class WasmGetMsg {
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

export class WasmListMsg {

    toJSON(): string {
        let obj = new JSONObject()
        return obj.toString()
    }
}

export class WasmRemoveMsg {
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

export class WasmRunMsg {
    moduleId: string
    input: string

    constructor(moduleId: string, input: string) {
        this.moduleId = moduleId
        this.input = input
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("moduleId", this.moduleId)
        if (this.input.length > 0) obj.set("input", JSONValue.parse(this.input))
        return obj.toString()
    }
}

export class WasmUndeployMsg {
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

export class WasmCompileResp {
    moduleId: string
    name: string
    size: i32
    exports: string
    text: string
    error: string

    constructor() {
        this.moduleId = ""
        this.name = ""
        this.size = 0
        this.exports = ""
        this.text = ""
        this.error = ""
    }

    static parse(json: string): WasmCompileResp {
        let resp = new WasmCompileResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("moduleId")) resp.moduleId = obj.getString("moduleId")
            if (obj.has("name")) resp.name = obj.getString("name")
            if (obj.has("size")) resp.size = obj.getInt("size")
            if (obj.has("exports")) resp.exports = obj.get("exports").toString()
            if (obj.has("text")) resp.text = obj.getString("text")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class WasmDeployResp {
    module: string
    mode: string
    handlers: string
    error: string

    constructor() {
        this.module = ""
        this.mode = ""
        this.handlers = ""
        this.error = ""
    }

    static parse(json: string): WasmDeployResp {
        let resp = new WasmDeployResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("module")) resp.module = obj.getString("module")
            if (obj.has("mode")) resp.mode = obj.getString("mode")
            if (obj.has("handlers")) resp.handlers = obj.get("handlers").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class WasmDescribeResp {
    module: string
    mode: string
    handlers: string
    error: string

    constructor() {
        this.module = ""
        this.mode = ""
        this.handlers = ""
        this.error = ""
    }

    static parse(json: string): WasmDescribeResp {
        let resp = new WasmDescribeResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("module")) resp.module = obj.getString("module")
            if (obj.has("mode")) resp.mode = obj.getString("mode")
            if (obj.has("handlers")) resp.handlers = obj.get("handlers").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class WasmGetResp {
    module: string
    error: string

    constructor() {
        this.module = ""
        this.error = ""
    }

    static parse(json: string): WasmGetResp {
        let resp = new WasmGetResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("module")) resp.module = obj.get("module").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class WasmListResp {
    modules: string
    error: string

    constructor() {
        this.modules = ""
        this.error = ""
    }

    static parse(json: string): WasmListResp {
        let resp = new WasmListResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("modules")) resp.modules = obj.get("modules").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class WasmRemoveResp {
    removed: bool
    error: string

    constructor() {
        this.removed = false
        this.error = ""
    }

    static parse(json: string): WasmRemoveResp {
        let resp = new WasmRemoveResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("removed")) resp.removed = obj.getBool("removed")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class WasmRunResp {
    exitCode: i32
    value: string
    error: string

    constructor() {
        this.exitCode = 0
        this.value = ""
        this.error = ""
    }

    static parse(json: string): WasmRunResp {
        let resp = new WasmRunResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("exitCode")) resp.exitCode = obj.getInt("exitCode")
            if (obj.has("value")) resp.value = obj.get("value").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class WasmUndeployResp {
    undeployed: bool
    error: string

    constructor() {
        this.undeployed = false
        this.error = ""
    }

    static parse(json: string): WasmUndeployResp {
        let resp = new WasmUndeployResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("undeployed")) resp.undeployed = obj.getBool("undeployed")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export namespace wasm_ops {
    export function compile(msg: WasmCompileMsg, callback: string): void {
        _invokeAsync("wasm.compile", msg.toJSON(), callback)
    }

    export function deploy(msg: WasmDeployMsg, callback: string): void {
        _invokeAsync("wasm.deploy", msg.toJSON(), callback)
    }

    export function describe(msg: WasmDescribeMsg, callback: string): void {
        _invokeAsync("wasm.describe", msg.toJSON(), callback)
    }

    export function get(msg: WasmGetMsg, callback: string): void {
        _invokeAsync("wasm.get", msg.toJSON(), callback)
    }

    export function list(msg: WasmListMsg, callback: string): void {
        _invokeAsync("wasm.list", msg.toJSON(), callback)
    }

    export function remove(msg: WasmRemoveMsg, callback: string): void {
        _invokeAsync("wasm.remove", msg.toJSON(), callback)
    }

    export function run(msg: WasmRunMsg, callback: string): void {
        _invokeAsync("wasm.run", msg.toJSON(), callback)
    }

    export function undeploy(msg: WasmUndeployMsg, callback: string): void {
        _invokeAsync("wasm.undeploy", msg.toJSON(), callback)
    }

}

// ════════════════════════════════════════════════════════════
// Source: kit/runtime/wasm/generated/workflows.ts
// ════════════════════════════════════════════════════════════

// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: workflows

export class WorkflowCancelMsg {
    runId: string

    constructor(runId: string) {
        this.runId = runId
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("runId", this.runId)
        return obj.toString()
    }
}

export class WorkflowResumeMsg {
    runId: string
    stepId: string
    data: string

    constructor(runId: string, stepId: string, data: string) {
        this.runId = runId
        this.stepId = stepId
        this.data = data
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("runId", this.runId)
        obj.setString("stepId", this.stepId)
        if (this.data.length > 0) obj.set("data", JSONValue.parse(this.data))
        return obj.toString()
    }
}

export class WorkflowRunMsg {
    name: string
    input: string

    constructor(name: string, input: string) {
        this.name = name
        this.input = input
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        if (this.input.length > 0) obj.set("input", JSONValue.parse(this.input))
        return obj.toString()
    }
}

export class WorkflowStatusMsg {
    runId: string

    constructor(runId: string) {
        this.runId = runId
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("runId", this.runId)
        return obj.toString()
    }
}

export class WorkflowCancelResp {
    ok: bool
    error: string

    constructor() {
        this.ok = false
        this.error = ""
    }

    static parse(json: string): WorkflowCancelResp {
        let resp = new WorkflowCancelResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("ok")) resp.ok = obj.getBool("ok")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class WorkflowResumeResp {
    result: string
    error: string

    constructor() {
        this.result = ""
        this.error = ""
    }

    static parse(json: string): WorkflowResumeResp {
        let resp = new WorkflowResumeResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("result")) resp.result = obj.get("result").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class WorkflowRunResp {
    result: string
    error: string

    constructor() {
        this.result = ""
        this.error = ""
    }

    static parse(json: string): WorkflowRunResp {
        let resp = new WorkflowRunResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("result")) resp.result = obj.get("result").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class WorkflowStatusResp {
    status: string
    step: string
    error: string

    constructor() {
        this.status = ""
        this.step = ""
        this.error = ""
    }

    static parse(json: string): WorkflowStatusResp {
        let resp = new WorkflowStatusResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("status")) resp.status = obj.getString("status")
            if (obj.has("step")) resp.step = obj.getString("step")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export namespace workflows {
    export function cancel(msg: WorkflowCancelMsg, callback: string): void {
        _invokeAsync("workflows.cancel", msg.toJSON(), callback)
    }

    export function resume(msg: WorkflowResumeMsg, callback: string): void {
        _invokeAsync("workflows.resume", msg.toJSON(), callback)
    }

    export function run(msg: WorkflowRunMsg, callback: string): void {
        _invokeAsync("workflows.run", msg.toJSON(), callback)
    }

    export function status(msg: WorkflowStatusMsg, callback: string): void {
        _invokeAsync("workflows.status", msg.toJSON(), callback)
    }

}

// ════════════════════════════════════════════════════════════
// Source: kit/runtime/wasm/index.ts
// ════════════════════════════════════════════════════════════

// runtime/wasm/index.ts — Re-exports everything as the "brainkit" module.
// This file is concatenated LAST in the bundle.
// All exports from the namespace files are already in scope due to concatenation.

// Domain namespaces: ai, tools, agents, wasm_ops, memory, workflows, vectors, fs_ops, bus
// Shard functions: on, tool, reply, setMode, setModeKey
// State functions: getState, setState, hasState
// Log functions: log, logAt, debug, warn, error
// JSON library: JSONValue, JSONObject, JSONArray

