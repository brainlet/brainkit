package jsbridge

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
