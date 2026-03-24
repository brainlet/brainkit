// Pattern: WASM calls .ts agent service via bus, persists result in state.
// Note: agents.request was removed — agent calls now go to .ts services via bus.
import { publish, setState, getState, log } from "brainkit";

export function run(): i32 {
  // 1. Call the agent service via bus
  publish("ts.agent-service.ask", '{"agent":"test-helper","prompt":"say hello"}', "onAgentResult");
  return 0;
}

export function onAgentResult(topic: string, payload: string): void {
  if (payload.length == 0) {
    setState("error", "empty");
    return;
  }
  log("agent result: " + payload);

  // 2. Persist the agent result into state
  setState("agent-result", payload);

  // 3. Read it back and verify non-empty
  const stored = getState("agent-result");
  if (stored.length == 0) {
    setState("error", "state empty after set");
    return;
  }

  // 4. Verify the stored value matches what we set
  if (stored != payload) {
    setState("error", "state mismatch");
    return;
  }

  log("agent result persisted: " + stored);
  setState("ok", "true");
}
