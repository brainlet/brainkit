import { setState, getState } from "wasm";

export function run(): i32 {
  // 1. Get non-existent key returns empty
  if (getState("missing") != "") return 1;

  // 2. Set and get
  setState("key1", "value1");
  if (getState("key1") != "value1") return 2;

  // 3. Overwrite
  setState("key1", "value2");
  if (getState("key1") != "value2") return 3;

  // 4. Multiple keys
  setState("a", "1");
  setState("b", "2");
  if (getState("a") != "1") return 4;
  if (getState("b") != "2") return 5;

  // 5. Empty value
  setState("empty", "");
  if (getState("empty") != "") return 6;

  return 0;
}
