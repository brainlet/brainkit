// testutil.go provides shared test infrastructure for the wasm-kit test suite.
//
// It extracts the common compile pipeline and wazero runtime instantiation
// so that compiler, runtime, allocator, and glue tests can all share the same
// reliable foundation.
package tests

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"unicode/utf16"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/compiler"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/parser"
	"github.com/brainlet/brainkit/wasm-kit/program"
)

// ---------------------------------------------------------------------------
// Path helpers
// ---------------------------------------------------------------------------

func testsRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filename)
}

func fixturesRoot() string {
	return filepath.Join(testsRoot(), "compiler")
}

func stdAssemblyRoot() string {
	return filepath.Join(testsRoot(), "..", "std", "assembly")
}

func allocatorsRoot() string {
	return filepath.Join(testsRoot(), "allocators")
}

// ---------------------------------------------------------------------------
// Std library source caching (parsed once, reused across tests)
// ---------------------------------------------------------------------------

var (
	stdSourcesOnce  sync.Once
	stdSourcesCache []string
)

// collectStdSources walks std/assembly/ for all .ts files.
// Results are cached for the process lifetime.
func collectStdSources(t *testing.T) []string {
	t.Helper()
	stdSourcesOnce.Do(func() {
		root := stdAssemblyRoot()
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || filepath.Ext(path) != ".ts" {
				return nil
			}
			stdSourcesCache = append(stdSourcesCache, path)
			return nil
		})
		if err != nil {
			// can't use t.Fatal inside sync.Once — just leave cache empty
			stdSourcesCache = nil
		}
		sort.Strings(stdSourcesCache)
	})
	if stdSourcesCache == nil {
		t.Fatalf("std sources not found at %s", stdAssemblyRoot())
	}
	return stdSourcesCache
}

// ---------------------------------------------------------------------------
// Compile pipeline
// ---------------------------------------------------------------------------

// CompileResult holds everything produced by a compilation.
type CompileResult struct {
	Module      *module.Module
	Diagnostics []*diagnostics.DiagnosticMessage
	Panicked    bool
	PanicValue  any
}

// compileFixture parses and compiles a fixture, returning the result.
// This is the core compile pipeline shared by all test levels.
func compileFixture(t *testing.T, basename string, release bool, config fixtureConfig) CompileResult {
	t.Helper()

	var result CompileResult

	// Catch panics anywhere in the pipeline
	defer func() {
		if r := recover(); r != nil {
			result.Panicked = true
			result.PanicValue = r
			buf := make([]byte, 8192)
			n := runtime.Stack(buf, false)
			t.Logf("PANIC: %v\n%s", r, buf[:n])
		}
	}()

	opts := program.NewOptions()
	if release {
		opts.OptimizeLevelHint = 3
		opts.DebugInfo = false
	} else {
		opts.DebugInfo = true
	}

	if skip := applyConfig(opts, config); skip != "" {
		t.Skipf("skipping: %s", skip)
	}

	// Parse the test source
	p := parser.NewParser(nil)
	sourcePath := filepath.Join(fixturesRoot(), basename+".ts")
	sourceText, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source %s: %v", sourcePath, err)
	}
	p.ParseFile(string(sourceText), basename+".ts", true)

	// Parse std library
	stdFiles := collectStdSources(t)
	for _, filename := range stdFiles {
		data, err := os.ReadFile(filename)
		if err != nil {
			t.Fatalf("read std %s: %v", filename, err)
		}
		relativePath, err := filepath.Rel(stdAssemblyRoot(), filename)
		if err != nil {
			t.Fatalf("rel %s: %v", filename, err)
		}
		logicalPath := common.LIBRARY_PREFIX + filepath.ToSlash(relativePath)
		p.ParseFile(string(data), logicalPath, false)
	}

	// Drain parser file queue
	for next := p.NextFile(); next != ""; next = p.NextFile() {
	}

	parserDiags := append([]*diagnostics.DiagnosticMessage(nil), p.Diagnostics...)
	sources := p.Sources()
	p.Finish()

	prog := program.NewProgram(opts, nil)
	prog.Sources = sources
	c := compiler.NewCompiler(prog)
	mod := c.CompileProgram()

	if release && mod != nil {
		mod.Optimize(
			int(opts.OptimizeLevelHint),
			int(opts.ShrinkLevelHint),
			opts.DebugInfo,
			opts.ZeroFilledMemory,
		)
	}

	result.Diagnostics = append(result.Diagnostics, parserDiags...)
	result.Diagnostics = append(result.Diagnostics, prog.Diagnostics...)
	result.Diagnostics = append(result.Diagnostics, c.Diagnostics...)
	result.Module = mod
	return result
}

