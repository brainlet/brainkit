package jsbridge

import "github.com/fastschema/qjs"

// EventsPolyfill provides a Node.js-compatible EventEmitter (pure JS).
type EventsPolyfill struct{}

// Events creates an events polyfill.
func Events() *EventsPolyfill { return &EventsPolyfill{} }

func (p *EventsPolyfill) Name() string { return "events" }

func (p *EventsPolyfill) Setup(ctx *qjs.Context) error {
	return evalJS(ctx, eventsJS)
}

const eventsJS = `
globalThis.EventEmitter = class EventEmitter {
  constructor() { this._e = {}; }
  on(ev, fn) {
    (this._e[ev] = this._e[ev] || []).push(fn);
    return this;
  }
  once(ev, fn) {
    const w = (...a) => { this.removeListener(ev, w); fn.apply(this, a); };
    w._orig = fn;
    return this.on(ev, w);
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
  removeAllListeners(ev) {
    if (ev) delete this._e[ev]; else this._e = {};
    return this;
  }
  listenerCount(ev) { return (this._e[ev] || []).length; }
  listeners(ev) { return (this._e[ev] || []).slice(); }
};
`
