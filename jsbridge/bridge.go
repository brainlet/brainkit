package jsbridge

import (
	"fmt"
	"io"
	"os"

	"github.com/fastschema/qjs"
)

// Config configures a Bridge.
type Config struct {
	MemoryLimit  int       // bytes; default 256MB
	MaxStackSize int       // bytes; default 4MB
	GCThreshold  int       // bytes; default 256KB
	Stdout       io.Writer // default os.Stdout
	Stderr       io.Writer // default os.Stderr
	CWD          string    // working directory
}

// Bridge wraps a QuickJS runtime with polyfills and bridge functions.
type Bridge struct {
	runtime *qjs.Runtime
}

// New creates a Bridge, sets up all polyfills, and returns it ready for use.
func New(cfg Config, polyfills ...Polyfill) (*Bridge, error) {
	if cfg.MemoryLimit == 0 {
		cfg.MemoryLimit = 256 * 1024 * 1024
	}
	if cfg.MaxStackSize == 0 {
		cfg.MaxStackSize = 4 * 1024 * 1024
	}
	if cfg.GCThreshold == 0 {
		cfg.GCThreshold = 256 * 1024
	}
	if cfg.Stdout == nil {
		cfg.Stdout = os.Stdout
	}
	if cfg.Stderr == nil {
		cfg.Stderr = os.Stderr
	}

	rt, err := qjs.New(qjs.Option{
		MemoryLimit:  cfg.MemoryLimit,
		MaxStackSize: cfg.MaxStackSize,
		GCThreshold:  cfg.GCThreshold,
		Stdout:       cfg.Stdout,
		Stderr:       cfg.Stderr,
		CWD:          cfg.CWD,
	})
	if err != nil {
		return nil, fmt.Errorf("jsbridge: create runtime: %w", err)
	}

	b := &Bridge{runtime: rt}

	ctx := rt.Context()
	for _, p := range polyfills {
		if err := p.Setup(ctx); err != nil {
			rt.Close()
			return nil, fmt.Errorf("jsbridge: polyfill %s: %w", p.Name(), err)
		}
	}

	return b, nil
}

// Close shuts down the runtime and frees all resources.
func (b *Bridge) Close() {
	if b.runtime != nil {
		b.runtime.Close()
	}
}

// Runtime returns the underlying QuickJS runtime.
func (b *Bridge) Runtime() *qjs.Runtime { return b.runtime }

// Context returns the QuickJS execution context.
func (b *Bridge) Context() *qjs.Context { return b.runtime.Context() }

// Global returns the global JavaScript object.
func (b *Bridge) Global() *qjs.Value { return b.runtime.Context().Global() }

// Eval evaluates JavaScript code.
func (b *Bridge) Eval(file string, flags ...qjs.EvalOptionFunc) (*qjs.Value, error) {
	return b.runtime.Eval(file, flags...)
}

// Compile compiles JavaScript to bytecode without executing.
func (b *Bridge) Compile(file string, flags ...qjs.EvalOptionFunc) ([]byte, error) {
	return b.runtime.Compile(file, flags...)
}
