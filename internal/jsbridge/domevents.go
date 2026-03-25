package jsbridge

import quickjs "github.com/buke/quickjs-go"

// DOMEventsPolyfill provides EventTarget, Event, and CustomEvent.
// AbortSignal inheritance and some SDK code depend on these DOM APIs.
// Separate from jsbridge/events.go (which provides Node.js EventEmitter).
type DOMEventsPolyfill struct{}

func DOMEvents() *DOMEventsPolyfill { return &DOMEventsPolyfill{} }

func (p *DOMEventsPolyfill) Name() string { return "domevents" }

func (p *DOMEventsPolyfill) Setup(ctx *quickjs.Context) error {
	return evalJS(ctx, `
if (typeof EventTarget === "undefined") {
  globalThis.EventTarget = class EventTarget {
    constructor() { this._listeners = {}; }
    addEventListener(type, fn) {
      (this._listeners[type] = this._listeners[type] || []).push(fn);
    }
    removeEventListener(type, fn) {
      var a = this._listeners[type];
      if (a) this._listeners[type] = a.filter(function(f) { return f !== fn; });
    }
    dispatchEvent(event) {
      var a = this._listeners[event.type];
      if (a) a.forEach(function(fn) { fn(event); });
      return true;
    }
  };
}
if (typeof Event === "undefined") {
  globalThis.Event = class Event {
    constructor(type, opts) {
      this.type = type;
      this.bubbles = !!(opts && opts.bubbles);
      this.cancelable = !!(opts && opts.cancelable);
      this.defaultPrevented = false;
      this.target = null;
      this.currentTarget = null;
      this.timeStamp = Date.now();
    }
    preventDefault() { this.defaultPrevented = true; }
    stopPropagation() {}
    stopImmediatePropagation() {}
  };
}
if (typeof CustomEvent === "undefined") {
  globalThis.CustomEvent = class CustomEvent extends Event {
    constructor(type, opts) {
      super(type, opts);
      this.detail = opts && opts.detail !== undefined ? opts.detail : null;
    }
  };
}
`)
}
