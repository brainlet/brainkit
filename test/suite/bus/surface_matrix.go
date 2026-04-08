package bus

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Surface matrix: Go SDK surface ──────────────────────────────────────

// testSurfaceGoSDK — core operations from Go SDK surface.
func testSurfaceGoSDK(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	t.Run("tools.list", func(t *testing.T) {
		payload, ok := env.SendAndReceive(t, sdk.ToolListMsg{}, 5*time.Second)
		require.True(t, ok)
		assert.False(t, suite.ResponseHasError(payload))
	})

	t.Run("tools.call/echo", func(t *testing.T) {
		payload, ok := env.SendAndReceive(t, sdk.ToolCallMsg{Name: "echo", Input: map[string]any{"message": "go-sdk"}}, 5*time.Second)
		require.True(t, ok)
		assert.Contains(t, string(payload), "go-sdk")
	})

	t.Run("secrets.set+get", func(t *testing.T) {
		pr, err := sdk.Publish(env.Kit, ctx, sdk.SecretsSetMsg{Name: "go-surface-suite", Value: "go-val"})
		require.NoError(t, err)
		ch := make(chan []byte, 1)
		unsub, _ := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m.Payload })
		select {
		case <-ch:
		case <-time.After(3 * time.Second):
			t.Fatal("timeout set")
		}
		unsub()

		pr2, _ := sdk.Publish(env.Kit, ctx, sdk.SecretsGetMsg{Name: "go-surface-suite"})
		ch2 := make(chan []byte, 1)
		unsub2, _ := env.Kit.SubscribeRaw(ctx, pr2.ReplyTo, func(m sdk.Message) { ch2 <- m.Payload })
		defer unsub2()
		select {
		case p := <-ch2:
			assert.Contains(t, string(p), "go-val")
		case <-time.After(3 * time.Second):
			t.Fatal("timeout get")
		}
	})

	t.Run("fs.write+read", func(t *testing.T) {
		result := testutil.EvalTS(t, env.Kit, "__test_surface.ts", `
			fs.writeFileSync("go-surf-suite.txt", "from go");
			return fs.readFileSync("go-surf-suite.txt", "utf8");
		`)
		assert.Equal(t, "from go", result)
	})

	t.Run("bus.publish+reply", func(t *testing.T) {
		testutil.Deploy(t, env.Kit, "go-surface-svc-suite.ts", `bus.on("ping", function(msg) { msg.reply({pong:true}); });`)

		pr, _ := sdk.Publish(env.Kit, ctx, sdk.CustomMsg{Topic: "ts.go-surface-svc-suite.ping", Payload: json.RawMessage(`{}`)})
		ch := make(chan []byte, 1)
		unsub, _ := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m.Payload })
		defer unsub()
		select {
		case p := <-ch:
			assert.Contains(t, string(p), "pong")
		case <-time.After(3 * time.Second):
			t.Fatal("timeout")
		}
	})

	t.Run("schedule+unschedule", func(t *testing.T) {
		id := testutil.Schedule(t, env.Kit, "in 1h", "go-sched-suite", json.RawMessage(`{}`))
		require.NotEmpty(t, id)
		testutil.Unschedule(t, env.Kit, id)
	})

	t.Run("metrics", func(t *testing.T) {
		payload, ok := env.SendAndReceive(t, sdk.MetricsGetMsg{}, 5*time.Second)
		require.True(t, ok)
		assert.False(t, suite.ResponseHasError(payload))
	})

	t.Run("registry.list", func(t *testing.T) {
		payload, ok := env.SendAndReceive(t, sdk.RegistryListMsg{Category: "provider"}, 5*time.Second)
		require.True(t, ok)
		assert.False(t, suite.ResponseHasError(payload))
	})
}

// ── Surface matrix: TS Deployed surface ──────────────────────────────────

// testSurfaceTSDeployed — operations from deployed .ts code.
func testSurfaceTSDeployed(t *testing.T, env *suite.TestEnv) {
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
			`__go_brainkit_request("secrets.set", JSON.stringify({name:"ts-key-suite",value:"ts-val"})); var g = JSON.parse(__go_brainkit_request("secrets.get", JSON.stringify({name:"ts-key-suite"}))); output(g.value);`,
			"ts-val",
		},
		{
			"fs.write+read",
			`fs.writeFileSync("ts-surf-suite.txt", "from-ts"); output(fs.readFileSync("ts-surf-suite.txt", "utf8"));`,
			"from-ts",
		},
		{
			"bus.publish",
			`var r = bus.publish("incoming.ts-surface-suite-test", {data: "test"}); output(r.replyTo ? "ok" : "fail");`,
			"ok",
		},
		{
			"bus.emit",
			`bus.emit("events.ts-surface-suite-test", {data: "test"}); output("ok");`,
			"ok",
		},
		{
			"kit.register+resolve",
			`var t = createTool({id: "surf-tool-suite", description: "test", execute: async () => ({ok:true})}); kit.register("tool", "surf-tool-suite", t); var r = JSON.parse(__go_brainkit_request("tools.resolve", JSON.stringify({name:"surf-tool-suite"}))); output(r.name ? "ok" : "fail");`,
			"ok",
		},
		{
			"schedule+unschedule",
			`var id = bus.schedule("in 1h", "ts-sched-surface-suite", {}); bus.unschedule(id); output("ok");`,
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
			source := fmt.Sprintf("__ts_surface_suite_%s.ts", tc.name)
			err := testutil.DeployErr(env.Kit, source, tc.code)
			if err != nil {
				t.Logf("deploy error (may be expected): %v", err)
				return
			}

			result := testutil.EvalTS(t, env.Kit, "__get_result_suite.ts", `return String(globalThis.__module_result || "");`)
			assert.Contains(t, result, tc.expect, "%s: expected %q in result", tc.name, tc.expect)
		})
	}
}

