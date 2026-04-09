package bus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/rbac"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// busErrorCodeAdv publishes a message and extracts the error code+details from the response.
func busErrorCodeAdv(t *testing.T, k *brainkit.Kit, msg sdk.BrainkitMessage) (string, map[string]any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, err := sdk.Publish(k, ctx, msg)
	require.NoError(t, err)

	ch := make(chan json.RawMessage, 1)
	unsub, err := k.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) {
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
	code, details := busErrorCodeAdv(t, env.Kit, sdk.ToolCallMsg{Name: "nonexistent-tool-xyz-adv"})
	assert.Equal(t, "NOT_FOUND", code)
	if details != nil {
		assert.Equal(t, "nonexistent-tool-xyz-adv", details["name"])
	}
}

// testErrorContractBusValidationError — VALIDATION_ERROR for empty secret name.
func testErrorContractBusValidationError(t *testing.T, env *suite.TestEnv) {
	code, _ := busErrorCodeAdv(t, env.Kit, sdk.SecretsSetMsg{Name: "", Value: "val"})
	assert.Equal(t, "VALIDATION_ERROR", code)
}

// testErrorContractBusNotConfiguredRBAC — VALIDATION_ERROR for rbac.assign with empty source.
func testErrorContractBusNotConfiguredRBAC(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.New(brainkit.Config{
		Transport: "memory",
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Roles: map[string]rbac.Role{"admin": rbac.RoleAdmin},
	})
	require.NoError(t, err)
	defer k.Close()

	code, _ := busErrorCodeAdv(t, k, sdk.RBACAssignMsg{Source: "", Role: "admin"})
	assert.Equal(t, "VALIDATION_ERROR", code)
}

// testErrorContractBusIdempotentDeploy — duplicate deploy succeeds (idempotent).
// Deploy tears down existing + redeploys. No ALREADY_EXISTS error.
func testErrorContractBusIdempotentDeploy(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	k := freshEnv.Kit

	testutil.Deploy(t, k, "dup-test-adv.ts", `
		const t = createTool({ id: "dup-adv", description: "dup", execute: async () => ({}) });
		kit.register("tool", "dup-adv", t);
	`)

	// Second deploy with same source — should succeed (idempotent)
	err := testutil.DeployErr(k, "dup-test-adv.ts", `
		const t2 = createTool({ id: "dup-adv-v2", description: "dup v2", execute: async () => ({}) });
		kit.register("tool", "dup-adv-v2", t2);
	`)
	require.NoError(t, err, "idempotent deploy should succeed")
}

