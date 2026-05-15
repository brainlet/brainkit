package jsbridge

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	quickjs "github.com/buke/quickjs-go"
)

// IntlPolyfill provides a minimal Intl.DateTimeFormat for QuickJS.
// Observational memory uses Intl.DateTimeFormat for timestamp formatting.
// QuickJS doesn't have Intl natively.
type IntlPolyfill struct{}

func Intl() *IntlPolyfill { return &IntlPolyfill{} }

func (p *IntlPolyfill) Name() string { return "intl" }

func (p *IntlPolyfill) Setup(ctx *quickjs.Context) error {
	ctx.Globals().Set("__go_intl_datetime_format_parts", ctx.NewFunction(func(qctx *quickjs.Context, _ *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.NewString(`{"error":"Intl.DateTimeFormat: expected timestamp"}`)
		}
		ms := args[0].ToFloat64()
		timeZone := "UTC"
		if len(args) >= 2 && !args[1].IsUndefined() && !args[1].IsNull() {
			timeZone = args[1].String()
		}
		out, err := intlDateTimeFormatParts(ms, timeZone)
		if err != nil {
			b, _ := json.Marshal(map[string]string{"error": err.Error()})
			return qctx.NewString(string(b))
		}
		b, _ := json.Marshal(map[string]any{"parts": out})
		return qctx.NewString(string(b))
	}))

	return evalJS(ctx, `
(function() {
  function _DateTimeFormat(locale, opts) {
    if (!(this instanceof _DateTimeFormat)) return new _DateTimeFormat(locale, opts);
    this._locale = locale || "en-US";
    this._opts = opts || {};
    this._timeZone = this._opts.timeZone || "UTC";
  }
  function _dateValue(date) {
    var d = date === undefined ? new Date() : date;
    if (!(d instanceof Date)) d = new Date(d);
    var ms = d.getTime();
    if (!isFinite(ms)) throw new RangeError("Invalid time value");
    return ms;
  }
  function _parts(ms, timeZone) {
    var raw = __go_intl_datetime_format_parts(ms, timeZone || "UTC");
    var result = JSON.parse(raw);
    if (result && result.error) throw new RangeError(result.error);
    return result.parts || [];
  }
  function _part(parts, type) {
    for (var i = 0; i < parts.length; i++) {
      if (parts[i].type === type) return parts[i].value;
    }
    return "";
  }
  _DateTimeFormat.prototype.formatToParts = function(date) {
    return _parts(_dateValue(date), this._timeZone).slice();
  };
  _DateTimeFormat.prototype.format = function(date) {
    var parts = _parts(_dateValue(date), this._timeZone);
    var Y = _part(parts, "year");
    var M = String(_part(parts, "month")).padStart(2, "0");
    var D = String(_part(parts, "day")).padStart(2, "0");
    var h = String(_part(parts, "hour")).padStart(2, "0");
    var m = String(_part(parts, "minute")).padStart(2, "0");
    return Y + "-" + M + "-" + D + " " + h + ":" + m;
  };
  _DateTimeFormat.prototype.resolvedOptions = function() {
    return { locale: this._locale || "en-US", timeZone: this._timeZone || "UTC" };
  };
  _DateTimeFormat.supportedLocalesOf = function(locales) {
    if (locales === undefined) return ["en-US"];
    return Array.isArray(locales) ? locales.slice() : [locales];
  };

  if (typeof Intl === "undefined") {
    globalThis.Intl = { DateTimeFormat: _DateTimeFormat };
    return;
  }
  if (typeof Intl.DateTimeFormat !== "function") {
    Intl.DateTimeFormat = _DateTimeFormat;
    return;
  }
  if (typeof Intl.DateTimeFormat.prototype.formatToParts !== "function") {
    Intl.DateTimeFormat = _DateTimeFormat;
  }
})();
`)
}

type intlDateTimePart struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func intlDateTimeFormatParts(ms float64, timeZone string) ([]intlDateTimePart, error) {
	if math.IsNaN(ms) || math.IsInf(ms, 0) {
		return nil, fmt.Errorf("invalid date")
	}
	if timeZone == "" {
		timeZone = "UTC"
	}
	loc, err := time.LoadLocation(timeZone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %q: %w", timeZone, err)
	}
	t := time.Unix(0, int64(ms)*int64(time.Millisecond)).In(loc)
	return []intlDateTimePart{
		{Type: "month", Value: fmt.Sprintf("%d", int(t.Month()))},
		{Type: "literal", Value: "/"},
		{Type: "day", Value: fmt.Sprintf("%d", t.Day())},
		{Type: "literal", Value: "/"},
		{Type: "year", Value: fmt.Sprintf("%04d", t.Year())},
		{Type: "literal", Value: ", "},
		{Type: "hour", Value: fmt.Sprintf("%02d", t.Hour())},
		{Type: "literal", Value: ":"},
		{Type: "minute", Value: fmt.Sprintf("%02d", t.Minute())},
		{Type: "literal", Value: ":"},
		{Type: "second", Value: fmt.Sprintf("%02d", t.Second())},
	}, nil
}
