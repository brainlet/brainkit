// Stateless shard: parses incoming JSON and replies with transformed data.
// Tests: JSON handling in handler, structured reply
import { setMode, on, reply, log } from "brainkit";

export function init(): void {
  setMode("stateless");
  on("test.transform", "handleTransform");
}

export function handleTransform(topic: string, payload: string): void {
  // Extract "name" field from payload manually (AS has no built-in JSON.parse to object)
  const nameStart = payload.indexOf('"name":"') + 7;
  const nameEnd = payload.indexOf('"', nameStart + 1);
  const name = payload.substring(nameStart + 1, nameEnd);

  reply('{"greeting":"hello ' + name + '","original":' + payload + '}');
}
