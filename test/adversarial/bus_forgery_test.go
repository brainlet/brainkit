package adversarial_test

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ════════════════════════════════════════════════════════════════════════════
// BUS MESSAGE FORGERY
// Messages carry metadata (correlationId, replyTo, callerId). Can we forge them?
// ════════════════════════════════════════════════════════════════════════════

// Attack: subscribe to someone else's replyTo to steal their response
func TestBusForgery_StealReplyTo(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := tk.Deploy(ctx, "steal-reply.ts", `
		bus.on("api", function(msg) {
			msg.reply({secret: "classified-response-data"});
		});
	`)
	require.NoError(t, err)

	// Legitimate caller publishes
	pr, _ := sdk.Publish(tk, ctx, messages.CustomMsg{
		Topic: "ts.steal-reply.api", Payload: json.RawMessage(`{"q":"legit"}`),
	})

	// Attacker subscribes to the SAME replyTo before the legitimate subscriber
	var attackerGot atomic.Int64
	attackerUnsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) {
		attackerGot.Add(1)
	})
	defer attackerUnsub()

	// Legitimate subscriber
	legitimateCh := make(chan []byte, 1)
	legitUnsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) {
		legitimateCh <- m.Payload
	})
	defer legitUnsub()

	select {
	case p := <-legitimateCh:
		assert.Contains(t, string(p), "classified")
		// On GoChannel (memory), BOTH subscribers get the message (fanout).
		// This is a transport property, not a security bug — but worth documenting.
		t.Logf("Attacker intercepted: %d messages (GoChannel fanout behavior)", attackerGot.Load())
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// Attack: publish to a replyTo topic to inject a fake response
func TestBusForgery_InjectFakeReply(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := tk.Deploy(ctx, "slow-service.ts", `
		bus.on("slow", async function(msg) {
			// Simulate slow processing
			await new Promise(r => setTimeout(r, 500));
			msg.reply({real: true, data: "real-response"});
		});
	`)
	require.NoError(t, err)

	// Legitimate call
	pr, _ := sdk.Publish(tk, ctx, messages.CustomMsg{
		Topic: "ts.slow-service.slow", Payload: json.RawMessage(`{}`),
	})

	// Attacker knows the replyTo pattern and injects a fake response BEFORE the real one
	go func() {
		time.Sleep(50 * time.Millisecond) // beat the 500ms service
		fakeResponse, _ := json.Marshal(map[string]any{
			"real": false, "data": "INJECTED-BY-ATTACKER",
		})
		tk.PublishRaw(ctx, pr.ReplyTo, fakeResponse)
	}()

	// The caller gets the FIRST response — which might be the attacker's
	ch := make(chan []byte, 2)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) {
		ch <- m.Payload
	})
	defer unsub()

	select {
	case first := <-ch:
		// Document whether the injected response arrives first
		if string(first) != "" {
			var parsed struct{ Real bool `json:"real"` }
			json.Unmarshal(first, &parsed)
			if !parsed.Real {
				t.Logf("FINDING: attacker's fake response arrived before the real one")
			} else {
				t.Logf("Real response arrived first (race won by legitimate service)")
			}
		}
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// Attack: correlationId collision — two callers use same correlationId
func TestBusForgery_CorrelationIdCollision(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := tk.Deploy(ctx, "collision-svc.ts", `
		bus.on("echo", function(msg) {
			msg.reply({echoed: msg.payload.data, correlationId: msg.correlationId});
		});
	`)
	require.NoError(t, err)

	// Two callers publish with the same forced replyTo
	sharedReplyTo := "ts.collision-svc.echo.reply.shared-id-12345"

	// Caller A subscribes
	chA := make(chan []byte, 2)
	unsubA, _ := tk.SubscribeRaw(ctx, sharedReplyTo, func(m messages.Message) {
		chA <- m.Payload
	})
	defer unsubA()

	// Both publish to the same replyTo
	sdk.Publish(tk, ctx, messages.CustomMsg{
		Topic: "ts.collision-svc.echo", Payload: json.RawMessage(`{"data":"from-A"}`),
	}, sdk.WithReplyTo(sharedReplyTo))

	sdk.Publish(tk, ctx, messages.CustomMsg{
		Topic: "ts.collision-svc.echo", Payload: json.RawMessage(`{"data":"from-B"}`),
	}, sdk.WithReplyTo(sharedReplyTo))

	// Caller A receives BOTH responses — including B's
	var received []string
	for i := 0; i < 2; i++ {
		select {
		case p := <-chA:
			received = append(received, string(p))
		case <-time.After(3 * time.Second):
		}
	}
	t.Logf("Received %d responses on shared replyTo: %v", len(received), received)
	// This is by design (replyTo is just a topic) but worth documenting
}

// Attack: recursive bus message — handler publishes to its own topic
func TestBusForgery_RecursiveBusLoop(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
	})
	require.NoError(t, err)
	defer k.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Deploy handler that re-publishes to itself (infinite loop attempt)
	_, err = k.Deploy(ctx, "recursive.ts", `
		var count = 0;
		bus.on("loop", function(msg) {
			count++;
			if (count < 100) {
				bus.emit("ts.recursive.loop", {depth: count});
			}
			msg.reply({count: count});
		});
	`)
	require.NoError(t, err)

	// Trigger the loop
	pr, _ := sdk.Publish(k, ctx, messages.CustomMsg{
		Topic: "ts.recursive.loop", Payload: json.RawMessage(`{"depth":0}`),
	})
	ch := make(chan []byte, 1)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case <-ch:
		// Got a reply — kernel survived the loop
	case <-time.After(5 * time.Second):
		// Timeout is also OK — the depth middleware should have killed the cascade
	}

	// Kernel must still be alive
	assert.True(t, k.Alive(ctx), "kernel should survive recursive bus loop")
}

