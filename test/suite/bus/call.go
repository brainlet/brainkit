package bus

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
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

// testCallHappyPath — deploy a .ts handler, Call() it, assert decoded reply.
func testCallHappyPath(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	testutil.Deploy(t, env.Kit, "call-happy.ts", `
		bus.on("ping", (msg) => {
			msg.reply({ pong: "hi-" + msg.payload.x });
		});
	`)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := brainkit.Call[sdk.CustomMsg, map[string]string](env.Kit, ctx, sdk.CustomMsg{
		Topic:   "ts.call-happy.ping",
		Payload: []byte(`{"x":1}`),
	})
	require.NoError(t, err)
	assert.Equal(t, "hi-1", resp["pong"])
}

// testCallRequiresDeadline — no ctx deadline + no WithCallTimeout → error.
func testCallRequiresDeadline(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	_, err := brainkit.Call[sdk.CustomMsg, map[string]any](env.Kit, context.Background(), sdk.CustomMsg{
		Topic:   "ts.call-none.ping",
		Payload: []byte(`{}`),
	})
	require.Error(t, err)
	var ndl *caller.NoDeadlineError
	assert.True(t, errors.As(err, &ndl), "want NoDeadlineError, got %T", err)
}

// testCallWithCallTimeout — no ctx deadline but WithCallTimeout supplies one.
func testCallWithCallTimeout(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	testutil.Deploy(t, env.Kit, "call-timeout.ts", `
		bus.on("ping", (msg) => msg.reply({ ok: true }));
	`)
	resp, err := brainkit.Call[sdk.CustomMsg, map[string]bool](env.Kit, context.Background(), sdk.CustomMsg{
		Topic:   "ts.call-timeout.ping",
		Payload: []byte(`{}`),
	}, brainkit.WithCallTimeout(3*time.Second))
	require.NoError(t, err)
	assert.True(t, resp["ok"])
}

// testCallTimeoutError — handler never replies, deadline elapses, typed error.
func testCallTimeoutError(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	testutil.Deploy(t, env.Kit, "call-silent.ts", `
		bus.on("ping", (msg) => { /* never reply */ });
	`)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := brainkit.Call[sdk.CustomMsg, map[string]any](env.Kit, ctx, sdk.CustomMsg{
		Topic:   "ts.call-silent.ping",
		Payload: []byte(`{}`),
	})
	require.Error(t, err)
	var te *caller.CallTimeoutError
	assert.True(t, errors.As(err, &te), "want CallTimeoutError, got %T", err)
}

// testCallCancelledError — explicit cancel returns CallCancelledError.
func testCallCancelledError(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	testutil.Deploy(t, env.Kit, "call-cancel.ts", `
		bus.on("ping", (msg) => { /* never reply */ });
	`)
	ctx, cancel := context.WithCancel(context.Background())
	// Still need a deadline to pass the Call gate.
	ctx, dlCancel := context.WithTimeout(ctx, 5*time.Second)
	defer dlCancel()
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	_, err := brainkit.Call[sdk.CustomMsg, map[string]any](env.Kit, ctx, sdk.CustomMsg{
		Topic:   "ts.call-cancel.ping",
		Payload: []byte(`{}`),
	})
	require.Error(t, err)
	var ce *caller.CallCancelledError
	assert.True(t, errors.As(err, &ce), "want CallCancelledError, got %T", err)
}

// testCallConcurrentDemux — 50 concurrent Calls each get their own reply.
func testCallConcurrentDemux(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	testutil.Deploy(t, env.Kit, "call-demux.ts", `
		bus.on("echo", (msg) => msg.reply({ n: msg.payload.n }));
	`)
	const N = 50
	var seen atomic.Int64
	errs := make(chan error, N)
	for i := 0; i < N; i++ {
		n := i
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			payload := []byte(`{"n":` + itoa(n) + `}`)
			resp, err := brainkit.Call[sdk.CustomMsg, map[string]int](env.Kit, ctx, sdk.CustomMsg{
				Topic:   "ts.call-demux.echo",
				Payload: payload,
			})
			if err != nil {
				errs <- err
				return
			}
			if resp["n"] != n {
				errs <- errTooFar(n, resp["n"])
				return
			}
			seen.Add(1)
			errs <- nil
		}()
	}
	for i := 0; i < N; i++ {
		require.NoError(t, <-errs)
	}
	assert.Equal(t, int64(N), seen.Load())
}

// testCallRawPayload — Resp=json.RawMessage short-circuits decode.
func testCallRawPayload(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	testutil.Deploy(t, env.Kit, "call-raw.ts", `
		bus.on("ping", (msg) => msg.reply({ mirror: msg.payload }));
	`)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	raw, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](env.Kit, ctx, sdk.CustomMsg{
		Topic:   "ts.call-raw.ping",
		Payload: []byte(`{"val":42}`),
	})
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"val":42`)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var buf [20]byte
	n := len(buf)
	for i > 0 {
		n--
		buf[n] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		n--
		buf[n] = '-'
	}
	return string(buf[n:])
}

func errTooFar(want, got int) error {
	return &numericMismatch{want: want, got: got}
}

type numericMismatch struct{ want, got int }

func (e *numericMismatch) Error() string {
	return "mismatch: want=" + itoa(e.want) + " got=" + itoa(e.got)
}
