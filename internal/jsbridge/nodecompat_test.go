package jsbridge

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNodeCompatPureModules(t *testing.T) {
	b := newTestBridge(t, Inspect(), Encoding(), Buffer(), URL(), Performance(), Process(), NodeCompat())

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
			, httpRequest: (function() {
				var req = http.request("http://example.test");
				return typeof req.end === "function" && typeof req.setHeader === "function";
			})()
			, httpCreateServerThrows: (function() {
				try { http.createServer(); return false; }
				catch (_) { return true; }
			})()
			, httpStatus: http.STATUS_CODES[404]
			, httpsAgent: typeof https.Agent === "function"
			, moduleAssert: node_module.createRequire("file:///tmp/pkg/index.js")("node:assert").strictEqual === assert.strictEqual
			, moduleBufferPrototype: typeof node_module.createRequire("file:///tmp/pkg/index.js")("node:buffer").Buffer === "function" &&
				!!Object.create(node_module.createRequire("file:///tmp/pkg/index.js")("buffer").Buffer.prototype)
			, processBuiltinUtil: process.getBuiltinModule("node:util").format("m:%s", "ok")
			, moduleIsBuiltin: node_module.isBuiltin("node:fs") && !node_module.isBuiltin("left-pad")
			, dynamicRequireBoundary: (function() {
				try { node_module.createRequire("file:///tmp/pkg/index.js")("left-pad"); return ""; }
				catch (err) { return err.code + ":" + err.boundaryClass + ":" + err.importPath; }
			})()
		});
	`)

	var parsed struct {
		AssertOK               bool   `json:"assertOk"`
		Query                  string `json:"query"`
		Encoded                string `json:"encoded"`
		Decoded                string `json:"decoded"`
		IsDate                 bool   `json:"isDate"`
		IsTypedArray           bool   `json:"isTypedArray"`
		HasOwn                 bool   `json:"hasOwn"`
		Formatted              string `json:"formatted"`
		PerfNow                bool   `json:"perfNow"`
		Observer               bool   `json:"observer"`
		Delay                  bool   `json:"delay"`
		AsyncStore             string `json:"asyncStore"`
		Diagnostics            bool   `json:"diagnostics"`
		WorkerThrows           bool   `json:"workerThrows"`
		MessageChan            bool   `json:"messageChannel"`
		HTTPRequest            bool   `json:"httpRequest"`
		HTTPCreateServerThrows bool   `json:"httpCreateServerThrows"`
		HTTPStatus             string `json:"httpStatus"`
		HTTPSAgent             bool   `json:"httpsAgent"`
		ModuleAssert           bool   `json:"moduleAssert"`
		ModuleBufferPrototype  bool   `json:"moduleBufferPrototype"`
		ProcessBuiltinUtil     string `json:"processBuiltinUtil"`
		ModuleIsBuiltin        bool   `json:"moduleIsBuiltin"`
		DynamicRequireBoundary string `json:"dynamicRequireBoundary"`
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
	if !parsed.HTTPRequest || !parsed.HTTPCreateServerThrows || parsed.HTTPStatus != "Not Found" || !parsed.HTTPSAgent {
		t.Fatalf("http/https shape failed: %+v", parsed)
	}
	if !parsed.ModuleAssert || !parsed.ModuleBufferPrototype || parsed.ProcessBuiltinUtil != "m:ok" || !parsed.ModuleIsBuiltin {
		t.Fatalf("module/getBuiltinModule shape failed: %+v", parsed)
	}
	if parsed.DynamicRequireBoundary != "BRAINKIT_UNSUPPORTED_BOUNDARY:dynamic-require:left-pad" {
		t.Fatalf("dynamic require boundary = %q", parsed.DynamicRequireBoundary)
	}
}

func TestNodeCompatHTTPClientRequestAndGet(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("X-Reply", r.Header.Get("X-Test")+"-"+r.URL.Query().Get("x"))
		fmt.Fprintf(w, "%s %s", r.Method, string(body))
	})
	httpSrv := httptest.NewServer(handler)
	defer httpSrv.Close()
	httpsSrv := httptest.NewTLSServer(handler)
	defer httpsSrv.Close()

	b := newTestBridge(t,
		Encoding(),
		Streams(),
		URL(),
		Timers(),
		Abort(),
		Events(),
		Buffer(),
		NodeCompat(),
		Fetch(FetchClient(httpsSrv.Client())),
	)

	val, err := b.EvalAsync("http-client.js", fmt.Sprintf(`(async function() {
		function collect(res) {
			return new Promise(function(resolve, reject) {
				var chunks = [];
				res.setEncoding("utf8");
				res.on("data", function(chunk) { chunks.push(chunk); });
				res.on("end", function() {
					resolve({
						status: res.statusCode,
						reply: res.headers["x-reply"],
						body: chunks.join("")
					});
				});
				res.on("error", reject);
				res.resume();
			});
		}
		var post = await new Promise(function(resolve, reject) {
			var req = http.request(%q + "/echo?x=1", {
				method: "POST",
				headers: { "X-Test": "yes" }
			}, function(res) {
				collect(res).then(resolve, reject);
			});
			req.on("error", reject);
			req.write("hello");
			req.end(" world");
		});
		var get = await new Promise(function(resolve, reject) {
			var req = https.get(%q + "/echo?x=2", function(res) {
				collect(res).then(resolve, reject);
			});
			req.on("error", reject);
		});
		return JSON.stringify({ post: post, get: get });
	})()`, httpSrv.URL, httpsSrv.URL))
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()

	var got struct {
		Post struct {
			Status int    `json:"status"`
			Reply  string `json:"reply"`
			Body   string `json:"body"`
		} `json:"post"`
		Get struct {
			Status int    `json:"status"`
			Reply  string `json:"reply"`
			Body   string `json:"body"`
		} `json:"get"`
	}
	if err := json.Unmarshal([]byte(val.String()), &got); err != nil {
		t.Fatalf("json: %v\n%s", err, val.String())
	}
	if got.Post.Status != 200 || got.Post.Reply != "yes-1" || got.Post.Body != "POST hello world" {
		t.Fatalf("post response mismatch: %+v", got.Post)
	}
	if got.Get.Status != 200 || got.Get.Reply != "-2" || got.Get.Body != "GET " {
		t.Fatalf("get response mismatch: %+v", got.Get)
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
			var other = new async_hooks.AsyncLocalStorage();
			var nested = als.run("outer", function() {
				var before = als.getStore();
				var otherBefore = typeof other.getStore();
				var otherNested = other.run("other", function() {
					return als.getStore() + ":" + other.getStore();
				});
				var inner = als.run("inner", function(a, b) {
					return als.getStore() + ":" + a + b;
				}, "a", "b");
				return before + "|" + otherBefore + "|" + otherNested + "|" + inner + "|" + als.getStore() + "|" + typeof other.getStore();
			});
			var after = typeof als.getStore();
			als.enterWith("entered");
			var entered = als.getStore();
			als.disable();
			var disabled = typeof als.getStore();
			var resource = als.run("resource-store", function() {
				return new async_hooks.AsyncResource("brainkit-test");
			});
			var scoped = resource.runInAsyncScope(function(a, b) {
				return this.prefix + a + b + ":" + als.getStore();
			}, { prefix: "scope:" }, "x", "y");
			var hook = async_hooks.createHook({ init: function() {} });
			var hookChain = hook.enable() === hook && hook.disable() === hook;

			var channel = diagnostics_channel.channel("brainkit.test");
			var diagnosticsInitial = channel.hasSubscribers || diagnostics_channel.hasSubscribers("brainkit.test");
			var order = [];
			function first(message, name) {
				order.push("first:" + message.value + ":" + name + ":" + channel.hasSubscribers);
			}
			function second(message, name) {
				order.push("second:" + message.value + ":" + name);
			}
			channel.subscribe(first);
			diagnostics_channel.subscribe("brainkit.test", second);
			var diagnosticsSubscribed = channel.hasSubscribers && diagnostics_channel.hasSubscribers("brainkit.test");
			channel.publish({ value: "one" });
			channel.unsubscribe(first);
			diagnostics_channel.unsubscribe("brainkit.test", second);
			var diagnosticsAfterUnsubscribe = channel.hasSubscribers || diagnostics_channel.hasSubscribers("brainkit.test");

			var errorChannel = diagnostics_channel.channel("brainkit.error");
			var errorMessage = "";
			errorChannel.subscribe(function() {
				order.push("before-error");
				throw new Error("boom");
			});
			try {
				errorChannel.publish({});
			} catch (err) {
				errorMessage = err.message;
			}

			var tracing = diagnostics_channel.tracingChannel("brainkit.trace");
			var traceEvents = [];
			function onTrace(message, name) {
				traceEvents.push(name + ":" + message.step);
			}
			tracing.start.subscribe(onTrace);
			var tracingBefore = tracing.hasSubscribers;
			tracing.start.publish({ step: "start" });
			tracing.start.unsubscribe(onTrace);
			var tracingAfter = tracing.hasSubscribers;
			return {
				nested: nested,
				after: after,
				entered: entered,
				disabled: disabled,
				scoped: scoped,
				ids: [async_hooks.executionAsyncId(), async_hooks.triggerAsyncId()],
				hookChain: hookChain,
				diagnosticsInitial: diagnosticsInitial,
				diagnosticsSubscribed: diagnosticsSubscribed,
				diagnosticsAfterUnsubscribe: diagnosticsAfterUnsubscribe,
				order: order,
				errorMessage: errorMessage,
				tracingBefore: tracingBefore,
				tracingAfter: tracingAfter,
				traceEvents: traceEvents
			};
		})());
	`)

	var parsed struct {
		Nested                      string   `json:"nested"`
		After                       string   `json:"after"`
		Entered                     string   `json:"entered"`
		Disabled                    string   `json:"disabled"`
		Scoped                      string   `json:"scoped"`
		IDs                         []int    `json:"ids"`
		HookChain                   bool     `json:"hookChain"`
		DiagnosticsInitial          bool     `json:"diagnosticsInitial"`
		DiagnosticsSubscribed       bool     `json:"diagnosticsSubscribed"`
		DiagnosticsAfterUnsubscribe bool     `json:"diagnosticsAfterUnsubscribe"`
		Order                       []string `json:"order"`
		ErrorMessage                string   `json:"errorMessage"`
		TracingBefore               bool     `json:"tracingBefore"`
		TracingAfter                bool     `json:"tracingAfter"`
		TraceEvents                 []string `json:"traceEvents"`
	}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, result)
	}
	if parsed.Nested != "outer|undefined|outer:other|inner:ab|outer|undefined" || parsed.After != "undefined" {
		t.Fatalf("AsyncLocalStorage nesting/restoration failed: %+v", parsed)
	}
	if parsed.Entered != "entered" || parsed.Disabled != "undefined" {
		t.Fatalf("AsyncLocalStorage enter/disable failed: %+v", parsed)
	}
	if parsed.Scoped != "scope:xy:resource-store" {
		t.Fatalf("AsyncResource scope = %q", parsed.Scoped)
	}
	if len(parsed.IDs) != 2 || parsed.IDs[0] != 0 || parsed.IDs[1] != 0 || !parsed.HookChain {
		t.Fatalf("async_hooks ids/hook contract failed: %+v", parsed)
	}
	if parsed.DiagnosticsInitial || !parsed.DiagnosticsSubscribed || parsed.DiagnosticsAfterUnsubscribe {
		t.Fatalf("diagnostics_channel hasSubscribers transitions failed: %+v", parsed)
	}
	if got, want := parsed.Order, []string{"first:one:brainkit.test:true", "second:one:brainkit.test", "before-error"}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("diagnostics_channel publish order = %v, want %v", got, want)
	}
	if parsed.ErrorMessage != "boom" {
		t.Fatalf("diagnostics_channel publish error = %q", parsed.ErrorMessage)
	}
	if !parsed.TracingBefore || parsed.TracingAfter || fmt.Sprint(parsed.TraceEvents) != fmt.Sprint([]string{"brainkit.trace:start:start"}) {
		t.Fatalf("diagnostics_channel tracing channels failed: %+v", parsed)
	}
}

