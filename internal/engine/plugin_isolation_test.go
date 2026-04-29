package engine

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	toolreg "github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/internal/tracing"
	"github.com/brainlet/brainkit/internal/transport"
	transportbackends "github.com/brainlet/brainkit/internal/transport/backends"
	"github.com/brainlet/brainkit/internal/types"
	bkmodule "github.com/brainlet/brainkit/module"
	toolsmod "github.com/brainlet/brainkit/modules/tools"
	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/ctxkeys"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestToolsDomain(runtimeID string) (*toolsmod.Domain, *toolreg.ToolRegistry) {
	reg := toolreg.New()
	tracer := tracing.NewTracer(nil, 1.0)
	domain := toolsmod.NewDomain(reg, nil, tracer, nil, "test-caller", runtimeID)
	return domain, reg
}

// TestLocalToolCallFromSameRuntime verifies that local tools work when called
// from the same runtime (runtimeID matches).
func TestLocalToolCallFromSameRuntime(t *testing.T) {
	domain, reg := newTestToolsDomain("runtime-abc")

	reg.Register(toolreg.RegisteredTool{
		Name:      "test/plugin@1.0.0/secret",
		ShortName: "secret",
		Local:     true,
		Executor: &toolreg.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				return json.RawMessage(`{"data":"secret-value"}`), nil
			},
		},
	})

	// Same runtimeID — should succeed
	ctx := context.WithValue(context.Background(), ctxkeys.RuntimeID, "runtime-abc")
	resp, err := domain.Call(ctx, bkmodule.ToolCallRequest{Name: "secret"})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Contains(t, string(resp.Result), "secret-value")
}

// TestLocalToolCallFromLocalNoRuntimeID verifies that local tools work when
// called locally without a runtimeID in context (direct Go call, LocalInvoker, JS bridge).
func TestLocalToolCallFromLocalNoRuntimeID(t *testing.T) {
	domain, reg := newTestToolsDomain("runtime-abc")

	reg.Register(toolreg.RegisteredTool{
		Name:      "test/plugin@1.0.0/secret",
		ShortName: "secret",
		Local:     true,
		Executor: &toolreg.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				return json.RawMessage(`{"data":"local-data"}`), nil
			},
		},
	})

	// No runtimeID in context — local call (Go bridge, JS bridge, direct)
	resp, err := domain.Call(context.Background(), bkmodule.ToolCallRequest{Name: "secret"})
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
	reg.Register(toolreg.RegisteredTool{
		Name:      "test/plugin@1.0.0/private",
		ShortName: "private",
		Local:     true,
		Executor: &toolreg.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				called = true
				return json.RawMessage(`{"data":"should-not-see"}`), nil
			},
		},
	})

	// Different runtimeID — remote call, must be rejected
	ctx := context.WithValue(context.Background(), ctxkeys.RuntimeID, "attacker-runtime-xyz")
	resp, err := domain.Call(ctx, bkmodule.ToolCallRequest{Name: "private"})

	assert.Nil(t, resp, "response should be nil for denied calls")
	require.Error(t, err)
	assert.False(t, called, "executor must NOT be invoked for remote calls to local tools")

	// Verify it's specifically a ValidationError (local-only tool called from remote)
	var valErr *sdkerrors.ValidationError
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "VALIDATION_ERROR", valErr.Code())
}

// TestNonLocalToolCallFromRemoteAllowed verifies that non-local tools (Go-registered,
// .ts-registered) remain callable from remote runtimes.
func TestNonLocalToolCallFromRemoteAllowed(t *testing.T) {
	domain, reg := newTestToolsDomain("runtime-abc")

	reg.Register(toolreg.RegisteredTool{
		Name:      "test/service@1.0.0/public",
		ShortName: "public",
		Local:     false, // not a plugin — callable from anywhere
		Executor: &toolreg.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				return json.RawMessage(`{"data":"public-data"}`), nil
			},
		},
	})

	// Remote runtimeID — should succeed for non-local tools
	ctx := context.WithValue(context.Background(), ctxkeys.RuntimeID, "remote-runtime")
	resp, err := domain.Call(ctx, bkmodule.ToolCallRequest{Name: "public"})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Contains(t, string(resp.Result), "public-data")
}

// --- Attack scenarios ---

// TestAttackRemotePluginToolViaDirectBus simulates an attacker on the same NATS
// transport attempting to call a plugin tool on another Kit by publishing a
// toolreg.call message to that Kit's namespace.
func TestAttackRemotePluginToolViaDirectBus(t *testing.T) {
	domain, reg := newTestToolsDomain("victim-runtime")

	reg.Register(toolreg.RegisteredTool{
		Name:      "acme/db-plugin@1.0.0/query",
		ShortName: "query",
		Local:     true,
		Executor: &toolreg.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				t.Fatal("ATTACK SUCCEEDED — plugin tool was invoked by remote runtime")
				return nil, nil
			},
		},
	})

	// Attacker sends toolreg.call from different runtime
	ctx := context.WithValue(context.Background(), ctxkeys.RuntimeID, "attacker-runtime")
	_, err := domain.Call(ctx, bkmodule.ToolCallRequest{Name: "query", Input: map[string]any{"sql": "DROP TABLE users"}})

	require.Error(t, err, "attack must be blocked")
	var valErr *sdkerrors.ValidationError
	require.ErrorAs(t, err, &valErr)
}

