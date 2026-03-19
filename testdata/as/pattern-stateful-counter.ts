import { getState, setState, log } from "brainkit";

export function run(): i32 {
  // 1. Read counter state (should be empty on first run)
  let raw = getState("counter");
  if (raw == "") {
    raw = "0";
  }

  // 2. Parse as integer and increment
  const current = I32.parseInt(raw);
  const next = current + 1;

  // 3. Store incremented value
  setState("counter", next.toString());
  log("counter incremented to " + next.toString());

  // 4. Read back and verify
  const stored = getState("counter");
  if (stored != "1") return 1;

  return 0;
}
