import { callToolRaw } from "wasm";

export function run(): i32 {
  // Call a tool that does not exist
  const result = callToolRaw("nonexistent_tool", "{}");

  // 1. Result should not be empty
  if (result.length == 0) return 1;

  // 2. Result should contain "error" substring
  if (!result.includes("error")) return 2;

  return 0;
}
