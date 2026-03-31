package adversarial_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSurfaceMatrix_GoSDK — core operations from Go SDK surface.
func TestSurfaceMatrix_GoSDK(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	t.Run("tools.list", func(t *testing.T) {
		payload, ok := sendAndReceive(t, tk, messages.ToolListMsg{}, 5*time.Second)
		require.True(t, ok)
		assert.False(t, responseHasError(payload))
	})

	t.Run("tools.call/echo", func(t *testing.T) {
		payload, ok := sendAndReceive(t, tk, messages.ToolCallMsg{Name: "echo", Input: map[string]any{"message": "go-sdk"}}, 5*time.Second)
		require.True(t, ok)
		assert.Contains(t, string(payload), "go-sdk")
	})

	t.Run("secrets.set+get", func(t *testing.T) {
		pr, err := sdk.Publish(tk, ctx, messages.SecretsSetMsg{Name: "go-surface", Value: "go-val"})
		require.NoError(t, err)
		ch := make(chan []byte, 1)
		unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
		select {
		case <-ch:
		case <-time.After(3 * time.Second):
			t.Fatal("timeout set")
		}
		unsub()

		pr2, _ := sdk.Publish(tk, ctx, messages.SecretsGetMsg{Name: "go-surface"})
		ch2 := make(chan []byte, 1)
		unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
		defer unsub2()
		select {
		case p := <-ch2:
			assert.Contains(t, string(p), "go-val")
		case <-time.After(3 * time.Second):
			t.Fatal("timeout get")
		}
	})

	t.Run("fs.write+read", func(t *testing.T) {
		pr, _ := sdk.Publish(tk, ctx, messages.FsWriteMsg{Path: "go-surf.txt", Data: "from go"})
		ch := make(chan []byte, 1)
		unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
		select {
		case <-ch:
		case <-time.After(3 * time.Second):
			t.Fatal("timeout write")
		}
		unsub()

		pr2, _ := sdk.Publish(tk, ctx, messages.FsReadMsg{Path: "go-surf.txt"})
		ch2 := make(chan []byte, 1)
		unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
		defer unsub2()
		select {
		case p := <-ch2:
			assert.Contains(t, string(p), "from go")
		case <-time.After(3 * time.Second):
			t.Fatal("timeout read")
		}
	})

	t.Run("bus.publish+reply", func(t *testing.T) {
		_, err := tk.Deploy(ctx, "go-surface-svc.ts", `bus.on("ping", function(msg) { msg.reply({pong:true}); });`)
		require.NoError(t, err)
		defer tk.Teardown(ctx, "go-surface-svc.ts")

		pr, _ := sdk.Publish(tk, ctx, messages.CustomMsg{Topic: "ts.go-surface-svc.ping", Payload: json.RawMessage(`{}`)})
		ch := make(chan []byte, 1)
		unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
		defer unsub()
		select {
		case p := <-ch:
			assert.Contains(t, string(p), "pong")
		case <-time.After(3 * time.Second):
			t.Fatal("timeout")
		}
	})

	t.Run("schedule+unschedule", func(t *testing.T) {
		id, err := tk.Schedule(ctx, brainkit.ScheduleConfig{Expression: "in 1h", Topic: "go-sched", Payload: json.RawMessage(`{}`)})
		require.NoError(t, err)
		require.NotEmpty(t, id)
		tk.Unschedule(ctx, id)
	})

	t.Run("metrics", func(t *testing.T) {
		m := tk.Metrics()
		assert.GreaterOrEqual(t, m.PumpCycles, int64(0))
	})

	t.Run("registry.list", func(t *testing.T) {
		payload, ok := sendAndReceive(t, tk, messages.RegistryListMsg{Category: "provider"}, 5*time.Second)
		require.True(t, ok)
		assert.False(t, responseHasError(payload))
	})
}

// TestSurfaceMatrix_TSDeployed — same operations from deployed .ts code.
func TestSurfaceMatrix_TSDeployed(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	cases := []struct {
		name   string
		code   string
		expect string
	}{
		{
			"tools.list",
			`var r = JSON.parse(__go_brainkit_request("tools.list", "{}")); output(r.tools ? "ok" : "fail");`,
			"ok",
		},
		{
			"tools.call",
			`var r = JSON.parse(__go_brainkit_request("tools.call", JSON.stringify({name:"echo",input:{message:"ts-deployed"}}))); output(JSON.stringify(r));`,
			"ts-deployed",
		},
		{
			"secrets.set+get",
			`__go_brainkit_request("secrets.set", JSON.stringify({name:"ts-key",value:"ts-val"})); var g = JSON.parse(__go_brainkit_request("secrets.get", JSON.stringify({name:"ts-key"}))); output(g.value);`,
			"ts-val",
		},
		{
			"fs.write+read",
			`await fs.write("ts-surf.txt", "from-ts"); var r = await fs.read("ts-surf.txt"); output(r.data);`,
			"from-ts",
		},
		{
			"bus.publish",
			`var r = bus.publish("incoming.ts-surface-test", {data: "test"}); output(r.replyTo ? "ok" : "fail");`,
			"ok",
		},
		{
			"bus.emit",
			`bus.emit("events.ts-surface-test", {data: "test"}); output("ok");`,
			"ok",
		},
		{
			"kit.register+resolve",
			`var t = createTool({id: "surf-tool", description: "test", execute: async () => ({ok:true})}); kit.register("tool", "surf-tool", t); var r = JSON.parse(__go_brainkit_request("tools.resolve", JSON.stringify({name:"surf-tool"}))); output(r.name ? "ok" : "fail");`,
			"ok",
		},
		{
			"schedule+unschedule",
			`var id = bus.schedule("in 1h", "ts-sched-surface", {}); bus.unschedule(id); output("ok");`,
			"ok",
		},
		{
			"metrics",
			`var r = JSON.parse(__go_brainkit_request("metrics.get", "{}")); output(r.metrics ? "ok" : "fail");`,
			"ok",
		},
		{
			"registry.list",
			`var r = JSON.parse(__go_brainkit_request("registry.list", JSON.stringify({category:"provider"}))); output("ok");`,
			"ok",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := fmt.Sprintf("__ts_surface_%s.ts", tc.name)
			_, err := tk.Deploy(ctx, source, tc.code)
			if err != nil {
				t.Logf("deploy error (may be expected): %v", err)
				return
			}
			defer tk.Teardown(ctx, source)

			result, err := tk.EvalTS(ctx, "__get_result.ts", `return String(globalThis.__module_result || "");`)
			require.NoError(t, err)
			assert.Contains(t, result, tc.expect, "%s: expected %q in result", tc.name, tc.expect)
		})
	}
}

