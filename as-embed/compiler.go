package asembed

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/jsbridge"
	"github.com/fastschema/qjs"
)

// Compiler wraps a QuickJS bridge with the AssemblyScript compiler loaded,
// ready to compile AS source files to Wasm via the real Binaryen C API.
type Compiler struct {
	bridge *jsbridge.Bridge
	memory *LinearMemory
}

// CompileOptions controls compilation behavior.
type CompileOptions struct {
	OptimizeLevel int
	ShrinkLevel   int
	Debug         bool
	Runtime       string // "stub", "incremental", or "minimal" (default: "stub")
}

// CompileResult holds the output of a successful compilation.
type CompileResult struct {
	Binary []byte
	Text   string // diagnostic/warning text
}

// NewCompiler creates a Compiler instance. It initializes the JS bridge,
// registers memory and binaryen bridges (stubs + real CGo implementations),
// loads the shim and the AS compiler bundle.
func NewCompiler() (*Compiler, error) {
	b, err := jsbridge.New(jsbridge.Config{},
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

// Close releases all resources held by the Compiler.
func (c *Compiler) Close() {
	if c.bridge != nil {
		c.bridge.Close()
	}
}

// Compile compiles one or more AssemblyScript source files to Wasm binary.
// The first key in sources is the entry file.
func (c *Compiler) Compile(sources map[string]string, opts CompileOptions) (*CompileResult, error) {
	c.memory.Reset()

	runtime := opts.Runtime
	if runtime == "" {
		runtime = "stub"
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
		try {
			var asc = globalThis.__as_compiler;
			var topLevelLib = %s;
			var onDemandSources = %s;
			var userSources = %s;
			var runtimeText = %s;
			var runtimePath = %s;

			var options = asc.newOptions();
			asc.setOptimizeLevelHints(options, %d, %d);
			if (%v) asc.setDebugInfo(options, true);

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
			var entryFile = Object.keys(userSources)[0];
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
			var module = asc.compile(program);

			var errors = [];
			var warnings = [];
			var diag;
			while ((diag = asc.nextDiagnostic(program)) !== null) {
				var msg = asc.formatDiagnostic(diag, false, false);
				if (asc.isError(diag)) errors.push(msg);
				else if (asc.isWarning(diag)) warnings.push(msg);
			}

			if (errors.length > 0) {
				return JSON.stringify({ error: errors.join("\n") });
			}

			asc.optimize(module);
			var valid = asc.validate(module);

			return JSON.stringify({
				valid: valid,
				moduleRef: asc.getBinaryenModuleRef(module),
				warnings: warnings,
			});
		} catch (e) {
			return JSON.stringify({ error: e.message + "\n" + (e.stack || "") });
		}
		})()
	`, string(topLevelJSON), string(onDemandJSON), string(userSourcesJSON),
		string(runtimeTextJSON), string(runtimePathJSON),
		opts.OptimizeLevel, opts.ShrinkLevel, opts.Debug)

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

	br := cgoModuleAllocateAndWrite(modulePtr, "")

	warningText := ""
	if len(result.Warnings) > 0 {
		for _, w := range result.Warnings {
			warningText += w + "\n"
		}
	}

	return &CompileResult{
		Binary: br.Binary,
		Text:   warningText,
	}, nil
}
