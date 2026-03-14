// Exports multiple arithmetic functions
export function add(a: i32, b: i32): i32 {
  return a + b;
}

export function multiply(a: i32, b: i32): i32 {
  return a * b;
}

export function run(): i32 {
  return add(multiply(6, 7), 1); // 43
}
