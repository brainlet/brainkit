import { setState, getState, log, busSendRaw } from "wasm";

export function run(): i32 {
  // 1. setState with empty string value
  setState("emptyKey", "");

  // 2. getState round-trip — should return empty string
  const val = getState("emptyKey");
  if (val != "") return 1;

  // 3. log with empty string — should not crash
  log("");

  // 4. busSendRaw with empty string payload — should not crash
  busSendRaw("test.empty", "");

  return 0;
}
