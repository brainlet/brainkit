package jsbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	quickjs "github.com/buke/quickjs-go"
)

// FSPolyfill provides async fs.readFile, fs.writeFile, fs.readdir, fs.stat, fs.mkdir, fs.unlink, fs.rm.
// All operations run in tracked goroutines via Bridge.Go().
type FSPolyfill struct {
	bridge *Bridge
}

// FS creates a file system polyfill.
func FS() *FSPolyfill { return &FSPolyfill{} }

func (p *FSPolyfill) Name() string { return "fs" }

func (p *FSPolyfill) SetBridge(b *Bridge) { p.bridge = b }

// fsErrCode maps Go os errors to Node.js errno codes.
// Mastra's LocalFilesystem checks error.code === 'ENOENT' etc.
func fsErrCode(err error) string {
	if os.IsNotExist(err) {
		return "ENOENT"
	}
	if os.IsExist(err) {
		return "EEXIST"
	}
	if os.IsPermission(err) {
		return "EACCES"
	}
	return ""
}

// fsAsync runs a function in a tracked goroutine and resolves/rejects a Promise.
// The rejected error has a .code property matching Node.js errno codes (ENOENT, EEXIST, etc.)
// so Mastra's error checks like isEnoentError(err) work correctly.
func (p *FSPolyfill) fsAsync(ctx *quickjs.Context, work func() (string, error)) *quickjs.Value {
	return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
		p.bridge.Go(func(goCtx context.Context) {
			result, err := work()
			if goCtx.Err() != nil {
				return
			}
			if err != nil {
				code := fsErrCode(err)
				ctx.Schedule(func(ctx *quickjs.Context) {
					errVal := ctx.NewError(err)
					if code != "" {
						codeVal := ctx.NewString(code)
						errVal.Set("code", codeVal)
						codeVal.Free()
					}
					defer errVal.Free()
					reject(errVal)
				})
				return
			}
			ctx.Schedule(func(ctx *quickjs.Context) {
				resolve(ctx.NewString(result))
			})
		})
	})
}

func (p *FSPolyfill) Setup(ctx *quickjs.Context) error {
	ctx.Globals().Set("__go_fs_readFile", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("readFile: path argument required"))
		}
		path := args[0].ToString()
		return p.fsAsync(ctx, func() (string, error) {
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
		return p.fsAsync(ctx, func() (string, error) {
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
		return p.fsAsync(ctx, func() (string, error) {
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
		return p.fsAsync(ctx, func() (string, error) {
			info, err := os.Stat(path)
			if err != nil {
				return "", fmt.Errorf("stat: %w", err)
			}
			b, err := json.Marshal(map[string]interface{}{
				"size":           info.Size(),
				"isFile":         info.Mode().IsRegular(),
				"isDirectory":    info.IsDir(),
				"isSymbolicLink": false,
				"mode":           int(info.Mode()),
				"mtimeMs":        info.ModTime().UnixMilli(),
				"modTime":        info.ModTime().Unix(),
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
		return p.fsAsync(ctx, func() (string, error) {
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
		return p.fsAsync(ctx, func() (string, error) {
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
		return p.fsAsync(ctx, func() (string, error) {
			if err := os.RemoveAll(path); err != nil {
				return "", fmt.Errorf("rm: %w", err)
			}
			return "true", nil
		})
	}))

	// lstat — like stat but doesn't follow symlinks
	ctx.Globals().Set("__go_fs_lstat", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("lstat: path argument required"))
		}
		path := args[0].ToString()
		return p.fsAsync(ctx, func() (string, error) {
			info, err := os.Lstat(path)
			if err != nil {
				return "", fmt.Errorf("lstat: %w", err)
			}
			isSymlink := info.Mode()&os.ModeSymlink != 0
			b, err := json.Marshal(map[string]interface{}{
				"size":           info.Size(),
				"isFile":         info.Mode().IsRegular(),
				"isDirectory":    info.IsDir(),
				"isSymbolicLink": isSymlink,
				"mode":           int(info.Mode()),
				"mtimeMs":        info.ModTime().UnixMilli(),
			})
			if err != nil {
				return "", fmt.Errorf("lstat: json: %w", err)
			}
			return string(b), nil
		})
	}))

	// copyFile — copy src to dest
	ctx.Globals().Set("__go_fs_copyFile", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return ctx.ThrowError(fmt.Errorf("copyFile: src and dest required"))
		}
		src := args[0].ToString()
		dest := args[1].ToString()
		return p.fsAsync(ctx, func() (string, error) {
			data, err := os.ReadFile(src)
			if err != nil {
				return "", fmt.Errorf("copyFile: read: %w", err)
			}
			srcInfo, _ := os.Stat(src)
			perm := os.FileMode(0644)
			if srcInfo != nil {
				perm = srcInfo.Mode().Perm()
			}
			if err := os.WriteFile(dest, data, perm); err != nil {
				return "", fmt.Errorf("copyFile: write: %w", err)
			}
			return "true", nil
		})
	}))

	// rename — move/rename a file
	ctx.Globals().Set("__go_fs_rename", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return ctx.ThrowError(fmt.Errorf("rename: old and new path required"))
		}
		oldPath := args[0].ToString()
		newPath := args[1].ToString()
		return p.fsAsync(ctx, func() (string, error) {
			if err := os.Rename(oldPath, newPath); err != nil {
				return "", fmt.Errorf("rename: %w", err)
			}
			return "true", nil
		})
	}))

	// realpath — resolve symlinks to real path
	ctx.Globals().Set("__go_fs_realpath", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("realpath: path argument required"))
		}
		path := args[0].ToString()
		return p.fsAsync(ctx, func() (string, error) {
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				return "", fmt.Errorf("realpath: %w", err)
			}
			abs, err := filepath.Abs(resolved)
			if err != nil {
				return "", fmt.Errorf("realpath: abs: %w", err)
			}
			return abs, nil
		})
	}))

	// access — check if path is accessible (throws if not)
	ctx.Globals().Set("__go_fs_access", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("access: path argument required"))
		}
		path := args[0].ToString()
		return p.fsAsync(ctx, func() (string, error) {
			if _, err := os.Stat(path); err != nil {
				return "", fmt.Errorf("access: %w", err)
			}
			return "true", nil
		})
	}))

	// appendFile — append data to a file
	ctx.Globals().Set("__go_fs_appendFile", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return ctx.ThrowError(fmt.Errorf("appendFile: path and data required"))
		}
		path := args[0].ToString()
		data := args[1].ToString()
		return p.fsAsync(ctx, func() (string, error) {
			f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return "", fmt.Errorf("appendFile: %w", err)
			}
			defer f.Close()
			if _, err := f.WriteString(data); err != nil {
				return "", fmt.Errorf("appendFile: write: %w", err)
			}
			return "true", nil
		})
	}))

	// symlink — create a symbolic link
	ctx.Globals().Set("__go_fs_symlink", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return ctx.ThrowError(fmt.Errorf("symlink: target and path required"))
		}
		target := args[0].ToString()
		linkPath := args[1].ToString()
		return p.fsAsync(ctx, func() (string, error) {
			if err := os.Symlink(target, linkPath); err != nil {
				return "", fmt.Errorf("symlink: %w", err)
			}
			return "true", nil
		})
	}))

	// readlink — read symlink target
	ctx.Globals().Set("__go_fs_readlink", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("readlink: path argument required"))
		}
		path := args[0].ToString()
		return p.fsAsync(ctx, func() (string, error) {
			target, err := os.Readlink(path)
			if err != nil {
				return "", fmt.Errorf("readlink: %w", err)
			}
			return target, nil
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
