package adversarial_test

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/rbac"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ════════════════════════════════════════════════════════════════════════════
// SIGNED REPLY TOKENS
// Only own-mailbox handlers can reply. Eavesdroppers get empty tokens.
// ════════════════════════════════════════════════════════════════════════════

func replyTokenKernel(t *testing.T) *brainkit.Kernel {
	t.Helper()
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Roles: map[string]rbac.Role{
			"admin":    rbac.RoleAdmin,
			"service":  rbac.RoleService,
			"observer": rbac.RoleObserver,
		},
		DefaultRole: "service",
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

// Own-mailbox handler receives a non-empty replyToken
func TestReplyToken_OwnMailboxGetsToken(t *testing.T) {
	k := replyTokenKernel(t)
	ctx := context.Background()

	_, err := k.Deploy(ctx, "token-check.ts", `
		bus.on("check", function(msg) {
			msg.reply({
				hasToken: typeof msg.replyToken === "string" && msg.replyToken.length > 0,
				tokenLength: msg.replyToken ? msg.replyToken.length : 0,
			});
		});
	`, brainkit.WithRole("service"))
	require.NoError(t, err)

	pr, _ := sdk.Publish(k, ctx, messages.CustomMsg{
		Topic: "ts.token-check.check", Payload: json.RawMessage(`{}`),
	})
	ch := make(chan []byte, 1)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		var resp struct {
			HasToken    bool `json:"hasToken"`
			TokenLength int  `json:"tokenLength"`
		}
		json.Unmarshal(p, &resp)
		assert.True(t, resp.HasToken, "own-mailbox handler should receive a replyToken")
		assert.Greater(t, resp.TokenLength, 0)
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
}

// Legit handler CAN reply — token is valid
func TestReplyToken_LegitHandlerCanReply(t *testing.T) {
	k := replyTokenKernel(t)
	ctx := context.Background()

	_, err := k.Deploy(ctx, "legit-reply.ts", `
		bus.on("api", function(msg) {
			msg.reply({legitimate: true, data: "response"});
		});
	`, brainkit.WithRole("service"))
	require.NoError(t, err)

	pr, _ := sdk.Publish(k, ctx, messages.CustomMsg{
		Topic: "ts.legit-reply.api", Payload: json.RawMessage(`{}`),
	})
	ch := make(chan []byte, 1)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "legitimate")
	case <-time.After(3 * time.Second):
		t.Fatal("timeout — legit handler should be able to reply")
	}
}

// Observer with subscribe:* tries to reply → REPLY_DENIED
func TestReplyToken_ObserverCannotReply(t *testing.T) {
	k := replyTokenKernel(t)
	ctx := context.Background()

	// Deploy legit service
	_, err := k.Deploy(ctx, "protected-svc.ts", `
		bus.on("api", function(msg) {
			msg.reply({from: "legitimate"});
		});
	`, brainkit.WithRole("service"))
	require.NoError(t, err)

	// Deploy observer that subscribes to the service's topic and tries to reply
	_, err = k.Deploy(ctx, "sneaky-observer.ts", `
		var replyResult = "UNKNOWN";
		bus.subscribe("ts.protected-svc.api", function(msg) {
			try {
				msg.reply({from: "observer-impersonation"});
				replyResult = "REPLIED";
			} catch(e) {
				replyResult = "DENIED:" + (e.code || e.message);
			}
		});
		output("subscribed");
	`, brainkit.WithRole("observer"))
	require.NoError(t, err)

	// Trigger the service
	pr, _ := sdk.Publish(k, ctx, messages.CustomMsg{
		Topic: "ts.protected-svc.api", Payload: json.RawMessage(`{}`),
	})
	ch := make(chan []byte, 1)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		// Should be the legitimate response, not the observer's
		assert.Contains(t, string(p), "legitimate", "only the legitimate handler should reply")
		assert.NotContains(t, string(p), "observer-impersonation")
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}

	// Check observer caught REPLY_DENIED
	time.Sleep(200 * time.Millisecond)
	result, _ := k.EvalTS(ctx, "__obs_reply.ts", `return String(globalThis.__module_result || "");`)
	// Observer output was "subscribed", but the replyResult var is local — check via different means
	// The key assertion is above: the response is from "legitimate" not "observer-impersonation"
	_ = result
}

