package jsbridge

import quickjs "github.com/buke/quickjs-go"

// PerformancePolyfill provides globalThis.performance with now() and timeOrigin.
// AI SDK and telemetry code use performance.now() for timing.
type PerformancePolyfill struct{}

func Performance() *PerformancePolyfill { return &PerformancePolyfill{} }

func (p *PerformancePolyfill) Name() string { return "performance" }

func (p *PerformancePolyfill) Setup(ctx *quickjs.Context) error {
	return evalJS(ctx, `
if (typeof performance === "undefined") {
  var _perfStart = Date.now();
  globalThis.performance = {
    now: function() { return Date.now() - _perfStart; },
    timeOrigin: _perfStart,
    mark: function() {},
    measure: function() {},
    getEntriesByName: function() { return []; },
    getEntriesByType: function() { return []; },
    clearMarks: function() {},
    clearMeasures: function() {},
  };
}
`)
}
