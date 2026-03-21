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
// TextEncoder/TextDecoder — pure JS implementation.
// Cannot use Go bridge for these because Go's ToString() truncates at null bytes,
// and both pg wire protocol and BSON use null bytes in strings extensively.
globalThis.TextEncoder = class TextEncoder {
  get encoding() { return 'utf-8'; }
  encode(s) {
    s = String(s || '');
    // Fast path: ASCII-only (common for pg protocol messages)
    var bytes = [];
    for (var i = 0; i < s.length; i++) {
      var c = s.charCodeAt(i);
      if (c < 0x80) {
        bytes.push(c);
      } else if (c < 0x800) {
        bytes.push(0xC0 | (c >> 6), 0x80 | (c & 0x3F));
      } else if (c >= 0xD800 && c < 0xDC00 && i + 1 < s.length) {
        // Surrogate pair
        var c2 = s.charCodeAt(++i);
        var cp = ((c - 0xD800) << 10) + (c2 - 0xDC00) + 0x10000;
        bytes.push(0xF0 | (cp >> 18), 0x80 | ((cp >> 12) & 0x3F), 0x80 | ((cp >> 6) & 0x3F), 0x80 | (cp & 0x3F));
      } else {
        bytes.push(0xE0 | (c >> 12), 0x80 | ((c >> 6) & 0x3F), 0x80 | (c & 0x3F));
      }
    }
    return new Uint8Array(bytes);
  }
  encodeInto(s, dest) {
    var encoded = this.encode(s);
    var len = Math.min(encoded.length, dest.length);
    for (var i = 0; i < len; i++) dest[i] = encoded[i];
    return { read: s.length, written: len };
  }
};
globalThis.TextDecoder = class TextDecoder {
  constructor(enc, opts) { this._enc = enc || 'utf-8'; this._fatal = opts && opts.fatal; }
  get encoding() { return this._enc; }
  decode(input) {
    if (!input) return '';
    // Get the actual bytes from the input
    var bytes;
    if (input instanceof Uint8Array) {
      bytes = input;
    } else if (ArrayBuffer.isView(input)) {
      bytes = new Uint8Array(input.buffer, input.byteOffset, input.byteLength);
    } else if (input instanceof ArrayBuffer) {
      bytes = new Uint8Array(input);
    } else {
      return '';
    }
    // Pure JS UTF-8 decode — handles null bytes correctly
    var result = '';
    for (var i = 0; i < bytes.length; ) {
      var b = bytes[i];
      if (b < 0x80) {
        result += String.fromCharCode(b);
        i++;
      } else if ((b & 0xE0) === 0xC0) {
        result += String.fromCharCode(((b & 0x1F) << 6) | (bytes[i+1] & 0x3F));
        i += 2;
      } else if ((b & 0xF0) === 0xE0) {
        result += String.fromCharCode(((b & 0x0F) << 12) | ((bytes[i+1] & 0x3F) << 6) | (bytes[i+2] & 0x3F));
        i += 3;
      } else if ((b & 0xF8) === 0xF0) {
        var cp = ((b & 0x07) << 18) | ((bytes[i+1] & 0x3F) << 12) | ((bytes[i+2] & 0x3F) << 6) | (bytes[i+3] & 0x3F);
        cp -= 0x10000;
        result += String.fromCharCode(0xD800 + (cp >> 10), 0xDC00 + (cp & 0x3FF));
        i += 4;
      } else {
        result += '\uFFFD';
        i++;
      }
    }
    return result;
  }
};
globalThis.btoa = (s) => __go_btoa(s);
globalThis.atob = (s) => __go_atob(s);
`)
}
