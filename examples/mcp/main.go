// Command mcp wires an external Model Context Protocol server
// as first-class tools on a brainkit Kit. Uses the npx-published
// `@modelcontextprotocol/server-filesystem` MCP server pointed
// at a temp directory seeded with a known file, then:
//
//   1. Lists tools the server advertises.
//   2. Invokes read_file and prints the content.
//
// Prerequisites: node + npx on PATH.
//
// Run from the repo root:
//
//	go run ./examples/mcp
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/brainlet/brainkit"
	mcpmod "github.com/brainlet/brainkit/modules/mcp"
	"github.com/brainlet/brainkit/sdk"
)

const seededFile = "hello.txt"
const seededBody = "hello from an MCP-managed filesystem\n"

func main() {
	if err := run(); err != nil {
		log.Fatalf("mcp: %v", err)
	}
}

func run() error {
	if _, err := exec.LookPath("npx"); err != nil {
		return fmt.Errorf("npx not found on PATH — install node.js / npm")
	}

	tmpRaw, err := os.MkdirTemp("", "brainkit-mcp-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpRaw)

	// macOS symlinks /var → /private/var; the MCP filesystem
	// server's "allowed directories" check compares the resolved
	// path, so we resolve before passing it in.
	tmp, err := filepath.EvalSymlinks(tmpRaw)
	if err != nil {
		return fmt.Errorf("resolve tempdir: %w", err)
	}

	// Seed the tempdir with a known file the MCP filesystem
	// server will expose.
	if err := os.WriteFile(filepath.Join(tmp, seededFile), []byte(seededBody), 0644); err != nil {
		return fmt.Errorf("seed: %w", err)
	}

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "mcp-demo",
		Transport: brainkit.Memory(),
		FSRoot:    tmp,
		Modules: []brainkit.Module{
			mcpmod.New(map[string]mcpmod.ServerConfig{
				"fs": {
					Command: "npx",
					Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", tmp},
				},
			}),
		},
	})
	if err != nil {
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()

	// MCP server startup can take a few seconds on a cold npx
	// cache; give it room.
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Println("MCP server 'fs' tool catalog:")
	list, err := brainkit.CallMcpListTools(kit, ctx, sdk.McpListToolsMsg{Server: "fs"},
		brainkit.WithCallTimeout(45*time.Second))
	if err != nil {
		return fmt.Errorf("mcp.listTools: %w", err)
	}
	for _, t := range list.Tools {
		fmt.Printf("  %s  %s\n", t.Name, t.Description)
	}
	if len(list.Tools) == 0 {
		return fmt.Errorf("MCP server returned zero tools — likely failed to start")
	}

	fmt.Println()
	fmt.Printf("mcp.callTool fs/read_text_file path=%s:\n", seededFile)
	res, err := brainkit.CallMcpCallTool(kit, ctx, sdk.McpCallToolMsg{
		Server: "fs",
		Tool:   "read_text_file",
		Args:   map[string]any{"path": filepath.Join(tmp, seededFile)},
	}, brainkit.WithCallTimeout(30*time.Second))
	if err != nil {
		return fmt.Errorf("mcp.callTool: %w", err)
	}

	// The MCP call response is structured JSON from the server;
	// print it raw so the tool output is visible end-to-end.
	var pretty any
	if err := json.Unmarshal(res.Result, &pretty); err != nil {
		fmt.Println(string(res.Result))
	} else {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("  ", "  ")
		_ = enc.Encode(pretty)
	}
	return nil
}
