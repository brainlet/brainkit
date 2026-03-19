// Stateless shard that registers tools via tool().
// Tests: tool() registration during init, tool handler invocation
import { setMode, tool, reply } from "brainkit";

export function init(): void {
  setMode("stateless");
  tool("double", "handleDouble");
  tool("greet", "handleGreet");
}

export function handleDouble(topic: string, payload: string): void {
  // payload: {"value":5}
  const numStart = payload.indexOf('"value":') + 8;
  const numEnd = payload.indexOf('}', numStart);
  const num = I32.parseInt(payload.substring(numStart, numEnd));
  reply('{"result":' + (num * 2).toString() + '}');
}

export function handleGreet(topic: string, payload: string): void {
  const nameStart = payload.indexOf('"name":"') + 7;
  const nameEnd = payload.indexOf('"', nameStart + 1);
  const name = payload.substring(nameStart + 1, nameEnd);
  reply('{"message":"hello ' + name + '"}');
}
