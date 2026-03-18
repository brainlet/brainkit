import { setState, getState } from "wasm";

export function run(): i32 {
  // Build a 500-char key
  let longKey = "";
  for (let i = 0; i < 500; i++) {
    longKey += "k";
  }

  // Build a 500-char value
  let longVal = "";
  for (let i = 0; i < 500; i++) {
    longVal += "v";
  }

  // 1. Verify key length
  if (longKey.length != 500) return 1;

  // 2. Verify value length
  if (longVal.length != 500) return 2;

  // 3. setState with long key and value
  setState(longKey, longVal);

  // 4. getState round-trip
  const got = getState(longKey);
  if (got != longVal) return 3;
  if (got.length != 500) return 4;

  return 0;
}
