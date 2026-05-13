package jsbridge

import (
	"encoding/json"
	"testing"
)

func TestNodeCompatPureModules(t *testing.T) {
	b := newTestBridge(t, Inspect(), Encoding(), Buffer(), Performance(), NodeCompat())

	result := evalString(t, b, `
		JSON.stringify({
			assertOk: (function() { assert.strictEqual(1, 1); return true; })(),
			query: querystring.parse("a=1&b=two+words").b,
			encoded: querystring.stringify({ a: 1, b: "two words" }),
			decoded: new StringDecoder("utf-8").end(new Uint8Array([104, 105])),
			isDate: util.types.isDate(new Date()),
			isTypedArray: utilTypes.isTypedArray(new Uint8Array([1])),
			hasOwn: (function() {
				var shadow = { hasOwnProperty: function() { return false; }, own: 1 };
				var inherited = Object.create({ inherited: 1 });
				inherited.own = 2;
				var nullProto = Object.create(null);
				nullProto.own = 3;
				return Object.hasOwn(shadow, "own") &&
					Object.hasOwn(inherited, "own") &&
					!Object.hasOwn(inherited, "inherited") &&
					Object.hasOwn(nullProto, "own");
			})(),
			formatted: util.format("x:%s", "ok"),
			perfNow: typeof perf_hooks.performance.now === "function",
			observer: typeof perf_hooks.PerformanceObserver === "function",
			delay: typeof perf_hooks.monitorEventLoopDelay().enable === "function"
			, asyncStore: (function() {
				var als = new async_hooks.AsyncLocalStorage();
				return als.run("store", function() { return als.getStore(); });
			})()
			, diagnostics: diagnostics_channel.channel("test").hasSubscribers === false
			, workerThrows: (function() {
				try { new worker_threads.Worker("x"); return false; }
				catch (_) { return true; }
			})()
			, messageChannel: !!new worker_threads.MessageChannel().port1
			, httpThrows: (function() {
				try { http.request(); return false; }
				catch (_) { return true; }
			})()
			, httpStatus: http.STATUS_CODES[404]
			, httpsAgent: typeof https.Agent === "function"
		});
	`)

	var parsed struct {
		AssertOK     bool   `json:"assertOk"`
		Query        string `json:"query"`
		Encoded      string `json:"encoded"`
		Decoded      string `json:"decoded"`
		IsDate       bool   `json:"isDate"`
		IsTypedArray bool   `json:"isTypedArray"`
		HasOwn       bool   `json:"hasOwn"`
		Formatted    string `json:"formatted"`
		PerfNow      bool   `json:"perfNow"`
		Observer     bool   `json:"observer"`
		Delay        bool   `json:"delay"`
		AsyncStore   string `json:"asyncStore"`
		Diagnostics  bool   `json:"diagnostics"`
		WorkerThrows bool   `json:"workerThrows"`
		MessageChan  bool   `json:"messageChannel"`
		HTTPThrows   bool   `json:"httpThrows"`
		HTTPStatus   string `json:"httpStatus"`
		HTTPSAgent   bool   `json:"httpsAgent"`
	}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, result)
	}
	if !parsed.AssertOK {
		t.Fatal("assert.strictEqual did not run")
	}
	if parsed.Query != "two words" {
		t.Fatalf("querystring.parse decoded b = %q", parsed.Query)
	}
	if parsed.Encoded != "a=1&b=two%20words" {
		t.Fatalf("querystring.stringify = %q", parsed.Encoded)
	}
	if parsed.Decoded != "hi" {
		t.Fatalf("StringDecoder decoded = %q", parsed.Decoded)
	}
	if !parsed.IsDate || !parsed.IsTypedArray {
		t.Fatalf("util type checks failed: %+v", parsed)
	}
	if !parsed.HasOwn {
		t.Fatalf("Object.hasOwn compatibility failed: %+v", parsed)
	}
	if parsed.Formatted != "x:ok" {
		t.Fatalf("util.format = %q", parsed.Formatted)
	}
	if !parsed.PerfNow || !parsed.Observer || !parsed.Delay {
		t.Fatalf("perf_hooks shape failed: %+v", parsed)
	}
	if parsed.AsyncStore != "store" || !parsed.Diagnostics || !parsed.WorkerThrows || !parsed.MessageChan {
		t.Fatalf("async/diagnostics/worker shape failed: %+v", parsed)
	}
	if !parsed.HTTPThrows || parsed.HTTPStatus != "Not Found" || !parsed.HTTPSAgent {
		t.Fatalf("http/https shape failed: %+v", parsed)
	}
}

func TestTimerPromises(t *testing.T) {
	b := newTestBridge(t, Timers(), Scheduling())

	val, err := b.EvalAsync("timer-promises.js", `(async function() {
		var timeout = await timersPromises.setTimeout(1, "done");
		var iter = timersPromises.setInterval(1, "tick")[Symbol.asyncIterator]();
		var first = await iter.next();
		await iter.return();
		return JSON.stringify({ timeout: timeout, interval: first.value, done: first.done });
	})()`)
	if err != nil {
		t.Fatal(err)
	}
	defer val.Free()
	if val.String() != `{"timeout":"done","interval":"tick","done":false}` {
		t.Fatalf("timer promises = %s", val.String())
	}
}

