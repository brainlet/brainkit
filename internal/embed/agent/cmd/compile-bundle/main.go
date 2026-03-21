// compile-bundle compiles the agent-embed JavaScript bundle to QuickJS bytecode.
//
// Usage:
//
//	go run ./cmd/compile-bundle
//
// Reads agent_embed_bundle.js, compiles to bytecode, writes agent_embed_bundle.bc.
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
	_, thisFile, _, _ := runtime.Caller(0)
	agentEmbedDir := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	srcPath := filepath.Join(agentEmbedDir, "agent_embed_bundle.js")
	dstPath := filepath.Join(agentEmbedDir, "agent_embed_bundle.bc")

	source, err := os.ReadFile(srcPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read source: %v\n", err)
		os.Exit(1)
	}

	rt := quickjs.NewRuntime()
	defer rt.Close()

	ctx := rt.NewContext()
	defer ctx.Close()

	bytecode, err := ctx.Compile(string(source), quickjs.EvalFileName("agent-embed-bundle.js"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "compile: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(dstPath, bytecode, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write bytecode: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Compiled: %s (%.1f MB) → %s (%.1f MB)\n",
		filepath.Base(srcPath), float64(len(source))/1024/1024,
		filepath.Base(dstPath), float64(len(bytecode))/1024/1024)
}
