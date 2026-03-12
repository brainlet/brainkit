// compile-bundle compiles the AI SDK JavaScript bundle to QuickJS bytecode.
//
// Usage:
//
//	go run ./cmd/compile-bundle
//
// Reads ai_sdk_bundle.js, compiles to bytecode, writes ai_sdk_bundle.bc.
// Run this after rebuilding the JS bundle (npm run build).
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/fastschema/qjs"
)

func main() {
	// Find the ai-embed directory (parent of cmd/compile-bundle)
	_, thisFile, _, _ := runtime.Caller(0)
	aiEmbedDir := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	srcPath := filepath.Join(aiEmbedDir, "ai_sdk_bundle.js")
	dstPath := filepath.Join(aiEmbedDir, "ai_sdk_bundle.bc")

	source, err := os.ReadFile(srcPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read source: %v\n", err)
		os.Exit(1)
	}

	rt, err := qjs.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "create runtime: %v\n", err)
		os.Exit(1)
	}
	defer rt.Close()

	bytecode, err := rt.Context().Compile("ai-sdk-bundle.js", qjs.Code(string(source)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "compile: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(dstPath, bytecode, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write bytecode: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Compiled: %s (%.1f KB) → %s (%.1f KB)\n",
		filepath.Base(srcPath), float64(len(source))/1024,
		filepath.Base(dstPath), float64(len(bytecode))/1024)
}
