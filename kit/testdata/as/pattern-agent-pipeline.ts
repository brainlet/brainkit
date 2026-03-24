// Pattern: multi-stage WASM pipeline calling .ts agent service via bus.
// Note: agents.request was removed — agent calls now go to .ts services via bus.
import { publish, setState, log, JSONValue } from "brainkit";

export function run(): i32 {
  // Stage 1: initial agent call via bus
  publish("ts.agent-service.ask", '{"agent":"test-helper","prompt":"first prompt"}', "onStage1");
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
  publish("ts.agent-service.ask", '{"agent":"test-helper","prompt":"' + followUp + '"}', "onStage2");
}

export function onStage2(topic: string, payload: string): void {
  if (payload.length == 0) {
    setState("error", "stage2 empty");
    return;
  }
  log("stage2: " + payload);
  setState("ok", "true");
}
