import { JSONValue, JSONObject, JSONArray } from "wasm";

export function run(): i32 {
  // Build a complex object
  const original = new JSONObject()
    .setString("name", "roundtrip")
    .setInt("version", 3)
    .setBool("enabled", true)
    .setNull("nothing")
    .setArray("tags", new JSONArray().pushString("a").pushString("b"))
    .setObject("nested", new JSONObject().setInt("depth", 1));

  // Serialize
  const json = original.toString();

  // Parse back
  const parsed = JSONValue.parse(json);
  if (parsed.isNull()) return 1;

  const obj = parsed.asObject();
  if (obj.getString("name") != "roundtrip") return 2;
  if (obj.getInt("version") != 3) return 3;
  if (obj.getBool("enabled") != true) return 4;
  if (!obj.get("nothing").isNull()) return 5;
  if (obj.getArray("tags").length != 2) return 6;
  if (obj.getArray("tags").at(0).asString() != "a") return 7;
  if (obj.getObject("nested").getInt("depth") != 1) return 8;

  return 0;
}
