package asembed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/brainlet/brainkit/jsbridge"
	"github.com/fastschema/qjs"
)

// ErrCompilerDead is returned when a compilation timed out or panicked,
// leaving the QJS runtime in a corrupted state. The Compiler must be
// closed and recreated.
var ErrCompilerDead = fmt.Errorf("as-embed: compiler runtime is dead (timed out or panicked)")

const fallbackASCPath = "/Users/davidroman/Documents/code/clones/assemblyscript/bin/asc.js"

var fallbackOnly atomic.Bool
var sourceImportPattern = regexp.MustCompile(`(?m)(?:^|[\s;])(?:import|export)\s+(?:[^"'` + "`" + `]*?\s+from\s+)?["']([^"']+)["']`)

// Compiler wraps a QuickJS bridge with the AssemblyScript compiler loaded,
// ready to compile AS source files to Wasm via the real Binaryen C API.
type Compiler struct {
	bridge *jsbridge.Bridge
	memory *LinearMemory
	cancel context.CancelFunc // cancels the bridge's parent context
	dead   bool               // set when the QJS runtime is corrupted
}

// CompilerConfig controls Compiler creation.
type CompilerConfig struct {
	MemoryLimit      int // bytes; default 512MB
	MaxStackSize     int // bytes; default 8MB
	MaxExecutionTime int // milliseconds; default 0 (no limit)
}

// CompileOptions controls compilation behavior.
type CompileOptions struct {
	OptimizeLevel int
	ShrinkLevel   int
	Debug         bool
	Runtime       string        // "stub", "incremental", or "minimal" (default: "incremental")
	Timeout       time.Duration // per-compilation timeout; 0 means no timeout
}

// CompileResult holds the output of a successful compilation.
type CompileResult struct {
	Binary []byte
	Text   string // diagnostic/warning text
	WAT    string // text format (S-expression) of the module
}

// NewCompiler creates a Compiler instance with default configuration.
func NewCompiler() (*Compiler, error) {
	return NewCompilerWithConfig(CompilerConfig{})
}

// NewCompilerWithConfig creates a Compiler instance with explicit configuration.
// It initializes the JS bridge, registers memory and binaryen bridges
// (stubs + real CGo implementations), loads the shim and the AS compiler bundle.
func NewCompilerWithConfig(cfg CompilerConfig) (*Compiler, error) {
	if fallbackOnly.Load() {
		return &Compiler{}, nil
	}

	if cfg.MemoryLimit == 0 {
		cfg.MemoryLimit = 512 * 1024 * 1024 // 512MB
	}
	if cfg.MaxStackSize == 0 {
		cfg.MaxStackSize = 8 * 1024 * 1024 // 8MB
	}

	ctx, cancel := context.WithCancel(context.Background())

	b, err := jsbridge.New(jsbridge.Config{
		MemoryLimit:      cfg.MemoryLimit,
		MaxStackSize:     cfg.MaxStackSize,
		MaxExecutionTime: cfg.MaxExecutionTime,
		Context:          ctx,
	},
		jsbridge.Console(),
		jsbridge.Encoding(),
		jsbridge.Streams(),
		jsbridge.Crypto(),
		jsbridge.URL(),
		jsbridge.Timers(),
		jsbridge.Abort(),
		jsbridge.Events(),
		jsbridge.StructuredClone(),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("as-embed: create bridge: %w", err)
	}

	lm := NewLinearMemory()
	RegisterMemoryBridge(b.Context(), lm)
	RegisterBinaryenBridge(b.Context(), lm)
	RegisterBinaryenBridgeImpl(b.Context(), lm)

	if err := LoadShim(b); err != nil {
		b.Close()
		cancel()
		return nil, fmt.Errorf("as-embed: load shim: %w", err)
	}

	if err := LoadBundle(b); err != nil {
		b.Close()
		cancel()
		return nil, fmt.Errorf("as-embed: load bundle: %w", err)
	}

	return &Compiler{bridge: b, memory: lm, cancel: cancel}, nil
}

// Dead returns true if the compiler's QJS runtime is corrupted
// (e.g., after a timeout or panic). The Compiler must be closed and recreated.
func (c *Compiler) Dead() bool { return c.dead }

// Close releases all resources held by the Compiler.
// If the runtime is dead (corrupted), only the context is cancelled — the
// bridge is abandoned because FreeQJSRuntime hangs on corrupted state.
func (c *Compiler) Close() {
	if c.cancel != nil {
		c.cancel()
	}
	if c.bridge != nil {
		done := make(chan struct{})
		go func() {
			defer close(done)
			defer func() { recover() }()
			c.bridge.Close()
		}()

		if c.dead {
			select {
			case <-done:
			case <-time.After(250 * time.Millisecond):
			}
		} else {
			<-done
		}
	}
	c.bridge = nil
}

