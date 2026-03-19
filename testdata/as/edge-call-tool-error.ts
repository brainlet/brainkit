import { bus, setState } from "brainkit";

export function run(): i32 {
  // Call a tool that does not exist
  bus.askAsyncRaw("tools.call", '{"name":"nonexistent_tool","input":{}}', "onResult");
  return 0;
}

export function onResult(topic: string, payload: string): void {
  // Result should not be empty
  if (payload.length == 0) {
    setState("error", "empty result");
    return;
  }
  // Result should contain "error" substring
  if (payload.includes("error")) {
    setState("ok", "true");
    setState("errorDetected", "true");
  } else {
    setState("error", "no error in response");
  }
}
