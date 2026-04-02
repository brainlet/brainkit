package jsbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	quickjs "github.com/buke/quickjs-go"
)

// FSPolyfill provides the complete Node.js 22 fs module as a jsbridge polyfill.
// All path operations resolve via workspace root with escape protection.
// Sync methods are direct Go calls. Async methods use bridge.Go() + Promise.
type FSPolyfill struct {
	bridge *Bridge
	root   string // workspace root — all paths resolve relative to this

	// Track open file handles for auto-close on shutdown
	handlesMu sync.Mutex
	handles   map[int]*os.File
	nextFD    int

	// Track watchers for auto-close on shutdown
	watchersMu sync.Mutex
	watchers   []*fsnotify.Watcher
}

// FS creates a filesystem polyfill scoped to the given workspace root.
// If root is empty, fs operations will throw "workspace not configured".
func FS(root string) *FSPolyfill {
	return &FSPolyfill{
		root:    root,
		handles: make(map[int]*os.File),
		nextFD:  10, // start above stdin/stdout/stderr
	}
}

func (p *FSPolyfill) Name() string { return "fs" }

func (p *FSPolyfill) SetBridge(b *Bridge) { p.bridge = b }

// ---------------------------------------------------------------------------
// Path resolution — workspace escape protection
// ---------------------------------------------------------------------------

func (p *FSPolyfill) resolve(userPath string) (string, error) {
	if p.root == "" {
		return "", fmt.Errorf("workspace not configured")
	}
	abs := filepath.Join(p.root, filepath.Clean("/"+userPath))
	cleanRoot := filepath.Clean(p.root)
	if abs != cleanRoot && !strings.HasPrefix(abs, cleanRoot+string(filepath.Separator)) {
		return "", &fsError{code: "EACCES", syscall: "open", path: userPath, message: fmt.Sprintf("path %q escapes workspace", userPath)}
	}
	return abs, nil
}

// ---------------------------------------------------------------------------
// Node.js error codes — Go os errors → {code, errno, syscall, path}
// ---------------------------------------------------------------------------

type fsError struct {
	code    string // ENOENT, EACCES, EEXIST, EISDIR, ENOTDIR, ENOTEMPTY
	errno   int
	syscall string
	path    string
	message string
}

func (e *fsError) Error() string { return e.message }

// mapError converts a Go os error to a Node.js-style fsError.
func mapError(err error, sc, path string) error {
	if err == nil {
		return nil
	}
	// Unwrap to root cause
	root := err
	for {
		if u, ok := root.(interface{ Unwrap() error }); ok {
			root = u.Unwrap()
		} else {
			break
		}
	}

	code := ""
	errno := 0
	switch {
	case os.IsNotExist(root):
		code, errno = "ENOENT", -2
	case os.IsPermission(root):
		code, errno = "EACCES", -13
	case os.IsExist(root):
		code, errno = "EEXIST", -17
	default:
		// Check errno strings from the underlying error for platform-agnostic matching.
		// os.PathError.Err contains the syscall errno; its Error() gives "is a directory", etc.
		errStr := strings.ToLower(root.Error())
		switch {
		case strings.Contains(errStr, "is a directory"):
			code, errno = "EISDIR", -21
		case strings.Contains(errStr, "not a directory"):
			code, errno = "ENOTDIR", -20
		case strings.Contains(errStr, "directory not empty"):
			code, errno = "ENOTEMPTY", -66
		}
	}
	if code == "" {
		// Unknown error — return as-is with no code
		return &fsError{code: "", syscall: sc, path: path, message: err.Error()}
	}
	return &fsError{
		code:    code,
		errno:   errno,
		syscall: sc,
		path:    path,
		message: fmt.Sprintf("%s: %s, %s '%s'", code, sc, err.Error(), path),
	}
}

// throwFSError creates a JS Error with .code, .errno, .syscall, .path properties.
func throwFSError(qctx *quickjs.Context, err error, sc, path string) *quickjs.Value {
	mapped := mapError(err, sc, path)
	fe, ok := mapped.(*fsError)
	if !ok || fe.code == "" {
		return qctx.ThrowError(err)
	}
	script := fmt.Sprintf(`(function() {
		var e = new Error(%q);
		e.code = %q;
		e.errno = %d;
		e.syscall = %q;
		e.path = %q;
		return e;
	})()`, fe.message, fe.code, fe.errno, fe.syscall, fe.path)
	errVal := qctx.Eval(script)
	if errVal.IsException() {
		return qctx.ThrowError(err)
	}
	return qctx.Throw(errVal)
}

// scheduleReject rejects a Promise with a Node.js-style fs error.
func scheduleReject(qctx *quickjs.Context, reject func(*quickjs.Value), err error, sc, path string) {
	mapped := mapError(err, sc, path)
	fe, ok := mapped.(*fsError)
	msg := err.Error()
	code := ""
	errno := 0
	if ok && fe.code != "" {
		msg = fe.message
		code = fe.code
		errno = fe.errno
	}
	qctx.Schedule(func(qctx *quickjs.Context) {
		if code != "" {
			script := fmt.Sprintf(`(function() {
				var e = new Error(%q);
				e.code = %q;
				e.errno = %d;
				e.syscall = %q;
				e.path = %q;
				return e;
			})()`, msg, code, errno, sc, path)
			errVal := qctx.Eval(script)
			if !errVal.IsException() {
				defer errVal.Free()
				reject(errVal)
				return
			}
			errVal.Free()
		}
		errVal := qctx.NewError(fmt.Errorf("%s", msg))
		defer errVal.Free()
		reject(errVal)
	})
}

// ---------------------------------------------------------------------------
// Stats builder — os.FileInfo → JS Stats object
// ---------------------------------------------------------------------------

func buildStatsJSON(info os.FileInfo) string {
	mode := info.Mode()
	isSymlink := mode&os.ModeSymlink != 0
	isBlock := mode&os.ModeDevice != 0 && mode&os.ModeCharDevice == 0
	isChar := mode&os.ModeCharDevice != 0
	isFIFO := mode&os.ModeNamedPipe != 0
	isSocket := mode&os.ModeSocket != 0

	mtimeMs := info.ModTime().UnixMilli()
	atimeMs := mtimeMs
	ctimeMs := mtimeMs
	birthtimeMs := mtimeMs

	// Platform-specific: extract real atime/ctime/birthtime from syscall.Stat_t
	if a, c, bt, ok := platformStatTimes(info.Sys()); ok {
		atimeMs = a
		ctimeMs = c
		birthtimeMs = bt
	}

	stats := map[string]any{
		"dev":         0,
		"ino":         0,
		"mode":        int(mode.Perm()) | unixFileType(mode),
		"nlink":       1,
		"uid":         0,
		"gid":         0,
		"rdev":        0,
		"size":        info.Size(),
		"blksize":     4096,
		"blocks":      (info.Size() + 511) / 512,
		"atimeMs":     float64(atimeMs),
		"mtimeMs":     float64(mtimeMs),
		"ctimeMs":     float64(ctimeMs),
		"birthtimeMs": float64(birthtimeMs),
		"atime":       time.UnixMilli(atimeMs).UTC().Format(time.RFC3339Nano),
		"mtime":       info.ModTime().UTC().Format(time.RFC3339Nano),
		"ctime":       time.UnixMilli(ctimeMs).UTC().Format(time.RFC3339Nano),
		"birthtime":   time.UnixMilli(birthtimeMs).UTC().Format(time.RFC3339Nano),
		"_isFile":         mode.IsRegular(),
		"_isDirectory":    info.IsDir(),
		"_isSymbolicLink": isSymlink,
		"_isBlockDevice":  isBlock,
		"_isCharDevice":   isChar,
		"_isFIFO":         isFIFO,
		"_isSocket":       isSocket,
	}

	// Platform-specific: extract uid/gid/dev/ino/nlink/rdev/blksize/blocks
	if uid, gid, dev, ino, nlink, rdev, blksize, blocks, ok := platformStatFields(info.Sys()); ok {
		stats["dev"] = dev
		stats["ino"] = ino
		stats["nlink"] = nlink
		stats["uid"] = uid
		stats["gid"] = gid
		stats["rdev"] = rdev
		stats["blksize"] = blksize
		stats["blocks"] = blocks
	}

	b, _ := json.Marshal(stats)
	return string(b)
}

func unixFileType(mode os.FileMode) int {
	switch {
	case mode.IsRegular():
		return 0o100000 // S_IFREG
	case mode.IsDir():
		return 0o040000 // S_IFDIR
	case mode&os.ModeSymlink != 0:
		return 0o120000 // S_IFLNK
	case mode&os.ModeNamedPipe != 0:
		return 0o010000 // S_IFIFO
	case mode&os.ModeSocket != 0:
		return 0o140000 // S_IFSOCK
	case mode&os.ModeCharDevice != 0:
		return 0o020000 // S_IFCHR
	case mode&os.ModeDevice != 0:
		return 0o060000 // S_IFBLK
	default:
		return 0
	}
}

