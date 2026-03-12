package jsbridge

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/fastschema/qjs"
)

// PathPolyfill provides path.join, path.resolve, path.dirname, path.basename, path.extname.
type PathPolyfill struct{}

// Path creates a path polyfill.
func Path() *PathPolyfill { return &PathPolyfill{} }

func (p *PathPolyfill) Name() string { return "path" }

func (p *PathPolyfill) Setup(ctx *qjs.Context) error {
	ctx.SetFunc("__go_path_join", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("path.join: parts_json argument required")
		}
		var parts []string
		if err := json.Unmarshal([]byte(args[0].String()), &parts); err != nil {
			return nil, fmt.Errorf("path.join: json unmarshal: %w", err)
		}
		return this.Context().NewString(filepath.Join(parts...)), nil
	})

	ctx.SetFunc("__go_path_resolve", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("path.resolve: parts_json argument required")
		}
		var parts []string
		if err := json.Unmarshal([]byte(args[0].String()), &parts); err != nil {
			return nil, fmt.Errorf("path.resolve: json unmarshal: %w", err)
		}
		joined := filepath.Join(parts...)
		abs, err := filepath.Abs(joined)
		if err != nil {
			return nil, fmt.Errorf("path.resolve: %w", err)
		}
		return this.Context().NewString(abs), nil
	})

	ctx.SetFunc("__go_path_dirname", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("path.dirname: path argument required")
		}
		return this.Context().NewString(filepath.Dir(args[0].String())), nil
	})

	ctx.SetFunc("__go_path_basename", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("path.basename: path argument required")
		}
		return this.Context().NewString(filepath.Base(args[0].String())), nil
	})

	ctx.SetFunc("__go_path_extname", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("path.extname: path argument required")
		}
		return this.Context().NewString(filepath.Ext(args[0].String())), nil
	})

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
