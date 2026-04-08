package security

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testRBACObserverRegistersToolThenCalls — observer registers a tool (should be denied), then escalates.
func testRBACObserverRegistersToolThenCalls(t *testing.T, env *suite.TestEnv) {
	k := secRBACKernel(t)

	err := secDeployWithRole(k, "obs-escalate-sec.ts", `
		var results = {};

		try {
			var t = createTool({id: "backdoor-sec", description: "escalation", execute: async () => ({pwned: true})});
			kit.register("tool", "backdoor-sec", t);
			results.registerTool = "ALLOWED";
		} catch(e) { results.registerTool = "DENIED"; }

		try {
			var r = await tools.call("echo", {message: "escalated"});
			results.callTool = "ALLOWED:" + JSON.stringify(r);
		} catch(e) { results.callTool = "DENIED:" + (e.code || e.message); }

		try {
			var raw = __go_brainkit_request("tools.call", JSON.stringify({name: "echo", input: {message: "raw"}}));
			results.rawBridge = "ALLOWED:" + raw;
		} catch(e) { results.rawBridge = "DENIED_OR_UNAVAILABLE"; }

		output(results);
	`, "observer")
	require.NoError(t, err)

	result, _ := secEvalTSErr(k, "__obs_esc.ts", `
		var r = globalThis.__module_result;
		if (typeof r === "string") return r;
		return JSON.stringify(r || {});
	`)
	assert.Contains(t, result, `registerTool`)
	assert.Contains(t, result, `DENIED`)
}

