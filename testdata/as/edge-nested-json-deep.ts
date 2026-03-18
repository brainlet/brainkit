import { JSONValue, JSONObject } from "wasm";

export function run(): i32 {
  // Build 10-level deep nested JSONObject
  let inner = new JSONObject().setString("leaf", "found");

  for (let i: i32 = 0; i < 9; i++) {
    const outer = new JSONObject();
    outer.setObject("child", inner);
    inner = outer;
  }

  // inner is now the root with 10 levels deep

  // 1. Serialize
  const json = inner.toString();
  if (json.length == 0) return 1;

  // 2. Parse back
  const parsed = JSONValue.parse(json);
  if (parsed.isNull()) return 2;
  if (!parsed.isObject()) return 3;

  // 3. Walk 9 levels of "child" to reach the leaf
  let current = parsed.asObject();
  for (let i: i32 = 0; i < 9; i++) {
    if (!current.has("child")) return 4 + i;
    current = current.getObject("child");
  }

  // 4. Verify the leaf value
  if (!current.has("leaf")) return 13;
  if (current.getString("leaf") != "found") return 14;

  return 0;
}