// ---------------------------------------------------------------------------
// Dirent builder — os.DirEntry → JSON
// ---------------------------------------------------------------------------

func buildDirentJSON(entry os.DirEntry, parentPath string) string {
	info, _ := entry.Info()
	isSymlink := entry.Type()&os.ModeSymlink != 0
	isBlock := entry.Type()&os.ModeDevice != 0 && entry.Type()&os.ModeCharDevice == 0
	isChar := entry.Type()&os.ModeCharDevice != 0
	isFIFO := entry.Type()&os.ModeNamedPipe != 0
	isSocket := entry.Type()&os.ModeSocket != 0
	isFile := false
	if info != nil {
		isFile = info.Mode().IsRegular()
	}

	d := map[string]any{
		"name":              entry.Name(),
		"parentPath":        parentPath,
		"_isFile":           isFile,
		"_isDirectory":      entry.IsDir(),
		"_isSymbolicLink":   isSymlink,
		"_isBlockDevice":    isBlock,
		"_isCharacterDevice": isChar,
		"_isFIFO":           isFIFO,
		"_isSocket":         isSocket,
	}
	b, _ := json.Marshal(d)
	return string(b)
}

// ---------------------------------------------------------------------------
// Flag parsing — "r", "w", "a", etc. → os flags
// ---------------------------------------------------------------------------

func parseFlags(flag string) int {
	switch flag {
	case "r":
		return os.O_RDONLY
	case "r+":
		return os.O_RDWR
	case "w":
		return os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	case "w+":
		return os.O_RDWR | os.O_CREATE | os.O_TRUNC
	case "a":
		return os.O_WRONLY | os.O_CREATE | os.O_APPEND
	case "a+":
		return os.O_RDWR | os.O_CREATE | os.O_APPEND
	case "ax":
		return os.O_WRONLY | os.O_CREATE | os.O_EXCL
	case "ax+":
		return os.O_RDWR | os.O_CREATE | os.O_EXCL
	case "wx":
		return os.O_WRONLY | os.O_CREATE | os.O_EXCL
	case "wx+":
		return os.O_RDWR | os.O_CREATE | os.O_EXCL
	default:
		return os.O_RDONLY
	}
}

// getEncoding extracts encoding from options arg (string or {encoding: string}).
func getEncoding(val *quickjs.Value) string {
	if val == nil || val.IsUndefined() || val.IsNull() {
		return ""
	}
	s := val.String()
	if s == "utf8" || s == "utf-8" || s == "ascii" || s == "latin1" || s == "hex" || s == "base64" {
		return s
	}
	// Might be an options object {encoding: "utf8"}
	if strings.HasPrefix(s, "{") {
		var opts struct {
			Encoding string `json:"encoding"`
		}
		if json.Unmarshal([]byte(s), &opts) == nil && opts.Encoding != "" {
			return opts.Encoding
		}
	}
	return s
}

// parseOpts unmarshals a QuickJS value as a JSON options object.
// Returns nil fields as zero values — callers check specific fields.
type fsOpts struct {
	WithFileTypes bool `json:"withFileTypes"`
	Recursive     bool `json:"recursive"`
	Force         bool `json:"force"`
}

func parseOpts(val *quickjs.Value) fsOpts {
	if val == nil || val.IsUndefined() || val.IsNull() {
		return fsOpts{}
	}
	s := val.String()
	var opts fsOpts
	json.Unmarshal([]byte(s), &opts)
	return opts
}

// ---------------------------------------------------------------------------
// FileHandle tracking
// ---------------------------------------------------------------------------

func (p *FSPolyfill) trackHandle(f *os.File) int {
	p.handlesMu.Lock()
	fd := p.nextFD
	p.nextFD++
	p.handles[fd] = f
	p.handlesMu.Unlock()
	return fd
}

func (p *FSPolyfill) getHandle(fd int) *os.File {
	p.handlesMu.Lock()
	defer p.handlesMu.Unlock()
	return p.handles[fd]
}

func (p *FSPolyfill) closeHandle(fd int) error {
	p.handlesMu.Lock()
	f, ok := p.handles[fd]
	if ok {
		delete(p.handles, fd)
	}
	p.handlesMu.Unlock()
	if ok && f != nil {
		return f.Close()
	}
	return nil
}

func (p *FSPolyfill) closeAllHandles() {
	p.handlesMu.Lock()
	for fd, f := range p.handles {
		f.Close()
		delete(p.handles, fd)
	}
	p.handlesMu.Unlock()
}

func (p *FSPolyfill) closeAllWatchers() {
	p.watchersMu.Lock()
	for _, w := range p.watchers {
		w.Close()
	}
	p.watchers = nil
	p.watchersMu.Unlock()
}

// ---------------------------------------------------------------------------
// Recursive copy helper
// ---------------------------------------------------------------------------

func recursiveCopy(src, dst string) error {
	// WalkDir doesn't follow symlinks — we can detect them properly.
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)

		if d.IsDir() {
			info, _ := d.Info()
			perm := os.FileMode(0o755)
			if info != nil {
				perm = info.Mode().Perm()
			}
			return os.MkdirAll(target, perm)
		}
		if d.Type()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}

// ---------------------------------------------------------------------------
// Setup — registers globalThis.fs with all methods
// ---------------------------------------------------------------------------

func (p *FSPolyfill) Setup(ctx *quickjs.Context) error {
	b := p.bridge
	root := p.root

	// Inject fs.constants from Go — matches actual OS values for the current platform
	constantsJSON, _ := json.Marshal(map[string]int{
		"F_OK": 0, "R_OK": 4, "W_OK": 2, "X_OK": 1,
		"COPYFILE_EXCL": 1, "COPYFILE_FICLONE": 2, "COPYFILE_FICLONE_FORCE": 4,
		"O_RDONLY": os.O_RDONLY, "O_WRONLY": os.O_WRONLY, "O_RDWR": os.O_RDWR,
		"O_CREAT": os.O_CREATE, "O_EXCL": os.O_EXCL, "O_TRUNC": os.O_TRUNC, "O_APPEND": os.O_APPEND,
	})
	ctx.Globals().Set("__go_fs_constants_json", ctx.NewString(string(constantsJSON)))
	// Register all Go bridge functions
	p.registerSyncBridges(ctx, root)
	p.registerAsyncBridges(ctx, b, root)
	p.registerFileHandleBridges(ctx, b, root)
	p.registerStreamBridges(ctx, b, root)
	p.registerWatchBridges(ctx, b, root)

	// Register cleanup goroutine — closes file handles and watchers when bridge shuts down.
	// bridge.Go tracks this via wg, and goCtx is cancelled on Close().
	b.Go(func(goCtx context.Context) {
		<-goCtx.Done()
		p.closeAllHandles()
		p.closeAllWatchers()
	})

	// Build globalThis.fs with JS wrappers
	return evalJS(ctx, fsSetupJS)
}

// ---------------------------------------------------------------------------
// Sync bridge functions — direct Go calls, no goroutines
// ---------------------------------------------------------------------------

