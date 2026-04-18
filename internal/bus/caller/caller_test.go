package caller_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/brainlet/brainkit/internal/bus/caller"
	"github.com/brainlet/brainkit/sdk"
	"github.com/stretchr/testify/require"
)

// fakeRuntime is a minimal sdk.Runtime that records subscriptions. It
// exists so NewCallerWithInbox can be exercised without a full Kit.
type fakeRuntime struct {
	mu         sync.Mutex
	subscribed []string
}

func (f *fakeRuntime) PublishRaw(_ context.Context, _ string, _ json.RawMessage) (string, error) {
	return "correlation-id-fake", nil
}

func (f *fakeRuntime) SubscribeRaw(_ context.Context, topic string, _ func(sdk.Message)) (func(), error) {
	f.mu.Lock()
	f.subscribed = append(f.subscribed, topic)
	f.mu.Unlock()
	return func() {}, nil
}

func (f *fakeRuntime) Close() error { return nil }

// TestNewCallerWithInboxUsesExplicitTopic verifies the plugin-side
// constructor subscribes to the inbox topic verbatim (no
// "_brainkit.inbox." prefixing). This is the entry-point the plugin
// SDK relies on so its inbox is `_brainkit.plugin-inbox.<owner>.<name>`
// rather than the Kit's scheme.
func TestNewCallerWithInboxUsesExplicitTopic(t *testing.T) {
	rt := &fakeRuntime{}
	inbox := "_brainkit.plugin-inbox.acme.demo"

	c, err := caller.NewCallerWithInbox(rt, inbox, nil)
	require.NoError(t, err)
	defer c.Close()

	require.Equal(t, inbox, c.Inbox())
	require.Contains(t, rt.subscribed, inbox,
		"caller must subscribe to the inbox topic as provided")
	require.Contains(t, rt.subscribed, "bus.handler.exhausted",
		"caller must still subscribe to the fail-fast channel")
}

// TestNewCallerWithInboxValidation confirms required arguments are
// checked.
func TestNewCallerWithInboxValidation(t *testing.T) {
	_, err := caller.NewCallerWithInbox(nil, "inbox", nil)
	require.Error(t, err)

	_, err = caller.NewCallerWithInbox(&fakeRuntime{}, "", nil)
	require.Error(t, err)
}