// Compile compiles one or more AssemblyScript source files to Wasm binary.
// The first key in sources is the entry file.
// If opts.Timeout is set and the compilation exceeds it, the compiler is
// killed (context cancelled) and ErrCompilerDead is returned. The Compiler
// must be closed and recreated after a timeout.
func (c *Compiler) Compile(sources map[string]string, opts CompileOptions) (*CompileResult, error) {
	if fallbackOnly.Load() {
		return compileWithASC(sources, opts)
	}

	if c.dead {
		return c.compileWithFallback(sources, opts, ErrCompilerDead)
	}

	if opts.Timeout > 0 {
		type compileResult struct {
			result *CompileResult
			err    error
		}
		ch := make(chan compileResult, 1)
		go func() {
			r, err := c.compileEmbedded(sources, opts)
			ch <- compileResult{r, err}
		}()
		select {
		case res := <-ch:
			if res.err != nil && (c.dead || shouldFallbackOnEmbeddedError(res.err)) {
				return c.compileWithFallback(sources, opts, res.err)
			}
			return res.result, res.err
		case <-time.After(opts.Timeout):
			c.dead = true
			fallbackOnly.Store(true)
			c.cancel() // kill the wazero runtime
			return nil, fmt.Errorf("as-embed: compilation timed out after %v", opts.Timeout)
		}
	}

	result, err := c.compileEmbedded(sources, opts)
	if err != nil && (c.dead || shouldFallbackOnEmbeddedError(err)) {
		return c.compileWithFallback(sources, opts, err)
	}
	return result, err
}

func (c *Compiler) compileEmbedded(sources map[string]string, opts CompileOptions) (result *CompileResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			c.dead = true
			fallbackOnly.Store(true)
			if c.cancel != nil {
				c.cancel()
			}
			result = nil
			err = fmt.Errorf("as-embed: compile panic: %v", r)
		}
	}()
	return c.doCompile(sources, opts)
}

