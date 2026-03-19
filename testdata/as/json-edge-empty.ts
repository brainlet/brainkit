import { JSONValue } from "brainkit";

export function run(): i32 {
  // 1. Parse empty object
  const obj = JSONValue.parse("{}");
  if (!obj.isObject()) return 1;
  if (obj.asObject().size() != 0) return 2;

  // 2. Parse empty array
  const arr = JSONValue.parse("[]");
  if (!arr.isArray()) return 3;
  if (arr.asArray().length != 0) return 4;

  // 3. Parse empty string
  const str = JSONValue.parse('""');
  if (!str.isString()) return 5;
  if (str.asString() != "") return 6;
  if (str.asString().length != 0) return 7;

  return 0;
}
