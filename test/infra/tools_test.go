package infra_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
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
		{"Kernel", testutil.NewTestKernel},
		{"Node", testutil.NewTestNode},
	} {
		t.Run(factory.name, func(t *testing.T) {
			rt := factory.make(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			t.Run("List_FindsRegisteredTools", func(t *testing.T) {
				_pr1, err := sdk.Publish(rt, ctx, messages.ToolListMsg{})
				require.NoError(t, err)
				_ch1 := make(chan messages.ToolListResp, 1)
				_us1, err := sdk.SubscribeTo[messages.ToolListResp](rt, ctx, _pr1.ReplyTo, func(r messages.ToolListResp, m messages.Message) { _ch1 <- r })
				require.NoError(t, err)
				defer _us1()
				var resp messages.ToolListResp
				select {
				case resp = <-_ch1:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				names := make(map[string]bool)
				for _, tool := range resp.Tools {
					names[tool.ShortName] = true
				}
				assert.True(t, names["echo"], "echo tool should be registered")
				assert.True(t, names["add"], "add tool should be registered")
			})

			t.Run("Resolve_Echo", func(t *testing.T) {
				_pr2, err := sdk.Publish(rt, ctx, messages.ToolResolveMsg{Name: "echo"})
				require.NoError(t, err)
				_ch2 := make(chan messages.ToolResolveResp, 1)
				_us2, err := sdk.SubscribeTo[messages.ToolResolveResp](rt, ctx, _pr2.ReplyTo, func(r messages.ToolResolveResp, m messages.Message) { _ch2 <- r })
				require.NoError(t, err)
				defer _us2()
				var resp messages.ToolResolveResp
				select {
				case resp = <-_ch2:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.Equal(t, "echo", resp.ShortName)
				assert.Equal(t, "echoes the input message", resp.Description)
				assert.NotNil(t, resp.InputSchema)
			})

			t.Run("Resolve_NotFound", func(t *testing.T) {
				pr, err := sdk.Publish(rt, ctx, messages.ToolResolveMsg{Name: "nonexistent"})
				require.NoError(t, err)
				ch := make(chan messages.ToolResolveResp, 1)
				un, _ := sdk.SubscribeTo[messages.ToolResolveResp](rt, ctx, pr.ReplyTo, func(r messages.ToolResolveResp, m messages.Message) { ch <- r })
				defer un()
				select {
				case resp := <-ch:
					assert.NotEmpty(t, resp.Error, "should have error in response")
				case <-ctx.Done():
					t.Fatal("timeout")
				}
			})

			t.Run("Call_Echo", func(t *testing.T) {
				_pr4, err := sdk.Publish(rt, ctx, messages.ToolCallMsg{
					Name:  "echo",
					Input: map[string]any{"message": "hello world"},
				})
				require.NoError(t, err)
				_ch4 := make(chan messages.ToolCallResp, 1)
				_us4, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr4.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch4 <- r })
				require.NoError(t, err)
				defer _us4()
				var resp messages.ToolCallResp
				select {
				case resp = <-_ch4:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				var result map[string]string
				json.Unmarshal(resp.Result, &result)
				assert.Equal(t, "hello world", result["echoed"])
			})

			t.Run("Call_Add", func(t *testing.T) {
				_pr5, err := sdk.Publish(rt, ctx, messages.ToolCallMsg{
					Name:  "add",
					Input: map[string]any{"a": 17, "b": 25},
				})
				require.NoError(t, err)
				_ch5 := make(chan messages.ToolCallResp, 1)
				_us5, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr5.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch5 <- r })
				require.NoError(t, err)
				defer _us5()
				var resp messages.ToolCallResp
				select {
				case resp = <-_ch5:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				var result map[string]int
				json.Unmarshal(resp.Result, &result)
				assert.Equal(t, 42, result["sum"])
			})

			t.Run("Call_NotFound", func(t *testing.T) {
				pr, err := sdk.Publish(rt, ctx, messages.ToolCallMsg{
					Name:  "nonexistent",
					Input: map[string]any{},
				})
				require.NoError(t, err)
				ch := make(chan messages.ToolCallResp, 1)
				un, _ := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, pr.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { ch <- r })
				defer un()
				select {
				case resp := <-ch:
					assert.NotEmpty(t, resp.Error, "should have error in response")
				case <-ctx.Done():
					t.Fatal("timeout")
				}
			})
		})
	}
}
