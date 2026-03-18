import { JSONValue } from "wasm";

export function run(): i32 {
  const v = JSONValue.parse('[1,"two",true,null,3.14]');
  if (v.isNull()) return 1;
  if (!v.isArray()) return 2;

  const arr = v.asArray();

  // 1. Verify length
  if (arr.length != 5) return 3;

  // 2. First element: number 1
  if (!arr.at(0).isNumber()) return 4;
  if (arr.at(0).asInt() != 1) return 5;

  // 3. Second element: string "two"
  if (!arr.at(1).isString()) return 6;
  if (arr.at(1).asString() != "two") return 7;

  // 4. Third element: bool true
  if (!arr.at(2).isBool()) return 8;
  if (arr.at(2).asBool() != true) return 9;

  // 5. Fourth element: null
  if (!arr.at(3).isNull()) return 10;

  // 6. Fifth element: number 3.14
  if (!arr.at(4).isNumber()) return 11;
  const n = arr.at(4).asNumber();
  if (n < 3.13 || n > 3.15) return 12;

  return 0;
}