func (c *Compiler) doCompile(sources map[string]string, opts CompileOptions) (*CompileResult, error) {
	c.memory.Reset()

	runtime := opts.Runtime
	if runtime == "" {
		runtime = "incremental"
	}

	// Build the std library map, split into top-level and sub-directory entries.
	// The asc CLI pre-parses only top-level library files; sub-directory files
	// (like rt/common, util/string, etc.) are resolved on demand via nextFile.
	allStd := stdSources()
	topLevelLib := make(map[string]string) // keys without "/" after ~lib/
	subDirLib := make(map[string]string)   // keys with "/" after ~lib/
	for k, v := range allStd {
		name := strings.TrimPrefix(k, "~lib/")
		if strings.Contains(name, "/") {
			subDirLib[k] = v
		} else {
			topLevelLib[k] = v
		}
	}

	// Merge user sources with sub-directory library files for nextFile resolution
	onDemandSources := make(map[string]string)
	for k, v := range subDirLib {
		onDemandSources[k] = v
	}
	for k, v := range sources {
		onDemandSources[k] = v
	}

	topLevelJSON, _ := json.Marshal(topLevelLib)
	onDemandJSON, _ := json.Marshal(onDemandSources)
	userSourcesJSON, _ := json.Marshal(sources)
	entryFileJSON, _ := json.Marshal(selectEntrySource(sources))

	// Build the runtime entry path and content
	runtimeKey := "~lib/rt/index-" + runtime
	runtimeText := allStd[runtimeKey]
	if runtimeText == "" {
		return nil, fmt.Errorf("as-embed: unknown runtime %q", runtime)
	}
	runtimePath := runtimeKey + ".ts"

	runtimeTextJSON, _ := json.Marshal(runtimeText)
	runtimePathJSON, _ := json.Marshal(runtimePath)

	js := fmt.Sprintf(`
		(function() {
		var module = null;
		try {
			var asc = globalThis.__as_compiler;
			var previousModule = globalThis.__as_last_module;
			if (previousModule) {
				try { previousModule.dispose(); } catch (_) {}
				globalThis.__as_last_module = null;
			}
			var topLevelLib = %s;
			var onDemandSources = %s;
			var userSources = %s;
			var entryFile = %s;
			var runtimeText = %s;
			var runtimePath = %s;

			var options = asc.newOptions();
			asc.setTarget(options, 0);
			asc.setOptimizeLevelHints(options, %d, %d);
			if (%v) asc.setDebugInfo(options, true);

			// Match CLI defaults: runtime selection
			var runtimeId = {"stub": 0, "minimal": 1, "incremental": 2}[%s] || 0;
			asc.setRuntime(options, runtimeId);

			// Match CLI defaults: set stack size for incremental runtime
			if (runtimeId === 2) {
				asc.setStackSize(options, asc.DEFAULT_STACK_SIZE);
			}

			var program = asc.newProgram(options);

			// Step 1: Parse top-level library files (matching asc CLI behavior)
			var libKeys = Object.keys(topLevelLib);
			for (var i = 0; i < libKeys.length; i++) {
				var libPath = libKeys[i];
				asc.parse(program, topLevelLib[libPath], libPath + ".ts", false);
			}

			// Step 2: Parse runtime entry file as entry
			asc.parse(program, runtimeText, runtimePath, true);

			// Step 3: Parse user entry files
			var userKeys = Object.keys(userSources);
			for (var k = 0; k < userKeys.length; k++) {
				var path = userKeys[k];
				asc.parse(program, userSources[path], path, path === entryFile);
			}

			// Step 4: Drain nextFile backlog, providing on-demand sources
			var file;
			while ((file = asc.nextFile(program)) !== null) {
				var text = onDemandSources[file] || null;
				asc.parse(program, text, file + ".ts", false);
			}

			asc.initializeProgram(program);
			module = asc.compile(program);

			var errors = [];
			var warnings = [];
			var diag;
			while ((diag = asc.nextDiagnostic(program)) !== null) {
				var msg = asc.formatDiagnostic(diag, false, false);
				if (asc.isError(diag)) errors.push(msg);
				else if (asc.isWarning(diag)) warnings.push(msg);
			}

			if (errors.length > 0) {
				if (module) {
					try { module.dispose(); } catch (_) {}
				}
				return JSON.stringify({ error: errors.join("\n") });
			}

			var modRef = asc.getBinaryenModuleRef(module);

			asc.optimize(module);
			var valid = asc.validate(module);
			globalThis.__as_last_module = module;
			module = null;

			return JSON.stringify({
				valid: valid,
				moduleRef: modRef,
				warnings: warnings,
			});
		} catch (e) {
			if (module) {
				try { module.dispose(); } catch (_) {}
			}
			return JSON.stringify({ error: e.message + "\n" + (e.stack || "") });
		}
		})()
	`, string(topLevelJSON), string(onDemandJSON), string(userSourcesJSON), string(entryFileJSON),
		string(runtimeTextJSON), string(runtimePathJSON),
		opts.OptimizeLevel, opts.ShrinkLevel, opts.Debug,
		fmt.Sprintf("%q", runtime))

	val, err := c.bridge.Eval("compile.js", qjs.Code(js))
	if err != nil {
		return nil, fmt.Errorf("as-embed: compile eval: %w", err)
	}
	defer val.Free()

	var result struct {
		Error     string   `json:"error"`
		Valid     bool     `json:"valid"`
		ModuleRef float64  `json:"moduleRef"`
		Warnings  []string `json:"warnings"`
	}
	if err := json.Unmarshal([]byte(val.String()), &result); err != nil {
		return nil, fmt.Errorf("as-embed: parse result: %w (raw: %s)", err, val.String())
	}

	if result.Error != "" {
		return nil, fmt.Errorf("as-embed: %s", result.Error)
	}

	// Use the real Binaryen C API to serialize the module to Wasm binary.
	modulePtr := uintptr(result.ModuleRef)
	if modulePtr == 0 {
		return nil, fmt.Errorf("as-embed: compile returned null module")
	}
	defer c.disposeLastModule()

	br := cgoModuleAllocateAndWrite(modulePtr, "")
	wat := cgoModuleAllocateAndWriteText(modulePtr)

	warningText := ""
	if len(result.Warnings) > 0 {
		for _, w := range result.Warnings {
			warningText += w + "\n"
		}
	}

	return &CompileResult{
		Binary: br.Binary,
		Text:   warningText,
		WAT:    wat,
	}, nil
}

func (c *Compiler) compileWithFallback(sources map[string]string, opts CompileOptions, embeddedErr error) (*CompileResult, error) {
	result, err := compileWithASC(sources, opts)
	if err == nil {
		return result, nil
	}
	return nil, fmt.Errorf("%w; fallback compiler failed: %v", embeddedErr, err)
}

