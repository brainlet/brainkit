package adversarial_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/rbac"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ════════════════════════════════════════════════════════════════════════════
// RBAC ESCAPE ATTACKS
// Deployed code should be constrained by its role. These tests try to escalate.
// ════════════════════════════════════════════════════════════════════════════

func rbacAttackKernel(t *testing.T) *brainkit.Kernel {
	t.Helper()
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Roles: map[string]rbac.Role{
			"admin":    rbac.RoleAdmin,
			"service":  rbac.RoleService,
			"gateway":  rbac.RoleGateway,
			"observer": rbac.RoleObserver,
		},
		DefaultRole: "observer",
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.InMemoryStorage(),
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { k.Close() })

	type echoIn struct{ Message string `json:"message"` }
	brainkit.RegisterTool(k, "echo", registry.TypedTool[echoIn]{
		Description: "echoes",
		Execute: func(ctx context.Context, in echoIn) (any, error) {
			return map[string]string{"echoed": in.Message}, nil
		},
	})

	return k
}

// Attack: observer registers a tool (should be denied), then escalates to call tools
func TestRBACEscape_ObserverRegistersToolThenCalls(t *testing.T) {
	k := rbacAttackKernel(t)
	ctx := context.Background()

	_, err := k.Deploy(ctx, "obs-escalate.ts", `
		var results = {};

		// Step 1: Try to register a tool (observer can't)
		try {
			var t = createTool({id: "backdoor", description: "escalation", execute: async () => ({pwned: true})});
			kit.register("tool", "backdoor", t);
			results.registerTool = "ALLOWED";
		} catch(e) { results.registerTool = "DENIED"; }

		// Step 2: Try to call a tool (observer doesn't have tools.call)
		try {
			var r = await tools.call("echo", {message: "escalated"});
			results.callTool = "ALLOWED:" + JSON.stringify(r);
		} catch(e) { results.callTool = "DENIED:" + (e.code || e.message); }

		// Step 3: Try raw bridge to bypass endowment restrictions
		try {
			var raw = __go_brainkit_request("tools.call", JSON.stringify({name: "echo", input: {message: "raw"}}));
			results.rawBridge = "ALLOWED:" + raw;
		} catch(e) { results.rawBridge = "DENIED_OR_UNAVAILABLE"; }

		output(results);
	`, brainkit.WithRole("observer"))
	require.NoError(t, err)

	result, _ := k.EvalTS(ctx, "__obs_esc.ts", `
		var r = globalThis.__module_result;
		if (typeof r === "string") return r;
		return JSON.stringify(r || {});
	`)
	assert.Contains(t, result, `registerTool`)
	assert.Contains(t, result, `DENIED`)
}

