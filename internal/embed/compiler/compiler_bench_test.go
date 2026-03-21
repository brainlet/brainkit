package asembed

import (
	"fmt"
	"testing"

	"github.com/brainlet/brainkit/internal/jsbridge"
)

// === Initialization Benchmarks ===

// BenchmarkNewCompiler measures the full compiler initialization cost:
// QuickJS runtime + polyfills + binaryen bridge + bundle load.
func BenchmarkNewCompiler(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		c, err := NewCompiler()
		if err != nil {
			b.Fatal(err)
		}
		c.Close()
	}
}

// BenchmarkBundleLoadBytecode measures loading the AS compiler from
// precompiled QuickJS bytecode (the fast path).
func BenchmarkBundleLoadBytecode(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		br := newBenchBridge(b)
		if err := LoadShim(br); err != nil {
			b.Fatal(err)
		}
		if err := LoadBundle(br); err != nil {
			b.Fatal(err)
		}
		br.Close()
	}
}

// BenchmarkBundleLoadSource measures loading the AS compiler from
// raw JavaScript source (the fallback path).
func BenchmarkBundleLoadSource(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		br := newBenchBridge(b)
		if err := LoadShim(br); err != nil {
			b.Fatal(err)
		}
		val, err := br.Eval("as-compiler-bundle.js", bundleSource)
		if err != nil {
			b.Fatal(err)
		}
		val.Free()
		br.Close()
	}
}

// === Compilation Benchmarks ===

