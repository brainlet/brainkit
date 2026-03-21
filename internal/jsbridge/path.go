package jsbridge

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	quickjs "github.com/buke/quickjs-go"
)

// PathPolyfill provides path.join, path.resolve, path.dirname, path.basename, path.extname.
type PathPolyfill struct{}

// Path creates a path polyfill.
func Path() *PathPolyfill { return &PathPolyfill{} }

func (p *PathPolyfill) Name() string { return "path" }

func (p *PathPolyfill) Setup(ctx *quickjs.Context) error {
	ctx.Globals().Set("__go_path_join", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("path.join: parts_json argument required"))
		}
		var parts []string
		if err := json.Unmarshal([]byte(args[0].ToString()), &parts); err != nil {
			return ctx.ThrowError(fmt.Errorf("path.join: json unmarshal: %w", err))
		}
		return ctx.NewString(filepath.Join(parts...))
	}))

	ctx.Globals().Set("__go_path_resolve", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("path.resolve: parts_json argument required"))
		}
		var parts []string
		if err := json.Unmarshal([]byte(args[0].ToString()), &parts); err != nil {
			return ctx.ThrowError(fmt.Errorf("path.resolve: json unmarshal: %w", err))
		}
		joined := filepath.Join(parts...)
		abs, err := filepath.Abs(joined)
		if err != nil {
			return ctx.ThrowError(fmt.Errorf("path.resolve: %w", err))
		}
		return ctx.NewString(abs)
	}))

	ctx.Globals().Set("__go_path_dirname", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("path.dirname: path argument required"))
		}
		return ctx.NewString(filepath.Dir(args[0].ToString()))
	}))

	ctx.Globals().Set("__go_path_basename", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("path.basename: path argument required"))
		}
		return ctx.NewString(filepath.Base(args[0].ToString()))
	}))

	ctx.Globals().Set("__go_path_extname", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("path.extname: path argument required"))
		}
		return ctx.NewString(filepath.Ext(args[0].ToString()))
	}))

	return evalJS(ctx, `
globalThis.path = {
  join(...parts) { return __go_path_join(JSON.stringify(parts)); },
  resolve(...parts) { return __go_path_resolve(JSON.stringify(parts)); },
  dirname(p) { return __go_path_dirname(p); },
  basename(p) { return __go_path_basename(p); },
  extname(p) { return __go_path_extname(p); },
};
`)
}
