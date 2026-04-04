package bus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/sdkerrors"
	"github.com/brainlet/brainkit/rbac"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// busErrorCodeAdv publishes a message and extracts the error code+details from the response.
func busErrorCodeAdv(t *testing.T, k *brainkit.Kernel, msg messages.BrainkitMessage) (string, map[string]any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, err := sdk.Publish(k, ctx, msg)
	require.NoError(t, err)

	ch := make(chan json.RawMessage, 1)
	unsub, err := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) {
		ch <- json.RawMessage(m.Payload)
	})
	require.NoError(t, err)
	defer unsub()

	select {
	case payload := <-ch:
		var resp struct {
			Error   string         `json:"error"`
			Code    string         `json:"code"`
			Details map[string]any `json:"details"`
		}
		require.NoError(t, json.Unmarshal(payload, &resp))
		return resp.Code, resp.Details
	case <-ctx.Done():
		t.Fatal("timeout waiting for bus response")
		return "", nil
	}
}

// testErrorContractBusNotFound — NOT_FOUND for nonexistent tool.
func testErrorContractBusNotFound(t *testing.T, env *suite.TestEnv) {
	code, details := busErrorCodeAdv(t, env.Kernel, messages.ToolCallMsg{Name: "nonexistent-tool-xyz-adv"})
	assert.Equal(t, "NOT_FOUND", code)
	if details != nil {
		assert.Equal(t, "nonexistent-tool-xyz-adv", details["name"])
	}
}

// testErrorContractBusValidationError — VALIDATION_ERROR for empty secret name.
func testErrorContractBusValidationError(t *testing.T, env *suite.TestEnv) {
	code, _ := busErrorCodeAdv(t, env.Kernel, messages.SecretsSetMsg{Name: "", Value: "val"})
	assert.Equal(t, "VALIDATION_ERROR", code)
}

// testErrorContractBusNotConfiguredRBAC — VALIDATION_ERROR for rbac.assign with empty source.
func testErrorContractBusNotConfiguredRBAC(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Roles: map[string]rbac.Role{"admin": rbac.RoleAdmin},
	})
	require.NoError(t, err)
	defer k.Close()

	code, _ := busErrorCodeAdv(t, k, messages.RBACAssignMsg{Source: "", Role: "admin"})
	assert.Equal(t, "VALIDATION_ERROR", code)
}

// testErrorContractBusAlreadyExists — ALREADY_EXISTS for duplicate deploy.
func testErrorContractBusAlreadyExists(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	k := freshEnv.Kernel
	ctx := context.Background()

	_, err := k.Deploy(ctx, "dup-test-adv.ts", `
		const t = createTool({ id: "dup-adv", description: "dup", execute: async () => ({}) });
		kit.register("tool", "dup-adv", t);
	`)
	require.NoError(t, err)

	code, _ := busErrorCodeAdv(t, k, messages.KitDeployMsg{Source: "dup-test-adv.ts", Code: "// different"})
	assert.Equal(t, "ALREADY_EXISTS", code)
}

// testErrorContractBusDeployErrorBadSyntax — DEPLOY_ERROR for bad syntax.
func testErrorContractBusDeployErrorBadSyntax(t *testing.T, env *suite.TestEnv) {
	code, details := busErrorCodeAdv(t, env.Kernel, messages.KitDeployMsg{
		Source: "bad-syntax-adv.ts",
		Code:   "const x: number = {{{invalid syntax;;;",
	})
	assert.Equal(t, "DEPLOY_ERROR", code)
	if details != nil {
		assert.Equal(t, "bad-syntax-adv.ts", details["source"])
	}
}

