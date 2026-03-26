package jsbridge

import (
	"crypto/hmac"
	"crypto/md5"
	crand "crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"

	"golang.org/x/crypto/pbkdf2"

	"github.com/google/uuid"
	quickjs "github.com/buke/quickjs-go"
)

// CryptoPolyfill provides crypto.randomUUID and Node.js-style createHash/createHmac.
type CryptoPolyfill struct{}

// Crypto creates a crypto polyfill.
func Crypto() *CryptoPolyfill { return &CryptoPolyfill{} }

func (p *CryptoPolyfill) Name() string { return "crypto" }

func (p *CryptoPolyfill) Setup(ctx *quickjs.Context) error {
	ctx.Globals().Set("__go_crypto_randomUUID", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewString(uuid.NewString())
	}))

	ctx.Globals().Set("__go_crypto_hash", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return ctx.ThrowError(fmt.Errorf("crypto.hash: requires algorithm and data"))
		}
		alg := args[0].ToString()
		data := args[1].ToString()

		var h hash.Hash
		switch alg {
		case "md5":
			h = md5.New()
		case "sha1":
			h = sha1.New()
		case "sha256":
			h = sha256.New()
		case "sha512":
			h = sha512.New()
		default:
			return ctx.ThrowError(fmt.Errorf("crypto.hash: unsupported algorithm %q", alg))
		}
		h.Write([]byte(data))
		return ctx.NewString(hex.EncodeToString(h.Sum(nil)))
	}))

	ctx.Globals().Set("__go_crypto_hmac", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 3 {
			return ctx.ThrowError(fmt.Errorf("crypto.hmac: requires algorithm, key, and data"))
		}
		alg := args[0].ToString()
		key := args[1].ToString()
		data := args[2].ToString()

		var hf func() hash.Hash
		switch alg {
		case "sha256":
			hf = sha256.New
		case "sha512":
			hf = sha512.New
		default:
			return ctx.ThrowError(fmt.Errorf("crypto.hmac: unsupported algorithm %q", alg))
		}
		mac := hmac.New(hf, []byte(key))
		mac.Write([]byte(data))
		return ctx.NewString(hex.EncodeToString(mac.Sum(nil)))
	}))

	// __go_crypto_subtle_digest(algorithm, dataBase64) → resultBase64
	ctx.Globals().Set("__go_crypto_subtle_digest", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return ctx.ThrowError(fmt.Errorf("subtle.digest: requires algorithm and data"))
		}
		alg := args[0].ToString()
		dataB64 := args[1].ToString()

		data, err := base64.StdEncoding.DecodeString(dataB64)
		if err != nil {
			return ctx.ThrowError(fmt.Errorf("subtle.digest: invalid base64: %w", err))
		}

		var h hash.Hash
		switch alg {
		case "SHA-1":
			h = sha1.New()
		case "SHA-256":
			h = sha256.New()
		case "SHA-512":
			h = sha512.New()
		case "MD5":
			h = md5.New()
		default:
			return ctx.ThrowError(fmt.Errorf("subtle.digest: unsupported algorithm %q", alg))
		}
		h.Write(data)
		return ctx.NewString(base64.StdEncoding.EncodeToString(h.Sum(nil)))
	}))

	// __go_crypto_subtle_sign(algorithm, keyBase64, dataBase64) → resultBase64
	ctx.Globals().Set("__go_crypto_subtle_sign", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 3 {
			return ctx.ThrowError(fmt.Errorf("subtle.sign: requires algorithm, key, and data"))
		}
		alg := args[0].ToString()
		keyB64 := args[1].ToString()
		dataB64 := args[2].ToString()

		key, err := base64.StdEncoding.DecodeString(keyB64)
		if err != nil {
			return ctx.ThrowError(fmt.Errorf("subtle.sign: invalid key base64: %w", err))
		}
		data, err := base64.StdEncoding.DecodeString(dataB64)
		if err != nil {
			return ctx.ThrowError(fmt.Errorf("subtle.sign: invalid data base64: %w", err))
		}

		var hf func() hash.Hash
		switch alg {
		case "SHA-256":
			hf = sha256.New
		case "SHA-512":
			hf = sha512.New
		default:
			return ctx.ThrowError(fmt.Errorf("subtle.sign: unsupported hash %q", alg))
		}
		mac := hmac.New(hf, key)
		mac.Write(data)
		return ctx.NewString(base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	}))

	// __go_crypto_subtle_deriveBits(passwordBase64, saltBase64, iterations, bitLength, hashAlg) → resultBase64
	ctx.Globals().Set("__go_crypto_subtle_deriveBits", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 5 {
			return ctx.ThrowError(fmt.Errorf("subtle.deriveBits: requires password, salt, iterations, bitLength, hash"))
		}
		passwordB64 := args[0].ToString()
		saltB64 := args[1].ToString()
		iterations := int(args[2].ToInt64())
		bitLength := int(args[3].ToInt64())
		hashAlg := args[4].ToString()

		password, err := base64.StdEncoding.DecodeString(passwordB64)
		if err != nil {
			return ctx.ThrowError(fmt.Errorf("subtle.deriveBits: invalid password base64: %w", err))
		}
		salt, err := base64.StdEncoding.DecodeString(saltB64)
		if err != nil {
			return ctx.ThrowError(fmt.Errorf("subtle.deriveBits: invalid salt base64: %w", err))
		}

		var hf func() hash.Hash
		switch hashAlg {
		case "SHA-256":
			hf = sha256.New
		case "SHA-512":
			hf = sha512.New
		default:
			return ctx.ThrowError(fmt.Errorf("subtle.deriveBits: unsupported hash %q", hashAlg))
		}

		keyLen := bitLength / 8
		derived := pbkdf2.Key(password, salt, iterations, keyLen, hf)
		return ctx.NewString(base64.StdEncoding.EncodeToString(derived))
	}))

	// __go_crypto_getRandomValues fills a buffer with random bytes and returns base64
	ctx.Globals().Set("__go_crypto_getRandomValues", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("getRandomValues: requires size"))
		}
		size := int(args[0].ToInt64())
		if size <= 0 {
			return ctx.NewString("")
		}
		buf := make([]byte, size)
		if _, err := crand.Read(buf); err != nil {
			return ctx.ThrowError(fmt.Errorf("getRandomValues: %w", err))
		}
		return ctx.NewString(base64.StdEncoding.EncodeToString(buf))
	}))

	return evalJS(ctx, `
globalThis.crypto = globalThis.crypto || {};
globalThis.crypto.randomUUID = () => __go_crypto_randomUUID();
globalThis.crypto.getRandomValues = function(arr) {
  var b64 = __go_crypto_getRandomValues(arr.length);
  // Decode base64 to fill the typed array
  var chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
  var lookup = {};
  for (var i = 0; i < chars.length; i++) lookup[chars[i]] = i;
  var p = 0;
  for (var i = 0; i < b64.length && p < arr.length; i += 4) {
    var a = lookup[b64[i]] || 0;
    var b = lookup[b64[i+1]] || 0;
    var c = lookup[b64[i+2]] || 0;
    var d = lookup[b64[i+3]] || 0;
    if (p < arr.length) arr[p++] = (a << 2) | (b >> 4);
    if (p < arr.length && b64[i+2] !== "=") arr[p++] = ((b << 4) | (c >> 2)) & 0xff;
    if (p < arr.length && b64[i+3] !== "=") arr[p++] = ((c << 6) | d) & 0xff;
  }
  return arr;
};

// SubtleCrypto — required by pg's crypto/utils-webcrypto.js
// All methods return Promises (WebCrypto spec). Data is passed as base64.
(function() {
  // Pure-JS base64 — cannot use btoa/atob because they go through Go's ToString()
  // which truncates at null bytes. Crypto data (HMAC, PBKDF2) contains null bytes.
  var _b64chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
  var _b64lookup = {};
  for (var _bi = 0; _bi < _b64chars.length; _bi++) _b64lookup[_b64chars[_bi]] = _bi;

  function toBase64(data) {
    var bytes;
    if (typeof data === 'string') {
      bytes = new TextEncoder().encode(data);
    } else if (data instanceof ArrayBuffer) {
      bytes = new Uint8Array(data);
    } else if (data instanceof Uint8Array) {
      bytes = data;
    } else if (data && data.buffer instanceof ArrayBuffer) {
      bytes = new Uint8Array(data.buffer, data.byteOffset, data.byteLength);
    } else {
      bytes = new Uint8Array(0);
    }
    var b64 = '';
    for (var i = 0; i < bytes.length; i += 3) {
      var b0 = bytes[i];
      var b1 = i + 1 < bytes.length ? bytes[i + 1] : 0;
      var b2 = i + 2 < bytes.length ? bytes[i + 2] : 0;
      b64 += _b64chars[(b0 >> 2) & 0x3f];
      b64 += _b64chars[((b0 << 4) | (b1 >> 4)) & 0x3f];
      b64 += (i + 1 < bytes.length) ? _b64chars[((b1 << 2) | (b2 >> 6)) & 0x3f] : '=';
      b64 += (i + 2 < bytes.length) ? _b64chars[b2 & 0x3f] : '=';
    }
    return b64;
  }

  function fromBase64(b64) {
    var bufLen = Math.floor(b64.length * 3 / 4);
    if (b64.length > 1 && b64[b64.length - 1] === '=') bufLen--;
    if (b64.length > 2 && b64[b64.length - 2] === '=') bufLen--;
    var bytes = new Uint8Array(bufLen);
    var p = 0;
    for (var i = 0; i < b64.length; i += 4) {
      var a = _b64lookup[b64[i]] || 0;
      var b = _b64lookup[b64[i+1]] || 0;
      var c = _b64lookup[b64[i+2]] || 0;
      var d = _b64lookup[b64[i+3]] || 0;
      bytes[p++] = (a << 2) | (b >> 4);
      if (b64[i+2] !== '=') bytes[p++] = ((b << 4) | (c >> 2)) & 0xff;
      if (b64[i+3] !== '=') bytes[p++] = ((c << 6) | d) & 0xff;
    }
    return bytes.buffer;
  }

  globalThis.crypto.subtle = {
    digest: function(algorithm, data) {
      var alg = typeof algorithm === 'string' ? algorithm : algorithm.name;
      var b64 = __go_crypto_subtle_digest(alg, toBase64(data));
      return Promise.resolve(fromBase64(b64));
    },

    importKey: function(format, keyData, algorithm, extractable, keyUsages) {
      // Return a CryptoKey-like object that stores the raw key data
      var alg = typeof algorithm === 'string' ? algorithm : algorithm.name;
      var hash = (algorithm && algorithm.hash) ? (typeof algorithm.hash === 'string' ? algorithm.hash : algorithm.hash.name) : 'SHA-256';
      return Promise.resolve({
        type: 'secret',
        algorithm: { name: alg, hash: hash },
        extractable: !!extractable,
        usages: keyUsages || [],
        _rawKey: toBase64(keyData),
      });
    },

    sign: function(algorithm, key, data) {
      var alg = typeof algorithm === 'string' ? algorithm : algorithm.name;
      var hash = key.algorithm && key.algorithm.hash ? key.algorithm.hash : 'SHA-256';
      if (typeof hash === 'object') hash = hash.name || 'SHA-256';
      var b64 = __go_crypto_subtle_sign(hash, key._rawKey, toBase64(data));
      return Promise.resolve(fromBase64(b64));
    },

    deriveBits: function(algorithm, baseKey, length) {
      var alg = typeof algorithm === 'string' ? algorithm : algorithm.name;
      var hash = algorithm.hash ? (typeof algorithm.hash === 'string' ? algorithm.hash : algorithm.hash.name) : 'SHA-256';
      var saltB64 = toBase64(algorithm.salt);
      var iterations = algorithm.iterations || 4096;
      var b64 = __go_crypto_subtle_deriveBits(baseKey._rawKey, saltB64, iterations, length, hash);
      return Promise.resolve(fromBase64(b64));
    },

    exportKey: function(format, key) {
      if (format === 'raw' && key._rawKey) {
        return Promise.resolve(fromBase64(key._rawKey));
      }
      return Promise.reject(new Error('subtle.exportKey: unsupported format'));
    },

    generateKey: function() {
      return Promise.reject(new Error('subtle.generateKey: not implemented'));
    },

    verify: function() {
      return Promise.reject(new Error('subtle.verify: not implemented'));
    },

    encrypt: function() {
      return Promise.reject(new Error('subtle.encrypt: not implemented'));
    },

    decrypt: function() {
      return Promise.reject(new Error('subtle.decrypt: not implemented'));
    },
  };
})();

// Node.js crypto methods merged onto globalThis.crypto alongside WebCrypto.
// MongoDB SCRAM auth passes Uint8Array keys and data. We accumulate bytes
// and encode to base64 before calling Go bridges to avoid null-byte truncation.
(function() {
  var _b64c = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
  var _b64l = {}; for (var _i = 0; _i < _b64c.length; _i++) _b64l[_b64c[_i]] = _i;

  function bytesOf(d) {
    if (d instanceof Uint8Array) return d;
    if (d && d._isBuffer) return d;
    if (typeof d === "string") {
      // Manual UTF-8 encode — works even without TextEncoder polyfill
      var bytes = [];
      for (var i = 0; i < d.length; i++) {
        var c = d.charCodeAt(i);
        if (c < 0x80) bytes.push(c);
        else if (c < 0x800) { bytes.push(0xC0|(c>>6), 0x80|(c&0x3F)); }
        else { bytes.push(0xE0|(c>>12), 0x80|((c>>6)&0x3F), 0x80|(c&0x3F)); }
      }
      return new Uint8Array(bytes);
    }
    if (d && d.buffer) return new Uint8Array(d.buffer, d.byteOffset, d.byteLength);
    return new Uint8Array(0);
  }

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

  function concatBytes(a, b) {
    var r = new Uint8Array(a.length + b.length);
    r.set(a, 0); r.set(b, a.length);
    return r;
  }

  // Merge Node.js crypto methods onto globalThis.crypto (which already has WebCrypto: subtle, randomUUID, getRandomValues).
  // This matches Node.js behavior where require('crypto') returns one object with both WebCrypto and Node.js APIs.
  var _cryptoTarget = globalThis.crypto || {};
  Object.assign(_cryptoTarget, {
    createHash: function(alg) {
      var _bytes = new Uint8Array(0);
      return {
        update: function(d, enc) { _bytes = concatBytes(_bytes, bytesOf(d)); return this; },
        copy: function() {
          var c = _cryptoTarget.createHash(alg);
          c._bytes = _bytes.slice(); return c;
        },
        digest: function(enc) {
          var resultB64 = __go_crypto_subtle_digest(
            alg === "md5" ? "MD5" : alg === "sha1" ? "SHA-1" : alg === "sha256" ? "SHA-256" : alg === "sha512" ? "SHA-512" : alg.toUpperCase(),
            toB64(_bytes)
          );
          if (enc === "hex") {
            var raw = fromB64(resultB64);
            var hex = ""; for (var i = 0; i < raw.length; i++) hex += (raw[i] < 16 ? "0" : "") + raw[i].toString(16);
            return hex;
          }
          return fromB64(resultB64);
        }
      };
    },
    createHmac: function(alg, key) {
      var _keyB64 = toB64(bytesOf(key));
      var _bytes = new Uint8Array(0);
      var hashName = alg === "sha1" ? "SHA-1" : alg === "sha256" ? "SHA-256" : alg === "sha512" ? "SHA-512" : alg.toUpperCase();
      return {
        update: function(d, enc) { _bytes = concatBytes(_bytes, bytesOf(d)); return this; },
        digest: function(enc) {
          var resultB64 = __go_crypto_subtle_sign(hashName, _keyB64, toB64(_bytes));
          if (enc === "hex") {
            var raw = fromB64(resultB64);
            var hex = ""; for (var i = 0; i < raw.length; i++) hex += (raw[i] < 16 ? "0" : "") + raw[i].toString(16);
            return hex;
          }
          return fromB64(resultB64);
        }
      };
    },
    // pbkdf2Sync — synchronous PBKDF2 key derivation (MongoDB SCRAM needs this)
    pbkdf2Sync: function(password, salt, iterations, keylen, hash) {
      var pwB64 = toB64(bytesOf(password));
      var saltB64 = toB64(bytesOf(salt));
      var hashName = hash === "sha1" ? "SHA-1" : hash === "sha256" ? "SHA-256" : hash === "sha512" ? "SHA-512" : hash.toUpperCase();
      var resultB64 = __go_crypto_subtle_deriveBits(pwB64, saltB64, iterations, keylen * 8, hashName);
      var raw = fromB64(resultB64);
      // Return a Buffer if available, otherwise Uint8Array
      if (typeof globalThis.Buffer !== "undefined" && globalThis.Buffer.from) {
        return globalThis.Buffer.from(raw);
      }
      return raw;
    },

    // pbkdf2 — async callback form
    pbkdf2: function(password, salt, iterations, keylen, hash, callback) {
      try {
        var result = _cryptoTarget.pbkdf2Sync(password, salt, iterations, keylen, hash);
        if (typeof callback === "function") callback(null, result);
      } catch(e) {
        if (typeof callback === "function") callback(e);
      }
    },

    // randomBytes — sync + Node.js callback form
    randomBytes: function(n, cb) {
      var b = new Uint8Array(n);
      if (globalThis.crypto && globalThis.crypto.getRandomValues) {
        globalThis.crypto.getRandomValues(b);
      }
      // Wrap as Buffer if available
      if (typeof globalThis.Buffer !== "undefined" && globalThis.Buffer.from) {
        b = globalThis.Buffer.from(b);
      }
      // Support Node.js callback form: crypto.randomBytes(size, (err, buf) => {...})
      if (typeof cb === "function") { cb(null, b); }
      return b;
    },

    // randomFillSync
    randomFillSync: function(buf) {
      if (globalThis.crypto && globalThis.crypto.getRandomValues) {
        globalThis.crypto.getRandomValues(buf);
      }
      return buf;
    },

    // randomInt
    randomInt: function(min, max) {
      if (max === undefined) { max = min; min = 0; }
      return min + Math.floor(Math.random() * (max - min));
    },

    // timingSafeEqual — constant-time comparison
    timingSafeEqual: function(a, b) {
      if (a.length !== b.length) return false;
      var r = 0;
      for (var i = 0; i < a.length; i++) r |= a[i] ^ b[i];
      return r === 0;
    },

    // getHashes / getCiphers / getFips — feature detection
    getHashes: function() { return ["md5", "sha1", "sha256", "sha512"]; },
    getCiphers: function() { return []; },
    getFips: function() { return 0; },
  });
  if (!globalThis.crypto) globalThis.crypto = _cryptoTarget;
})();
`)
}
