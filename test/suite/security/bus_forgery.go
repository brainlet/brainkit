package security

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	tools "github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testForgeryStealReplyTo — subscribe to someone else's replyTo to steal their response.
func testForgeryStealReplyTo(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := secDeployErr(k, "steal-reply-sec.ts", `
		bus.on("api", function(msg) {
			msg.reply({secret: "classified-response-data"});
		});
	`)
	require.NoError(t, err)

	pr, _ := sdk.Publish(k, ctx, sdk.CustomMsg{
		Topic: "ts.steal-reply-sec.api", Payload: json.RawMessage(`{"q":"legit"}`),
	})

	var attackerGot atomic.Int64
	attackerUnsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) {
		attackerGot.Add(1)
	})
	defer attackerUnsub()

	legitimateCh := make(chan []byte, 1)
	legitUnsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) {
		legitimateCh <- m.Payload
	})
	defer legitUnsub()

	select {
	case p := <-legitimateCh:
		assert.Contains(t, string(p), "classified")
		t.Logf("Attacker intercepted: %d messages (GoChannel fanout behavior)", attackerGot.Load())
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// testForgeryInjectFakeReply — publish to a replyTo topic to inject a fake response.
func testForgeryInjectFakeReply(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := secDeployErr(k, "slow-service-sec.ts", `
		bus.on("slow", async function(msg) {
			await new Promise(r => setTimeout(r, 500));
			msg.reply({real: true, data: "real-response"});
		});
	`)
	require.NoError(t, err)

	pr, _ := sdk.Publish(k, ctx, sdk.CustomMsg{
		Topic: "ts.slow-service-sec.slow", Payload: json.RawMessage(`{}`),
	})

	go func() {
		time.Sleep(50 * time.Millisecond)
		fakeResponse, _ := json.Marshal(map[string]any{
			"real": false, "data": "INJECTED-BY-ATTACKER",
		})
		k.PublishRaw(ctx, pr.ReplyTo, fakeResponse)
	}()

	ch := make(chan []byte, 2)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) {
		ch <- m.Payload
	})
	defer unsub()

	select {
	case first := <-ch:
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

// testForgeryCorrelationIdCollision — two callers use same correlationId.
func testForgeryCorrelationIdCollision(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := secDeployErr(k, "collision-svc-sec.ts", `
		bus.on("echo", function(msg) {
			msg.reply({echoed: msg.payload.data, correlationId: msg.correlationId});
		});
	`)
	require.NoError(t, err)

	sharedReplyTo := "ts.collision-svc-sec.echo.reply.shared-id-12345"

	chA := make(chan []byte, 2)
	unsubA, _ := k.SubscribeRaw(ctx, sharedReplyTo, func(m sdk.Message) {
		chA <- m.Payload
	})
	defer unsubA()

	sdk.Publish(k, ctx, sdk.CustomMsg{
		Topic: "ts.collision-svc-sec.echo", Payload: json.RawMessage(`{"data":"from-A"}`),
	}, sdk.WithReplyTo(sharedReplyTo))

	sdk.Publish(k, ctx, sdk.CustomMsg{
		Topic: "ts.collision-svc-sec.echo", Payload: json.RawMessage(`{"data":"from-B"}`),
	}, sdk.WithReplyTo(sharedReplyTo))

	var received []string
	for i := 0; i < 2; i++ {
		select {
		case p := <-chA:
			received = append(received, string(p))
		case <-time.After(3 * time.Second):
		}
	}
	t.Logf("Received %d responses on shared replyTo: %v", len(received), received)
}

// testForgeryRecursiveBusLoop — recursive bus message handler publishes to its own topic.
func testForgeryRecursiveBusLoop(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
	})
	require.NoError(t, err)
	defer k.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = secDeployErr(k, "recursive-sec.ts", `
		var count = 0;
		bus.on("loop", function(msg) {
			count++;
			if (count < 100) {
				bus.emit("ts.recursive-sec.loop", {depth: count});
			}
			msg.reply({count: count});
		});
	`)
	require.NoError(t, err)

	pr, _ := sdk.Publish(k, ctx, sdk.CustomMsg{
		Topic: "ts.recursive-sec.loop", Payload: json.RawMessage(`{"depth":0}`),
	})
	ch := make(chan []byte, 1)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case <-ch:
		// Got a reply — kit survived
	case <-time.After(5 * time.Second):
		// Timeout is also OK
	}

	assert.True(t, secAlive(t, k), "kit should survive recursive bus loop")
}

// testForgeryFloodBus — publish millions of messages to overwhelm the bus.
func testForgeryFloodBus(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit
	ctx := context.Background()

	secDeploy(t, k, "flood-target-sec.ts", `
		var count = 0;
		bus.on("flood", function(msg) { count++; });
	`)

	for i := 0; i < 10000; i++ {
		k.PublishRaw(ctx, "ts.flood-target-sec.flood", json.RawMessage(`{"i":1}`))
	}

	time.Sleep(2 * time.Second)
	assert.True(t, secAlive(t, k), "kit should survive 10K message flood")
}