// BenchmarkCompile/trivial - single exported function (1 line)
// BenchmarkCompile/simple - a few functions with control flow
// BenchmarkCompile/classes - class hierarchy with methods and fields
// BenchmarkCompile/complex - generics, arrays, closures, runtime
func BenchmarkCompile(b *testing.B) {
	type program struct {
		name    string
		sources map[string]string
	}
	// Use a slice to keep deterministic order
	programs := []program{
		{"trivial", map[string]string{
			"input.ts": `export function add(a: i32, b: i32): i32 { return a + b; }`,
		}},
		{"simple", map[string]string{
			"input.ts": `
export function fibonacci(n: i32): i32 {
  if (n <= 1) return n;
  let a: i32 = 0, b: i32 = 1;
  for (let i: i32 = 2; i <= n; i++) {
    let t = a + b; a = b; b = t;
  }
  return b;
}
export function factorial(n: i32): i32 {
  let result: i32 = 1;
  for (let i: i32 = 2; i <= n; i++) result *= i;
  return result;
}
export function isPrime(n: i32): bool {
  if (n < 2) return false;
  for (let i: i32 = 2; i * i <= n; i++) {
    if (n % i == 0) return false;
  }
  return true;
}`,
		}},
		{"classes", map[string]string{
			"input.ts": `
class Vec2 {
  x: f64; y: f64;
  constructor(x: f64, y: f64) { this.x = x; this.y = y; }
  length(): f64 { return Math.sqrt(this.x * this.x + this.y * this.y); }
  add(other: Vec2): Vec2 { return new Vec2(this.x + other.x, this.y + other.y); }
}
class Particle {
  pos: Vec2; vel: Vec2; mass: f64;
  constructor(x: f64, y: f64, vx: f64, vy: f64, m: f64) {
    this.pos = new Vec2(x, y); this.vel = new Vec2(vx, vy); this.mass = m;
  }
  step(dt: f64): void {
    this.pos = this.pos.add(new Vec2(this.vel.x * dt, this.vel.y * dt));
  }
}
export function simulate(steps: i32): f64 {
  let p = new Particle(0, 0, 1.0, 0.5, 2.0);
  for (let i: i32 = 0; i < steps; i++) p.step(0.01);
  return p.pos.length();
}`,
		}},
		{"complex", map[string]string{
			"input.ts": `
class Stack<T> {
  private items: Array<T> = [];
  push(item: T): void { this.items.push(item); }
  pop(): T { return this.items.pop(); }
  get size(): i32 { return this.items.length; }
  isEmpty(): bool { return this.items.length == 0; }
}
class TreeNode {
  value: i32; left: TreeNode | null; right: TreeNode | null;
  constructor(v: i32) { this.value = v; this.left = null; this.right = null; }
}
function insertBST(root: TreeNode | null, value: i32): TreeNode {
  if (root === null) return new TreeNode(value);
  if (value < root.value) root.left = insertBST(root.left, value);
  else root.right = insertBST(root.right, value);
  return root;
}
function inorderSum(node: TreeNode | null): i32 {
  if (node === null) return 0;
  return inorderSum(node.left) + node.value + inorderSum(node.right);
}
export function testStack(): i32 {
  let s = new Stack<i32>();
  for (let i: i32 = 0; i < 100; i++) s.push(i);
  let sum: i32 = 0;
  while (!s.isEmpty()) sum += s.pop();
  return sum;
}
export function testBST(): i32 {
  let root: TreeNode | null = null;
  let values: i32[] = [50, 30, 70, 20, 40, 60, 80, 10, 25, 35, 45, 55, 65, 75, 90];
  for (let i = 0; i < values.length; i++) root = insertBST(root, values[i]);
  return inorderSum(root);
}`,
		}},
	}

	runtimes := []string{"stub", "incremental"}

	for _, p := range programs {
		for _, rt := range runtimes {
			sources := p.sources
			benchName := fmt.Sprintf("%s/rt=%s", p.name, rt)
			b.Run(benchName, func(b *testing.B) {
				c, err := NewCompiler()
				if err != nil {
					b.Fatal(err)
				}
				defer c.Close()

				opts := CompileOptions{
					OptimizeLevel: 0,
					ShrinkLevel:   0,
					Runtime:       rt,
				}

				// Warmup
				result, err := c.Compile(sources, opts)
				if err != nil {
					b.Fatal(err)
				}
				b.ReportMetric(float64(len(result.Binary)), "wasm_bytes")

				b.ResetTimer()
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					_, err := c.Compile(sources, opts)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// BenchmarkCompileOptimized measures the impact of optimization levels.
func BenchmarkCompileOptimized(b *testing.B) {
	source := map[string]string{
		"input.ts": `
export function fibonacci(n: i32): i32 {
  if (n <= 1) return n;
  let a: i32 = 0, b: i32 = 1;
  for (let i: i32 = 2; i <= n; i++) {
    let t = a + b; a = b; b = t;
  }
  return b;
}`,
	}

	levels := []struct {
		name string
		opt  int
		shrk int
	}{
		{"O0", 0, 0},
		{"O1", 1, 0},
		{"O2", 2, 0},
		{"O3", 3, 0},
		{"O2s1", 2, 1},
		{"O2s2", 2, 2},
	}

	for _, l := range levels {
		b.Run(l.name, func(b *testing.B) {
			c, err := NewCompiler()
			if err != nil {
				b.Fatal(err)
			}
			defer c.Close()

			opts := CompileOptions{
				OptimizeLevel: l.opt,
				ShrinkLevel:   l.shrk,
				Runtime:       "stub",
			}

			result, err := c.Compile(source, opts)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportMetric(float64(len(result.Binary)), "wasm_bytes")

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, err := c.Compile(source, opts)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// === Throughput Benchmark ===

// BenchmarkSequentialCompilations measures sustained throughput
// across many compilations on a single compiler instance.
func BenchmarkSequentialCompilations(b *testing.B) {
	c, err := NewCompiler()
	if err != nil {
		b.Fatal(err)
	}
	defer c.Close()

	sources := map[string]string{
		"input.ts": `export function add(a: i32, b: i32): i32 { return a + b; }`,
	}
	opts := CompileOptions{Runtime: "stub"}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := c.Compile(sources, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// === Helpers ===

func newBenchBridge(b *testing.B) *jsbridge.Bridge {
	b.Helper()
	br, err := jsbridge.New(jsbridge.Config{
		MemoryLimit:  512 * 1024 * 1024,
		MaxStackSize: 256 * 1024 * 1024,
	},
		jsbridge.Console(),
		jsbridge.Encoding(),
		jsbridge.Streams(),
		jsbridge.Crypto(),
		jsbridge.URL(),
		jsbridge.Timers(),
		jsbridge.Abort(),
		jsbridge.Events(),
		jsbridge.StructuredClone(),
	)
	if err != nil {
		b.Fatal(err)
	}

	lm := NewLinearMemory()
	RegisterMemoryBridge(br.Context(), lm)
	RegisterBinaryenBridge(br.Context(), lm)
	RegisterBinaryenBridgeImpl(br.Context(), lm)

	return br
}
