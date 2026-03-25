// Stateless shard that calls a tool via bus publish inside a handler.
// Tests: publish to tools.call with callback, callback receives result
import { setMode, on, setState, publish, JSONObject, JSONValue } from "brainkit";

export function init(): void {
  setMode("stateless");
  on("test.ask", "handleAsk");
}

export function handleAsk(topic: string, payload: string): void {
  const toolPayload = new JSONObject()
    .setString("name", "echo")
    .set("input", JSONValue.parse('{"value":"hello"}'));
  publish("tools.call", toolPayload.toString(), "onToolResult");
}

export function onToolResult(topic: string, payload: string): void {
  setState("askResult", payload);
}
