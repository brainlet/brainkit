import { hasState, setState } from "brainkit";

export function run(): i32 {
  // 1. Missing key returns false
  if (hasState("nonexistent")) return 1;

  // 2. Set key then hasState returns true
  setState("mykey", "myval");
  if (!hasState("mykey")) return 2;

  // 3. Set empty string value, hasState still returns true
  setState("emptyval", "");
  if (!hasState("emptyval")) return 3;

  return 0;
}
