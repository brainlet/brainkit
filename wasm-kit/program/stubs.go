package program

// This file contains interface stubs and function variables for packages
// that are not yet ported (compiler, builtins) and wiring for ported
// packages (module).

import (
	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/flow"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

// --- compiler package stubs ---
// Options is the full compiler options. Defined here (not in compiler/) to
// break the circular dependency: compiler imports program, program uses Options.
// Ported from: assemblyscript/src/compiler.ts Options class (lines 233-350).

// Options represents compiler options.
type Options struct {
	// Target is the WebAssembly target. Defaults to Wasm32.
	Target common.Target
	// Runtime type. Defaults to Incremental.
	Runtime common.Runtime
	// DebugInfo indicates that debug information will be emitted.
	DebugInfo bool
	// NoAssert replaces assertions with nops.
	NoAssert bool
	// ExportMemory exports the memory to the embedder.
	ExportMemory bool
	// ImportMemory imports the memory provided by the embedder.
	ImportMemory bool
	// InitialMemory is the initial memory size, in pages.
	InitialMemory uint32
	// MaximumMemory is the maximum memory size, in pages.
	MaximumMemory uint32
	// SharedMemory declares memory as shared.
	SharedMemory bool
	// ZeroFilledMemory indicates imported memory is zero filled.
	ZeroFilledMemory bool
	// ImportTable imports the function table provided by the embedder.
	ImportTable bool
	// ExportTable exports the function table.
	ExportTable bool
	// SourceMap generates information necessary for source maps.
	SourceMap bool
	// UncheckedBehavior controls how unchecked operations are handled.
	UncheckedBehavior int32
	// ExportStart exports the start function instead of calling it implicitly.
	// Empty string means null (don't export).
	ExportStart string
	// ExportStartSet indicates whether ExportStart was explicitly set.
	ExportStartSet bool
	// MemoryBase is the static memory start offset.
	MemoryBase uint32
	// TableBase is the static table start offset.
	TableBase uint32
	// GlobalAliases maps alias names to internal names.
	GlobalAliases map[string]string
	// Features are the activated features.
	Features common.Feature
	// NoUnsafe disallows unsafe features in user code.
	NoUnsafe bool
	// Pedantic enables pedantic diagnostics.
	Pedantic bool
	// LowMemoryLimit indicates a very low (<64k) memory limit.
	LowMemoryLimit uint32
	// ExportRuntime exports the runtime helpers.
	ExportRuntime bool
	// StackSize in bytes. >0 enables stack pointer.
	StackSize int32
	// BundleMajorVersion from root package.json.
	BundleMajorVersion int32
	// BundleMinorVersion from root package.json.
	BundleMinorVersion int32
	// BundlePatchVersion from root package.json.
	BundlePatchVersion int32
	// OptimizeLevelHint is the hinted optimization level. Not applied by the compiler itself.
	OptimizeLevelHint int32
	// ShrinkLevelHint is the hinted shrink level. Not applied by the compiler itself.
	ShrinkLevelHint int32
	// BasenameHint is the hinted basename.
	BasenameHint string
	// BindingsHint indicates hinted bindings generation.
	BindingsHint bool
}

// NewOptions creates new Options with default values.
func NewOptions() *Options {
	return &Options{
		Target:       common.TargetWasm32,
		Runtime:      common.RuntimeIncremental,
		ExportMemory: true,
		Features:     common.FeatureMutableGlobals | common.FeatureSignExtension | common.FeatureNontrappingF2I | common.FeatureBulkMemory,
		BasenameHint: "output",
	}
}

// IsWasm64 tests if the target is WASM64.
func (o *Options) IsWasm64() bool {
	return o.Target == common.TargetWasm64
}

// UsizeType returns the unsigned size type matching the target.
func (o *Options) UsizeType() *types.Type {
	if o.Target == common.TargetWasm64 {
		return types.TypeUsize64
	}
	return types.TypeUsize32
}

// IsizeType returns the signed size type matching the target.
func (o *Options) IsizeType() *types.Type {
	if o.Target == common.TargetWasm64 {
		return types.TypeIsize64
	}
	return types.TypeIsize32
}

// SizeTypeRef returns the Binaryen size type ref matching the target.
func (o *Options) SizeTypeRef() uintptr {
	if o.Target == common.TargetWasm64 {
		return module.TypeRefI64
	}
	return module.TypeRefI32
}

// WillOptimize returns true if any optimizations will be performed.
func (o *Options) WillOptimize() bool {
	return o.OptimizeLevelHint > 0 || o.ShrinkLevelHint > 0
}

// SetFeature sets whether a feature is enabled.
func (o *Options) SetFeature(feature common.Feature, on bool) {
	if on {
		// Enabling Stringref also enables GC
		if feature&common.FeatureStringref != 0 {
			feature |= common.FeatureGC
		}
		// Enabling GC also enables Reference Types
		if feature&common.FeatureGC != 0 {
			feature |= common.FeatureReferenceTypes
		}
		// Enabling Relaxed SIMD also enables SIMD
		if feature&common.FeatureRelaxedSimd != 0 {
			feature |= common.FeatureSimd
		}
		o.Features |= feature
	} else {
		// Disabling Reference Types also disables GC
		if feature&common.FeatureReferenceTypes != 0 {
			feature |= common.FeatureGC
		}
		// Disabling GC also disables Stringref
		if feature&common.FeatureGC != 0 {
			feature |= common.FeatureStringref
		}
		// Disabling SIMD also disables Relaxed SIMD
		if feature&common.FeatureSimd != 0 {
			feature |= common.FeatureRelaxedSimd
		}
		o.Features &^= feature
	}
}

// HasFeature tests if a specific feature is activated.
func (o *Options) HasFeature(feature common.Feature) bool {
	return o.Features&feature != 0
}

// --- module package types (wired to real module package) ---

// Module is the real module.Module type.
type Module = module.Module

// ExpressionRef is a Binaryen expression reference.
type ExpressionRef = module.ExpressionRef

// FunctionRef is a Binaryen function reference.
type FunctionRef = module.FunctionRef

// MemorySegment represents a memory segment in the module.
type MemorySegment = module.MemorySegment

// ModuleCreate creates a new module. Wired to module.Create.
var ModuleCreate func(hasStack bool, sizeTypeRef uintptr) *Module

// GetFunctionName gets the name of a function by its ref.
// Wired to module.GetFunctionName.
var GetFunctionName func(ref FunctionRef) string

// ModuleSetDebugLocation sets a debug location in the module.
// Wired to Module.SetDebugLocation.
var ModuleSetDebugLocation func(m *Module, funcRef FunctionRef, exprRef ExpressionRef, fileIndex int32, line int32, col int32)

// ModuleSetLocalName sets a local variable name for debug info.
// Wired to Module.SetLocalName.
var ModuleSetLocalName func(m *Module, funcRef FunctionRef, index int32, name string)

// --- resolver helpers ---

// ResolveFunction is a function variable that resolves a function prototype.
// It delegates to Resolver.ResolveFunction with default contextual types and ReportModeReport.
// This exists so that program.go RequireFunction can call it without knowing Resolver internals.
var ResolveFunction func(r *Resolver, prototype *FunctionPrototype, typeArguments []*types.Type) *Function

func init() {
	types.DecoratorFlagUnmanaged = uint32(DecoratorFlagsUnmanaged)
	types.LeastUpperBoundFunc = func(a, b types.ClassReference) types.ClassReference {
		left, lok := a.(*Class)
		right, rok := b.(*Class)
		if !lok || !rok {
			return nil
		}
		return LeastUpperBound(left, right)
	}

	// Wire types.ToRef() to map TypeKind → module.TypeRef.
	// Ported from: assemblyscript/src/types.ts Type.toRef().
	types.TypeToRefFunc = func(t *types.Type) uintptr {
		switch t.Kind {
		case types.TypeKindBool, types.TypeKindI8, types.TypeKindI16, types.TypeKindI32,
			types.TypeKindU8, types.TypeKindU16, types.TypeKindU32:
			return module.TypeRefI32
		case types.TypeKindI64, types.TypeKindU64:
			return module.TypeRefI64
		case types.TypeKindIsize, types.TypeKindUsize:
			if t.Size == 64 {
				return module.TypeRefI64
			}
			return module.TypeRefI32
		case types.TypeKindF32:
			return module.TypeRefF32
		case types.TypeKindF64:
			return module.TypeRefF64
		case types.TypeKindV128:
			return module.TypeRefV128
		case types.TypeKindFunc:
			return module.TypeRefFuncref
		case types.TypeKindExtern:
			return module.TypeRefExternref
		case types.TypeKindAny:
			return module.TypeRefAnyref
		case types.TypeKindEq:
			return module.TypeRefEqref
		case types.TypeKindStruct:
			return module.TypeRefStructref
		case types.TypeKindArray:
			return module.TypeRefArrayref
		case types.TypeKindI31:
			return module.TypeRefI31ref
		case types.TypeKindString:
			return module.TypeRefStringref
		// TODO: Stringview types need binaryen runtime calls not yet ported.
		// case types.TypeKindStringviewWTF8: return binaryen.TypeStringviewWTF8()
		// case types.TypeKindStringviewWTF16: return binaryen.TypeStringviewWTF16()
		// case types.TypeKindStringviewIter: return binaryen.TypeStringviewIter()
		case types.TypeKindVoid:
			return module.TypeRefNone
		default:
			return module.TypeRefUnreachable
		}
	}

	// Wire types.CreateTypeFunc to module.CreateType.
	types.CreateTypeFunc = func(refs []uintptr) uintptr {
		return module.CreateType(refs)
	}

	// Wire the real resolver implementation.
	ResolveFunction = func(r *Resolver, prototype *FunctionPrototype, typeArguments []*types.Type) *Function {
		return r.ResolveFunction(prototype, typeArguments, nil, ReportModeReport)
	}

	// Wire module package functions.
	ModuleCreate = func(hasStack bool, sizeTypeRef uintptr) *Module {
		return module.Create(hasStack, module.TypeRef(sizeTypeRef))
	}
	GetFunctionName = module.GetFunctionName
	ModuleSetDebugLocation = func(m *Module, funcRef FunctionRef, exprRef ExpressionRef, fileIndex int32, line int32, col int32) {
		m.SetDebugLocation(funcRef, exprRef, uint32(fileIndex), uint32(line), uint32(col))
	}
	ModuleSetLocalName = func(m *Module, funcRef FunctionRef, index int32, name string) {
		m.SetLocalName(funcRef, uint32(index), name)
	}

	flow.NewLocalFunc = func(name string, index int32, typ *types.Type, parent flow.FlowFunctionRef) flow.FlowLocalRef {
		fn, ok := parent.(*Function)
		if !ok {
			return nil
		}
		return NewLocal(name, index, typ, fn, nil)
	}
	flow.MangleInternalNameFunc = func(name string, parent flow.FlowElementRef, asGlobal bool) string {
		if parent == nil {
			return name
		}
		if element, ok := parent.(Element); ok {
			return MangleInternalName(name, element, false, asGlobal)
		}
		return name
	}
	flow.ElementKindFunction = int32(ElementKindFunction)
	flow.ElementKindClass = int32(ElementKindClass)
	flow.ElementKindPropertyPrototype = int32(ElementKindPropertyPrototype)
	flow.ElementKindGlobal = int32(ElementKindGlobal)
	flow.ElementKindEnumValue = int32(ElementKindEnumValue)
	flow.CommonFlagsConstructor = uint32(common.CommonFlagsConstructor)
	flow.CommonFlagsScoped = uint32(common.CommonFlagsScoped)
	flow.DiagnosticCodeCannotRedeclare = int32(diagnostics.DiagnosticCodeCannotRedeclareBlockScopedVariable0)
	flow.DiagnosticCodeDuplicateIdentifier = int32(diagnostics.DiagnosticCodeDuplicateIdentifier0)
	flow.BuiltinNameStringEq = common.BuiltinNameStringEq
	flow.BuiltinNameStringNe = common.BuiltinNameStringNe
	flow.BuiltinNameStringNot = common.BuiltinNameStringNot
	flow.BuiltinNameTostack = common.BuiltinNameTostack

	// Wire flow package's Binaryen IR inspection functions to real module package.
	flow.GetExpressionId = func(expr module.ExpressionRef) int32 { return int32(module.GetExpressionId(expr)) }
	flow.GetExpressionType = func(expr module.ExpressionRef) int32 { return int32(module.GetExpressionType(expr)) }
	flow.GetLocalGetIndex = func(expr module.ExpressionRef) int32 { return int32(module.GetLocalGetIndex(expr)) }
	flow.IsLocalTee = module.IsLocalTee
	flow.GetLocalSetValue = module.GetLocalSetValue
	flow.GetLocalSetIndex = func(expr module.ExpressionRef) int32 { return int32(module.GetLocalSetIndex(expr)) }
	flow.GetGlobalGetName = module.GetGlobalGetName
	flow.GetBinaryOp = func(expr module.ExpressionRef) int32 { return int32(module.GetBinaryOp(expr)) }
	flow.GetBinaryLeft = module.GetBinaryLeft
	flow.GetBinaryRight = module.GetBinaryRight
	flow.GetUnaryOp = func(expr module.ExpressionRef) int32 { return int32(module.GetUnaryOp(expr)) }
	flow.GetUnaryValue = module.GetUnaryValue
	flow.GetConstValueI32 = module.GetConstValueI32
	flow.GetConstValueI64Low = module.GetConstValueI64Low
	flow.GetConstValueF32 = module.GetConstValueF32
	flow.GetConstValueF64 = module.GetConstValueF64
	flow.GetLoadBytes = func(expr module.ExpressionRef) int32 { return int32(module.GetLoadBytes(expr)) }
	flow.IsLoadSigned = module.IsLoadSigned
	flow.GetBlockName = module.GetBlockName
	flow.GetBlockChildCount = func(expr module.ExpressionRef) int32 { return int32(module.GetBlockChildCount(expr)) }
	flow.GetBlockChildAt = func(expr module.ExpressionRef, index int32) module.ExpressionRef {
		return module.GetBlockChildAt(expr, uint32(index))
	}
	flow.GetIfCondition = module.GetIfCondition
	flow.GetIfTrue = module.GetIfTrue
	flow.GetIfFalse = module.GetIfFalse
	flow.GetSelectThen = module.GetSelectThen
	flow.GetSelectElse = module.GetSelectElse
	flow.GetCallTarget = module.GetCallTarget
	flow.GetCallOperandAt = func(expr module.ExpressionRef, index int32) module.ExpressionRef {
		return module.GetCallOperandAt(expr, uint32(index))
	}
	flow.GetCallOperandCount = func(expr module.ExpressionRef) int32 { return int32(module.GetCallOperandCount(expr)) }
	flow.IsConstZero = module.IsConstZero
	flow.IsConstNonZero = module.IsConstNonZero

	// Wire flow package's Binaryen ExpressionId constants.
	flow.ExpressionIdLocalGet = int32(module.ExpressionIdLocalGet)
	flow.ExpressionIdLocalSet = int32(module.ExpressionIdLocalSet)
	flow.ExpressionIdGlobalGet = int32(module.ExpressionIdGlobalGet)
	flow.ExpressionIdBinary = int32(module.ExpressionIdBinary)
	flow.ExpressionIdUnary = int32(module.ExpressionIdUnary)
	flow.ExpressionIdConst = int32(module.ExpressionIdConst)
	flow.ExpressionIdLoad = int32(module.ExpressionIdLoad)
	flow.ExpressionIdBlock = int32(module.ExpressionIdBlock)
	flow.ExpressionIdIf = int32(module.ExpressionIdIf)
	flow.ExpressionIdSelect = int32(module.ExpressionIdSelect)
	flow.ExpressionIdCall = int32(module.ExpressionIdCall)
	flow.ExpressionIdUnreachable = int32(module.ExpressionIdUnreachable)

	// Wire flow package's BinaryOp constants.
	flow.BinaryOpEqI32 = int32(module.BinaryOpEqI32)
	flow.BinaryOpEqI64 = int32(module.BinaryOpEqI64)
	flow.BinaryOpNeI32 = int32(module.BinaryOpNeI32)
	flow.BinaryOpNeI64 = int32(module.BinaryOpNeI64)

	// Wire flow package's UnaryOp constants.
	flow.UnaryOpEqzI32 = int32(module.UnaryOpEqzI32)
	flow.UnaryOpEqzI64 = int32(module.UnaryOpEqzI64)
}

// --- parser package stubs ---
// Parser is already ported, but to avoid circular imports we define
// the narrow interface needed here.

// ParserRef represents a reference to the parser. Stub to break circular dependency.
type ParserRef struct{}

// NewParserRef creates a new parser reference.
var NewParserRef func(diagnostics []*diagnostics.DiagnosticMessage, sources []*ast.Source) *ParserRef

// --- builtins package stubs ---

// BuiltinFunctions is the set of builtin function handlers. Stub.
var BuiltinFunctions map[string]interface{}

// BuiltinVariablesOnCompile is the set of builtin variables with on-compile callbacks. Stub.
// Maps internal name -> func(compiler interface{}, global *Global).
// Ported from: assemblyscript/src/builtins.ts builtinVariables_onCompile.
var BuiltinVariablesOnCompile map[string]func(compiler interface{}, global *Global)

// BuiltinVariablesOnAccess is the set of builtin variables with on-access callbacks. Stub.
var BuiltinVariablesOnAccess map[string]interface{}
