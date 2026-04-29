package bus

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/packages/packagemsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/systemmsg"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
)

// deployInlinePkg deploys a single-file inline .ts source via PackageDeployMsg
// and blocks until the deploy reply arrives. Shared across failure tests.
func deployInlinePkg(t *testing.T, kit *brainkit.Kit, _ context.Context, source, code string) {
	t.Helper()
	name := strings.TrimSuffix(source, ".ts")
	manifest, _ := json.Marshal(map[string]string{"name": name, "entry": source})
	_, ok := publishAndWaitPayload(t, kit, packagemsg.PackageDeployMsg{
		Manifest: manifest,
		Files:    map[string]string{source: code},
	}, 10*time.Second)
	if !ok {
		t.Fatal("deploy timeout")
	}
	time.Sleep(100 * time.Millisecond)
}

func testSyncThrowErrorResponse(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deployInlinePkg(t, env.Kit, ctx, "thrower.ts", `bus.on("fail", (msg) => { throw new Error("sync boom"); });`)

	_, err := env.Kit.Caller().Call(ctx, sdk.ResolveServiceTopic("thrower.ts", "fail"), json.RawMessage(`{"x":true}`), sdk.CallerConfig{})
	if assert.Error(t, err, "caller should get error response, not silent timeout") {
		assert.Contains(t, err.Error(), "sync boom")
	}
}

func testAsyncRejectionErrorResponse(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deployInlinePkg(t, env.Kit, ctx, "async-fail.ts", `bus.on("fail", async (msg) => { throw new Error("async boom"); });`)

	_, err := env.Kit.Caller().Call(ctx, sdk.ResolveServiceTopic("async-fail.ts", "fail"), json.RawMessage(`{"x":true}`), sdk.CallerConfig{})
	if assert.Error(t, err, "caller should get error response for async rejection") {
		assert.Contains(t, err.Error(), "async boom")
	}
}

// testHandlerFailedEventEmitted needs its own kernel to avoid picking up
// stale bus.handler.failed events from previous failure tests on the shared kernel.
func testHandlerFailedEventEmitted(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Subscribe to bus.handler.failed BEFORE deploying+sending
	eventCh := make(chan systemmsg.HandlerFailedEvent, 1)
	eventUnsub, _ := sdk.SubscribeTo[systemmsg.HandlerFailedEvent](freshEnv.Kit, ctx, systemmsg.TopicHandlerFailed, func(evt systemmsg.HandlerFailedEvent, _ sdk.Message) {
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

	payload, err := retryEnv.Kit.Caller().Call(ctx, sdk.ResolveServiceTopic("retry-test.ts", "try"), json.RawMessage(`{"x":true}`), sdk.CallerConfig{})
	if assert.NoError(t, err, "retries should eventually succeed") {
		var resp map[string]any
		requireNoJSONError(t, json.Unmarshal(payload, &resp))
		assert.Equal(t, true, resp["ok"])
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
	go func() {
		_, err := dlEnv.Kit.Caller().Call(ctx, sdk.ResolveServiceTopic("dl-test.ts", "fail"), json.RawMessage(`{"x":true}`), sdk.CallerConfig{})
		if err != nil {
			errCh <- err.Error()
		}
	}()

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

	exhaustedCh := make(chan systemmsg.HandlerExhaustedEvent, 1)
	exUnsub, _ := sdk.SubscribeTo[systemmsg.HandlerExhaustedEvent](exEnv.Kit, ctx, systemmsg.TopicHandlerExhausted, func(evt systemmsg.HandlerExhaustedEvent, _ sdk.Message) {
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

	payload, err := rpEnv.Kit.Caller().Call(ctx, sdk.ResolveServiceTopic("replyto-test.ts", "try"), json.RawMessage(`{"x":true}`), sdk.CallerConfig{})
	if assert.NoError(t, err, "original caller should receive the success reply after retry") {
		var resp map[string]any
		requireNoJSONError(t, json.Unmarshal(payload, &resp))
		assert.Equal(t, true, resp["ok"])
		assert.Equal(t, float64(2), resp["attempt"])
	}
}

func requireNoJSONError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
}