// compileFixtureToBinary compiles a fixture and emits Wasm binary bytes.
func compileFixtureToBinary(t *testing.T, basename string, release bool, config fixtureConfig) ([]byte, CompileResult) {
	t.Helper()
	result := compileFixture(t, basename, release, config)
	if result.Panicked || result.Module == nil {
		return nil, result
	}
	bin := result.Module.ToBinary("")
	return bin.Binary, result
}

// ---------------------------------------------------------------------------
// Wazero runtime instantiation
// ---------------------------------------------------------------------------

// Glue holds optional host import overrides for fixtures that need custom imports.
type Glue struct {
	// PreInstantiate is called before the Wasm module is instantiated.
	// Use it to register host modules (e.g., rt.NewHostModuleBuilder("declare").Instantiate(ctx)).
	PreInstantiate func(rt wazero.Runtime, ctx context.Context) error

	// PostInstantiate is called after the module is instantiated but before _start/_initialize.
	PostInstantiate func(mod api.Module) error

	// PostStart is called after _start/_initialize completes.
	PostStart func(mod api.Module) error
}

// RuntimeResult holds the outcome of instantiating and running a Wasm module.
type RuntimeResult struct {
	Module    api.Module
	Runtime   wazero.Runtime
	Aborted   bool
	AbortMsg  string
	AbortFile string
	AbortLine uint32
	AbortCol  uint32
	TraceLog  []string
	InstErr   error
	StartErr  error
}

