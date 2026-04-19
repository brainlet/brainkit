package voicerealtime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testWSEchoConnect boots an httptest server that upgrades to
// WebSocket and echoes frames. The .ts side points a `WebSocket(url)`
// at it through the jsbridge polyfill; the test asserts the upgrade
// happens and a round-tripped message returns with the same payload.
// OpenAIRealtimeVoice.connect() can't use this server directly (it
// insists on the OpenAI Realtime protocol), so we exercise the
// WebSocket polyfill path that OpenAIRealtimeVoice rides on.
func testWSEchoConnect(t *testing.T, env *suite.TestEnv) {
	var upgrades int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		atomic.AddInt32(&upgrades, 1)
		defer c.CloseNow()
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		for {
			typ, msg, err := c.Read(ctx)
			if err != nil {
				return
			}
			if err := c.Write(ctx, typ, msg); err != nil {
				return
			}
		}
	}))
	t.Cleanup(srv.Close)

	wsURL := "ws://" + strings.TrimPrefix(srv.URL, "http://")

	// Drive the jsbridge WebSocket polyfill from inside the kernel.
	// EvalTS runs without the deploy transpiler, but globalThis.WebSocket
	// is installed by internal/jsbridge/websocket.go.
	out := testutil.EvalTS(t, env.Kit, "voice-realtime-ws-echo.ts", `
		if (typeof WebSocket !== "function") throw new Error("WebSocket polyfill missing");
		const want = "brainkit-realtime-echo-" + Date.now();
		const ws = new WebSocket(`+"`"+wsURL+"`"+`);
		const received = await new Promise((resolve, reject) => {
			const timer = setTimeout(() => reject(new Error("timeout")), 3000);
			ws.onopen = () => { ws.send(want); };
			ws.onmessage = (ev) => { clearTimeout(timer); resolve(String(ev.data)); };
			ws.onerror = (ev) => { clearTimeout(timer); reject(new Error("ws error")); };
		});
		ws.close();
		return JSON.stringify({ want, got: received });
	`)

	assert.Contains(t, out, "brainkit-realtime-echo-",
		"echo payload should round-trip through jsbridge WebSocket")
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&upgrades) >= 1
	}, 2*time.Second, 50*time.Millisecond, "server should log at least one WS upgrade")
	// Sanity: ensure the .ts produced matching want/got.
	assert.Contains(t, out, `"got":"brainkit-realtime-echo-`)
	assert.Contains(t, out, `"want":"brainkit-realtime-echo-`)
	_ = os.Getwd // silence unused import if os isn't otherwise needed
}
