package jsbridge

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	quickjs "github.com/buke/quickjs-go"
)

// NetPolyfill provides net.Socket and tls.connect for TCP connections.
// Each socket is backed by a Go net.Conn with async reads via Bridge.Go().
type NetPolyfill struct {
	bridge *Bridge
	mu     sync.Mutex
	conns  map[int64]*goConn
	nextID atomic.Int64
}

type goConn struct {
	id   int64
	conn net.Conn
	done chan struct{}
}

// Net creates a net polyfill.
func Net() *NetPolyfill {
	return &NetPolyfill{conns: make(map[int64]*goConn)}
}

func (p *NetPolyfill) Name() string { return "net" }

func (p *NetPolyfill) SetBridge(b *Bridge) { p.bridge = b }

func (p *NetPolyfill) Setup(ctx *quickjs.Context) error {
	polyfill := p

	// __go_net_connect(host, port, useTLS) → connID
	ctx.Globals().Set("__go_net_connect", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return qctx.ThrowError(fmt.Errorf("net_connect: expected (host, port)"))
		}
		host := args[0].String()
		port := args[1].ToInt32()
		useTLS := len(args) > 2 && args[2].ToBool()

		id := polyfill.nextID.Add(1)

		polyfill.bridge.Go(func(goCtx context.Context) {
			addr := fmt.Sprintf("%s:%d", host, port)
			var conn net.Conn
			var err error

			if useTLS {
				conn, err = tls.Dial("tcp", addr, &tls.Config{ServerName: host})
			} else {
				conn, err = net.Dial("tcp", addr)
			}

			if err != nil {
				qctx.Schedule(func(qctx *quickjs.Context) {
					qctx.Eval(fmt.Sprintf(
						`globalThis.__net_sockets[%d]?._onError(%q)`,
						id, err.Error(),
					))
				})
				return
			}

			gc := &goConn{id: id, conn: conn, done: make(chan struct{})}
			polyfill.mu.Lock()
			polyfill.conns[id] = gc
			polyfill.mu.Unlock()

			// Signal connected
			qctx.Schedule(func(qctx *quickjs.Context) {
				qctx.Eval(fmt.Sprintf(`globalThis.__net_sockets[%d]?._onConnect()`, id))
			})

			// Read loop
			buf := make([]byte, 16384)
			for {
				select {
				case <-goCtx.Done():
					conn.Close()
					return
				case <-gc.done:
					conn.Close()
					return
				default:
				}

				n, readErr := conn.Read(buf)
				if goCtx.Err() != nil {
					return
				}

				if n > 0 {
					// Send data as base64 to avoid escaping issues
					data := base64.StdEncoding.EncodeToString(buf[:n])
					qctx.Schedule(func(qctx *quickjs.Context) {
						qctx.Eval(fmt.Sprintf(
							`globalThis.__net_sockets[%d]?._onData("%s")`,
							id, data,
						))
					})
				}

				if readErr != nil {
					qctx.Schedule(func(qctx *quickjs.Context) {
						qctx.Eval(fmt.Sprintf(`globalThis.__net_sockets[%d]?._onClose()`, id))
					})
					return
				}
			}
		})

		return qctx.NewInt64(id)
	}))

	// __go_net_write(connID, base64data) → bool
	ctx.Globals().Set("__go_net_write", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return qctx.NewBool(false)
		}
		id := args[0].ToInt64()
		b64data := args[1].String()

		data, err := base64.StdEncoding.DecodeString(b64data)
		if err != nil {
			return qctx.NewBool(false)
		}

		polyfill.mu.Lock()
		gc, ok := polyfill.conns[id]
		polyfill.mu.Unlock()
		if !ok {
			return qctx.NewBool(false)
		}

		_, err = gc.conn.Write(data)
		return qctx.NewBool(err == nil)
	}))

	// __go_net_end(connID)
	ctx.Globals().Set("__go_net_end", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.NewUndefined()
		}
		id := args[0].ToInt64()

		polyfill.mu.Lock()
		gc, ok := polyfill.conns[id]
		if ok {
			delete(polyfill.conns, id)
		}
		polyfill.mu.Unlock()

		if ok {
			close(gc.done)
		}
		return qctx.NewUndefined()
	}))

	// JS-side net.Socket implementation
	return evalJS(ctx, netJS)
}

