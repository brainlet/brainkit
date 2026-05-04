package sdk_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/ctxkeys"
	"github.com/stretchr/testify/require"
)

// fakeRuntime is a minimal sdk.Runtime that records subscriptions. It
// exists so NewCallerWithInbox can be exercised without a full Kit.
type fakeRuntime struct {
	mu         sync.Mutex
	subscribed []string
	published  []publishedCall
	handlers   map[string]func(sdk.Message)
}

type publishedCall struct {
	ctx     context.Context
	topic   string
	payload json.RawMessage
}

func (f *fakeRuntime) PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (string, error) {
	f.mu.Lock()
	f.published = append(f.published, publishedCall{
		ctx:     ctx,
		topic:   topic,
		payload: append(json.RawMessage(nil), payload...),
	})
	f.mu.Unlock()
	return "correlation-id-fake", nil
}

func (f *fakeRuntime) SubscribeRaw(_ context.Context, topic string, handler func(sdk.Message)) (func(), error) {
	f.mu.Lock()
	if f.handlers == nil {
		f.handlers = map[string]func(sdk.Message){}
	}
	f.subscribed = append(f.subscribed, topic)
	f.handlers[topic] = handler
	f.mu.Unlock()
	return func() {
		f.mu.Lock()
		delete(f.handlers, topic)
		f.mu.Unlock()
	}, nil
}

func (f *fakeRuntime) Close() error { return nil }

func (f *fakeRuntime) deliver(topic string, msg sdk.Message) {
	f.mu.Lock()
	handler := f.handlers[topic]
	f.mu.Unlock()
	if handler != nil {
		handler(msg)
	}
}

// TestNewCallerWithInboxUsesExplicitTopic verifies the plugin-side
// constructor subscribes to the inbox topic verbatim (no
// "_brainkit.inbox." prefixing). This is the entry-point the plugin
// SDK relies on so its inbox is `_brainkit.plugin-inbox.<owner>.<name>`
// rather than the Kit's scheme.
func TestNewCallerWithInboxUsesExplicitTopic(t *testing.T) {
	rt := &fakeRuntime{}
	inbox := "_brainkit.plugin-inbox.acme.demo"

	c, err := sdk.NewCallerWithInbox(rt, inbox, nil)
	require.NoError(t, err)
	defer c.Close()

	require.Equal(t, inbox, c.Inbox())
	require.Contains(t, rt.subscribed, inbox,
		"caller must subscribe to the inbox topic as provided")
	require.Contains(t, rt.subscribed, "bus.handler.exhausted",
		"caller must still subscribe to the fail-fast channel")
}

// TestNewCallerWithInboxValidation confirms required arguments are checked.
func TestNewCallerWithInboxValidation(t *testing.T) {
	_, err := sdk.NewCallerWithInbox(nil, "inbox", nil)
	require.Error(t, err)

	_, err = sdk.NewCallerWithInbox(&fakeRuntime{}, "", nil)
	require.Error(t, err)
}

func TestCallerCallStampsCallerIDAndMetadata(t *testing.T) {
	rt := &fakeRuntime{}
	c, err := sdk.NewCallerWithInbox(rt, "_brainkit.inbox.test", nil)
	require.NoError(t, err)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	_, err = c.Call(ctx, "demo.topic", json.RawMessage(`{"ok":true}`), sdk.CallerConfig{
		CallerID: "caller.ts",
		Metadata: map[string]string{
			"custom":   "value",
			"callerId": "ignored-by-transport",
		},
	})
	require.Error(t, err)

	rt.mu.Lock()
	require.NotEmpty(t, rt.published)
	published := rt.published[0]
	rt.mu.Unlock()

	require.Equal(t, "demo.topic", published.topic)
	require.Equal(t, "caller.ts", published.ctx.Value(ctxkeys.CallerID))
	require.Equal(t, "value", ctxkeys.MetadataFromContext(published.ctx)["custom"])
}

func TestCallerTimeoutFinalizesStreamingDrain(t *testing.T) {
	rt := &fakeRuntime{}
	c, err := sdk.NewCallerWithInbox(rt, "_brainkit.inbox.test", nil)
	require.NoError(t, err)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err = c.Call(ctx, "demo.stream", json.RawMessage(`{}`), sdk.CallerConfig{
		StreamHandler: func(sdk.Message) error { return nil },
	})
	require.Error(t, err)

	require.Eventually(t, func() bool {
		snapshot := c.DebugSnapshot()
		return snapshot.PendingCalls == 0 && snapshot.ActiveStreamDrains == 0
	}, time.Second, 10*time.Millisecond)
}

func TestCallerCloseContextReportsBlockedStreamingDrain(t *testing.T) {
	rt := &fakeRuntime{}
	c, err := sdk.NewCallerWithInbox(rt, "_brainkit.inbox.test", nil)
	require.NoError(t, err)
	defer c.Close()

	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	callDone := make(chan error, 1)
	go func() {
		ctx := context.Background()
		_, callErr := c.Call(ctx, "demo.stream", json.RawMessage(`{}`), sdk.CallerConfig{
			StreamHandler: func(sdk.Message) error {
				once.Do(func() { close(started) })
				<-release
				return nil
			},
		})
		callDone <- callErr
	}()

	require.Eventually(t, func() bool {
		rt.mu.Lock()
		published := len(rt.published)
		var cid string
		if published > 0 {
			cid, _ = rt.published[0].ctx.Value(ctxkeys.CorrelationID).(string)
		}
		rt.mu.Unlock()
		if cid == "" {
			return false
		}
		rt.deliver(c.Inbox(), sdk.Message{
			Topic:   c.Inbox(),
			Payload: json.RawMessage(`{"chunk":1}`),
			Metadata: map[string]string{
				"correlationId": cid,
			},
		})
		return true
	}, time.Second, 10*time.Millisecond)

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("stream handler did not start")
	}
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer closeCancel()
	err = c.CloseContext(closeCtx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	snapshot := c.DebugSnapshot()
	require.True(t, snapshot.Closed)
	require.Equal(t, int64(1), snapshot.ActiveStreamDrains)

	close(release)
	require.Eventually(t, func() bool {
		return c.DebugSnapshot().ActiveStreamDrains == 0
	}, time.Second, 10*time.Millisecond)
	require.ErrorIs(t, <-callDone, sdk.ErrCallerClosed)
}
