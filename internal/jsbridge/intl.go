package jsbridge

import quickjs "github.com/buke/quickjs-go"

// IntlPolyfill provides a minimal Intl.DateTimeFormat for QuickJS.
// Observational memory uses Intl.DateTimeFormat for timestamp formatting.
// QuickJS doesn't have Intl natively.
type IntlPolyfill struct{}

func Intl() *IntlPolyfill { return &IntlPolyfill{} }

func (p *IntlPolyfill) Name() string { return "intl" }

func (p *IntlPolyfill) Setup(ctx *quickjs.Context) error {
	return evalJS(ctx, `
if (typeof Intl === "undefined") {
  function _DateTimeFormat(locale, opts) {
    if (!(this instanceof _DateTimeFormat)) return new _DateTimeFormat(locale, opts);
    this._opts = opts || {};
  }
  _DateTimeFormat.prototype.format = function(date) {
    var d = date || new Date();
    if (!(d instanceof Date)) d = new Date(d);
    var Y = d.getFullYear();
    var M = String(d.getMonth() + 1).padStart(2, "0");
    var D = String(d.getDate()).padStart(2, "0");
    var h = String(d.getHours()).padStart(2, "0");
    var m = String(d.getMinutes()).padStart(2, "0");
    return Y + "-" + M + "-" + D + " " + h + ":" + m;
  };
  _DateTimeFormat.prototype.resolvedOptions = function() {
    return { locale: "en-US", timeZone: "UTC" };
  };
  _DateTimeFormat.supportedLocalesOf = function() { return ["en-US"]; };
  globalThis.Intl = { DateTimeFormat: _DateTimeFormat };
}
`)
}
