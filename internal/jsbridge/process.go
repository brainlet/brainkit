package jsbridge

import (
	"fmt"
	"os"

	quickjs "github.com/buke/quickjs-go"
)

// ProcessPolyfill provides process.env and process.cwd.
type ProcessPolyfill struct{}

// Process creates a process polyfill.
func Process() *ProcessPolyfill { return &ProcessPolyfill{} }

func (p *ProcessPolyfill) Name() string { return "process" }

func (p *ProcessPolyfill) Setup(ctx *quickjs.Context) error {
	ctx.Globals().Set("__go_process_env", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("process.env: key argument required"))
		}
		return ctx.NewString(os.Getenv(args[0].ToString()))
	}))

	ctx.Globals().Set("__go_process_env_set", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return ctx.ThrowError(fmt.Errorf("process.env.set: key and value arguments required"))
		}
		if err := os.Setenv(args[0].ToString(), args[1].ToString()); err != nil {
			return ctx.ThrowError(fmt.Errorf("process.env.set: %w", err))
		}
		return ctx.NewBool(true)
	}))

	ctx.Globals().Set("__go_process_cwd", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		cwd, err := os.Getwd()
		if err != nil {
			return ctx.ThrowError(fmt.Errorf("process.cwd: %w", err))
		}
		return ctx.NewString(cwd)
	}))

	// IMPORTANT: Capture bridge functions in a closure BEFORE SES lockdown.
	// SES harden() removes __go_process_env from globalThis ("unpermitted intrinsic").
	// The Proxy's get/set traps must use the captured reference, not globalThis lookup.
	return evalJS(ctx, `
(function() {
  var _env = __go_process_env;
  var _envSet = __go_process_env_set;
  var _cwd = __go_process_cwd;

  globalThis.process = globalThis.process || {};
  globalThis.process.env = new Proxy({}, {
    get: function(_, key) {
      if (typeof key !== "string") return undefined;
      var v = _env(key);
      // Return undefined (not empty string) for unset vars.
      // SDKs like OpenAI treat "" as "user set a custom value" vs
      // undefined as "use default". os.Getenv returns "" for unset.
      if (v === "") return undefined;
      return v;
    },
    set: function(_, key, value) {
      _envSet(key, String(value));
      return true;
    },
    has: function(_, key) {
      if (typeof key !== "string") return false;
      var v = _env(key);
      return v !== undefined && v !== "";
    },
    ownKeys: function() { return []; },
    getOwnPropertyDescriptor: function(_, key) {
      if (typeof key !== "string") return undefined;
      var v = _env(key);
      if (v === undefined || v === "") return undefined;
      return { value: v, writable: true, enumerable: true, configurable: true };
    },
  });
  globalThis.process.cwd = function() { return _cwd(); };

  // ─── Extended process properties ─────────────────────────────
  // Required by AI SDK (process.version), pg (process.nextTick),
  // Mastra (process.stdout), and various libraries.
  var p = globalThis.process;

  if (!p.version) p.version = "v20.0.0";
  if (!p.versions) p.versions = { node: "20.0.0" };
  if (!p.platform) p.platform = "linux";
  if (!p.arch) p.arch = "x64";
  if (p.pid === undefined) p.pid = 1;
  if (!p.argv) p.argv = [];
  if (!p.argv0) p.argv0 = "";
  if (!p.execArgv) p.execArgv = [];
  if (!p.execPath) p.execPath = "/usr/local/bin/node";
  if (!p.title) p.title = "node";
  if (!p.release) p.release = { name: "node" };
  if (!p.config) p.config = {};

  if (!p.chdir) p.chdir = function() {};
  if (!p.exit) p.exit = function() {};
  if (!p.kill) p.kill = function() {};
  if (!p.abort) p.abort = function() {};
  if (!p.umask) p.umask = function() { return 0o22; };
  if (!p.uptime) p.uptime = function() { return 0; };
  if (!p.cpuUsage) p.cpuUsage = function() { return { user: 0, system: 0 }; };
  if (!p.memoryUsage) p.memoryUsage = function() {
    return { rss: 0, heapTotal: 0, heapUsed: 0, external: 0, arrayBuffers: 0 };
  };
  if (!p.hrtime) {
    p.hrtime = function(prev) {
      var now = Date.now();
      var s = Math.floor(now / 1000);
      var ns = (now % 1000) * 1e6;
      if (prev) { s -= prev[0]; ns -= prev[1]; if (ns < 0) { s--; ns += 1e9; } }
      return [s, ns];
    };
    p.hrtime.bigint = function() { return BigInt(Date.now()) * BigInt(1e6); };
  }
  if (!p.nextTick) p.nextTick = function(fn) {
    var args = [];
    for (var i = 1; i < arguments.length; i++) args.push(arguments[i]);
    queueMicrotask(function() { fn.apply(null, args); });
  };

  // Streams
  if (!p.stdout) p.stdout = {
    write: function() { return true; }, isTTY: false, columns: 80, rows: 24,
    on: function() { return this; }, once: function() { return this; }, emit: function() { return false; },
  };
  if (!p.stderr) p.stderr = {
    write: function() { return true; }, isTTY: false, columns: 80, rows: 24,
    on: function() { return this; }, once: function() { return this; }, emit: function() { return false; },
  };
  if (!p.stdin) p.stdin = {
    isTTY: false, on: function() { return this; }, once: function() { return this; },
    resume: function() { return this; }, pause: function() { return this; }, read: function() { return null; },
  };

  // Event-related
  if (!p.on) p.on = function() { return p; };
  if (!p.once) p.once = function() { return p; };
  if (!p.off) p.off = function() { return p; };
  if (!p.emit) p.emit = function() { return false; };
  if (!p.removeListener) p.removeListener = function() { return p; };
  if (!p.removeAllListeners) p.removeAllListeners = function() { return p; };
  if (!p.listeners) p.listeners = function() { return []; };
  if (!p.listenerCount) p.listenerCount = function() { return 0; };
  if (!p.addListener) p.addListener = p.on;
  if (!p.prependListener) p.prependListener = p.on;
  if (!p.prependOnceListener) p.prependOnceListener = p.once;

  if (!p.emitWarning) p.emitWarning = function() {};
  if (!p.features) p.features = {};
  if (!p.allowedNodeEnvironmentFlags) p.allowedNodeEnvironmentFlags = new Set();
})();
`)
}
