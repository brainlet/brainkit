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
  if (fib(10) != 55) return 1;
  if (fib(0) != 0) return 2;
  if (fib(1) != 1) return 3;
  if (fib(20) != 6765) return 4;
  return 0;
}
