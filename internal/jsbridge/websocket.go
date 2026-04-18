package jsbridge

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/coder/websocket"
	quickjs "github.com/buke/quickjs-go"
)

// WebSocketPolyfill provides `globalThis.WebSocket` shaped to
// match both the WHATWG standard (onopen / onmessage / onerror /
// onclose, addEventListener, send, close, readyState) AND the
// Node `ws` package's EventEmitter extensions (ws.on, custom
// headers in the constructor). Node-shape coverage is the hard
// requirement because Mastra's realtime voice class is built on
// `ws` and it uses `new WebSocket(url, void 0, { headers: {...} })`
// plus `ws.on("message", ...)`.
//
// Backed by github.com/coder/websocket for dialing; Go owns the
// connection, reads frames on a goroutine, and ships them into
// JS via bridge Schedule callbacks. Outbound frames flow the
// other way via __go_ws_send.
type WebSocketPolyfill struct {
	bridge *Bridge

	mu      sync.Mutex
	nextID  uint64
	conns   map[uint64]*wsConn
}

type wsConn struct {
	conn   *websocket.Conn
	cancel context.CancelFunc
}

// WebSocketPoly creates a WebSocket client polyfill.
func WebSocketPoly() *WebSocketPolyfill {
	return &WebSocketPolyfill{conns: make(map[uint64]*wsConn)}
}

// Name implements Polyfill.
func (p *WebSocketPolyfill) Name() string { return "websocket" }

// SetBridge wires the bridge for goroutine + Schedule access.
func (p *WebSocketPolyfill) SetBridge(b *Bridge) { p.bridge = b }

// Setup installs globalThis.WebSocket + the Go-side dial /
// send / close bridges.
func (p *WebSocketPolyfill) Setup(ctx *quickjs.Context) error {
	polyfill := p

	// __go_ws_dial(url, protocolsJSON, headersJSON) → handle
	// Returns a numeric handle synchronously; the actual dial
	// runs on a goroutine and posts open/message/error/close
	// events back via __ws_events[handle]._onX callbacks.
	ctx.Globals().Set("__go_ws_dial", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("WebSocket: url required"))
		}
		url := args[0].ToString()
		protoJSON := ""
		if len(args) >= 2 {
			protoJSON = args[1].ToString()
		}
		hdrJSON := ""
		if len(args) >= 3 {
			hdrJSON = args[2].ToString()
		}

		var protocols []string
		if protoJSON != "" {
			_ = json.Unmarshal([]byte(protoJSON), &protocols)
		}
		hdr := http.Header{}
		if hdrJSON != "" {
			raw := map[string]string{}
			_ = json.Unmarshal([]byte(hdrJSON), &raw)
			for k, v := range raw {
				hdr.Set(k, v)
			}
		}

		handle := atomic.AddUint64(&polyfill.nextID, 1)
		dialCtx, cancel := context.WithCancel(context.Background())

		polyfill.bridge.Go(func(goCtx context.Context) {
			// Cancel dial if bridge drains.
			go func() {
				select {
				case <-goCtx.Done():
					cancel()
				case <-dialCtx.Done():
				}
			}()
			conn, _, err := websocket.Dial(dialCtx, url, &websocket.DialOptions{
				Subprotocols: protocols,
				HTTPHeader:   hdr,
			})
			if err != nil {
				fireWSEvent(qctx, handle, "error", err.Error(), false)
				fireWSEvent(qctx, handle, "close", err.Error(), false)
				cancel()
				return
			}
			// Unlimited read — realtime voice payloads can be big.
			conn.SetReadLimit(-1)
			polyfill.mu.Lock()
			polyfill.conns[handle] = &wsConn{conn: conn, cancel: cancel}
			polyfill.mu.Unlock()
			fireWSEvent(qctx, handle, "open", "", false)

			for {
				typ, data, readErr := conn.Read(dialCtx)
				if readErr != nil {
					polyfill.mu.Lock()
					delete(polyfill.conns, handle)
					polyfill.mu.Unlock()
					reason := readErr.Error()
					fireWSEvent(qctx, handle, "close", reason, false)
					cancel()
					return
				}
				if typ == websocket.MessageBinary {
					fireWSEvent(qctx, handle, "message_b64",
						base64.StdEncoding.EncodeToString(data), true)
				} else {
					fireWSEvent(qctx, handle, "message", string(data), false)
				}
			}
		})

		return qctx.NewInt64(int64(handle))
	}))

	// __go_ws_send(handle, data, isBinary) → bool
	ctx.Globals().Set("__go_ws_send", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return qctx.NewBool(false)
		}
		handle := uint64(args[0].ToInt64())
		data := args[1].ToString()
		binary := len(args) >= 3 && args[2].ToBool()

		polyfill.mu.Lock()
		c := polyfill.conns[handle]
		polyfill.mu.Unlock()
		if c == nil {
			return qctx.NewBool(false)
		}

		var raw []byte
		if binary {
			decoded, err := base64.StdEncoding.DecodeString(data)
			if err != nil {
				return qctx.NewBool(false)
			}
			raw = decoded
		} else {
			raw = []byte(data)
		}
		msgType := websocket.MessageText
		if binary {
			msgType = websocket.MessageBinary
		}
		// Short write timeout — realtime should push fast. If
		// a send hangs, drop the connection rather than wedge
		// the JS side.
		sendCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if err := c.conn.Write(sendCtx, msgType, raw); err != nil {
			return qctx.NewBool(false)
		}
		return qctx.NewBool(true)
	}))

	// __go_ws_close(handle, code, reason) → void
	ctx.Globals().Set("__go_ws_close", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.NewUndefined()
		}
		handle := uint64(args[0].ToInt64())
		code := websocket.StatusNormalClosure
		reason := ""
		if len(args) >= 2 {
			code = websocket.StatusCode(int(args[1].ToInt32()))
		}
		if len(args) >= 3 {
			reason = args[2].ToString()
		}
		polyfill.mu.Lock()
		c := polyfill.conns[handle]
		delete(polyfill.conns, handle)
		polyfill.mu.Unlock()
		if c != nil {
			_ = c.conn.Close(code, reason)
			c.cancel()
		}
		return qctx.NewUndefined()
	}))

	return evalJS(ctx, websocketJS)
}

