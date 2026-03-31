package infra_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/kit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_TwoServiceInteraction(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Service B: processes and responds
	_, err := k.Deploy(ctx, "service-b.ts", `
		bus.on("process", (msg) => {
			msg.reply({ processed: msg.payload.data + "-done" });
		});
	`)
	require.NoError(t, err)

	// Service A: receives request, calls B, forwards B's response back
	_, err = k.Deploy(ctx, "service-a.ts", `
		bus.on("ask", async (msg) => {
			var resp = await bus.sendTo("service-b.ts", "process", { data: msg.payload.data });
			msg.reply({ forwarded: resp.processed || resp });
		});
	`)
	require.NoError(t, err)
	time.Sleep(300 * time.Millisecond)

	// Send to A, expect A→B→A→caller chain
	sendPR, err := sdk.SendToService(k, ctx, "service-a.ts", "ask", map[string]string{"data": "hello"})
	require.NoError(t, err)

	replyCh := make(chan json.RawMessage, 1)
	unsub, _ := k.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) {
		replyCh <- json.RawMessage(msg.Payload)
	})
	defer unsub()

	select {
	case raw := <-replyCh:
		// Service A forwarded B's response — verify we got something back
		assert.NotEmpty(t, raw, "should receive response from two-service chain")
		t.Logf("two-service response: %s", string(raw))
	case <-time.After(10 * time.Second):
		t.Fatal("timeout — two-service interaction failed")
	}
}

func TestIntegration_ConcurrentDeploySameSource(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Deploy once normally
	_, err := k.Deploy(ctx, "shared.ts", `bus.on("x", (msg) => msg.reply({}));`)
	require.NoError(t, err)

	// Second deploy to same source should fail with AlreadyExistsError
	_, err = k.Deploy(ctx, "shared.ts", `bus.on("y", (msg) => msg.reply({}));`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestIntegration_ScheduleDuringDrain(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Create a fast schedule
	var fired atomic.Int32
	unsub, _ := k.SubscribeRaw(ctx, "drain.test.tick", func(_ messages.Message) {
		fired.Add(1)
	})
	defer unsub()

	_, err := k.Schedule(ctx, kit.ScheduleConfig{
		Expression: "every 100ms", Topic: "drain.test.tick",
		Payload: json.RawMessage(`{}`), Source: "test",
	})
	require.NoError(t, err)
	time.Sleep(350 * time.Millisecond) // let it fire a few times

	// Start drain — schedule should stop firing
	k.SetDraining(true)
	beforeDrain := fired.Load()
	time.Sleep(500 * time.Millisecond)
	afterDrain := fired.Load()

	assert.Equal(t, beforeDrain, afterDrain, "schedule should not fire during drain")
}

func TestIntegration_SecretsRotation(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Set a secret via bus
	pr, _ := sdk.Publish(k, ctx, messages.SecretsSetMsg{Name: "TEST_KEY", Value: "old-value"})
	setCh := make(chan struct{}, 1)
	setUnsub, _ := sdk.SubscribeTo[messages.SecretsSetResp](k, ctx, pr.ReplyTo, func(_ messages.SecretsSetResp, _ messages.Message) {
		setCh <- struct{}{}
	})
	select {
	case <-setCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout setting secret")
	}
	setUnsub()

	// Rotate it
	rpr, _ := sdk.Publish(k, ctx, messages.SecretsRotateMsg{Name: "TEST_KEY", NewValue: "new-value"})
	rotateCh := make(chan messages.SecretsRotateResp, 1)
	rotateUnsub, _ := sdk.SubscribeTo[messages.SecretsRotateResp](k, ctx, rpr.ReplyTo, func(resp messages.SecretsRotateResp, _ messages.Message) {
		rotateCh <- resp
	})
	select {
	case resp := <-rotateCh:
		assert.True(t, resp.Rotated)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout rotating secret")
	}
	rotateUnsub()

	// Verify the new value is returned
	gpr, _ := sdk.Publish(k, ctx, messages.SecretsGetMsg{Name: "TEST_KEY"})
	getCh := make(chan messages.SecretsGetResp, 1)
	getUnsub, _ := sdk.SubscribeTo[messages.SecretsGetResp](k, ctx, gpr.ReplyTo, func(resp messages.SecretsGetResp, _ messages.Message) {
		getCh <- resp
	})
	defer getUnsub()
	select {
	case resp := <-getCh:
		assert.Equal(t, "new-value", resp.Value)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout getting rotated secret")
	}
}

func TestIntegration_DeployOrderPreservedAfterRestart(t *testing.T) {
	_, k2 := testutil.RestartKernel(t, kit.KernelConfig{}, func(k1 *kit.Kernel) {
		ctx := context.Background()
		for i, name := range []string{"alpha.ts", "beta.ts", "gamma.ts"} {
			_, err := k1.Deploy(ctx, name, fmt.Sprintf(`bus.on("ping", (msg) => msg.reply({ order: %d }));`, i))
			require.NoError(t, err)
		}
	})

	// Verify all 3 services are running after restart
	deployments := k2.ListDeployments()
	require.Len(t, deployments, 3, "all 3 deployments should survive restart")

	// Verify one of them responds
	ctx := context.Background()
	time.Sleep(200 * time.Millisecond)
	sendPR, _ := sdk.SendToService(k2, ctx, "alpha.ts", "ping", map[string]bool{"x": true})
	replyCh := make(chan map[string]any, 1)
	unsub, _ := k2.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) {
		var r map[string]any
		json.Unmarshal(msg.Payload, &r)
		replyCh <- r
	})
	defer unsub()

	select {
	case r := <-replyCh:
		assert.Equal(t, float64(0), r["order"])
	case <-time.After(5 * time.Second):
		t.Fatal("timeout — restarted service did not respond")
	}
}
