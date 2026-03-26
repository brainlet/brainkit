package jsbridge

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"io"

	quickjs "github.com/buke/quickjs-go"
)

// ZlibPolyfill provides zlib.inflate, zlib.deflate, zlib.gunzipSync, and related
// compression functions. MongoDB wire protocol uses zlib compression when the server
// has compressor: ["zlib"] configured. The driver calls zlib.inflate/deflate with
// Node.js-style (buf, callback) signatures.
type ZlibPolyfill struct{}

// Zlib creates a zlib polyfill.
func Zlib() *ZlibPolyfill { return &ZlibPolyfill{} }

func (p *ZlibPolyfill) Name() string { return "zlib" }

func (p *ZlibPolyfill) Setup(ctx *quickjs.Context) error {
	// __go_zlib_inflate(base64data) → base64result
	ctx.Globals().Set("__go_zlib_inflate", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("zlib.inflate: data required"))
		}
		data, err := base64.StdEncoding.DecodeString(args[0].String())
		if err != nil {
			return qctx.ThrowError(fmt.Errorf("zlib.inflate: invalid base64: %w", err))
		}
		r, err := zlib.NewReader(bytes.NewReader(data))
		if err != nil {
			return qctx.ThrowError(fmt.Errorf("zlib.inflate: %w", err))
		}
		defer r.Close()
		out, err := io.ReadAll(r)
		if err != nil {
			return qctx.ThrowError(fmt.Errorf("zlib.inflate: %w", err))
		}
		return qctx.NewString(base64.StdEncoding.EncodeToString(out))
	}))

	// __go_zlib_deflate(base64data, level) → base64result
	ctx.Globals().Set("__go_zlib_deflate", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("zlib.deflate: data required"))
		}
		data, err := base64.StdEncoding.DecodeString(args[0].String())
		if err != nil {
			return qctx.ThrowError(fmt.Errorf("zlib.deflate: invalid base64: %w", err))
		}
		level := flate.DefaultCompression
		if len(args) > 1 {
			l := int(args[1].ToInt32())
			if l >= -1 && l <= 9 {
				level = l
			}
		}
		var buf bytes.Buffer
		w, err := zlib.NewWriterLevel(&buf, level)
		if err != nil {
			return qctx.ThrowError(fmt.Errorf("zlib.deflate: %w", err))
		}
		if _, err := w.Write(data); err != nil {
			return qctx.ThrowError(fmt.Errorf("zlib.deflate: %w", err))
		}
		if err := w.Close(); err != nil {
			return qctx.ThrowError(fmt.Errorf("zlib.deflate: %w", err))
		}
		return qctx.NewString(base64.StdEncoding.EncodeToString(buf.Bytes()))
	}))

	// __go_gzip_decompress(base64data) → base64result
	ctx.Globals().Set("__go_gzip_decompress", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("gunzip: data required"))
		}
		data, err := base64.StdEncoding.DecodeString(args[0].String())
		if err != nil {
			return qctx.ThrowError(fmt.Errorf("gunzip: invalid base64: %w", err))
		}
		r, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return qctx.ThrowError(fmt.Errorf("gunzip: %w", err))
		}
		defer r.Close()
		out, err := io.ReadAll(r)
		if err != nil {
			return qctx.ThrowError(fmt.Errorf("gunzip: %w", err))
		}
		return qctx.NewString(base64.StdEncoding.EncodeToString(out))
	}))

	// __go_gzip_compress(base64data) → base64result
	ctx.Globals().Set("__go_gzip_compress", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("gzip: data required"))
		}
		data, err := base64.StdEncoding.DecodeString(args[0].String())
		if err != nil {
			return qctx.ThrowError(fmt.Errorf("gzip: invalid base64: %w", err))
		}
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		if _, err := w.Write(data); err != nil {
			return qctx.ThrowError(fmt.Errorf("gzip: %w", err))
		}
		if err := w.Close(); err != nil {
			return qctx.ThrowError(fmt.Errorf("gzip: %w", err))
		}
		return qctx.NewString(base64.StdEncoding.EncodeToString(buf.Bytes()))
	}))

	// __go_raw_inflate(base64data) → base64result (raw deflate, no zlib header)
	ctx.Globals().Set("__go_raw_inflate", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("inflateRaw: data required"))
		}
		data, err := base64.StdEncoding.DecodeString(args[0].String())
		if err != nil {
			return qctx.ThrowError(fmt.Errorf("inflateRaw: invalid base64: %w", err))
		}
		r := flate.NewReader(bytes.NewReader(data))
		defer r.Close()
		out, err := io.ReadAll(r)
		if err != nil {
			return qctx.ThrowError(fmt.Errorf("inflateRaw: %w", err))
		}
		return qctx.NewString(base64.StdEncoding.EncodeToString(out))
	}))

	// __go_raw_deflate(base64data, level) → base64result (raw deflate, no zlib header)
	ctx.Globals().Set("__go_raw_deflate", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("deflateRaw: data required"))
		}
		data, err := base64.StdEncoding.DecodeString(args[0].String())
		if err != nil {
			return qctx.ThrowError(fmt.Errorf("deflateRaw: invalid base64: %w", err))
		}
		level := flate.DefaultCompression
		if len(args) > 1 {
			l := int(args[1].ToInt32())
			if l >= -1 && l <= 9 {
				level = l
			}
		}
		var buf bytes.Buffer
		w, err := flate.NewWriter(&buf, level)
		if err != nil {
			return qctx.ThrowError(fmt.Errorf("deflateRaw: %w", err))
		}
		if _, err := w.Write(data); err != nil {
			return qctx.ThrowError(fmt.Errorf("deflateRaw: %w", err))
		}
		if err := w.Close(); err != nil {
			return qctx.ThrowError(fmt.Errorf("deflateRaw: %w", err))
		}
		return qctx.NewString(base64.StdEncoding.EncodeToString(buf.Bytes()))
	}))

	return evalJS(ctx, zlibJS)
}

