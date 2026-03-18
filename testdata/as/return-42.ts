// Simplest WASM module — verifies basic compilation + execution
export function run(): i32 {
  const x: i32 = 42;
  if (x != 42) return 1;
  return 0;
}
