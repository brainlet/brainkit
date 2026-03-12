// gen-bridge parses the AssemblyScript binaryen.js glue file and generates
// Go + JS bridge code for all exported Binaryen C API functions.
//
// Usage:
//
//	go run ./cmd/gen-bridge <path-to-binaryen.js> [--output-dir <dir>]
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Task 6: Parse binaryen.js
// ---------------------------------------------------------------------------

// identRe matches lines like "  _BinaryenTypeCreate," inside the
// export const { ... } = binaryen; destructuring block.
var identRe = regexp.MustCompile(`^\s+(_\w+),?\s*$`)

// parseGlueFile extracts all identifiers from the destructuring export in
// binaryen.js.  It returns them in source order.
func parseGlueFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var names []string
	seen := make(map[string]bool)
	inBlock := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// Detect the opening "export const {" line.
		if !inBlock {
			if strings.Contains(line, "export const {") {
				inBlock = true
			}
			continue
		}

		// Detect the closing "} = binaryen;" line.
		if strings.Contains(line, "} = binaryen") {
			break
		}

		// Skip comment lines (e.g. "  // _BinaryenHeapTypeExn,")
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}

		m := identRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := m[1]
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no identifiers found in %s", path)
	}
	return names, nil
}

// ---------------------------------------------------------------------------
// Task 7: Classify functions by signature pattern
// ---------------------------------------------------------------------------

// SignatureKind categorises a Binaryen C API function by its calling pattern.
type SignatureKind int

const (
	SigMemoryOp       SignatureKind = iota // _malloc, _free, __* — handled by bridge.go
	SigNoArgs                              // _BinaryenTypeNone() → value
	SigModuleOnly                          // _BinaryenModuleCreate() → handle
	SigModuleArg                           // _BinaryenModuleValidate(module) → int
	SigExprGetter                          // _BinaryenBlockGetName(expr) → value
	SigExprSetter                          // _BinaryenBlockSetName(expr, value) → void
	SigExprConstructor                     // _BinaryenBlock(module, ...) → ExpressionRef
	SigFuncOp                              // _BinaryenAddFunction(module, name, ...) → varies
	SigGlobalSetting                       // _BinaryenGetOptimizeLevel() / _BinaryenSetOptimizeLevel(level)
	SigLiteral                             // _BinaryenLiteralInt32(litPtr, value) → fills struct
	SigComplex                             // Needs manual handling
)

var sigNames = [...]string{
	"SigMemoryOp",
	"SigNoArgs",
	"SigModuleOnly",
	"SigModuleArg",
	"SigExprGetter",
	"SigExprSetter",
	"SigExprConstructor",
	"SigFuncOp",
	"SigGlobalSetting",
	"SigLiteral",
	"SigComplex",
}

func (s SignatureKind) String() string {
	if int(s) < len(sigNames) {
		return sigNames[s]
	}
	return fmt.Sprintf("SignatureKind(%d)", s)
}

// noArgTypes lists the _BinaryenType* / _BinaryenHeapType* functions that
// take no arguments and return a constant type/heap-type value.
var noArgTypes = map[string]bool{
	// Type queries
	"_BinaryenTypeFuncref":     true,
	"_BinaryenTypeExternref":   true,
	"_BinaryenTypeAnyref":      true,
	"_BinaryenTypeEqref":       true,
	"_BinaryenTypeI31ref":      true,
	"_BinaryenTypeStructref":   true,
	"_BinaryenTypeArrayref":    true,
	"_BinaryenTypeStringref":   true,
	"_BinaryenTypeNullref":     true,
	"_BinaryenTypeNullExternref": true,
	"_BinaryenTypeNullFuncref":  true,
	// Heap type queries
	"_BinaryenHeapTypeFunc":   true,
	"_BinaryenHeapTypeExt":    true,
	"_BinaryenHeapTypeAny":    true,
	"_BinaryenHeapTypeEq":     true,
	"_BinaryenHeapTypeI31":    true,
	"_BinaryenHeapTypeStruct": true,
	"_BinaryenHeapTypeArray":  true,
	"_BinaryenHeapTypeString": true,
	"_BinaryenHeapTypeNone":   true,
	"_BinaryenHeapTypeNoext":  true,
	"_BinaryenHeapTypeNofunc": true,
}

