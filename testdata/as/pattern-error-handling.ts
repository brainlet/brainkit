import { callToolRaw, log } from "wasm";

export function run(): i32 {
  // Call a tool that does not exist
  const result = callToolRaw("nonexistent", "{}");
  log("raw result: " + result);

  // The result should contain an error indication
  if (result.includes("error") || result.includes("Error") || result.length == 0) {
    log("error correctly detected");
    return 0;
  }

  // If we get here, the call unexpectedly succeeded
  return 1;
}
