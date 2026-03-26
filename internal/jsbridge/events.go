package jsbridge

import quickjs "github.com/buke/quickjs-go"

// EventsPolyfill provides a Node.js-compatible EventEmitter (pure JS).
type EventsPolyfill struct{}

// Events creates an events polyfill.
func Events() *EventsPolyfill { return &EventsPolyfill{} }

func (p *EventsPolyfill) Name() string { return "events" }

func (p *EventsPolyfill) Setup(ctx *quickjs.Context) error {
	return evalJS(ctx, eventsJS)
}

const eventsJS = `
globalThis.EventEmitter = class EventEmitter {
  constructor() { this._e = {}; this._maxListeners = 0; }
  on(ev, fn) {
    (this._e[ev] = this._e[ev] || []).push(fn);
    return this;
  }
  addListener(ev, fn) { return this.on(ev, fn); }
  prependListener(ev, fn) {
    (this._e[ev] = this._e[ev] || []).unshift(fn);
    return this;
  }
  once(ev, fn) {
    const w = (...a) => { this.removeListener(ev, w); fn.apply(this, a); };
    w._orig = fn;
    return this.on(ev, w);
  }
  prependOnceListener(ev, fn) {
    const w = (...a) => { this.removeListener(ev, w); fn.apply(this, a); };
    w._orig = fn;
    return this.prependListener(ev, w);
  }
  emit(ev, ...a) {
    const ls = this._e[ev];
    if (!ls) return false;
    ls.slice().forEach(l => l.apply(this, a));
    return true;
  }
  removeListener(ev, fn) {
    const ls = this._e[ev];
    if (!ls) return this;
    this._e[ev] = ls.filter(l => l !== fn && l._orig !== fn);
    return this;
  }
  off(ev, fn) { return this.removeListener(ev, fn); }
  removeAllListeners(ev) {
    if (ev) delete this._e[ev]; else this._e = {};
    return this;
  }
  setMaxListeners(n) { this._maxListeners = n; return this; }
  getMaxListeners() { return this._maxListeners || 10; }
  listenerCount(ev) { return (this._e[ev] || []).length; }
  listeners(ev) { return (this._e[ev] || []).slice(); }
  rawListeners(ev) { return (this._e[ev] || []).slice(); }
  eventNames() { return Object.keys(this._e).filter(k => this._e[k] && this._e[k].length > 0); }
};
EventEmitter.captureRejections = false;
EventEmitter.defaultMaxListeners = 10;
EventEmitter.setMaxListeners = function() {};
EventEmitter.listenerCount = function(emitter, ev) { return emitter.listenerCount ? emitter.listenerCount(ev) : 0; };
`
