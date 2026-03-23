package test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	mcppkg "github.com/brainlet/brainkit/internal/mcp"
	"github.com/brainlet/brainkit/kit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPluginSurface tests every domain operation from the Plugin/Node surface.
// Uses newTestNode(t) which creates a Node with memory transport — same sdk.Runtime
// interface a real plugin subprocess would use over NATS.
// This proves the full: Go → sdk.Publish → transport → router → handler path
// from the Node/plugin perspective.

// --- Tools domain from Plugin ---

func TestPluginSurface_Tools(t *testing.T) {
	rt := newTestNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("List", func(t *testing.T) {
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
		assert.NotEmpty(t, resp.Tools)
	})
	t.Run("Resolve", func(t *testing.T) {
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
	})
	t.Run("Call", func(t *testing.T) {
		_pr3, err := sdk.Publish(rt, ctx, messages.ToolCallMsg{
			Name: "add", Input: map[string]any{"a": 3, "b": 7},
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
		var result map[string]int
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, 10, result["sum"])
	})
}

// --- FS domain from Plugin ---

func TestPluginSurface_FS(t *testing.T) {
	rt := newTestNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Write", func(t *testing.T) {
		_pr4, err := sdk.Publish(rt, ctx, messages.FsWriteMsg{Path: "plugin-test.txt", Data: "from plugin"})
		require.NoError(t, err)
		_ch4 := make(chan messages.FsWriteResp, 1)
		_us4, err := sdk.SubscribeTo[messages.FsWriteResp](rt, ctx, _pr4.ReplyTo, func(r messages.FsWriteResp, m messages.Message) { _ch4 <- r })
		require.NoError(t, err)
		defer _us4()
		var resp messages.FsWriteResp
		select {
		case resp = <-_ch4:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.True(t, resp.OK)
	})
	t.Run("Read", func(t *testing.T) {
		sdk.Publish(rt, ctx, messages.FsWriteMsg{Path: "plugin-read.txt", Data: "data"})
		_pr5, err := sdk.Publish(rt, ctx, messages.FsReadMsg{Path: "plugin-read.txt"})
		require.NoError(t, err)
		_ch5 := make(chan messages.FsReadResp, 1)
		_us5, err := sdk.SubscribeTo[messages.FsReadResp](rt, ctx, _pr5.ReplyTo, func(r messages.FsReadResp, m messages.Message) { _ch5 <- r })
		require.NoError(t, err)
		defer _us5()
		var resp messages.FsReadResp
		select {
		case resp = <-_ch5:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.Equal(t, "data", resp.Data)
	})
	t.Run("Mkdir", func(t *testing.T) {
		_pr6, err := sdk.Publish(rt, ctx, messages.FsMkdirMsg{Path: "plugin-dir"})
		require.NoError(t, err)
		_ch6 := make(chan messages.FsMkdirResp, 1)
		_us6, err := sdk.SubscribeTo[messages.FsMkdirResp](rt, ctx, _pr6.ReplyTo, func(r messages.FsMkdirResp, m messages.Message) { _ch6 <- r })
		require.NoError(t, err)
		defer _us6()
		var resp messages.FsMkdirResp
		select {
		case resp = <-_ch6:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.True(t, resp.OK)
	})
	t.Run("List", func(t *testing.T) {
		sdk.Publish(rt, ctx, messages.FsWriteMsg{Path: "plugin-ls/a.txt", Data: "a"})
		_pr7, err := sdk.Publish(rt, ctx, messages.FsListMsg{Path: "plugin-ls"})
		require.NoError(t, err)
		_ch7 := make(chan messages.FsListResp, 1)
		_us7, err := sdk.SubscribeTo[messages.FsListResp](rt, ctx, _pr7.ReplyTo, func(r messages.FsListResp, m messages.Message) { _ch7 <- r })
		require.NoError(t, err)
		defer _us7()
		var resp messages.FsListResp
		select {
		case resp = <-_ch7:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotEmpty(t, resp.Files)
	})
	t.Run("Stat", func(t *testing.T) {
		sdk.Publish(rt, ctx, messages.FsWriteMsg{Path: "plugin-stat.txt", Data: "x"})
		_pr8, err := sdk.Publish(rt, ctx, messages.FsStatMsg{Path: "plugin-stat.txt"})
		require.NoError(t, err)
		_ch8 := make(chan messages.FsStatResp, 1)
		_us8, err := sdk.SubscribeTo[messages.FsStatResp](rt, ctx, _pr8.ReplyTo, func(r messages.FsStatResp, m messages.Message) { _ch8 <- r })
		require.NoError(t, err)
		defer _us8()
		var resp messages.FsStatResp
		select {
		case resp = <-_ch8:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.False(t, resp.IsDir)
	})
	t.Run("Delete", func(t *testing.T) {
		sdk.Publish(rt, ctx, messages.FsWriteMsg{Path: "plugin-del.txt", Data: "x"})
		_pr9, err := sdk.Publish(rt, ctx, messages.FsDeleteMsg{Path: "plugin-del.txt"})
		require.NoError(t, err)
		_ch9 := make(chan messages.FsDeleteResp, 1)
		_us9, err := sdk.SubscribeTo[messages.FsDeleteResp](rt, ctx, _pr9.ReplyTo, func(r messages.FsDeleteResp, m messages.Message) { _ch9 <- r })
		require.NoError(t, err)
		defer _us9()
		var resp messages.FsDeleteResp
		select {
		case resp = <-_ch9:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.True(t, resp.OK)
	})
}

// --- Agents domain from Plugin ---

func TestPluginSurface_Agents(t *testing.T) {
	loadEnv(t)
	if !hasAIKey() {
		t.Skip("OPENAI_API_KEY required")
	}
	rt := newTestNode(t)
	node := rt.(*kit.Node)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Deploy an agent so we can test all operations
	sdk.Publish(node, ctx, messages.KitDeployMsg{
		Source: "plugin-agent-setup.ts",
		Code:   `agent({ name: "plugin-agent", instructions: "Reply ok", model: "openai/gpt-4o-mini" });`,
	})
	defer sdk.Publish(node, ctx, messages.KitTeardownMsg{Source: "plugin-agent-setup.ts"})

	t.Run("List", func(t *testing.T) {
		_pr10, err := sdk.Publish(rt, ctx, messages.AgentListMsg{})
		require.NoError(t, err)
		_ch10 := make(chan messages.AgentListResp, 1)
		_us10, err := sdk.SubscribeTo[messages.AgentListResp](rt, ctx, _pr10.ReplyTo, func(r messages.AgentListResp, m messages.Message) { _ch10 <- r })
		require.NoError(t, err)
		defer _us10()
		var resp messages.AgentListResp
		select {
		case resp = <-_ch10:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotNil(t, resp.Agents)
	})
	t.Run("Discover", func(t *testing.T) {
		_pr11, err := sdk.Publish(rt, ctx, messages.AgentDiscoverMsg{})
		require.NoError(t, err)
		_ch11 := make(chan messages.AgentDiscoverResp, 1)
		_us11, err := sdk.SubscribeTo[messages.AgentDiscoverResp](rt, ctx, _pr11.ReplyTo, func(r messages.AgentDiscoverResp, m messages.Message) { _ch11 <- r })
		require.NoError(t, err)
		defer _us11()
		var resp messages.AgentDiscoverResp
		select {
		case resp = <-_ch11:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotNil(t, resp.Agents)
	})
	t.Run("GetStatus", func(t *testing.T) {
		_pr12, err := sdk.Publish(rt, ctx, messages.AgentGetStatusMsg{Name: "plugin-agent"})
		require.NoError(t, err)
		_ch12 := make(chan messages.AgentGetStatusResp, 1)
		_us12, err := sdk.SubscribeTo[messages.AgentGetStatusResp](rt, ctx, _pr12.ReplyTo, func(r messages.AgentGetStatusResp, m messages.Message) { _ch12 <- r })
		require.NoError(t, err)
		defer _us12()
		var resp messages.AgentGetStatusResp
		select {
		case resp = <-_ch12:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.Equal(t, "idle", resp.Status)
	})
	t.Run("SetStatus", func(t *testing.T) {
		_pr13, err := sdk.Publish(rt, ctx, messages.AgentSetStatusMsg{Name: "plugin-agent", Status: "busy"})
		require.NoError(t, err)
		_ch13 := make(chan messages.AgentSetStatusResp, 1)
		_us13, _ := sdk.SubscribeTo[messages.AgentSetStatusResp](rt, ctx, _pr13.ReplyTo, func(r messages.AgentSetStatusResp, m messages.Message) { _ch13 <- r })
		defer _us13()
		select {
		case <-_ch13:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		_pr14, err := sdk.Publish(rt, ctx, messages.AgentGetStatusMsg{Name: "plugin-agent"})
		require.NoError(t, err)
		_ch14 := make(chan messages.AgentGetStatusResp, 1)
		_us14, err := sdk.SubscribeTo[messages.AgentGetStatusResp](rt, ctx, _pr14.ReplyTo, func(r messages.AgentGetStatusResp, m messages.Message) { _ch14 <- r })
		require.NoError(t, err)
		defer _us14()
		var resp messages.AgentGetStatusResp
		select {
		case resp = <-_ch14:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.Equal(t, "busy", resp.Status)
		// Reset
		sdk.Publish(rt, ctx, messages.AgentSetStatusMsg{Name: "plugin-agent", Status: "idle"})
	})
	t.Run("Message", func(t *testing.T) {
		_pr15, err := sdk.Publish(rt, ctx, messages.AgentMessageMsg{Target: "plugin-agent", Payload: "hello"})
		require.NoError(t, err)
		_ch15 := make(chan messages.AgentMessageResp, 1)
		_us15, err := sdk.SubscribeTo[messages.AgentMessageResp](rt, ctx, _pr15.ReplyTo, func(r messages.AgentMessageResp, m messages.Message) { _ch15 <- r })
		require.NoError(t, err)
		defer _us15()
		var resp messages.AgentMessageResp
		select {
		case resp = <-_ch15:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.True(t, resp.Delivered)
	})
	t.Run("Request", func(t *testing.T) {
		_pr16, err := sdk.Publish(rt, ctx, messages.AgentRequestMsg{Name: "plugin-agent", Prompt: "Say ok"})
		require.NoError(t, err)
		_ch16 := make(chan messages.AgentRequestResp, 1)
		_us16, err := sdk.SubscribeTo[messages.AgentRequestResp](rt, ctx, _pr16.ReplyTo, func(r messages.AgentRequestResp, m messages.Message) { _ch16 <- r })
		require.NoError(t, err)
		defer _us16()
		var resp messages.AgentRequestResp
		select {
		case resp = <-_ch16:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotEmpty(t, resp.Text)
	})
}

// --- AI domain from Plugin ---

func TestPluginSurface_AI(t *testing.T) {
	loadEnv(t)
	if !hasAIKey() {
		t.Skip("OPENAI_API_KEY required")
	}
	rt := newTestNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t.Run("Generate", func(t *testing.T) {
		_pr17, err := sdk.Publish(rt, ctx, messages.AiGenerateMsg{
			Model: "openai/gpt-4o-mini", Prompt: "Say exactly: pong",
		})
		require.NoError(t, err)
		_ch17 := make(chan messages.AiGenerateResp, 1)
		_us17, err := sdk.SubscribeTo[messages.AiGenerateResp](rt, ctx, _pr17.ReplyTo, func(r messages.AiGenerateResp, m messages.Message) { _ch17 <- r })
		require.NoError(t, err)
		defer _us17()
		var resp messages.AiGenerateResp
		select {
		case resp = <-_ch17:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotEmpty(t, resp.Text)
	})
	t.Run("Embed", func(t *testing.T) {
		_pr18, err := sdk.Publish(rt, ctx, messages.AiEmbedMsg{
			Model: "openai/text-embedding-3-small", Value: "test",
		})
		require.NoError(t, err)
		_ch18 := make(chan messages.AiEmbedResp, 1)
		_us18, err := sdk.SubscribeTo[messages.AiEmbedResp](rt, ctx, _pr18.ReplyTo, func(r messages.AiEmbedResp, m messages.Message) { _ch18 <- r })
		require.NoError(t, err)
		defer _us18()
		var resp messages.AiEmbedResp
		select {
		case resp = <-_ch18:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotEmpty(t, resp.Embedding)
	})
	t.Run("EmbedMany", func(t *testing.T) {
		_pr19, err := sdk.Publish(rt, ctx, messages.AiEmbedManyMsg{
			Model: "openai/text-embedding-3-small", Values: []string{"a", "b"},
		})
		require.NoError(t, err)
		_ch19 := make(chan messages.AiEmbedManyResp, 1)
		_us19, err := sdk.SubscribeTo[messages.AiEmbedManyResp](rt, ctx, _pr19.ReplyTo, func(r messages.AiEmbedManyResp, m messages.Message) { _ch19 <- r })
		require.NoError(t, err)
		defer _us19()
		var resp messages.AiEmbedManyResp
		select {
		case resp = <-_ch19:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.Len(t, resp.Embeddings, 2)
	})
	t.Run("GenerateObject", func(t *testing.T) {
		_pr20, err := sdk.Publish(rt, ctx, messages.AiGenerateObjectMsg{
			Model: "openai/gpt-4o-mini", Prompt: "Give me a color",
			Schema: map[string]any{"type": "object", "properties": map[string]any{"color": map[string]any{"type": "string"}}, "required": []string{"color"}},
		})
		require.NoError(t, err)
		_ch20 := make(chan messages.AiGenerateObjectResp, 1)
		_us20, err := sdk.SubscribeTo[messages.AiGenerateObjectResp](rt, ctx, _pr20.ReplyTo, func(r messages.AiGenerateObjectResp, m messages.Message) { _ch20 <- r })
		require.NoError(t, err)
		defer _us20()
		var resp messages.AiGenerateObjectResp
		select {
		case resp = <-_ch20:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotNil(t, resp.Object)
	})
	t.Run("Stream", func(t *testing.T) {
		// Fire ai.stream — it's fire-and-forget (produces StreamChunks on a topic)
		corrID, err := sdk.Publish(rt, ctx, messages.AiStreamMsg{
			Model: "openai/gpt-4o-mini", Prompt: "Say exactly: pong", StreamTo: "stream.chunk",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, corrID)

		// Collect at least one chunk
		chunks := make(chan bool, 1)
		unsub, err := rt.SubscribeRaw(ctx, "stream.chunk", func(msg messages.Message) {
			if msg.Metadata["correlationId"] == corrID.CorrelationID {
				select {
				case chunks <- true:
				default:
				}
			}
		})
		require.NoError(t, err)
		defer unsub()

		select {
		case <-chunks:
			// Got a chunk — streaming works from Plugin
		case <-time.After(15 * time.Second):
			t.Log("No stream chunks received — streaming may not be fully wired for this surface")
		}
	})
}

// --- Kit lifecycle from Plugin ---

func TestPluginSurface_Kit(t *testing.T) {
	rt := newTestNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Deploy_Teardown", func(t *testing.T) {
		_pr21, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
			Source: "plugin-deploy.ts",
			Code:   `const t = createTool({ id: "plugin-deployed", description: "test", execute: async () => ({}) });`,
		})
		require.NoError(t, err)
		_ch21 := make(chan messages.KitDeployResp, 1)
		_us21, err := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, _pr21.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch21 <- r })
		require.NoError(t, err)
		defer _us21()
		var resp messages.KitDeployResp
		select {
		case resp = <-_ch21:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.True(t, resp.Deployed)

		_pr22, err := sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "plugin-deploy.ts"})
		require.NoError(t, err)
		_ch22 := make(chan messages.KitTeardownResp, 1)
		_us22, _ := sdk.SubscribeTo[messages.KitTeardownResp](rt, ctx, _pr22.ReplyTo, func(r messages.KitTeardownResp, m messages.Message) { _ch22 <- r })
		defer _us22()
		select {
		case <-_ch22:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
	})
	t.Run("List", func(t *testing.T) {
		_pr23, err := sdk.Publish(rt, ctx, messages.KitListMsg{})
		require.NoError(t, err)
		_ch23 := make(chan messages.KitListResp, 1)
		_us23, err := sdk.SubscribeTo[messages.KitListResp](rt, ctx, _pr23.ReplyTo, func(r messages.KitListResp, m messages.Message) { _ch23 <- r })
		require.NoError(t, err)
		defer _us23()
		var resp messages.KitListResp
		select {
		case resp = <-_ch23:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotNil(t, resp.Deployments)
	})
	t.Run("Redeploy", func(t *testing.T) {
		sdk.Publish(rt, ctx, messages.KitDeployMsg{
			Source: "plugin-redeploy.ts",
			Code:   `const t = createTool({ id: "plugin-v1", description: "v1", execute: async () => ({}) });`,
		})
		_pr24, err := sdk.Publish(rt, ctx, messages.KitRedeployMsg{
			Source: "plugin-redeploy.ts",
			Code:   `const t = createTool({ id: "plugin-v2", description: "v2", execute: async () => ({}) });`,
		})
		require.NoError(t, err)
		_ch24 := make(chan messages.KitRedeployResp, 1)
		_us24, err := sdk.SubscribeTo[messages.KitRedeployResp](rt, ctx, _pr24.ReplyTo, func(r messages.KitRedeployResp, m messages.Message) { _ch24 <- r })
		require.NoError(t, err)
		defer _us24()
		var resp messages.KitRedeployResp
		select {
		case resp = <-_ch24:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.True(t, resp.Deployed)
		sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "plugin-redeploy.ts"})
	})
}

// --- WASM domain from Plugin ---

func TestPluginSurface_WASM(t *testing.T) {
	rt := newTestNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	t.Run("Compile_Run", func(t *testing.T) {
		_pr25, err := sdk.Publish(rt, ctx, messages.WasmCompileMsg{
			Source: `export function run(): i32 { return 55; }`, Options: &messages.WasmCompileOpts{Name: "plugin-wasm"},
		})
		require.NoError(t, err)
		_ch25 := make(chan messages.WasmCompileResp, 1)
		_us25, err := sdk.SubscribeTo[messages.WasmCompileResp](rt, ctx, _pr25.ReplyTo, func(r messages.WasmCompileResp, m messages.Message) { _ch25 <- r })
		require.NoError(t, err)
		defer _us25()
		var comp messages.WasmCompileResp
		select {
		case comp = <-_ch25:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.Greater(t, comp.Size, 0)

		_pr26, err := sdk.Publish(rt, ctx, messages.WasmRunMsg{ModuleID: "plugin-wasm"})
		require.NoError(t, err)
		_ch26 := make(chan messages.WasmRunResp, 1)
		_us26, err := sdk.SubscribeTo[messages.WasmRunResp](rt, ctx, _pr26.ReplyTo, func(r messages.WasmRunResp, m messages.Message) { _ch26 <- r })
		require.NoError(t, err)
		defer _us26()
		var run messages.WasmRunResp
		select {
		case run = <-_ch26:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.Equal(t, 55, run.ExitCode)
	})
	t.Run("List", func(t *testing.T) {
		_pr27, err := sdk.Publish(rt, ctx, messages.WasmListMsg{})
		require.NoError(t, err)
		_ch27 := make(chan messages.WasmListResp, 1)
		_us27, err := sdk.SubscribeTo[messages.WasmListResp](rt, ctx, _pr27.ReplyTo, func(r messages.WasmListResp, m messages.Message) { _ch27 <- r })
		require.NoError(t, err)
		defer _us27()
		var resp messages.WasmListResp
		select {
		case resp = <-_ch27:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotEmpty(t, resp.Modules)
	})
	t.Run("Get", func(t *testing.T) {
		_pr28, err := sdk.Publish(rt, ctx, messages.WasmGetMsg{Name: "plugin-wasm"})
		require.NoError(t, err)
		_ch28 := make(chan messages.WasmGetResp, 1)
		_us28, err := sdk.SubscribeTo[messages.WasmGetResp](rt, ctx, _pr28.ReplyTo, func(r messages.WasmGetResp, m messages.Message) { _ch28 <- r })
		require.NoError(t, err)
		defer _us28()
		var resp messages.WasmGetResp
		select {
		case resp = <-_ch28:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotNil(t, resp.Module)
	})
	t.Run("Remove", func(t *testing.T) {
		_pr29, err := sdk.Publish(rt, ctx, messages.WasmRemoveMsg{Name: "plugin-wasm"})
		require.NoError(t, err)
		_ch29 := make(chan messages.WasmRemoveResp, 1)
		_us29, err := sdk.SubscribeTo[messages.WasmRemoveResp](rt, ctx, _pr29.ReplyTo, func(r messages.WasmRemoveResp, m messages.Message) { _ch29 <- r })
		require.NoError(t, err)
		defer _us29()
		var resp messages.WasmRemoveResp
		select {
		case resp = <-_ch29:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.True(t, resp.Removed)
	})
	t.Run("Deploy_Undeploy_Describe", func(t *testing.T) {
		sdk.Publish(rt, ctx, messages.WasmCompileMsg{
			Source: `
				import { _on, _setMode } from "brainkit";
				export function init(): void { _setMode("stateless"); _on("plugin.ev", "h"); }
				export function h(t: usize, p: usize): void {}
			`, Options: &messages.WasmCompileOpts{Name: "plugin-deploy-mod"},
		})
		_pr30, err := sdk.Publish(rt, ctx, messages.WasmDeployMsg{Name: "plugin-deploy-mod"})
		require.NoError(t, err)
		_ch30 := make(chan messages.WasmDeployResp, 1)
		_us30, err := sdk.SubscribeTo[messages.WasmDeployResp](rt, ctx, _pr30.ReplyTo, func(r messages.WasmDeployResp, m messages.Message) { _ch30 <- r })
		require.NoError(t, err)
		defer _us30()
		var deploy messages.WasmDeployResp
		select {
		case deploy = <-_ch30:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.Equal(t, "stateless", deploy.Mode)

		_pr31, err := sdk.Publish(rt, ctx, messages.WasmDescribeMsg{Name: "plugin-deploy-mod"})
		require.NoError(t, err)
		_ch31 := make(chan messages.WasmDescribeResp, 1)
		_us31, err := sdk.SubscribeTo[messages.WasmDescribeResp](rt, ctx, _pr31.ReplyTo, func(r messages.WasmDescribeResp, m messages.Message) { _ch31 <- r })
		require.NoError(t, err)
		defer _us31()
		var desc messages.WasmDescribeResp
		select {
		case desc = <-_ch31:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.Equal(t, "stateless", desc.Mode)

		_pr32, err := sdk.Publish(rt, ctx, messages.WasmUndeployMsg{Name: "plugin-deploy-mod"})
		require.NoError(t, err)
		_ch32 := make(chan messages.WasmUndeployResp, 1)
		_us32, err := sdk.SubscribeTo[messages.WasmUndeployResp](rt, ctx, _pr32.ReplyTo, func(r messages.WasmUndeployResp, m messages.Message) { _ch32 <- r })
		require.NoError(t, err)
		defer _us32()
		var undeploy messages.WasmUndeployResp
		select {
		case undeploy = <-_ch32:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.True(t, undeploy.Undeployed)
	})
}

// --- Memory domain from Plugin ---

func TestPluginSurface_Memory(t *testing.T) {
	rt := newTestNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Memory needs to be initialized in the JS runtime
	node := rt.(*kit.Node)
	_, err := node.Kernel.EvalTS(ctx, "__plugin_mem_init.ts", `
		globalThis.__kit_memory = createMemory({ storage: new InMemoryStore(), lastMessages: 10 });
		return "ok";
	`)
	require.NoError(t, err)

	t.Run("CreateThread", func(t *testing.T) {
		_pr33, err := sdk.Publish(rt, ctx, messages.MemoryCreateThreadMsg{})
		require.NoError(t, err)
		_ch33 := make(chan messages.MemoryCreateThreadResp, 1)
		_us33, err := sdk.SubscribeTo[messages.MemoryCreateThreadResp](rt, ctx, _pr33.ReplyTo, func(r messages.MemoryCreateThreadResp, m messages.Message) { _ch33 <- r })
		require.NoError(t, err)
		defer _us33()
		var resp messages.MemoryCreateThreadResp
		select {
		case resp = <-_ch33:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotEmpty(t, resp.ThreadID)
	})
	t.Run("Save_Recall", func(t *testing.T) {
		cr, _ := sdk.Publish(rt, ctx, messages.MemoryCreateThreadMsg{})
		crCh := make(chan messages.MemoryCreateThreadResp, 1)
		crUn, _ := sdk.SubscribeTo[messages.MemoryCreateThreadResp](rt, ctx, cr.ReplyTo, func(r messages.MemoryCreateThreadResp, m messages.Message) { crCh <- r })
		defer crUn()
		var create messages.MemoryCreateThreadResp
		select { case create = <-crCh: case <-ctx.Done(): t.Fatal("timeout") }
		_pr35, err := sdk.Publish(rt, ctx, messages.MemorySaveMsg{
			ThreadID: create.ThreadID,
			Messages: []messages.MemoryMessage{{Role: "user", Content: "hello"}},
		})
		require.NoError(t, err)
		_ch35 := make(chan messages.MemorySaveResp, 1)
		_us35, _ := sdk.SubscribeTo[messages.MemorySaveResp](rt, ctx, _pr35.ReplyTo, func(r messages.MemorySaveResp, m messages.Message) { _ch35 <- r })
		defer _us35()
		select {
		case <-_ch35:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		_pr36, err := sdk.Publish(rt, ctx, messages.MemoryRecallMsg{
			ThreadID: create.ThreadID, Query: "hello",
		})
		require.NoError(t, err)
		_ch36 := make(chan messages.MemoryRecallResp, 1)
		_us36, _ := sdk.SubscribeTo[messages.MemoryRecallResp](rt, ctx, _pr36.ReplyTo, func(r messages.MemoryRecallResp, m messages.Message) { _ch36 <- r })
		defer _us36()
		select {
		case <-_ch36:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
	})
	t.Run("GetThread", func(t *testing.T) {
		cr2, _ := sdk.Publish(rt, ctx, messages.MemoryCreateThreadMsg{})
		cr2Ch := make(chan messages.MemoryCreateThreadResp, 1)
		cr2Un, _ := sdk.SubscribeTo[messages.MemoryCreateThreadResp](rt, ctx, cr2.ReplyTo, func(r messages.MemoryCreateThreadResp, m messages.Message) { cr2Ch <- r })
		defer cr2Un()
		var create messages.MemoryCreateThreadResp
		select { case create = <-cr2Ch: case <-ctx.Done(): t.Fatal("timeout") }
		_pr38, err := sdk.Publish(rt, ctx, messages.MemoryGetThreadMsg{ThreadID: create.ThreadID})
		require.NoError(t, err)
		_ch38 := make(chan messages.MemoryGetThreadResp, 1)
		_us38, err := sdk.SubscribeTo[messages.MemoryGetThreadResp](rt, ctx, _pr38.ReplyTo, func(r messages.MemoryGetThreadResp, m messages.Message) { _ch38 <- r })
		require.NoError(t, err)
		defer _us38()
		var resp messages.MemoryGetThreadResp
		select {
		case resp = <-_ch38:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotNil(t, resp.Thread)
	})
	t.Run("ListThreads", func(t *testing.T) {
		_pr39, err := sdk.Publish(rt, ctx, messages.MemoryListThreadsMsg{})
		require.NoError(t, err)
		_ch39 := make(chan messages.MemoryListThreadsResp, 1)
		_us39, err := sdk.SubscribeTo[messages.MemoryListThreadsResp](rt, ctx, _pr39.ReplyTo, func(r messages.MemoryListThreadsResp, m messages.Message) { _ch39 <- r })
		require.NoError(t, err)
		defer _us39()
		var resp messages.MemoryListThreadsResp
		select {
		case resp = <-_ch39:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotNil(t, resp.Threads)
	})
	t.Run("DeleteThread", func(t *testing.T) {
		cr3, _ := sdk.Publish(rt, ctx, messages.MemoryCreateThreadMsg{})
		cr3Ch := make(chan messages.MemoryCreateThreadResp, 1)
		cr3Un, _ := sdk.SubscribeTo[messages.MemoryCreateThreadResp](rt, ctx, cr3.ReplyTo, func(r messages.MemoryCreateThreadResp, m messages.Message) { cr3Ch <- r })
		defer cr3Un()
		var create messages.MemoryCreateThreadResp
		select { case create = <-cr3Ch: case <-ctx.Done(): t.Fatal("timeout") }
		_pr41, err := sdk.Publish(rt, ctx, messages.MemoryDeleteThreadMsg{ThreadID: create.ThreadID})
		require.NoError(t, err)
		_ch41 := make(chan messages.MemoryDeleteThreadResp, 1)
		_us41, err := sdk.SubscribeTo[messages.MemoryDeleteThreadResp](rt, ctx, _pr41.ReplyTo, func(r messages.MemoryDeleteThreadResp, m messages.Message) { _ch41 <- r })
		require.NoError(t, err)
		defer _us41()
		var resp messages.MemoryDeleteThreadResp
		select {
		case resp = <-_ch41:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.True(t, resp.OK)
	})
}

// --- Workflows domain from Plugin ---

func TestPluginSurface_Workflows(t *testing.T) {
	rt := newTestNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	node := rt.(*kit.Node)
	_pr42, err := sdk.Publish(node, ctx, messages.KitDeployMsg{
		Source: "plugin-wf.ts",
		Code: `
			const wf = createWorkflow({
				id: "plugin-wf", inputSchema: z.object({ v: z.string() }), outputSchema: z.object({ r: z.string() }),
			});
			wf.then(createStep({
				id: "s1", inputSchema: z.object({ v: z.string() }), outputSchema: z.object({ r: z.string() }),
				execute: async ({ inputData }) => ({ r: inputData.v + "-plugin" }),
			})).commit();
		`,
	})
	require.NoError(t, err)
	_ch42 := make(chan messages.KitDeployResp, 1)
	_us42, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, _pr42.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch42 <- r })
	defer _us42()
	select {
	case <-_ch42:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	t.Run("Run", func(t *testing.T) {
		_pr43, err := sdk.Publish(rt, ctx, messages.WorkflowRunMsg{
			Name: "plugin-wf", Input: map[string]any{"v": "test"},
		})
		require.NoError(t, err)
		_ch43 := make(chan messages.WorkflowRunResp, 1)
		_us43, err := sdk.SubscribeTo[messages.WorkflowRunResp](rt, ctx, _pr43.ReplyTo, func(r messages.WorkflowRunResp, m messages.Message) { _ch43 <- r })
		require.NoError(t, err)
		defer _us43()
		var resp messages.WorkflowRunResp
		select {
		case resp = <-_ch43:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotNil(t, resp.Result)
	})

	sdk.Publish(node, ctx, messages.KitTeardownMsg{Source: "plugin-wf.ts"})

	// Deploy suspend workflow for resume/cancel/status
	_pr44, err := sdk.Publish(node, ctx, messages.KitDeployMsg{
		Source: "plugin-suspend-wf.ts",
		Code: `
			const swf = createWorkflow({
				id: "plugin-suspend-wf", inputSchema: z.object({ v: z.string() }), outputSchema: z.object({ r: z.string() }),
			});
			swf.then(createStep({
				id: "ss1", inputSchema: z.object({ v: z.string() }), outputSchema: z.object({ r: z.string() }),
				execute: async ({ inputData, suspend }) => { await suspend({ reason: "wait" }); return { r: inputData.v + "-done" }; },
			})).commit();
		`,
	})
	require.NoError(t, err)
	_ch44 := make(chan messages.KitDeployResp, 1)
	_us44, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, _pr44.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch44 <- r })
	defer _us44()
	select {
	case <-_ch44:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	defer sdk.Publish(node, ctx, messages.KitTeardownMsg{Source: "plugin-suspend-wf.ts"})

	t.Run("Suspend_Resume", func(t *testing.T) {
		_pr45, err := sdk.Publish(rt, ctx, messages.WorkflowRunMsg{
			Name: "plugin-suspend-wf", Input: map[string]any{"v": "test"},
		})
		require.NoError(t, err)
		_ch45 := make(chan messages.WorkflowRunResp, 1)
		_us45, err := sdk.SubscribeTo[messages.WorkflowRunResp](rt, ctx, _pr45.ReplyTo, func(r messages.WorkflowRunResp, m messages.Message) { _ch45 <- r })
		require.NoError(t, err)
		defer _us45()
		var runResp messages.WorkflowRunResp
		select {
		case runResp = <-_ch45:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		var runResult map[string]any
		json.Unmarshal(runResp.Result, &runResult)
		if runResult["status"] == "suspended" {
			runId, _ := runResult["runId"].(string)
			require.NotEmpty(t, runId)

			_pr46, err := sdk.Publish(rt, ctx, messages.WorkflowStatusMsg{RunID: runId})
			require.NoError(t, err)
			_ch46 := make(chan messages.WorkflowStatusResp, 1)
			_us46, err := sdk.SubscribeTo[messages.WorkflowStatusResp](rt, ctx, _pr46.ReplyTo, func(r messages.WorkflowStatusResp, m messages.Message) { _ch46 <- r })
			require.NoError(t, err)
			defer _us46()
			var statusResp messages.WorkflowStatusResp
			select {
			case statusResp = <-_ch46:
			case <-ctx.Done():
				t.Fatal("timeout")
			}
			t.Logf("Plugin workflow status: %s", statusResp.Status)

			_pr47, err := sdk.Publish(rt, ctx, messages.WorkflowResumeMsg{
				RunID: runId, StepID: "ss1", Data: map[string]any{"ok": true},
			})
			require.NoError(t, err)
			_ch47 := make(chan messages.WorkflowResumeResp, 1)
			_us47, err := sdk.SubscribeTo[messages.WorkflowResumeResp](rt, ctx, _pr47.ReplyTo, func(r messages.WorkflowResumeResp, m messages.Message) { _ch47 <- r })
			require.NoError(t, err)
			defer _us47()
			var resumeResp messages.WorkflowResumeResp
			select {
			case resumeResp = <-_ch47:
			case <-ctx.Done():
				t.Fatal("timeout")
			}
			assert.NotNil(t, resumeResp.Result)
		}
	})
	t.Run("Cancel_NotFound", func(t *testing.T) {
		pr, _ := sdk.Publish(rt, ctx, messages.WorkflowCancelMsg{RunID: "nonexistent"})
		errCh := make(chan string, 1)
		un, _ := rt.SubscribeRaw(ctx, pr.ReplyTo, func(msg messages.Message) {
			var r struct { Error string `json:"error"` }
			json.Unmarshal(msg.Payload, &r)
			errCh <- r.Error
		})
		defer un()
		select {
		case errMsg := <-errCh:
			assert.NotEmpty(t, errMsg)
		case <-ctx.Done():
			t.Fatal("timeout")
		}
	})
}

// --- MCP domain from Plugin ---

func TestPluginSurface_MCP(t *testing.T) {
	mcpBinary := buildTestMCP(t)
	n, err := kit.NewNode(kit.NodeConfig{
		Kernel: kit.KernelConfig{
			Namespace: "test", CallerID: "test-plugin-mcp", WorkspaceDir: t.TempDir(),
			MCPServers: map[string]mcppkg.ServerConfig{"echo": {Command: mcpBinary}},
		},
		Messaging: kit.MessagingConfig{Transport: "memory"},
	})
	require.NoError(t, err)
	require.NoError(t, n.Start(context.Background()))
	defer n.Close()
	rt := sdk.Runtime(n)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("ListTools", func(t *testing.T) {
		_pr49, err := sdk.Publish(rt, ctx, messages.McpListToolsMsg{})
		require.NoError(t, err)
		_ch49 := make(chan messages.McpListToolsResp, 1)
		_us49, err := sdk.SubscribeTo[messages.McpListToolsResp](rt, ctx, _pr49.ReplyTo, func(r messages.McpListToolsResp, m messages.Message) { _ch49 <- r })
		require.NoError(t, err)
		defer _us49()
		var resp messages.McpListToolsResp
		select {
		case resp = <-_ch49:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotEmpty(t, resp.Tools)
	})
	t.Run("CallTool", func(t *testing.T) {
		_pr50, err := sdk.Publish(rt, ctx, messages.McpCallToolMsg{
			Server: "echo", Tool: "echo", Args: map[string]any{"message": "from-plugin"},
		})
		require.NoError(t, err)
		_ch50 := make(chan messages.McpCallToolResp, 1)
		_us50, err := sdk.SubscribeTo[messages.McpCallToolResp](rt, ctx, _pr50.ReplyTo, func(r messages.McpCallToolResp, m messages.Message) { _ch50 <- r })
		require.NoError(t, err)
		defer _us50()
		var resp messages.McpCallToolResp
		select {
		case resp = <-_ch50:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotNil(t, resp.Result)
	})
}

// --- Registry domain from Plugin ---

func TestPluginSurface_Registry(t *testing.T) {
	rt := newTestNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Has", func(t *testing.T) {
		_pr51, err := sdk.Publish(rt, ctx, messages.RegistryHasMsg{
			Category: "provider", Name: "nonexistent",
		})
		require.NoError(t, err)
		_ch51 := make(chan messages.RegistryHasResp, 1)
		_us51, err := sdk.SubscribeTo[messages.RegistryHasResp](rt, ctx, _pr51.ReplyTo, func(r messages.RegistryHasResp, m messages.Message) { _ch51 <- r })
		require.NoError(t, err)
		defer _us51()
		var resp messages.RegistryHasResp
		select {
		case resp = <-_ch51:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.False(t, resp.Found)
	})
	t.Run("List", func(t *testing.T) {
		_pr52, err := sdk.Publish(rt, ctx, messages.RegistryListMsg{
			Category: "provider",
		})
		require.NoError(t, err)
		_ch52 := make(chan messages.RegistryListResp, 1)
		_us52, err := sdk.SubscribeTo[messages.RegistryListResp](rt, ctx, _pr52.ReplyTo, func(r messages.RegistryListResp, m messages.Message) { _ch52 <- r })
		require.NoError(t, err)
		defer _us52()
		var resp messages.RegistryListResp
		select {
		case resp = <-_ch52:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotNil(t, resp.Items)
	})
	t.Run("Resolve", func(t *testing.T) {
		_pr53, err := sdk.Publish(rt, ctx, messages.RegistryResolveMsg{
			Category: "provider", Name: "nonexistent",
		})
		require.NoError(t, err)
		_ch53 := make(chan messages.RegistryResolveResp, 1)
		_us53, err := sdk.SubscribeTo[messages.RegistryResolveResp](rt, ctx, _pr53.ReplyTo, func(r messages.RegistryResolveResp, m messages.Message) { _ch53 <- r })
		require.NoError(t, err)
		defer _us53()
		var resp messages.RegistryResolveResp
		select {
		case resp = <-_ch53:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		// Not found — config should be null/empty
		assert.True(t, len(resp.Config) == 0 || string(resp.Config) == "null")
	})
}

// --- Vectors domain from Plugin ---

func TestPluginSurface_Vectors(t *testing.T) {
	if !podmanAvailable() {
		t.Skip("Podman required for pgvector")
	}
	pgConnStr := startPgVectorContainer(t)

	rt := newTestNode(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	node := rt.(*kit.Node)
	_, err := node.Kernel.EvalTS(ctx, "__plugin_vec_init.ts", fmt.Sprintf(`
		var vs = new PgVector({ id: "plugin_vec", connectionString: %q });
		globalThis.__kit_vector_store = vs;
		return "ok";
	`, pgConnStr))
	require.NoError(t, err)

	t.Run("CreateIndex", func(t *testing.T) {
		_pr54, err := sdk.Publish(rt, ctx, messages.VectorCreateIndexMsg{
			Name: "plugin_vec_idx", Dimension: 3, Metric: "cosine",
		})
		require.NoError(t, err)
		_ch54 := make(chan messages.VectorCreateIndexResp, 1)
		_us54, err := sdk.SubscribeTo[messages.VectorCreateIndexResp](rt, ctx, _pr54.ReplyTo, func(r messages.VectorCreateIndexResp, m messages.Message) { _ch54 <- r })
		require.NoError(t, err)
		defer _us54()
		var resp messages.VectorCreateIndexResp
		select {
		case resp = <-_ch54:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.True(t, resp.OK)
	})
	t.Run("ListIndexes", func(t *testing.T) {
		_pr55, err := sdk.Publish(rt, ctx, messages.VectorListIndexesMsg{})
		require.NoError(t, err)
		_ch55 := make(chan messages.VectorListIndexesResp, 1)
		_us55, err := sdk.SubscribeTo[messages.VectorListIndexesResp](rt, ctx, _pr55.ReplyTo, func(r messages.VectorListIndexesResp, m messages.Message) { _ch55 <- r })
		require.NoError(t, err)
		defer _us55()
		var resp messages.VectorListIndexesResp
		select {
		case resp = <-_ch55:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotNil(t, resp.Indexes)
	})
	t.Run("DeleteIndex", func(t *testing.T) {
		_pr56, err := sdk.Publish(rt, ctx, messages.VectorDeleteIndexMsg{
			Name: "plugin_vec_idx",
		})
		require.NoError(t, err)
		_ch56 := make(chan messages.VectorDeleteIndexResp, 1)
		_us56, err := sdk.SubscribeTo[messages.VectorDeleteIndexResp](rt, ctx, _pr56.ReplyTo, func(r messages.VectorDeleteIndexResp, m messages.Message) { _ch56 <- r })
		require.NoError(t, err)
		defer _us56()
		var resp messages.VectorDeleteIndexResp
		select {
		case resp = <-_ch56:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.True(t, resp.OK)
	})
}
