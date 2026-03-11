// A simple AssemblyScript module compiled by wasm-kit and run with wazero.

export function add(a: i32, b: i32): i32 {
  return a + b;
}

export function fibonacci(n: i32): i32 {
  if (n <= 1) return n;
  let a: i32 = 0;
  let b: i32 = 1;
  for (let i: i32 = 2; i <= n; i++) {
    let tmp = a + b;
    a = b;
    b = tmp;
  }
  return b;
}

export function factorial(n: i32): i32 {
  if (n <= 1) return 1;
  let result: i32 = 1;
  for (let i: i32 = 2; i <= n; i++) {
    result *= i;
  }
  return result;
}
