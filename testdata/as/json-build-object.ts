import { JSONObject, JSONArray } from "brainkit";

export function run(): i32 {
  const obj = new JSONObject()
    .setString("name", "test")
    .setInt("count", 42)
    .setBool("active", true)
    .setNull("empty");

  const json = obj.toString();

  // Verify key parts exist in serialized output
  if (!json.includes('"name"')) return 1;
  if (!json.includes('"test"')) return 2;
  if (!json.includes('"count"')) return 3;
  if (!json.includes('42')) return 4;
  if (!json.includes('true')) return 5;
  if (!json.includes('null')) return 6;

  // Verify typed getters
  if (obj.getString("name") != "test") return 7;
  if (obj.getInt("count") != 42) return 8;
  if (obj.getBool("active") != true) return 9;
  if (!obj.get("empty").isNull()) return 10;

  // Verify array serialization
  const arr = new JSONArray()
    .pushString("hello")
    .pushInt(1)
    .pushBool(false)
    .pushNull();

  const arrJson = arr.toString();
  if (!arrJson.includes('"hello"')) return 11;
  if (!arrJson.includes('false')) return 12;

  // Verify nested
  const nested = new JSONObject()
    .setObject("child", obj)
    .setArray("list", arr);

  const nestedJson = nested.toString();
  if (!nestedJson.includes('"child"')) return 13;
  if (!nestedJson.includes('"list"')) return 14;

  return 0;
}
