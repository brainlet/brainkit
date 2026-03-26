package jsbridge

import quickjs "github.com/buke/quickjs-go"

// NodeStreamsPolyfill provides Node.js stream classes: Readable, Writable, Duplex,
// Transform, PassThrough. These are the Node.js stream API (events + pipe + backpressure),
// NOT the Web Streams API (ReadableStream/WritableStream) which is in streams.go.
//
// The MongoDB driver, pg driver, and many npm packages depend on these classes.
// Previously implemented inline in agent/bundle/build.mjs — now a proper jsbridge
// polyfill with Go test coverage.
//
// IMPORTANT: Must be loaded AFTER Events polyfill (uses EventEmitter).
type NodeStreamsPolyfill struct{}

// NodeStreams creates a Node.js streams polyfill.
func NodeStreams() *NodeStreamsPolyfill { return &NodeStreamsPolyfill{} }

func (p *NodeStreamsPolyfill) Name() string { return "nodestreams" }

func (p *NodeStreamsPolyfill) Setup(ctx *quickjs.Context) error {
	return evalJS(ctx, nodeStreamsJS)
}

// nodeStreamsJS implements Node.js stream classes on globalThis.stream.
//
// Key design decisions for MongoDB SCRAM compatibility:
//
// 1. Readable._buffer: data pushed while paused or with no listeners is buffered.
//    Adding a "data" listener flushes the buffer (switches to flowing mode).
//
// 2. Readable[Symbol.asyncIterator]: creates an iterator that listens for "data"
//    events. When the iterator's return() is called (for-await exits early), it
//    removes its listener BUT transfers any unconsumed data back to the Readable's
//    _buffer via unshift(). The NEXT async iterator will see that buffered data.
//    This is critical for MongoDB's conn.command() pattern where hello and saslStart
//    responses can arrive before the first for-await chain fully unwinds.
//
// 3. Transform._write → _transform → push: synchronous by default. The push()
//    goes to Readable's buffer/emit path. Subclasses override _transform.
const nodeStreamsJS = `
(function() {
  "use strict";

  var EE = globalThis.EventEmitter;

  // ─── Readable ──────────────────────────────────────────────────
  class Readable extends EE {
    constructor(opts) {
      super();
      this.readable = true;
      this.destroyed = false;
      this._paused = true;
      this._buffer = [];
      this._ended = false;
      this._readableObjectMode = opts && opts.readableObjectMode;
      if (opts && typeof opts.read === "function") this._read = opts.read;
    }

    // Adding a "data" listener switches to flowing mode (flushes buffer).
    on(ev, fn) {
      super.on(ev, fn);
      if (ev === "data" && this._paused) this.resume();
      return this;
    }

    push(chunk) {
      if (chunk === null) {
        this._ended = true;
        this.emit("end");
        return false;
      }
      // Buffer if paused OR no "data" listeners
      if (this._paused || !this._e["data"] || !this._e["data"].length) {
        this._buffer.push(chunk);
        return true;
      }
      this.emit("data", chunk);
      return true;
    }

    read(size) {
      return this._buffer.length ? this._buffer.shift() : null;
    }

    unshift(chunk) {
      this._buffer.unshift(chunk);
    }

    resume() {
      this._paused = false;
      while (this._buffer.length && this._e["data"] && this._e["data"].length) {
        this.emit("data", this._buffer.shift());
      }
      return this;
    }

    pause() {
      this._paused = true;
      return this;
    }

    isPaused() {
      return this._paused;
    }

    pipe(dest, opts) {
      var self = this;
      this.on("data", function(chunk) {
        if (dest.write) {
          var ok = dest.write(chunk);
          if (ok === false && self.pause) self.pause();
        }
      });
      if (!opts || opts.end !== false) {
        this.on("end", function() { if (dest.end) dest.end(); });
      }
      if (dest.on) {
        dest.on("drain", function() { if (self.resume) self.resume(); });
      }
      if (dest.emit) dest.emit("pipe", this);
      return dest;
    }

    unpipe() { return this; }

    destroy(err) {
      if (this.destroyed) return this;
      this.destroyed = true;
      if (err) this.emit("error", err);
      this.emit("close");
      return this;
    }

    setEncoding() { return this; }

    // Symbol.asyncIterator — MongoDB's onData pattern depends on this.
    // CRITICAL: when return() is called (for-await exits), unconsumed data
    // is transferred back to the Readable's _buffer via unshift(). This
    // prevents data loss between consecutive conn.command() calls.
    [Symbol.asyncIterator]() {
      var self = this;
      var done = false;
      var queue = [];
      var waiting = null;

      function onData(chunk) {
        if (waiting) {
          var r = waiting;
          waiting = null;
          r({ value: chunk, done: false });
        } else {
          queue.push({ value: chunk, done: false });
        }
      }
      function onEnd() {
        done = true;
        if (waiting) {
          var r = waiting;
          waiting = null;
          r({ value: undefined, done: true });
        }
      }
      function onError(err) {
        done = true;
        if (waiting) {
          var r = waiting;
          waiting = null;
          r(Promise.reject(err));
        }
      }

      self.on("data", onData);
      self.on("end", onEnd);
      self.on("error", onError);
      self.resume();

      return {
        next: function() {
          if (queue.length) return Promise.resolve(queue.shift());
          if (done) return Promise.resolve({ value: undefined, done: true });
          return new Promise(function(r) { waiting = r; });
        },
        return: function() {
          // Remove our listeners — stop receiving new data
          self.removeListener("data", onData);
          self.removeListener("end", onEnd);
          self.removeListener("error", onError);
          self.pause();

          // CRITICAL FIX for MongoDB SCRAM:
          // Transfer unconsumed items back to the Readable's buffer.
          // The NEXT async iterator (from the next conn.command() call)
          // will see this data when it calls resume().
          for (var i = queue.length - 1; i >= 0; i--) {
            self.unshift(queue[i].value);
          }
          queue = [];
          done = true;

          return Promise.resolve({ value: undefined, done: true });
        },
        [Symbol.asyncIterator]: function() { return this; },
      };
    }
  }

  Readable.prototype.addListener = Readable.prototype.on;

  Readable.from = function(iterable) {
    var r = new Readable();
    if (iterable && iterable[Symbol.iterator]) {
      for (var v of iterable) r.push(v);
      r.push(null);
    }
    return r;
  };

  // ─── Writable ──────────────────────────────────────────────────
  class Writable extends EE {
    constructor(opts) {
      super();
      this.writable = true;
      this.destroyed = false;
      this._writableObjectMode = opts && opts.writableObjectMode;
      if (opts && typeof opts.write === "function") this._write = opts.write;
    }

    write(chunk, enc, cb) {
      if (typeof enc === "function") { cb = enc; enc = undefined; }
      if (this._write) {
        this._write(chunk, enc || "utf8", cb || function() {});
      } else {
        if (cb) cb();
      }
      return true;
    }

    end(chunk, enc, cb) {
      if (typeof chunk === "function") { cb = chunk; chunk = undefined; }
      if (typeof enc === "function") { cb = enc; enc = undefined; }
      if (chunk !== undefined && chunk !== null) this.write(chunk, enc);
      this.writable = false;
      if (this._final) {
        this._final(function() { if (cb) cb(); });
      } else {
        if (cb) cb();
      }
      this.emit("finish");
    }

    destroy(err) {
      if (this.destroyed) return this;
      this.destroyed = true;
      if (err) this.emit("error", err);
      this.emit("close");
      return this;
    }

    cork() {}
    uncork() {}
  }

  // ─── Duplex ────────────────────────────────────────────────────
  class Duplex extends Readable {
    constructor(opts) {
      super(opts);
      this.writable = true;
      this._writableObjectMode = opts && opts.writableObjectMode;
      if (opts && typeof opts.write === "function") this._write = opts.write;
    }

    write(chunk, enc, cb) {
      if (typeof enc === "function") { cb = enc; enc = undefined; }
      if (this._write) {
        this._write(chunk, enc || "utf8", cb || function() {});
      } else {
        if (cb) cb();
      }
      return true;
    }

    end(chunk, enc, cb) {
      if (typeof chunk === "function") { cb = chunk; chunk = undefined; }
      if (typeof enc === "function") { cb = enc; enc = undefined; }
      if (chunk !== undefined && chunk !== null) this.write(chunk, enc);
      this.writable = false;
      if (this._final) {
        this._final(function() { if (cb) cb(); });
      } else {
        if (cb) cb();
      }
      this.emit("finish");
    }

    destroy(err) {
      if (this.destroyed) return this;
      this.destroyed = true;
      if (err) this.emit("error", err);
      this.emit("close");
      return this;
    }

    cork() {}
    uncork() {}
  }

  // ─── Transform ─────────────────────────────────────────────────
  class Transform extends Duplex {
    constructor(opts) {
      super(opts);
      this._readableObjectMode = opts && opts.readableObjectMode;
      this._writableObjectMode = opts && (opts.writableObjectMode !== undefined ? opts.writableObjectMode : false);
      if (opts && typeof opts.transform === "function") this._transform = opts.transform;
    }

    // write() → _transform() → push() → data events
    _write(chunk, enc, cb) {
      this._transform(chunk, enc, cb);
    }

    _transform(chunk, enc, cb) {
      this.push(chunk);
      cb();
    }

    _flush(cb) { cb(); }
  }

  // ─── PassThrough ───────────────────────────────────────────────
  class PassThrough extends Transform {}

  // ─── pipeline / finished stubs ─────────────────────────────────
  function pipeline() {
    var args = Array.prototype.slice.call(arguments);
    var cb = args.pop();
    if (typeof cb === "function") cb();
  }
  function finished(stream, cb) {
    if (cb) cb();
  }

  // ─── Export on globalThis ──────────────────────────────────────
  globalThis.stream = {
    Readable: Readable,
    Writable: Writable,
    Duplex: Duplex,
    Transform: Transform,
    PassThrough: PassThrough,
    pipeline: pipeline,
    finished: finished,
    Stream: Readable,
  };
})();
`