func (p *FSPolyfill) registerSyncBridges(ctx *quickjs.Context, root string) {
	resolve := func(userPath string) (string, error) { return p.resolve(userPath) }

	// readFileSync(path, options?) → string
	ctx.Globals().Set("__go_fs_readFileSync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("readFileSync: path required"))
		}
		userPath := args[0].String()
		absPath, err := resolve(userPath)
		if err != nil {
			return throwFSError(qctx, err, "open", userPath)
		}
		data, err := os.ReadFile(absPath)
		if err != nil {
			return throwFSError(qctx, err, "read", userPath)
		}
		enc := ""
		if len(args) > 1 {
			enc = getEncoding(args[1])
		}
		if enc == "utf8" || enc == "utf-8" || enc == "" {
			return qctx.NewString(string(data))
		}
		return qctx.NewString(string(data))
	}))

	// writeFileSync(path, data, options?)
	ctx.Globals().Set("__go_fs_writeFileSync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return qctx.ThrowError(fmt.Errorf("writeFileSync: path and data required"))
		}
		userPath := args[0].String()
		data := args[1].String()
		absPath, err := resolve(userPath)
		if err != nil {
			return throwFSError(qctx, err, "open", userPath)
		}
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			return throwFSError(qctx, err, "mkdir", userPath)
		}
		if err := os.WriteFile(absPath, []byte(data), 0o644); err != nil {
			return throwFSError(qctx, err, "write", userPath)
		}
		return qctx.NewUndefined()
	}))

	// appendFileSync(path, data, options?)
	ctx.Globals().Set("__go_fs_appendFileSync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return qctx.ThrowError(fmt.Errorf("appendFileSync: path and data required"))
		}
		userPath := args[0].String()
		data := args[1].String()
		absPath, err := resolve(userPath)
		if err != nil {
			return throwFSError(qctx, err, "open", userPath)
		}
		f, err := os.OpenFile(absPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return throwFSError(qctx, err, "open", userPath)
		}
		defer f.Close()
		if _, err := f.WriteString(data); err != nil {
			return throwFSError(qctx, err, "write", userPath)
		}
		return qctx.NewUndefined()
	}))

	// readdirSync(path, options?) → string[] or Dirent[]
	ctx.Globals().Set("__go_fs_readdirSync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("readdirSync: path required"))
		}
		userPath := args[0].String()
		absPath, err := resolve(userPath)
		if err != nil {
			return throwFSError(qctx, err, "scandir", userPath)
		}
		entries, err := os.ReadDir(absPath)
		if err != nil {
			return throwFSError(qctx, err, "scandir", userPath)
		}
		opts := fsOpts{}
		if len(args) > 1 {
			opts = parseOpts(args[1])
		}
		withFileTypes := opts.WithFileTypes
		if withFileTypes {
			var dirents []string
			for _, e := range entries {
				dirents = append(dirents, buildDirentJSON(e, userPath))
			}
			return qctx.NewString("[" + strings.Join(dirents, ",") + "]")
		}
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		b, _ := json.Marshal(names)
		return qctx.NewString(string(b))
	}))

	// statSync(path, options?) → Stats JSON
	ctx.Globals().Set("__go_fs_statSync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("statSync: path required"))
		}
		userPath := args[0].String()
		absPath, err := resolve(userPath)
		if err != nil {
			return throwFSError(qctx, err, "stat", userPath)
		}
		info, err := os.Stat(absPath)
		if err != nil {
			return throwFSError(qctx, err, "stat", userPath)
		}
		return qctx.NewString(buildStatsJSON(info))
	}))

	// lstatSync(path, options?) → Stats JSON
	ctx.Globals().Set("__go_fs_lstatSync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("lstatSync: path required"))
		}
		userPath := args[0].String()
		absPath, err := resolve(userPath)
		if err != nil {
			return throwFSError(qctx, err, "lstat", userPath)
		}
		info, err := os.Lstat(absPath)
		if err != nil {
			return throwFSError(qctx, err, "lstat", userPath)
		}
		return qctx.NewString(buildStatsJSON(info))
	}))

	// accessSync(path, mode?)
	ctx.Globals().Set("__go_fs_accessSync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("accessSync: path required"))
		}
		userPath := args[0].String()
		absPath, err := resolve(userPath)
		if err != nil {
			return throwFSError(qctx, err, "access", userPath)
		}
		if _, err := os.Stat(absPath); err != nil {
			return throwFSError(qctx, err, "access", userPath)
		}
		return qctx.NewUndefined()
	}))

	// mkdirSync(path, options?)
	ctx.Globals().Set("__go_fs_mkdirSync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("mkdirSync: path required"))
		}
		userPath := args[0].String()
		absPath, err := resolve(userPath)
		if err != nil {
			return throwFSError(qctx, err, "mkdir", userPath)
		}
		opts := fsOpts{}
		if len(args) > 1 {
			opts = parseOpts(args[1])
		}
		recursive := opts.Recursive
		if recursive {
			err = os.MkdirAll(absPath, 0o755)
		} else {
			err = os.Mkdir(absPath, 0o755)
		}
		if err != nil {
			return throwFSError(qctx, err, "mkdir", userPath)
		}
		if recursive {
			return qctx.NewString(userPath)
		}
		return qctx.NewUndefined()
	}))

	// mkdtempSync(prefix, options?) → string
	ctx.Globals().Set("__go_fs_mkdtempSync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		prefix := ""
		if len(args) > 0 {
			prefix = args[0].String()
		}
		absPrefix := prefix
		if root != "" {
			absPrefix = filepath.Join(root, filepath.Clean("/"+prefix))
		}
		dir, err := os.MkdirTemp(filepath.Dir(absPrefix), filepath.Base(absPrefix))
		if err != nil {
			return throwFSError(qctx, err, "mkdtemp", prefix)
		}
		// Return path relative to root
		if root != "" {
			rel, _ := filepath.Rel(root, dir)
			return qctx.NewString(rel)
		}
		return qctx.NewString(dir)
	}))

	// rmdirSync(path, options?)
	ctx.Globals().Set("__go_fs_rmdirSync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("rmdirSync: path required"))
		}
		userPath := args[0].String()
		absPath, err := resolve(userPath)
		if err != nil {
			return throwFSError(qctx, err, "rmdir", userPath)
		}
		if err := os.Remove(absPath); err != nil {
			return throwFSError(qctx, err, "rmdir", userPath)
		}
		return qctx.NewUndefined()
	}))

	// rmSync(path, options?)
	ctx.Globals().Set("__go_fs_rmSync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("rmSync: path required"))
		}
		userPath := args[0].String()
		absPath, err := resolve(userPath)
		if err != nil {
			return throwFSError(qctx, err, "rm", userPath)
		}
		opts := fsOpts{}
		if len(args) > 1 {
			opts = parseOpts(args[1])
		}
		if opts.Recursive {
			err = os.RemoveAll(absPath)
		} else {
			err = os.Remove(absPath)
		}
		if err != nil && !opts.Force {
			return throwFSError(qctx, err, "rm", userPath)
		}
		return qctx.NewUndefined()
	}))

	// unlinkSync(path)
	ctx.Globals().Set("__go_fs_unlinkSync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("unlinkSync: path required"))
		}
		userPath := args[0].String()
		absPath, err := resolve(userPath)
		if err != nil {
			return throwFSError(qctx, err, "unlink", userPath)
		}
		if err := os.Remove(absPath); err != nil {
			return throwFSError(qctx, err, "unlink", userPath)
		}
		return qctx.NewUndefined()
	}))

	// renameSync(oldPath, newPath)
	ctx.Globals().Set("__go_fs_renameSync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return qctx.ThrowError(fmt.Errorf("renameSync: oldPath and newPath required"))
		}
		oldUser := args[0].String()
		newUser := args[1].String()
		oldAbs, err := resolve(oldUser)
		if err != nil {
			return throwFSError(qctx, err, "rename", oldUser)
		}
		newAbs, err := resolve(newUser)
		if err != nil {
			return throwFSError(qctx, err, "rename", newUser)
		}
		if err := os.Rename(oldAbs, newAbs); err != nil {
			return throwFSError(qctx, err, "rename", oldUser)
		}
		return qctx.NewUndefined()
	}))

	// copyFileSync(src, dest, mode?)
	ctx.Globals().Set("__go_fs_copyFileSync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return qctx.ThrowError(fmt.Errorf("copyFileSync: src and dest required"))
		}
		srcUser := args[0].String()
		dstUser := args[1].String()
		srcAbs, err := resolve(srcUser)
		if err != nil {
			return throwFSError(qctx, err, "copyfile", srcUser)
		}
		dstAbs, err := resolve(dstUser)
		if err != nil {
			return throwFSError(qctx, err, "copyfile", dstUser)
		}
		in, err := os.Open(srcAbs)
		if err != nil {
			return throwFSError(qctx, err, "copyfile", srcUser)
		}
		defer in.Close()
		out, err := os.Create(dstAbs)
		if err != nil {
			return throwFSError(qctx, err, "copyfile", dstUser)
		}
		defer out.Close()
		if _, err := io.Copy(out, in); err != nil {
			return throwFSError(qctx, err, "copyfile", srcUser)
		}
		return qctx.NewUndefined()
	}))

	// cpSync(src, dest, options?) — recursive copy
	ctx.Globals().Set("__go_fs_cpSync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return qctx.ThrowError(fmt.Errorf("cpSync: src and dest required"))
		}
		srcUser := args[0].String()
		dstUser := args[1].String()
		srcAbs, err := resolve(srcUser)
		if err != nil {
			return throwFSError(qctx, err, "cp", srcUser)
		}
		dstAbs, err := resolve(dstUser)
		if err != nil {
			return throwFSError(qctx, err, "cp", dstUser)
		}
		if err := recursiveCopy(srcAbs, dstAbs); err != nil {
			return throwFSError(qctx, err, "cp", srcUser)
		}
		return qctx.NewUndefined()
	}))

	// linkSync(existingPath, newPath)
	ctx.Globals().Set("__go_fs_linkSync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return qctx.ThrowError(fmt.Errorf("linkSync: existingPath and newPath required"))
		}
		existing, err := resolve(args[0].String())
		if err != nil {
			return throwFSError(qctx, err, "link", args[0].String())
		}
		newPath, err := resolve(args[1].String())
		if err != nil {
			return throwFSError(qctx, err, "link", args[1].String())
		}
		if err := os.Link(existing, newPath); err != nil {
			return throwFSError(qctx, err, "link", args[0].String())
		}
		return qctx.NewUndefined()
	}))

	// symlinkSync(target, path, type?)
	ctx.Globals().Set("__go_fs_symlinkSync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return qctx.ThrowError(fmt.Errorf("symlinkSync: target and path required"))
		}
		target := args[0].String()
		linkPath, err := resolve(args[1].String())
		if err != nil {
			return throwFSError(qctx, err, "symlink", args[1].String())
		}
		if err := os.Symlink(target, linkPath); err != nil {
			return throwFSError(qctx, err, "symlink", args[1].String())
		}
		return qctx.NewUndefined()
	}))

	// readlinkSync(path, options?) → string
	ctx.Globals().Set("__go_fs_readlinkSync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("readlinkSync: path required"))
		}
		userPath := args[0].String()
		absPath, err := resolve(userPath)
		if err != nil {
			return throwFSError(qctx, err, "readlink", userPath)
		}
		target, err := os.Readlink(absPath)
		if err != nil {
			return throwFSError(qctx, err, "readlink", userPath)
		}
		return qctx.NewString(target)
	}))

	// realpathSync(path, options?) → string
	ctx.Globals().Set("__go_fs_realpathSync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("realpathSync: path required"))
		}
		userPath := args[0].String()
		absPath, err := resolve(userPath)
		if err != nil {
			return throwFSError(qctx, err, "realpath", userPath)
		}
		resolved, err := filepath.EvalSymlinks(absPath)
		if err != nil {
			return throwFSError(qctx, err, "realpath", userPath)
		}
		// Return relative to root
		if root != "" {
			rel, _ := filepath.Rel(root, resolved)
			return qctx.NewString(rel)
		}
		return qctx.NewString(resolved)
	}))

	// chmodSync(path, mode)
	ctx.Globals().Set("__go_fs_chmodSync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return qctx.ThrowError(fmt.Errorf("chmodSync: path and mode required"))
		}
		userPath := args[0].String()
		mode := os.FileMode(args[1].ToInt32())
		absPath, err := resolve(userPath)
		if err != nil {
			return throwFSError(qctx, err, "chmod", userPath)
		}
		if err := os.Chmod(absPath, mode); err != nil {
			return throwFSError(qctx, err, "chmod", userPath)
		}
		return qctx.NewUndefined()
	}))

	// chownSync(path, uid, gid)
	ctx.Globals().Set("__go_fs_chownSync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 3 {
			return qctx.ThrowError(fmt.Errorf("chownSync: path, uid, gid required"))
		}
		userPath := args[0].String()
		uid := int(args[1].ToInt32())
		gid := int(args[2].ToInt32())
		absPath, err := resolve(userPath)
		if err != nil {
			return throwFSError(qctx, err, "chown", userPath)
		}
		if err := os.Chown(absPath, uid, gid); err != nil {
			return throwFSError(qctx, err, "chown", userPath)
		}
		return qctx.NewUndefined()
	}))

	// truncateSync(path, len?)
	ctx.Globals().Set("__go_fs_truncateSync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("truncateSync: path required"))
		}
		userPath := args[0].String()
		size := int64(0)
		if len(args) > 1 {
			size = args[1].ToInt64()
		}
		absPath, err := resolve(userPath)
		if err != nil {
			return throwFSError(qctx, err, "truncate", userPath)
		}
		if err := os.Truncate(absPath, size); err != nil {
			return throwFSError(qctx, err, "truncate", userPath)
		}
		return qctx.NewUndefined()
	}))

	// utimesSync(path, atime, mtime)
	ctx.Globals().Set("__go_fs_utimesSync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 3 {
			return qctx.ThrowError(fmt.Errorf("utimesSync: path, atime, mtime required"))
		}
		userPath := args[0].String()
		atime := time.Unix(int64(args[1].ToFloat64()), 0)
		mtime := time.Unix(int64(args[2].ToFloat64()), 0)
		absPath, err := resolve(userPath)
		if err != nil {
			return throwFSError(qctx, err, "utimes", userPath)
		}
		if err := os.Chtimes(absPath, atime, mtime); err != nil {
			return throwFSError(qctx, err, "utimes", userPath)
		}
		return qctx.NewUndefined()
	}))

	// existsSync(path) → bool — Node.js legacy, still widely used
	ctx.Globals().Set("__go_fs_existsSync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.NewBool(false)
		}
		userPath := args[0].String()
		absPath, err := resolve(userPath)
		if err != nil {
			return qctx.NewBool(false)
		}
		_, err = os.Stat(absPath)
		return qctx.NewBool(err == nil)
	}))
}