// testForgerySubscriptionBomb — create thousands of subscriptions.
func testForgerySubscriptionBomb(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit

	secDeploy(t, k, "sub-bomb-sec.ts", `
		var ids = [];
		for (var i = 0; i < 1000; i++) {
			try {
				var id = bus.subscribe("bomb.topic.sec." + i, function(msg) {});
				ids.push(id);
			} catch(e) { break; }
		}
		output({created: ids.length});
	`)

	result, _ := secEvalTSErr(k, "__bomb.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	t.Logf("Subscription bomb: %s", result)
	assert.True(t, secAlive(t, k), "kit should survive 1000 subscriptions")

	secTeardown(t, k, "sub-bomb-sec.ts")
	time.Sleep(500 * time.Millisecond)
	assert.True(t, secAlive(t, k), "kit should survive teardown of 1000 subscriptions")
}

// testForgeryScheduleBomb — create thousands of schedules that fire immediately.
func testForgeryScheduleBomb(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit

	secDeploy(t, k, "sched-bomb-sec.ts", `
		var ids = [];
		for (var i = 0; i < 500; i++) {
			try {
				var id = bus.schedule("in 1ms", "bomb.fire.sec", {i: i});
				ids.push(id);
			} catch(e) { break; }
		}
		output({scheduled: ids.length});
	`)

	time.Sleep(3 * time.Second)
	assert.True(t, secAlive(t, k), "kit should survive 500 immediate schedules")
}

// testForgeryCommandTopicBypass — publish to command topics that should be blocked from JS.
func testForgeryCommandTopicBypass(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit

	secDeploy(t, k, "cmd-bypass-sec.ts", `
		var results = {};
		var commandTopics = [
			"tools.call", "tools.list", "tools.resolve",
			"secrets.set", "secrets.get", "secrets.delete",
			"kit.deploy", "kit.teardown",
			"rbac.assign", "rbac.revoke",
			"wasm.compile", "wasm.run",
		];

		for (var i = 0; i < commandTopics.length; i++) {
			var topic = commandTopics[i];
			try {
				bus.emit(topic, {bypass: true});
				results[topic] = "EMITTED";
			} catch(e) {
				results[topic] = "BLOCKED:" + (e.code || "");
			}
		}
		output(results);
	`)

	result, _ := secEvalTSErr(k, "__cmd_bypass.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	t.Logf("Command topic bypass via bus.emit: %s", result)
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

// testForgeryToolNameCollision — two deployments register the same tool name.
func testForgeryToolNameCollision(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit

	secDeploy(t, k, "tool-a-sec.ts", `
		var t = createTool({id: "shared-tool-sec", description: "v1", execute: async () => ({version: 1})});
		kit.register("tool", "shared-tool-sec", t);
	`)

	secDeploy(t, k, "tool-b-sec.ts", `
		var t = createTool({id: "shared-tool-sec", description: "v2", execute: async () => ({version: 2})});
		kit.register("tool", "shared-tool-sec", t);
	`)

	payload, ok := secSendAndReceive(t, k, sdk.ToolCallMsg{Name: "shared-tool-sec", Input: map[string]any{}}, 5*time.Second)
	require.True(t, ok)
	t.Logf("Tool collision result: %s", string(payload))
}

// testForgeryMetadataInjection — deploy code that sends messages with crafted metadata.
func testForgeryMetadataInjection(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit

	secDeploy(t, k, "meta-inject-sec.ts", `
		var results = {};

		try {
			var r = bus.publish("incoming.meta-test-sec", {
				replyTo: "evil.reply.topic",
				correlationId: "forged-correlation",
				callerId: "admin",
				depth: "999",
			});
			results.published = r.replyTo;
		} catch(e) { results.error = e.message; }

		output(results);
	`)

	result, _ := secEvalTSErr(k, "__meta_inj.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	assert.NotContains(t, result, "evil.reply.topic")
}

// testForgeryCrossDeploymentResult — deploy code that modifies __module_result of other deployments.
func testForgeryCrossDeploymentResult(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit

	secDeploy(t, k, "set-result-sec.ts", `output({realData: "from-set-result"});`)

	secDeploy(t, k, "overwrite-result-sec.ts", `
		try {
			globalThis.__module_result = {hijacked: true};
		} catch(e) {}
		output("attacker");
	`)

	result, _ := secEvalTSErr(k, "__check_result.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || "");
	`)
	t.Logf("Cross-deployment result: %s", result)
}

// testForgeryMaliciousGoTool — RegisterTool from Go with crafted payload to escape sandbox.
func testForgeryMaliciousGoTool(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
	})
	require.NoError(t, err)
	defer k.Close()

	type injectionInput struct{ Cmd string `json:"cmd"` }
	brainkit.RegisterTool(k, "injector-sec", tools.TypedTool[injectionInput]{
		Description: "returns crafted payload",
		Execute: func(ctx context.Context, in injectionInput) (any, error) {
			return map[string]string{
				"result":      "normal",
				"__proto__":   `{"polluted": true}`,
				"constructor": `function() { return "hijacked"; }`,
			}, nil
		},
	})

	secDeploy(t, k, "call-injector-sec.ts", `
		var r = await tools.call("injector-sec", {cmd: "test"});
		output({
			result: r.result,
			hasPollution: ({}).polluted === true,
		});
	`)

	result, _ := secEvalTSErr(k, "__inj.ts", `
		var r = globalThis.__module_result;
		if (typeof r === "string") return r;
		return JSON.stringify(r || {});
	`)
	assert.Contains(t, result, `hasPollution`)
	assert.NotContains(t, result, `"hasPollution":true`, "tool response should not cause prototype pollution")
}