func TestAsyncLocalStoragePropagatesAcrossOwnedAsyncBoundaries(t *testing.T) {
	b := newTestBridge(t, Timers(), Scheduling(), Events(), NodeCompat())

	val, err := b.EvalAsync("als-owned-boundaries.js", `(async function() {
		var als = new async_hooks.AsyncLocalStorage();
		var seen = [];
		var emitter = new EventEmitter();

		als.run("timer-zero", function() {
			setTimeout(function(label) {
				seen.push("timeout0:" + als.getStore() + ":" + label);
			}, 0, "a");
		});
		await new Promise(function(resolve) { setTimeout(resolve, 2); });

		als.run("timer-delay", function() {
			setTimeout(function() {
				seen.push("timeout-delay:" + als.getStore());
			}, 1);
		});
		await new Promise(function(resolve) { setTimeout(resolve, 5); });

		als.run("immediate", function() {
			setImmediate(function() {
				seen.push("immediate:" + als.getStore());
			});
		});
		await new Promise(function(resolve) { setTimeout(resolve, 2); });

		als.run("interval", function() {
			var id = setInterval(function() {
				seen.push("interval:" + als.getStore());
				clearInterval(id);
			}, 1);
		});
		await new Promise(function(resolve) { setTimeout(resolve, 5); });

		als.run("timer-promise", function() {
			timersPromises.setTimeout(1, "tp").then(function(value) {
				seen.push("timer-promise:" + als.getStore() + ":" + value);
			});
		});
		await new Promise(function(resolve) { setTimeout(resolve, 5); });

		als.run("event", function() {
			emitter.on("hit", function(value) {
				seen.push("event:" + als.getStore() + ":" + value);
			});
		});
		emitter.emit("hit", "ok");

		var listenerRemoved = false;
		function removable() {
			seen.push("removed");
		}
		als.run("remove", function() {
			emitter.on("remove-test", removable);
			emitter.removeListener("remove-test", removable);
			listenerRemoved = emitter.listenerCount("remove-test") === 0;
			emitter.emit("remove-test");
		});

		return JSON.stringify({ seen: seen, after: typeof als.getStore(), listenerRemoved: listenerRemoved });
	})()`)
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()

	var parsed struct {
		Seen            []string `json:"seen"`
		After           string   `json:"after"`
		ListenerRemoved bool     `json:"listenerRemoved"`
	}
	if err := json.Unmarshal([]byte(val.String()), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, val.String())
	}
	want := []string{
		"timeout0:timer-zero:a",
		"timeout-delay:timer-delay",
		"immediate:immediate",
		"interval:interval",
		"timer-promise:timer-promise:tp",
		"event:event:ok",
	}
	if fmt.Sprint(parsed.Seen) != fmt.Sprint(want) || parsed.After != "undefined" || !parsed.ListenerRemoved {
		t.Fatalf("AsyncLocalStorage owned-boundary propagation failed: got %+v want seen %v", parsed, want)
	}
}

