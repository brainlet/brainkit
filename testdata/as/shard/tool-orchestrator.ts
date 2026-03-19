// Stateless shard: receives a request, calls a tool via askAsync, stores result.
// Tests: tools.call via askAsync, callback processes tool output
import { setMode, on, reply, setState, log, JSONValue, JSONObject } from "brainkit";
import { bus } from "brainkit";

export function init(): void {
  setMode("stateless");
  on("orchestrate.query", "handleQuery");
}

export function handleQuery(topic: string, payload: string): void {
  const parsed = JSONValue.parse(payload);
  if (parsed.isNull()) {
    reply('{"error":"invalid payload"}');
    return;
  }
  const obj = parsed.asObject();
  const toolName = obj.getString("tool");
  const input = obj.get("input");

  // Call the tool via askAsync
  const toolPayload = new JSONObject()
    .setString("name", toolName)
    .set("input", input);
  bus.askAsyncRaw("tools.call", toolPayload.toString(), "onToolResult");
  log("calling tool: " + toolName);
}

export function onToolResult(topic: string, payload: string): void {
  setState("toolResult", payload);
  log("tool result received");
}
