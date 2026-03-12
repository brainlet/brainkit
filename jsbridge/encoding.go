package jsbridge

import (
	"encoding/base64"

	"github.com/fastschema/qjs"
)

// EncodingPolyfill provides TextEncoder, TextDecoder, btoa, and atob.
type EncodingPolyfill struct{}

// Encoding creates an encoding polyfill.
func Encoding() *EncodingPolyfill { return &EncodingPolyfill{} }

func (p *EncodingPolyfill) Name() string { return "encoding" }

func (p *EncodingPolyfill) Setup(ctx *qjs.Context) error {
	ctx.SetFunc("__go_text_encode", func(this *qjs.This) (*qjs.Value, error) {
		s := ""
		if args := this.Args(); len(args) > 0 {
			s = args[0].String()
		}
		return this.Context().NewArrayBuffer([]byte(s)), nil
	})

	ctx.SetFunc("__go_text_decode", func(this *qjs.This) (*qjs.Value, error) {
		if args := this.Args(); len(args) > 0 {
			return this.Context().NewString(string(args[0].ToByteArray())), nil
		}
		return this.Context().NewString(""), nil
	})

	ctx.SetFunc("__go_btoa", func(this *qjs.This) (*qjs.Value, error) {
		s := ""
		if args := this.Args(); len(args) > 0 {
			s = args[0].String()
		}
		return this.Context().NewString(base64.StdEncoding.EncodeToString([]byte(s))), nil
	})

	ctx.SetFunc("__go_atob", func(this *qjs.This) (*qjs.Value, error) {
		if args := this.Args(); len(args) > 0 {
			b, err := base64.StdEncoding.DecodeString(args[0].String())
			if err != nil {
				return nil, err
			}
			return this.Context().NewString(string(b)), nil
		}
		return this.Context().NewString(""), nil
	})

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