const zlibJS = `
(function() {
  "use strict";

  // Pure-JS base64 helpers (same as crypto.go — null-byte safe)
  var _b64c = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
  var _b64l = {}; for (var _i = 0; _i < _b64c.length; _i++) _b64l[_b64c[_i]] = _i;

  function toB64(bytes) {
    var b64 = "";
    for (var i = 0; i < bytes.length; i += 3) {
      var b0 = bytes[i], b1 = i+1<bytes.length?bytes[i+1]:0, b2 = i+2<bytes.length?bytes[i+2]:0;
      b64 += _b64c[(b0>>2)&0x3f]; b64 += _b64c[((b0<<4)|(b1>>4))&0x3f];
      b64 += (i+1<bytes.length) ? _b64c[((b1<<2)|(b2>>6))&0x3f] : "=";
      b64 += (i+2<bytes.length) ? _b64c[b2&0x3f] : "=";
    }
    return b64;
  }

  function fromB64(b64) {
    var bufLen = Math.floor(b64.length * 3 / 4);
    if (b64.length > 1 && b64[b64.length-1] === "=") bufLen--;
    if (b64.length > 2 && b64[b64.length-2] === "=") bufLen--;
    var bytes = new Uint8Array(bufLen); var p = 0;
    for (var i = 0; i < b64.length; i += 4) {
      var a = _b64l[b64[i]]||0, b = _b64l[b64[i+1]]||0, c = _b64l[b64[i+2]]||0, d = _b64l[b64[i+3]]||0;
      bytes[p++] = (a<<2)|(b>>4);
      if (b64[i+2] !== "=") bytes[p++] = ((b<<4)|(c>>2))&0xff;
      if (b64[i+3] !== "=") bytes[p++] = ((c<<6)|d)&0xff;
    }
    return bytes;
  }

  function bytesOf(d) {
    if (d instanceof Uint8Array) return d;
    if (d && d._isBuffer) return d;
    if (typeof d === "string") return new TextEncoder().encode(d);
    if (d && d.buffer) return new Uint8Array(d.buffer, d.byteOffset, d.byteLength);
    return new Uint8Array(0);
  }

  function toBuf(bytes) {
    return typeof globalThis.Buffer !== "undefined" ? globalThis.Buffer.from(bytes) : bytes;
  }

  // Callback-style wrappers (Node.js convention: zlib.inflate(buf, cb))
  globalThis.zlib = {
    inflate: function(buf, cb) {
      try {
        var result = fromB64(__go_zlib_inflate(toB64(bytesOf(buf))));
        if (typeof cb === "function") cb(null, toBuf(result));
      } catch(e) {
        if (typeof cb === "function") cb(e);
      }
    },
    deflate: function(buf, optsOrCb, cb) {
      if (typeof optsOrCb === "function") { cb = optsOrCb; optsOrCb = {}; }
      var level = (optsOrCb && optsOrCb.level !== undefined) ? optsOrCb.level : -1;
      try {
        var result = fromB64(__go_zlib_deflate(toB64(bytesOf(buf)), level));
        if (typeof cb === "function") cb(null, toBuf(result));
      } catch(e) {
        if (typeof cb === "function") cb(e);
      }
    },
    gunzip: function(buf, cb) {
      try {
        var result = fromB64(__go_gzip_decompress(toB64(bytesOf(buf))));
        if (typeof cb === "function") cb(null, toBuf(result));
      } catch(e) {
        if (typeof cb === "function") cb(e);
      }
    },
    gzip: function(buf, cb) {
      try {
        var result = fromB64(__go_gzip_compress(toB64(bytesOf(buf))));
        if (typeof cb === "function") cb(null, toBuf(result));
      } catch(e) {
        if (typeof cb === "function") cb(e);
      }
    },
    inflateRaw: function(buf, cb) {
      try {
        var result = fromB64(__go_raw_inflate(toB64(bytesOf(buf))));
        if (typeof cb === "function") cb(null, toBuf(result));
      } catch(e) {
        if (typeof cb === "function") cb(e);
      }
    },
    deflateRaw: function(buf, optsOrCb, cb) {
      if (typeof optsOrCb === "function") { cb = optsOrCb; optsOrCb = {}; }
      var level = (optsOrCb && optsOrCb.level !== undefined) ? optsOrCb.level : -1;
      try {
        var result = fromB64(__go_raw_deflate(toB64(bytesOf(buf)), level));
        if (typeof cb === "function") cb(null, toBuf(result));
      } catch(e) {
        if (typeof cb === "function") cb(e);
      }
    },
    // Sync variants
    inflateSync: function(buf) { return toBuf(fromB64(__go_zlib_inflate(toB64(bytesOf(buf))))); },
    deflateSync: function(buf, opts) {
      var level = (opts && opts.level !== undefined) ? opts.level : -1;
      return toBuf(fromB64(__go_zlib_deflate(toB64(bytesOf(buf)), level)));
    },
    gunzipSync: function(buf) { return toBuf(fromB64(__go_gzip_decompress(toB64(bytesOf(buf))))); },
    gzipSync: function(buf) { return toBuf(fromB64(__go_gzip_compress(toB64(bytesOf(buf))))); },
    inflateRawSync: function(buf) { return toBuf(fromB64(__go_raw_inflate(toB64(bytesOf(buf))))); },
    deflateRawSync: function(buf, opts) {
      var level = (opts && opts.level !== undefined) ? opts.level : -1;
      return toBuf(fromB64(__go_raw_deflate(toB64(bytesOf(buf)), level)));
    },
    // Stream creators (return Transform-like objects)
    createGzip: function() {
      var S = globalThis.stream;
      if (!S) throw new Error("createGzip: streams not available");
      return new S.Transform({
        transform: function(chunk, enc, cb) {
          try {
            var result = fromB64(__go_gzip_compress(toB64(bytesOf(chunk))));
            this.push(toBuf(result));
            cb();
          } catch(e) { cb(e); }
        }
      });
    },
    createGunzip: function() {
      var S = globalThis.stream;
      if (!S) throw new Error("createGunzip: streams not available");
      return new S.Transform({
        transform: function(chunk, enc, cb) {
          try {
            var result = fromB64(__go_gzip_decompress(toB64(bytesOf(chunk))));
            this.push(toBuf(result));
            cb();
          } catch(e) { cb(e); }
        }
      });
    },
    createDeflate: function(opts) {
      var S = globalThis.stream;
      if (!S) throw new Error("createDeflate: streams not available");
      var level = (opts && opts.level !== undefined) ? opts.level : -1;
      return new S.Transform({
        transform: function(chunk, enc, cb) {
          try {
            var result = fromB64(__go_zlib_deflate(toB64(bytesOf(chunk)), level));
            this.push(toBuf(result));
            cb();
          } catch(e) { cb(e); }
        }
      });
    },
    createInflate: function() {
      var S = globalThis.stream;
      if (!S) throw new Error("createInflate: streams not available");
      return new S.Transform({
        transform: function(chunk, enc, cb) {
          try {
            var result = fromB64(__go_zlib_inflate(toB64(bytesOf(chunk))));
            this.push(toBuf(result));
            cb();
          } catch(e) { cb(e); }
        }
      });
    },
    constants: {
      Z_NO_COMPRESSION: 0, Z_BEST_SPEED: 1, Z_BEST_COMPRESSION: 9,
      Z_DEFAULT_COMPRESSION: -1, Z_FILTERED: 1, Z_HUFFMAN_ONLY: 2,
      Z_RLE: 3, Z_FIXED: 4, Z_DEFAULT_STRATEGY: 0,
    },
  };
})();
`
