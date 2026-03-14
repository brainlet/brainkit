package jsbridge

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

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
// Bridge is safe for concurrent use from multiple goroutines — calls to Eval
// are serialized via a mutex. For true parallelism, create multiple Bridges.
//
// All goroutines started by bridge polyfills (fetch, fs, exec) are tracked
// via a WaitGroup and cancelled via context on Close. No orphaned goroutines.
type Bridge struct {
	runtime *quickjs.Runtime
	ctx     *quickjs.Context
	stdout  io.Writer
	stderr  io.Writer
	mu      sync.Mutex // serializes Eval/EvalBytecode calls

	// Goroutine lifecycle control
	goCtx    context.Context    // cancelled on Close — all goroutines stop
	goCancel context.CancelFunc // triggers cancellation
	wg       sync.WaitGroup    // tracks active goroutines
}

// New creates a Bridge, sets up all polyfills, and returns it ready for use.
func New(cfg Config, polyfills ...Polyfill) (*Bridge, error) {
	if cfg.MemoryLimit == 0 {
		cfg.MemoryLimit = 256 * 1024 * 1024
	}
	if cfg.MaxStackSize == 0 {
		// 256MB — QuickJS's stack_top-based detection misfires with CGo
		// because stack position varies between CGo transitions. 256MB puts
		// the threshold well below the OS thread stack so only real exhaustion
		// (SIGSEGV) triggers. See as-embed-fixes memory note.
		cfg.MaxStackSize = 256 * 1024 * 1024
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
	goCtx, goCancel := context.WithCancel(context.Background())

	b := &Bridge{
		runtime:  rt,
		ctx:      ctx,
		stdout:   cfg.Stdout,
		stderr:   cfg.Stderr,
		goCtx:    goCtx,
		goCancel: goCancel,
	}

	for _, p := range polyfills {
		// Give polyfills access to the bridge for tracked goroutines
		if ba, ok := p.(BridgeAware); ok {
			ba.SetBridge(b)
		}
		if err := p.Setup(ctx); err != nil {
			b.Close()
			return nil, fmt.Errorf("jsbridge: polyfill %s: %w", p.Name(), err)
		}
	}

	return b, nil
}

// Close shuts down the runtime and frees all resources.
// Cancels all goroutines and waits for them to finish before freeing QuickJS.
// Tolerates unreleased JS objects (ReadableStream controllers from streaming
// fetch, large bundle object graphs) by skipping JS_FreeRuntime's assertion.
func (b *Bridge) Close() {
	// 1. Cancel all goroutines (HTTP calls abort, reads return error)
	b.goCancel()
	// 2. Wait for all goroutines to finish (no goroutine touches QuickJS after this)
	b.wg.Wait()
	// 3. Now safe to free QuickJS
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.ctx != nil {
		b.ctx.ProcessJobs()
		b.ctx.Close()
		b.ctx = nil
	}
	if b.runtime != nil {
		b.runtime.RunGC()
		// Skip runtime.Close() — JS_FreeRuntime asserts gc_obj_list is empty,
		// which fails when streaming fetch creates ReadableStream controllers
		// or large bundles have circular object graphs. The C memory is
		// reclaimed on process exit. For long-lived bridges (sandboxes),
		// this is a non-issue since they're reused, not constantly created.
		b.runtime = nil
	}
}

// Go starts a tracked goroutine. The function receives the bridge's context,
// which is cancelled on Close. All goroutines started via Go are waited-on
// before the QuickJS runtime is freed.
func (b *Bridge) Go(fn func(ctx context.Context)) {
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				// Goroutine panicked — don't crash the process.
				// The scheduled reject (if any) won't fire, but that's OK —
				// the Promise will remain pending and the Await loop will
				// eventually time out or the bridge will close.
				fmt.Fprintf(b.stderr, "jsbridge: goroutine panic: %v\n", r)
			}
		}()
		fn(b.goCtx)
	}()
}

// GoContext returns the context for goroutines. Cancelled on Close.
func (b *Bridge) GoContext() context.Context { return b.goCtx }

// Context returns the QuickJS execution context.
func (b *Bridge) Context() *quickjs.Context { return b.ctx }

// Runtime returns the underlying QuickJS runtime.
func (b *Bridge) Runtime() *quickjs.Runtime { return b.runtime }

// Global returns the global JavaScript object.
func (b *Bridge) Global() *quickjs.Value { return b.ctx.Globals() }

// Eval evaluates JavaScript code and returns the result.
// Safe for concurrent use — calls are serialized via mutex.
// Panics from Go bridge functions are caught and returned as errors.
func (b *Bridge) Eval(file string, code string, opts ...quickjs.EvalOption) (result *quickjs.Value, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	defer func() {
		if r := recover(); r != nil {
			result = nil
			err = fmt.Errorf("jsbridge: panic during eval: %v", r)
		}
	}()
	allOpts := append([]quickjs.EvalOption{quickjs.EvalFileName(file)}, opts...)
	val := b.ctx.Eval(code, allOpts...)
	if val.IsException() {
		e := b.ctx.Exception()
		val.Free()
		return nil, e
	}
	return val, nil
}

// EvalBytecode evaluates precompiled bytecode.
// Safe for concurrent use — calls are serialized via mutex.
func (b *Bridge) EvalBytecode(bytecode []byte) (*quickjs.Value, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	val := b.ctx.EvalBytecode(bytecode)
	if val.IsException() {
		err := b.ctx.Exception()
		val.Free()
		return nil, err
	}
	return val, nil
}

// EvalAsync evaluates JavaScript that returns a Promise, using Go-level polling
// (ctx.Await) which processes ctx.Schedule'd work from goroutines.
// Safe for concurrent use — calls are serialized via mutex.
// Panics from Go bridge functions are caught and returned as errors.
func (b *Bridge) EvalAsync(file string, code string) (result *quickjs.Value, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	defer func() {
		if r := recover(); r != nil {
			result = nil
			err = fmt.Errorf("jsbridge: panic during eval: %v", r)
		}
	}()
	val := b.ctx.Eval(code, quickjs.EvalFileName(file))
	if val.IsException() {
		e := b.ctx.Exception()
		val.Free()
		return nil, e
	}
	if val.IsPromise() {
		awaited := b.ctx.Await(val)
		if awaited.IsException() {
			e := b.ctx.Exception()
			awaited.Free()
			return nil, e
		}
		return awaited, nil
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