// globalSettingPrefixes are Get/Set pairs for global config (no module arg).
var globalSettingPrefixes = []string{
	"_BinaryenGetOptimizeLevel",
	"_BinaryenSetOptimizeLevel",
	"_BinaryenGetShrinkLevel",
	"_BinaryenSetShrinkLevel",
	"_BinaryenGetDebugInfo",
	"_BinaryenSetDebugInfo",
	"_BinaryenGetTrapsNeverHappen",
	"_BinaryenSetTrapsNeverHappen",
	"_BinaryenGetClosedWorld",
	"_BinaryenSetClosedWorld",
	"_BinaryenGetLowMemoryUnused",
	"_BinaryenSetLowMemoryUnused",
	"_BinaryenGetZeroFilledMemory",
	"_BinaryenSetZeroFilledMemory",
	"_BinaryenGetFastMath",
	"_BinaryenSetFastMath",
	"_BinaryenGetGenerateStackIR",
	"_BinaryenSetGenerateStackIR",
	"_BinaryenGetOptimizeStackIR",
	"_BinaryenSetOptimizeStackIR",
	"_BinaryenGetPassArgument",
	"_BinaryenSetPassArgument",
	"_BinaryenClearPassArguments",
	"_BinaryenHasPassToSkip",
	"_BinaryenAddPassToSkip",
	"_BinaryenClearPassesToSkip",
	"_BinaryenGetAlwaysInlineMaxSize",
	"_BinaryenSetAlwaysInlineMaxSize",
	"_BinaryenGetFlexibleInlineMaxSize",
	"_BinaryenSetFlexibleInlineMaxSize",
	"_BinaryenGetOneCallerInlineMaxSize",
	"_BinaryenSetOneCallerInlineMaxSize",
	"_BinaryenGetAllowInliningFunctionsWithLoops",
	"_BinaryenSetAllowInliningFunctionsWithLoops",
}

var globalSettingSet map[string]bool

func init() {
	globalSettingSet = make(map[string]bool, len(globalSettingPrefixes))
	for _, name := range globalSettingPrefixes {
		globalSettingSet[name] = true
	}
}

// getterRe matches names like _BinaryenBlockGetName, _BinaryenLoadIsAtomic,
// _BinaryenCallRefGetOperandAt, etc.  The key pattern is a Get/Is/Has
// suffix after the node-type prefix.
var getterRe = regexp.MustCompile(`^_Binaryen\w+(Get\w+|Is\w+|Has\w+)$`)

// setterRe matches names like _BinaryenBlockSetName, _BinaryenLoadSetAtomic, etc.
var setterRe = regexp.MustCompile(`^_Binaryen\w+Set\w+$`)

// funcOpRe matches module-level operations like _BinaryenAddFunction,
// _BinaryenGetFunction, _BinaryenRemoveFunction, etc.
var funcOpRe = regexp.MustCompile(`^_Binaryen(Add|Get|Remove|GetNum)\w+$`)

func classifyFunction(name string) SignatureKind {
	// Memory ops: _malloc, _free, __*
	if name == "_malloc" || name == "_free" || strings.HasPrefix(name, "__") {
		return SigMemoryOp
	}

	// Literal constructors
	if strings.HasPrefix(name, "_BinaryenLiteral") || name == "_BinaryenSizeofLiteral" {
		return SigLiteral
	}

	// No-arg type/heap-type queries
	if noArgTypes[name] {
		return SigNoArgs
	}

	// Module create (no-arg, returns handle)
	if name == "_BinaryenModuleCreate" {
		return SigModuleOnly
	}

	// Global settings (no module argument)
	if globalSettingSet[name] {
		return SigGlobalSetting
	}

	// Module-level operations that take module as first arg
	// These are operations like _BinaryenModuleDispose, _BinaryenModuleValidate,
	// _BinaryenModuleParse, _BinaryenModulePrint, etc.
	if strings.HasPrefix(name, "_BinaryenModule") && name != "_BinaryenModuleCreate" {
		return SigModuleArg
	}

	// Setters: _BinaryenXxxSetYyy — must check before getters
	// because "Set" also matches funcOpRe.
	// But exclude module-level Set operations (already handled above).
	if setterRe.MatchString(name) && !globalSettingSet[name] {
		// Distinguish between expr setters and module-level setters.
		// Module-level setters like _BinaryenSetStart are SigModuleArg;
		// but they were already caught by the _BinaryenModule prefix.
		// _BinaryenSetStart / _BinaryenSetMemory are top-level module ops.
		if name == "_BinaryenSetStart" || name == "_BinaryenSetMemory" {
			return SigModuleArg
		}
		return SigExprSetter
	}

	// Getters: _BinaryenXxxGetYyy, _BinaryenXxxIsYyy, _BinaryenXxxHasYyy
	if getterRe.MatchString(name) && !globalSettingSet[name] {
		// _BinaryenGetStart is a module-level getter.
		if name == "_BinaryenGetStart" {
			return SigModuleArg
		}
		return SigExprGetter
	}

	// Module-level function/global/tag/table/export/element/data management
	if funcOpRe.MatchString(name) {
		return SigFuncOp
	}

	// Relooper and ExpressionRunner and TypeBuilder are complex
	if strings.HasPrefix(name, "_Relooper") ||
		strings.HasPrefix(name, "_ExpressionRunner") ||
		strings.HasPrefix(name, "_TypeBuilder") {
		return SigComplex
	}

	// Remaining _Binaryen* names that don't have Get/Set/Is/Has suffixes
	// are typically expression constructors: _BinaryenBlock, _BinaryenIf, etc.
	// Also includes _BinaryenConst, _BinaryenUnary, _BinaryenBinary, etc.
	if strings.HasPrefix(name, "_Binaryen") {
		return SigExprConstructor
	}

	return SigComplex
}

// ---------------------------------------------------------------------------
// Task 8: Generate bridge code
// ---------------------------------------------------------------------------

