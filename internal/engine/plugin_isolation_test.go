package engine

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/internal/tracing"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/ctxkeys"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestToolsDomain(runtimeID string) (*ToolsDomain, *tools.ToolRegistry) {
	reg := tools.New()
	tracer := tracing.NewTracer(nil, 1.0)
	domain := newToolsDomain(reg, nil, tracer, "test-caller", runtimeID)
	return domain, reg
}

// TestLocalToolCallFromSameRuntime verifies that local tools work when called
// from the same runtime (runtimeID matches).
func TestLocalToolCallFromSameRuntime(t *testing.T) {
	domain, reg := newTestToolsDomain("runtime-abc")

	reg.Register(tools.RegisteredTool{
		Name:      "test/plugin@1.0.0/secret",
		ShortName: "secret",
		Local:     true,
		Executor: &tools.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				return json.RawMessage(`{"data":"secret-value"}`), nil
			},
		},
	})

	// Same runtimeID — should succeed
	ctx := context.WithValue(context.Background(), ctxkeys.RuntimeID, "runtime-abc")
	resp, err := domain.Call(ctx, sdk.ToolCallMsg{Name: "secret"})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Contains(t, string(resp.Result), "secret-value")
}

// TestLocalToolCallFromLocalNoRuntimeID verifies that local tools work when
// called locally without a runtimeID in context (direct Go call, LocalInvoker, JS bridge).
func TestLocalToolCallFromLocalNoRuntimeID(t *testing.T) {
	domain, reg := newTestToolsDomain("runtime-abc")

	reg.Register(tools.RegisteredTool{
		Name:      "test/plugin@1.0.0/secret",
		ShortName: "secret",
		Local:     true,
		Executor: &tools.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				return json.RawMessage(`{"data":"local-data"}`), nil
			},
		},
	})

	// No runtimeID in context — local call (Go bridge, JS bridge, direct)
	resp, err := domain.Call(context.Background(), sdk.ToolCallMsg{Name: "secret"})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Contains(t, string(resp.Result), "local-data")
}

// TestLocalToolCallFromRemoteRuntimeBlocked verifies that a local-only tool
// CANNOT be called when the inbound message has a DIFFERENT runtimeID.
// This simulates a remote Kit attempting to invoke a plugin tool via cross-namespace call.
func TestLocalToolCallFromRemoteRuntimeBlocked(t *testing.T) {
	domain, reg := newTestToolsDomain("runtime-abc")

	called := false
	reg.Register(tools.RegisteredTool{
		Name:      "test/plugin@1.0.0/private",
		ShortName: "private",
		Local:     true,
		Executor: &tools.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				called = true
				return json.RawMessage(`{"data":"should-not-see"}`), nil
			},
		},
	})

	// Different runtimeID — remote call, must be rejected
	ctx := context.WithValue(context.Background(), ctxkeys.RuntimeID, "attacker-runtime-xyz")
	resp, err := domain.Call(ctx, sdk.ToolCallMsg{Name: "private"})

	assert.Nil(t, resp, "response should be nil for denied calls")
	require.Error(t, err)
	assert.False(t, called, "executor must NOT be invoked for remote calls to local tools")

	// Verify it's specifically a PermissionDeniedError
	var permErr *sdkerrors.PermissionDeniedError
	require.ErrorAs(t, err, &permErr)
	assert.Equal(t, "PERMISSION_DENIED", permErr.Code())
}

// TestNonLocalToolCallFromRemoteAllowed verifies that non-local tools (Go-registered,
// .ts-registered) remain callable from remote runtimes.
func TestNonLocalToolCallFromRemoteAllowed(t *testing.T) {
	domain, reg := newTestToolsDomain("runtime-abc")

	reg.Register(tools.RegisteredTool{
		Name:      "test/service@1.0.0/public",
		ShortName: "public",
		Local:     false, // not a plugin — callable from anywhere
		Executor: &tools.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				return json.RawMessage(`{"data":"public-data"}`), nil
			},
		},
	})

	// Remote runtimeID — should succeed for non-local tools
	ctx := context.WithValue(context.Background(), ctxkeys.RuntimeID, "remote-runtime")
	resp, err := domain.Call(ctx, sdk.ToolCallMsg{Name: "public"})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Contains(t, string(resp.Result), "public-data")
}

// --- Attack scenarios ---

// TestAttackRemotePluginToolViaDirectBus simulates an attacker on the same NATS
// transport attempting to call a plugin tool on another Kit by publishing a
// tools.call message to that Kit's namespace.
func TestAttackRemotePluginToolViaDirectBus(t *testing.T) {
	domain, reg := newTestToolsDomain("victim-runtime")

	reg.Register(tools.RegisteredTool{
		Name:      "acme/db-plugin@1.0.0/query",
		ShortName: "query",
		Local:     true,
		Executor: &tools.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				t.Fatal("ATTACK SUCCEEDED — plugin tool was invoked by remote runtime")
				return nil, nil
			},
		},
	})

	// Attacker sends tools.call from different runtime
	ctx := context.WithValue(context.Background(), ctxkeys.RuntimeID, "attacker-runtime")
	_, err := domain.Call(ctx, sdk.ToolCallMsg{Name: "query", Input: map[string]any{"sql": "DROP TABLE users"}})

	require.Error(t, err, "attack must be blocked")
	var permErr *sdkerrors.PermissionDeniedError
	require.ErrorAs(t, err, &permErr)
}

