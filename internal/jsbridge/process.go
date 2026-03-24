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
})();
`)
}
