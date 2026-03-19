import { JSONValue } from "brainkit";

export function run(): i32 {
  const v = JSONValue.parse('{"user":{"name":"Alice","scores":[10,20,30]},"ok":true}');
  if (v.isNull()) return 1;

  const obj = v.asObject();
  const user = obj.getObject("user");
  if (user.getString("name") != "Alice") return 2;

  const scores = user.getArray("scores");
  if (scores.length != 3) return 3;
  if (scores.at(0).asInt() != 10) return 4;
  if (scores.at(1).asInt() != 20) return 5;
  if (scores.at(2).asInt() != 30) return 6;

  if (obj.getBool("ok") != true) return 7;

  return 0;
}
