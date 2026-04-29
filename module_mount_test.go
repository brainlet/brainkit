package brainkit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	"github.com/stretchr/testify/require"
)

type hotEchoMsg struct {
	Text string `json:"text"`
}

func (hotEchoMsg) BusTopic() string { return "test.hot.echo" }

type hotEchoResp struct {
	Text string `json:"text"`
}

type hotEchoModule struct {
	id     string
	suffix string
}

func (m hotEchoModule) ID() string { return m.id }

func (m hotEchoModule) Mount(_ context.Context, host bkmodule.Host) error {
	_, err := host.Commands().Handle(bkmodule.Command(func(_ context.Context, req hotEchoMsg) (*hotEchoResp, error) {
		return &hotEchoResp{Text: req.Text + m.suffix}, nil
	}))
	return err
}

func TestKitMountCommandAfterStartUnmountAndRemount(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	require.NoError(t, k.Mount(ctx, hotEchoModule{id: "hot-echo", suffix: "-one"}))
	require.True(t, k.HasCommand("test.hot.echo"))

	resp, err := Call[hotEchoMsg, hotEchoResp](k, ctx, hotEchoMsg{Text: "hello"}, WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	require.Equal(t, "hello-one", resp.Text)

	require.NoError(t, k.Unmount(ctx, "hot-echo"))
	require.False(t, k.HasCommand("test.hot.echo"))

	require.NoError(t, k.Mount(ctx, hotEchoModule{id: "hot-echo", suffix: "-two"}))
	resp, err = Call[hotEchoMsg, hotEchoResp](k, ctx, hotEchoMsg{Text: "hello"}, WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	require.Equal(t, "hello-two", resp.Text)
}

type cleanupModule struct {
	id     string
	closed *bool
}

func (m cleanupModule) ID() string { return m.id }

func (m cleanupModule) Mount(_ context.Context, host bkmodule.Host) error {
	host.Scope().Defer(func(context.Context) error {
		*m.closed = true
		return nil
	})
	return nil
}

func TestKitCloseClosesMountedScopes(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)

	closed := false
	require.NoError(t, k.Mount(context.Background(), cleanupModule{id: "cleanup", closed: &closed}))

	require.NoError(t, k.Close())
	require.True(t, closed)
}

type toolModule struct {
	id string
}

func (m toolModule) ID() string { return m.id }

func (m toolModule) Mount(ctx context.Context, host bkmodule.Host) error {
	_, err := host.Tools().Register(ctx, bkmodule.ToolSpec{
		Name: "test.tool",
		Executor: bkmodule.ToolExecutorFunc(func(context.Context, string, json.RawMessage) (json.RawMessage, error) {
			return json.RawMessage(`{"ok":true}`), nil
		}),
	})
	return err
}

func TestKitUnmountUnregistersScopedTools(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	require.NoError(t, k.Mount(ctx, toolModule{id: "tool"}))
	_, err = k.kernel.Tools.Resolve("test.tool")
	require.NoError(t, err)

	require.NoError(t, k.Unmount(ctx, "tool"))
	_, err = k.kernel.Tools.Resolve("test.tool")
	require.ErrorAs(t, err, new(*sdkerrors.NotFoundError))
}

type eventModule struct {
	id   string
	seen chan string
}

func (m eventModule) ID() string { return m.id }

func (m eventModule) Mount(ctx context.Context, host bkmodule.Host) error {
	_, err := host.Messages().SubscribeRaw(ctx, "test.event", func(msg sdk.Message) {
		select {
		case m.seen <- string(msg.Payload):
		default:
		}
	})
	return err
}

func TestKitUnmountCancelsScopedSubscriptions(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	seen := make(chan string, 1)
	require.NoError(t, k.Mount(ctx, eventModule{id: "events", seen: seen}))

	_, err = k.PublishRaw(ctx, "test.event", json.RawMessage(`"one"`))
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		select {
		case got := <-seen:
			return got == `"one"`
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)

	require.NoError(t, k.Unmount(ctx, "events"))
	_, err = k.PublishRaw(ctx, "test.event", json.RawMessage(`"two"`))
	require.NoError(t, err)

	select {
	case got := <-seen:
		t.Fatalf("received event after unmount: %s", got)
	case <-time.After(100 * time.Millisecond):
	}
}

type capabilityModule struct {
	id string
}

func (m capabilityModule) ID() string { return m.id }

func (m capabilityModule) Mount(ctx context.Context, host bkmodule.Host) error {
	_, err := host.Capabilities().Provide(ctx, "test.capability", "value")
	return err
}

func TestKitUnmountRemovesScopedCapabilities(t *testing.T) {
	k, err := New(Config{Transport: Memory()})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	require.NoError(t, k.Mount(ctx, capabilityModule{id: "capability"}))
	value, ok := k.caps.Get("test.capability")
	require.True(t, ok)
	require.Equal(t, "value", value)

	require.NoError(t, k.Unmount(ctx, "capability"))
	_, ok = k.caps.Get("test.capability")
	require.False(t, ok)
}
