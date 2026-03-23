// testmcp is a minimal MCP server for testing mcp.listTools and mcp.callTool.
// It exposes one tool: "echo" that returns its input.
// Uses stdio transport — launched as a subprocess by the test.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	s := server.NewMCPServer("testmcp", "1.0.0")

	s.AddTool(
		mcp.Tool{
			Name:        "echo",
			Description: "Echoes the input message",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"message": map[string]any{"type": "string", "description": "Message to echo"},
				},
			},
		},
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := request.GetArguments()
			msg, _ := args["message"].(string)
			result, _ := json.Marshal(map[string]string{"echoed": msg, "server": "testmcp"})
			return mcp.NewToolResultText(string(result)), nil
		},
	)

	stdio := server.NewStdioServer(s)
	if err := stdio.Listen(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "testmcp: %v\n", err)
		os.Exit(1)
	}
}
