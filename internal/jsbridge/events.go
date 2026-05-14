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
function EventEmitter() {
  if (this === undefined || this === null || this === globalThis) return new EventEmitter();
  this._e = {};
  this._maxListeners = 0;
}
EventEmitter.prototype.on = function(ev, fn) {
    (this._e[ev] = this._e[ev] || []).push(fn);
    return this;
};
EventEmitter.prototype.addListener = function(ev, fn) { return this.on(ev, fn); };
EventEmitter.prototype.prependListener = function(ev, fn) {
    (this._e[ev] = this._e[ev] || []).unshift(fn);
    return this;
};
EventEmitter.prototype.once = function(ev, fn) {
    const w = (...a) => { this.removeListener(ev, w); fn.apply(this, a); };
    w._orig = fn;
    return this.on(ev, w);
};
EventEmitter.prototype.prependOnceListener = function(ev, fn) {
    const w = (...a) => { this.removeListener(ev, w); fn.apply(this, a); };
    w._orig = fn;
    return this.prependListener(ev, w);
};
EventEmitter.prototype.emit = function(ev, ...a) {
    const ls = this._e[ev];
    if (!ls) return false;
    ls.slice().forEach(l => l.apply(this, a));
    return true;
};
EventEmitter.prototype.removeListener = function(ev, fn) {
    const ls = this._e[ev];
    if (!ls) return this;
    this._e[ev] = ls.filter(l => l !== fn && l._orig !== fn);
    return this;
};
EventEmitter.prototype.off = function(ev, fn) { return this.removeListener(ev, fn); };
EventEmitter.prototype.removeAllListeners = function(ev) {
    if (ev) delete this._e[ev]; else this._e = {};
    return this;
};
EventEmitter.prototype.setMaxListeners = function(n) { this._maxListeners = n; return this; };
EventEmitter.prototype.getMaxListeners = function() { return this._maxListeners || 10; };
EventEmitter.prototype.listenerCount = function(ev) { return (this._e[ev] || []).length; };
EventEmitter.prototype.listeners = function(ev) { return (this._e[ev] || []).slice(); };
EventEmitter.prototype.rawListeners = function(ev) { return (this._e[ev] || []).slice(); };
EventEmitter.prototype.eventNames = function() { return Object.keys(this._e).filter(k => this._e[k] && this._e[k].length > 0); };
EventEmitter.captureRejections = false;
EventEmitter.defaultMaxListeners = 10;
EventEmitter.setMaxListeners = function() {};
EventEmitter.listenerCount = function(emitter, ev) { return emitter.listenerCount ? emitter.listenerCount(ev) : 0; };
globalThis.EventEmitter = EventEmitter;
`
