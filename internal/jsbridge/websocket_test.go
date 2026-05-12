package jsbridge

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// wsEchoServer spins a tiny echo server: replies every text
// frame with "echo:<body>" and every binary frame with a byte
// sentinel + the original bytes. Used to exercise both code
// paths in the polyfill.
func wsEchoServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.EqualFold(r.Header.Get("Upgrade"), "") {
			return
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")
		// Keep the peer-supplied Authorization header so the
		// test can assert the polyfill forwarded it on the
		// dial handshake.
		auth := r.Header.Get("Authorization")
		if auth != "" {
			_ = conn.Write(context.Background(), websocket.MessageText, []byte("auth:"+auth))
		}
		for {
			typ, data, err := conn.Read(r.Context())
			if err != nil {
				return
			}
			if typ == websocket.MessageBinary {
				out := append([]byte{0xAA}, data...)
				_ = conn.Write(r.Context(), websocket.MessageBinary, out)
			} else {
				_ = conn.Write(r.Context(), websocket.MessageText, []byte("echo:"+string(data)))
			}
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// wsURL converts the httptest http:// URL to ws:// so the
// polyfill dials the WebSocket handshake correctly.
func wsURL(s *httptest.Server) string {
	return "ws" + strings.TrimPrefix(s.URL, "http")
}

func TestWebSocketTextRoundTrip(t *testing.T) {
	srv := wsEchoServer(t)
	url := wsURL(srv)
	b := newTestBridge(t, Encoding(), Events(), NodeStreams(), Buffer(), Timers(), WebSocketPoly())

	val, err := b.EvalAsync("ws-text.js", `(async function() {
		return new Promise(function(resolve, reject) {
			var ws = new WebSocket("`+url+`");
			var results = [];
			ws.on("open", function() { ws.send("hello"); });
			ws.on("message", function(data) {
				results.push(data instanceof Uint8Array ? new TextDecoder().decode(data) : String(data));
				if (results.length === 1) { ws.close(1000, "bye"); resolve(results[0]); }
			});
			ws.on("error", function(e) { reject(String(e)); });
			setTimeout(function() { reject("timeout"); }, 5000);
		});
	})()`)
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()
	if got := val.String(); got != "echo:hello" {
		t.Errorf("got %q, want echo:hello", got)
	}
}

func TestWebSocketBinaryRoundTrip(t *testing.T) {
	srv := wsEchoServer(t)
	url := wsURL(srv)
	b := newTestBridge(t, Encoding(), Events(), NodeStreams(), Buffer(), Timers(), WebSocketPoly())

	val, err := b.EvalAsync("ws-bin.js", `(async function() {
		return new Promise(function(resolve, reject) {
			var ws = new WebSocket("`+url+`");
			ws.on("open", function() {
				// Send non-ASCII bytes to prove binary passes through
				// the base64 hop without utf-8 corruption.
				ws.send(new Uint8Array([0xFF, 0x00, 0x7F, 0xC3, 0xA9]));
			});
			ws.on("message", function(data) {
				var u8 = data instanceof Uint8Array ? data :
				        (data && data.byteLength ? new Uint8Array(data.buffer || data) : null);
				if (!u8) return reject("non-binary reply: " + typeof data);
				ws.close();
				resolve(Array.from(u8).join(","));
			});
			ws.on("error", function(e) { reject(String(e)); });
			setTimeout(function() { reject("timeout"); }, 5000);
		});
	})()`)
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()
	// Echo server prepends 0xAA.
	want := "170,255,0,127,195,169"
	if got := val.String(); got != want {
		t.Errorf("binary round-trip got %q, want %q", got, want)
	}
}

func TestWebSocketForwardsHeaders(t *testing.T) {
	srv := wsEchoServer(t)
	url := wsURL(srv)
	b := newTestBridge(t, Encoding(), Events(), NodeStreams(), Buffer(), Timers(), WebSocketPoly())

	val, err := b.EvalAsync("ws-hdr.js", `(async function() {
		return new Promise(function(resolve, reject) {
			var ws = new WebSocket("`+url+`", undefined, { headers: { Authorization: "Bearer t-42" } });
			ws.on("message", function(data) {
				var s = data instanceof Uint8Array ? new TextDecoder().decode(data) : String(data);
				if (s.indexOf("auth:Bearer t-42") === 0) { ws.close(); resolve("ok"); }
			});
			ws.on("error", function(e) { reject(String(e)); });
			setTimeout(function() { reject("timeout"); }, 5000);
		});
	})()`)
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()
	if got := val.String(); got != "ok" {
		t.Errorf("header forward got %q, want ok", got)
	}
}

func TestWebSocketCloseCancelsPendingDial(t *testing.T) {
	entered := make(chan struct{}, 1)
	canceled := make(chan struct{}, 1)
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.EqualFold(r.Header.Get("Upgrade"), "") {
			return
		}
		select {
		case entered <- struct{}{}:
		default:
		}
		select {
		case <-r.Context().Done():
			select {
			case canceled <- struct{}{}:
			default:
			}
		case <-release:
		}
	}))
	t.Cleanup(srv.Close)

	b := newTestBridge(t, Encoding(), Events(), NodeStreams(), Buffer(), Timers(), WebSocketPoly())
	done := make(chan error, 1)
	go func() {
		val, err := b.EvalAsync("ws-close-pending.js", `(async function() {
			return new Promise(function(resolve, reject) {
				var ws = new WebSocket("`+wsURL(srv)+`");
				setTimeout(function() { ws.close(1000, "cancel"); resolve("closed"); }, 100);
			});
		})()`)
		if val != nil {
			val.Free()
		}
		done <- err
	}()

	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("websocket server did not receive pending dial")
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("EvalAsync: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for websocket close script")
	}
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("ws.close did not cancel the pending websocket dial")
	}
	deadline := time.Now().Add(time.Second)
	for {
		if b.DebugSnapshot().ActiveGoroutines == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for websocket pending dial goroutine")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestWebSocketSendContextHasDeadline(t *testing.T) {
	ctx, cancel := newWebSocketSendContext(context.Background())
	defer cancel()
	if _, ok := ctx.Deadline(); !ok {
		t.Fatal("websocket send context has no deadline")
	}
}