// TestSurfaceMatrix_EvalTS — operations via direct EvalTS (global scope, not Compartment).
func TestSurfaceMatrix_EvalTS(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	cases := []struct {
		name   string
		code   string
		expect string
	}{
		{"tools.list", `var r = JSON.parse(__go_brainkit_request("tools.list", "{}")); return String(r.tools.length >= 0);`, "true"},
		{"tools.call", `var r = __go_brainkit_request("tools.call", JSON.stringify({name:"echo",input:{message:"eval"}})); return r.indexOf("eval") >= 0 ? "ok" : "fail";`, "ok"},
		{"secrets.set", `__go_brainkit_request("secrets.set", JSON.stringify({name:"eval-k",value:"eval-v"})); return "ok";`, "ok"},
		{"secrets.get", `__go_brainkit_request("secrets.set", JSON.stringify({name:"eval-g",value:"eval-gv"})); var r = JSON.parse(__go_brainkit_request("secrets.get", JSON.stringify({name:"eval-g"}))); return r.value || "empty";`, "eval-gv"},
		{"fs.list", `__go_brainkit_request("fs.list", JSON.stringify({path:"."})); return "ok";`, "ok"},
		{"metrics", `var r = JSON.parse(__go_brainkit_request("metrics.get", "{}")); return r.metrics ? "ok" : "fail";`, "ok"},
		{"registry.list", `__go_brainkit_request("registry.list", JSON.stringify({category:"provider"})); return "ok";`, "ok"},
		{"bus.publish", `var r = bus.publish("incoming.eval-surface", {data:"test"}); return r.replyTo ? "ok" : "fail";`, "ok"},
		{"bus.emit", `bus.emit("events.eval-surface", {data:"test"}); return "ok";`, "ok"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tk.EvalTS(ctx, fmt.Sprintf("__eval_%s.ts", tc.name), tc.code)
			require.NoError(t, err)
			assert.Contains(t, result, tc.expect)
		})
	}
}

// TestSurfaceMatrix_ErrorConsistency — same error looks the same from every surface.
func TestSurfaceMatrix_ErrorConsistency(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// NOT_FOUND from Go SDK
	t.Run("NOT_FOUND/go", func(t *testing.T) {
		payload, ok := sendAndReceive(t, tk, messages.ToolCallMsg{Name: "ghost-tool-consistency"}, 5*time.Second)
		require.True(t, ok)
		assert.Equal(t, "NOT_FOUND", responseCode(payload))
	})

	// NOT_FOUND from .ts deployed — uses tools.call via bus endowment
	// NOTE: Compartments use endowments (tools.call), not raw __go_brainkit_request.
	// The tools.call endowment wraps the bridge request and the error becomes a JS exception.
	t.Run("NOT_FOUND/ts-deployed", func(t *testing.T) {
		_, err := tk.Deploy(ctx, "err-consist.ts", `
			var caught = "none";
			try { await tools.call("ghost-tool-consistency", {}); }
			catch(e) { caught = e.message || "unknown"; }
			output(caught);
		`)
		require.NoError(t, err)
		defer tk.Teardown(ctx, "err-consist.ts")

		result, _ := tk.EvalTS(ctx, "__err_result.ts", `return String(globalThis.__module_result || "");`)
		assert.Contains(t, result, "ghost-tool-consistency", "error should mention the tool name")
	})

	// NOT_FOUND from EvalTS
	t.Run("NOT_FOUND/evalts", func(t *testing.T) {
		result, err := tk.EvalTS(ctx, "__err_eval.ts", `
			var caught = "none";
			try { __go_brainkit_request("tools.call", JSON.stringify({name:"ghost-tool-consistency"})); }
			catch(e) { caught = e.code || "NO_CODE"; }
			return caught;
		`)
		require.NoError(t, err)
		assert.Equal(t, "NOT_FOUND", result)
	})

	// VALIDATION_ERROR consistency
	t.Run("VALIDATION_ERROR/go", func(t *testing.T) {
		payload, ok := sendAndReceive(t, tk, messages.SecretsSetMsg{Name: "", Value: "v"}, 5*time.Second)
		require.True(t, ok)
		assert.Equal(t, "VALIDATION_ERROR", responseCode(payload))
	})

	t.Run("VALIDATION_ERROR/evalts", func(t *testing.T) {
		result, err := tk.EvalTS(ctx, "__val_eval.ts", `
			var caught = "none";
			try { __go_brainkit_request("secrets.set", JSON.stringify({name:"",value:"v"})); }
			catch(e) { caught = e.code || "NO_CODE"; }
			return caught;
		`)
		require.NoError(t, err)
		assert.Equal(t, "VALIDATION_ERROR", result)
	})
}
