import { JSONValue, JSONObject } from "wasm";

export function run(): i32 {
  const obj = new JSONObject()
    .setString("q", 'say "hi"')
    .setString("b", "back\\slash");

  // 1. Verify original values via getter
  if (obj.getString("q") != 'say "hi"') return 1;
  if (obj.getString("b") != "back\\slash") return 2;

  // 2. Serialize — should contain escaped sequences
  const json = obj.toString();
  if (!json.includes("\\\"")) return 3;  // escaped quote
  if (!json.includes("\\\\")) return 4;  // escaped backslash

  // 3. Parse back
  const parsed = JSONValue.parse(json);
  if (parsed.isNull()) return 5;

  const obj2 = parsed.asObject();
  if (obj2.getString("q") != 'say "hi"') return 6;
  if (obj2.getString("b") != "back\\slash") return 7;

  return 0;
}
