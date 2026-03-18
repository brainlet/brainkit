import { busSend, JSONObject } from "wasm";

export function run(): i32 {
  const payload = new JSONObject().setString("msg", "hello");
  busSend("as.test.hello", payload);
  return 0;
}