// testRBACGatewayExfiltratesSecrets — gateway role tries to read secrets.
func testRBACGatewayExfiltratesSecrets(t *testing.T, env *suite.TestEnv) {
	k := secRBACKernel(t)
	ctx := context.Background()

	sdk.Publish(k, ctx, messages.SecretsSetMsg{Name: "API_KEY_SEC", Value: "sk-secret-12345"})
	time.Sleep(100 * time.Millisecond)

	err := secDeployWithRole(k, "gw-secret-steal-sec.ts", `
		var results = {};

		try {
			var val = secrets.get("API_KEY_SEC");
			results.directGet = "GOT:" + val;
		} catch(e) { results.directGet = "DENIED:" + (e.code || ""); }

		try {
			var r = bus.publish("secrets.get", {name: "API_KEY_SEC"});
			results.busGet = "PUBLISHED:" + r.replyTo;
		} catch(e) { results.busGet = "DENIED:" + (e.code || ""); }

		try {
			bus.emit("secrets.get", {name: "API_KEY_SEC"});
			results.emitGet = "EMITTED";
		} catch(e) { results.emitGet = "DENIED:" + (e.code || ""); }

		output(results);
	`, "gateway")
	require.NoError(t, err)

	result, _ := secEvalTSErr(k, "__gw_sec.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	assert.NotContains(t, result, "sk-secret", "gateway should never see secret values")
}

// testRBACObserverHijacksServiceHandler — observer deploys .ts that hijacks another deployment's bus handler.
func testRBACObserverHijacksServiceHandler(t *testing.T, env *suite.TestEnv) {
	k := secRBACKernel(t)
	ctx := context.Background()

	require.NoError(t, secDeployWithRole(k, "legit-service-sec.ts", `
		bus.on("api", function(msg) {
			msg.reply({legitimate: true, secret: "internal-data"});
		});
	`, "service"))

	require.NoError(t, secDeployWithRole(k, "observer-hijack-sec.ts", `
		var stolen = [];
		try {
			bus.subscribe("ts.legit-service-sec.api", function(msg) {
				stolen.push(JSON.stringify(msg.payload));
				try { msg.reply({hijacked: true}); } catch(e) {}
			});
		} catch(e) {}
		output("subscribed");
	`, "observer"))

	pr, _ := sdk.Publish(k, ctx, messages.CustomMsg{
		Topic: "ts.legit-service-sec.api", Payload: json.RawMessage(`{"q":"test"}`),
	})
	ch := make(chan []byte, 2)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		resp := string(p)
		if strings.Contains(resp, "hijacked") {
			t.Logf("FINDING #10: observer reply-impersonation — got %s instead of legitimate response", resp)
		} else {
			t.Logf("Legitimate response won the race this time: %s", resp)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
}

// testRBACServiceTriesAdmin — service role tries to escalate to admin via RBAC bus commands.
func testRBACServiceTriesAdmin(t *testing.T, env *suite.TestEnv) {
	k := secRBACKernel(t)

	require.NoError(t, secDeployWithRole(k, "svc-escalate-sec.ts", `
		var results = {};

		try {
			var raw = __go_brainkit_request("rbac.assign", JSON.stringify({source: "svc-escalate-sec.ts", role: "admin"}));
			results.selfAssign = "ALLOWED:" + raw;
		} catch(e) { results.selfAssign = "DENIED:" + (e.code || e.message); }

		try {
			var raw2 = __go_brainkit_request("rbac.list", "{}");
			results.listRBAC = "ALLOWED";
		} catch(e) { results.listRBAC = "DENIED:" + (e.code || e.message); }

		try {
			var raw3 = __go_brainkit_request("rbac.revoke", JSON.stringify({source: "legit-admin.ts"}));
			results.revokeOther = "ALLOWED";
		} catch(e) { results.revokeOther = "DENIED:" + (e.code || e.message); }

		output(results);
	`, "service"))

	result, _ := secEvalTSErr(k, "__svc_esc.ts", `
		var r = globalThis.__module_result;
		if (typeof r === "string") return r;
		return JSON.stringify(r || {});
	`)
	assert.Contains(t, result, `selfAssign`)
	assert.Contains(t, result, `DENIED`)
}

// testRBACBrokenReplyPatternExploit — FIXED: *.reply.* now works.
func testRBACBrokenReplyPatternExploit(t *testing.T, env *suite.TestEnv) {
	k := secRBACKernel(t)

	require.NoError(t, secDeployWithRole(k, "reply-bug-sec.ts", `
		var results = {};

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
	`, "service"))

	result, _ := secEvalTSErr(k, "__reply_bug.ts", `
		var r = globalThis.__module_result;
		if (typeof r === "string") return r;
		return JSON.stringify(r || {});
	`)
	assert.Contains(t, result, "ALLOWED")
}

// testRBACRoleSwapToolPersistence — deploy as one role, register tool, teardown, redeploy as different role.
func testRBACRoleSwapToolPersistence(t *testing.T, env *suite.TestEnv) {
	k := secRBACKernel(t)

	require.NoError(t, secDeployWithRole(k, "role-swap-sec.ts", `
		var t = createTool({id: "admin-power-sec", description: "admin only", execute: async ({cmd}) => ({executed: cmd})});
		kit.register("tool", "admin-power-sec", t);
		output("registered");
	`, "admin"))

	secTeardown(t, k, "role-swap-sec.ts")

	require.NoError(t, secDeployWithRole(k, "role-swap-sec.ts", `
		var result = "UNKNOWN";
		try {
			var r = await tools.call("admin-power-sec", {cmd: "rm -rf"});
			result = "STILL_WORKS:" + JSON.stringify(r);
		} catch(e) {
			result = "DENIED:" + (e.code || e.message);
		}
		output(result);
	`, "observer"))

	result, _ := secEvalTSErr(k, "__swap.ts", `return String(globalThis.__module_result || "");`)
	assert.Contains(t, result, "DENIED", "tool from torn-down admin deployment should not persist")
}

// testRBACScheduleToCommandTopic — service deploys code that schedules messages to command topics.
func testRBACScheduleToCommandTopic(t *testing.T, env *suite.TestEnv) {
	k := secRBACKernel(t)

	require.NoError(t, secDeployWithRole(k, "sched-cmd-sec.ts", `
		var results = {};

		try {
			var id = bus.schedule("in 100ms", "tools.call", {name: "echo", input: {message: "scheduled-bypass"}});
			results.scheduledCmd = "SCHEDULED:" + id;
		} catch(e) { results.scheduledCmd = "DENIED:" + (e.code || e.message); }

		try {
			var id2 = bus.schedule("in 100ms", "secrets.set", {name: "backdoor-sec", value: "planted"});
			results.scheduledSecret = "SCHEDULED:" + id2;
		} catch(e) { results.scheduledSecret = "DENIED:" + (e.code || e.message); }

		output(results);
	`, "service"))

	result, _ := secEvalTSErr(k, "__sched_cmd.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	t.Logf("Schedule-to-command result: %s", result)
}

// testRBACDeployInception — observer tries to deploy code that deploys MORE code.
func testRBACDeployInception(t *testing.T, env *suite.TestEnv) {
	k := secRBACKernel(t)

	require.NoError(t, secDeployWithRole(k, "inception-sec.ts", `
		var results = {};

		try {
			var r = bus.publish("kit.deploy", {source: "evil-sec.ts", code: 'bus.publish("incoming.evil.sec", {pwned: true});'});
			results.deployViaBus = "PUBLISHED:" + r.replyTo;
		} catch(e) { results.deployViaBus = "DENIED:" + (e.code || e.message); }

		output(results);
	`, "observer"))

	result, _ := secEvalTSErr(k, "__inception.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	assert.Contains(t, result, "DENIED", "observer should not be able to deploy code via bus")
}

// testRBACCallerIDForgery — forge callerID metadata to impersonate admin.
func testRBACCallerIDForgery(t *testing.T, env *suite.TestEnv) {
	k := secRBACKernel(t)

	_ = secDeployWithRole(k, "forge-caller-sec.ts", `
		var results = {};

		try {
			var r = bus.publish("incoming.forged.sec", {forged: true});
			results.publish = "ok";
		} catch(e) { results.publish = "denied"; }

		output(results);
	`, "observer")
}
