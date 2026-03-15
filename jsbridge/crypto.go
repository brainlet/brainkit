package jsbridge

import (
	"crypto/hmac"
	"crypto/md5"
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

	return evalJS(ctx, `
globalThis.crypto = globalThis.crypto || {};
globalThis.crypto.randomUUID = () => __go_crypto_randomUUID();

// SubtleCrypto — required by pg's crypto/utils-webcrypto.js
// All methods return Promises (WebCrypto spec). Data is passed as base64.
(function() {
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
    var binary = '';
    for (var i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i]);
    return btoa(binary);
  }

  function fromBase64(b64) {
    var binary = atob(b64);
    var bytes = new Uint8Array(binary.length);
    for (var i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
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

globalThis.__node_crypto = {
  createHash: (alg) => {
    let _data = '';
    return {
      update(d) { _data += d; return this; },
      digest(enc) { return __go_crypto_hash(alg, _data); }
    };
  },
  createHmac: (alg, key) => {
    let _data = '';
    return {
      update(d) { _data += d; return this; },
      digest(enc) { return __go_crypto_hmac(alg, key, _data); }
    };
  },
  webcrypto: globalThis.crypto,
};
`)
}
