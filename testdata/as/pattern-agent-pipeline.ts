import { bus, setState, log, JSONValue } from "brainkit";

export function run(): i32 {
  // Stage 1: initial agent call
  bus.askAsyncRaw("agents.request", '{"name":"test-helper","prompt":"first prompt"}', "onStage1");
  return 0;
}

export function onStage1(topic: string, payload: string): void {
  if (payload.length == 0) {
    setState("error", "stage1 empty");
    return;
  }
  log("stage1: " + payload);

  const parsed = JSONValue.parse(payload);
  if (parsed.isNull()) {
    setState("error", "stage1 null");
    return;
  }

  const obj = parsed.asObject();
  const text1 = obj.getString("text");
  if (text1.length == 0) {
    setState("error", "stage1 no text");
    return;
  }

  // Stage 2: follow-up call using first result
  const followUp = "based on: " + text1 + " - continue";
  const stage2Payload = '{"name":"test-helper","prompt":"' + followUp + '"}';
  bus.askAsyncRaw("agents.request", stage2Payload, "onStage2");
}

export function onStage2(topic: string, payload: string): void {
  if (payload.length == 0) {
    setState("error", "stage2 empty");
    return;
  }
  log("stage2: " + payload);
  setState("ok", "true");
}
