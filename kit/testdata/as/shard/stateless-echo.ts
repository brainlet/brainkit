// Stateless shard: receives a message, echoes it back via reply().
// Tests: init, setMode("stateless"), on, handler signature, reply
import { setMode, on, reply } from "brainkit";

export function init(): void {
  setMode("stateless");
  on("test.echo", "handleEcho");
}

export function handleEcho(topic: string, payload: string): void {
  reply(payload);
}