// instantiateAndRun compiles Wasm bytes with wazero, provides standard AS host
// imports (env.abort, env.trace, env.seed), runs _start/_initialize, and returns
// the full result.
func instantiateAndRun(t *testing.T, ctx context.Context, wasmBytes []byte, glue *Glue) *RuntimeResult {
	t.Helper()

	rr := &RuntimeResult{}

	rt := wazero.NewRuntime(ctx)
	rr.Runtime = rt

	// Helper to read AS strings from memory. Captured by the abort/trace closures.
	getString := func(mod api.Module, ptr uint32) string {
		if ptr == 0 {
			return "null"
		}
		mem := mod.Memory()
		if mem == nil {
			return "<no memory>"
		}
		return readASString(mem, ptr)
	}

	// Provide standard AS host imports: env.abort, env.trace, env.seed
	envBuilder := rt.NewHostModuleBuilder("env")

	// abort(message: usize, fileName: usize, lineNumber: u32, columnNumber: u32): void
	envBuilder.NewFunctionBuilder().WithFunc(func(ctx2 context.Context, m api.Module, msgPtr, filePtr, line, col uint32) {
		rr.Aborted = true
		rr.AbortMsg = getString(m, msgPtr)
		rr.AbortFile = getString(m, filePtr)
		rr.AbortLine = line
		rr.AbortCol = col
	}).Export("abort")

	// trace(message: usize, n: i32, a0-a4: f64): void
	envBuilder.NewFunctionBuilder().WithFunc(func(ctx2 context.Context, m api.Module, msgPtr uint32, n int32, a0, a1, a2, a3, a4 float64) {
		msg := getString(m, msgPtr)
		args := []float64{a0, a1, a2, a3, a4}[:n]
		entry := fmt.Sprintf("trace: %s", msg)
		if len(args) > 0 {
			parts := make([]string, len(args))
			for i, a := range args {
				parts[i] = fmt.Sprintf("%v", a)
			}
			entry += " " + strings.Join(parts, ", ")
		}
		rr.TraceLog = append(rr.TraceLog, entry)
	}).Export("trace")

	// seed(): f64 — deterministic seed for tests
	envBuilder.NewFunctionBuilder().WithFunc(func() float64 {
		return float64(0xA5534817) // same as AS tests: make tests deterministic
	}).Export("seed")

	_, err := envBuilder.Instantiate(ctx)
	if err != nil {
		rr.InstErr = fmt.Errorf("env module: %w", err)
		return rr
	}

	// Run glue PreInstantiate if provided
	if glue != nil && glue.PreInstantiate != nil {
		if err := glue.PreInstantiate(rt, ctx); err != nil {
			rr.InstErr = fmt.Errorf("glue preInstantiate: %w", err)
			return rr
		}
	}

	// Compile and instantiate the Wasm module.
	// Use InstantiateWithConfig so we control when _start runs.
	compiled, err := rt.CompileModule(ctx, wasmBytes)
	if err != nil {
		rr.InstErr = fmt.Errorf("compile module: %w", err)
		return rr
	}
	mod, err := rt.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().WithName(""))
	if err != nil {
		rr.InstErr = fmt.Errorf("instantiate: %w", err)
		return rr
	}
	rr.Module = mod

	// Run glue PostInstantiate if provided
	if glue != nil && glue.PostInstantiate != nil {
		if err := glue.PostInstantiate(mod); err != nil {
			rr.InstErr = fmt.Errorf("glue postInstantiate: %w", err)
			return rr
		}
	}

	// Call _start or _initialize if exported
	if startFn := mod.ExportedFunction("_start"); startFn != nil {
		_, err := startFn.Call(ctx)
		if err != nil {
			rr.StartErr = err
		}
	} else if initFn := mod.ExportedFunction("_initialize"); initFn != nil {
		_, err := initFn.Call(ctx)
		if err != nil {
			rr.StartErr = err
		}
	}

	// Run glue PostStart if provided
	if glue != nil && glue.PostStart != nil {
		if err := glue.PostStart(mod); err != nil {
			rr.StartErr = fmt.Errorf("glue postStart: %w", err)
		}
	}

	return rr
}

// ---------------------------------------------------------------------------
// AS string reading
// ---------------------------------------------------------------------------

// readASString reads an AssemblyScript string from wasm linear memory.
// AS strings are UTF-16LE encoded with a 20-byte runtime header:
//   - Offset -20: mmInfo (u32)
//   - Offset -16: gcInfo (u32)
//   - Offset -12: gcInfo2 (u32)
//   - Offset  -8: rtId (u32)
//   - Offset  -4: rtSize (u32) — byte length of the string data
//   - Offset   0: UTF-16LE data
func readASString(mem api.Memory, ptr uint32) string {
	if ptr == 0 {
		return "null"
	}
	// rtSize is at ptr - 4
	rtSizeBytes, ok := mem.Read(ptr-4, 4)
	if !ok {
		return "<read error>"
	}
	rtSize := binary.LittleEndian.Uint32(rtSizeBytes)

	// String data is UTF-16LE at ptr, length = rtSize bytes
	if rtSize == 0 {
		return ""
	}
	data, ok := mem.Read(ptr, rtSize)
	if !ok {
		return "<read error>"
	}

	// Decode UTF-16LE
	u16s := make([]uint16, rtSize/2)
	for i := range u16s {
		u16s[i] = binary.LittleEndian.Uint16(data[i*2:])
	}
	runes := utf16.Decode(u16s)
	return string(runes)
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

func normalize(text string) string {
	return strings.ReplaceAll(text, "\r\n", "\n")
}

func firstDiffPos(a, b string) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

func formatDiagnostics(diags []*diagnostics.DiagnosticMessage) string {
	var b strings.Builder
	for i, d := range diags {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(d.String())
	}
	return b.String()
}
