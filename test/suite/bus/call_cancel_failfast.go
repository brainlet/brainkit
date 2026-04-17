package bus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/bus/caller"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCallEmitsCancelOnCtxCancel — when ctx cancels before reply, Caller
// publishes a CancelNotice on _brainkit.cancel with matching correlationId.
func testCallEmitsCancelOnCtxCancel(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	testutil.Deploy(t, env.Kit, "cancel-silent.ts", `
		bus.on("slow", (msg) => { /* never reply */ });
	`)

	noticeCh := make(chan caller.CancelNotice, 1)
	noticeCtx, noticeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer noticeCancel()
	unsub, err := env.Kit.SubscribeRaw(noticeCtx, caller.CancelTopic, func(msg sdk.Message) {
		var n caller.CancelNotice
		if err := json.Unmarshal(msg.Payload, &n); err == nil {
			select {
			case noticeCh <- n:
			default:
			}
		}
	})
	require.NoError(t, err)
	defer unsub()

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_, callErr := brainkit.Call[sdk.CustomMsg, map[string]any](env.Kit, ctx, sdk.CustomMsg{
		Topic:   "ts.cancel-silent.slow",
		Payload: []byte(`{}`),
	})
	require.Error(t, callErr)

	select {
	case n := <-noticeCh:
		assert.NotEmpty(t, n.CorrelationID, "cancel notice must carry correlationId")
		assert.Equal(t, "ts.cancel-silent.slow", n.Topic)
		assert.NotEmpty(t, n.Reason)
	case <-time.After(2 * time.Second):
		t.Fatal("expected CancelNotice on _brainkit.cancel after ctx timeout")
	}
}

// testCallNoCancelSignalSuppresses — WithCallNoCancelSignal disables the
// best-effort cancel publish on ctx timeout.
func testCallNoCancelSignalSuppresses(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	testutil.Deploy(t, env.Kit, "nocancel-silent.ts", `
		bus.on("slow", (msg) => { /* never reply */ });
	`)

	got := make(chan struct{}, 1)
	subCtx, subCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer subCancel()
	unsub, err := env.Kit.SubscribeRaw(subCtx, caller.CancelTopic, func(_ sdk.Message) {
		select {
		case got <- struct{}{}:
		default:
		}
	})
	require.NoError(t, err)
	defer unsub()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, callErr := brainkit.Call[sdk.CustomMsg, map[string]any](env.Kit, ctx, sdk.CustomMsg{
		Topic:   "ts.nocancel-silent.slow",
		Payload: []byte(`{}`),
	}, brainkit.WithCallNoCancelSignal())
	require.Error(t, callErr)

	select {
	case <-got:
		t.Fatal("WithCallNoCancelSignal must suppress _brainkit.cancel publish")
	case <-time.After(500 * time.Millisecond):
	}
}

// testExhaustedEventCarriesCorrelationId — retry-exhausted event carries
// correlationId metadata so fail-fast can match a pending call.
func testExhaustedEventCarriesCorrelationId(t *testing.T, _ *suite.TestEnv) {
	exEnv := suite.Full(t, suite.WithRetryPolicies(map[string]brainkit.RetryPolicy{
		"ts.exhaust-cid.*": {
			MaxRetries:   1,
			InitialDelay: 30 * time.Millisecond,
		},
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exhaustManifest, _ := json.Marshal(map[string]string{"name": "exhaust-cid", "entry": "exhaust-cid.ts"})
	pr, _ := sdk.Publish(exEnv.Kit, ctx, sdk.PackageDeployMsg{
		Manifest: exhaustManifest,
		Files:    map[string]string{"exhaust-cid.ts": `bus.on("fail", (msg) => { throw new Error("cid exhaust"); });`},
	})
	deployCh := make(chan struct{}, 1)
	dUnsub, _ := sdk.SubscribeTo[sdk.PackageDeployResp](exEnv.Kit, ctx, pr.ReplyTo, func(_ sdk.PackageDeployResp, _ sdk.Message) { deployCh <- struct{}{} })
	<-deployCh
	dUnsub()
	time.Sleep(100 * time.Millisecond)

	cidCh := make(chan string, 1)
	exUnsub, err := exEnv.Kit.SubscribeRaw(ctx, "bus.handler.exhausted", func(msg sdk.Message) {
		if msg.Metadata != nil {
			if cid := msg.Metadata["correlationId"]; cid != "" {
				select {
				case cidCh <- cid:
				default:
				}
			}
		}
	})
	require.NoError(t, err)
	defer exUnsub()

	sendPR, _ := sdk.SendToService(exEnv.Kit, ctx, "exhaust-cid.ts", "fail", map[string]bool{"x": true})
	require.NotEmpty(t, sendPR.CorrelationID)

	select {
	case cid := <-cidCh:
		assert.Equal(t, sendPR.CorrelationID, cid, "exhausted event must echo original correlationId")
	case <-ctx.Done():
		t.Fatal("timeout — exhausted event with correlationId metadata expected")
	}
}

// testCallerMetricsSnapshot — basic counter increment sanity: a completed
// call bumps Completed, a timed-out call bumps TimedOut.
func testCallerMetricsSnapshot(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	testutil.Deploy(t, env.Kit, "metrics-mix.ts", `
		bus.on("ok", (msg) => msg.reply({ ok: true }));
		bus.on("slow", (msg) => { /* never reply */ });
	`)

	before := env.Kit.Caller().Snapshot()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := brainkit.Call[sdk.CustomMsg, map[string]bool](env.Kit, ctx, sdk.CustomMsg{
		Topic:   "ts.metrics-mix.ok",
		Payload: []byte(`{}`),
	})
	require.NoError(t, err)

	toCtx, toCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer toCancel()
	_, err = brainkit.Call[sdk.CustomMsg, map[string]any](env.Kit, toCtx, sdk.CustomMsg{
		Topic:   "ts.metrics-mix.slow",
		Payload: []byte(`{}`),
	})
	require.Error(t, err)

	after := env.Kit.Caller().Snapshot()
	assert.Greater(t, after.Completed, before.Completed, "Completed must increment on happy call")
	assert.Greater(t, after.TimedOut, before.TimedOut, "TimedOut must increment on ctx deadline")
}