// Attack: publish millions of messages to overwhelm the bus
func TestBusForgery_FloodBus(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Deploy a handler
	_, err := tk.Deploy(ctx, "flood-target.ts", `
		var count = 0;
		bus.on("flood", function(msg) { count++; });
	`)
	require.NoError(t, err)

	// Flood: 10,000 messages from Go
	for i := 0; i < 10000; i++ {
		tk.PublishRaw(ctx, "ts.flood-target.flood", json.RawMessage(`{"i":1}`))
	}

	// Wait for some processing
	time.Sleep(2 * time.Second)

	// Kernel must survive
	assert.True(t, tk.Alive(ctx), "kernel should survive 10K message flood")
}

// Attack: create thousands of subscriptions
func TestBusForgery_SubscriptionBomb(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "sub-bomb.ts", `
		var ids = [];
		for (var i = 0; i < 1000; i++) {
			try {
				var id = bus.subscribe("bomb.topic." + i, function(msg) {});
				ids.push(id);
			} catch(e) { break; }
		}
		output({created: ids.length});
	`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__bomb.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	t.Logf("Subscription bomb: %s", result)
	assert.True(t, tk.Alive(ctx), "kernel should survive 1000 subscriptions")

	// Teardown should clean all of them
	tk.Teardown(ctx, "sub-bomb.ts")
	time.Sleep(500 * time.Millisecond)
	assert.True(t, tk.Alive(ctx), "kernel should survive teardown of 1000 subscriptions")
}

// Attack: schedule bomb — create thousands of schedules that fire immediately
func TestBusForgery_ScheduleBomb(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "sched-bomb.ts", `
		var ids = [];
		for (var i = 0; i < 500; i++) {
			try {
				var id = bus.schedule("in 1ms", "bomb.fire", {i: i});
				ids.push(id);
			} catch(e) { break; }
		}
		output({scheduled: ids.length});
	`)
	require.NoError(t, err)

	// Wait for all schedules to fire
	time.Sleep(3 * time.Second)
	assert.True(t, tk.Alive(ctx), "kernel should survive 500 immediate schedules")
}

// Attack: publish to command topics that should be blocked from JS
func TestBusForgery_CommandTopicBypass(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "cmd-bypass.ts", `
		var results = {};
		var commandTopics = [
			"tools.call", "tools.list", "tools.resolve",
			"secrets.set", "secrets.get", "secrets.delete",
			"kit.deploy", "kit.teardown",
			"rbac.assign", "rbac.revoke",
			"fs.read", "fs.write", "fs.delete",
			"wasm.compile", "wasm.run",
		];

		for (var i = 0; i < commandTopics.length; i++) {
			var topic = commandTopics[i];
			try {
				// bus.emit should be blocked for command topics (bug #8 says it isn't)
				bus.emit(topic, {bypass: true});
				results[topic] = "EMITTED";
			} catch(e) {
				results[topic] = "BLOCKED:" + (e.code || "");
			}
		}
		output(results);
	`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__cmd_bypass.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	// FINDING #8: bus.emit doesn't validate against command topics
	t.Logf("Command topic bypass via bus.emit: %s", result)
	// Count how many were emitted vs blocked
	var parsed map[string]string
	json.Unmarshal([]byte(result), &parsed)
	emitted := 0
	for _, v := range parsed {
		if v == "EMITTED" {
			emitted++
		}
	}
	if emitted > 0 {
		t.Logf("FINDING #8 confirmed: %d/%d command topics accepted by bus.emit", emitted, len(parsed))
	}
}

