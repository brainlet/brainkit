package jsbridge

import "github.com/fastschema/qjs"

// AbortPolyfill provides AbortController and AbortSignal (pure JS).
type AbortPolyfill struct{}

// Abort creates an abort polyfill.
func Abort() *AbortPolyfill { return &AbortPolyfill{} }

func (p *AbortPolyfill) Name() string { return "abort" }

func (p *AbortPolyfill) Setup(ctx *qjs.Context) error {
	return evalJS(ctx, abortJS)
}

const abortJS = `
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
    this.reason = reason || new Error('AbortError');
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
    setTimeout(() => s._abort(new Error('TimeoutError')), ms);
    return s;
  }
};

globalThis.AbortController = class AbortController {
  constructor() { this.signal = new AbortSignal(); }
  abort(reason) { this.signal._abort(reason); }
};
`