func TestAsyncLocalStoragePropagatesAcrossFetchThenCallbacks(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("fetch-ok"))
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	b := newTestBridge(t,
		Encoding(),
		Streams(),
		URL(),
		Timers(),
		Scheduling(),
		Events(),
		NodeCompat(),
		Fetch(FetchClient(srv.Client())),
	)

	val, err := b.EvalAsync("als-fetch-then.js", fmt.Sprintf(`(async function() {
		var als = new async_hooks.AsyncLocalStorage();
		var seen = [];
		await new Promise(function(resolve, reject) {
			als.run("fetch-store", function() {
				fetch(%q).then(function(res) {
					seen.push("fetch-then:" + als.getStore() + ":" + res.status);
					return res.text();
				}).then(function(text) {
					seen.push("fetch-text:" + als.getStore() + ":" + text);
					resolve();
				}, reject);
			});
		});
		return JSON.stringify({ seen: seen, after: typeof als.getStore() });
	})()`, srv.URL))
	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()

	var parsed struct {
		Seen  []string `json:"seen"`
		After string   `json:"after"`
	}
	if err := json.Unmarshal([]byte(val.String()), &parsed); err != nil {
		t.Fatalf("json: %v\n%s", err, val.String())
	}
	want := []string{"fetch-then:fetch-store:201", "fetch-text:fetch-store:fetch-ok"}
	if fmt.Sprint(parsed.Seen) != fmt.Sprint(want) || parsed.After != "undefined" {
		t.Fatalf("AsyncLocalStorage fetch callback propagation failed: got %+v want seen %v", parsed, want)
	}
}

