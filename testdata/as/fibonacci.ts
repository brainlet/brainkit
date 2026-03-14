// Computes the Nth Fibonacci number
export function fib(n: i32): i32 {
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

export function run(): i32 {
  return fib(10); // 55
}
