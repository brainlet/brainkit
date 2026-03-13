package jsbridge

import (
	"fmt"
	"io"
	"os"

	quickjs "github.com/buke/quickjs-go"
)

// Config configures a Bridge.
type Config struct {
	MemoryLimit  int       // bytes; default 256MB
	MaxStackSize int       // bytes; default 4MB
	GCThreshold  int64     // bytes; auto-GC trigger threshold; -1 disables (default: -1)
	Stdout       io.Writer // default os.Stdout
	Stderr       io.Writer // default os.Stderr
	CWD          string    // working directory
}

// Bridge wraps a native QuickJS runtime with polyfills and bridge functions.
// NOTE: Bridge does NOT call runtime.LockOSThread/UnlockOSThread.
// buke/quickjs-go's NewRuntime() handles thread locking internally.
type Bridge struct {
	runtime *quickjs.Runtime
	ctx     *quickjs.Context
	stdout  io.Writer
	stderr  io.Writer
}

// New creates a Bridge, sets up all polyfills, and returns it ready for use.
func New(cfg Config, polyfills ...Polyfill) (*Bridge, error) {
	if cfg.MemoryLimit == 0 {
		cfg.MemoryLimit = 256 * 1024 * 1024
	}
	if cfg.MaxStackSize == 0 {
		cfg.MaxStackSize = 4 * 1024 * 1024
	}
	if cfg.Stdout == nil {
		cfg.Stdout = os.Stdout
	}
	if cfg.Stderr == nil {
		cfg.Stderr = os.Stderr
	}

	rtOpts := []quickjs.Option{
		quickjs.WithMemoryLimit(uint64(cfg.MemoryLimit)),
		quickjs.WithMaxStackSize(uint64(cfg.MaxStackSize)),
	}
	if cfg.GCThreshold != 0 {
		rtOpts = append(rtOpts, quickjs.WithGCThreshold(cfg.GCThreshold))
	}

	rt := quickjs.NewRuntime(rtOpts...)

	ctx := rt.NewContext()

	b := &Bridge{
		runtime: rt,
		ctx:     ctx,
		stdout:  cfg.Stdout,
		stderr:  cfg.Stderr,
	}

	for _, p := range polyfills {
		if err := p.Setup(ctx); err != nil {
			b.Close()
			return nil, fmt.Errorf("jsbridge: polyfill %s: %w", p.Name(), err)
		}
	}

	return b, nil
}

// Close shuts down the runtime and frees all resources.
func (b *Bridge) Close() {
	if b.ctx != nil {
		b.ctx.Close()
		b.ctx = nil
	}
	if b.runtime != nil {
		b.runtime.Close()
		b.runtime = nil
	}
}

// Context returns the QuickJS execution context.
func (b *Bridge) Context() *quickjs.Context { return b.ctx }

// Runtime returns the underlying QuickJS runtime.
func (b *Bridge) Runtime() *quickjs.Runtime { return b.runtime }

// Global returns the global JavaScript object.
func (b *Bridge) Global() *quickjs.Value { return b.ctx.Globals() }

// Eval evaluates JavaScript code and returns the result.
// The file parameter is used for error reporting only.
// Extra opts (e.g. quickjs.EvalAwait(true)) are forwarded to the engine.
// Returns (*Value, error) — if the JS throws, error is set and *Value is nil.
func (b *Bridge) Eval(file string, code string, opts ...quickjs.EvalOption) (*quickjs.Value, error) {
	allOpts := append([]quickjs.EvalOption{quickjs.EvalFileName(file)}, opts...)
	val := b.ctx.Eval(code, allOpts...)
	if val.IsException() {
		err := b.ctx.Exception()
		val.Free()
		return nil, err
	}
	return val, nil
}

// EvalBytecode evaluates precompiled bytecode.
func (b *Bridge) EvalBytecode(bytecode []byte) (*quickjs.Value, error) {
	val := b.ctx.EvalBytecode(bytecode)
	if val.IsException() {
		err := b.ctx.Exception()
		val.Free()
		return nil, err
	}
	return val, nil
}

// Compile compiles JavaScript to bytecode without executing.
func (b *Bridge) Compile(file string, code string) ([]byte, error) {
	return b.ctx.Compile(code, quickjs.EvalFileName(file))
}

// Stdout returns the configured stdout writer.
func (b *Bridge) Stdout() io.Writer { return b.stdout }

// Stderr returns the configured stderr writer.
func (b *Bridge) Stderr() io.Writer { return b.stderr }