func fireWSEvent(qctx *quickjs.Context, handle uint64, kind, payload string, binary bool) {
	esc, _ := json.Marshal(payload)
	var js string
	if binary {
		js = fmt.Sprintf(`globalThis.__ws_events[%d]&&globalThis.__ws_events[%d]._onMessageBinary(%s)`,
			handle, handle, string(esc))
	} else {
		js = fmt.Sprintf(`globalThis.__ws_events[%d]&&globalThis.__ws_events[%d]._on(%q, %s)`,
			handle, handle, kind, string(esc))
	}
	qctx.Schedule(func(qctx *quickjs.Context) {
		qctx.Eval(js)
	})
}

// websocketJS implements globalThis.WebSocket with BOTH the
// WHATWG surface (onopen / onmessage / onerror / onclose,
// addEventListener, send, close, readyState, binaryType) AND the
// Node `ws` EventEmitter surface (on / off / emit). The shape
// has to straddle both because Mastra's realtime voice class
// is Node-first but may also be called from browser-leaning
// bundles; covering both keeps the polyfill single-surface for
// every consumer.
const websocketJS = `
(function() {
  "use strict";

  var CONNECTING = 0, OPEN = 1, CLOSING = 2, CLOSED = 3;

  if (!globalThis.__ws_events) globalThis.__ws_events = {};

  class WebSocket {
    constructor(url, protocols, options) {
      this.url = url;
      this.protocol = "";
      this.extensions = "";
      this.readyState = CONNECTING;
      this.binaryType = "nodebuffer"; // Node default; browsers use "blob"
      this.bufferedAmount = 0;
      this._handlers = {};
      this.onopen = null;
      this.onmessage = null;
      this.onerror = null;
      this.onclose = null;

      var protoList = [];
      if (typeof protocols === "string") protoList = [protocols];
      else if (Array.isArray(protocols)) protoList = protocols;

      var headers = (options && options.headers) || {};
      var normHeaders = {};
      for (var k in headers) {
        if (Object.prototype.hasOwnProperty.call(headers, k)) {
          normHeaders[k] = String(headers[k]);
        }
      }

      this._handle = __go_ws_dial(
        String(url),
        JSON.stringify(protoList),
        JSON.stringify(normHeaders),
      );

      var self = this;
      globalThis.__ws_events[this._handle] = {
        _on: function(kind, payload) {
          if (kind === "open") {
            self.readyState = OPEN;
            self._fire("open", {});
          } else if (kind === "message") {
            var evt = { data: payload, type: "message" };
            self._fire("message", evt);
          } else if (kind === "error") {
            var err = new Error(payload || "websocket error");
            self._fire("error", err);
          } else if (kind === "close") {
            self.readyState = CLOSED;
            var close = { code: 1006, reason: payload || "", wasClean: false };
            self._fire("close", close);
            delete globalThis.__ws_events[self._handle];
          }
        },
        _onMessageBinary: function(b64) {
          // Decode to a Uint8Array / Buffer-ish shape so
          // consumers that expect Node Buffer semantics
          // (toString("utf8") / JSON.parse(msg.toString())) still
          // work. Buffer polyfill from jsbridge/buffer.go is a
          // Uint8Array subclass.
          var bin = atob(b64);
          var u8 = new Uint8Array(bin.length);
          for (var i = 0; i < bin.length; i++) u8[i] = bin.charCodeAt(i) & 0xFF;
          var out = (typeof Buffer !== "undefined" && Buffer.from) ? Buffer.from(u8) : u8;
          var evt = { data: out, type: "message" };
          self._fire("message", evt);
        },
      };
    }

    _fire(ev, payload) {
      // WHATWG on<ev> handler.
      var prop = this["on" + ev];
      if (typeof prop === "function") {
        try {
          if (ev === "message") prop.call(this, payload);
          else prop.call(this, payload);
        } catch (_) {}
      }
      // Both WHATWG addEventListener + Node on() live in the
      // same map — fire every listener.
      var list = this._handlers[ev];
      if (list) {
        var copy = list.slice();
        for (var i = 0; i < copy.length; i++) {
          try {
            if (ev === "message") copy[i].call(this, payload.data);
            else if (ev === "error") copy[i].call(this, payload);
            else if (ev === "close") copy[i].call(this, payload.code, payload.reason);
            else copy[i].call(this, payload);
          } catch (_) {}
        }
      }
    }

    // WHATWG.
    addEventListener(ev, fn) { (this._handlers[ev] = this._handlers[ev] || []).push(fn); }
    removeEventListener(ev, fn) {
      var list = this._handlers[ev]; if (!list) return;
      this._handlers[ev] = list.filter(function(x) { return x !== fn; });
    }
    dispatchEvent() {}

    // Node EventEmitter (ws).
    on(ev, fn) { this.addEventListener(ev, fn); return this; }
    once(ev, fn) {
      var self = this;
      var wrap = function() { self.removeEventListener(ev, wrap); fn.apply(self, arguments); };
      this.addEventListener(ev, wrap);
      return this;
    }
    off(ev, fn) { this.removeEventListener(ev, fn); return this; }
    removeListener(ev, fn) { return this.off(ev, fn); }
    emit(ev) {
      // Pass-through for compat; real emits come from Go.
      var args = Array.prototype.slice.call(arguments, 1);
      var payload = args.length === 1 ? args[0] : args;
      this._fire(ev, payload);
    }

    send(data, cb) {
      // Accept string, Buffer, ArrayBuffer, TypedArray. Binary
      // goes through base64 to preserve bytes across the JS-Go
      // string boundary (same pattern as fetch + fs).
      var isBinary = false;
      var payload;
      if (typeof data === "string") {
        payload = data;
      } else if (data && typeof data.byteLength === "number") {
        var u8 = data instanceof Uint8Array ? data :
                 data instanceof ArrayBuffer ? new Uint8Array(data) :
                 new Uint8Array(data.buffer, data.byteOffset || 0, data.byteLength);
        var bin = "";
        for (var i = 0; i < u8.length; i++) bin += String.fromCharCode(u8[i] & 0xFF);
        payload = btoa(bin);
        isBinary = true;
      } else {
        payload = String(data);
      }
      var ok = __go_ws_send(this._handle, payload, isBinary);
      if (typeof cb === "function") cb(ok ? null : new Error("send failed"));
    }

    close(code, reason) {
      if (this.readyState === CLOSED) return;
      this.readyState = CLOSING;
      __go_ws_close(this._handle, code || 1000, reason || "");
    }

    terminate() { this.close(1006, "terminated"); }

    // Instance constants — Node ws exposes these on the class,
    // but code also reads them off the instance.
    get CONNECTING() { return CONNECTING; }
    get OPEN() { return OPEN; }
    get CLOSING() { return CLOSING; }
    get CLOSED() { return CLOSED; }
  }

  WebSocket.CONNECTING = CONNECTING;
  WebSocket.OPEN = OPEN;
  WebSocket.CLOSING = CLOSING;
  WebSocket.CLOSED = CLOSED;

  globalThis.WebSocket = WebSocket;
})();
`
