package jsbridge

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"strconv"
	"github.com/brainlet/brainkit/internal/syncx"
	"sync/atomic"
	"time"

	quickjs "github.com/buke/quickjs-go"
)

// NetPolyfill provides net.Socket and tls.connect for TCP connections.
// Each socket is backed by a Go net.Conn with async reads via Bridge.Go().
type NetPolyfill struct {
	bridge *Bridge
	mu     syncx.Mutex
	conns  map[int64]*goConn
	nextID atomic.Int64
}

type goConn struct {
	id       int64
	conn     net.Conn
	done     chan struct{}
	upgraded chan struct{} // closed when TLS upgrade completes; read loop restarts
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
			addr := net.JoinHostPort(host, strconv.Itoa(int(port)))
			var conn net.Conn
			var err error

			// Use context-aware dialer so connections respect bridge cancellation.
			// Without this, net.Dial blocks indefinitely if the server isn't ready.
			dialer := &net.Dialer{Timeout: 30 * time.Second}

			if useTLS {
				conn, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: host})
			} else {
				conn, err = dialer.DialContext(goCtx, "tcp", addr)
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

			// Read loop — use short read deadline so we can check for cancellation.
			// The 500ms deadline keeps the goroutine responsive to cancellation
			// (gc.done / goCtx) while avoiding busy-spinning.
			buf := make([]byte, 16384)
			for {
				conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
				n, readErr := conn.Read(buf)

				// Check cancellation
				if goCtx.Err() != nil {
					conn.Close()
					return
				}
				select {
				case <-gc.done:
					conn.Close()
					return
				default:
				}

				// Timeout is expected — just retry
				if netErr, ok := readErr.(net.Error); ok && netErr.Timeout() {
					continue
				}

				if n > 0 {
					// Split TCP data at wire protocol message boundaries.
					// Many protocols (MongoDB, PostgreSQL) use a 4-byte LE length
					// prefix. When TCP coalesces multiple messages into one read,
					// we schedule a SEPARATE callback for each message. This ensures
					// the QuickJS Await loop can drain microtasks between messages,
					// preventing data from landing in a stale async iterator.
					chunk := buf[:n]
					for len(chunk) > 0 {
						msgLen := len(chunk)
						// Try to read a 4-byte LE length header
						if len(chunk) >= 4 {
							wireLen := int(uint32(chunk[0]) | uint32(chunk[1])<<8 | uint32(chunk[2])<<16 | uint32(chunk[3])<<24)
							if wireLen >= 4 && wireLen <= len(chunk) {
								msgLen = wireLen
							}
						}
						msg := make([]byte, msgLen)
						copy(msg, chunk[:msgLen])
						chunk = chunk[msgLen:]
						data := base64.StdEncoding.EncodeToString(msg)
						qctx.Schedule(func(qctx *quickjs.Context) {
							qctx.Eval(fmt.Sprintf(
								`globalThis.__net_sockets[%d]?._onData("%s")`,
								id, data,
							))
						})
					}
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

	// __go_net_tls_upgrade(connID, servername) → bool
	// Upgrades an existing TCP connection to TLS. Used by pg SSL upgrade.
	// Stops the read loop, wraps conn in crypto/tls.Client, restarts read loop.
	ctx.Globals().Set("__go_net_tls_upgrade", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return qctx.NewBool(false)
		}
		id := args[0].ToInt64()
		servername := args[1].String()

		polyfill.mu.Lock()
		gc, ok := polyfill.conns[id]
		polyfill.mu.Unlock()
		if !ok {
			return qctx.NewBool(false)
		}

		// 1. Stop the current read loop
		close(gc.done)

		// 2. Wrap the raw TCP connection in TLS
		tlsConn := tls.Client(gc.conn, &tls.Config{
			ServerName:         servername,
			InsecureSkipVerify: true, // Containers use self-signed certs
		})

		// 3. Perform TLS handshake (blocks until complete)
		if err := tlsConn.Handshake(); err != nil {
			// Handshake failed — close and signal error
			qctx.Schedule(func(qctx *quickjs.Context) {
				qctx.Eval(fmt.Sprintf(
					`globalThis.__net_sockets[%d]?._onError(%q)`,
					id, err.Error(),
				))
			})
			return qctx.NewBool(false)
		}

		// 4. Create new goConn with TLS connection, start new read loop
		newGC := &goConn{id: id, conn: tlsConn, done: make(chan struct{})}
		polyfill.mu.Lock()
		polyfill.conns[id] = newGC
		polyfill.mu.Unlock()

		polyfill.bridge.Go(func(goCtx context.Context) {
			buf := make([]byte, 16384)
			for {
				tlsConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
				n, readErr := tlsConn.Read(buf)

				if goCtx.Err() != nil {
					return
				}
				select {
				case <-newGC.done:
					return
				default:
				}

				if netErr, ok := readErr.(net.Error); ok && netErr.Timeout() {
					continue
				}

				if n > 0 {
					// For TLS connections, don't try to split by wire protocol
					// length — TLS handles framing. Just forward the decrypted data.
					msg := make([]byte, n)
					copy(msg, buf[:n])
					data := base64.StdEncoding.EncodeToString(msg)
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

		return qctx.NewBool(true)
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

  connect(portOrOpts, host) {
    var self = this;
    var port, useTLS = false;
    if (typeof portOrOpts === "object" && portOrOpts !== null) {
      host = portOrOpts.host || "127.0.0.1";
      port = portOrOpts.port || 27017;
      useTLS = !!portOrOpts.tls || !!portOrOpts.ssl;
    } else {
      port = portOrOpts;
      host = host || "127.0.0.1";
    }
    this._id = __go_net_connect(host, port, useTLS);
    globalThis.__net_sockets[this._id] = this;

    // Store connection details for remoteAddress/remotePort
    this._remoteHost = host;
    this._remotePort = port;

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
      // Pure JS base64 decode — cannot use atob() because it goes through Go's
      // ToString() which truncates at null bytes in the decoded binary data.
      var chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
      var lookup = {};
      for (var ci = 0; ci < chars.length; ci++) lookup[chars[ci]] = ci;
      var bufLen = Math.floor(b64.length * 3 / 4);
      if (b64.length > 1 && b64[b64.length - 1] === "=") bufLen--;
      if (b64.length > 2 && b64[b64.length - 2] === "=") bufLen--;
      var bytes = new Uint8Array(bufLen);
      var p = 0;
      for (var ci = 0; ci < b64.length; ci += 4) {
        var a = lookup[b64[ci]] || 0;
        var b = lookup[b64[ci+1]] || 0;
        var c = lookup[b64[ci+2]] || 0;
        var d = lookup[b64[ci+3]] || 0;
        bytes[p++] = (a << 2) | (b >> 4);
        if (b64[ci+2] !== "=") bytes[p++] = ((b << 4) | (c >> 2)) & 0xff;
        if (b64[ci+3] !== "=") bytes[p++] = ((c << 6) | d) & 0xff;
      }
      var buf = globalThis.Buffer ? globalThis.Buffer.from(bytes) : bytes;
      self._emit("data", buf);
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
    // Encode binary data to base64 for Go bridge.
    // Cannot use btoa() because it goes through Go's ToString() which truncates at null bytes.
    // Must use pure-JS base64 encoding that handles all byte values including 0x00.
    var bytes;
    if (typeof data === "string") {
      bytes = [];
      for (var i = 0; i < data.length; i++) bytes.push(data.charCodeAt(i) & 0xff);
    } else if (data instanceof Uint8Array || (data && data._isBuffer)) {
      bytes = data;
    } else if (data && typeof data.length === "number") {
      bytes = data;
    } else {
      var s = String(data);
      bytes = [];
      for (var i = 0; i < s.length; i++) bytes.push(s.charCodeAt(i) & 0xff);
    }

    // Pure JS base64 encode (handles null bytes correctly)
    var chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    var b64 = "";
    var len = bytes.length;
    for (var i = 0; i < len; i += 3) {
      var b0 = bytes[i];
      var b1 = i + 1 < len ? bytes[i + 1] : 0;
      var b2 = i + 2 < len ? bytes[i + 2] : 0;
      b64 += chars[(b0 >> 2) & 0x3f];
      b64 += chars[((b0 << 4) | (b1 >> 4)) & 0x3f];
      b64 += (i + 1 < len) ? chars[((b1 << 2) | (b2 >> 6)) & 0x3f] : "=";
      b64 += (i + 2 < len) ? chars[b2 & 0x3f] : "=";
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

  destroy(err) {
    if (this._destroyed) return this;
    this._destroyed = true;
    this._connected = false;
    if (this._timeoutId) { clearTimeout(this._timeoutId); this._timeoutId = null; }
    if (this._id) {
      __go_net_end(this._id);
      delete globalThis.__net_sockets[this._id];
    }
    if (err) this._emit("error", err);
    this._emit("close");
    return this;
  }

  // pipe — Node.js Readable.pipe(). Forward data events to a Writable/Transform.
  pipe(dest, opts) {
    var self = this;
    this.on("data", function(chunk) {
      var ok = dest.write(chunk);
      if (ok === false && self.pause) self.pause();
    });
    if (!opts || opts.end !== false) {
      this.on("end", function() { if (dest.end) dest.end(); });
      this.on("close", function() { if (dest.end) dest.end(); });
    }
    if (dest.on) {
      dest.on("drain", function() { if (self.resume) self.resume(); });
    }
    dest.emit && dest.emit("pipe", this);
    return dest;
  }

  unpipe(dest) { return this; }
  pause() { this._paused = true; return this; }
  resume() { this._paused = false; return this; }

  get remoteAddress() { return this._remoteHost || ""; }
  get remotePort() { return this._remotePort || 0; }
  get writable() { return this._connected && !this._destroyed; }

  setNoDelay() { return this; }
  setKeepAlive() { return this; }
  setTimeout(ms, cb) {
    if (cb) this.once("timeout", cb);
    // Clear existing timeout
    if (this._timeoutId) { clearTimeout(this._timeoutId); this._timeoutId = null; }
    if (ms > 0) {
      var self = this;
      this._timeoutId = setTimeout(function() { self._emit("timeout"); }, ms);
    }
    return this;
  }
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
    var fns = (this._events[event] || []).slice();
    for (var i = 0; i < fns.length; i++) fns[i].apply(this, args);
    return fns.length > 0;
  }

  get readable() { return this._connected; }
  get writable() { return this._connected; }
}

// Expose GoSocket globally so the esbuild net stub can delegate to it
globalThis.GoSocket = GoSocket;

// ─── Socket extends Duplex ──────────────────────────────────────────
// net.Socket wraps GoSocket (raw TCP transport) with proper Node.js Duplex
// inheritance from globalThis.stream. This gives us Readable.pipe() with
// pause/resume/buffer semantics, Symbol.asyncIterator with correct data
// handoff, and the full Node.js stream contract.
//
// GoSocket pushes data INTO the Duplex via this.push(chunk).
// Writes go through GoSocket.write() directly (bypass Duplex._write).
(function() {
  var Duplex = globalThis.stream && globalThis.stream.Duplex;
  if (!Duplex) return; // NodeStreams not loaded yet

  class Socket extends Duplex {
    constructor() {
      super();
      this._gs = new GoSocket();
    }
    connect(portOrOpts, host) {
      this._gs.connect(portOrOpts, host);
      var self = this;
      // Wire GoSocket events → Duplex push/emit
      this._gs.on("data", function(chunk) { self.push(chunk); });
      this._gs.on("end", function() { self.push(null); });
      this._gs.on("error", function(err) { self.destroy(err); });
      this._gs.on("connect", function() { self.emit("connect"); });
      this._gs.on("close", function() { if (!self.destroyed) self.destroy(); });
      this._gs.on("timeout", function() { self.emit("timeout"); });
      return this;
    }
    // Write goes directly to GoSocket — bypass Duplex._write
    write(data, encoding, cb) {
      if (typeof encoding === "function") { cb = encoding; encoding = undefined; }
      return this._gs.write(data, encoding, cb);
    }
    end(data, encoding, cb) {
      if (typeof data === "function") { cb = data; data = undefined; }
      if (data) this.write(data, encoding);
      if (this._gs) this._gs.end();
      this.writable = false;
      if (cb) cb();
      return this;
    }
    destroy(err) {
      if (this.destroyed) return this;
      this.destroyed = true;
      if (this._gs) this._gs.destroy(err);
      if (err) this.emit("error", err);
      this.emit("close");
      return this;
    }
    setNoDelay() { return this; }
    setKeepAlive() { return this; }
    setTimeout(ms, cb) { if (this._gs) this._gs.setTimeout(ms, cb); return this; }
    ref() { return this; }
    unref() { return this; }
    cork() {}
    uncork() {}
    get remoteAddress() { return this._gs ? this._gs.remoteAddress : undefined; }
    get remotePort() { return this._gs ? this._gs.remotePort : undefined; }
  }

  var createConnection = function(portOrOpts, host) {
    var s = new Socket();
    s.connect(portOrOpts, host);
    return s;
  };

  var notAvailable = function(name) { return function() { throw new Error(name + ": not available in QuickJS"); }; };
  var isIP = function(input) { try { return input.includes(":") ? 6 : input.match(/^\d+\.\d+\.\d+\.\d+$/) ? 4 : 0; } catch(e) { return 0; } };

  globalThis.net = {
    Socket: Socket,
    createConnection: createConnection,
    connect: createConnection,
    createServer: notAvailable("net.createServer"),
    Server: class Server {},
    isIP: isIP,
    isIPv4: function(input) { return isIP(input) === 4; },
    isIPv6: function(input) { return isIP(input) === 6; },
  };
})();
`
