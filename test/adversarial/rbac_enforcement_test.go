package adversarial_test

import (
	"context"
	"testing"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/rbac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rbacKernel(t *testing.T, defaultRole string) *brainkit.Kernel {
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
		DefaultRole: defaultRole,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.InMemoryStorage(),
		},
	})
	require.NoError(t, err)

	type echoIn struct{ Message string `json:"message"` }
	brainkit.RegisterTool(k, "echo", registry.TypedTool[echoIn]{
		Description: "echoes",
		Execute: func(ctx context.Context, in echoIn) (any, error) {
			return map[string]string{"echoed": in.Message}, nil
		},
	})

	t.Cleanup(func() { k.Close() })
	return k
}

func deployAndCheck(t *testing.T, k *brainkit.Kernel, role, tsCode string) string {
	t.Helper()
	ctx := context.Background()
	src := "rbac-enforce-" + role + ".ts"

	_, err := k.Deploy(ctx, src, tsCode, brainkit.WithRole(role))
	if err != nil {
		return "DEPLOY_FAILED:" + err.Error()
	}
	defer k.Teardown(ctx, src)

	result, err := k.EvalTS(ctx, "__rbac_enforce_result.ts", `
		var r = globalThis.__module_result;
		if (typeof r === "string") return r;
		return JSON.stringify(r || "");
	`)
	if err != nil {
		return "EVAL_FAILED"
	}
	return result
}

// TestRBACEnforcement_ServiceCanPublishIncoming — service role publishes to incoming.*.
func TestRBACEnforcement_ServiceCanPublishIncoming(t *testing.T) {
	k := rbacKernel(t, "service")
	result := deployAndCheck(t, k, "service", `
		var caught = "ALLOWED";
		try { bus.publish("incoming.test-msg", {data: "hello"}); }
		catch(e) { caught = "DENIED:" + (e.message || ""); }
		output(caught);
	`)
	assert.Equal(t, "ALLOWED", result)
}

// TestRBACEnforcement_ServiceCannotPublishRandom — service denied on random topic.
func TestRBACEnforcement_ServiceCannotPublishRandom(t *testing.T) {
	k := rbacKernel(t, "service")
	result := deployAndCheck(t, k, "service", `
		var caught = "ALLOWED";
		try { bus.publish("random.forbidden", {}); }
		catch(e) { caught = "DENIED"; }
		output(caught);
	`)
	assert.Equal(t, "DENIED", result)
}

// TestRBACEnforcement_ServiceCanEmitEvents — service can emit events.*.
func TestRBACEnforcement_ServiceCanEmitEvents(t *testing.T) {
	k := rbacKernel(t, "service")
	result := deployAndCheck(t, k, "service", `
		var caught = "ALLOWED";
		try { bus.emit("events.service-event", {data: "test"}); }
		catch(e) { caught = "DENIED"; }
		output(caught);
	`)
	assert.Equal(t, "ALLOWED", result)
}

// TestRBACEnforcement_ServiceCannotEmitGateway — service denied on gateway.*.
func TestRBACEnforcement_ServiceCannotEmitGateway(t *testing.T) {
	k := rbacKernel(t, "service")
	result := deployAndCheck(t, k, "service", `
		var caught = "ALLOWED";
		try { bus.emit("gateway.test", {}); }
		catch(e) { caught = "DENIED"; }
		output(caught);
	`)
	assert.Equal(t, "DENIED", result)
}

// TestRBACEnforcement_ServiceCanRegisterTool — service role can register tools.
func TestRBACEnforcement_ServiceCanRegisterTool(t *testing.T) {
	k := rbacKernel(t, "service")
	result := deployAndCheck(t, k, "service", `
		var caught = "ALLOWED";
		try {
			var t = createTool({id: "svc-tool", description: "test", execute: async () => ({})});
			kit.register("tool", "svc-tool", t);
		} catch(e) { caught = "DENIED:" + (e.message || ""); }
		output(caught);
	`)
	assert.Equal(t, "ALLOWED", result)
}

