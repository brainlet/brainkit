// Stateless shard that uses askAsync inside a handler.
// Tests: askAsync with named callback, callback receives response
import { setMode, on, setState } from "brainkit";
import { bus } from "brainkit";

export function init(): void {
  setMode("stateless");
  on("test.ask", "handleAsk");
}

export function handleAsk(topic: string, payload: string): void {
  bus.askAsyncRaw("tools.call", '{"name":"echo","input":{"value":"hello"}}', "onToolResult");
}

export function onToolResult(topic: string, payload: string): void {
  setState("askResult", payload);
}
