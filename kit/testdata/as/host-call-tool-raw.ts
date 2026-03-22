import { tools, ToolCallMsg, setState } from "brainkit";

export function run(): i32 {
  tools.call(new ToolCallMsg("echo", '{"key":"val"}'), "onResult");
  return 0;
}

export function onResult(topic: string, payload: string): void {
  if (payload.length == 0) {
    setState("error", "empty");
    return;
  }
  setState("ok", "true");
}
