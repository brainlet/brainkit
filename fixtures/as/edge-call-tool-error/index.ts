import { publish, setState, JSONObject } from "brainkit";

export function run(): i32 {
  // Call a tool that does not exist
  const payload = new JSONObject()
    .setString("name", "nonexistent_tool")
    .setString("input", "{}");
  publish("tools.call", payload.toString(), "onResult");
  return 0;
}

export function onResult(topic: string, payload: string): void {
  if (payload.length == 0) {
    setState("error", "empty result");
    return;
  }
  if (payload.includes("error")) {
    setState("ok", "true");
    setState("errorDetected", "true");
  } else {
    setState("error", "no error in response");
  }
}