// ---------------------------------------------------------------------------
// Async bridge functions — bridge.Go() + Promise
// ---------------------------------------------------------------------------

func (p *FSPolyfill) registerAsyncBridges(ctx *quickjs.Context, b *Bridge, root string) {
	resolve := func(userPath string) (string, error) { return p.resolve(userPath) }

	// __go_fs_async_op(op, argsJSON) → Promise<resultJSON>
	// Single bridge function for all async fs.promises operations to reduce Go function count.
	ctx.Globals().Set("__go_fs_async", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return qctx.ThrowError(fmt.Errorf("fs async: op and args required"))
		}
		op := args[0].String()
		argsJSON := args[1].String()

		return qctx.NewPromise(func(resolveP, reject func(*quickjs.Value)) {
			b.Go(func(goCtx context.Context) {
				result, err := p.execAsyncOp(op, argsJSON, resolve)
				if goCtx.Err() != nil {
					return
				}
				if err != nil {
					// Extract path from args for error
					var a struct{ Path string `json:"path"` }
					json.Unmarshal([]byte(argsJSON), &a)
					scheduleReject(qctx, reject, err, op, a.Path)
					return
				}
				qctx.Schedule(func(qctx *quickjs.Context) {
					resolveP(qctx.NewString(result))
				})
			})
		})
	}))
}

