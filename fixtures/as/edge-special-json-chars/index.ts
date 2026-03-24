import { JSONValue, JSONObject } from "brainkit";

export function run(): i32 {
  const obj = new JSONObject()
    .setString("tab", "a\tb")
    .setString("newline", "a\nb")
    .setString("quote", 'a"b')
    .setString("backslash", "a\\b");

  // 1. Verify original values
  if (obj.getString("tab") != "a\tb") return 1;
  if (obj.getString("newline") != "a\nb") return 2;
  if (obj.getString("quote") != 'a"b') return 3;
  if (obj.getString("backslash") != "a\\b") return 4;

  // 2. Serialize
  const json = obj.toString();
  if (json.length == 0) return 5;

  // 3. Serialized form should contain escape sequences
  if (!json.includes("\\t")) return 6;
  if (!json.includes("\\n")) return 7;
  if (!json.includes("\\\"")) return 8;
  if (!json.includes("\\\\")) return 9;

  // 4. Parse back and verify round-trip
  const parsed = JSONValue.parse(json);
  if (parsed.isNull()) return 10;

  const obj2 = parsed.asObject();
  if (obj2.getString("tab") != "a\tb") return 11;
  if (obj2.getString("newline") != "a\nb") return 12;
  if (obj2.getString("quote") != 'a"b') return 13;
  if (obj2.getString("backslash") != "a\\b") return 14;

  return 0;
}
