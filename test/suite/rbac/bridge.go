package rbac

import (
	"context"
	"testing"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests migrated from test/adversarial/rbac_enforcement_test.go (14 tests).
// These verify RBAC enforcement at the bridge level via deploy+output()+EvalTS.

func testBridgeServiceCanPublishIncoming(t *testing.T, _ *suite.TestEnv) {
	k := newRBACKernel(t, "service")
	result := bridgeDeployAndCheck(t, k, "service", `
		var caught = "ALLOWED";
		try { bus.publish("incoming.test-msg", {data: "hello"}); }
		catch(e) { caught = "DENIED:" + (e.message || ""); }
		output(caught);
	`)
	assert.Equal(t, "ALLOWED", result)
}

func testBridgeServiceCannotPublishRandom(t *testing.T, _ *suite.TestEnv) {
	k := newRBACKernel(t, "service")
	result := bridgeDeployAndCheck(t, k, "service", `
		var caught = "ALLOWED";
		try { bus.publish("random.forbidden", {}); }
		catch(e) { caught = "DENIED"; }
		output(caught);
	`)
	assert.Equal(t, "DENIED", result)
}

func testBridgeServiceCanEmitEvents(t *testing.T, _ *suite.TestEnv) {
	k := newRBACKernel(t, "service")
	result := bridgeDeployAndCheck(t, k, "service", `
		var caught = "ALLOWED";
		try { bus.emit("events.service-event", {data: "test"}); }
		catch(e) { caught = "DENIED"; }
		output(caught);
	`)
	assert.Equal(t, "ALLOWED", result)
}

func testBridgeServiceCannotEmitGateway(t *testing.T, _ *suite.TestEnv) {
	k := newRBACKernel(t, "service")
	result := bridgeDeployAndCheck(t, k, "service", `
		var caught = "ALLOWED";
		try { bus.emit("gateway.test", {}); }
		catch(e) { caught = "DENIED"; }
		output(caught);
	`)
	assert.Equal(t, "DENIED", result)
}

func testBridgeServiceCanRegisterTool(t *testing.T, _ *suite.TestEnv) {
	k := newRBACKernel(t, "service")
	result := bridgeDeployAndCheck(t, k, "service", `
		var caught = "ALLOWED";
		try {
			var t = createTool({id: "svc-tool", description: "test", execute: async () => ({})});
			kit.register("tool", "svc-tool", t);
		} catch(e) { caught = "DENIED:" + (e.message || ""); }
		output(caught);
	`)
	assert.Equal(t, "ALLOWED", result)
}

func testBridgeServiceCannotRegisterAgent(t *testing.T, _ *suite.TestEnv) {
	k := newRBACKernel(t, "service")
	result := bridgeDeployAndCheck(t, k, "service", `
		var caught = "ALLOWED";
		try { kit.register("agent", "svc-agent", {}); }
		catch(e) { caught = "DENIED"; }
		output(caught);
	`)
	assert.Equal(t, "DENIED", result)
}

func testBridgeGatewayCanPublishGateway(t *testing.T, _ *suite.TestEnv) {
	k := newRBACKernel(t, "gateway")
	result := bridgeDeployAndCheck(t, k, "gateway", `
		var caught = "ALLOWED";
		try { bus.publish("gateway.request", {url: "/test"}); }
		catch(e) { caught = "DENIED"; }
		output(caught);
	`)
	assert.Equal(t, "ALLOWED", result)
}

func testBridgeGatewayCannotPublishEvents(t *testing.T, _ *suite.TestEnv) {
	k := newRBACKernel(t, "gateway")
	result := bridgeDeployAndCheck(t, k, "gateway", `
		var caught = "ALLOWED";
		try { bus.publish("events.test", {}); }
		catch(e) { caught = "DENIED"; }
		output(caught);
	`)
	assert.Equal(t, "DENIED", result)
}

func testBridgeGatewayCanEmitGateway(t *testing.T, _ *suite.TestEnv) {
	k := newRBACKernel(t, "gateway")
	result := bridgeDeployAndCheck(t, k, "gateway", `
		var caught = "ALLOWED";
		try { bus.emit("gateway.status", {status: "ok"}); }
		catch(e) { caught = "DENIED"; }
		output(caught);
	`)
	assert.Equal(t, "ALLOWED", result)
}

func testBridgeObserverCannotPublish(t *testing.T, _ *suite.TestEnv) {
	k := newRBACKernel(t, "observer")
	result := bridgeDeployAndCheck(t, k, "observer", `
		var caught = "ALLOWED";
		try { bus.publish("incoming.test", {}); }
		catch(e) { caught = "DENIED"; }
		output(caught);
	`)
	assert.Equal(t, "DENIED", result)
}

func testBridgeObserverCannotEmit(t *testing.T, _ *suite.TestEnv) {
	k := newRBACKernel(t, "observer")
	result := bridgeDeployAndCheck(t, k, "observer", `
		var caught = "ALLOWED";
		try { bus.emit("events.test", {}); }
		catch(e) { caught = "DENIED"; }
		output(caught);
	`)
	assert.Equal(t, "DENIED", result)
}

func testBridgeObserverCanSubscribe(t *testing.T, _ *suite.TestEnv) {
	k := newRBACKernel(t, "observer")
	result := bridgeDeployAndCheck(t, k, "observer", `
		var caught = "ALLOWED";
		try {
			var id = bus.subscribe("events.anything", function() {});
			bus.unsubscribe(id);
		} catch(e) { caught = "DENIED"; }
		output(caught);
	`)
	assert.Equal(t, "ALLOWED", result)
}

func testBridgeAdminCanDoEverything(t *testing.T, _ *suite.TestEnv) {
	k := newRBACKernel(t, "admin")
	ctx := context.Background()

	ops := []struct {
		name string
		code string
	}{
		{"publish-random", `bus.publish("random.any.topic", {}); output("ALLOWED");`},
		{"emit-events", `bus.emit("events.admin", {}); output("ALLOWED");`},
		{"emit-gateway", `bus.emit("gateway.admin", {}); output("ALLOWED");`},
		{"subscribe-any", `var id = bus.subscribe("anything", function(){}); bus.unsubscribe(id); output("ALLOWED");`},
		{"register-tool", `var t = createTool({id:"admin-t",description:"",execute:async()=>({})}); kit.register("tool","admin-t",t); output("ALLOWED");`},
		{"register-agent", `kit.register("agent", "admin-a", {}); output("ALLOWED");`},
	}

	for _, op := range ops {
		t.Run(op.name, func(t *testing.T) {
			src := "admin-" + op.name + ".ts"
			_, err := k.Deploy(ctx, src, op.code, brainkit.WithRole("admin"))
			require.NoError(t, err)
			defer k.Teardown(ctx, src)

			result, _ := k.EvalTS(ctx, "__admin_result.ts", `return String(globalThis.__module_result || "");`)
			assert.Equal(t, "ALLOWED", result, "admin should be allowed %s", op.name)
		})
	}
}

func testBridgeOwnMailboxAlwaysAllowed(t *testing.T, _ *suite.TestEnv) {
	roles := []string{"admin", "service", "gateway", "observer"}

	for _, role := range roles {
		t.Run(role, func(t *testing.T) {
			k := newRBACKernel(t, role)
			result := bridgeDeployAndCheck(t, k, role, `
				var caught = "ALLOWED";
				try {
					bus.on("ping", function(msg) { msg.reply({ok:true}); });
				} catch(e) { caught = "DENIED:" + (e.message || ""); }
				output(caught);
			`)
			assert.Equal(t, "ALLOWED", result, "%s should access own mailbox", role)
		})
	}
}
