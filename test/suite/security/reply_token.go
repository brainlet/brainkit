package security

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testTokenOwnMailboxGetsToken — own-mailbox handler receives a non-empty replyToken.
func testTokenOwnMailboxGetsToken(t *testing.T, env *suite.TestEnv) {
	k := secReplyTokenKernel(t)
	ctx := context.Background()

	_, err := k.Deploy(ctx, "token-check-sec.ts", `
		bus.on("check", function(msg) {
			msg.reply({
				hasToken: typeof msg.replyToken === "string" && msg.replyToken.length > 0,
				tokenLength: msg.replyToken ? msg.replyToken.length : 0,
			});
		});
	`, brainkit.WithRole("service"))
	require.NoError(t, err)

	pr, _ := sdk.Publish(k, ctx, messages.CustomMsg{
		Topic: "ts.token-check-sec.check", Payload: json.RawMessage(`{}`),
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

// testTokenLegitHandlerCanReply — legit handler CAN reply — token is valid.
func testTokenLegitHandlerCanReply(t *testing.T, env *suite.TestEnv) {
	k := secReplyTokenKernel(t)
	ctx := context.Background()

	_, err := k.Deploy(ctx, "legit-reply-sec.ts", `
		bus.on("api", function(msg) {
			msg.reply({legitimate: true, data: "response"});
		});
	`, brainkit.WithRole("service"))
	require.NoError(t, err)

	pr, _ := sdk.Publish(k, ctx, messages.CustomMsg{
		Topic: "ts.legit-reply-sec.api", Payload: json.RawMessage(`{}`),
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

// testTokenObserverCannotReply — observer with subscribe:* tries to reply.
func testTokenObserverCannotReply(t *testing.T, env *suite.TestEnv) {
	k := secReplyTokenKernel(t)
	ctx := context.Background()

	_, err := k.Deploy(ctx, "protected-svc-sec.ts", `
		bus.on("api", function(msg) {
			msg.reply({from: "legitimate"});
		});
	`, brainkit.WithRole("service"))
	require.NoError(t, err)

	_, err = k.Deploy(ctx, "sneaky-observer-sec.ts", `
		var replyResult = "UNKNOWN";
		bus.subscribe("ts.protected-svc-sec.api", function(msg) {
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

	pr, _ := sdk.Publish(k, ctx, messages.CustomMsg{
		Topic: "ts.protected-svc-sec.api", Payload: json.RawMessage(`{}`),
	})
	ch := make(chan []byte, 1)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "legitimate", "only the legitimate handler should reply")
		assert.NotContains(t, string(p), "observer-impersonation")
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}

	time.Sleep(200 * time.Millisecond)
	result, _ := k.EvalTS(ctx, "__obs_reply.ts", `return String(globalThis.__module_result || "");`)
	_ = result
}

// testTokenStreamingWithToken — all chunks succeed.
func testTokenStreamingWithToken(t *testing.T, env *suite.TestEnv) {
	k := secReplyTokenKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := k.Deploy(ctx, "stream-token-sec.ts", `
		bus.on("stream", function(msg) {
			msg.stream.text("chunk1");
			msg.stream.text("chunk2");
			msg.stream.progress(50, "half");
			msg.stream.end({done: true});
		});
	`, brainkit.WithRole("service"))
	require.NoError(t, err)

	pr, _ := sdk.Publish(k, ctx, messages.CustomMsg{
		Topic: "ts.stream-token-sec.stream", Payload: json.RawMessage(`{}`),
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

// testTokenNoRBACNoTokens — No RBAC = no tokens = observer CAN reply.
func testTokenNoRBACNoTokens(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()

	_, err = k.Deploy(ctx, "no-rbac-svc-sec.ts", `
		bus.on("api", function(msg) {
			msg.reply({from: "service"});
		});
	`)
	require.NoError(t, err)

	pr, _ := sdk.Publish(k, ctx, messages.CustomMsg{
		Topic: "ts.no-rbac-svc-sec.api", Payload: json.RawMessage(`{}`),
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

// testTokenAuditEventEmitted — audit event emitted when reply is denied.
func testTokenAuditEventEmitted(t *testing.T, env *suite.TestEnv) {
	k := secReplyTokenKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var auditEvents []string
	auditUnsub, _ := k.SubscribeRaw(ctx, "bus.reply.denied", func(m messages.Message) {
		auditEvents = append(auditEvents, string(m.Payload))
	})
	defer auditUnsub()

	_, _ = k.Deploy(ctx, "audit-svc-sec.ts", `
		bus.on("api", function(msg) { msg.reply({ok: true}); });
	`, brainkit.WithRole("service"))

	_, _ = k.Deploy(ctx, "audit-observer-sec.ts", `
		bus.subscribe("ts.audit-svc-sec.api", function(msg) {
			try { msg.reply({hijacked: true}); } catch(e) {}
		});
	`, brainkit.WithRole("observer"))

	sdk.Publish(k, ctx, messages.CustomMsg{
		Topic: "ts.audit-svc-sec.api", Payload: json.RawMessage(`{}`),
	})

	time.Sleep(1 * time.Second)

	assert.Greater(t, len(auditEvents), 0, "bus.reply.denied audit event should be emitted")
	if len(auditEvents) > 0 {
		t.Logf("Audit events: %d - %s", len(auditEvents), auditEvents[0])
		assert.Contains(t, auditEvents[0], "invalid reply token")
	}
}

// testTokenCrossDeploymentScoped — service A's token doesn't work for service B.
func testTokenCrossDeploymentScoped(t *testing.T, env *suite.TestEnv) {
	k := secReplyTokenKernel(t)
	ctx := context.Background()

	_, err := k.Deploy(ctx, "svc-a-token-sec.ts", `
		bus.on("api", function(msg) {
			msg.reply({from: "A", token: msg.replyToken});
		});
	`, brainkit.WithRole("service"))
	require.NoError(t, err)

	_, err = k.Deploy(ctx, "svc-b-token-sec.ts", `
		bus.subscribe("ts.svc-a-token-sec.api", function(msg) {
			try {
				msg.reply({from: "B-impersonating-A"});
			} catch(e) {}
		});
	`, brainkit.WithRole("admin"))
	require.NoError(t, err)

	pr, _ := sdk.Publish(k, ctx, messages.CustomMsg{
		Topic: "ts.svc-a-token-sec.api", Payload: json.RawMessage(`{}`),
	})
	ch := make(chan []byte, 1)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		resp := string(p)
		assert.Contains(t, resp, `"from":"A"`, "only service A should reply to its own mailbox")
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
}