func TestUnsupportedBoundaryDiagnostics(t *testing.T) {
	b := newTestBridge(t, Encoding(), Events(), NodeStreams(), Buffer(), Timers(), NodeCompat(), Net())

	result := evalString(t, b, `
		function capture(fn) {
			try {
				fn();
				return { threw: false };
			} catch (err) {
				return {
					threw: true,
					name: err && err.name || "",
					code: err && err.code || "",
					importPath: err && err.importPath || "",
					api: err && err.api || "",
					boundaryClass: err && err.boundaryClass || "",
					suggestedOwner: err && err.suggestedOwner || "",
					owner: err && err.owner || ""
				};
			}
		}
		JSON.stringify({
			http: capture(function() { http.createServer(); }),
			https: capture(function() { https.createServer(); }),
			net: capture(function() { net.createServer(); }),
			tls: capture(function() { tls.createServer(); }),
			worker: capture(function() { new worker_threads.Worker("worker.js"); })
		});
	`)

	var got map[string]struct {
		Threw          bool   `json:"threw"`
		Name           string `json:"name"`
		Code           string `json:"code"`
		ImportPath     string `json:"importPath"`
		API            string `json:"api"`
		BoundaryClass  string `json:"boundaryClass"`
		SuggestedOwner string `json:"suggestedOwner"`
		Owner          string `json:"owner"`
	}
	if err := json.Unmarshal([]byte(result), &got); err != nil {
		t.Fatalf("json: %v\n%s", err, result)
	}
	for name, diag := range got {
		if !diag.Threw || diag.Name != "BrainkitUnsupportedBoundaryError" || diag.Code != "BRAINKIT_UNSUPPORTED_BOUNDARY" {
			t.Fatalf("%s unsupported boundary base diagnostic failed: %+v", name, diag)
		}
		if diag.BoundaryClass == "" || diag.SuggestedOwner == "" || diag.API == "" || diag.Owner == "" {
			t.Fatalf("%s unsupported boundary metadata missing: %+v", name, diag)
		}
	}
	for _, name := range []string{"http", "https", "net", "tls"} {
		if got[name].BoundaryClass != "server-listener" {
			t.Fatalf("%s boundaryClass = %q, want server-listener", name, got[name].BoundaryClass)
		}
	}
	if got["worker"].BoundaryClass != "worker" || got["worker"].ImportPath != "worker_threads" {
		t.Fatalf("worker boundary diagnostic failed: %+v", got["worker"])
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