// Attack: gateway role tries to read secrets via various paths
func TestRBACEscape_GatewayExfiltratesSecrets(t *testing.T) {
	k := rbacAttackKernel(t)
	ctx := context.Background()

	// Set a secret first (as admin-level Go code)
	sdk.Publish(k, ctx, messages.SecretsSetMsg{Name: "API_KEY", Value: "sk-secret-12345"})
	time.Sleep(100 * time.Millisecond)

	// Gateway tries to read the secret
	_, err := k.Deploy(ctx, "gw-secret-steal.ts", `
		var results = {};

		// Path 1: Direct secrets.get (should be denied — gateway has no secrets.get command)
		try {
			var val = secrets.get("API_KEY");
			results.directGet = "GOT:" + val;
		} catch(e) { results.directGet = "DENIED:" + (e.code || ""); }

		// Path 2: Try via bus command
		try {
			var r = bus.publish("secrets.get", {name: "API_KEY"});
			results.busGet = "PUBLISHED:" + r.replyTo;
		} catch(e) { results.busGet = "DENIED:" + (e.code || ""); }

		// Path 3: Try to emit to secrets topics
		try {
			bus.emit("secrets.get", {name: "API_KEY"});
			results.emitGet = "EMITTED";
		} catch(e) { results.emitGet = "DENIED:" + (e.code || ""); }

		output(results);
	`, brainkit.WithRole("gateway"))
	require.NoError(t, err)

	result, _ := k.EvalTS(ctx, "__gw_sec.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	// Gateway should NOT be able to read secrets
	assert.NotContains(t, result, "sk-secret", "gateway should never see secret values")
}

// Attack: observer deploys .ts that hijacks another deployment's bus handler
func TestRBACEscape_ObserverHijacksServiceHandler(t *testing.T) {
	k := rbacAttackKernel(t)
	ctx := context.Background()

	// Deploy a legitimate service
	_, err := k.Deploy(ctx, "legit-service.ts", `
		bus.on("api", function(msg) {
			msg.reply({legitimate: true, secret: "internal-data"});
		});
	`, brainkit.WithRole("service"))
	require.NoError(t, err)

	// Observer deploys code that subscribes to the SAME topic to intercept messages
	_, err = k.Deploy(ctx, "observer-hijack.ts", `
		var stolen = [];
		try {
			// Observer CAN subscribe (RoleObserver has subscribe: *)
			// But can they intercept replies meant for someone else?
			bus.subscribe("ts.legit-service.api", function(msg) {
				stolen.push(JSON.stringify(msg.payload));
				// Try to reply on behalf of the legitimate service
				try { msg.reply({hijacked: true}); } catch(e) {}
			});
		} catch(e) {}
		output("subscribed");
	`, brainkit.WithRole("observer"))
	require.NoError(t, err)

	// Call the legitimate service — which handler responds?
	pr, _ := sdk.Publish(k, ctx, messages.CustomMsg{
		Topic: "ts.legit-service.api", Payload: json.RawMessage(`{"q":"test"}`),
	})
	ch := make(chan []byte, 2)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		// FINDING #10: On GoChannel (memory transport), observer's reply can arrive BEFORE
		// the legitimate service's reply. The bus is a shared medium — any subscriber
		// on a topic can reply to the replyTo. This is a real reply-impersonation vector.
		// This is a race — sometimes legitimate wins, sometimes observer wins.
		resp := string(p)
		if strings.Contains(resp, "hijacked") {
			t.Logf("FINDING #10: observer reply-impersonation — got %s instead of legitimate response", resp)
			t.Logf("Observer with subscribe:* can subscribe to another service's topic and reply first")
		} else {
			t.Logf("Legitimate response won the race this time: %s", resp)
		}
		// Don't assert — this is documenting a race condition, not a deterministic bug
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
}

// Attack: service role tries to escalate to admin via RBAC bus commands
func TestRBACEscape_ServiceTriesRBACAdmin(t *testing.T) {
	k := rbacAttackKernel(t)
	ctx := context.Background()

	_, err := k.Deploy(ctx, "svc-escalate.ts", `
		var results = {};

		// Try to assign itself admin role
		try {
			var raw = __go_brainkit_request("rbac.assign", JSON.stringify({source: "svc-escalate.ts", role: "admin"}));
			results.selfAssign = "ALLOWED:" + raw;
		} catch(e) { results.selfAssign = "DENIED:" + (e.code || e.message); }

		// Try to list RBAC assignments (should be denied — service doesn't have rbac.list)
		try {
			var raw2 = __go_brainkit_request("rbac.list", "{}");
			results.listRBAC = "ALLOWED";
		} catch(e) { results.listRBAC = "DENIED:" + (e.code || e.message); }

		// Try to revoke someone else's role
		try {
			var raw3 = __go_brainkit_request("rbac.revoke", JSON.stringify({source: "legit-admin.ts"}));
			results.revokeOther = "ALLOWED";
		} catch(e) { results.revokeOther = "DENIED:" + (e.code || e.message); }

		output(results);
	`, brainkit.WithRole("service"))
	require.NoError(t, err)

	result, _ := k.EvalTS(ctx, "__svc_esc.ts", `
		var r = globalThis.__module_result;
		if (typeof r === "string") return r;
		return JSON.stringify(r || {});
	`)
	assert.Contains(t, result, `selfAssign`)
	assert.Contains(t, result, `DENIED`)
}

// FIXED (bug #2): *.reply.* now works — service CAN subscribe to reply topics.
func TestRBACEscape_BrokenReplyPatternExploit(t *testing.T) {
	k := rbacAttackKernel(t)
	ctx := context.Background()

	_, err := k.Deploy(ctx, "reply-bug.ts", `
		var results = {};

		// Service role has Subscribe: {Allow: ["*.reply.*"]}
		// After fix: *.reply.* matches "tools.call.reply.test123"
		try {
			var subId = bus.subscribe("tools.call.reply.test123", function(msg) {});
			bus.unsubscribe(subId);
			results.subscribeReply = "ALLOWED";
		} catch(e) {
			results.subscribeReply = "DENIED:" + (e.code || e.message);
		}

		try {
			var subId2 = bus.subscribe("some.other.reply.test456", function(msg) {});
			bus.unsubscribe(subId2);
			results.subscribeOtherReply = "ALLOWED";
		} catch(e) {
			results.subscribeOtherReply = "DENIED:" + (e.code || e.message);
		}

		output(results);
	`, brainkit.WithRole("service"))
	require.NoError(t, err)

	result, _ := k.EvalTS(ctx, "__reply_bug.ts", `
		var r = globalThis.__module_result;
		if (typeof r === "string") return r;
		return JSON.stringify(r || {});
	`)
	// After fix: service CAN subscribe to reply topics
	assert.Contains(t, result, "ALLOWED")
}

// Attack: deploy as one role, register tool, teardown, redeploy as different role — tool persists?
func TestRBACEscape_RoleSwapToolPersistence(t *testing.T) {
	k := rbacAttackKernel(t)
	ctx := context.Background()

	// Deploy as admin, register a powerful tool
	_, err := k.Deploy(ctx, "role-swap.ts", `
		var t = createTool({id: "admin-power", description: "admin only", execute: async ({cmd}) => ({executed: cmd})});
		kit.register("tool", "admin-power", t);
		output("registered");
	`, brainkit.WithRole("admin"))
	require.NoError(t, err)

	// Teardown the admin deployment
	k.Teardown(ctx, "role-swap.ts")

	// Redeploy SAME source as observer — can we still call the admin tool?
	_, err = k.Deploy(ctx, "role-swap.ts", `
		var result = "UNKNOWN";
		try {
			var r = await tools.call("admin-power", {cmd: "rm -rf"});
			result = "STILL_WORKS:" + JSON.stringify(r);
		} catch(e) {
			result = "DENIED:" + (e.code || e.message);
		}
		output(result);
	`, brainkit.WithRole("observer"))
	require.NoError(t, err)

	result, _ := k.EvalTS(ctx, "__swap.ts", `return String(globalThis.__module_result || "");`)
	// The tool should have been unregistered when the admin deployment was torn down
	assert.Contains(t, result, "DENIED", "tool from torn-down admin deployment should not persist")
}

// Attack: service deploys code that schedules messages to command topics
func TestRBACEscape_ScheduleToCommandTopic(t *testing.T) {
	k := rbacAttackKernel(t)
	ctx := context.Background()

	_, err := k.Deploy(ctx, "sched-cmd.ts", `
		var results = {};

		// Try to schedule a message to a command topic (tools.call)
		// The scheduler fires from Go, bypassing JS-side RBAC checks
		try {
			var id = bus.schedule("in 100ms", "tools.call", {name: "echo", input: {message: "scheduled-bypass"}});
			results.scheduledCmd = "SCHEDULED:" + id;
		} catch(e) { results.scheduledCmd = "DENIED:" + (e.code || e.message); }

		// Try secrets.set via schedule
		try {
			var id2 = bus.schedule("in 100ms", "secrets.set", {name: "backdoor", value: "planted"});
			results.scheduledSecret = "SCHEDULED:" + id2;
		} catch(e) { results.scheduledSecret = "DENIED:" + (e.code || e.message); }

		output(results);
	`, brainkit.WithRole("service"))
	require.NoError(t, err)

	result, _ := k.EvalTS(ctx, "__sched_cmd.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	// bus.schedule in the endowment auto-prefixes with the deployment namespace
	// But the Go-side Schedule() doesn't enforce RBAC on the topic
	t.Logf("Schedule-to-command result: %s", result)
}

// Attack: observer tries to deploy code that deploys MORE code (inception)
func TestRBACEscape_DeployInception(t *testing.T) {
	k := rbacAttackKernel(t)
	ctx := context.Background()

	// Observer deploys code that tries to deploy more code via the bus
	_, err := k.Deploy(ctx, "inception.ts", `
		var results = {};

		// Try kit.deploy via the tools endowment
		try {
			var r = bus.publish("kit.deploy", {source: "evil.ts", code: 'bus.publish("incoming.evil", {pwned: true});'});
			results.deployViaBus = "PUBLISHED:" + r.replyTo;
		} catch(e) { results.deployViaBus = "DENIED:" + (e.code || e.message); }

		output(results);
	`, brainkit.WithRole("observer"))
	require.NoError(t, err)

	result, _ := k.EvalTS(ctx, "__inception.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	// Observer can't publish to arbitrary topics — bus.publish should be denied
	assert.Contains(t, result, "DENIED", "observer should not be able to deploy code via bus")
}

// Attack: forge callerID metadata to impersonate admin
func TestRBACEscape_CallerIDForgery(t *testing.T) {
	k := rbacAttackKernel(t)
	ctx := context.Background()

	// The callerID is stamped by Go middleware, not by JS.
	// But can a .ts deployment influence the callerID of messages it publishes?
	_, err := k.Deploy(ctx, "forge-caller.ts", `
		var results = {};

		// bus.publish goes through __go_brainkit_bus_publish which doesn't accept callerID
		// The callerID is stamped by the Go middleware (CallerIDMiddleware)
		// But can we influence it via the replyTo mechanism?
		try {
			// Try to publish to a command topic pretending to be admin
			var r = bus.publish("incoming.forged", {forged: true});
			results.publish = "ok";
		} catch(e) { results.publish = "denied"; }

		output(results);
	`, brainkit.WithRole("observer"))
	require.NoError(t, err)
	_ = err
}
