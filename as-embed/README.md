# as-embed

Embed the [AssemblyScript](https://www.assemblyscript.org/) compiler inside a Go binary. Compile `.ts` to `.wasm` without Node.js, npm, or any external runtime.

The compiler runs as original JavaScript inside [QuickJS](https://github.com/buke/quickjs-go) (native CGo). All ~900 Binaryen C API calls are bridged to Go's existing CGo bindings. The result is a single Go binary that compiles AssemblyScript to WebAssembly.

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    asembed "github.com/brainlet/brainkit/as-embed"
)

func main() {
    c, err := asembed.NewCompiler()
    if err != nil {
        log.Fatal(err)
    }
    defer c.Close()

    result, err := c.Compile(map[string]string{
        "input.ts": `export function add(a: i32, b: i32): i32 { return a + b; }`,
    }, asembed.CompileOptions{
        OptimizeLevel: 2,
        ShrinkLevel:   1,
        Runtime:       "stub",
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Compiled %d bytes of WebAssembly\n", len(result.Binary))
}
```

See [`examples/hello/main.go`](examples/hello/main.go) for a complete example that compiles, loads with [wazero](https://wazero.io/), and runs exported functions.

## Architecture

```
Go Application
  |
  v
asembed.Compiler  (Go API: Compile(sources, opts) -> wasm bytes)
  |
  v
jsbridge + QuickJS (native CGo, buke/quickjs-go v0.6.10)
  |
  +-- AS Compiler JS bundle (~631KB, bytecode-precompiled)
  |     tokenizer -> parser -> resolver -> compiler
  |     |
  |     | calls _BinaryenModuleCreate(), _BinaryenBlock(), etc.
  |     v
  +-- binaryen-shim.js (Proxy-based module routing)
        |
        | routes each _Binaryen*() call to Go
        v
      Go CGo Bindings (bridge_impl_*.go, ~4,500 lines)
        |
        v
      libbinaryen.a (Binaryen C library, built locally)
```

**Key properties:**

- **Single Go binary** -- no Node.js, no npm at runtime, no sidecar processes
- **Original source preserved** -- the AS compiler JS runs as-is (bundled with esbuild, not rewritten)
- **~3.3s per compilation** -- stable across 50+ sequential compilations
- **7ms bundle load** -- bytecode precompilation (4.24x faster than source eval)
- **Upstream sync** -- `npm update` in `bundle/`, re-bundle, re-test

## API Reference

### Types

```go
// Compiler wraps a QuickJS bridge with the AssemblyScript compiler loaded.
type Compiler struct { /* internal */ }

// CompilerConfig controls Compiler creation.
type CompilerConfig struct {
    MemoryLimit  int // bytes; default 512MB
    MaxStackSize int // bytes; default 256MB (effectively disables QuickJS stack check)
}

// CompileOptions controls compilation behavior.
type CompileOptions struct {
    OptimizeLevel int           // 0-3 (default: 0)
    ShrinkLevel   int           // 0-2 (default: 0)
    Debug         bool          // include debug info in output
    Runtime       string        // "stub", "minimal", or "incremental" (default: "incremental")
    Timeout       time.Duration // per-compilation timeout; 0 = no timeout
}

// CompileResult holds the output of a successful compilation.
type CompileResult struct {
    Binary []byte // .wasm binary
    Text   string // compiler warnings
    WAT    string // WebAssembly text format (S-expression)
}
```

### Functions

```go
// NewCompiler creates a Compiler with default configuration.
func NewCompiler() (*Compiler, error)

// NewCompilerWithConfig creates a Compiler with explicit configuration.
func NewCompilerWithConfig(cfg CompilerConfig) (*Compiler, error)

// Compile compiles AssemblyScript source files to WebAssembly.
// The first key in sources is the entry file (auto-detected by import analysis).
// Returns ErrCompilerDead if a previous compilation timed out or panicked.
func (c *Compiler) Compile(sources map[string]string, opts CompileOptions) (*CompileResult, error)

// Dead returns true if the compiler's QuickJS runtime is corrupted.
// The Compiler must be closed and recreated.
func (c *Compiler) Dead() bool

// Close releases all resources held by the Compiler.
func (c *Compiler) Close()
```

### Multi-file Projects

Pass all source files as a `map[string]string`. Import resolution works automatically:

```go
result, err := c.Compile(map[string]string{
    "main.ts":  `import { helper } from "./utils"; export function run(): i32 { return helper(42); }`,
    "utils.ts": `export function helper(x: i32): i32 { return x * 2; }`,
}, asembed.CompileOptions{})
```

The entry file is auto-detected: the file that isn't imported by any other file.

### Node Modules (Package Imports)

Bare package imports like `import "my-package"` are supported. Include the package files in your sources map with `node_modules/` prefixed keys:

```go
result, err := c.Compile(map[string]string{
    "main.ts":                       `import "my-lib"; /* ... */`,
    "node_modules/my-lib/index.ts":  `export function foo(): void { /* ... */ }`,
}, asembed.CompileOptions{})
```

Nested `node_modules` resolution follows Node.js conventions (walk-up search from the importing package's directory).

### Runtime Variants

| Runtime | Description | Use Case |
|---------|-------------|----------|
| `"stub"` | No GC, no runtime overhead | Pure computation, no heap allocations |
| `"minimal"` | Basic mark-sweep GC | Simple programs with some allocations |
| `"incremental"` | Full incremental GC (default) | General purpose, classes, strings, arrays |

### Sequential Compilations

A single `Compiler` instance can be reused for many compilations. Memory is managed automatically:

```go
c, _ := asembed.NewCompiler()
defer c.Close()

for _, source := range sources {
    result, err := c.Compile(source, opts)
    // QuickJS GC runs automatically between compilations
}
```

Validated stable across 50+ sequential compilations at ~3.3s each.

### Timeout & Error Recovery

```go
result, err := c.Compile(sources, asembed.CompileOptions{
    Timeout: 30 * time.Second,
})
if err != nil {
    if c.Dead() {
        // Runtime corrupted (timeout or panic) -- must recreate
        c.Close()
        c, _ = asembed.NewCompiler()
    }
}
```

## Standard Library

The full AssemblyScript standard library is embedded via `//go:embed std`. This includes:

- Core types: `string`, `array`, `map`, `set`, `arraybuffer`, `typedarray`, `dataview`
- Math: full `Math` API
- Runtime: `stub`, `minimal`, `incremental` GC variants with TLSF allocator
- Utilities: sorting, hashing, URI encoding, case mapping
- Platform bindings: DOM stubs, Node.js stubs

**63 TypeScript files, ~22K lines** -- all compiled into the Go binary.

## Build & Setup

### Prerequisites

- Go 1.24+
- C/C++ compiler (for CGo -- both QuickJS and Binaryen)
- CMake (for building Binaryen)
- Node.js + npm (only for rebuilding the JS bundle, not at runtime)

### First-Time Setup

```bash
# 1. Build Binaryen C library
./scripts/setup-binaryen.sh
# Downloads binaryen v123, builds libbinaryen.a into deps/binaryen/

# 2. Install npm dependencies and build the JS bundle
cd bundle && npm install && node build.mjs && cd ..

# 3. Compile JS to QuickJS bytecode (faster loading)
go generate ./...

# 4. Verify everything works
go test ./... -timeout 15m
```

### Rebuilding After Upstream Changes

```bash
cd bundle
npm update assemblyscript  # pull latest AS compiler
node build.mjs             # re-bundle with esbuild
cd ..
go generate ./...          # recompile to bytecode
go test ./...              # verify
```

No fork to maintain. The AS compiler source is pulled from npm (GitHub tarball for v0.28.10).

### Directory Structure

```
as-embed/
  compiler.go              # Main API: Compiler, Compile(), CompileOptions
  embed.go                 # go:embed directives, bundle/stdlib loading
  memory.go                # LinearMemory (256MB bump allocator for Binaryen)
  bridge.go                # Memory bridge: _malloc, _free, HEAP ops
  binaryen_shim.js         # JS module routing (Proxy-based require)
  binaryen_bridge.go       # Generated stub bridge (~4K lines)
  binaryen_bridge_impl.go  # Bridge registration + helpers
  binaryen_cgo.go          # CGo wrappers for Binaryen C API (~3.5K lines)
  binaryen_cgo_getters.go  # CGo property getters (~1.1K lines)
  bridge_impl_*.go         # Real bridge implementations (7 files, ~5K lines)
  as_compiler_bundle.js    # Bundled AS compiler (esbuild output, ~631KB)
  as_compiler_bundle.bc    # QuickJS bytecode (precompiled, ~817KB)
  std/                     # AssemblyScript standard library (63 .ts files)
  bundle/                  # npm package + esbuild config
    package.json           # assemblyscript v0.28.10 + as-float + long
    entry.mjs              # esbuild entry: imports AS compiler API
    build.mjs              # esbuild config: IIFE, browser, minified
  cmd/
    compile-bundle/        # go:generate tool: JS -> QuickJS bytecode
    gen-bridge/            # Code generator: Binaryen glue -> Go bridge
  deps/
    binaryen/              # Built Binaryen: include/ + lib/libbinaryen.a
  scripts/
    setup-binaryen.sh      # Downloads and builds Binaryen v123
  examples/
    hello/                 # Complete example: compile AS + run with wazero
```

## How It Works

### Compilation Flow

1. **Load** -- QuickJS evaluates the AS compiler bundle (bytecode: 7ms). The full compiler API is available on `globalThis.__as_compiler`.

2. **Parse** -- Standard library files (~63) are parsed first, then the runtime entry, then user source files. Additional files requested by the compiler via `nextFile()` are provided on demand (including `node_modules` resolution).

3. **Initialize** -- `initializeProgram()` resolves all types, validates imports, builds the symbol table.

4. **Compile** -- `compile()` generates Binaryen IR. Every `_Binaryen*()` call from JS routes through the bridge to Go's CGo bindings, which call `libbinaryen.a` directly.

5. **Optimize & Validate** -- Binaryen's optimizer and validator run on the module.

6. **Serialize** -- `BinaryenModuleAllocateAndWrite()` produces the final `.wasm` binary via CGo.

7. **Cleanup** -- The JS module is disposed, `Runtime.RunGC()` reclaims memory.

### Binaryen Bridge

The AS compiler calls ~900 Binaryen C API functions (e.g., `_BinaryenBlock`, `_BinaryenCall`, `_BinaryenModuleOptimize`). These are declared in `src/glue/binaryen.js` in the AS source tree.

`cmd/gen-bridge` parses this glue file and generates:
- **`binaryen_bridge.go`** -- registers each function as a QuickJS global
- **`bridge_impl_*.go`** -- real implementations using CGo
- **`binaryen_cgo.go`** -- CGo function declarations

The bridge uses a `LinearMemory` (256MB Go byte slice) to emulate Emscripten's shared memory model. JS calls `_malloc`, writes to HEAP via `__i32_store`, and Go reads the values for CGo calls. Binaryen handles (ExpressionRef, etc.) flow through JS as float64 numbers.

### Memory Management

- **LinearMemory**: 256MB bump allocator, reset between compilations
- **QuickJS GC**: `Runtime.RunGC()` called after each module disposal
- **GC Threshold**: 4MB auto-GC to prevent heap accumulation
- **Pointer safety**: ARM64 Binaryen pointers exceed 32 bits; `ptrOverrides` map preserves full 64-bit values

## Testing

```bash
# Quick smoke test (~15s)
go test ./as-embed/ -run "TestCompileSimpleProgram" -v

# Full compilation suite -- 116 AS test fixtures (~7 min)
go test ./as-embed/ -run "TestASCompilerSuite" -v -timeout 15m

# Runtime validation -- compile + execute in wazero (~1 min short, ~30 min full)
go test ./as-embed/ -run "TestASCompilerRuntime" -short -v
go test ./as-embed/ -run "TestASCompilerRuntime" -v -timeout 45m

# Sequential stability -- 50 compilations (~3 min)
go test ./as-embed/ -run "TestCumulativeCrashPoint" -v -timeout 10m

# All tests
go test ./as-embed/... -timeout 15m
```

### Test Coverage

| Test | What It Proves | Result |
|------|---------------|--------|
| `TestCompileSimpleProgram` | Basic compilation works | PASS |
| `TestCompileMatchesNodeJS` | Output has valid Wasm magic + version | PASS |
| `TestASCompilerSuite` | 116 upstream fixtures compile successfully | 116/116 PASS |
| `TestASCompilerRuntime -short` | 21 fixtures compile AND execute correctly | 21/21 PASS |
| `TestASCompilerRuntime` (full) | 103 fixtures execute, 14 known-skipped | 103/103 PASS |
| `TestCumulativeCrashPoint` | 50 sequential compilations, no OOM | 50/50 PASS |
| `TestBundleLoadTime` | Bundle loads in ~7ms (bytecode) | PASS |
| `TestMemoryBridge*` | Linear memory ops work correctly | PASS |

### Known Runtime Skips (14 tests)

These tests compile to valid Wasm but fail at execution time:

| Category | Count | Cause |
|----------|-------|-------|
| Invalid table access | 7 | Function pointer / indirect call table layout differs from upstream (likely Binaryen version mismatch) |
| Assert failures | 6 | Specific codegen edge cases producing wrong results |
| Memory bounds | 1 | Memory layout issue in compiled output |

## Performance

| Metric | Value |
|--------|-------|
| Bundle load (bytecode) | ~7ms |
| Bundle load (source) | ~30ms |
| Compilation (simple function) | ~3.5s |
| Compilation (complex program) | ~3.5s |
| Sequential compilations | Stable at ~3.3s/each for 50+ |
| Bridge call overhead | ~8us per Binaryen call |
| Memory per compilation | ~50-100MB (reclaimed by GC) |
| Binary size overhead | ~631KB JS + ~43MB libbinaryen |

## Comparison with Manual Go Port

| Metric | Manual Go Port (wasm-kit) | QuickJS Embedding (as-embed) |
|--------|--------------------------|------------------------------|
| Lines written | ~47K Go (20K remaining) | ~4.5K Go (bridge + API) |
| Correctness | 0/122 tests passing | 116/116 compile, 103 execute |
| Time invested | Months | Days |
| Upstream sync | Re-port every change | Re-bundle, re-test |
| Performance | Native Go | ~3.3s/compilation (acceptable for build tool) |

## Dependencies

| Dependency | Role | Type |
|------------|------|------|
| [buke/quickjs-go](https://github.com/buke/quickjs-go) v0.6.10 | JavaScript engine (native CGo) | Go module |
| [jsbridge](../jsbridge/) | QuickJS polyfills (console, encoding, crypto, etc.) | Internal package |
| [Binaryen](https://github.com/WebAssembly/binaryen) v123 | WebAssembly IR library | C/C++ static lib (CGo) |
| [AssemblyScript](https://github.com/AssemblyScript/assemblyscript) v0.28.10 | Compiler source | npm (bundled at build time) |
| [esbuild](https://esbuild.github.io/) | JS bundler | npm devDep (build time only) |
| [wazero](https://wazero.io/) v1.11.0 | Wasm runtime (tests + examples) | Go module (test only) |

## License

AssemblyScript is Apache-2.0 licensed. Binaryen is Apache-2.0 licensed.
