import { setState, getState, log } from "brainkit";

export function run(): i32 {
  // 1. Set the mode
  setState("mode", "production");

  // 2. Read it back
  const mode = getState("mode");
  if (mode != "production") return 1;

  // 3. Conditional logic based on mode
  let result: string;
  if (mode == "production") {
    result = "safe";
  } else {
    result = "test";
  }

  // 4. Store the computed result
  setState("result", result);
  log("mode=" + mode + " result=" + result);

  // 5. Verify final state
  const stored = getState("result");
  if (stored != "safe") return 2;

  return 0;
}