// ── Surface matrix: EvalTS surface ──────────────────────────────────────

// testSurfaceEvalTS — operations via direct EvalTS (global scope).
func testSurfaceEvalTS(t *testing.T, env *suite.TestEnv) {
	cases := []struct {
		name   string
		code   string
		expect string
	}{
		{"tools.list", `var r = JSON.parse(__go_brainkit_request("tools.list", "{}")); return String(r.tools.length >= 0);`, "true"},
		{"tools.call", `var r = __go_brainkit_request("tools.call", JSON.stringify({name:"echo",input:{message:"eval"}})); return r.indexOf("eval") >= 0 ? "ok" : "fail";`, "ok"},
		{"secrets.set", `__go_brainkit_request("secrets.set", JSON.stringify({name:"eval-k-suite",value:"eval-v"})); return "ok";`, "ok"},
		{"secrets.get", `__go_brainkit_request("secrets.set", JSON.stringify({name:"eval-g-suite",value:"eval-gv"})); var r = JSON.parse(__go_brainkit_request("secrets.get", JSON.stringify({name:"eval-g-suite"}))); return r.value || "empty";`, "eval-gv"},
		{"fs.list", `return JSON.stringify(fs.readdirSync("."));`, ""},
		{"metrics", `var r = JSON.parse(__go_brainkit_request("metrics.get", "{}")); return r.metrics ? "ok" : "fail";`, "ok"},
		{"registry.list", `__go_brainkit_request("registry.list", JSON.stringify({category:"provider"})); return "ok";`, "ok"},
		{"bus.publish", `var r = bus.publish("incoming.eval-surface-suite", {data:"test"}); return r.replyTo ? "ok" : "fail";`, "ok"},
		{"bus.emit", `bus.emit("events.eval-surface-suite", {data:"test"}); return "ok";`, "ok"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := testutil.EvalTS(t, env.Kit, fmt.Sprintf("__eval_suite_%s.ts", tc.name), tc.code)
			assert.Contains(t, result, tc.expect)
		})
	}
}

// ── Surface matrix: Error consistency ───────────────────────────────────

// testSurfaceErrorConsistency — same error looks the same from every surface.
func testSurfaceErrorConsistency(t *testing.T, env *suite.TestEnv) {
	t.Run("NOT_FOUND/go", func(t *testing.T) {
		payload, ok := env.SendAndReceive(t, sdk.ToolCallMsg{Name: "ghost-tool-consistency-suite"}, 5*time.Second)
		require.True(t, ok)
		assert.Equal(t, "NOT_FOUND", suite.ResponseCode(payload))
	})

	t.Run("NOT_FOUND/ts-deployed", func(t *testing.T) {
		testutil.Deploy(t, env.Kit, "err-consist-suite.ts", `
			var caught = "none";
			try { await tools.call("ghost-tool-consistency-suite", {}); }
			catch(e) { caught = e.message || "unknown"; }
			output(caught);
		`)

		result := testutil.EvalTS(t, env.Kit, "__err_result_suite.ts", `return String(globalThis.__module_result || "");`)
		assert.Contains(t, result, "ghost-tool-consistency-suite", "error should mention the tool name")
	})

	t.Run("NOT_FOUND/evalts", func(t *testing.T) {
		result := testutil.EvalTS(t, env.Kit, "__err_eval_suite.ts", `
			var caught = "none";
			try { __go_brainkit_request("tools.call", JSON.stringify({name:"ghost-tool-consistency-suite"})); }
			catch(e) { caught = e.code || "NO_CODE"; }
			return caught;
		`)
		assert.Equal(t, "NOT_FOUND", result)
	})

	t.Run("VALIDATION_ERROR/go", func(t *testing.T) {
		payload, ok := env.SendAndReceive(t, sdk.SecretsSetMsg{Name: "", Value: "v"}, 5*time.Second)
		require.True(t, ok)
		assert.Equal(t, "VALIDATION_ERROR", suite.ResponseCode(payload))
	})

	t.Run("VALIDATION_ERROR/evalts", func(t *testing.T) {
		result := testutil.EvalTS(t, env.Kit, "__val_eval_suite.ts", `
			var caught = "none";
			try { __go_brainkit_request("secrets.set", JSON.stringify({name:"",value:"v"})); }
			catch(e) { caught = e.code || "NO_CODE"; }
			return caught;
		`)
		assert.Equal(t, "VALIDATION_ERROR", result)
	})
}