func TestAsyncHooksAndDiagnosticsChannelContracts(t *testing.T) {
	b := newTestBridge(t, NodeCompat())

	result := evalString(t, b, `
		JSON.stringify((function() {
			var als = new async_hooks.AsyncLocalStorage();
			var nested = als.run("outer", function() {
				var before = als.getStore();
				var inner = als.run("inner", function(a, b) {
					return als.getStore() + ":" + a + b;
				}, "a", "b");
				return before + "|" + inner + "|" + als.getStore();
			});
			var after = typeof als.getStore();
			als.enterWith("entered");
			var entered = als.getStore();
			als.disable();
			var disabled = typeof als.getStore();
			var resource = new async_hooks.AsyncResource("brainkit-test");
			var scoped = resource.runInAsyncScope(function(a, b) {
				return this.prefix + a + b;
			}, { prefix: "scope:" }, "x", "y");
			var hook = async_hooks.createHook({ init: function() {} });
			var hookChain = hook.enable() === hook && hook.disable() === hook;
			var channel = diagnostics_channel.channel("brainkit.test");
			channel.subscribe(function() { throw new Error("diagnostics_channel is shape-only"); });
			channel.publish({ hidden: true });
			var tracing = diagnostics_channel.tracingChannel("brainkit.trace");
			return {
				nested: nested,
				after: after,
				entered: entered,
				disabled: disabled,
				scoped: scoped,
				ids: [async_hooks.executionAsyncId(), async_hooks.triggerAsyncId()],
				hookChain: hookChain,
				hasSubscribers: channel.hasSubscribers,
				globalHasSubscribers: diagnostics_channel.hasSubscribers("brainkit.test"),
				tracing: typeof tracing.start.publish + ":" + tracing.hasSubscribers
			};
		})());
	`)

	var parsed struct {
		Nested               string `json:"nested"`
		After                string `json:"after"`
		Entered              string `json:"entered"`
		Disabled             string `json:"disabled"`
		Scoped               string `json:"scoped"`
		IDs                  []int  `json:"ids"`
		HookChain            bool   `json:"hookChain"`
		HasSubscribers       bool   `json:"hasSubscribers"`
		GlobalHasSubscribers bool   `json:"globalHasSubscribers"`
		Tracing              string `json:"tracing"`
	}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, result)
	}
	if parsed.Nested != "outer|inner:ab|outer" || parsed.After != "undefined" {
		t.Fatalf("AsyncLocalStorage nesting/restoration failed: %+v", parsed)
	}
	if parsed.Entered != "entered" || parsed.Disabled != "undefined" {
		t.Fatalf("AsyncLocalStorage enter/disable failed: %+v", parsed)
	}
	if parsed.Scoped != "scope:xy" {
		t.Fatalf("AsyncResource scope = %q", parsed.Scoped)
	}
	if len(parsed.IDs) != 2 || parsed.IDs[0] != 0 || parsed.IDs[1] != 0 || !parsed.HookChain {
		t.Fatalf("async_hooks ids/hook contract failed: %+v", parsed)
	}
	if parsed.HasSubscribers || parsed.GlobalHasSubscribers || parsed.Tracing != "function:false" {
		t.Fatalf("diagnostics_channel shape-only contract failed: %+v", parsed)
	}
}

func TestPathExtendedSurface(t *testing.T) {
	b := newTestBridge(t, Path())

	result := evalString(t, b, `
		JSON.stringify({
			normalized: path.normalize("/tmp/a/../b"),
			absolute: path.isAbsolute("/tmp"),
			parsed: path.parse("/tmp/file.txt"),
			relative: path.relative("/tmp/a", "/tmp/a/b/c"),
			posixJoin: path.posix.join("/tmp", "x")
		});
	`)

	var parsed struct {
		Normalized string `json:"normalized"`
		Absolute   bool   `json:"absolute"`
		Parsed     struct {
			Dir  string `json:"dir"`
			Base string `json:"base"`
			Ext  string `json:"ext"`
			Name string `json:"name"`
		} `json:"parsed"`
		Relative  string `json:"relative"`
		PosixJoin string `json:"posixJoin"`
	}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, result)
	}
	if parsed.Normalized != "/tmp/b" {
		t.Fatalf("normalize = %q", parsed.Normalized)
	}
	if !parsed.Absolute {
		t.Fatal("isAbsolute returned false")
	}
	if parsed.Parsed.Dir != "/tmp" || parsed.Parsed.Base != "file.txt" || parsed.Parsed.Ext != ".txt" || parsed.Parsed.Name != "file" {
		t.Fatalf("parse = %+v", parsed.Parsed)
	}
	if parsed.Relative != "b/c" {
		t.Fatalf("relative = %q", parsed.Relative)
	}
	if parsed.PosixJoin != "/tmp/x" {
		t.Fatalf("posix.join = %q", parsed.PosixJoin)
	}
}
