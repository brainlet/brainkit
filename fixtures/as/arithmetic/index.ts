// Exports multiple arithmetic functions
export function add(a: i32, b: i32): i32 {
  return a + b;
}

export function multiply(a: i32, b: i32): i32 {
  return a * b;
}

export function run(): i32 {
  if (add(multiply(6, 7), 1) != 43) return 1;
  if (add(10, 20) != 30) return 2;
  if (multiply(3, 4) != 12) return 3;
  return 0;
}
