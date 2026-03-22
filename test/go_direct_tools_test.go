package test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoDirect_Tools(t *testing.T) {
	for _, factory := range []struct {
		name string
		make func(t *testing.T) sdk.Runtime
	}{
		{"Kernel", newTestKernel},
		{"Node", newTestNode},
	} {
		t.Run(factory.name, func(t *testing.T) {
			rt := factory.make(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			t.Run("List_FindsRegisteredTools", func(t *testing.T) {
				resp, err := sdk.PublishAwait[messages.ToolListMsg, messages.ToolListResp](rt, ctx, messages.ToolListMsg{})
				require.NoError(t, err)
				names := make(map[string]bool)
				for _, tool := range resp.Tools {
					names[tool.ShortName] = true
				}
				assert.True(t, names["echo"], "echo tool should be registered")
				assert.True(t, names["add"], "add tool should be registered")
			})

			t.Run("Resolve_Echo", func(t *testing.T) {
				resp, err := sdk.PublishAwait[messages.ToolResolveMsg, messages.ToolResolveResp](rt, ctx, messages.ToolResolveMsg{Name: "echo"})
				require.NoError(t, err)
				assert.Equal(t, "echo", resp.ShortName)
				assert.Equal(t, "echoes the input message", resp.Description)
				assert.NotNil(t, resp.InputSchema)
			})

			t.Run("Resolve_NotFound", func(t *testing.T) {
				_, err := sdk.PublishAwait[messages.ToolResolveMsg, messages.ToolResolveResp](rt, ctx, messages.ToolResolveMsg{Name: "nonexistent"})
				assert.Error(t, err)
			})

			t.Run("Call_Echo", func(t *testing.T) {
				resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](rt, ctx, messages.ToolCallMsg{
					Name:  "echo",
					Input: map[string]any{"message": "hello world"},
				})
				require.NoError(t, err)
				var result map[string]string
				json.Unmarshal(resp.Result, &result)
				assert.Equal(t, "hello world", result["echoed"])
			})

			t.Run("Call_Add", func(t *testing.T) {
				resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](rt, ctx, messages.ToolCallMsg{
					Name:  "add",
					Input: map[string]any{"a": 17, "b": 25},
				})
				require.NoError(t, err)
				var result map[string]int
				json.Unmarshal(resp.Result, &result)
				assert.Equal(t, 42, result["sum"])
			})

			t.Run("Call_NotFound", func(t *testing.T) {
				_, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](rt, ctx, messages.ToolCallMsg{
					Name:  "nonexistent",
					Input: map[string]any{},
				})
				assert.Error(t, err)
			})
		})
	}
}