// Attack: tool name collision — two deployments register the same tool name
func TestBusForgery_ToolNameCollision(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Deploy A registers "shared-tool" returning version 1
	_, err := tk.Deploy(ctx, "tool-a.ts", `
		var t = createTool({id: "shared-tool", description: "v1", execute: async () => ({version: 1})});
		kit.register("tool", "shared-tool", t);
	`)
	require.NoError(t, err)

	// Deploy B tries to register "shared-tool" returning version 2 (should this work?)
	_, err = tk.Deploy(ctx, "tool-b.ts", `
		var t = createTool({id: "shared-tool", description: "v2", execute: async () => ({version: 2})});
		kit.register("tool", "shared-tool", t);
	`)
	require.NoError(t, err)

	// Which version do we get when we call the tool?
	payload, ok := sendAndReceive(t, tk, messages.ToolCallMsg{Name: "shared-tool", Input: map[string]any{}}, 5*time.Second)
	require.True(t, ok)
	t.Logf("Tool collision result: %s", string(payload))
	// Document: does the second registration silently overwrite the first?
}

// Attack: deploy code that sends messages with crafted metadata
func TestBusForgery_MetadataInjection(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Deploy code that tries to inject metadata via the payload
	// The Go bridge stamps metadata separately — payload shouldn't leak into metadata
	_, err := tk.Deploy(ctx, "meta-inject.ts", `
		var results = {};

		// Publish with a payload that looks like metadata
		try {
			var r = bus.publish("incoming.meta-test", {
				replyTo: "evil.reply.topic",
				correlationId: "forged-correlation",
				callerId: "admin",
				depth: "999",
			});
			results.published = r.replyTo;
		} catch(e) { results.error = e.message; }

		output(results);
	`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__meta_inj.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	// The replyTo in the response should be auto-generated, not the forged one
	assert.NotContains(t, result, "evil.reply.topic")
}

// Attack: deploy code that modifies __module_result of other deployments
func TestBusForgery_CrossDeploymentResult(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "set-result.ts", `output({realData: "from-set-result"});`)
	require.NoError(t, err)

	// Attacker tries to overwrite the result
	_, err = tk.Deploy(ctx, "overwrite-result.ts", `
		try {
			globalThis.__module_result = {hijacked: true};
		} catch(e) {}
		output("attacker");
	`)
	require.NoError(t, err)

	// Check if the original deployment's result is preserved
	// EvalTS runs in global context, not in a Compartment
	result, _ := tk.EvalTS(ctx, "__check_result.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || "");
	`)
	// __module_result is global — last writer wins. This is by design for EvalTS.
	// But each Deploy's output() sets it, so the last Deploy's output is what's there.
	t.Logf("Cross-deployment result: %s", result)
}

// Attack: use RegisterTool from Go to register a tool with JS executor that escapes sandbox
func TestBusForgery_MaliciousGoTool(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
	})
	require.NoError(t, err)
	defer k.Close()

	// Register a tool that tries to inject code via its response
	type injectionInput struct{ Cmd string `json:"cmd"` }
	brainkit.RegisterTool(k, "injector", registry.TypedTool[injectionInput]{
		Description: "returns crafted payload",
		Execute: func(ctx context.Context, in injectionInput) (any, error) {
			// Return a payload that looks like it could be eval'd
			return map[string]string{
				"result":   "normal",
				"__proto__": `{"polluted": true}`,
				"constructor": `function() { return "hijacked"; }`,
			}, nil
		},
	})

	ctx := context.Background()
	_, err = k.Deploy(ctx, "call-injector.ts", `
		var r = await tools.call("injector", {cmd: "test"});
		output({
			result: r.result,
			hasPollution: ({}).polluted === true,
		});
	`)
	require.NoError(t, err)

	result, _ := k.EvalTS(ctx, "__inj.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	assert.Contains(t, result, `"hasPollution":false`, "tool response should not cause prototype pollution")
}
