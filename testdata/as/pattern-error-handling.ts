import { bus, setState, log } from "brainkit";

export function run(): i32 {
  // Call a tool that does not exist
  bus.askAsyncRaw("tools.call", '{"name":"nonexistent","input":{}}', "onResult");
  return 0;
}

export function onResult(topic: string, payload: string): void {
  log("raw result: " + payload);

  if (payload.includes("error") || payload.includes("Error") || payload.length == 0) {
    log("error correctly detected");
    setState("ok", "true");
  } else {
    setState("error", "expected error, got success");
  }
}
