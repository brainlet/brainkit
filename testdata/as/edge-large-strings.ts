import { setState, getState } from "brainkit";

export function run(): i32 {
  // Build a 10000-char string
  let big = "";
  for (let i = 0; i < 10000; i++) {
    big += "x";
  }

  // 1. Verify length
  if (big.length != 10000) return 1;

  // 2. setState with large string
  setState("bigKey", big);

  // 3. getState round-trip
  const got = getState("bigKey");
  if (got.length != 10000) return 2;

  // 4. Verify content matches
  if (got != big) return 3;

  return 0;
}
