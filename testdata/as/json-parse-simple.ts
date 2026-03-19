import { JSONValue } from "brainkit";

export function run(): i32 {
  const v = JSONValue.parse('{"name":"hello","count":42,"active":true,"empty":null}');
  if (v.isNull()) return 1;

  const obj = v.asObject();
  if (obj.getString("name") != "hello") return 2;
  if (obj.getInt("count") != 42) return 3;
  if (obj.getBool("active") != true) return 4;
  if (!obj.get("empty").isNull()) return 5;
  if (obj.size() != 4) return 6;

  return 0;
}