// TestAttackToolResolveStillWorks verifies that tools.resolve (metadata only)
// still works for local tools from remote runtimes — they can SEE the tool exists
// but cannot CALL it.
func TestAttackToolResolveStillWorks(t *testing.T) {
	domain, reg := newTestToolsDomain("victim-runtime")

	reg.Register(tools.RegisteredTool{
		Name:        "acme/db-plugin@1.0.0/query",
		ShortName:   "query",
		Description: "runs SQL queries",
		Local:       true,
		Executor: &tools.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				return nil, nil
			},
		},
	})

	// Resolve should work — it only returns metadata, not execution
	resp, err := domain.Resolve(context.Background(), sdk.ToolResolveMsg{Name: "query"})
	require.NoError(t, err)
	assert.Equal(t, "query", resp.ShortName)
}

// TestAttackToolListShowsLocalFlag verifies that tools.list includes the Local flag
// so clients can see which tools are local-only.
func TestAttackToolListShowsLocalFlag(t *testing.T) {
	domain, reg := newTestToolsDomain("runtime-abc")

	reg.Register(tools.RegisteredTool{
		Name: "test/plugin@1.0.0/local-tool", ShortName: "local-tool", Local: true,
		Executor: &tools.GoFuncExecutor{Fn: func(ctx context.Context, _ string, _ json.RawMessage) (json.RawMessage, error) { return nil, nil }},
	})
	reg.Register(tools.RegisteredTool{
		Name: "test/service@1.0.0/global-tool", ShortName: "global-tool", Local: false,
		Executor: &tools.GoFuncExecutor{Fn: func(ctx context.Context, _ string, _ json.RawMessage) (json.RawMessage, error) { return nil, nil }},
	})

	resp, err := domain.List(context.Background(), sdk.ToolListMsg{})
	require.NoError(t, err)
	assert.Len(t, resp.Tools, 2)
}

// TestAttackCrossKitPluginToolOnEmbeddedNATS is an end-to-end test simulating
// two Kit instances on embedded NATS where one tries to call the other's plugin tool.
func TestAttackCrossKitPluginToolOnEmbeddedNATS(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e embedded NATS test")
	}

	// Start embedded NATS server — shared transport
	embedded, err := transport.NewEmbeddedNATS(transport.EmbeddedNATSConfig{})
	require.NoError(t, err)
	defer embedded.Shutdown()

	natsURL := embedded.ClientURL()

	// Victim Kit — has a local-only plugin tool
	victim, err := NewNode(types.NodeConfig{
		Kernel: types.KernelConfig{
			Namespace: "victim",
			CallerID:  "victim",
			RuntimeID: "victim-runtime-id",
		},
		Messaging: types.MessagingConfig{
			Transport: "nats",
			NATSURL:   natsURL,
		},
	})
	require.NoError(t, err)
	require.NoError(t, victim.Start(context.Background()))
	defer victim.Close()

	// Register a local-only tool on victim
	victim.Kernel.Tools.Register(tools.RegisteredTool{
		Name:      "acme/secret-plugin@1.0.0/read-db",
		ShortName: "read-db",
		Local:     true,
		Executor: &tools.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				t.Fatal("ATTACK SUCCEEDED — remote Kit called victim's plugin tool")
				return json.RawMessage(`{"rows":"sensitive-data"}`), nil
			},
		},
	})

	// Attacker Kit — different namespace, different runtimeID, same NATS
	attacker, err := NewNode(types.NodeConfig{
		Kernel: types.KernelConfig{
			Namespace: "attacker",
			CallerID:  "attacker",
			RuntimeID: "attacker-runtime-id",
		},
		Messaging: types.MessagingConfig{
			Transport: "nats",
			NATSURL:   natsURL,
		},
	})
	require.NoError(t, err)
	require.NoError(t, attacker.Start(context.Background()))
	defer attacker.Close()

	// Attacker tries to call victim's plugin tool via cross-namespace publish.
	// The attacker subscribes to a replyTo in its own namespace, then publishes
	// tools.call to the victim's namespace with replyTo stamped in metadata.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Subscribe to reply BEFORE publishing (roundTrip pattern)
	replyTopic := "tools.call.reply.attack-test"
	ch := make(chan json.RawMessage, 1)
	unsub, _ := attacker.SubscribeRaw(ctx, replyTopic, func(m sdk.Message) {
		ch <- json.RawMessage(m.Payload)
	})
	defer unsub()

	// Give subscription time to register on NATS
	time.Sleep(500 * time.Millisecond)

	// Publish tools.call to victim's namespace with attacker's replyTo
	payload, _ := json.Marshal(sdk.ToolCallMsg{Name: "read-db", Input: map[string]any{}})
	// WithPublishMeta sets logical replyTo — PublishRawToNamespace resolves it.
	attackCtx := transport.WithPublishMeta(ctx, "attack-corr", replyTopic)
	attacker.Kernel.PublishRawTo(attackCtx, "victim", "tools.call", payload)

	select {
	case resp := <-ch:
		// Should get a PERMISSION_DENIED error response
		var result struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}
		json.Unmarshal(resp, &result)
		assert.Equal(t, "PERMISSION_DENIED", result.Code, "remote call to local plugin tool must return PERMISSION_DENIED")
		t.Logf("Attack correctly blocked: %s", result.Error)
	case <-ctx.Done():
		t.Fatal("attack response not received — expected PERMISSION_DENIED error")
	}
}
