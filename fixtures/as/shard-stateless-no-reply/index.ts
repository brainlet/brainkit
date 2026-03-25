// Stateless shard: handler that does NOT call reply().
// Tests: handler without reply is valid (fire-and-forget subscriber)
import { setMode, on, log } from "brainkit";

export function init(): void {
  setMode("stateless");
  on("test.silent", "handleSilent");
}

export function handleSilent(topic: string, payload: string): void {
  log("received silently: " + payload);
}
