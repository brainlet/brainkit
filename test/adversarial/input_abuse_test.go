package adversarial_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/rbac"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// === B01-B06: Deploy abuse ===

// FIXED (bug #3): empty source name is now rejected.
func TestInputAbuse_Deploy_EmptySource(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	_, err := tk.Deploy(context.Background(), "", `output("hello");`)
	assert.Error(t, err, "empty source should be rejected")
	_, err2 := tk.Deploy(context.Background(), "   ", `output("hello");`)
	assert.Error(t, err2, "whitespace-only source should be rejected")
}

func TestInputAbuse_Deploy_SpecialChars(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	cases := []string{
		"../escape.ts",
		"path/with\x00null.ts",
		`path"with"quotes.ts`,
		"path`with`backticks.ts",
		"path with spaces.ts",
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			_, err := tk.Deploy(ctx, source, `output("hi");`)
			// Should either succeed (source is just an identifier) or error cleanly — never panic
			_ = err
		})
	}
}

func TestInputAbuse_Deploy_DottedSourceName(t *testing.T) {
	// Dots in source name interact with bus topic resolution (ts.my.agent.topic)
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "my.dotted.agent.ts", `
		bus.on("ask", function(msg) { msg.reply({ answer: "dotted" }); });
	`)
	require.NoError(t, err)

	// Verify the mailbox resolves correctly despite dots
	result, err := tk.EvalTS(ctx, "__dotted_test.ts", `
		var r = bus.publish("ts.my.dotted.agent.ask", { q: "test" });
		return r.replyTo;
	`)
	require.NoError(t, err)
	assert.Contains(t, result, "ts.my.dotted.agent.ask.reply.")
}

func TestInputAbuse_Deploy_LargeCode(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	// 1MB of code (mostly comments)
	big := "// " + strings.Repeat("x", 1024*1024) + "\noutput('big');"
	_, err := tk.Deploy(context.Background(), "big.ts", big)
	// Should succeed or fail cleanly — never hang
	_ = err
}

func TestInputAbuse_Deploy_InvalidTS(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	_, err := tk.Deploy(context.Background(), "invalid.ts", "const x: = {{{;;;")
	assert.Error(t, err)
	// Error could come from transpiler OR QuickJS eval — both are valid
	assert.True(t, strings.Contains(err.Error(), "transpile") || strings.Contains(err.Error(), "eval"),
		"expected transpile or eval error, got: %s", err.Error())
}

func TestInputAbuse_Deploy_ThrowsDuringInit(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	_, err := tk.Deploy(context.Background(), "throw-init.ts", `
		throw new Error("init explosion");
	`)
	assert.Error(t, err)
	// Deployment should be cleaned up — not left in a half-state
	deployments := tk.ListDeployments()
	for _, d := range deployments {
		assert.NotEqual(t, "throw-init.ts", d.Source, "failed deployment should not be listed")
	}
}

func TestInputAbuse_Deploy_PartialCleanup(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Deploy .ts that registers a tool then throws
	_, err := tk.Deploy(ctx, "partial.ts", `
		const t = createTool({ id: "partial-tool", description: "partial", execute: async () => ({}) });
		kit.register("tool", "partial-tool", t);
		throw new Error("after registration");
	`)
	assert.Error(t, err)

	// The tool should have been cleaned up by teardown
	pr, pubErr := sdk.Publish(tk, ctx, messages.ToolResolveMsg{Name: "partial-tool"})
	require.NoError(t, pubErr)

	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case payload := <-ch:
		assert.Contains(t, string(payload), "not found")
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
}

func TestInputAbuse_Deploy_RedeployDifferentTools(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Deploy v1 with tool A
	_, err := tk.Deploy(ctx, "evolving.ts", `
		const a = createTool({ id: "tool-a", description: "v1", execute: async () => ({ v: 1 }) });
		kit.register("tool", "tool-a", a);
	`)
	require.NoError(t, err)

	// Redeploy with tool B (no tool A)
	_, err = tk.Redeploy(ctx, "evolving.ts", `
		const b = createTool({ id: "tool-b", description: "v2", execute: async () => ({ v: 2 }) });
		kit.register("tool", "tool-b", b);
	`)
	require.NoError(t, err)

	// tool-a should not exist
	pr, _ := sdk.Publish(tk, ctx, messages.ToolResolveMsg{Name: "tool-a"})
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()
	select {
	case payload := <-ch:
		assert.Contains(t, string(payload), "not found")
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
}

// === B07-B10: Bus abuse ===

func TestInputAbuse_Bus_EmptyTopic(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	result, err := tk.EvalTS(context.Background(), "__empty_topic.ts", `
		var caught = "none";
		try { bus.publish("", {}); }
		catch(e) { caught = e.code || e.message; }
		return caught;
	`)
	require.NoError(t, err)
	// FIXED (bug #4): empty topic is now rejected with VALIDATION_ERROR
	assert.NotEqual(t, "none", result, "empty topic should throw an error")
}

func TestInputAbuse_Bus_LargePayload(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	// 100KB payload (not 1MB — GoChannel has limits)
	result, err := tk.EvalTS(context.Background(), "__big_payload.ts", `
		var big = { data: "x".repeat(100000) };
		var r = bus.publish("incoming.big", big);
		return r.replyTo ? "ok" : "fail";
	`)
	require.NoError(t, err)
	assert.Equal(t, "ok", result)
}

func TestInputAbuse_Bus_DeeplyNestedJSON(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	result, err := tk.EvalTS(context.Background(), "__nested.ts", `
		var obj = {};
		var curr = obj;
		for (var i = 0; i < 50; i++) {
			curr.nested = {};
			curr = curr.nested;
		}
		curr.value = "deep";
		var r = bus.publish("incoming.nested", obj);
		return r.replyTo ? "ok" : "fail";
	`)
	require.NoError(t, err)
	assert.Equal(t, "ok", result)
}

func TestInputAbuse_Bus_SubscribeEmptyTopic(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	result, err := tk.EvalTS(context.Background(), "__sub_empty.ts", `
		var caught = "none";
		try { bus.subscribe("", function() {}); }
		catch(e) { caught = "error"; }
		return caught;
	`)
	require.NoError(t, err)
	// Empty subscribe should either work (subscribe to everything) or error — not panic
	_ = result
}

// === B11-B14: Registration abuse ===

func TestInputAbuse_Register_EmptyName(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	result, err := tk.EvalTS(context.Background(), "__reg_empty.ts", `
		var caught = "none";
		try { kit.register("tool", "", {}); }
		catch(e) { caught = e.message || "error"; }
		return caught;
	`)
	require.NoError(t, err)
	assert.Contains(t, result, "required")
}

func TestInputAbuse_Register_InvalidType(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	result, err := tk.EvalTS(context.Background(), "__reg_invalid.ts", `
		var caught = "none";
		try { kit.register("banana", "test", {}); }
		catch(e) { caught = e.message || "error"; }
		return caught;
	`)
	require.NoError(t, err)
	assert.Contains(t, result, "invalid type")
}

func TestInputAbuse_Tool_NonexistentName(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	code, _ := busErrorCode(t, tk, messages.ToolCallMsg{Name: "absolutely-does-not-exist"})
	assert.Equal(t, "NOT_FOUND", code)
}

func TestInputAbuse_Tool_MalformedInput(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	// Call the echo tool with wrong input shape
	ctx := context.Background()
	pr, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "echo", Input: "not-an-object"})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case payload := <-ch:
		// Should get a response (possibly error) — not a hang
		assert.NotEmpty(t, payload)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout — tool call with malformed input hung")
	}
}

