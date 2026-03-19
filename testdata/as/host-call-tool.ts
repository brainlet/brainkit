import { bus, setState, JSONObject, JSONValue } from "brainkit";

export function run(): i32 {
  const args = new JSONObject().setString("key", "val");
  const payload = new JSONObject()
    .setString("name", "echo")
    .set("input", args);
  bus.askAsyncRaw("tools.call", payload.toString(), "onToolResult");
  return 0;
}

export function onToolResult(topic: string, payload: string): void {
  if (payload.length == 0) {
    setState("error", "empty result");
    return;
  }
  const parsed = JSONValue.parse(payload);
  if (parsed.isNull()) {
    setState("error", "null result");
    return;
  }
  setState("ok", "true");
}
