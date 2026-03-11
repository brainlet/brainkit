// Example: compile an AssemblyScript file with wasm-kit, then run it with wazero.
//
// Usage:
//
//	go run ./wasm-kit/examples/hello-wazero
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/compiler"
	"github.com/brainlet/brainkit/wasm-kit/parser"
	"github.com/brainlet/brainkit/wasm-kit/program"
)

func main() {
	// ---------------------------------------------------------------
	// 1. Compile the AssemblyScript source with wasm-kit
	// ---------------------------------------------------------------
	wasmBytes, err := compileAS("hello.ts")
	if err != nil {
		fmt.Fprintf(os.Stderr, "compile error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Compiled %d bytes of Wasm\n", len(wasmBytes))


	// ---------------------------------------------------------------
	// 2. Run it with wazero
	// ---------------------------------------------------------------
	ctx := context.Background()
	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	// AssemblyScript may import env.abort for runtime assertions.
	// Provide a host implementation that prints and traps.
	_, err = rt.NewHostModuleBuilder("env").
		NewFunctionBuilder().
		WithFunc(func(_ context.Context, m api.Module, msgPtr, filePtr, line, col uint32) {
			fmt.Fprintf(os.Stderr, "abort called at line %d, col %d\n", line, col)
		}).
		Export("abort").
		Instantiate(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "host module error: %v\n", err)
		os.Exit(1)
	}

	mod, err := rt.Instantiate(ctx, wasmBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "instantiate error: %v\n", err)
		os.Exit(1)
	}

	// Call exported functions
	callI32(ctx, mod, "add", 3, 5)
	callI32(ctx, mod, "fibonacci", 10)
	callI32(ctx, mod, "factorial", 8)
}

// callI32 calls an exported i32-returning function and prints the result.
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
	// Format the call
	argStrs := make([]string, len(args))
	for i, a := range args {
		argStrs[i] = fmt.Sprintf("%d", int32(a))
	}
	fmt.Printf("%s(%s) = %d\n", name, strings.Join(argStrs, ", "), int32(results[0]))
}

// ---------------------------------------------------------------
// wasm-kit compile pipeline
// ---------------------------------------------------------------

// compileAS compiles an AssemblyScript .ts file (relative to this example
// directory) into a Wasm binary using the wasm-kit compiler.
func compileAS(filename string) ([]byte, error) {
	// Resolve paths relative to the source file location
	_, thisFile, _, _ := runtime.Caller(0)
	exampleDir := filepath.Dir(thisFile)
	stdRoot := filepath.Join(exampleDir, "..", "..", "std", "assembly")

	sourcePath := filepath.Join(exampleDir, filename)
	sourceText, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("read source: %w", err)
	}

	// Parse the user source
	p := parser.NewParser(nil)
	p.ParseFile(string(sourceText), filename, true)

	// Parse the standard library
	stdFiles, err := collectTS(stdRoot)
	if err != nil {
		return nil, fmt.Errorf("collect std: %w", err)
	}
	for _, f := range stdFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("read std %s: %w", f, err)
		}
		rel, _ := filepath.Rel(stdRoot, f)
		logicalPath := common.LIBRARY_PREFIX + filepath.ToSlash(rel)
		p.ParseFile(string(data), logicalPath, false)
	}

	// Drain the parser's file queue (transitive imports)
	for next := p.NextFile(); next != ""; next = p.NextFile() {
	}

	sources := p.Sources()
	p.Finish()

	// Note: p.Diagnostics may contain non-fatal warnings from std lib
	// (e.g. "Not implemented: union types"). These don't affect compilation.

	// Create program and compile
	opts := program.NewOptions()
	opts.Runtime = common.RuntimeStub // lightweight runtime, no GC
	opts.OptimizeLevelHint = 2
	opts.ShrinkLevelHint = 1

	prog := program.NewProgram(opts, nil)
	prog.Sources = sources

	c := compiler.NewCompiler(prog)
	mod := c.CompileProgram()
	if mod == nil {
		return nil, fmt.Errorf("compilation failed")
	}

	// Optimize
	mod.Optimize(
		int(opts.OptimizeLevelHint),
		int(opts.ShrinkLevelHint),
		opts.DebugInfo,
		opts.ZeroFilledMemory,
	)

	// Emit binary
	bin := mod.ToBinary("")
	return bin.Binary, nil
}

// collectTS walks a directory for all .ts files (sorted).
func collectTS(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".ts" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}
