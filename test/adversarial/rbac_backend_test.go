package adversarial_test

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/rbac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRBACBackend_EnforcementOnEveryBackend — RBAC publish deny works on every transport.
func TestRBACBackend_EnforcementOnEveryBackend(t *testing.T) {
	for _, backend := range testutil.AllBackends(t) {
		t.Run(backend, func(t *testing.T) {
			transport := testutil.CreateTestTransport(t, backend)
			tmpDir := t.TempDir()

			k, err := brainkit.NewKernel(brainkit.KernelConfig{
				Namespace: "test", CallerID: "test", FSRoot: tmpDir,
				Transport: transport,
				Roles: map[string]rbac.Role{
					"observer": rbac.RoleObserver,
				},
				DefaultRole: "observer",
			})
			require.NoError(t, err)
			defer k.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			_, err = k.Deploy(ctx, "rbac-backend.ts", `
				var caught = "ALLOWED";
				try { bus.publish("forbidden.topic", {}); }
				catch(e) { caught = "DENIED"; }
				output(caught);
			`, brainkit.WithRole("observer"))
			require.NoError(t, err)

			result, _ := k.EvalTS(ctx, "__rb.ts", `return String(globalThis.__module_result || "");`)
			assert.Equal(t, "DENIED", result, "RBAC should enforce on %s backend", backend)
		})
	}
}

// TestRBACBackend_ToolCallOnEveryBackend — service role can call tools on every backend.
func TestRBACBackend_ToolCallOnEveryBackend(t *testing.T) {
	for _, backend := range testutil.AllBackends(t) {
		t.Run(backend, func(t *testing.T) {
			transport := testutil.CreateTestTransport(t, backend)
			tmpDir := t.TempDir()

			k, err := brainkit.NewKernel(brainkit.KernelConfig{
				Namespace: "test", CallerID: "test", FSRoot: tmpDir,
				Transport: transport,
				Roles: map[string]rbac.Role{
					"service": rbac.RoleService,
				},
				DefaultRole: "service",
			})
			require.NoError(t, err)
			defer k.Close()

			type echoIn struct{ Message string `json:"message"` }
			brainkit.RegisterTool(k, "echo", registry.TypedTool[echoIn]{
				Description: "echoes",
				Execute: func(ctx context.Context, in echoIn) (any, error) {
					return map[string]string{"echoed": in.Message}, nil
				},
			})

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			_, err = k.Deploy(ctx, "rbac-tool-backend.ts", `
				var caught = "ALLOWED";
				try { await tools.call("echo", {message: "backend-test"}); }
				catch(e) { caught = "DENIED:" + (e.message || ""); }
				output(caught);
			`, brainkit.WithRole("service"))
			require.NoError(t, err)

			result, _ := k.EvalTS(ctx, "__rtb.ts", `return String(globalThis.__module_result || "");`)
			assert.Equal(t, "ALLOWED", result, "service should call tools on %s", backend)
		})
	}
}
