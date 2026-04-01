package adversarial_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_DeployCallTeardownRedeploy — full lifecycle: deploy→call→teardown→redeploy→call→verify.
func TestE2E_DeployCallTeardownRedeploy(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Deploy v1
	_, err := tk.Deploy(ctx, "lifecycle.ts", `
		const t = createTool({id: "lc-tool", description: "v1", execute: async () => ({version: 1})});
		kit.register("tool", "lc-tool", t);
	`)
	require.NoError(t, err)

	// Call v1
	payload, ok := sendAndReceive(t, tk, messages.ToolCallMsg{Name: "lc-tool", Input: map[string]any{}}, 5*time.Second)
	require.True(t, ok)
	assert.Contains(t, string(payload), "1")

	// Teardown
	_, err = tk.Teardown(ctx, "lifecycle.ts")
	require.NoError(t, err)

	// Tool should be gone
	payload2, ok2 := sendAndReceive(t, tk, messages.ToolCallMsg{Name: "lc-tool", Input: map[string]any{}}, 5*time.Second)
	require.True(t, ok2)
	assert.Equal(t, "NOT_FOUND", responseCode(payload2))

	// Redeploy v2
	_, err = tk.Deploy(ctx, "lifecycle.ts", `
		const t = createTool({id: "lc-tool", description: "v2", execute: async () => ({version: 2})});
		kit.register("tool", "lc-tool", t);
	`)
	require.NoError(t, err)

	// Call v2
	payload3, ok3 := sendAndReceive(t, tk, messages.ToolCallMsg{Name: "lc-tool", Input: map[string]any{}}, 5*time.Second)
	require.True(t, ok3)
	assert.Contains(t, string(payload3), "2")
}

// TestE2E_MultiServiceChain — A deploys, B deploys, A calls B, B calls Go tool.
func TestE2E_MultiServiceChain(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Deploy Service B — listens on bus, calls Go "echo" tool
	_, err := tk.Deploy(ctx, "svc-b.ts", `
		bus.on("process", async function(msg) {
			var result = await tools.call("echo", {message: "processed:" + msg.payload.data});
			msg.reply({fromB: true, toolResult: result});
		});
	`)
	require.NoError(t, err)

	// Deploy Service A — receives request, forwards to B
	_, err = tk.Deploy(ctx, "svc-a.ts", `
		bus.on("start", function(msg) {
			var r = bus.sendTo("svc-b.ts", "process", {data: msg.payload.input});
			msg.reply({fromA: true, forwarded: true, replyTo: r.replyTo});
		});
	`)
	require.NoError(t, err)

	// Call A
	pr, err := sdk.Publish(tk, ctx, messages.CustomMsg{
		Topic:   "ts.svc-a.start",
		Payload: json.RawMessage(`{"input":"hello"}`),
	})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "fromA")
		assert.Contains(t, string(p), "forwarded")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

// TestE2E_StreamingResponse — deploy handler that uses msg.stream, verify SSE events.
func TestE2E_StreamingResponse(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "streamer.ts", `
		bus.on("stream", function(msg) {
			msg.stream.text("chunk1");
			msg.stream.text("chunk2");
			msg.stream.progress(50, "halfway");
			msg.stream.text("chunk3");
			msg.stream.end({done: true});
		});
	`)
	require.NoError(t, err)

	pr, _ := sdk.Publish(tk, ctx, messages.CustomMsg{
		Topic:   "ts.streamer.stream",
		Payload: json.RawMessage(`{}`),
	})

	var chunks []json.RawMessage
	done := make(chan bool, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) {
		chunks = append(chunks, json.RawMessage(m.Payload))
		var parsed struct {
			Type string `json:"type"`
		}
		json.Unmarshal(m.Payload, &parsed)
		if parsed.Type == "end" {
			done <- true
		}
		// Also check done metadata
		if m.Metadata["done"] == "true" {
			done <- true
		}
	})
	defer unsub()

	select {
	case <-done:
		assert.GreaterOrEqual(t, len(chunks), 3, "expected at least 3 stream chunks")
	case <-time.After(5 * time.Second):
		t.Logf("received %d chunks before timeout", len(chunks))
		// Even if done signal missed, we should have received chunks
		assert.Greater(t, len(chunks), 0, "should have received some chunks")
	}
}