// testErrorContractErrorsAsAllTypes — errors.As works for every BrainkitError type through wrapping.
func testErrorContractErrorsAsAllTypes(t *testing.T, _ *suite.TestEnv) {
	cases := []struct {
		name string
		err  error
		code string
	}{
		{"NotFound", &sdkerrors.NotFoundError{Resource: "tool", Name: "x"}, "NOT_FOUND"},
		{"AlreadyExists", &sdkerrors.AlreadyExistsError{Resource: "deployment", Name: "x"}, "ALREADY_EXISTS"},
		{"Validation", &sdkerrors.ValidationError{Field: "name", Message: "required"}, "VALIDATION_ERROR"},
		{"Timeout", &sdkerrors.TimeoutError{Operation: "test"}, "TIMEOUT"},
		{"WorkspaceEscape", &sdkerrors.WorkspaceEscapeError{Path: "../etc"}, "WORKSPACE_ESCAPE"},
		{"PermissionDenied", &sdkerrors.PermissionDeniedError{Source: "a.ts", Action: "publish", Topic: "t", Role: "r"}, "PERMISSION_DENIED"},
		{"RateLimited", &sdkerrors.RateLimitedError{Role: "svc", Limit: 100}, "RATE_LIMITED"},
		{"NotConfigured", &sdkerrors.NotConfiguredError{Feature: "rbac"}, "NOT_CONFIGURED"},
		{"Transport", &sdkerrors.TransportError{Operation: "pub", Cause: fmt.Errorf("fail")}, "TRANSPORT_ERROR"},
		{"Persistence", &sdkerrors.PersistenceError{Operation: "Save", Source: "x", Cause: fmt.Errorf("fail")}, "PERSISTENCE_ERROR"},
		{"Deploy", &sdkerrors.DeployError{Source: "x.ts", Phase: "eval", Cause: fmt.Errorf("fail")}, "DEPLOY_ERROR"},
		{"Bridge", &sdkerrors.BridgeError{Function: "fn", Cause: fmt.Errorf("fail")}, "BRIDGE_ERROR"},
		{"Compiler", &sdkerrors.CompilerError{Cause: fmt.Errorf("fail")}, "COMPILER_ERROR"},
		{"CycleDetected", &sdkerrors.CycleDetectedError{Depth: 16}, "CYCLE_DETECTED"},
		{"Decode", &sdkerrors.DecodeError{Topic: "t", Cause: fmt.Errorf("fail")}, "DECODE_ERROR"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Wrap it twice — errors.As must still find it
			wrapped := fmt.Errorf("layer1: %w", fmt.Errorf("layer2: %w", tc.err))

			var bk sdkerrors.BrainkitError
			require.True(t, errors.As(wrapped, &bk), "errors.As should find BrainkitError")
			assert.Equal(t, tc.code, bk.Code())
			assert.NotNil(t, bk.Details())
		})
	}
}

// testErrorContractJSBridgePermissionDenied — JS bridge returns PERMISSION_DENIED error code.
func testErrorContractJSBridgePermissionDenied(t *testing.T, _ *suite.TestEnv) {
	tk := suite.Full(t, suite.WithRBAC(map[string]rbac.Role{
		"restricted": {
			Name:     "restricted",
			Bus:      rbac.BusPermissions{},
			Commands: rbac.CommandPermissions{Allow: []string{"*"}},
		},
	}, "restricted"))

	ctx := context.Background()
	_, err := tk.Kernel.Deploy(ctx, "perm-test-adv.ts", `
		var caught = "none";
		try { bus.publish("forbidden", {}); }
		catch(e) { caught = e.code || "NO_CODE"; }
		output(caught);
	`, brainkit.WithRole("restricted"))
	require.NoError(t, err)

	result, err := tk.Kernel.EvalTS(ctx, "__perm_result_adv.ts", `return String(globalThis.__module_result || "");`)
	require.NoError(t, err)
	assert.Equal(t, "PERMISSION_DENIED", result)
}

// testErrorContractJSBridgeValidationErrorMissingArgs — JS bridge returns VALIDATION_ERROR for missing args.
func testErrorContractJSBridgeValidationErrorMissingArgs(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	result, err := env.Kernel.EvalTS(context.Background(), "__val_test_adv.ts", `
		var caught = "none";
		try { __go_brainkit_bus_schedule("every 1s"); }
		catch(e) { caught = e.code || "NO_CODE"; }
		return caught;
	`)
	require.NoError(t, err)
	assert.Equal(t, "VALIDATION_ERROR", result)
}

// testErrorContractJSBridgeRateLimited — JS bridge returns RATE_LIMITED error code.
func testErrorContractJSBridgeRateLimited(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Roles:       map[string]rbac.Role{"service": rbac.RoleService},
		DefaultRole: "service",
		BusRateLimits: map[string]float64{
			"service": 1, // 1 req/s — very low
		},
	})
	require.NoError(t, err)
	defer k.Close()

	// Deploy a .ts that hammers bus.publish
	_, err = k.Deploy(context.Background(), "rate-test-adv.ts", `
		var codes = [];
		for (var i = 0; i < 10; i++) {
			try { bus.publish("incoming.test-adv", { i: i }); }
			catch(e) { codes.push(e.code || "NO_CODE"); }
		}
		output(codes);
	`)
	require.NoError(t, err)

	result, err := k.EvalTS(context.Background(), "__rate_result_adv.ts", `return String(globalThis.__module_result || "[]");`)
	require.NoError(t, err)
	assert.Contains(t, result, "RATE_LIMITED")
}

// testErrorContractJSBridgeNotConfiguredSecrets — JS bridge secrets.get returns empty for nonexistent keys.
func testErrorContractJSBridgeNotConfiguredSecrets(t *testing.T, _ *suite.TestEnv) {
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test",
	})
	require.NoError(t, err)
	defer k.Close()

	result, err := k.EvalTS(context.Background(), "__secret_default_adv.ts", `
		var val = secrets.get("NONEXISTENT_KEY_12345_ADV");
		return val === "" ? "empty" : "unexpected:" + val;
	`)
	require.NoError(t, err)
	assert.Equal(t, "empty", result)
}
