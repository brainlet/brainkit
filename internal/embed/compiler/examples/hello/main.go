// Example: compile AssemblyScript to WASM with as-embed, then run it with wazero.
//
// Usage:
//
//	go run ./as-embed/examples/hello
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	asembed "github.com/brainlet/brainkit/internal/embed/compiler"
)

func main() {
	// ---------------------------------------------------------------
	// 1. Create the AssemblyScript compiler (loads QuickJS + AS bundle)
	// ---------------------------------------------------------------
	compiler, err := asembed.NewCompiler()
	if err != nil {
		fmt.Fprintf(os.Stderr, "NewCompiler: %v\n", err)
		os.Exit(1)
	}
	defer compiler.Close()

	// ---------------------------------------------------------------
	// 2. Compile AssemblyScript source to WASM binary
	// ---------------------------------------------------------------
	source := `
export function add(a: i32, b: i32): i32 {
  return a + b;
}

export function fibonacci(n: i32): i32 {
  if (n <= 1) return n;
  let a: i32 = 0, b: i32 = 1;
  for (let i: i32 = 2; i <= n; i++) {
    let t = a + b;
    a = b;
    b = t;
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
`

	result, err := compiler.Compile(map[string]string{
		"hello.ts": source,
	}, asembed.CompileOptions{
		OptimizeLevel: 2,
		ShrinkLevel:   1,
		Runtime:       "stub",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Compile: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Compiled %d bytes of WASM\n", len(result.Binary))
	if result.Text != "" {
		fmt.Printf("Warnings:\n%s\n", result.Text)
	}

	// ---------------------------------------------------------------
	// 3. Run the WASM module with wazero
	// ---------------------------------------------------------------
	ctx := context.Background()
	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	// AssemblyScript requires env.abort for runtime assertions.
	_, err = rt.NewHostModuleBuilder("env").
		NewFunctionBuilder().
		WithFunc(func(_ context.Context, m api.Module, msgPtr, filePtr, line, col uint32) {
			fmt.Fprintf(os.Stderr, "abort called at line %d, col %d\n", line, col)
		}).
		Export("abort").
		Instantiate(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "host module: %v\n", err)
		os.Exit(1)
	}

	mod, err := rt.Instantiate(ctx, result.Binary)
	if err != nil {
		fmt.Fprintf(os.Stderr, "instantiate: %v\n", err)
		os.Exit(1)
	}

	// ---------------------------------------------------------------
	// 4. Call the exported functions
	// ---------------------------------------------------------------
	callI32(ctx, mod, "add", 3, 5)
	callI32(ctx, mod, "add", 100, 200)
	callI32(ctx, mod, "fibonacci", 10)
	callI32(ctx, mod, "fibonacci", 20)
	callI32(ctx, mod, "factorial", 8)
	callI32(ctx, mod, "factorial", 12)
}

func callI32(ctx context.Context, mod api.Module, name string, args ...uint64) {
	fn := mod.ExportedFunction(name)
	if fn == nil {
		fmt.Fprintf(os.Stderr, "function %q not exported\n", name)
		return
	}
	results, err := fn.Call(ctx, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s error: %v\n", name, err)
		return
	}
	argStrs := make([]string, len(args))
	for i, a := range args {
		argStrs[i] = fmt.Sprintf("%d", int32(a))
	}
	fmt.Printf("%s(%s) = %d\n", name, strings.Join(argStrs, ", "), int32(results[0]))
}
