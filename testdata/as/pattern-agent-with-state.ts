import { callAgent, setState, getState, log } from "wasm";

export function run(): i32 {
  // 1. Call the agent
  const raw = callAgent("test-helper", "say hello");
  if (raw.length == 0) return 1;
  log("agent result: " + raw);

  // 2. Persist the agent result into state
  setState("agent-result", raw);

  // 3. Read it back and verify non-empty
  const stored = getState("agent-result");
  if (stored.length == 0) return 2;

  // 4. Verify the stored value matches what we set
  if (stored != raw) return 3;

  log("agent result persisted: " + stored);
  return 0;
}
