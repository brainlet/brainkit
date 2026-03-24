// Tests: WASM module calling a .ts agent service via bus_publish.
// Note: agents.request was removed — agent calls now go to .ts services via bus.
import { publish, setState, JSONValue } from "brainkit";

export function run(): i32 {
  // Publish to a .ts agent service via bus
  publish("ts.agent-service.ask", '{"agent":"test-helper","prompt":"say hello"}', "onAgentResult");
  return 0;
}

export function onAgentResult(topic: string, payload: string): void {
  if (payload.length == 0) {
    setState("error", "empty");
    return;
  }
  const parsed = JSONValue.parse(payload);
  if (parsed.isNull()) {
    setState("error", "null");
    return;
  }
  const obj = parsed.asObject();
  const text = obj.getString("text");
  if (text.length == 0) {
    setState("error", "no text");
    return;
  }
  setState("ok", "true");
  setState("text", text);
}
