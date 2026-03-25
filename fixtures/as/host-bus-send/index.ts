import { emit, JSONObject } from "brainkit";

export function run(): i32 {
  const payload = new JSONObject().setString("msg", "hello");
  emit("as.test.hello", payload.toString());
  return 0;
}
