package bus

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
)

// deployInlinePkg deploys a single-file inline .ts source via PackageDeployMsg
// and blocks until the deploy reply arrives. Shared across failure tests.
func deployInlinePkg(t *testing.T, kit *brainkit.Kit, ctx context.Context, source, code string) {
	t.Helper()
	name := strings.TrimSuffix(source, ".ts")
	manifest, _ := json.Marshal(map[string]string{"name": name, "entry": source})
	pr, _ := sdk.Publish(kit, ctx, sdk.PackageDeployMsg{
		Manifest: manifest,
		Files:    map[string]string{source: code},
	})
	deployCh := make(chan struct{}, 1)
	unsub, _ := sdk.SubscribeTo[sdk.PackageDeployResp](kit, ctx, pr.ReplyTo, func(_ sdk.PackageDeployResp, _ sdk.Message) { deployCh <- struct{}{} })
	select {
	case <-deployCh:
	case <-ctx.Done():
		t.Fatal("deploy timeout")
	}
	unsub()
	time.Sleep(100 * time.Millisecond)
}

func testSyncThrowErrorResponse(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deployInlinePkg(t, env.Kit, ctx, "thrower.ts", `bus.on("fail", (msg) => { throw new Error("sync boom"); });`)

	sendPR, _ := sdk.SendToService(env.Kit, ctx, "thrower.ts", "fail", map[string]bool{"x": true})
	errCh := make(chan string, 1)
	replyUnsub, _ := env.Kit.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg sdk.Message) {
		if m := suite.ResponseErrorMessage(msg.Payload); m != "" {
			errCh <- m
		}
	})
	defer replyUnsub()

	select {
	case errMsg := <-errCh:
		assert.Contains(t, errMsg, "sync boom")
	case <-ctx.Done():
		t.Fatal("timeout — caller should get error response, not silent timeout")
	}
}

func testAsyncRejectionErrorResponse(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deployInlinePkg(t, env.Kit, ctx, "async-fail.ts", `bus.on("fail", async (msg) => { throw new Error("async boom"); });`)

	sendPR, _ := sdk.SendToService(env.Kit, ctx, "async-fail.ts", "fail", map[string]bool{"x": true})
	errCh := make(chan string, 1)
	replyUnsub, _ := env.Kit.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg sdk.Message) {
		if m := suite.ResponseErrorMessage(msg.Payload); m != "" {
			errCh <- m
		}
	})
	defer replyUnsub()

	select {
	case errMsg := <-errCh:
		assert.Contains(t, errMsg, "async boom")
	case <-ctx.Done():
		t.Fatal("timeout — caller should get error response for async rejection")
	}
}

// testHandlerFailedEventEmitted needs its own kernel to avoid picking up
// stale bus.handler.failed events from previous failure tests on the shared kernel.
func testHandlerFailedEventEmitted(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Subscribe to bus.handler.failed BEFORE deploying+sending
	eventCh := make(chan sdk.HandlerFailedEvent, 1)
	eventUnsub, _ := sdk.SubscribeTo[sdk.HandlerFailedEvent](freshEnv.Kit, ctx, "bus.handler.failed", func(evt sdk.HandlerFailedEvent, _ sdk.Message) {
		eventCh <- evt
	})
	defer eventUnsub()

	deployInlinePkg(t, freshEnv.Kit, ctx, "event-emitter-fail.ts", `bus.on("fail", (msg) => { throw new Error("event test"); });`)

	sdk.SendToService(freshEnv.Kit, ctx, "event-emitter-fail.ts", "fail", map[string]bool{"x": true})

	select {
	case evt := <-eventCh:
		assert.Contains(t, evt.Error, "event test")
		assert.False(t, evt.WillRetry)
	case <-ctx.Done():
		t.Fatal("timeout — bus.handler.failed event should be emitted")
	}
}