func generateBridgeGo(names []string, outputDir string) error {
	path := filepath.Join(outputDir, "binaryen_bridge.go")

	// Separate memory ops from bridge ops.
	var bridgeNames []string
	for _, name := range names {
		if classifyFunction(name) == SigMemoryOp {
			continue // handled by RegisterMemoryBridge in bridge.go
		}
		bridgeNames = append(bridgeNames, name)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)

	fmt.Fprintf(w, "// Code generated by gen-bridge on %s. DO NOT EDIT.\n",
		time.Now().Format("2006-01-02"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, "package asembed")
	fmt.Fprintln(w)
	fmt.Fprintln(w, `import "github.com/fastschema/qjs"`)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "// RegisterBinaryenBridge registers stub implementations for all Binaryen")
	fmt.Fprintln(w, "// C API functions exported by the AS compiler's binaryen.js glue.")
	fmt.Fprintln(w, "// Each stub returns 0 (as float64). They will be replaced with real")
	fmt.Fprintln(w, "// CGo implementations in later tasks.")
	fmt.Fprintln(w, "//")
	fmt.Fprintf(w, "// Total functions: %d (excluding %d memory ops handled by RegisterMemoryBridge)\n",
		len(bridgeNames), len(names)-len(bridgeNames))
	fmt.Fprintln(w, "func RegisterBinaryenBridge(ctx *qjs.Context, lm *LinearMemory) {")

	for _, name := range bridgeNames {
		kind := classifyFunction(name)
		fmt.Fprintf(w, "\t// %s (%s)\n", name, kind)
		fmt.Fprintf(w, "\tctx.SetFunc(%q, func(this *qjs.This) (*qjs.Value, error) {\n", name)
		fmt.Fprintf(w, "\t\treturn this.Context().NewFloat64(0), nil\n")
		fmt.Fprintf(w, "\t})\n\n")
	}

	fmt.Fprintln(w, "}")
	return w.Flush()
}

func generateShimJS(outputDir string) error {
	path := filepath.Join(outputDir, "binaryen_shim.js")

	content := `// binaryen_shim.js — module resolution glue for the esbuild bundle.
// Code generated by gen-bridge. DO NOT EDIT.
//
// All _Binaryen* functions are registered as globals by Go's RegisterBinaryenBridge().
// This shim routes binaryen.X to globalThis.X via a Proxy.

(function() {
  if (typeof self === "undefined") globalThis.self = globalThis;
  if (typeof global === "undefined") globalThis.global = globalThis;

  var binaryenProxy = new Proxy({}, {
    get: function(target, prop) {
      if (typeof prop === "string" && typeof globalThis[prop] !== "undefined") {
        return globalThis[prop];
      }
      // Return a stub function for any unregistered property.
      return function() { return 0; };
    },
    has: function(target, prop) {
      return true; // Claim all properties exist to prevent destructuring failures.
    }
  });

  globalThis.binaryen = binaryenProxy;

  if (typeof globalThis.require === "undefined") {
    globalThis.require = function(name) {
      if (name === "binaryen") return globalThis.binaryen;
      throw new Error("Cannot require module: " + name);
    };
  }
})();
`

	return os.WriteFile(path, []byte(content), 0644)
}

// ---------------------------------------------------------------------------
// Summary report
// ---------------------------------------------------------------------------

func printSummary(names []string) {
	counts := make(map[SignatureKind]int)
	for _, name := range names {
		counts[classifyFunction(name)]++
	}

	fmt.Printf("\n=== gen-bridge summary ===\n")
	fmt.Printf("Total identifiers parsed: %d\n\n", len(names))

	// Sort by kind for stable output.
	var kinds []SignatureKind
	for k := range counts {
		kinds = append(kinds, k)
	}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })

	for _, k := range kinds {
		fmt.Printf("  %-20s %4d\n", k, counts[k])
	}
	fmt.Println()
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: gen-bridge <binaryen.js> [--output-dir <dir>]\n")
		os.Exit(1)
	}

	glueFile := os.Args[1]
	outputDir := filepath.Join(filepath.Dir(os.Args[0]), "..", "..")

	// Parse --output-dir flag.
	for i := 2; i < len(os.Args); i++ {
		if os.Args[i] == "--output-dir" && i+1 < len(os.Args) {
			outputDir = os.Args[i+1]
			i++
		}
	}

	// If running via `go run`, os.Args[0] is a temp path.
	// Default to the as-embed directory relative to the working directory
	// if --output-dir is not explicitly set.
	if !hasFlag("--output-dir") {
		// Assume we're run from the as-embed directory.
		outputDir = "."
	}

	fmt.Printf("Parsing %s ...\n", glueFile)
	names, err := parseGlueFile(glueFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Found %d identifiers\n", len(names))

	printSummary(names)

	fmt.Printf("Generating %s/binaryen_bridge.go ...\n", outputDir)
	if err := generateBridgeGo(names, outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "generate bridge.go: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generating %s/binaryen_shim.js ...\n", outputDir)
	if err := generateShimJS(outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "generate shim.js: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Done.")
}

func hasFlag(flag string) bool {
	for _, arg := range os.Args[2:] {
		if arg == flag {
			return true
		}
	}
	return false
}
