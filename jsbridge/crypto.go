package jsbridge

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"

	"github.com/google/uuid"
	"github.com/fastschema/qjs"
)

// CryptoPolyfill provides crypto.randomUUID and Node.js-style createHash/createHmac.
type CryptoPolyfill struct{}

// Crypto creates a crypto polyfill.
func Crypto() *CryptoPolyfill { return &CryptoPolyfill{} }

func (p *CryptoPolyfill) Name() string { return "crypto" }

func (p *CryptoPolyfill) Setup(ctx *qjs.Context) error {
	ctx.SetFunc("__go_crypto_randomUUID", func(this *qjs.This) (*qjs.Value, error) {
		return this.Context().NewString(uuid.NewString()), nil
	})

	ctx.SetFunc("__go_crypto_hash", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 2 {
			return nil, fmt.Errorf("crypto.hash: requires algorithm and data")
		}
		alg := args[0].String()
		data := args[1].String()

		var h hash.Hash
		switch alg {
		case "sha256":
			h = sha256.New()
		case "sha512":
			h = sha512.New()
		default:
			return nil, fmt.Errorf("crypto.hash: unsupported algorithm %q", alg)
		}
		h.Write([]byte(data))
		return this.Context().NewString(hex.EncodeToString(h.Sum(nil))), nil
	})

	ctx.SetFunc("__go_crypto_hmac", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 3 {
			return nil, fmt.Errorf("crypto.hmac: requires algorithm, key, and data")
		}
		alg := args[0].String()
		key := args[1].String()
		data := args[2].String()

		var hf func() hash.Hash
		switch alg {
		case "sha256":
			hf = sha256.New
		case "sha512":
			hf = sha512.New
		default:
			return nil, fmt.Errorf("crypto.hmac: unsupported algorithm %q", alg)
		}
		mac := hmac.New(hf, []byte(key))
		mac.Write([]byte(data))
		return this.Context().NewString(hex.EncodeToString(mac.Sum(nil))), nil
	})

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