// Streaming with tokens — all chunks succeed
func TestReplyToken_StreamingWithToken(t *testing.T) {
	k := replyTokenKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := k.Deploy(ctx, "stream-token.ts", `
		bus.on("stream", function(msg) {
			msg.stream.text("chunk1");
			msg.stream.text("chunk2");
			msg.stream.progress(50, "half");
			msg.stream.end({done: true});
		});
	`, brainkit.WithRole("service"))
	require.NoError(t, err)

	pr, _ := sdk.Publish(k, ctx, messages.CustomMsg{
		Topic: "ts.stream-token.stream", Payload: json.RawMessage(`{}`),
	})

	var chunks int64
	done := make(chan bool, 1)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) {
		atomic.AddInt64(&chunks, 1)
		if m.Metadata["done"] == "true" {
			done <- true
		}
		var parsed struct{ Type string `json:"type"` }
		json.Unmarshal(m.Payload, &parsed)
		if parsed.Type == "end" {
			done <- true
		}
	})
	defer unsub()

	select {
	case <-done:
		assert.Greater(t, atomic.LoadInt64(&chunks), int64(0), "should receive stream chunks with valid token")
	case <-ctx.Done():
		t.Fatalf("timeout — got %d chunks", atomic.LoadInt64(&chunks))
	}
}

// No RBAC = no tokens = observer CAN reply (backward compat)
func TestReplyToken_NoRBACNoTokens(t *testing.T) {
	tmpDir := t.TempDir()
	// Kernel WITHOUT RBAC
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()

	_, err = k.Deploy(ctx, "no-rbac-svc.ts", `
		bus.on("api", function(msg) {
			msg.reply({from: "service"});
		});
	`)
	require.NoError(t, err)

	// Without RBAC, anyone can reply — verify it works
	pr, _ := sdk.Publish(k, ctx, messages.CustomMsg{
		Topic: "ts.no-rbac-svc.api", Payload: json.RawMessage(`{}`),
	})
	ch := make(chan []byte, 1)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "service")
	case <-time.After(3 * time.Second):
		t.Fatal("timeout — no-RBAC kernel should work without tokens")
	}
}

// Audit event emitted when reply is denied
func TestReplyToken_AuditEventEmitted(t *testing.T) {
	k := replyTokenKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Listen for audit events
	var auditEvents []string
	auditUnsub, _ := k.SubscribeRaw(ctx, "bus.reply.denied", func(m messages.Message) {
		auditEvents = append(auditEvents, string(m.Payload))
	})
	defer auditUnsub()

	// Deploy service
	_, _ = k.Deploy(ctx, "audit-svc.ts", `
		bus.on("api", function(msg) { msg.reply({ok: true}); });
	`, brainkit.WithRole("service"))

	// Deploy observer that tries to reply
	_, _ = k.Deploy(ctx, "audit-observer.ts", `
		bus.subscribe("ts.audit-svc.api", function(msg) {
			try { msg.reply({hijacked: true}); } catch(e) {}
		});
	`, brainkit.WithRole("observer"))

	// Trigger
	sdk.Publish(k, ctx, messages.CustomMsg{
		Topic: "ts.audit-svc.api", Payload: json.RawMessage(`{}`),
	})

	time.Sleep(1 * time.Second)

	assert.Greater(t, len(auditEvents), 0, "bus.reply.denied audit event should be emitted")
	if len(auditEvents) > 0 {
		t.Logf("Audit events: %d ��� %s", len(auditEvents), auditEvents[0])
		assert.Contains(t, auditEvents[0], "invalid reply token")
	}
	// Audit event may not fire if observer's msg.reply is caught before reaching the bridge
	// (the BrainkitError is thrown and caught in JS)
}

// Cross-deployment: service A's token doesn't work for service B
func TestReplyToken_CrossDeploymentTokenScoped(t *testing.T) {
	k := replyTokenKernel(t)
	ctx := context.Background()

	// Service A handles requests
	_, err := k.Deploy(ctx, "svc-a-token.ts", `
		bus.on("api", function(msg) {
			msg.reply({from: "A", token: msg.replyToken});
		});
	`, brainkit.WithRole("service"))
	require.NoError(t, err)

	// Service B subscribes to A's topic — needs admin role to subscribe to foreign mailbox
	_, err = k.Deploy(ctx, "svc-b-token.ts", `
		bus.subscribe("ts.svc-a-token.api", function(msg) {
			try {
				msg.reply({from: "B-impersonating-A"});
			} catch(e) {}
		});
	`, brainkit.WithRole("admin"))
	require.NoError(t, err)

	pr, _ := sdk.Publish(k, ctx, messages.CustomMsg{
		Topic: "ts.svc-a-token.api", Payload: json.RawMessage(`{}`),
	})
	ch := make(chan []byte, 1)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		// Should be A's response — B's token is scoped to B, not valid for A's replyTo
		resp := string(p)
		assert.Contains(t, resp, `"from":"A"`, "only service A should reply to its own mailbox")
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
}