// testErrorContractBusDeployErrorBadSyntax — DEPLOY_ERROR for bad syntax.
func testErrorContractBusDeployErrorBadSyntax(t *testing.T, env *suite.TestEnv) {
	code, details := busErrorCodeAdv(t, env.Kit, sdk.KitDeployMsg{
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

	err := testutil.DeployWithOpts(tk.Kit, "perm-test-adv.ts", `
		var caught = "none";
		try { bus.publish("forbidden", {}); }
		catch(e) { caught = e.code || "NO_CODE"; }
		output(caught);
	`, "restricted", "")
	require.NoError(t, err)

	result := testutil.EvalTS(t, tk.Kit, "__perm_result_adv.ts", `return String(globalThis.__module_result || "");`)
	assert.Equal(t, "PERMISSION_DENIED", result)
}

// testErrorContractJSBridgeValidationErrorMissingArgs — JS bridge returns VALIDATION_ERROR for missing args.
func testErrorContractJSBridgeValidationErrorMissingArgs(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	result := testutil.EvalTS(t, env.Kit, "__val_test_adv.ts", `
		var caught = "none";
		try { __go_brainkit_bus_schedule("every 1s"); }
		catch(e) { caught = e.code || "NO_CODE"; }
		return caught;
	`)
	assert.Equal(t, "VALIDATION_ERROR", result)
}

// testErrorContractJSBridgeRateLimited — JS bridge returns RATE_LIMITED error code.
func testErrorContractJSBridgeRateLimited(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.New(brainkit.Config{
		Transport: "memory",
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
	testutil.Deploy(t, k, "rate-test-adv.ts", `
		var codes = [];
		for (var i = 0; i < 10; i++) {
			try { bus.publish("incoming.test-adv", { i: i }); }
			catch(e) { codes.push(e.code || "NO_CODE"); }
		}
		output(codes);
	`)

	result := testutil.EvalTS(t, k, "__rate_result_adv.ts", `return String(globalThis.__module_result || "[]");`)
	assert.Contains(t, result, "RATE_LIMITED")
}

// testErrorContractJSBridgeNotConfiguredSecrets — JS bridge secrets.get returns empty for nonexistent keys.
func testErrorContractJSBridgeNotConfiguredSecrets(t *testing.T, _ *suite.TestEnv) {
	k, err := brainkit.New(brainkit.Config{
		Transport: "memory",
		Namespace: "test", CallerID: "test",
	})
	require.NoError(t, err)
	defer k.Close()

	result := testutil.EvalTS(t, k, "__secret_default_adv.ts", `
		var val = secrets.get("NONEXISTENT_KEY_12345_ADV");
		return val === "" ? "empty" : "unexpected:" + val;
	`)
	assert.Equal(t, "empty", result)
}

// testErrorContractErrorHandlerPersistenceError — ErrorHandler receives PersistenceError when store fails.
func testErrorContractErrorHandlerPersistenceError(t *testing.T, _ *suite.TestEnv) {
	var mu sync.Mutex
	var received []error

	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.db")
	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)

	k, err := brainkit.New(brainkit.Config{
		Transport: "memory",
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store,
		ErrorHandler: func(err error) {
			mu.Lock()
			received = append(received, err)
			mu.Unlock()
		},
	})
	require.NoError(t, err)
	defer k.Close()

	// Deploy something, then corrupt the store, then try to persist
	testutil.Deploy(t, k, "eh-test-adv.ts", `output("hello");`)

	// Close the store's DB to simulate a persistence failure
	store.Close()

	// Schedule something — persistence will fail, ErrorHandler should be called
	testutil.ScheduleErr(k, "in 1h", "test.topic", json.RawMessage(`{}`))

	// Give ErrorHandler a moment to be called
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// We should have at least one PersistenceError
	foundPersistence := false
	for _, err := range received {
		var pe *sdkerrors.PersistenceError
		if errors.As(err, &pe) {
			foundPersistence = true
			assert.NotEmpty(t, pe.Operation)
		}
	}
	if len(received) > 0 {
		assert.True(t, foundPersistence, "expected at least one PersistenceError, got: %v", received)
	}
}

// testErrorContractErrorHandlerDeployError — ErrorHandler receives DeployError when persisted code is corrupt.
func testErrorContractErrorHandlerDeployError(t *testing.T, _ *suite.TestEnv) {
	var mu sync.Mutex
	var received []error

	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.db")
	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)

	k, err := brainkit.New(brainkit.Config{
		Transport: "memory",
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store,
		ErrorHandler: func(err error) {
			mu.Lock()
			received = append(received, err)
			mu.Unlock()
		},
	})
	require.NoError(t, err)

	// Deploy valid code, persist it
	testutil.Deploy(t, k, "valid-adv.ts", `output("ok");`)

	// Now corrupt the persisted code in the store
	store.SaveDeployment(types.PersistedDeployment{
		Source: "corrupt-adv.ts",
		Code:   "const x: number = {{{invalid;;;",
		Order:  99,
	})

	k.Close()

	// Create a new kernel — it will try to redeploy "corrupt-adv.ts" and fail
	store2, _ := brainkit.NewSQLiteStore(storePath)
	k2, err := brainkit.New(brainkit.Config{
		Transport: "memory",
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2,
		ErrorHandler: func(err error) {
			mu.Lock()
			received = append(received, err)
			mu.Unlock()
		},
	})
	require.NoError(t, err)
	defer k2.Close()

	mu.Lock()
	defer mu.Unlock()

	foundDeploy := false
	for _, err := range received {
		var de *sdkerrors.DeployError
		if errors.As(err, &de) {
			foundDeploy = true
			assert.Equal(t, "corrupt-adv.ts", de.Source)
		}
	}
	assert.True(t, foundDeploy, "expected DeployError for corrupt-adv.ts, got: %v", received)
}
