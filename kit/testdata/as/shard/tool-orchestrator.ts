// Stateless shard: receives a request, calls a tool via the typed async wrapper, stores result.
// Tests: tools.call async wrapper, callback processes tool output
import { setMode, on, reply, setState, log, JSONValue, tools, ToolCallMsg } from "brainkit";

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

  // Call the tool via the typed async wrapper
  tools.call(new ToolCallMsg(toolName, input.toString()), "onToolResult");
  log("calling tool: " + toolName);
}

export function onToolResult(topic: string, payload: string): void {
  setState("toolResult", payload);
  log("tool result received");
}