// execAsyncOp dispatches async fs operations. Returns JSON string result.
func (p *FSPolyfill) execAsyncOp(op, argsJSON string, resolve func(string) (string, error)) (string, error) {
	var args struct {
		Path     string `json:"path"`
		Data     string `json:"data"`
		Dest     string `json:"dest"`
		OldPath  string `json:"oldPath"`
		NewPath  string `json:"newPath"`
		Target   string `json:"target"`
		Mode     int    `json:"mode"`
		UID      int    `json:"uid"`
		GID      int    `json:"gid"`
		Len      int64  `json:"len"`
		Atime    float64 `json:"atime"`
		Mtime    float64 `json:"mtime"`
		Encoding string `json:"encoding"`
		Recursive bool  `json:"recursive"`
		Force     bool  `json:"force"`
		Prefix    string `json:"prefix"`
		WithFileTypes bool `json:"withFileTypes"`
		Flags    string `json:"flags"`
		FileMode int    `json:"fileMode"`
	}
	json.Unmarshal([]byte(argsJSON), &args)

	switch op {
	case "readFile":
		absPath, err := resolve(args.Path)
		if err != nil {
			return "", err
		}
		data, err := os.ReadFile(absPath)
		if err != nil {
			return "", mapError(err, "read", args.Path)
		}
		return string(data), nil

	case "writeFile":
		absPath, err := resolve(args.Path)
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			return "", mapError(err, "mkdir", args.Path)
		}
		if err := os.WriteFile(absPath, []byte(args.Data), 0o644); err != nil {
			return "", mapError(err, "write", args.Path)
		}
		return "undefined", nil

	case "appendFile":
		absPath, err := resolve(args.Path)
		if err != nil {
			return "", err
		}
		f, err := os.OpenFile(absPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return "", mapError(err, "open", args.Path)
		}
		defer f.Close()
		if _, err := f.WriteString(args.Data); err != nil {
			return "", mapError(err, "write", args.Path)
		}
		return "undefined", nil

	case "readdir":
		absPath, err := resolve(args.Path)
		if err != nil {
			return "", err
		}
		entries, err := os.ReadDir(absPath)
		if err != nil {
			return "", mapError(err, "scandir", args.Path)
		}
		if args.WithFileTypes {
			var dirents []string
			for _, e := range entries {
				dirents = append(dirents, buildDirentJSON(e, args.Path))
			}
			return "[" + strings.Join(dirents, ",") + "]", nil
		}
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		b, _ := json.Marshal(names)
		return string(b), nil

	case "stat":
		absPath, err := resolve(args.Path)
		if err != nil {
			return "", err
		}
		info, err := os.Stat(absPath)
		if err != nil {
			return "", mapError(err, "stat", args.Path)
		}
		return buildStatsJSON(info), nil

	case "lstat":
		absPath, err := resolve(args.Path)
		if err != nil {
			return "", err
		}
		info, err := os.Lstat(absPath)
		if err != nil {
			return "", mapError(err, "lstat", args.Path)
		}
		return buildStatsJSON(info), nil

	case "access":
		absPath, err := resolve(args.Path)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(absPath); err != nil {
			return "", mapError(err, "access", args.Path)
		}
		return "undefined", nil

	case "mkdir":
		absPath, err := resolve(args.Path)
		if err != nil {
			return "", err
		}
		if args.Recursive {
			err = os.MkdirAll(absPath, 0o755)
		} else {
			err = os.Mkdir(absPath, 0o755)
		}
		if err != nil {
			return "", mapError(err, "mkdir", args.Path)
		}
		if args.Recursive {
			return `"` + args.Path + `"`, nil
		}
		return "undefined", nil

	case "mkdtemp":
		absPrefix := args.Prefix
		if p.root != "" {
			absPrefix = filepath.Join(p.root, filepath.Clean("/"+args.Prefix))
		}
		dir, err := os.MkdirTemp(filepath.Dir(absPrefix), filepath.Base(absPrefix))
		if err != nil {
			return "", mapError(err, "mkdtemp", args.Prefix)
		}
		if p.root != "" {
			rel, _ := filepath.Rel(p.root, dir)
			return `"` + rel + `"`, nil
		}
		return `"` + dir + `"`, nil

	case "rmdir":
		absPath, err := resolve(args.Path)
		if err != nil {
			return "", err
		}
		if err := os.Remove(absPath); err != nil {
			return "", mapError(err, "rmdir", args.Path)
		}
		return "undefined", nil

	case "rm":
		absPath, err := resolve(args.Path)
		if err != nil {
			return "", err
		}
		if args.Recursive {
			err = os.RemoveAll(absPath)
		} else {
			err = os.Remove(absPath)
		}
		if err != nil && !args.Force {
			return "", mapError(err, "rm", args.Path)
		}
		return "undefined", nil

	case "unlink":
		absPath, err := resolve(args.Path)
		if err != nil {
			return "", err
		}
		if err := os.Remove(absPath); err != nil {
			return "", mapError(err, "unlink", args.Path)
		}
		return "undefined", nil

	case "rename":
		oldAbs, err := resolve(args.OldPath)
		if err != nil {
			return "", err
		}
		newAbs, err := resolve(args.NewPath)
		if err != nil {
			return "", err
		}
		if err := os.Rename(oldAbs, newAbs); err != nil {
			return "", mapError(err, "rename", args.OldPath)
		}
		return "undefined", nil

	case "copyFile":
		srcAbs, err := resolve(args.Path)
		if err != nil {
			return "", err
		}
		dstAbs, err := resolve(args.Dest)
		if err != nil {
			return "", err
		}
		in, err := os.Open(srcAbs)
		if err != nil {
			return "", mapError(err, "copyfile", args.Path)
		}
		defer in.Close()
		out, err := os.Create(dstAbs)
		if err != nil {
			return "", mapError(err, "copyfile", args.Dest)
		}
		defer out.Close()
		if _, err := io.Copy(out, in); err != nil {
			return "", mapError(err, "copyfile", args.Path)
		}
		return "undefined", nil

	case "cp":
		srcAbs, err := resolve(args.Path)
		if err != nil {
			return "", err
		}
		dstAbs, err := resolve(args.Dest)
		if err != nil {
			return "", err
		}
		if err := recursiveCopy(srcAbs, dstAbs); err != nil {
			return "", mapError(err, "cp", args.Path)
		}
		return "undefined", nil

	case "link":
		existing, err := resolve(args.Path)
		if err != nil {
			return "", err
		}
		newPath, err := resolve(args.NewPath)
		if err != nil {
			return "", err
		}
		if err := os.Link(existing, newPath); err != nil {
			return "", mapError(err, "link", args.Path)
		}
		return "undefined", nil

	case "symlink":
		linkPath, err := resolve(args.Path)
		if err != nil {
			return "", err
		}
		if err := os.Symlink(args.Target, linkPath); err != nil {
			return "", mapError(err, "symlink", args.Path)
		}
		return "undefined", nil

	case "readlink":
		absPath, err := resolve(args.Path)
		if err != nil {
			return "", err
		}
		target, err := os.Readlink(absPath)
		if err != nil {
			return "", mapError(err, "readlink", args.Path)
		}
		return `"` + target + `"`, nil

	case "realpath":
		absPath, err := resolve(args.Path)
		if err != nil {
			return "", err
		}
		resolved, err := filepath.EvalSymlinks(absPath)
		if err != nil {
			return "", mapError(err, "realpath", args.Path)
		}
		if p.root != "" {
			rel, _ := filepath.Rel(p.root, resolved)
			return `"` + rel + `"`, nil
		}
		return `"` + resolved + `"`, nil

	case "chmod":
		absPath, err := resolve(args.Path)
		if err != nil {
			return "", err
		}
		if err := os.Chmod(absPath, os.FileMode(args.Mode)); err != nil {
			return "", mapError(err, "chmod", args.Path)
		}
		return "undefined", nil

	case "chown":
		absPath, err := resolve(args.Path)
		if err != nil {
			return "", err
		}
		if err := os.Chown(absPath, args.UID, args.GID); err != nil {
			return "", mapError(err, "chown", args.Path)
		}
		return "undefined", nil

	case "lchown":
		absPath, err := resolve(args.Path)
		if err != nil {
			return "", err
		}
		if err := os.Lchown(absPath, args.UID, args.GID); err != nil {
			return "", mapError(err, "lchown", args.Path)
		}
		return "undefined", nil

	case "truncate":
		absPath, err := resolve(args.Path)
		if err != nil {
			return "", err
		}
		if err := os.Truncate(absPath, args.Len); err != nil {
			return "", mapError(err, "truncate", args.Path)
		}
		return "undefined", nil

	case "utimes":
		absPath, err := resolve(args.Path)
		if err != nil {
			return "", err
		}
		atime := time.Unix(int64(args.Atime), 0)
		mtime := time.Unix(int64(args.Mtime), 0)
		if err := os.Chtimes(absPath, atime, mtime); err != nil {
			return "", mapError(err, "utimes", args.Path)
		}
		return "undefined", nil

	case "open":
		absPath, err := resolve(args.Path)
		if err != nil {
			return "", err
		}
		flags := parseFlags(args.Flags)
		mode := os.FileMode(0o666)
		if args.FileMode > 0 {
			mode = os.FileMode(args.FileMode)
		}
		f, err := os.OpenFile(absPath, flags, mode)
		if err != nil {
			return "", mapError(err, "open", args.Path)
		}
		fd := p.trackHandle(f)
		return fmt.Sprintf(`{"fd":%d}`, fd), nil

	default:
		return "", fmt.Errorf("fs: unknown async op %q", op)
	}
}

// ---------------------------------------------------------------------------
// FileHandle bridges
// ---------------------------------------------------------------------------

