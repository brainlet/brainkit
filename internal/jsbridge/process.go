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

	return evalJS(ctx, `
globalThis.process = globalThis.process || {};
globalThis.process.env = new Proxy({}, {
  get(_, key) { return __go_process_env(key); },
  set(_, key, value) { __go_process_env_set(key, String(value)); return true; },
});
globalThis.process.cwd = function() { return __go_process_cwd(); };
`)
}
