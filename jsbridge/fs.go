package jsbridge

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fastschema/qjs"
)

// FSPolyfill provides fs.readFile, fs.writeFile, fs.readdir, fs.stat, fs.mkdir, fs.unlink, fs.rm.
type FSPolyfill struct{}

// FS creates a file system polyfill.
func FS() *FSPolyfill { return &FSPolyfill{} }

func (p *FSPolyfill) Name() string { return "fs" }

func (p *FSPolyfill) Setup(ctx *qjs.Context) error {
	ctx.SetFunc("__go_fs_readFile", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("readFile: path argument required")
		}
		data, err := os.ReadFile(args[0].String())
		if err != nil {
			return nil, fmt.Errorf("readFile: %w", err)
		}
		return this.Context().NewString(string(data)), nil
	})

	ctx.SetFunc("__go_fs_writeFile", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 2 {
			return nil, fmt.Errorf("writeFile: path and data arguments required")
		}
		if err := os.WriteFile(args[0].String(), []byte(args[1].String()), 0644); err != nil {
			return nil, fmt.Errorf("writeFile: %w", err)
		}
		return this.Context().NewBool(true), nil
	})

	ctx.SetFunc("__go_fs_readdir", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("readdir: path argument required")
		}
		entries, err := os.ReadDir(args[0].String())
		if err != nil {
			return nil, fmt.Errorf("readdir: %w", err)
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
			return nil, fmt.Errorf("readdir: json marshal: %w", err)
		}
		return this.Context().NewString(string(b)), nil
	})

	ctx.SetFunc("__go_fs_stat", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("stat: path argument required")
		}
		info, err := os.Stat(args[0].String())
		if err != nil {
			return nil, fmt.Errorf("stat: %w", err)
		}
		b, err := json.Marshal(map[string]interface{}{
			"size":        info.Size(),
			"isFile":      info.Mode().IsRegular(),
			"isDirectory": info.IsDir(),
			"modTime":     info.ModTime().Unix(),
		})
		if err != nil {
			return nil, fmt.Errorf("stat: json marshal: %w", err)
		}
		return this.Context().NewString(string(b)), nil
	})

	ctx.SetFunc("__go_fs_mkdir", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("mkdir: path argument required")
		}
		if err := os.MkdirAll(args[0].String(), 0755); err != nil {
			return nil, fmt.Errorf("mkdir: %w", err)
		}
		return this.Context().NewBool(true), nil
	})

	ctx.SetFunc("__go_fs_unlink", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("unlink: path argument required")
		}
		if err := os.Remove(args[0].String()); err != nil {
			return nil, fmt.Errorf("unlink: %w", err)
		}
		return this.Context().NewBool(true), nil
	})

	ctx.SetFunc("__go_fs_rm", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("rm: path argument required")
		}
		if err := os.RemoveAll(args[0].String()); err != nil {
			return nil, fmt.Errorf("rm: %w", err)
		}
		return this.Context().NewBool(true), nil
	})

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