func (p *FSPolyfill) registerFileHandleBridges(ctx *quickjs.Context, b *Bridge, root string) {
	// __go_fs_fh_read(fd, length, position) → Promise<{data, bytesRead}>
	ctx.Globals().Set("__go_fs_fh_read", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 3 {
			return qctx.ThrowError(fmt.Errorf("fh.read: fd, length, position required"))
		}
		fd := int(args[0].ToInt32())
		length := int(args[1].ToInt32())
		position := args[2].ToInt64()

		return qctx.NewPromise(func(resolveP, reject func(*quickjs.Value)) {
			b.Go(func(goCtx context.Context) {
				f := p.getHandle(fd)
				if f == nil {
					scheduleReject(qctx, reject, fmt.Errorf("bad file descriptor"), "read", "")
					return
				}
				buf := make([]byte, length)
				var n int
				var err error
				if position >= 0 {
					n, err = f.ReadAt(buf, position)
				} else {
					n, err = f.Read(buf)
				}
				if err != nil && err != io.EOF {
					scheduleReject(qctx, reject, err, "read", "")
					return
				}
				data := string(buf[:n])
				qctx.Schedule(func(qctx *quickjs.Context) {
					result := fmt.Sprintf(`{"bytesRead":%d,"data":%q}`, n, data)
					resolveP(qctx.NewString(result))
				})
			})
		})
	}))

	// __go_fs_fh_write(fd, data, position) → Promise<{bytesWritten}>
	ctx.Globals().Set("__go_fs_fh_write", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 3 {
			return qctx.ThrowError(fmt.Errorf("fh.write: fd, data, position required"))
		}
		fd := int(args[0].ToInt32())
		data := args[1].String()
		position := args[2].ToInt64()

		return qctx.NewPromise(func(resolveP, reject func(*quickjs.Value)) {
			b.Go(func(goCtx context.Context) {
				f := p.getHandle(fd)
				if f == nil {
					scheduleReject(qctx, reject, fmt.Errorf("bad file descriptor"), "write", "")
					return
				}
				var n int
				var err error
				if position >= 0 {
					n, err = f.WriteAt([]byte(data), position)
				} else {
					n, err = f.Write([]byte(data))
				}
				if err != nil {
					scheduleReject(qctx, reject, err, "write", "")
					return
				}
				qctx.Schedule(func(qctx *quickjs.Context) {
					resolveP(qctx.NewString(fmt.Sprintf(`{"bytesWritten":%d}`, n)))
				})
			})
		})
	}))

	// __go_fs_fh_close(fd) → Promise<void>
	ctx.Globals().Set("__go_fs_fh_close", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("fh.close: fd required"))
		}
		fd := int(args[0].ToInt32())
		return qctx.NewPromise(func(resolveP, reject func(*quickjs.Value)) {
			b.Go(func(goCtx context.Context) {
				if err := p.closeHandle(fd); err != nil {
					scheduleReject(qctx, reject, err, "close", "")
					return
				}
				qctx.Schedule(func(qctx *quickjs.Context) {
					resolveP(qctx.NewUndefined())
				})
			})
		})
	}))

	// __go_fs_fh_stat(fd) → Promise<StatsJSON>
	ctx.Globals().Set("__go_fs_fh_stat", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("fh.stat: fd required"))
		}
		fd := int(args[0].ToInt32())
		return qctx.NewPromise(func(resolveP, reject func(*quickjs.Value)) {
			b.Go(func(goCtx context.Context) {
				f := p.getHandle(fd)
				if f == nil {
					scheduleReject(qctx, reject, fmt.Errorf("bad file descriptor"), "fstat", "")
					return
				}
				info, err := f.Stat()
				if err != nil {
					scheduleReject(qctx, reject, err, "fstat", "")
					return
				}
				qctx.Schedule(func(qctx *quickjs.Context) {
					resolveP(qctx.NewString(buildStatsJSON(info)))
				})
			})
		})
	}))

	// __go_fs_fh_truncate(fd, len) → Promise<void>
	ctx.Globals().Set("__go_fs_fh_truncate", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return qctx.ThrowError(fmt.Errorf("fh.truncate: fd, len required"))
		}
		fd := int(args[0].ToInt32())
		size := args[1].ToInt64()
		return qctx.NewPromise(func(resolveP, reject func(*quickjs.Value)) {
			b.Go(func(goCtx context.Context) {
				f := p.getHandle(fd)
				if f == nil {
					scheduleReject(qctx, reject, fmt.Errorf("bad file descriptor"), "ftruncate", "")
					return
				}
				if err := f.Truncate(size); err != nil {
					scheduleReject(qctx, reject, err, "ftruncate", "")
					return
				}
				qctx.Schedule(func(qctx *quickjs.Context) {
					resolveP(qctx.NewUndefined())
				})
			})
		})
	}))

	// __go_fs_fh_readFile(fd, encoding) → Promise<string>
	ctx.Globals().Set("__go_fs_fh_readFile", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("fh.readFile: fd required"))
		}
		fd := int(args[0].ToInt32())
		return qctx.NewPromise(func(resolveP, reject func(*quickjs.Value)) {
			b.Go(func(goCtx context.Context) {
				f := p.getHandle(fd)
				if f == nil {
					scheduleReject(qctx, reject, fmt.Errorf("bad file descriptor"), "read", "")
					return
				}
				f.Seek(0, 0)
				data, err := io.ReadAll(f)
				if err != nil {
					scheduleReject(qctx, reject, err, "read", "")
					return
				}
				qctx.Schedule(func(qctx *quickjs.Context) {
					resolveP(qctx.NewString(string(data)))
				})
			})
		})
	}))

	// __go_fs_fh_writeFile(fd, data) → Promise<void>
	ctx.Globals().Set("__go_fs_fh_writeFile", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return qctx.ThrowError(fmt.Errorf("fh.writeFile: fd, data required"))
		}
		fd := int(args[0].ToInt32())
		data := args[1].String()
		return qctx.NewPromise(func(resolveP, reject func(*quickjs.Value)) {
			b.Go(func(goCtx context.Context) {
				f := p.getHandle(fd)
				if f == nil {
					scheduleReject(qctx, reject, fmt.Errorf("bad file descriptor"), "write", "")
					return
				}
				f.Truncate(0)
				f.Seek(0, 0)
				if _, err := f.WriteString(data); err != nil {
					scheduleReject(qctx, reject, err, "write", "")
					return
				}
				qctx.Schedule(func(qctx *quickjs.Context) {
					resolveP(qctx.NewUndefined())
				})
			})
		})
	}))
}

// ---------------------------------------------------------------------------
// Stream bridges — createReadStream / createWriteStream
// ---------------------------------------------------------------------------

func (p *FSPolyfill) registerStreamBridges(ctx *quickjs.Context, b *Bridge, root string) {
	// __go_fs_createReadStream(path, optionsJSON) → streamID
	// The Go side opens the file and pushes chunks to the JS Readable via scheduled callbacks.
	ctx.Globals().Set("__go_fs_createReadStream", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("createReadStream: path required"))
		}
		userPath := args[0].String()
		absPath, err := p.resolve(userPath)
		if err != nil {
			return throwFSError(qctx, err, "open", userPath)
		}
		streamID := fmt.Sprintf("rs_%d", time.Now().UnixNano())

		b.Go(func(goCtx context.Context) {
			f, err := os.Open(absPath)
			if err != nil {
				qctx.Schedule(func(qctx *quickjs.Context) {
					qctx.Eval(fmt.Sprintf(`globalThis.__fs_streams[%q]&&globalThis.__fs_streams[%q]._onError(%q)`, streamID, streamID, err.Error()))
				})
				return
			}
			defer f.Close()
			buf := make([]byte, 16384)
			for {
				if goCtx.Err() != nil {
					return
				}
				n, readErr := f.Read(buf)
				if n > 0 {
					chunk := string(buf[:n])
					escaped, _ := json.Marshal(chunk)
					qctx.Schedule(func(qctx *quickjs.Context) {
						qctx.Eval(fmt.Sprintf(`globalThis.__fs_streams[%q]&&globalThis.__fs_streams[%q]._onData(%s)`, streamID, streamID, string(escaped)))
					})
				}
				if readErr == io.EOF {
					qctx.Schedule(func(qctx *quickjs.Context) {
						qctx.Eval(fmt.Sprintf(`globalThis.__fs_streams[%q]&&globalThis.__fs_streams[%q]._onEnd()`, streamID, streamID))
					})
					return
				}
				if readErr != nil {
					qctx.Schedule(func(qctx *quickjs.Context) {
						qctx.Eval(fmt.Sprintf(`globalThis.__fs_streams[%q]&&globalThis.__fs_streams[%q]._onError(%q)`, streamID, streamID, readErr.Error()))
					})
					return
				}
			}
		})

		return qctx.NewString(streamID)
	}))

	// __go_fs_createWriteStream(path, optionsJSON) → fd
	ctx.Globals().Set("__go_fs_createWriteStream", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("createWriteStream: path required"))
		}
		userPath := args[0].String()
		absPath, err := p.resolve(userPath)
		if err != nil {
			return throwFSError(qctx, err, "open", userPath)
		}
		f, err := os.Create(absPath)
		if err != nil {
			return throwFSError(qctx, err, "open", userPath)
		}
		fd := p.trackHandle(f)
		return qctx.NewInt32(int32(fd))
	}))

	// __go_fs_ws_write(fd, chunk) → bool
	ctx.Globals().Set("__go_fs_ws_write", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return qctx.NewBool(false)
		}
		fd := int(args[0].ToInt32())
		chunk := args[1].String()
		f := p.getHandle(fd)
		if f == nil {
			return qctx.NewBool(false)
		}
		_, err := f.WriteString(chunk)
		return qctx.NewBool(err == nil)
	}))

	// __go_fs_ws_close(fd)
	ctx.Globals().Set("__go_fs_ws_close", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.NewUndefined()
		}
		fd := int(args[0].ToInt32())
		p.closeHandle(fd)
		return qctx.NewUndefined()
	}))
}

// ---------------------------------------------------------------------------
// Watch bridges — fsnotify
// ---------------------------------------------------------------------------

