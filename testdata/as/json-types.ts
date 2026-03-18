import { JSONValue, JSONObject, JSONArray } from "wasm";

export function run(): i32 {
  // Create one of each type
  const strVal = JSONValue.Str("hello");
  const numVal = JSONValue.Number(42.0);
  const boolVal = JSONValue.Bool(true);
  const nullVal = JSONValue.Null();
  const objVal = new JSONObject();
  const arrVal = new JSONArray();

  // 1. String type checks
  if (!strVal.isString()) return 1;
  if (strVal.isNumber()) return 2;
  if (strVal.isBool()) return 3;
  if (strVal.isNull()) return 4;
  if (strVal.isObject()) return 5;
  if (strVal.isArray()) return 6;

  // 2. Number type checks
  if (!numVal.isNumber()) return 7;
  if (numVal.isString()) return 8;
  if (numVal.isBool()) return 9;
  if (numVal.isNull()) return 10;
  if (numVal.isObject()) return 11;
  if (numVal.isArray()) return 12;

  // 3. Bool type checks
  if (!boolVal.isBool()) return 13;
  if (boolVal.isString()) return 14;
  if (boolVal.isNumber()) return 15;
  if (boolVal.isNull()) return 16;
  if (boolVal.isObject()) return 17;
  if (boolVal.isArray()) return 18;

  // 4. Null type checks
  if (!nullVal.isNull()) return 19;
  if (nullVal.isString()) return 20;
  if (nullVal.isNumber()) return 21;
  if (nullVal.isBool()) return 22;
  if (nullVal.isObject()) return 23;
  if (nullVal.isArray()) return 24;

  // 5. Object type checks
  const objAsVal = changetype<JSONValue>(objVal);
  if (!objAsVal.isObject()) return 25;
  if (objAsVal.isString()) return 26;
  if (objAsVal.isNumber()) return 27;
  if (objAsVal.isBool()) return 28;
  if (objAsVal.isNull()) return 29;
  if (objAsVal.isArray()) return 30;

  // 6. Array type checks
  const arrAsVal = changetype<JSONValue>(arrVal);
  if (!arrAsVal.isArray()) return 31;
  if (arrAsVal.isString()) return 32;
  if (arrAsVal.isNumber()) return 33;
  if (arrAsVal.isBool()) return 34;
  if (arrAsVal.isNull()) return 35;
  if (arrAsVal.isObject()) return 36;

  return 0;
}
