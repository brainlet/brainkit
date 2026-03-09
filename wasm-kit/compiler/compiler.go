// Ported from: assemblyscript/src/compiler.ts Compiler class (lines 428-10628)
package compiler

import (
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/flow"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/program"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

// Compiler is the AssemblyScript-to-WebAssembly compiler.
// It walks the typed AST (program) and emits Binaryen IR (module).
// Ported from: assemblyscript/src/compiler.ts Compiler class.
type Compiler struct {
	diagnostics.DiagnosticEmitter

	// Program reference.
	Program *program.Program

	// Current control flow.
	CurrentFlow *flow.Flow
	// Current parent element if not a function, i.e. an enum or namespace.
	CurrentParent program.Element
	// Current type in compilation.
	CurrentType *types.Type
	// Start function statements.
	CurrentBody []module.ExpressionRef
	// Counting memory offset.
	MemoryOffset int64
	// Memory segments being compiled.
	MemorySegments []*module.MemorySegment
	// Map of already compiled static string segments.
	StringSegments map[string]*module.MemorySegment
	// Set of static GC object offsets. tostack is unnecessary for them.
	StaticGcObjectOffsets map[int32]map[int32]struct{}
	// Function table being compiled. First elem is blank.
	FunctionTable []*program.Function
	// Arguments length helper global.
	BuiltinArgumentsLength module.GlobalRef
	// Requires runtime features.
	RuntimeFeatures RuntimeFeatures
	// Current inline functions stack.
	InlineStack []*program.Function
	// Lazily compiled functions.
	LazyFunctions map[*program.Function]struct{}
	// Pending instanceof helpers and their names.
	PendingInstanceOf map[program.DeclaredElement]string
	// Stubs to defer calls to overridden methods.
	OverrideStubs map[*program.Function]struct{}
	// Elements currently undergoing compilation.
	PendingElements map[program.Element]struct{}
	// Elements that are module exports, already processed.
	DoneModuleExports map[program.Element]struct{}
	// Whether the module has custom function exports.
	HasCustomFunctionExports bool
	// Whether the module would use the exported runtime to lift/lower.
	DesiresExportRuntime bool

	// ShadowStack is the shadow stack pass. Stub interface until passes are ported.
	ShadowStack ShadowStackPass
}

// ShadowStackPass is the interface for the shadow stack transformation pass.
// Stub until passes/shadowstack is ported.
type ShadowStackPass interface {
	WalkModule()
}

// RtracePass is the interface for the rtrace memory pass.
// Stub until passes/rtrace is ported.
type RtracePass interface {
	WalkModule()
}

// Module returns the module being compiled.
func (c *Compiler) Module() *module.Module {
	return c.Program.Module_
}

// Options returns the compiler options.
func (c *Compiler) Options() *program.Options {
	return c.Program.Options
}

// Resolver returns the program's resolver.
func (c *Compiler) Resolver() *program.Resolver {
	return c.Program.Resolver_
}

// Compile compiles a Program to a Module.
// This is the static entry point matching TS's Compiler.compile(program).
func Compile(prog *program.Program) *module.Module {
	c := NewCompiler(prog)
	return c.CompileProgram()
}

// NewCompiler constructs a new compiler for a Program.
// Ported from: assemblyscript/src/compiler.ts constructor (lines 486-527).
func NewCompiler(prog *program.Program) *Compiler {
	c := &Compiler{
		DiagnosticEmitter:     diagnostics.NewDiagnosticEmitter(prog.DiagnosticEmitter.Diagnostics),
		Program:               prog,
		CurrentType:           types.TypeVoid,
		MemorySegments:        make([]*module.MemorySegment, 0),
		StringSegments:        make(map[string]*module.MemorySegment),
		StaticGcObjectOffsets: make(map[int32]map[int32]struct{}),
		FunctionTable:         make([]*program.Function, 0),
		LazyFunctions:         make(map[*program.Function]struct{}),
		PendingInstanceOf:     make(map[program.DeclaredElement]string),
		OverrideStubs:         make(map[*program.Function]struct{}),
		PendingElements:       make(map[program.Element]struct{}),
		DoneModuleExports:     make(map[program.Element]struct{}),
	}

	mod := prog.Module_
	options := prog.Options

	// Set memory offset based on options.
	if options.MemoryBase != 0 {
		c.MemoryOffset = int64(options.MemoryBase)
		mod.SetLowMemoryUnused(false)
	} else {
		if options.LowMemoryLimit == 0 && options.OptimizeLevelHint >= 2 {
			c.MemoryOffset = 1024
			mod.SetLowMemoryUnused(true)
		} else {
			c.MemoryOffset = 8
			mod.SetLowMemoryUnused(false)
		}
	}

	// Map common.Feature flags to module.Features (Binaryen feature flags).
	var featureFlags module.Features
	if options.HasFeature(common.FeatureSignExtension) {
		featureFlags |= module.FeatureFlagSignExt
	}
	if options.HasFeature(common.FeatureMutableGlobals) {
		featureFlags |= module.FeatureFlagMutableGlobals
	}
	if options.HasFeature(common.FeatureNontrappingF2I) {
		featureFlags |= module.FeatureFlagTruncSat
	}
	if options.HasFeature(common.FeatureBulkMemory) {
		featureFlags |= module.FeatureFlagBulkMemory
	}
	if options.HasFeature(common.FeatureSimd) {
		featureFlags |= module.FeatureFlagSIMD
	}
	if options.HasFeature(common.FeatureThreads) {
		featureFlags |= module.FeatureFlagAtomics
	}
	if options.HasFeature(common.FeatureExceptionHandling) {
		featureFlags |= module.FeatureFlagExceptionHandling
	}
	if options.HasFeature(common.FeatureTailCalls) {
		featureFlags |= module.FeatureFlagTailCall
	}
	if options.HasFeature(common.FeatureReferenceTypes) {
		featureFlags |= module.FeatureFlagReferenceTypes
	}
	if options.HasFeature(common.FeatureMultiValue) {
		featureFlags |= module.FeatureFlagMultiValue
	}
	if options.HasFeature(common.FeatureGC) {
		featureFlags |= module.FeatureFlagGC
	}
	if options.HasFeature(common.FeatureMemory64) {
		featureFlags |= module.FeatureFlagMemory64
	}
	if options.HasFeature(common.FeatureRelaxedSimd) {
		featureFlags |= module.FeatureFlagRelaxedSIMD
	}
	if options.HasFeature(common.FeatureExtendedConst) {
		featureFlags |= module.FeatureFlagExtendedConst
	}
	if options.HasFeature(common.FeatureStringref) {
		featureFlags |= module.FeatureFlagStringref
	}
	mod.SetFeatures(featureFlags)

	// Set up the main start function.
	startSig := types.CreateSignature(prog, nil, types.TypeVoid, nil, 0, false)
	startFunctionInstance := prog.MakeNativeFunction(
		common.BuiltinNameStart,
		startSig,
		nil, // parent: top-level
		common.CommonFlagsAmbient,
		0, // no decorator flags
	)
	startFunctionInstance.SetInternalName(common.BuiltinNameStart)
	c.CurrentFlow = startFunctionInstance.Flow
	c.CurrentBody = make([]module.ExpressionRef, 0)

	// ShadowStack will be set by the caller or a stub until passes are ported.

	return c
}
