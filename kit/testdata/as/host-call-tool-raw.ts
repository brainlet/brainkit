import { bus, setState } from "brainkit";

export function run(): i32 {
  bus.askAsyncRaw("tools.call", '{"name":"echo","input":{"key":"val"}}', "onResult");
  return 0;
}

export function onResult(topic: string, payload: string): void {
  if (payload.length == 0) {
    setState("error", "empty");
    return;
  }
  setState("ok", "true");
}
