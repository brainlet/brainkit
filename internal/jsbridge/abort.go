package jsbridge

import quickjs "github.com/buke/quickjs-go"

// AbortPolyfill provides AbortController and AbortSignal (pure JS).
type AbortPolyfill struct{}

// Abort creates an abort polyfill.
func Abort() *AbortPolyfill { return &AbortPolyfill{} }

func (p *AbortPolyfill) Name() string { return "abort" }

func (p *AbortPolyfill) Setup(ctx *quickjs.Context) error {
	return evalJS(ctx, abortJS)
}

const abortJS = `
// DOMException — used by AbortSignal, fetch abort, Mastra tool suspension.
// Standard in browsers and Node.js, missing in QuickJS.
globalThis.DOMException = class DOMException extends Error {
  constructor(message, name) {
    super(message || '');
    this.name = name || 'DOMException';
    this.code = 0;
  }
};

globalThis.AbortSignal = class AbortSignal {
  constructor() {
    this.aborted = false;
    this.reason = undefined;
    this._listeners = [];
  }
  addEventListener(type, fn) {
    if (type === 'abort') this._listeners.push(fn);
  }
  removeEventListener(type, fn) {
    if (type === 'abort') this._listeners = this._listeners.filter(l => l !== fn);
  }
  _abort(reason) {
    if (this.aborted) return;
    this.aborted = true;
    this.reason = reason || new DOMException('The operation was aborted', 'AbortError');
    if (this.onabort) this.onabort({ type: 'abort', target: this });
    this._listeners.forEach(l => l({ type: 'abort', target: this }));
  }
  throwIfAborted() {
    if (this.aborted) throw this.reason;
  }
  static abort(reason) {
    const s = new AbortSignal();
    s._abort(reason);
    return s;
  }
  static timeout(ms) {
    const s = new AbortSignal();
    setTimeout(() => s._abort(new DOMException('The operation timed out', 'TimeoutError')), ms);
    return s;
  }
};

globalThis.AbortController = class AbortController {
  constructor() { this.signal = new AbortSignal(); }
  abort(reason) { this.signal._abort(reason); }
};
`
