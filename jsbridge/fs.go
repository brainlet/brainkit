package jsbridge

import (
	"encoding/json"
	"fmt"
	"os"

	quickjs "github.com/buke/quickjs-go"
)

// FSPolyfill provides async fs.readFile, fs.writeFile, fs.readdir, fs.stat, fs.mkdir, fs.unlink, fs.rm.
// All operations run in separate goroutines — the bridge is NOT held during disk I/O.
type FSPolyfill struct{}

// FS creates a file system polyfill.
func FS() *FSPolyfill { return &FSPolyfill{} }

func (p *FSPolyfill) Name() string { return "fs" }

// fsAsync is a helper that runs a function in a goroutine and resolves/rejects a Promise.
func fsAsync(ctx *quickjs.Context, work func() (string, error)) *quickjs.Value {
	return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					ctx.Schedule(func(ctx *quickjs.Context) {
						errVal := ctx.NewError(fmt.Errorf("fs panic: %v", r))
						defer errVal.Free()
						reject(errVal)
					})
				}
			}()
			result, err := work()
			if err != nil {
				ctx.Schedule(func(ctx *quickjs.Context) {
					errVal := ctx.NewError(err)
					defer errVal.Free()
					reject(errVal)
				})
				return
			}
			ctx.Schedule(func(ctx *quickjs.Context) {
				resolve(ctx.NewString(result))
			})
		}()
	})
}

func (p *FSPolyfill) Setup(ctx *quickjs.Context) error {
	ctx.Globals().Set("__go_fs_readFile", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("readFile: path argument required"))
		}
		path := args[0].ToString()
		return fsAsync(ctx, func() (string, error) {
			data, err := os.ReadFile(path)
			if err != nil {
				return "", fmt.Errorf("readFile: %w", err)
			}
			return string(data), nil
		})
	}))

	ctx.Globals().Set("__go_fs_writeFile", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return ctx.ThrowError(fmt.Errorf("writeFile: path and data arguments required"))
		}
		path := args[0].ToString()
		data := args[1].ToString()
		return fsAsync(ctx, func() (string, error) {
			if err := os.WriteFile(path, []byte(data), 0644); err != nil {
				return "", fmt.Errorf("writeFile: %w", err)
			}
			return "true", nil
		})
	}))

	ctx.Globals().Set("__go_fs_readdir", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("readdir: path argument required"))
		}
		path := args[0].ToString()
		return fsAsync(ctx, func() (string, error) {
			entries, err := os.ReadDir(path)
			if err != nil {
				return "", fmt.Errorf("readdir: %w", err)
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
				return "", fmt.Errorf("readdir: json marshal: %w", err)
			}
			return string(b), nil
		})
	}))

	ctx.Globals().Set("__go_fs_stat", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("stat: path argument required"))
		}
		path := args[0].ToString()
		return fsAsync(ctx, func() (string, error) {
			info, err := os.Stat(path)
			if err != nil {
				return "", fmt.Errorf("stat: %w", err)
			}
			b, err := json.Marshal(map[string]interface{}{
				"size":        info.Size(),
				"isFile":      info.Mode().IsRegular(),
				"isDirectory": info.IsDir(),
				"modTime":     info.ModTime().Unix(),
			})
			if err != nil {
				return "", fmt.Errorf("stat: json marshal: %w", err)
			}
			return string(b), nil
		})
	}))

	ctx.Globals().Set("__go_fs_mkdir", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("mkdir: path argument required"))
		}
		path := args[0].ToString()
		return fsAsync(ctx, func() (string, error) {
			if err := os.MkdirAll(path, 0755); err != nil {
				return "", fmt.Errorf("mkdir: %w", err)
			}
			return "true", nil
		})
	}))

	ctx.Globals().Set("__go_fs_unlink", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("unlink: path argument required"))
		}
		path := args[0].ToString()
		return fsAsync(ctx, func() (string, error) {
			if err := os.Remove(path); err != nil {
				return "", fmt.Errorf("unlink: %w", err)
			}
			return "true", nil
		})
	}))

	ctx.Globals().Set("__go_fs_rm", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("rm: path argument required"))
		}
		path := args[0].ToString()
		return fsAsync(ctx, func() (string, error) {
			if err := os.RemoveAll(path); err != nil {
				return "", fmt.Errorf("rm: %w", err)
			}
			return "true", nil
		})
	}))

	return evalJS(ctx, `
globalThis.fs = {
  async readFile(path, encoding) { return await __go_fs_readFile(path, encoding || 'utf8'); },
  async writeFile(path, data, encoding) { return (await __go_fs_writeFile(path, data, encoding || 'utf8')) === 'true'; },
  async readdir(path) { return JSON.parse(await __go_fs_readdir(path)); },
  async stat(path) { return JSON.parse(await __go_fs_stat(path)); },
  async mkdir(path, opts) { return (await __go_fs_mkdir(path, opts && opts.recursive)) === 'true'; },
  async unlink(path) { return (await __go_fs_unlink(path)) === 'true'; },
  async rm(path, opts) { return (opts && opts.recursive) ? (await __go_fs_rm(path)) === 'true' : (await __go_fs_unlink(path)) === 'true'; },
};
`)
}