// TestRBACEnforcement_ServiceCannotRegisterAgent — service denied agent registration.
func TestRBACEnforcement_ServiceCannotRegisterAgent(t *testing.T) {
	k := rbacKernel(t, "service")
	result := deployAndCheck(t, k, "service", `
		var caught = "ALLOWED";
		try { kit.register("agent", "svc-agent", {}); }
		catch(e) { caught = "DENIED"; }
		output(caught);
	`)
	assert.Equal(t, "DENIED", result)
}

// TestRBACEnforcement_GatewayCanPublishGateway — gateway publishes to gateway.*.
func TestRBACEnforcement_GatewayCanPublishGateway(t *testing.T) {
	k := rbacKernel(t, "gateway")
	result := deployAndCheck(t, k, "gateway", `
		var caught = "ALLOWED";
		try { bus.publish("gateway.request", {url: "/test"}); }
		catch(e) { caught = "DENIED"; }
		output(caught);
	`)
	assert.Equal(t, "ALLOWED", result)
}

// TestRBACEnforcement_GatewayCannotPublishEvents — gateway denied on events.*.
func TestRBACEnforcement_GatewayCannotPublishEvents(t *testing.T) {
	k := rbacKernel(t, "gateway")
	result := deployAndCheck(t, k, "gateway", `
		var caught = "ALLOWED";
		try { bus.publish("events.test", {}); }
		catch(e) { caught = "DENIED"; }
		output(caught);
	`)
	assert.Equal(t, "DENIED", result)
}

// TestRBACEnforcement_GatewayCanEmitGateway — gateway can emit gateway.*.
func TestRBACEnforcement_GatewayCanEmitGateway(t *testing.T) {
	k := rbacKernel(t, "gateway")
	result := deployAndCheck(t, k, "gateway", `
		var caught = "ALLOWED";
		try { bus.emit("gateway.status", {status: "ok"}); }
		catch(e) { caught = "DENIED"; }
		output(caught);
	`)
	assert.Equal(t, "ALLOWED", result)
}

// TestRBACEnforcement_ObserverCannotPublish — observer denied on all publish.
func TestRBACEnforcement_ObserverCannotPublish(t *testing.T) {
	k := rbacKernel(t, "observer")
	result := deployAndCheck(t, k, "observer", `
		var caught = "ALLOWED";
		try { bus.publish("incoming.test", {}); }
		catch(e) { caught = "DENIED"; }
		output(caught);
	`)
	assert.Equal(t, "DENIED", result)
}

// TestRBACEnforcement_ObserverCannotEmit — observer denied on all emit.
func TestRBACEnforcement_ObserverCannotEmit(t *testing.T) {
	k := rbacKernel(t, "observer")
	result := deployAndCheck(t, k, "observer", `
		var caught = "ALLOWED";
		try { bus.emit("events.test", {}); }
		catch(e) { caught = "DENIED"; }
		output(caught);
	`)
	assert.Equal(t, "DENIED", result)
}

// TestRBACEnforcement_ObserverCanSubscribe — observer can subscribe to *.
func TestRBACEnforcement_ObserverCanSubscribe(t *testing.T) {
	k := rbacKernel(t, "observer")
	result := deployAndCheck(t, k, "observer", `
		var caught = "ALLOWED";
		try {
			var id = bus.subscribe("events.anything", function() {});
			bus.unsubscribe(id);
		} catch(e) { caught = "DENIED"; }
		output(caught);
	`)
	assert.Equal(t, "ALLOWED", result)
}

// TestRBACEnforcement_AdminCanDoEverything — admin has no restrictions.
func TestRBACEnforcement_AdminCanDoEverything(t *testing.T) {
	k := rbacKernel(t, "admin")
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

// TestRBACEnforcement_OwnMailboxAlwaysAllowed — every role can access its own mailbox.
func TestRBACEnforcement_OwnMailboxAlwaysAllowed(t *testing.T) {
	roles := []string{"admin", "service", "gateway", "observer"}

	for _, role := range roles {
		t.Run(role, func(t *testing.T) {
			k := rbacKernel(t, role)
			result := deployAndCheck(t, k, role, `
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
