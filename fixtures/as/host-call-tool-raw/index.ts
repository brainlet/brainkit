import { publish, setState, JSONObject, JSONValue } from "brainkit";

export function run(): i32 {
  const payload = new JSONObject()
    .setString("name", "echo")
    .set("input", JSONValue.parse('{"key":"val"}'));
  publish("tools.call", payload.toString(), "onResult");
  return 0;
}

export function onResult(topic: string, payload: string): void {
  if (payload.length == 0) {
    setState("error", "empty");
    return;
  }
  setState("ok", "true");
}
