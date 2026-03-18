import { setState, getState } from "wasm";

export function run(): i32 {
  // Set 100 keys
  for (let i: i32 = 0; i < 100; i++) {
    setState("k" + i.toString(), "v" + i.toString());
  }

  // Verify all 100 keys
  for (let i: i32 = 0; i < 100; i++) {
    const expected = "v" + i.toString();
    const actual = getState("k" + i.toString());
    if (actual != expected) return i + 1;
  }

  return 0;
}