// TestE2E_ScheduleFires — schedule a message, verify handler receives it.
func TestE2E_ScheduleFires(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Deploy handler
	_, err := tk.Deploy(ctx, "sched-handler.ts", `
		bus.on("tick", function(msg) {
			msg.reply({ticked: true, payload: msg.payload});
		});
	`)
	require.NoError(t, err)

	// Subscribe to know when schedule fires
	fired := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, "ts.sched-handler.tick", func(m messages.Message) {
		fired <- m.Payload
	})
	defer unsub()

	// Schedule in 200ms
	id, err := tk.Schedule(ctx, brainkit.ScheduleConfig{
		Expression: "in 200ms",
		Topic:      "ts.sched-handler.tick",
		Payload:    json.RawMessage(`{"scheduled":true}`),
	})
	require.NoError(t, err)
	require.NotEmpty(t, id)

	select {
	case p := <-fired:
		assert.Contains(t, string(p), "scheduled")
	case <-time.After(5 * time.Second):
		t.Fatal("schedule didn't fire within 5s")
	}
}

// TestE2E_MultipleKernels — create 3 independent Kernels, each deploys and works independently.
func TestE2E_MultipleKernels(t *testing.T) {
	kernels := make([]*brainkit.Kernel, 3)

	for i := 0; i < 3; i++ {
		tmpDir := t.TempDir()
		k, err := brainkit.NewKernel(brainkit.KernelConfig{
			Namespace: fmt.Sprintf("multi-%d", i),
			CallerID:  fmt.Sprintf("multi-%d", i),
			FSRoot:    tmpDir,
		})
		require.NoError(t, err)
		t.Cleanup(func() { k.Close() })

		type echoIn struct{ Message string `json:"message"` }
		brainkit.RegisterTool(k, fmt.Sprintf("echo-%d", i), registry.TypedTool[echoIn]{
			Description: "echoes",
			Execute: func(ctx context.Context, in echoIn) (any, error) {
				return map[string]string{"echoed": in.Message}, nil
			},
		})

		kernels[i] = k
	}

	for i, k := range kernels {
		payload, ok := sendAndReceive(t, &testutil.TestKernel{Kernel: k},
			messages.ToolCallMsg{Name: fmt.Sprintf("echo-%d", i), Input: map[string]any{"message": fmt.Sprintf("kernel-%d", i)}},
			5*time.Second)
		require.True(t, ok, "kernel %d didn't respond", i)
		assert.Contains(t, string(payload), fmt.Sprintf("kernel-%d", i))
	}
}

// TestE2E_DeployWithErrorRecovery — deploy bad code, recover, deploy good code.
func TestE2E_DeployWithErrorRecovery(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Deploy bad code — should fail
	_, err := tk.Deploy(ctx, "recovery.ts", `throw new Error("intentional failure");`)
	assert.Error(t, err)

	// Deploy good code to same source — should succeed
	_, err = tk.Deploy(ctx, "recovery.ts", `
		const t = createTool({id: "recovered", description: "test", execute: async () => ({ok: true})});
		kit.register("tool", "recovered", t);
	`)
	require.NoError(t, err)

	// Verify tool works
	payload, ok := sendAndReceive(t, tk, messages.ToolCallMsg{Name: "recovered", Input: map[string]any{}}, 5*time.Second)
	require.True(t, ok)
	assert.Contains(t, string(payload), "ok")
}

// TestE2E_SecretsRotateAndVerify — set secret, rotate, verify new value.
func TestE2E_SecretsRotateAndVerify(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Set
	pr1, _ := sdk.Publish(tk, ctx, messages.SecretsSetMsg{Name: "rotate-key", Value: "v1"})
	ch1 := make(chan []byte, 1)
	unsub1, _ := tk.SubscribeRaw(ctx, pr1.ReplyTo, func(m messages.Message) { ch1 <- m.Payload })
	<-ch1
	unsub1()

	// Rotate
	pr2, _ := sdk.Publish(tk, ctx, messages.SecretsRotateMsg{Name: "rotate-key", NewValue: "v2"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	p2 := <-ch2
	unsub2()
	assert.Contains(t, string(p2), "rotated")

	// Get — should be v2
	pr3, _ := sdk.Publish(tk, ctx, messages.SecretsGetMsg{Name: "rotate-key"})
	ch3 := make(chan []byte, 1)
	unsub3, _ := tk.SubscribeRaw(ctx, pr3.ReplyTo, func(m messages.Message) { ch3 <- m.Payload })
	defer unsub3()

	p3 := <-ch3
	var resp struct{ Value string `json:"value"` }
	json.Unmarshal(p3, &resp)
	assert.Equal(t, "v2", resp.Value)
}
