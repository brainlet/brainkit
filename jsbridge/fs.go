package jsbridge

import (
	"encoding/json"
	"fmt"
	"os"

	quickjs "github.com/buke/quickjs-go"
)

// FSPolyfill provides fs.readFile, fs.writeFile, fs.readdir, fs.stat, fs.mkdir, fs.unlink, fs.rm.
type FSPolyfill struct{}

// FS creates a file system polyfill.
func FS() *FSPolyfill { return &FSPolyfill{} }

func (p *FSPolyfill) Name() string { return "fs" }

func (p *FSPolyfill) Setup(ctx *quickjs.Context) error {
	ctx.Globals().Set("__go_fs_readFile", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("readFile: path argument required"))
		}
		data, err := os.ReadFile(args[0].ToString())
		if err != nil {
			return ctx.ThrowError(fmt.Errorf("readFile: %w", err))
		}
		return ctx.NewString(string(data))
	}))

	ctx.Globals().Set("__go_fs_writeFile", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return ctx.ThrowError(fmt.Errorf("writeFile: path and data arguments required"))
		}
		if err := os.WriteFile(args[0].ToString(), []byte(args[1].ToString()), 0644); err != nil {
			return ctx.ThrowError(fmt.Errorf("writeFile: %w", err))
		}
		return ctx.NewBool(true)
	}))

	ctx.Globals().Set("__go_fs_readdir", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("readdir: path argument required"))
		}
		entries, err := os.ReadDir(args[0].ToString())
		if err != nil {
			return ctx.ThrowError(fmt.Errorf("readdir: %w", err))
		}
		type dirEntry struct {
			Name        string `json:"name"`
			IsDirectory bool   `json:"isDirectory"`
		}
		result := make([]dirEntry, len(entries))
		for i, e := range entries {
			result[i] = dirEntry{Name: e.Name(), IsDirectory: e.IsDir()}
		}
		b, err := json.Marshal(result)
		if err != nil {
			return ctx.ThrowError(fmt.Errorf("readdir: json marshal: %w", err))
		}
		return ctx.NewString(string(b))
	}))

	ctx.Globals().Set("__go_fs_stat", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("stat: path argument required"))
		}
		info, err := os.Stat(args[0].ToString())
		if err != nil {
			return ctx.ThrowError(fmt.Errorf("stat: %w", err))
		}
		b, err := json.Marshal(map[string]interface{}{
			"size":        info.Size(),
			"isFile":      info.Mode().IsRegular(),
			"isDirectory": info.IsDir(),
			"modTime":     info.ModTime().Unix(),
		})
		if err != nil {
			return ctx.ThrowError(fmt.Errorf("stat: json marshal: %w", err))
		}
		return ctx.NewString(string(b))
	}))

	ctx.Globals().Set("__go_fs_mkdir", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("mkdir: path argument required"))
		}
		if err := os.MkdirAll(args[0].ToString(), 0755); err != nil {
			return ctx.ThrowError(fmt.Errorf("mkdir: %w", err))
		}
		return ctx.NewBool(true)
	}))

	ctx.Globals().Set("__go_fs_unlink", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("unlink: path argument required"))
		}
		if err := os.Remove(args[0].ToString()); err != nil {
			return ctx.ThrowError(fmt.Errorf("unlink: %w", err))
		}
		return ctx.NewBool(true)
	}))

	ctx.Globals().Set("__go_fs_rm", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("rm: path argument required"))
		}
		if err := os.RemoveAll(args[0].ToString()); err != nil {
			return ctx.ThrowError(fmt.Errorf("rm: %w", err))
		}
		return ctx.NewBool(true)
	}))

	return evalJS(ctx, `
globalThis.fs = {
  readFile(path, encoding) { return __go_fs_readFile(path, encoding || 'utf8'); },
  writeFile(path, data, encoding) { return __go_fs_writeFile(path, data, encoding || 'utf8'); },
  readdir(path) { return JSON.parse(__go_fs_readdir(path)); },
  stat(path) { return JSON.parse(__go_fs_stat(path)); },
  mkdir(path, opts) { return __go_fs_mkdir(path, opts && opts.recursive); },
  unlink(path) { return __go_fs_unlink(path); },
  rm(path, opts) { return (opts && opts.recursive) ? __go_fs_rm(path) : __go_fs_unlink(path); },
};
`)
}
