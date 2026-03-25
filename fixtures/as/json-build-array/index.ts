import { JSONArray } from "brainkit";

export function run(): i32 {
  const arr = new JSONArray()
    .pushString("hello")
    .pushInt(42)
    .pushBool(true)
    .pushNull()
    .pushNumber(3.14);

  // 1. Verify length
  if (arr.length != 5) return 1;

  // 2. Verify string at index 0
  if (!arr.at(0).isString()) return 2;
  if (arr.at(0).asString() != "hello") return 3;

  // 3. Verify int at index 1
  if (!arr.at(1).isNumber()) return 4;
  if (arr.at(1).asInt() != 42) return 5;

  // 4. Verify bool at index 2
  if (!arr.at(2).isBool()) return 6;
  if (arr.at(2).asBool() != true) return 7;

  // 5. Verify null at index 3
  if (!arr.at(3).isNull()) return 8;

  // 6. Verify number at index 4
  if (!arr.at(4).isNumber()) return 9;
  const n = arr.at(4).asNumber();
  if (n < 3.13 || n > 3.15) return 10;

  // 7. Verify serialization contains expected values
  const json = arr.toString();
  if (!json.includes('"hello"')) return 11;
  if (!json.includes("42")) return 12;
  if (!json.includes("true")) return 13;
  if (!json.includes("null")) return 14;

  return 0;
}
