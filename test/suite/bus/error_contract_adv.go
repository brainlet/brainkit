package bus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
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
