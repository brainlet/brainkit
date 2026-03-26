package jsbridge

import quickjs "github.com/buke/quickjs-go"

// BufferPolyfill provides Node.js Buffer on globalThis.
// Buffer extends Uint8Array with read/write methods for binary protocols
// (pg wire protocol uses BE, MongoDB BSON uses LE), base64/hex encoding,
// and the _isBuffer flag for instanceof checks.
//
// Previously inlined in agent/embed.go's runtimeGlobalsJS (~300 lines).
// Now a proper jsbridge polyfill with Go test coverage.
//
// IMPORTANT: Must be loaded AFTER Encoding polyfill (uses TextEncoder/TextDecoder).
type BufferPolyfill struct{}

// Buffer creates a Node.js Buffer polyfill.
func Buffer() *BufferPolyfill { return &BufferPolyfill{} }

func (p *BufferPolyfill) Name() string { return "buffer" }

func (p *BufferPolyfill) Setup(ctx *quickjs.Context) error {
	return evalJS(ctx, bufferJS)
}

const bufferJS = `
(function() {
  "use strict";
  if (typeof globalThis.Buffer !== "undefined") return; // already loaded

  var _te = new TextEncoder();
  var _td = new TextDecoder();

  // Pure-JS base64 — cannot use btoa/atob because Go's ToString() truncates at null bytes.
  var _b64c = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
  var _b64l = {};
  for (var _i = 0; _i < _b64c.length; _i++) _b64l[_b64c[_i]] = _i;

  function _b64decode(v) {
    var bLen = Math.floor(v.length * 3 / 4);
    if (v.length > 1 && v[v.length - 1] === "=") bLen--;
    if (v.length > 2 && v[v.length - 2] === "=") bLen--;
    var arr = new Uint8Array(bLen);
    var p = 0;
    for (var ci = 0; ci < v.length; ci += 4) {
      var a0 = _b64l[v[ci]] || 0, b0 = _b64l[v[ci+1]] || 0, c0 = _b64l[v[ci+2]] || 0, d0 = _b64l[v[ci+3]] || 0;
      arr[p++] = (a0 << 2) | (b0 >> 4);
      if (v[ci+2] !== "=") arr[p++] = ((b0 << 4) | (c0 >> 2)) & 0xff;
      if (v[ci+3] !== "=") arr[p++] = ((c0 << 6) | d0) & 0xff;
    }
    return arr;
  }

  function _b64encode(bytes) {
    var b64 = "";
    for (var i = 0; i < bytes.length; i += 3) {
      var a0 = bytes[i], a1 = i+1 < bytes.length ? bytes[i+1] : 0, a2 = i+2 < bytes.length ? bytes[i+2] : 0;
      b64 += _b64c[(a0>>2)&0x3f] + _b64c[((a0<<4)|(a1>>4))&0x3f];
      b64 += (i+1 < bytes.length) ? _b64c[((a1<<2)|(a2>>6))&0x3f] : "=";
      b64 += (i+2 < bytes.length) ? _b64c[a2&0x3f] : "=";
    }
    return b64;
  }

  function _addBufferMethods(buf) {
    if (buf._isBuffer) return buf;
    buf._isBuffer = true;

    buf.write = function(string, offset, lengthOrEnc, encoding) {
      if (typeof offset === "string") { encoding = offset; offset = 0; }
      if (typeof lengthOrEnc === "string") { encoding = lengthOrEnc; lengthOrEnc = undefined; }
      offset = offset || 0;
      var bytes = _te.encode(string);
      var len = lengthOrEnc !== undefined ? Math.min(bytes.length, lengthOrEnc) : bytes.length;
      for (var i = 0; i < len && (offset + i) < buf.length; i++) buf[offset + i] = bytes[i];
      return len;
    };

    buf.copy = function(target, targetStart, sourceStart, sourceEnd) {
      targetStart = targetStart || 0;
      sourceStart = sourceStart || 0;
      sourceEnd = sourceEnd || buf.length;
      for (var i = sourceStart; i < sourceEnd && (targetStart + i - sourceStart) < target.length; i++) {
        target[targetStart + i - sourceStart] = buf[i];
      }
      return sourceEnd - sourceStart;
    };

    // ── Big-endian (pg wire protocol) ──
    buf.writeInt32BE = function(value, offset) {
      offset = offset || 0;
      buf[offset]     = (value >>> 24) & 0xff;
      buf[offset + 1] = (value >>> 16) & 0xff;
      buf[offset + 2] = (value >>> 8) & 0xff;
      buf[offset + 3] = value & 0xff;
      return offset + 4;
    };
    buf.writeUInt32BE = buf.writeInt32BE;

    buf.writeInt16BE = function(value, offset) {
      offset = offset || 0;
      buf[offset]     = (value >>> 8) & 0xff;
      buf[offset + 1] = value & 0xff;
      return offset + 2;
    };
    buf.writeUInt16BE = buf.writeInt16BE;

    buf.readInt32BE = function(offset) {
      offset = offset || 0;
      return (buf[offset] << 24) | (buf[offset+1] << 16) | (buf[offset+2] << 8) | buf[offset+3];
    };
    buf.readUInt32BE = function(offset) {
      offset = offset || 0;
      return ((buf[offset] << 24) | (buf[offset+1] << 16) | (buf[offset+2] << 8) | buf[offset+3]) >>> 0;
    };
    buf.readInt16BE = function(offset) {
      offset = offset || 0;
      var val = (buf[offset] << 8) | buf[offset+1];
      return val > 0x7FFF ? val - 0x10000 : val;
    };
    buf.readUInt16BE = function(offset) {
      offset = offset || 0;
      return (buf[offset] << 8) | buf[offset+1];
    };

    // ── Little-endian (MongoDB BSON) ──
    buf.readInt32LE = function(offset) {
      offset = offset || 0;
      return buf[offset] | (buf[offset+1] << 8) | (buf[offset+2] << 16) | (buf[offset+3] << 24);
    };
    buf.readUInt32LE = function(offset) {
      offset = offset || 0;
      return (buf[offset] | (buf[offset+1] << 8) | (buf[offset+2] << 16) | (buf[offset+3] << 24)) >>> 0;
    };
    buf.readInt16LE = function(offset) {
      offset = offset || 0;
      var val = buf[offset] | (buf[offset+1] << 8);
      return val > 0x7FFF ? val - 0x10000 : val;
    };
    buf.readUInt16LE = function(offset) {
      offset = offset || 0;
      return buf[offset] | (buf[offset+1] << 8);
    };
    buf.writeInt32LE = function(value, offset) {
      offset = offset || 0;
      buf[offset]     = value & 0xff;
      buf[offset + 1] = (value >>> 8) & 0xff;
      buf[offset + 2] = (value >>> 16) & 0xff;
      buf[offset + 3] = (value >>> 24) & 0xff;
      return offset + 4;
    };
    buf.writeUInt32LE = buf.writeInt32LE;
    buf.writeInt16LE = function(value, offset) {
      offset = offset || 0;
      buf[offset]     = value & 0xff;
      buf[offset + 1] = (value >>> 8) & 0xff;
      return offset + 2;
    };
    buf.writeUInt16LE = buf.writeInt16LE;

    // ── Float/Double (LE) ──
    buf.readFloatLE = function(offset) {
      offset = offset || 0;
      var tmp = new Uint8Array(4);
      for (var i = 0; i < 4; i++) tmp[i] = buf[offset + i];
      return new DataView(tmp.buffer).getFloat32(0, true);
    };
    buf.readDoubleLE = function(offset) {
      offset = offset || 0;
      var tmp = new Uint8Array(8);
      for (var i = 0; i < 8; i++) tmp[i] = buf[offset + i];
      return new DataView(tmp.buffer).getFloat64(0, true);
    };
    buf.writeFloatLE = function(value, offset) {
      offset = offset || 0;
      var tmp = new Uint8Array(4);
      new DataView(tmp.buffer).setFloat32(0, value, true);
      for (var i = 0; i < 4; i++) buf[offset + i] = tmp[i];
      return offset + 4;
    };
    buf.writeDoubleLE = function(value, offset) {
      offset = offset || 0;
      var tmp = new Uint8Array(8);
      new DataView(tmp.buffer).setFloat64(0, value, true);
      for (var i = 0; i < 8; i++) buf[offset + i] = tmp[i];
      return offset + 8;
    };

    // ── Single byte ──
    buf.readUInt8 = function(offset) { return buf[offset || 0]; };
    buf.writeUInt8 = function(value, offset) { buf[offset || 0] = value & 0xff; return (offset || 0) + 1; };

    // ── Slice/subarray — return Buffers ──
    var origSlice = buf.slice.bind(buf);
    buf.slice = function(start, end) {
      return _addBufferMethods(origSlice(start, end));
    };
    buf.subarray = function(start, end) {
      return _addBufferMethods(Uint8Array.prototype.subarray.call(buf, start, end));
    };

    // ── toString with encoding ──
    buf.toString = function(encoding, start, end) {
      start = start || 0;
      end = end !== undefined ? end : buf.length;
      var sub = buf.subarray(start, end);
      encoding = (encoding || "utf8").toLowerCase();
      if (encoding === "utf8" || encoding === "utf-8") {
        return _td.decode(sub);
      }
      if (encoding === "hex") {
        var hex = "";
        for (var i = 0; i < sub.length; i++) hex += (sub[i] < 16 ? "0" : "") + sub[i].toString(16);
        return hex;
      }
      if (encoding === "base64") {
        return _b64encode(sub);
      }
      if (encoding === "ascii" || encoding === "latin1" || encoding === "binary") {
        var str = "";
        for (var i = 0; i < sub.length; i++) str += String.fromCharCode(sub[i]);
        return str;
      }
      return _td.decode(sub);
    };

    buf.toJSON = function() {
      return { type: "Buffer", data: Array.from(buf) };
    };

    buf.equals = function(other) {
      if (buf.length !== other.length) return false;
      for (var i = 0; i < buf.length; i++) if (buf[i] !== other[i]) return false;
      return true;
    };

    buf.compare = function(other) {
      var len = Math.min(buf.length, other.length);
      for (var i = 0; i < len; i++) {
        if (buf[i] < other[i]) return -1;
        if (buf[i] > other[i]) return 1;
      }
      return buf.length < other.length ? -1 : buf.length > other.length ? 1 : 0;
    };

    buf.fill = function(value, start, end) {
      start = start || 0;
      end = end || buf.length;
      var fillVal = typeof value === "number" ? value : 0;
      Uint8Array.prototype.fill.call(buf, fillVal, start, end);
      return buf;
    };

    buf.indexOf = function(val, byteOffset) {
      byteOffset = byteOffset || 0;
      if (typeof val === "number") {
        for (var i = byteOffset; i < buf.length; i++) if (buf[i] === val) return i;
        return -1;
      }
      return -1;
    };

    buf.map = function(fn) {
      return _addBufferMethods(Uint8Array.prototype.map.call(buf, fn));
    };

    return buf;
  }

  var _Buffer = {
    from: function(v, encOrOffset, length) {
      if (v instanceof ArrayBuffer) {
        var offset = encOrOffset || 0;
        var len = length !== undefined ? length : v.byteLength - offset;
        return _addBufferMethods(new Uint8Array(v, offset, len));
      }
      if (v instanceof Uint8Array || ArrayBuffer.isView(v)) {
        if (typeof encOrOffset === "number") {
          return _addBufferMethods(new Uint8Array(v.buffer, (v.byteOffset || 0) + encOrOffset, length));
        }
        return _addBufferMethods(new Uint8Array(v));
      }
      if (typeof v === "string") {
        var enc = encOrOffset;
        if (enc === "base64") {
          return _addBufferMethods(_b64decode(v));
        }
        if (enc === "hex") {
          var arr = new Uint8Array(v.length / 2);
          for (var i = 0; i < v.length; i += 2) arr[i / 2] = parseInt(v.substr(i, 2), 16);
          return _addBufferMethods(arr);
        }
        return _addBufferMethods(_te.encode(v));
      }
      if (Array.isArray(v)) {
        return _addBufferMethods(new Uint8Array(v));
      }
      if (typeof v === "number") {
        return _addBufferMethods(new Uint8Array(v));
      }
      return _addBufferMethods(new Uint8Array(0));
    },
    alloc: function(n, fill) {
      var b = new Uint8Array(n);
      if (fill !== undefined) b.fill(typeof fill === "number" ? fill : 0);
      return _addBufferMethods(b);
    },
    allocUnsafe: function(n) { return _addBufferMethods(new Uint8Array(n)); },
    allocUnsafeSlow: function(n) { return _addBufferMethods(new Uint8Array(n)); },
    isBuffer: function(obj) { return !!(obj && obj._isBuffer); },
    isEncoding: function(enc) {
      return ["utf8","utf-8","ascii","latin1","binary","hex","base64","ucs2","ucs-2","utf16le","utf-16le"]
        .indexOf((enc || "").toLowerCase()) !== -1;
    },
    byteLength: function(str, enc) {
      if (typeof str === "string") {
        if (enc === "base64") return Math.ceil(str.length * 3 / 4);
        return _te.encode(str).length;
      }
      if (str instanceof Uint8Array || str instanceof ArrayBuffer) return str.byteLength || str.length;
      return 0;
    },
    concat: function(bufs, totalLength) {
      if (!totalLength) {
        totalLength = 0;
        for (var i = 0; i < bufs.length; i++) totalLength += bufs[i].length;
      }
      var r = new Uint8Array(totalLength);
      var off = 0;
      for (var i = 0; i < bufs.length; i++) {
        r.set(bufs[i], off);
        off += bufs[i].length;
      }
      return _addBufferMethods(r);
    },
    compare: function(a, b) {
      var len = Math.min(a.length, b.length);
      for (var i = 0; i < len; i++) {
        if (a[i] < b[i]) return -1;
        if (a[i] > b[i]) return 1;
      }
      return a.length < b.length ? -1 : a.length > b.length ? 1 : 0;
    },
  };

  // Support "x instanceof Buffer" — pg uses this check
  Object.defineProperty(_Buffer, Symbol.hasInstance, {
    value: function(obj) { return !!(obj && obj._isBuffer); }
  });

  _Buffer.poolSize = 8192;

  globalThis.Buffer = _Buffer;
})();
`