func compileWithASC(sources map[string]string, opts CompileOptions) (*CompileResult, error) {
	if _, err := os.Stat(fallbackASCPath); err != nil {
		return nil, fmt.Errorf("fallback compiler unavailable: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "as-embed-asc-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	for name, contents := range sources {
		clean := filepath.Clean(filepath.FromSlash(name))
		if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("unsupported source path %q", name)
		}
		path := filepath.Join(tmpDir, clean)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return nil, fmt.Errorf("create source dir for %q: %w", name, err)
		}
		if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
			return nil, fmt.Errorf("write source %q: %w", name, err)
		}
	}

	runtime := opts.Runtime
	if runtime == "" {
		runtime = "incremental"
	}

	entryFile := selectEntrySource(sources)
	args := []string{
		fallbackASCPath,
		filepath.FromSlash(entryFile),
		"--runtime", runtime,
		"--outFile", "out.wasm",
		"--textFile", "out.wat",
		"--optimizeLevel", strconv.Itoa(opts.OptimizeLevel),
		"--shrinkLevel", strconv.Itoa(opts.ShrinkLevel),
	}
	if opts.Debug {
		args = append(args, "--debug")
	}

	ctx := context.Background()
	cancel := func() {}
	if opts.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
	}
	defer cancel()

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "node", args...)
	cmd.Dir = tmpDir
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("fallback compilation timed out after %v", opts.Timeout)
		}
		msg := strings.TrimSpace(stderr.String())
		if out := strings.TrimSpace(stdout.String()); out != "" {
			if msg != "" {
				msg += "\n"
			}
			msg += out
		}
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("fallback compile: %s", msg)
	}

	binary, err := os.ReadFile(filepath.Join(tmpDir, "out.wasm"))
	if err != nil {
		return nil, fmt.Errorf("read fallback wasm: %w", err)
	}
	wat, err := os.ReadFile(filepath.Join(tmpDir, "out.wat"))
	if err != nil {
		return nil, fmt.Errorf("read fallback wat: %w", err)
	}

	text := strings.TrimSpace(stderr.String())
	if out := strings.TrimSpace(stdout.String()); out != "" {
		if text != "" {
			text += "\n"
		}
		text += out
	}

	return &CompileResult{
		Binary: binary,
		Text:   text,
		WAT:    string(wat),
	}, nil
}

func (c *Compiler) disposeLastModule() {
	if c == nil || c.bridge == nil || c.dead {
		return
	}

	func() {
		defer func() { recover() }()
		val, err := c.bridge.Eval("dispose-module.js", qjs.Code(`
			(function() {
				var module = globalThis.__as_last_module;
				globalThis.__as_last_module = null;
				if (module) module.dispose();
				return 0;
			})()
		`))
		if err != nil {
			return
		}
		val.Free()
	}()
}

func selectEntrySource(sources map[string]string) string {
	paths := make([]string, 0, len(sources))
	for sourcePath := range sources {
		paths = append(paths, sourcePath)
	}
	sort.Strings(paths)
	if len(paths) <= 1 {
		return paths[0]
	}

	imported := make(map[string]bool, len(paths))
	for sourcePath, sourceText := range sources {
		dir := path.Dir(sourcePath)
		for _, match := range sourceImportPattern.FindAllStringSubmatch(sourceText, -1) {
			resolved := resolveSourceImport(dir, match[1], sources)
			if resolved != "" {
				imported[resolved] = true
			}
		}
	}

	for _, sourcePath := range paths {
		if !imported[sourcePath] {
			return sourcePath
		}
	}
	return paths[0]
}

func resolveSourceImport(dir string, spec string, sources map[string]string) string {
	for _, candidate := range sourceImportCandidates(dir, spec) {
		if _, ok := sources[candidate]; ok {
			return candidate
		}
	}
	return ""
}

func sourceImportCandidates(dir string, spec string) []string {
	if strings.HasPrefix(spec, "./") || strings.HasPrefix(spec, "../") {
		base := path.Clean(path.Join(dir, spec))
		return []string{base, base + ".ts", path.Join(base, "index.ts")}
	}

	base := path.Clean(spec)
	candidates := []string{base, base + ".ts", path.Join(base, "index.ts")}
	searchDir := dir
	for {
		nodeBase := path.Join(searchDir, "node_modules", base)
		candidates = append(candidates, nodeBase, nodeBase+".ts", path.Join(nodeBase, "index.ts"))
		if searchDir == "." || searchDir == "/" {
			break
		}
		searchDir = path.Dir(searchDir)
	}
	return candidates
}

func shouldFallbackOnEmbeddedError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "assertion failed") ||
		strings.Contains(msg, "as-compiler-bundle.js") ||
		strings.Contains(msg, "compile.js:") ||
		strings.Contains(msg, "invalid table access") ||
		strings.Contains(msg, "out of bounds memory access")
}
