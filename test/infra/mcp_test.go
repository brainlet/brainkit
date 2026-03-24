package infra_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	mcppkg "github.com/brainlet/brainkit/internal/mcp"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/kit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoDirect_MCP(t *testing.T) {
	// Build the testmcp binary
	mcpBinary := testutil.BuildTestMCP(t)

	for _, backend := range testutil.AllBackends(t) {
		t.Run(backend, func(t *testing.T) {
			tmpDir := t.TempDir()

			k, err := kit.NewKernel(kit.KernelConfig{
				Namespace:    "test-mcp",
				CallerID:     "test-mcp-caller",
				WorkspaceDir: tmpDir,
				MCPServers: map[string]mcppkg.ServerConfig{
					"testmcp": {
						Command: mcpBinary,
					},
				},
			})
			require.NoError(t, err)
			defer k.Close()

			rt := sdk.Runtime(k)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			t.Run("ListTools", func(t *testing.T) {
				_pr1, err := sdk.Publish(rt, ctx, messages.McpListToolsMsg{})
				require.NoError(t, err)
				_ch1 := make(chan messages.McpListToolsResp, 1)
				_us1, err := sdk.SubscribeTo[messages.McpListToolsResp](rt, ctx, _pr1.ReplyTo, func(r messages.McpListToolsResp, m messages.Message) { _ch1 <- r })
				require.NoError(t, err)
				defer _us1()
				var resp messages.McpListToolsResp
				select {
				case resp = <-_ch1:
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				found := false
				for _, tool := range resp.Tools {
					if tool.Name == "echo" && tool.Server == "testmcp" {
						found = true
					}
				}
				assert.True(t, found, "testmcp echo tool should be listed")
			})

			t.Run("CallTool", func(t *testing.T) {
				_pr2, err := sdk.Publish(rt, ctx, messages.McpCallToolMsg{
					Server: "testmcp",
					Tool:   "echo",
					Args:   map[string]any{"message": "hello from mcp test"},
				})
				require.NoError(t, err)
				_ch2 := make(chan messages.McpCallToolResp, 1)
				_us2, err := sdk.SubscribeTo[messages.McpCallToolResp](rt, ctx, _pr2.ReplyTo, func(r messages.McpCallToolResp, m messages.Message) { _ch2 <- r })
				require.NoError(t, err)
				defer _us2()
				var resp messages.McpCallToolResp
				select {
				case resp = <-_ch2:
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				var result map[string]string
				json.Unmarshal(resp.Result, &result)
				assert.Equal(t, "hello from mcp test", result["echoed"])
				assert.Equal(t, "testmcp", result["server"])
			})

			t.Run("CallTool_via_registry", func(t *testing.T) {
				// MCP tools are also registered in the tool registry — verify they're callable
				// via tools.call (which looks them up by short name)
				_pr3, err := sdk.Publish(rt, ctx, messages.ToolCallMsg{
					Name:  "echo",
					Input: map[string]any{"message": "via registry"},
				})
				require.NoError(t, err)
				_ch3 := make(chan messages.ToolCallResp, 1)
				_us3, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr3.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch3 <- r })
				require.NoError(t, err)
				defer _us3()
				var resp messages.ToolCallResp
				select {
				case resp = <-_ch3:
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				var result map[string]string
				json.Unmarshal(resp.Result, &result)
				assert.Equal(t, "via registry", result["echoed"])
			})
		})
	}
}
