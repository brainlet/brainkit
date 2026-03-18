import { callToolRaw } from "wasm";

export function run(): i32 {
  const result = callToolRaw("echo", '{"key":"val"}');
  if (result.length == 0) return 1;
  return 0;
}
