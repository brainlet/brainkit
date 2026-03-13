// compile-bundle compiles the AS compiler JavaScript bundle to QuickJS bytecode.
//
// Usage:
//
//	go run ./cmd/compile-bundle
//
// Reads as_compiler_bundle.js, compiles to bytecode, writes as_compiler_bundle.bc.
// Run this after rebuilding the JS bundle (npm run build).
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	quickjs "github.com/buke/quickjs-go"
)

func main() {
	// Find the as-embed directory (parent of cmd/compile-bundle)
	_, thisFile, _, _ := runtime.Caller(0)
	asEmbedDir := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	srcPath := filepath.Join(asEmbedDir, "as_compiler_bundle.js")
	dstPath := filepath.Join(asEmbedDir, "as_compiler_bundle.bc")

	source, err := os.ReadFile(srcPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read source: %v\n", err)
		os.Exit(1)
	}

	rt := quickjs.NewRuntime()
	defer rt.Close()

	ctx := rt.NewContext()
	defer ctx.Close()

	bytecode, err := ctx.Compile(string(source), quickjs.EvalFileName("as-compiler-bundle.js"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "compile: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(dstPath, bytecode, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write bytecode: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Compiled: %s (%.1f KB) → %s (%.1f KB) [%.1fx larger]\n",
		filepath.Base(srcPath), float64(len(source))/1024,
		filepath.Base(dstPath), float64(len(bytecode))/1024,
		float64(len(bytecode))/float64(len(source)))
}
