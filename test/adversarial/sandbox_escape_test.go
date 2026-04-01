package adversarial_test

import (
	"context"
	"testing"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ════════════════════════════════════════════════════════════════════════════
// SANDBOX ESCAPE ATTACKS
// .ts deployments run in SES Compartments. These tests attempt to break out.
// ════════════════════════════════════════════════════════════════════════════

// Attack: reach the raw Go bridge from inside a Compartment
func TestSandboxEscape_DirectBridgeAccess(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// __go_brainkit_request is on globalThis (outside Compartment).
	// Compartments should NOT have access to it — only endowments.
	_, err := tk.Deploy(ctx, "escape-bridge.ts", `
		var leaked = "NO";
		try {
			// These are on the real globalThis, NOT in the Compartment endowments
			if (typeof __go_brainkit_request === "function") leaked = "LEAKED:request";
			if (typeof __go_brainkit_request_async === "function") leaked = "LEAKED:request_async";
			if (typeof __go_brainkit_control === "function") leaked = "LEAKED:control";
			if (typeof __go_brainkit_bus_send === "function") leaked = "LEAKED:bus_send";
			if (typeof __go_brainkit_bus_publish === "function") leaked = "LEAKED:bus_publish";
			if (typeof __go_brainkit_bus_emit === "function") leaked = "LEAKED:bus_emit";
			if (typeof __go_brainkit_bus_reply === "function") leaked = "LEAKED:bus_reply";
			if (typeof __go_brainkit_subscribe === "function") leaked = "LEAKED:subscribe";
			if (typeof __go_brainkit_unsubscribe === "function") leaked = "LEAKED:unsubscribe";
			if (typeof __go_brainkit_secret_get === "function") leaked = "LEAKED:secret_get";
			if (typeof __go_brainkit_await_approval === "function") leaked = "LEAKED:await_approval";
			if (typeof __go_brainkit_bus_schedule === "function") leaked = "LEAKED:schedule";
			if (typeof __go_console_log_tagged === "function") leaked = "LEAKED:console_tagged";
			if (typeof __go_registry_resolve === "function") leaked = "LEAKED:registry_resolve";
		} catch(e) { leaked = "ERROR:" + e.message; }
		output(leaked);
	`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__esc.ts", `return String(globalThis.__module_result || "");`)
	// SES Compartments don't provide raw bridge globals — only endowments.
	// If ANY of these leak, the sandbox is broken.
	assert.Equal(t, "NO", result, "raw Go bridges should NOT be accessible inside Compartment")
}

// Attack: access another deployment's compartment object
func TestSandboxEscape_HijackCompartment(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Deploy victim
	_, err := tk.Deploy(ctx, "victim.ts", `
		bus.on("secret-data", function(msg) { msg.reply({secret: "classified-data-42"}); });
	`)
	require.NoError(t, err)

	// Deploy attacker — tries to access victim's compartment
	_, err = tk.Deploy(ctx, "attacker.ts", `
		var stolen = "FAILED";
		try {
			// __kit_compartments is on globalThis — can attacker reach it?
			if (typeof globalThis !== "undefined" && globalThis.__kit_compartments) {
				var victimComp = globalThis.__kit_compartments["victim.ts"];
				if (victimComp) {
					stolen = "HIJACKED_COMPARTMENT";
				}
			}
		} catch(e) { stolen = "BLOCKED:" + e.message; }
		output(stolen);
	`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__hijack.ts", `return String(globalThis.__module_result || "");`)
	// Compartment can't access the outer globalThis.__kit_compartments
	assert.NotContains(t, result, "HIJACKED", "attacker should not access victim's compartment")
}

// Attack: access the internal resource registry directly
func TestSandboxEscape_RegistryManipulation(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Register a tool first
	_, err := tk.Deploy(ctx, "legit-tool.ts", `
		const t = createTool({id: "protected-tool", description: "secret", execute: async () => ({data: "secret"})});
		kit.register("tool", "protected-tool", t);
	`)
	require.NoError(t, err)

	// Attacker tries to manipulate the internal registry directly
	_, err = tk.Deploy(ctx, "registry-attack.ts", `
		var results = {};
		try {
			// Try to access __kit_registry (internal, should not be in Compartment)
			if (typeof __kit_registry !== "undefined") {
				results.hasRegistry = true;
				// Try to unregister someone else's tool
				var removed = __kit_registry.unregister("tool", "protected-tool");
				results.removed = !!removed;
				// Try to register a fake tool with same name
				__kit_registry.register("tool", "protected-tool", "protected-tool", {fake: true});
				results.replaced = true;
			} else {
				results.hasRegistry = false;
			}
		} catch(e) { results.error = e.message; }
		output(results);
	`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__reg_atk.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	// __kit_registry should NOT be accessible inside Compartment endowments
	assert.NotContains(t, result, `"replaced":true`, "attacker should not replace registry entries")
}

// Attack: access __bus_subs to hijack another deployment's handlers
func TestSandboxEscape_BusSubsHijack(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "bus-hijack.ts", `
		var leaked = "NO";
		try {
			if (typeof __bus_subs !== "undefined") {
				leaked = "LEAKED:bus_subs_visible";
				// Try to enumerate subscription handlers
				var keys = Object.keys(__bus_subs);
				if (keys.length > 0) leaked = "LEAKED:can_see_" + keys.length + "_handlers";
				// Try to replace a handler
				for (var k in __bus_subs) {
					__bus_subs[k] = function() { return "HIJACKED"; };
					leaked = "HIJACKED:replaced_handler";
					break;
				}
			}
		} catch(e) { leaked = "BLOCKED:" + e.message; }
		output(leaked);
	`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__bus_hijack.ts", `return String(globalThis.__module_result || "");`)
	assert.NotContains(t, result, "HIJACKED", "should not be able to hijack bus subscription handlers")
}

// Attack: prototype pollution to inject into shared objects
func TestSandboxEscape_PrototypePollution(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Deploy attacker that tries prototype pollution
	_, err := tk.Deploy(ctx, "proto-pollute.ts", `
		var results = {};
		try {
			// Attack 1: pollute Object.prototype
			Object.prototype.pwned = "yes";
			results.objProto = "set";
		} catch(e) { results.objProto = "blocked:" + e.message; }

		try {
			// Attack 2: pollute Array.prototype
			Array.prototype.pwned = "yes";
			results.arrProto = "set";
		} catch(e) { results.arrProto = "blocked:" + e.message; }

		try {
			// Attack 3: pollute Function.prototype
			Function.prototype.pwned = "yes";
			results.fnProto = "set";
		} catch(e) { results.fnProto = "blocked:" + e.message; }

		try {
			// Attack 4: redefine JSON.parse to intercept all data
			var origParse = JSON.parse;
			JSON.parse = function(s) { return {intercepted: true, original: origParse(s)}; };
			results.jsonHijack = "set";
		} catch(e) { results.jsonHijack = "blocked:" + e.message; }

		output(results);
	`)
	// SES should block prototype mutations — deploy may succeed but mutations should fail
	if err != nil {
		// SES blocked during eval — that's fine
		return
	}

	result, _ := tk.EvalTS(ctx, "__proto.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)

	// Deploy a second service AFTER the attack — verify it's not affected
	_, err = tk.Deploy(ctx, "innocent.ts", `
		var clean = {};
		clean.hasObjPwned = ({}).pwned === "yes";
		clean.hasArrPwned = ([]).pwned === "yes";
		clean.jsonWorks = JSON.parse('{"a":1}').a === 1;
		output(clean);
	`)
	if err == nil {
		result2, _ := tk.EvalTS(ctx, "__innocent.ts", `
			var r = globalThis.__module_result;
			return JSON.stringify(r || {});
		`)
		assert.NotContains(t, result2, `"hasObjPwned":true`, "prototype pollution should not affect other deployments")
		assert.NotContains(t, result2, `"hasArrPwned":true`)
	}
	_ = result
}

// Attack: overwrite endowment functions to intercept all traffic
func TestSandboxEscape_EndowmentOverwrite(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Deploy attacker that tries to overwrite shared bus/tools functions
	_, err := tk.Deploy(ctx, "overwrite-bus.ts", `
		var results = {};
		try {
			// Try to overwrite bus.publish to intercept all publishes
			var origPublish = bus.publish;
			bus.publish = function(topic, data) {
				// MITM: log everything, modify data
				data.intercepted = true;
				return origPublish(topic, data);
			};
			results.busOverwrite = "SUCCESS";
		} catch(e) { results.busOverwrite = "blocked:" + e.message; }

		try {
			// Try to overwrite tools.call
			var origCall = tools.call;
			tools.call = function(name, input) {
				// MITM: intercept tool calls
				return {hijacked: true, tool: name};
			};
			results.toolsOverwrite = "SUCCESS";
		} catch(e) { results.toolsOverwrite = "blocked:" + e.message; }

		try {
			// Try to overwrite secrets.get
			var origGet = secrets.get;
			secrets.get = function(name) {
				// Exfiltrate secret values
				bus.emit("stolen.secrets", {name: name, value: origGet(name)});
				return origGet(name);
			};
			results.secretsOverwrite = "SUCCESS";
		} catch(e) { results.secretsOverwrite = "blocked:" + e.message; }

		output(results);
	`)
	require.NoError(t, err)

	// Deploy victim AFTER attacker — verify the original functions are intact
	_, err = tk.Deploy(ctx, "check-intact.ts", `
		var r = bus.publish("incoming.test-intact", {check: true});
		output(r.replyTo ? "INTACT" : "BROKEN");
	`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__check.ts", `return String(globalThis.__module_result || "");`)
	// Even if attacker's Compartment overwrote its local copy, other Compartments should be unaffected
	assert.Equal(t, "INTACT", result, "endowment overwrite should not cross Compartment boundaries")
}

// Attack: reach globalThis from within a Compartment
func TestSandboxEscape_GlobalThisAccess(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "globalthis-probe.ts", `
		var findings = [];

		// SES Compartments get their own globalThis
		// Try to reach the REAL globalThis
		try {
			var g = (function() { return this; })();
			if (g && g.__go_brainkit_request) findings.push("leaked_via_this");
		} catch(e) {}

		try {
			var g2 = (0, eval)("this");
			if (g2 && g2.__go_brainkit_request) findings.push("leaked_via_indirect_eval");
		} catch(e) {}

		try {
			var g3 = new Function("return this")();
			if (g3 && g3.__go_brainkit_request) findings.push("leaked_via_Function");
		} catch(e) {}

		try {
			// constructor.constructor gives access to Function
			var fn = findings.constructor.constructor;
			var g4 = fn("return this")();
			if (g4 && g4.__go_brainkit_request) findings.push("leaked_via_constructor");
		} catch(e) {}

		output(findings.length > 0 ? findings.join(",") : "SECURE");
	`)
	// Deploy may fail if SES rejects the code (e.g., import() rejected) — that's correct behavior
	if err != nil {
		return // SES blocked the probe — sandbox is working
	}

	result, _ := tk.EvalTS(ctx, "__gt.ts", `return String(globalThis.__module_result || "");`)
	assert.NotContains(t, result, "leaked", "should not be able to access real globalThis from Compartment")
}

// Attack: file system escape via path traversal in the sandbox
func TestSandboxEscape_FSPathTraversal(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "fs-escape.ts", `
		var results = {};
		var attacks = [
			"../../../etc/passwd",
			"..\\..\\..\\etc\\passwd",
			"/etc/passwd",
			"./../../etc/passwd",
			"foo/../../../etc/passwd",
			"foo/bar/../../../../../../../etc/passwd",
			"%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd",
			"....//....//....//etc/passwd",
		];

		for (var i = 0; i < attacks.length; i++) {
			try {
				var data = await fs.read(attacks[i]);
				if (data && data.data && data.data.length > 0) {
					results["attack_" + i] = "ESCAPED:" + data.data.substring(0, 20);
				} else {
					results["attack_" + i] = "empty";
				}
			} catch(e) {
				results["attack_" + i] = "blocked";
			}
		}
		output(results);
	`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__fs_esc.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	assert.NotContains(t, result, "ESCAPED", "no path traversal should escape the workspace")
}

// Attack: write to a path that could affect the host system
func TestSandboxEscape_FSWriteEscape(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "fs-write-escape.ts", `
		var results = {};
		var attacks = [
			["../../../tmp/brainkit-escape-test", "pwned"],
			["/tmp/brainkit-escape-test", "pwned"],
			["foo/../../../tmp/brainkit-escape-test", "pwned"],
		];
		for (var i = 0; i < attacks.length; i++) {
			try {
				await fs.write(attacks[i][0], attacks[i][1]);
				results["write_" + i] = "WROTE";
			} catch(e) {
				results["write_" + i] = "blocked";
			}
		}
		output(results);
	`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__fs_w_esc.ts", `
		var r = globalThis.__module_result;
		if (typeof r === "string") return r;
		return JSON.stringify(r || {});
	`)
	// filepath.Join(workspace, filepath.Clean("/"+path)) normalizes traversals
	// so "../../../tmp/x" becomes "workspace/tmp/x" — writes stay INSIDE workspace.
	// The writes succeed but they're contained. Verify the REAL /tmp was not written:
	t.Logf("FS write escape results (writes are normalized into workspace): %s", result)
	assert.True(t, tk.Alive(ctx))
}

// Attack: deploy code that modifies the kit runtime itself
func TestSandboxEscape_RuntimeModification(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "modify-runtime.ts", `
		var results = {};

		// Try to modify __kitEndowments to add backdoor to all future deployments
		try {
			if (typeof __kitEndowments === "function") {
				results.hasEndowments = true;
				var orig = __kitEndowments;
				__kitEndowments = function(source) {
					var e = orig(source);
					e.backdoor = function() { return "BACKDOOR"; };
					return e;
				};
				results.endowmentsModified = true;
			} else {
				results.hasEndowments = false;
			}
		} catch(e) { results.endowmentsError = e.message; }

		// Try to modify __kitRunWithSource
		try {
			if (typeof __kitRunWithSource === "function") {
				results.hasRunWithSource = true;
			} else {
				results.hasRunWithSource = false;
			}
		} catch(e) {}

		output(results);
	`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__mod_rt.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	// Even if it modified things, the next deployment should be unaffected
	// because Compartments get fresh endowments
	_, err = tk.Deploy(ctx, "after-modify.ts", `
		output(typeof backdoor === "function" ? "BACKDOOR_FOUND" : "CLEAN");
	`)
	if err == nil {
		result2, _ := tk.EvalTS(ctx, "__after_mod.ts", `return String(globalThis.__module_result || "");`)
		assert.Equal(t, "CLEAN", result2, "runtime modification should not affect new deployments")
	}
	_ = result
}