const netJS = `
globalThis.__net_sockets = {};

class GoSocket {
  constructor() {
    this._id = null;
    this._events = {};
    this._connected = false;
    this._destroyed = false;
    this._pendingWrites = [];
  }

  connect(port, host) {
    var self = this;
    this._id = __go_net_connect(host || "127.0.0.1", port, false);
    globalThis.__net_sockets[this._id] = this;

    this._onConnect = function() {
      self._connected = true;
      // Flush pending writes
      for (var i = 0; i < self._pendingWrites.length; i++) {
        self._doWrite(self._pendingWrites[i].data, self._pendingWrites[i].cb);
      }
      self._pendingWrites = [];
      self._emit("connect");
    };
    this._onData = function(b64) {
      var binary = atob(b64);
      var bytes = new Uint8Array(binary.length);
      for (var j = 0; j < binary.length; j++) bytes[j] = binary.charCodeAt(j);
      // pg expects Buffer-like objects
      bytes.toString = function(enc) {
        if (enc === "utf8" || enc === "utf-8" || !enc) return new TextDecoder().decode(this);
        return "[binary " + this.length + " bytes]";
      };
      bytes.slice = function(start, end) {
        var sliced = Uint8Array.prototype.slice.call(this, start, end);
        sliced.toString = bytes.toString;
        sliced.slice = bytes.slice;
        return sliced;
      };
      self._emit("data", bytes);
    };
    this._onError = function(msg) {
      self._emit("error", new Error(msg));
    };
    this._onClose = function() {
      self._connected = false;
      self._destroyed = true;
      if (self._id) delete globalThis.__net_sockets[self._id];
      self._emit("close");
      self._emit("end");
    };

    return this;
  }

  write(data, encoding, cb) {
    if (typeof encoding === "function") { cb = encoding; encoding = undefined; }
    if (!this._connected) {
      this._pendingWrites.push({ data: data, cb: cb });
      return true;
    }
    return this._doWrite(data, cb);
  }

  _doWrite(data, cb) {
    var b64;
    if (typeof data === "string") {
      b64 = btoa(data);
    } else if (data instanceof Uint8Array) {
      var bin = "";
      for (var i = 0; i < data.length; i++) bin += String.fromCharCode(data[i]);
      b64 = btoa(bin);
    } else {
      b64 = btoa(String(data));
    }
    var ok = __go_net_write(this._id, b64);
    if (cb) cb(ok ? null : new Error("write failed"));
    return ok;
  }

  end(data, encoding, cb) {
    if (typeof data === "function") { cb = data; data = undefined; }
    if (data) this.write(data, encoding);
    if (this._id) {
      __go_net_end(this._id);
      delete globalThis.__net_sockets[this._id];
    }
    this._connected = false;
    this._destroyed = true;
    if (cb) cb();
    this._emit("end");
    this._emit("close");
  }

  destroy() {
    this.end();
  }

  setNoDelay() { return this; }
  setKeepAlive() { return this; }
  setTimeout(ms, cb) { if (cb) this.on("timeout", cb); return this; }
  ref() { return this; }
  unref() { return this; }

  on(event, fn) {
    (this._events[event] = this._events[event] || []).push(fn);
    return this;
  }
  once(event, fn) {
    var self = this;
    var wrapper = function() {
      self.removeListener(event, wrapper);
      fn.apply(this, arguments);
    };
    return this.on(event, wrapper);
  }
  removeListener(event, fn) {
    var a = this._events[event];
    if (a) this._events[event] = a.filter(function(f) { return f !== fn; });
    return this;
  }
  emit(event) {
    var args = Array.prototype.slice.call(arguments, 1);
    return this._emit(event, ...args);
  }
  _emit(event) {
    var args = Array.prototype.slice.call(arguments, 1);
    var fns = this._events[event] || [];
    for (var i = 0; i < fns.length; i++) fns[i].apply(this, args);
    return fns.length > 0;
  }

  get readable() { return this._connected; }
  get writable() { return this._connected; }
}

// Override the net stub so pg picks up our real implementation
if (typeof globalThis.__net_override === "undefined") {
  globalThis.__net_override = true;
}
`
