package jsbridge

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"

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

	return evalJS(ctx, `
globalThis.crypto = globalThis.crypto || {};
globalThis.crypto.randomUUID = () => __go_crypto_randomUUID();

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
  }
};
`)
}