func (p *FSPolyfill) registerWatchBridges(ctx *quickjs.Context, b *Bridge, root string) {
	// __go_fs_watch(path, watchID) → void
	// Creates a fsnotify watcher and sends events to JS via __fs_watchers[watchID]._onEvent
	ctx.Globals().Set("__go_fs_watch", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return qctx.ThrowError(fmt.Errorf("watch: path and watchID required"))
		}
		userPath := args[0].String()
		watchID := args[1].String()

		absPath, err := p.resolve(userPath)
		if err != nil {
			return throwFSError(qctx, err, "watch", userPath)
		}

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return throwFSError(qctx, err, "watch", userPath)
		}
		if err := watcher.Add(absPath); err != nil {
			watcher.Close()
			return throwFSError(qctx, err, "watch", userPath)
		}

		p.watchersMu.Lock()
		p.watchers = append(p.watchers, watcher)
		p.watchersMu.Unlock()

		b.Go(func(goCtx context.Context) {
			defer watcher.Close()
			for {
				select {
				case <-goCtx.Done():
					return
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}
					eventType := "change"
					if event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
						eventType = "rename"
					}
					filename := filepath.Base(event.Name)
					escapedName, _ := json.Marshal(filename)
					qctx.Schedule(func(qctx *quickjs.Context) {
						qctx.Eval(fmt.Sprintf(`globalThis.__fs_watchers[%q]&&globalThis.__fs_watchers[%q]._onEvent(%q,%s)`,
							watchID, watchID, eventType, string(escapedName)))
					})
				case err, ok := <-watcher.Errors:
					if !ok {
						return
					}
					qctx.Schedule(func(qctx *quickjs.Context) {
						qctx.Eval(fmt.Sprintf(`globalThis.__fs_watchers[%q]&&globalThis.__fs_watchers[%q]._onError(%q)`,
							watchID, watchID, err.Error()))
					})
				}
			}
		})

		return qctx.NewUndefined()
	}))
}

// ---------------------------------------------------------------------------
// JS setup — builds globalThis.fs with all methods and proper Node.js shapes
// ---------------------------------------------------------------------------

