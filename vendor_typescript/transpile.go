// Package typescript provides a native Go TypeScript-to-JavaScript transpiler.
//
// It vendors microsoft/typescript-go's parser, transformer, and printer pipeline
// to strip TypeScript types and emit clean JavaScript. No type checking.
// No dependency resolution. Pure syntactic transform.
//
// This is the same approach as tsc --isolatedModules --noCheck.
package typescript

import (
	"fmt"

	"github.com/brainlet/brainkit/vendor_typescript/internal/ast"
	"github.com/brainlet/brainkit/vendor_typescript/internal/binder"
	"github.com/brainlet/brainkit/vendor_typescript/internal/core"
	"github.com/brainlet/brainkit/vendor_typescript/internal/parser"
	"github.com/brainlet/brainkit/vendor_typescript/internal/printer"
	"github.com/brainlet/brainkit/vendor_typescript/internal/transformers"
	"github.com/brainlet/brainkit/vendor_typescript/internal/transformers/tstransforms"
	"github.com/brainlet/brainkit/vendor_typescript/internal/tspath"
)

// TranspileOptions configures the transpiler.
type TranspileOptions struct {
	// FileName for diagnostics (default: "input.ts")
	FileName string
	// JSX controls JSX transform: "preserve", "react", "react-jsx" (default: "preserve")
	JSX string
}

// Transpile converts TypeScript source to JavaScript by stripping types.
// No type checking. No dependency resolution. Pure syntactic transform.
//
// Pipeline: parse → bind → TypeEraser transform → print
func Transpile(source string, opts TranspileOptions) (string, error) {
	fileName := opts.FileName
	if fileName == "" {
		fileName = "input.ts"
	}

	// 1. Parse the TypeScript source into an AST
	// The parser requires a normalized, absolute path for the filename.
	absFileName := "/" + fileName
	normalizedFileName := tspath.NormalizePath(absFileName)
	parseOpts := ast.SourceFileParseOptions{
		FileName: normalizedFileName,
		Path:     tspath.Path(normalizedFileName),
	}

	sourceFile := parser.ParseSourceFile(parseOpts, source, core.ScriptKindTS)
	if sourceFile == nil {
		return "", fmt.Errorf("typescript: parse returned nil")
	}

	// 2. Bind the source file — resolves symbol references needed by transforms
	binder.BindSourceFile(sourceFile)

	// 3. Set up the transform pipeline
	emitContext := printer.NewEmitContext()

	compilerOptions := &core.CompilerOptions{}
	compilerOptions.Target = core.ScriptTargetESNext
	compilerOptions.Module = core.ModuleKindESNext
	// IsolatedModules mode: per-file transform without checker
	compilerOptions.IsolatedModules = core.TSTrue

	transformOpts := &transformers.TransformOptions{
		Context:         emitContext,
		CompilerOptions: compilerOptions,
		// No Resolver or EmitResolver — we're in isolatedModules mode.
		// TypeEraser doesn't need them.
	}

	// Build the transform chain: TypeEraser strips type annotations.
	// We skip ImportElision (needs checker) and RuntimeSyntax (needs
	// EmitResolver for enum values) — imports pass through as-is,
	// enums stay as-is (our QuickJS runtime handles TS enums fine
	// since we're only stripping types, not downleveling).
	chain := transformers.Chain(
		tstransforms.NewTypeEraserTransformer,
	)

	transformer := chain(transformOpts)
	if transformer == nil {
		return "", fmt.Errorf("typescript: transform chain returned nil")
	}

	// 4. Transform
	transformed := transformer.TransformSourceFile(sourceFile)

	// 5. Print the transformed AST to JavaScript
	p := printer.NewPrinter(printer.PrinterOptions{
		RemoveComments: false,
		NewLine:        core.NewLineKindLF,
	}, printer.PrintHandlers{}, emitContext)

	result := p.EmitSourceFile(transformed)
	return result, nil
}
