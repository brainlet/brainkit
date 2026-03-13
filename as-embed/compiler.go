package asembed

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/brainlet/brainkit/jsbridge"
)

// ErrCompilerDead is returned when a compilation timed out or panicked,
// leaving the QJS runtime in a corrupted state. The Compiler must be
// closed and recreated.
var ErrCompilerDead = fmt.Errorf("as-embed: compiler runtime is dead (timed out or panicked)")

var sourceImportPattern = regexp.MustCompile(`(?m)(?:^|[\s;])(?:import|export)\s+(?:[^"'` + "`" + `]*?\s+from\s+)?["']([^"']+)["']`)

// Compiler wraps a QuickJS bridge with the AssemblyScript compiler loaded,
// ready to compile AS source files to Wasm via the real Binaryen C API.
type Compiler struct {
	bridge *jsbridge.Bridge
	memory *LinearMemory
	dead   bool // set when the QJS runtime is corrupted
}

// CompilerConfig controls Compiler creation.
type CompilerConfig struct {
	MemoryLimit  int // bytes; default 512MB
	MaxStackSize int // bytes; default 256MB (effectively disables QuickJS stack check)
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
	if cfg.MemoryLimit == 0 {
		cfg.MemoryLimit = 512 * 1024 * 1024 // 512MB
	}
	if cfg.MaxStackSize == 0 {
		// 256MB effectively disables QuickJS's stack overflow check.
		// The AS compiler's deep recursion (200-500 JS frames during std
		// library compilation) combined with CGo stack position variability
		// causes QuickJS's stack_top-based detection to fire false positives
		// at smaller limits (8MB). Setting 256MB puts the check threshold
		// well below the OS thread stack, so only real exhaustion (SIGSEGV)
		// triggers. compileWithRecover() catches panics from such cases.
		cfg.MaxStackSize = 256 * 1024 * 1024
	}

	b, err := jsbridge.New(jsbridge.Config{
		MemoryLimit:  cfg.MemoryLimit,
		MaxStackSize: cfg.MaxStackSize,
		GCThreshold:  4 * 1024 * 1024, // 4MB — auto-GC to prevent heap accumulation across compilations
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
		return nil, fmt.Errorf("as-embed: create bridge: %w", err)
	}

	lm := NewLinearMemory()
	RegisterMemoryBridge(b.Context(), lm)
	RegisterBinaryenBridge(b.Context(), lm)
	RegisterBinaryenBridgeImpl(b.Context(), lm)

	if err := LoadShim(b); err != nil {
		b.Close()
		return nil, fmt.Errorf("as-embed: load shim: %w", err)
	}

	if err := LoadBundle(b); err != nil {
		b.Close()
		return nil, fmt.Errorf("as-embed: load bundle: %w", err)
	}

	return &Compiler{bridge: b, memory: lm}, nil
}

// Dead returns true if the compiler's QJS runtime is corrupted
// (e.g., after a timeout or panic). The Compiler must be closed and recreated.
func (c *Compiler) Dead() bool { return c.dead }

// Close releases all resources held by the Compiler.
// If the runtime is dead (corrupted), Close uses a short timeout to avoid
// hanging on FreeQJSRuntime with corrupted state.
func (c *Compiler) Close() {
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
	if c.dead {
		return nil, ErrCompilerDead
	}

	if opts.Timeout > 0 {
		type compileResult struct {
			result *CompileResult
			err    error
		}
		ch := make(chan compileResult, 1)
		go func() {
			r, err := c.compileWithRecover(sources, opts)
			ch <- compileResult{r, err}
		}()
		select {
		case res := <-ch:
			return res.result, res.err
		case <-time.After(opts.Timeout):
			c.dead = true
			return nil, fmt.Errorf("as-embed: compilation timed out after %v", opts.Timeout)
		}
	}

	return c.compileWithRecover(sources, opts)
}

func (c *Compiler) compileWithRecover(sources map[string]string, opts CompileOptions) (result *CompileResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			c.dead = true
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
		var program = null;
		var options = null;
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

			options = asc.newOptions();
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

			program = asc.newProgram(options);

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

			// Release the Program — its internal Maps (filesByName, elementsByName,
			// instancesByName, etc.) and cached prototypes accumulate across compilations
			// and increase GC traversal depth which eventually causes stack overflow.
			program = null;
			options = null;
			topLevelLib = null;
			onDemandSources = null;
			userSources = null;

			if (errors.length > 0) {
				if (module) {
					try { module.dispose(); } catch (_) {}
				}
				module = null;
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
			module = null;
			program = null;
			options = null;
			return JSON.stringify({ error: e.message + "\n" + (e.stack || "") });
		}
		})()
	`, string(topLevelJSON), string(onDemandJSON), string(userSourcesJSON), string(entryFileJSON),
		string(runtimeTextJSON), string(runtimePathJSON),
		opts.OptimizeLevel, opts.ShrinkLevel, opts.Debug,
		fmt.Sprintf("%q", runtime))

	val, err := c.bridge.Eval("compile.js", js)
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

func (c *Compiler) disposeLastModule() {
	if c == nil || c.bridge == nil || c.dead {
		return
	}

	func() {
		defer func() { recover() }()
		val, err := c.bridge.Eval("dispose-module.js", `
			(function() {
				var module = globalThis.__as_last_module;
				globalThis.__as_last_module = null;
				if (module) module.dispose();
				return 0;
			})()
		`)
		if err != nil {
			return
		}
		val.Free()
	}()

	// Force QuickJS garbage collection to reclaim memory from disposed modules.
	// Without this, sequential compilations accumulate unreachable JS objects
	// until the runtime's memory limit is hit.
	c.bridge.Runtime().RunGC()
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