// === B15-B17: Secrets abuse ===

func TestInputAbuse_Secrets_EmptyName(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	code, _ := busErrorCode(t, tk, messages.SecretsSetMsg{Name: "", Value: "val"})
	assert.Equal(t, "VALIDATION_ERROR", code)
}

func TestInputAbuse_Secrets_LargeValue(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()
	big := strings.Repeat("x", 100000) // 100KB secret
	pr, err := sdk.Publish(tk, ctx, messages.SecretsSetMsg{Name: "big-secret", Value: big})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()
	select {
	case payload := <-ch:
		assert.Contains(t, string(payload), "stored")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout storing large secret")
	}
}

func TestInputAbuse_Secrets_SpecialCharsInName(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()
	names := []string{"key/with/slashes", "key.with.dots", "key with spaces", "key=with=equals"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			pr, err := sdk.Publish(tk, ctx, messages.SecretsSetMsg{Name: name, Value: "val"})
			require.NoError(t, err)
			ch := make(chan []byte, 1)
			unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
			defer unsub()
			select {
			case payload := <-ch:
				// Should succeed or error cleanly — never panic
				_ = payload
			case <-time.After(5 * time.Second):
				t.Fatal("timeout")
			}
		})
	}
}

// === B18-B19: RBAC abuse ===

func TestInputAbuse_RBAC_NonexistentRole(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Roles: map[string]rbac.Role{"admin": rbac.RoleAdmin},
	})
	require.NoError(t, err)
	defer k.Close()

	tk := &testutil.TestKernel{Kernel: k}
	ctx := context.Background()
	pr, err := sdk.Publish(tk, ctx, messages.RBACAssignMsg{Source: "test.ts", Role: "nonexistent-role"})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()
	select {
	case payload := <-ch:
		assert.Contains(t, string(payload), "error")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestInputAbuse_RBAC_EmptySource(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Roles: map[string]rbac.Role{"admin": rbac.RoleAdmin},
	})
	require.NoError(t, err)
	defer k.Close()

	tk := &testutil.TestKernel{Kernel: k}
	code, _ := busErrorCode(t, tk, messages.RBACAssignMsg{Source: "", Role: "admin"})
	assert.Equal(t, "VALIDATION_ERROR", code)
}

// === B20-B21: Schedule abuse ===

func TestInputAbuse_Schedule_InvalidExpression(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	_, err := tk.Schedule(context.Background(), brainkit.ScheduleConfig{
		Expression: "bananas at midnight",
		Topic:      "test",
	})
	assert.Error(t, err)
}

func TestInputAbuse_Schedule_EmptyTopic(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	// Empty topic schedule — should work (topic is just a string, empty is valid in Watermill)
	id, err := tk.Schedule(context.Background(), brainkit.ScheduleConfig{
		Expression: "in 1h",
		Topic:      "",
	})
	// Either succeeds or errors — no panic
	if err == nil {
		tk.Unschedule(context.Background(), id)
	}
}

// === B22-B23: EvalTS abuse ===

func TestInputAbuse_EvalTS_InfiniteLoop(t *testing.T) {
	// KNOWN LIMITATION: QuickJS interrupt handler checks periodically but an infinite
	// synchronous loop (while(true){}) blocks the JS thread. The interrupt fires
	// eventually but may take longer than the context deadline.
	// This test verifies the kernel doesn't permanently hang — it will eventually stop
	// via the bridge's closing interrupt handler.
	t.Skip("KNOWN: infinite JS loop blocks until QuickJS interrupt fires (~5-10s). Not a crash, but slow to interrupt.")
}
