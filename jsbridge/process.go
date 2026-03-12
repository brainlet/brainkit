package jsbridge

import (
	"fmt"
	"os"

	"github.com/fastschema/qjs"
)

// ProcessPolyfill provides process.env and process.cwd.
type ProcessPolyfill struct{}

// Process creates a process polyfill.
func Process() *ProcessPolyfill { return &ProcessPolyfill{} }

func (p *ProcessPolyfill) Name() string { return "process" }

func (p *ProcessPolyfill) Setup(ctx *qjs.Context) error {
	ctx.SetFunc("__go_process_env", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("process.env: key argument required")
		}
		return this.Context().NewString(os.Getenv(args[0].String())), nil
	})

	ctx.SetFunc("__go_process_env_set", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 2 {
			return nil, fmt.Errorf("process.env.set: key and value arguments required")
		}
		if err := os.Setenv(args[0].String(), args[1].String()); err != nil {
			return nil, fmt.Errorf("process.env.set: %w", err)
		}
		return this.Context().NewBool(true), nil
	})

	ctx.SetFunc("__go_process_cwd", func(this *qjs.This) (*qjs.Value, error) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("process.cwd: %w", err)
		}
		return this.Context().NewString(cwd), nil
	})

	return evalJS(ctx, `
globalThis.process = globalThis.process || {};
globalThis.process.env = new Proxy({}, {
  get(_, key) { return __go_process_env(key); },
  set(_, key, value) { __go_process_env_set(key, String(value)); return true; },
});
globalThis.process.cwd = function() { return __go_process_cwd(); };
`)
}
