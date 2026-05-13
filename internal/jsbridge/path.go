package jsbridge

import (
	"encoding/json"
	"fmt"
	"os"
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

	ctx.Globals().Set("__go_path_normalize", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("path.normalize: path argument required"))
		}
		return ctx.NewString(filepath.Clean(args[0].ToString()))
	}))

	ctx.Globals().Set("__go_path_is_abs", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("path.isAbsolute: path argument required"))
		}
		return ctx.NewBool(filepath.IsAbs(args[0].ToString()))
	}))

	ctx.Globals().Set("__go_path_relative", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return ctx.ThrowError(fmt.Errorf("path.relative: from and to arguments required"))
		}
		rel, err := filepath.Rel(args[0].ToString(), args[1].ToString())
		if err != nil {
			return ctx.NewString(args[1].ToString())
		}
		return ctx.NewString(rel)
	}))

	return evalJS(ctx, fmt.Sprintf(`
(function() {
  "use strict";

  var sep = %q;
  var delimiter = %q;

  function basename(p) { return __go_path_basename(String(p)); }
  function extname(p) { return __go_path_extname(String(p)); }
  function dirname(p) { return __go_path_dirname(String(p)); }
  function isAbsolute(p) { return __go_path_is_abs(String(p)); }

  var pathAPI = {
    join: function() { return __go_path_join(JSON.stringify(Array.prototype.slice.call(arguments).map(String))); },
    resolve: function() { return __go_path_resolve(JSON.stringify(Array.prototype.slice.call(arguments).map(String))); },
    dirname: dirname,
    basename: basename,
    extname: extname,
    normalize: function(p) { return __go_path_normalize(String(p)); },
    isAbsolute: isAbsolute,
    parse: function(p) {
      p = String(p);
      var base = basename(p);
      var ext = extname(p);
      var dir = dirname(p);
      return {
        root: isAbsolute(p) ? sep : "",
        dir: dir,
        base: base,
        ext: ext,
        name: ext ? base.slice(0, -ext.length) : base,
      };
    },
    relative: function(from, to) { return __go_path_relative(String(from), String(to)); },
    sep: sep,
    delimiter: delimiter,
  };
  pathAPI.posix = pathAPI;
  globalThis.path = pathAPI;
})();
`, string(os.PathSeparator), string(os.PathListSeparator)))
}
