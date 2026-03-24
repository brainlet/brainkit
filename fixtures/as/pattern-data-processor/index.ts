import { emit, publish, setState, log, JSONObject, JSONValue } from "brainkit";

export function run(): i32 {
  // 1. Build input data
  const input = new JSONObject()
    .setString("action", "process")
    .setInt("value", 100);

  // 2. Call echo tool via bus publish
  const payload = new JSONObject()
    .setString("name", "echo")
    .set("input", JSONValue.parse(input.toString()));
  publish("tools.call", payload.toString(), "onToolResult");
  return 0;
}

export function onToolResult(topic: string, payload: string): void {
  if (payload.length == 0) {
    setState("error", "empty");
    return;
  }

  // 3. Log the result
  log("processed: " + payload);

  // 4. Forward result via bus
  const output = new JSONObject()
    .setString("status", "processed")
    .setString("raw", payload);
  emit("data.processed", output.toString());

  setState("ok", "true");
}
