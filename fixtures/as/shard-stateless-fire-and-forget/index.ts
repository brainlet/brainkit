// Stateless shard that uses send() (fire-and-forget) inside a handler.
// Tests: bus.sendRaw from within a handler, no reply
import { setMode, on, log } from "brainkit";
import { emit } from "brainkit";

export function init(): void {
  setMode("stateless");
  on("test.forward", "handleForward");
}

export function handleForward(topic: string, payload: string): void {
  emit("test.forwarded", payload);
  log("forwarded");
}
