package jsbridge

import (
	"encoding/base64"

	quickjs "github.com/buke/quickjs-go"
)

// EncodingPolyfill provides TextEncoder, TextDecoder, btoa, and atob.
type EncodingPolyfill struct{}

// Encoding creates an encoding polyfill.
func Encoding() *EncodingPolyfill { return &EncodingPolyfill{} }

func (p *EncodingPolyfill) Name() string { return "encoding" }

func (p *EncodingPolyfill) Setup(ctx *quickjs.Context) error {
	ctx.Globals().Set("__go_text_encode", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		s := ""
		if len(args) > 0 {
			s = args[0].ToString()
		}
		return ctx.NewArrayBuffer([]byte(s))
	}))

	ctx.Globals().Set("__go_text_decode", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) > 0 {
			size := args[0].ByteLen()
			if size > 0 {
				b, err := args[0].ToByteArray(uint(size))
				if err != nil {
					return ctx.ThrowError(err)
				}
				return ctx.NewString(string(b))
			}
			return ctx.NewString("")
		}
		return ctx.NewString("")
	}))

	ctx.Globals().Set("__go_btoa", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		s := ""
		if len(args) > 0 {
			s = args[0].ToString()
		}
		return ctx.NewString(base64.StdEncoding.EncodeToString([]byte(s)))
	}))

	ctx.Globals().Set("__go_atob", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) > 0 {
			b, err := base64.StdEncoding.DecodeString(args[0].ToString())
			if err != nil {
				return ctx.ThrowError(err)
			}
			return ctx.NewString(string(b))
		}
		return ctx.NewString("")
	}))

	return evalJS(ctx, `
globalThis.TextEncoder = class TextEncoder {
  get encoding() { return 'utf-8'; }
  encode(s) { return new Uint8Array(__go_text_encode(String(s || ''))); }
};
globalThis.TextDecoder = class TextDecoder {
  constructor(enc) { this._enc = enc || 'utf-8'; }
  get encoding() { return this._enc; }
  decode(input) {
    if (!input) return '';
    return __go_text_decode(input.buffer || input);
  }
};
globalThis.btoa = (s) => __go_btoa(s);
globalThis.atob = (s) => __go_atob(s);
`)
}
