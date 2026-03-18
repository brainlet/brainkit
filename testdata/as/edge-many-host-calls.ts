import { setState, getState } from "wasm";

export function run(): i32 {
  // Loop 200 times: set, get, verify
  for (let i: i32 = 0; i < 200; i++) {
    const key = "k";
    const val = i.toString();
    setState(key, val);
    const got = getState(key);
    if (got != val) return i + 1;
  }

  return 0;
}