// TestAttackToolResolveStillWorks verifies that toolreg.resolve (metadata only)
// still works for local tools from remote runtimes — they can SEE the tool exists
// but cannot CALL it.
func TestAttackToolResolveStillWorks(t *testing.T) {
	domain, reg := newTestToolsDomain("victim-runtime")

	reg.Register(toolreg.RegisteredTool{
		Name:        "acme/db-plugin@1.0.0/query",
		ShortName:   "query",
		Description: "runs SQL queries",
		Local:       true,
		Executor: &toolreg.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				return nil, nil
			},
		},
	})

	// Resolve should work — it only returns metadata, not execution
	resp, err := domain.Resolve(context.Background(), bkmodule.ToolResolveRequest{Name: "query"})
	require.NoError(t, err)
	assert.Equal(t, "query", resp.ShortName)
}

// TestAttackToolListShowsLocalFlag verifies that toolreg.list includes the Local flag
// so clients can see which tools are local-only.
func TestAttackToolListShowsLocalFlag(t *testing.T) {
	domain, reg := newTestToolsDomain("runtime-abc")

	reg.Register(toolreg.RegisteredTool{
		Name: "test/plugin@1.0.0/local-tool", ShortName: "local-tool", Local: true,
		Executor: &toolreg.GoFuncExecutor{Fn: func(ctx context.Context, _ string, _ json.RawMessage) (json.RawMessage, error) { return nil, nil }},
	})
	reg.Register(toolreg.RegisteredTool{
		Name: "test/service@1.0.0/global-tool", ShortName: "global-tool", Local: false,
		Executor: &toolreg.GoFuncExecutor{Fn: func(ctx context.Context, _ string, _ json.RawMessage) (json.RawMessage, error) { return nil, nil }},
	})

	resp, err := domain.List(context.Background(), bkmodule.ToolListRequest{})
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
	embedded, err := transportbackends.NewEmbeddedNATS(transportbackends.EmbeddedNATSConfig{})
	require.NoError(t, err)
	defer embedded.Shutdown()

	natsURL := embedded.ClientURL()
	victimTransport, err := transportbackends.NewTransportSet(transport.TransportConfig{
		Type:      "nats",
		Namespace: "victim",
		NATSURL:   natsURL,
	})
	require.NoError(t, err)

	// Victim Kit — has a local-only plugin tool
	victim, err := NewNode(types.NodeConfig{
		Kernel: types.KernelConfig{
			Namespace: "victim",
			CallerID:  "victim",
			RuntimeID: "victim-runtime-id",
			Transport: victimTransport,
		},
	})
	require.NoError(t, err)
	require.NoError(t, victim.Start(context.Background()))
	defer victim.Close()
	_, err = victim.Kernel.MountCommand(context.Background(), bkmodule.Command(func(ctx context.Context, req toolmsg.ToolCallMsg) (*toolmsg.ToolCallResp, error) {
		resp, err := victim.Kernel.CallTool(ctx, bkmodule.ToolCallRequest{Name: req.Name, Input: req.Input})
		if err != nil || resp == nil {
			return nil, err
		}
		return &toolmsg.ToolCallResp{Result: resp.Result}, nil
	}))
	require.NoError(t, err)

	// Register a local-only tool on victim
	victim.Kernel.Tools.Register(toolreg.RegisteredTool{
		Name:      "acme/secret-plugin@1.0.0/read-db",
		ShortName: "read-db",
		Local:     true,
		Executor: &toolreg.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				t.Fatal("ATTACK SUCCEEDED — remote Kit called victim's plugin tool")
				return json.RawMessage(`{"rows":"sensitive-data"}`), nil
			},
		},
	})

	// Attacker Kit — different namespace, different runtimeID, same NATS
	attackerTransport, err := transportbackends.NewTransportSet(transport.TransportConfig{
		Type:      "nats",
		Namespace: "attacker",
		NATSURL:   natsURL,
	})
	require.NoError(t, err)
	attacker, err := NewNode(types.NodeConfig{
		Kernel: types.KernelConfig{
			Namespace: "attacker",
			CallerID:  "attacker",
			RuntimeID: "attacker-runtime-id",
			Transport: attackerTransport,
		},
	})
	require.NoError(t, err)
	require.NoError(t, attacker.Start(context.Background()))
	defer attacker.Close()

	// Attacker tries to call victim's plugin tool via cross-namespace publish.
	// The attacker subscribes to a replyTo in its own namespace, then publishes
	// toolreg.call to the victim's namespace with replyTo stamped in metadata.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Subscribe to reply BEFORE publishing (roundTrip pattern)
	replyTopic := "toolreg.call.reply.attack-test"
	ch := make(chan json.RawMessage, 1)
	unsub, _ := attacker.SubscribeRaw(ctx, replyTopic, func(m sdk.Message) {
		ch <- json.RawMessage(m.Payload)
	})
	defer unsub()

	// Give subscription time to register on NATS
	time.Sleep(500 * time.Millisecond)

	// Publish toolreg.call to victim's namespace with attacker's replyTo
	payload, _ := json.Marshal(toolmsg.ToolCallMsg{Name: "read-db", Input: map[string]any{}})
	// WithPublishMeta sets logical replyTo — PublishRawToNamespace resolves it.
	attackCtx := transport.WithPublishMeta(ctx, "attack-corr", replyTopic)
	attacker.Kernel.PublishRawTo(attackCtx, "victim", "toolreg.call", payload)

	select {
	case resp := <-ch:
		// Should get a VALIDATION_ERROR response (runtimeId check rejects remote calls to local-only tools)
		envelope, err := sdk.DecodeEnvelope(resp)
		require.NoError(t, err)
		require.NotNil(t, envelope.Error)
		assert.Equal(t, "VALIDATION_ERROR", envelope.Error.Code, "remote call to local plugin tool must return VALIDATION_ERROR")
		t.Logf("Attack correctly blocked: %s", envelope.Error.Message)
	case <-ctx.Done():
		t.Fatal("attack response not received — expected VALIDATION_ERROR error")
	}
}