const fsSetupJS = `
(function() {
  "use strict";

  // ─── Stats class ────────────────────────────────────────
  function Stats(raw) {
    var data = typeof raw === "string" ? JSON.parse(raw) : raw;
    this.dev = data.dev || 0;
    this.ino = data.ino || 0;
    this.mode = data.mode || 0;
    this.nlink = data.nlink || 1;
    this.uid = data.uid || 0;
    this.gid = data.gid || 0;
    this.rdev = data.rdev || 0;
    this.size = data.size || 0;
    this.blksize = data.blksize || 4096;
    this.blocks = data.blocks || 0;
    this.atimeMs = data.atimeMs || 0;
    this.mtimeMs = data.mtimeMs || 0;
    this.ctimeMs = data.ctimeMs || 0;
    this.birthtimeMs = data.birthtimeMs || 0;
    this.atime = new Date(this.atimeMs);
    this.mtime = new Date(this.mtimeMs);
    this.ctime = new Date(this.ctimeMs);
    this.birthtime = new Date(this.birthtimeMs);
    this._isFile = data._isFile || false;
    this._isDirectory = data._isDirectory || false;
    this._isSymbolicLink = data._isSymbolicLink || false;
    this._isBlockDevice = data._isBlockDevice || false;
    this._isCharDevice = data._isCharDevice || false;
    this._isFIFO = data._isFIFO || false;
    this._isSocket = data._isSocket || false;
  }
  Stats.prototype.isFile = function() { return this._isFile; };
  Stats.prototype.isDirectory = function() { return this._isDirectory; };
  Stats.prototype.isSymbolicLink = function() { return this._isSymbolicLink; };
  Stats.prototype.isBlockDevice = function() { return this._isBlockDevice; };
  Stats.prototype.isCharacterDevice = function() { return this._isCharDevice; };
  Stats.prototype.isFIFO = function() { return this._isFIFO; };
  Stats.prototype.isSocket = function() { return this._isSocket; };

  function parseStats(raw) { return new Stats(raw); }

  // ─── Dirent class ───────────────────────────────────────
  function Dirent(raw) {
    var data = typeof raw === "string" ? JSON.parse(raw) : raw;
    this.name = data.name;
    this.parentPath = data.parentPath || "";
    this._isFile = data._isFile || false;
    this._isDirectory = data._isDirectory || false;
    this._isSymbolicLink = data._isSymbolicLink || false;
    this._isBlockDevice = data._isBlockDevice || false;
    this._isCharacterDevice = data._isCharacterDevice || false;
    this._isFIFO = data._isFIFO || false;
    this._isSocket = data._isSocket || false;
  }
  Dirent.prototype.isFile = function() { return this._isFile; };
  Dirent.prototype.isDirectory = function() { return this._isDirectory; };
  Dirent.prototype.isSymbolicLink = function() { return this._isSymbolicLink; };
  Dirent.prototype.isBlockDevice = function() { return this._isBlockDevice; };
  Dirent.prototype.isCharacterDevice = function() { return this._isCharacterDevice; };
  Dirent.prototype.isFIFO = function() { return this._isFIFO; };
  Dirent.prototype.isSocket = function() { return this._isSocket; };

  function parseDirents(raw) {
    var arr = typeof raw === "string" ? JSON.parse(raw) : raw;
    return arr.map(function(d) { return new Dirent(d); });
  }

  // ─── FileHandle class ───────────────────────────────────
  function FileHandle(fd) {
    this.fd = fd;
  }
  FileHandle.prototype.read = function(opts) {
    var len = (opts && opts.length) || 16384;
    var pos = (opts && opts.position !== undefined) ? opts.position : -1;
    return __go_fs_fh_read(this.fd, len, pos).then(function(r) { return JSON.parse(r); });
  };
  FileHandle.prototype.readFile = function(opts) {
    return __go_fs_fh_readFile(this.fd);
  };
  FileHandle.prototype.write = function(data, offset, length, position) {
    var pos = (typeof position === "number") ? position : -1;
    return __go_fs_fh_write(this.fd, data, pos).then(function(r) { return JSON.parse(r); });
  };
  FileHandle.prototype.writeFile = function(data) {
    return __go_fs_fh_writeFile(this.fd, typeof data === "string" ? data : String(data));
  };
  FileHandle.prototype.close = function() {
    return __go_fs_fh_close(this.fd);
  };
  FileHandle.prototype.stat = function() {
    return __go_fs_fh_stat(this.fd).then(parseStats);
  };
  FileHandle.prototype.truncate = function(len) {
    return __go_fs_fh_truncate(this.fd, len || 0);
  };

  // ─── Async helper ───────────────────────────────────────
  function fsAsync(op, args) {
    return __go_fs_async(op, JSON.stringify(args));
  }

  // ─── fs.promises ────────────────────────────────────────
  var promises = {
    readFile: function(path, opts) {
      return fsAsync("readFile", { path: path, encoding: (opts && opts.encoding) || (typeof opts === "string" ? opts : "") });
    },
    writeFile: function(path, data, opts) {
      return fsAsync("writeFile", { path: path, data: typeof data === "string" ? data : String(data) }).then(function() {});
    },
    appendFile: function(path, data, opts) {
      return fsAsync("appendFile", { path: path, data: typeof data === "string" ? data : String(data) }).then(function() {});
    },
    readdir: function(path, opts) {
      var wft = opts && opts.withFileTypes;
      return fsAsync("readdir", { path: path, withFileTypes: !!wft }).then(function(r) {
        if (wft) return parseDirents(r);
        return JSON.parse(r);
      });
    },
    stat: function(path, opts) {
      return fsAsync("stat", { path: path }).then(parseStats);
    },
    lstat: function(path, opts) {
      return fsAsync("lstat", { path: path }).then(parseStats);
    },
    access: function(path, mode) {
      return fsAsync("access", { path: path }).then(function() {});
    },
    mkdir: function(path, opts) {
      var recursive = opts && opts.recursive;
      return fsAsync("mkdir", { path: path, recursive: !!recursive }).then(function(r) {
        if (recursive && r !== "undefined") return JSON.parse(r);
      });
    },
    mkdtemp: function(prefix, opts) {
      return fsAsync("mkdtemp", { prefix: prefix }).then(function(r) { return JSON.parse(r); });
    },
    rmdir: function(path, opts) {
      return fsAsync("rmdir", { path: path }).then(function() {});
    },
    rm: function(path, opts) {
      return fsAsync("rm", { path: path, recursive: !!(opts && opts.recursive), force: !!(opts && opts.force) }).then(function() {});
    },
    unlink: function(path) {
      return fsAsync("unlink", { path: path }).then(function() {});
    },
    rename: function(oldPath, newPath) {
      return fsAsync("rename", { oldPath: oldPath, newPath: newPath }).then(function() {});
    },
    copyFile: function(src, dest, mode) {
      return fsAsync("copyFile", { path: src, dest: dest }).then(function() {});
    },
    cp: function(src, dest, opts) {
      return fsAsync("cp", { path: src, dest: dest }).then(function() {});
    },
    link: function(existingPath, newPath) {
      return fsAsync("link", { path: existingPath, newPath: newPath }).then(function() {});
    },
    symlink: function(target, path, type) {
      return fsAsync("symlink", { target: target, path: path }).then(function() {});
    },
    readlink: function(path, opts) {
      return fsAsync("readlink", { path: path }).then(function(r) { return JSON.parse(r); });
    },
    realpath: function(path, opts) {
      return fsAsync("realpath", { path: path }).then(function(r) { return JSON.parse(r); });
    },
    chmod: function(path, mode) {
      return fsAsync("chmod", { path: path, mode: mode }).then(function() {});
    },
    chown: function(path, uid, gid) {
      return fsAsync("chown", { path: path, uid: uid, gid: gid }).then(function() {});
    },
    lchown: function(path, uid, gid) {
      return fsAsync("lchown", { path: path, uid: uid, gid: gid }).then(function() {});
    },
    truncate: function(path, len) {
      return fsAsync("truncate", { path: path, len: len || 0 }).then(function() {});
    },
    utimes: function(path, atime, mtime) {
      return fsAsync("utimes", { path: path, atime: atime, mtime: mtime }).then(function() {});
    },
    open: function(path, flags, mode) {
      return fsAsync("open", { path: path, flags: flags || "r", fileMode: mode || 0 }).then(function(r) {
        var parsed = JSON.parse(r);
        return new FileHandle(parsed.fd);
      });
    },
    watch: function(path, opts) {
      // Returns an AsyncIterable
      var watchID = "w_" + Date.now() + "_" + Math.random().toString(36).slice(2);
      var queue = [];
      var waiters = [];
      var closed = false;

      if (!globalThis.__fs_watchers) globalThis.__fs_watchers = {};
      globalThis.__fs_watchers[watchID] = {
        _onEvent: function(eventType, filename) {
          var evt = { eventType: eventType, filename: filename };
          if (waiters.length > 0) {
            var w = waiters.shift();
            w({ done: false, value: evt });
          } else {
            queue.push(evt);
          }
        },
        _onError: function(msg) {
          closed = true;
          while (waiters.length) waiters.shift()({ done: true, value: undefined });
        },
      };

      __go_fs_watch(path, watchID);

      return {
        [Symbol.asyncIterator]: function() {
          return {
            next: function() {
              if (queue.length > 0) return Promise.resolve({ done: false, value: queue.shift() });
              if (closed) return Promise.resolve({ done: true, value: undefined });
              return new Promise(function(resolve) { waiters.push(resolve); });
            },
            return: function() {
              closed = true;
              delete globalThis.__fs_watchers[watchID];
              while (waiters.length) waiters.shift()({ done: true, value: undefined });
              return Promise.resolve({ done: true, value: undefined });
            },
          };
        },
        close: function() {
          closed = true;
          delete globalThis.__fs_watchers[watchID];
          while (waiters.length) waiters.shift()({ done: true, value: undefined });
        },
      };
    },
  };

  // ─── Sync wrappers ──────────────────────────────────────
  function _parseReaddirSync(raw, opts) {
    if (opts && opts.withFileTypes) return parseDirents(raw);
    return JSON.parse(raw);
  }

  // ─── Stream helpers ─────────────────────────────────────
  if (!globalThis.__fs_streams) globalThis.__fs_streams = {};

  // ─── Build globalThis.fs ────────────────────────────────
  globalThis.fs = {
    promises: promises,

    // Sync methods
    readFileSync: function(path, opts) { return __go_fs_readFileSync(path, typeof opts === "string" ? opts : JSON.stringify(opts || {})); },
    writeFileSync: function(path, data, opts) { return __go_fs_writeFileSync(path, typeof data === "string" ? data : String(data), JSON.stringify(opts || {})); },
    appendFileSync: function(path, data, opts) { return __go_fs_appendFileSync(path, typeof data === "string" ? data : String(data), JSON.stringify(opts || {})); },
    readdirSync: function(path, opts) { return _parseReaddirSync(__go_fs_readdirSync(path, JSON.stringify(opts || {})), opts); },
    statSync: function(path, opts) { return parseStats(__go_fs_statSync(path, JSON.stringify(opts || {}))); },
    lstatSync: function(path, opts) { return parseStats(__go_fs_lstatSync(path, JSON.stringify(opts || {}))); },
    accessSync: function(path, mode) { return __go_fs_accessSync(path, mode || 0); },
    mkdirSync: function(path, opts) { return __go_fs_mkdirSync(path, JSON.stringify(opts || {})); },
    mkdtempSync: function(prefix, opts) { return __go_fs_mkdtempSync(prefix || ""); },
    rmdirSync: function(path, opts) { return __go_fs_rmdirSync(path, JSON.stringify(opts || {})); },
    rmSync: function(path, opts) { return __go_fs_rmSync(path, JSON.stringify(opts || {})); },
    unlinkSync: function(path) { return __go_fs_unlinkSync(path); },
    renameSync: function(oldPath, newPath) { return __go_fs_renameSync(oldPath, newPath); },
    copyFileSync: function(src, dest, mode) { return __go_fs_copyFileSync(src, dest); },
    cpSync: function(src, dest, opts) { return __go_fs_cpSync(src, dest); },
    linkSync: function(existingPath, newPath) { return __go_fs_linkSync(existingPath, newPath); },
    symlinkSync: function(target, path, type) { return __go_fs_symlinkSync(target, path); },
    readlinkSync: function(path, opts) { return __go_fs_readlinkSync(path); },
    realpathSync: function(path, opts) { return __go_fs_realpathSync(path); },
    chmodSync: function(path, mode) { return __go_fs_chmodSync(path, mode); },
    chownSync: function(path, uid, gid) { return __go_fs_chownSync(path, uid, gid); },
    truncateSync: function(path, len) { return __go_fs_truncateSync(path, len || 0); },
    utimesSync: function(path, atime, mtime) { return __go_fs_utimesSync(path, atime, mtime); },
    existsSync: function(path) { return __go_fs_existsSync(path); },

    // Streams
    createReadStream: function(path, opts) {
      var streamID = __go_fs_createReadStream(path, JSON.stringify(opts || {}));
      var ee = new EventEmitter();
      globalThis.__fs_streams[streamID] = {
        _onData: function(chunk) { ee.emit("data", chunk); },
        _onEnd: function() { ee.emit("end"); ee.emit("close"); delete globalThis.__fs_streams[streamID]; },
        _onError: function(msg) { ee.emit("error", new Error(msg)); ee.emit("close"); delete globalThis.__fs_streams[streamID]; },
      };
      ee.path = path;
      return ee;
    },
    createWriteStream: function(path, opts) {
      var fd = __go_fs_createWriteStream(path, JSON.stringify(opts || {}));
      var ee = new EventEmitter();
      ee.fd = fd;
      ee.path = path;
      ee.write = function(chunk, encoding, callback) {
        var ok = __go_fs_ws_write(fd, typeof chunk === "string" ? chunk : String(chunk));
        if (callback) callback(ok ? null : new Error("write failed"));
        return ok;
      };
      ee.end = function(chunk, encoding, callback) {
        if (chunk) ee.write(chunk);
        __go_fs_ws_close(fd);
        ee.emit("finish");
        ee.emit("close");
        if (callback) callback();
      };
      ee.close = function(callback) { ee.end(null, null, callback); };
      return ee;
    },

    // Watch
    watch: function(path, opts, listener) {
      var watcher = promises.watch(path, opts);
      if (typeof listener === "function") {
        (async function() {
          for await (var evt of watcher) {
            listener(evt.eventType, evt.filename);
          }
        })();
      }
      return watcher;
    },

    // Constants — injected from Go for platform correctness
    constants: (function() { try { return JSON.parse(__go_fs_constants_json); } catch(e) { return {
      F_OK: 0, R_OK: 4, W_OK: 2, X_OK: 1,
      COPYFILE_EXCL: 1, COPYFILE_FICLONE: 2, COPYFILE_FICLONE_FORCE: 4,
      O_RDONLY: 0, O_WRONLY: 1, O_RDWR: 2,
      O_CREAT: 64, O_EXCL: 128, O_TRUNC: 512, O_APPEND: 1024}; } })(),

    // Classes (for instanceof checks)
    Stats: Stats,
    Dirent: Dirent,
  };
})();
`
