// Stateless shard that uses the typed async tool wrapper inside a handler.
// Tests: async typed wrapper with named callback, callback receives result
import { setMode, on, setState, tools, ToolCallMsg } from "brainkit";

export function init(): void {
  setMode("stateless");
  on("test.ask", "handleAsk");
}

export function handleAsk(topic: string, payload: string): void {
  tools.call(new ToolCallMsg("echo", '{"value":"hello"}'), "onToolResult");
}

export function onToolResult(topic: string, payload: string): void {
  setState("askResult", payload);
}
