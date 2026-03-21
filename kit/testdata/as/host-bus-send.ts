import { bus, JSONObject } from "brainkit";

export function run(): i32 {
  const payload = new JSONObject().setString("msg", "hello");
  bus.sendRaw("as.test.hello", payload.toString());
  return 0;
}
