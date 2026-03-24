// Stateless shard with multiple handlers on different topics.
// Tests: multiple on() registrations, correct dispatch
import { setMode, on, reply } from "brainkit";

export function init(): void {
  setMode("stateless");
  on("test.ping", "handlePing");
  on("test.pong", "handlePong");
}

export function handlePing(topic: string, payload: string): void {
  reply('{"handler":"ping"}');
}

export function handlePong(topic: string, payload: string): void {
  reply('{"handler":"pong"}');
}