// testRetryPolicyRetries creates its own kernel with RetryPolicies configured.
// Cannot use the shared env because retry policies are kernel-level config.
func testRetryPolicyRetries(t *testing.T, _ *suite.TestEnv) {
	retryEnv := suite.Full(t, suite.WithRetryPolicies(map[string]brainkit.RetryPolicy{
		"ts.retry-test.*": {
			MaxRetries:    2,
			InitialDelay:  100 * time.Millisecond,
			BackoffFactor: 2.0,
		},
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	deployInlinePkg(t, retryEnv.Kit, ctx, "retry-test.ts", `
			var _attempts = 0;
			bus.on("try", (msg) => {
				_attempts++;
				if (_attempts <= 2) {
					throw new Error("attempt " + _attempts + " failed");
				}
				msg.reply({ attempts: _attempts, ok: true });
			});
		`)

	sendPR, _ := sdk.SendToService(retryEnv.Kit, ctx, "retry-test.ts", "try", map[string]bool{"x": true})
	replyCh := make(chan map[string]any, 1)
	replyUnsub, _ := retryEnv.Kit.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg sdk.Message) {
		var resp map[string]any
		json.Unmarshal(msg.Payload, &resp)
		replyCh <- resp
	})
	defer replyUnsub()

	select {
	case resp := <-replyCh:
		assert.Equal(t, true, resp["ok"])
	case <-ctx.Done():
		t.Fatal("timeout — retries should eventually succeed")
	}
}

// testRetryExhaustedDeadLetter creates its own kernel with dead letter config.
func testRetryExhaustedDeadLetter(t *testing.T, _ *suite.TestEnv) {
	dlEnv := suite.Full(t, suite.WithRetryPolicies(map[string]brainkit.RetryPolicy{
		"ts.dl-test.*": {
			MaxRetries:      1,
			InitialDelay:    50 * time.Millisecond,
			DeadLetterTopic: "dead-letter",
		},
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deployInlinePkg(t, dlEnv.Kit, ctx, "dl-test.ts", `bus.on("fail", (msg) => { throw new Error("always fails"); });`)

	dlCh := make(chan json.RawMessage, 1)
	dlUnsub, _ := dlEnv.Kit.SubscribeRaw(ctx, "dead-letter", func(msg sdk.Message) {
		dlCh <- json.RawMessage(msg.Payload)
	})
	defer dlUnsub()

	errCh := make(chan string, 1)
	sendPR, _ := sdk.SendToService(dlEnv.Kit, ctx, "dl-test.ts", "fail", map[string]bool{"x": true})
	replyUnsub, _ := dlEnv.Kit.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg sdk.Message) {
		if m := suite.ResponseErrorMessage(msg.Payload); m != "" {
			errCh <- m
		}
	})
	defer replyUnsub()

	select {
	case dl := <-dlCh:
		var parsed map[string]any
		json.Unmarshal(dl, &parsed)
		assert.Contains(t, parsed["error"], "always fails")
	case <-ctx.Done():
		t.Fatal("timeout — dead letter should be published")
	}

	select {
	case errMsg := <-errCh:
		assert.Contains(t, errMsg, "1 retries")
	case <-time.After(2 * time.Second):
		t.Fatal("error response should be sent after exhaustion")
	}
}

// testExhaustedEventEmitted creates its own kernel with retry config.
func testExhaustedEventEmitted(t *testing.T, _ *suite.TestEnv) {
	exEnv := suite.Full(t, suite.WithRetryPolicies(map[string]brainkit.RetryPolicy{
		"ts.exhaust-evt.*": {
			MaxRetries:   1,
			InitialDelay: 50 * time.Millisecond,
		},
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deployInlinePkg(t, exEnv.Kit, ctx, "exhaust-evt.ts", `bus.on("fail", (msg) => { throw new Error("exhaust event"); });`)

	exhaustedCh := make(chan sdk.HandlerExhaustedEvent, 1)
	exUnsub, _ := sdk.SubscribeTo[sdk.HandlerExhaustedEvent](exEnv.Kit, ctx, "bus.handler.exhausted", func(evt sdk.HandlerExhaustedEvent, _ sdk.Message) {
		exhaustedCh <- evt
	})
	defer exUnsub()

	sdk.SendToService(exEnv.Kit, ctx, "exhaust-evt.ts", "fail", map[string]bool{"x": true})

	select {
	case evt := <-exhaustedCh:
		assert.Contains(t, evt.Error, "exhaust event")
		assert.Equal(t, 1, evt.RetryCount)
	case <-ctx.Done():
		t.Fatal("timeout — bus.handler.exhausted event should be emitted")
	}
}

// testRetryPreservesReplyTo creates its own kernel with retry config.
func testRetryPreservesReplyTo(t *testing.T, _ *suite.TestEnv) {
	rpEnv := suite.Full(t, suite.WithRetryPolicies(map[string]brainkit.RetryPolicy{
		"ts.replyto-test.*": {
			MaxRetries:   1,
			InitialDelay: 100 * time.Millisecond,
		},
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deployInlinePkg(t, rpEnv.Kit, ctx, "replyto-test.ts", `
			var _count = 0;
			bus.on("try", (msg) => {
				_count++;
				if (_count === 1) throw new Error("first fail");
				msg.reply({ ok: true, attempt: _count });
			});
		`)

	sendPR, _ := sdk.SendToService(rpEnv.Kit, ctx, "replyto-test.ts", "try", map[string]bool{"x": true})
	replyCh := make(chan map[string]any, 1)
	replyUnsub, _ := rpEnv.Kit.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg sdk.Message) {
		var resp map[string]any
		json.Unmarshal(suite.ResponseData(msg.Payload), &resp)
		replyCh <- resp
	})
	defer replyUnsub()

	select {
	case resp := <-replyCh:
		assert.Equal(t, true, resp["ok"])
		assert.Equal(t, float64(2), resp["attempt"])
	case <-ctx.Done():
		t.Fatal("timeout — original caller should receive the success reply after retry")
	}
}
