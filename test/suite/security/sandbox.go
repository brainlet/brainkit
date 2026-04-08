package security

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
)

// testSandboxDirectBridgeAccess — reach the raw Go bridge from inside a Compartment.
func testSandboxDirectBridgeAccess(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit

	secDeploy(t, k, "escape-bridge-sec.ts", `
		var leaked = "NO";
		try {
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

	result, _ := secEvalTSErr(k, "__esc.ts", `return String(globalThis.__module_result || "");`)
	assert.Equal(t, "NO", result, "raw Go bridges should NOT be accessible inside Compartment")
}

// testSandboxHijackCompartment — access another deployment's compartment object.
func testSandboxHijackCompartment(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit

	secDeploy(t, k, "victim-sec.ts", `
		bus.on("secret-data", function(msg) { msg.reply({secret: "classified-data-42"}); });
	`)

	secDeploy(t, k, "attacker-sec.ts", `
		var stolen = "FAILED";
		try {
			if (typeof globalThis !== "undefined" && globalThis.__kit_compartments) {
				var victimComp = globalThis.__kit_compartments["victim-sec.ts"];
				if (victimComp) {
					stolen = "HIJACKED_COMPARTMENT";
				}
			}
		} catch(e) { stolen = "BLOCKED:" + e.message; }
		output(stolen);
	`)

	result, _ := secEvalTSErr(k, "__hijack.ts", `return String(globalThis.__module_result || "");`)
	assert.NotContains(t, result, "HIJACKED", "attacker should not access victim's compartment")
}

// testSandboxRegistryManipulation — access the internal resource registry directly.
func testSandboxRegistryManipulation(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit

	secDeploy(t, k, "legit-tool-sec.ts", `
		const t = createTool({id: "protected-tool-sec", description: "secret", execute: async () => ({data: "secret"})});
		kit.register("tool", "protected-tool-sec", t);
	`)

	secDeploy(t, k, "registry-attack-sec.ts", `
		var results = {};
		try {
			if (typeof __kit_registry !== "undefined") {
				results.hasRegistry = true;
				var removed = __kit_registry.unregister("tool", "protected-tool-sec");
				results.removed = !!removed;
				__kit_registry.register("tool", "protected-tool-sec", "protected-tool-sec", {fake: true});
				results.replaced = true;
			} else {
				results.hasRegistry = false;
			}
		} catch(e) { results.error = e.message; }
		output(results);
	`)

	result, _ := secEvalTSErr(k, "__reg_atk.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	assert.NotContains(t, result, `"replaced":true`, "attacker should not replace registry entries")
}

// testSandboxBusSubsHijack — access __bus_subs to hijack another deployment's handlers.
func testSandboxBusSubsHijack(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit

	secDeploy(t, k, "bus-hijack-sec.ts", `
		var leaked = "NO";
		try {
			if (typeof __bus_subs !== "undefined") {
				leaked = "LEAKED:bus_subs_visible";
				var keys = Object.keys(__bus_subs);
				if (keys.length > 0) leaked = "LEAKED:can_see_" + keys.length + "_handlers";
				for (var k in __bus_subs) {
					__bus_subs[k] = function() { return "HIJACKED"; };
					leaked = "HIJACKED:replaced_handler";
					break;
				}
			}
		} catch(e) { leaked = "BLOCKED:" + e.message; }
		output(leaked);
	`)

	result, _ := secEvalTSErr(k, "__bus_hijack.ts", `return String(globalThis.__module_result || "");`)
	assert.NotContains(t, result, "HIJACKED", "should not be able to hijack bus subscription handlers")
}

// testSandboxPrototypePollution — prototype pollution to inject into shared objects.
func testSandboxPrototypePollution(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit

	err := secDeployErr(k, "proto-pollute-sec.ts", `
		var results = {};
		try {
			Object.prototype.pwned = "yes";
			results.objProto = "set";
		} catch(e) { results.objProto = "blocked:" + e.message; }

		try {
			Array.prototype.pwned = "yes";
			results.arrProto = "set";
		} catch(e) { results.arrProto = "blocked:" + e.message; }

		try {
			Function.prototype.pwned = "yes";
			results.fnProto = "set";
		} catch(e) { results.fnProto = "blocked:" + e.message; }

		try {
			var origParse = JSON.parse;
			JSON.parse = function(s) { return {intercepted: true, original: origParse(s)}; };
			results.jsonHijack = "set";
		} catch(e) { results.jsonHijack = "blocked:" + e.message; }

		output(results);
	`)
	if err != nil {
		return // SES blocked during eval — that's fine
	}

	result, _ := secEvalTSErr(k, "__proto.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)

	err = secDeployErr(k, "innocent-sec.ts", `
		var clean = {};
		clean.hasObjPwned = ({}).pwned === "yes";
		clean.hasArrPwned = ([]).pwned === "yes";
		clean.jsonWorks = JSON.parse('{"a":1}').a === 1;
		output(clean);
	`)
	if err == nil {
		result2, _ := secEvalTSErr(k, "__innocent.ts", `
			var r = globalThis.__module_result;
			return JSON.stringify(r || {});
		`)
		assert.NotContains(t, result2, `"hasObjPwned":true`, "prototype pollution should not affect other deployments")
		assert.NotContains(t, result2, `"hasArrPwned":true`)
	}
	_ = result
}

// testSandboxEndowmentOverwrite — overwrite endowment functions to intercept all traffic.
func testSandboxEndowmentOverwrite(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit

	secDeploy(t, k, "overwrite-bus-sec.ts", `
		var results = {};
		try {
			var origPublish = bus.publish;
			bus.publish = function(topic, data) {
				data.intercepted = true;
				return origPublish(topic, data);
			};
			results.busOverwrite = "SUCCESS";
		} catch(e) { results.busOverwrite = "blocked:" + e.message; }

		try {
			var origCall = tools.call;
			tools.call = function(name, input) {
				return {hijacked: true, tool: name};
			};
			results.toolsOverwrite = "SUCCESS";
		} catch(e) { results.toolsOverwrite = "blocked:" + e.message; }

		try {
			var origGet = secrets.get;
			secrets.get = function(name) {
				bus.emit("stolen.secrets", {name: name, value: origGet(name)});
				return origGet(name);
			};
			results.secretsOverwrite = "SUCCESS";
		} catch(e) { results.secretsOverwrite = "blocked:" + e.message; }

		output(results);
	`)

	secDeploy(t, k, "check-intact-sec.ts", `
		var r = bus.publish("incoming.test-intact-sec", {check: true});
		output(r.replyTo ? "INTACT" : "BROKEN");
	`)

	result, _ := secEvalTSErr(k, "__check.ts", `return String(globalThis.__module_result || "");`)
	assert.Equal(t, "INTACT", result, "endowment overwrite should not cross Compartment boundaries")
}

// testSandboxGlobalThisAccess — reach globalThis from within a Compartment.
func testSandboxGlobalThisAccess(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit

	err := secDeployErr(k, "globalthis-probe-sec.ts", `
		var findings = [];

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
			var fn = findings.constructor.constructor;
			var g4 = fn("return this")();
			if (g4 && g4.__go_brainkit_request) findings.push("leaked_via_constructor");
		} catch(e) {}

		output(findings.length > 0 ? findings.join(",") : "SECURE");
	`)
	if err != nil {
		return // SES blocked the probe — sandbox is working
	}

	result, _ := secEvalTSErr(k, "__gt.ts", `return String(globalThis.__module_result || "");`)
	assert.NotContains(t, result, "leaked", "should not be able to access real globalThis from Compartment")
}

// testSandboxFSPathTraversal — file system escape via path traversal.
func testSandboxFSPathTraversal(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit

	secDeploy(t, k, "fs-escape-sec.ts", `
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
				var data = fs.readFileSync(attacks[i], "utf8");
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

	result, _ := secEvalTSErr(k, "__fs_esc.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	assert.NotContains(t, result, "ESCAPED", "no path traversal should escape the workspace")
}

// testSandboxFSWriteEscape — write to a path that could affect the host system.
func testSandboxFSWriteEscape(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit

	secDeploy(t, k, "fs-write-escape-sec.ts", `
		var results = {};
		var attacks = [
			["../../../tmp/brainkit-escape-test", "pwned"],
			["/tmp/brainkit-escape-test", "pwned"],
			["foo/../../../tmp/brainkit-escape-test", "pwned"],
		];
		for (var i = 0; i < attacks.length; i++) {
			try {
				fs.writeFileSync(attacks[i][0], attacks[i][1]);
				results["write_" + i] = "WROTE";
			} catch(e) {
				results["write_" + i] = "blocked";
			}
		}
		output(results);
	`)

	result, _ := secEvalTSErr(k, "__fs_w_esc.ts", `
		var r = globalThis.__module_result;
		if (typeof r === "string") return r;
		return JSON.stringify(r || {});
	`)
	t.Logf("FS write escape results (writes are normalized into workspace): %s", result)
	assert.True(t, secAlive(t, k))
}

// testSandboxRuntimeModification — deploy code that modifies the kit runtime itself.
func testSandboxRuntimeModification(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit

	secDeploy(t, k, "modify-runtime-sec.ts", `
		var results = {};

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

		try {
			if (typeof __kitRunWithSource === "function") {
				results.hasRunWithSource = true;
			} else {
				results.hasRunWithSource = false;
			}
		} catch(e) {}

		output(results);
	`)

	result, _ := secEvalTSErr(k, "__mod_rt.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)

	err := secDeployErr(k, "after-modify-sec.ts", `
		output(typeof backdoor === "function" ? "BACKDOOR_FOUND" : "CLEAN");
	`)
	if err == nil {
		result2, _ := secEvalTSErr(k, "__after_mod.ts", `return String(globalThis.__module_result || "");`)
		assert.Equal(t, "CLEAN", result2, "runtime modification should not affect new deployments")
	}
	_ = result
}
