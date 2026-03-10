package program

import (
	"fmt"

	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/flow"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

// Program represents an AssemblyScript program.
// It is a 1:1 port of the TypeScript Program class.
type Program struct {
	diagnostics.DiagnosticEmitter

	// Configuration and infrastructure
	Options           *Options
	Module_           *Module
	Parser_           *ParserRef
	Resolver_         *Resolver
	Sources           []*ast.Source
	DiagnosticsOffset int32
	NativeFile        *File
	NextClassId       uint32
	NextSignatureId   uint32
	Initialized       bool

	// File lookup by normalized path.
	FilesByName map[string]*File

	// Lookup maps
	// Note: these fields use the "Map" suffix to avoid name clashes with
	// the flow.FlowProgramRef interface methods ElementsByName() and
	// InstancesByName(), which Program implements via flowProgramRef adapter.
	ElementsByNameMap     map[string]Element
	ElementsByDeclaration map[ast.Node]DeclaredElement
	InstancesByNameMap    map[string]Element
	WrapperClasses        map[*types.Type]*Class
	ManagedClasses        map[int32]*Class
	UniqueSignatures      map[string]*types.Signature
	ModuleImports         map[string]map[string]Element

	// Cached stdlib elements (lazy-initialized on first access)
	cachedArrayBufferViewInstance    *Class
	cachedArrayBufferInstance        *Class
	cachedArrayPrototype             *ClassPrototype
	cachedStaticArrayPrototype       *ClassPrototype
	cachedSetPrototype               *ClassPrototype
	cachedMapPrototype               *ClassPrototype
	cachedFunctionPrototype          *ClassPrototype
	cachedStringInstance             *Class
	cachedRegexpInstance             *Class
	cachedObjectPrototype            *ClassPrototype
	cachedObjectInstance             *Class
	cachedAbortInstance              *Function
	cachedAllocInstance              *Function
	cachedNewInstance                *Function
	cachedVisitInstance              *Function
	cachedLinkInstance               *Function
	cachedInt8ArrayPrototype         *ClassPrototype
	cachedInt16ArrayPrototype        *ClassPrototype
	cachedInt32ArrayPrototype        *ClassPrototype
	cachedInt64ArrayPrototype        *ClassPrototype
	cachedUint8ArrayPrototype        *ClassPrototype
	cachedUint8ClampedArrayPrototype *ClassPrototype
	cachedUint16ArrayPrototype       *ClassPrototype
	cachedUint32ArrayPrototype       *ClassPrototype
	cachedUint64ArrayPrototype       *ClassPrototype
	cachedFloat32ArrayPrototype      *ClassPrototype
	cachedFloat64ArrayPrototype      *ClassPrototype

	// Cached native declarations used by MakeNative* helpers
	nativeDummySignature *ast.FunctionTypeNode

	// flowRef is the cached flow.FlowProgramRef adapter for this program.
	flowRef *flowProgramRef
}

// Compile-time interface satisfaction check.
var _ types.ProgramReference = (*Program)(nil)

// NewProgram creates a new program.
func NewProgram(options *Options, diags []*diagnostics.DiagnosticMessage) *Program {
	p := &Program{
		Options:               options,
		FilesByName:           make(map[string]*File),
		ElementsByNameMap:     make(map[string]Element),
		ElementsByDeclaration: make(map[ast.Node]DeclaredElement),
		InstancesByNameMap:    make(map[string]Element),
		WrapperClasses:        make(map[*types.Type]*Class),
		ManagedClasses:        make(map[int32]*Class),
		UniqueSignatures:      make(map[string]*types.Signature),
		ModuleImports:         make(map[string]map[string]Element),
	}
	if diags != nil {
		p.DiagnosticEmitter = diagnostics.NewDiagnosticEmitter(diags)
	} else {
		p.DiagnosticEmitter = diagnostics.NewDiagnosticEmitter(nil)
	}

	// Create module if factory is set
	if ModuleCreate != nil {
		p.Module_ = ModuleCreate(options.StackSize > 0, options.SizeTypeRef())
	}

	// Create resolver
	p.Resolver_ = NewResolver(p)

	// Create native file
	nativeFile := NewFile(p, ast.NativeSource())
	p.NativeFile = nativeFile
	p.FilesByName[nativeFile.GetInternalName()] = nativeFile
	p.ElementsByNameMap[nativeFile.GetInternalName()] = nativeFile

	return p
}

// ---------------------------------------------------------------------------
// Diagnostic methods (shadow embedded DiagnosticEmitter with variadic args)
// ---------------------------------------------------------------------------

// Error emits an error diagnostic message. Accepts 0-3 format arguments.
func (p *Program) Error(code diagnostics.DiagnosticCode, rng *diagnostics.Range, args ...string) {
	arg0, arg1, arg2 := extractArgs(args)
	p.DiagnosticEmitter.Error(code, rng, arg0, arg1, arg2)
}

// ErrorRelated emits an error diagnostic message with a related range.
func (p *Program) ErrorRelated(code diagnostics.DiagnosticCode, rng *diagnostics.Range, relatedRange *diagnostics.Range, args ...string) {
	arg0, arg1, arg2 := extractArgs(args)
	p.DiagnosticEmitter.ErrorRelated(code, rng, relatedRange, arg0, arg1, arg2)
}

// Warning emits a warning diagnostic message. Accepts 0-3 format arguments.
func (p *Program) Warning(code diagnostics.DiagnosticCode, rng *diagnostics.Range, args ...string) {
	arg0, arg1, arg2 := extractArgs(args)
	p.DiagnosticEmitter.Warning(code, rng, arg0, arg1, arg2)
}

// Info emits an informatory diagnostic message. Accepts 0-3 format arguments.
func (p *Program) Info(code diagnostics.DiagnosticCode, rng *diagnostics.Range, args ...string) {
	arg0, arg1, arg2 := extractArgs(args)
	p.DiagnosticEmitter.Info(code, rng, arg0, arg1, arg2)
}

// Pedantic emits a pedantic diagnostic message. Accepts 0-3 format arguments.
func (p *Program) Pedantic(code diagnostics.DiagnosticCode, rng *diagnostics.Range, args ...string) {
	arg0, arg1, arg2 := extractArgs(args)
	p.DiagnosticEmitter.Pedantic(code, rng, arg0, arg1, arg2)
}

// extractArgs pads variadic string args to exactly 3 values.
func extractArgs(args []string) (string, string, string) {
	var a0, a1, a2 string
	if len(args) > 0 {
		a0 = args[0]
	}
	if len(args) > 1 {
		a1 = args[1]
	}
	if len(args) > 2 {
		a2 = args[2]
	}
	return a0, a1, a2
}

// CheckTypeSupported checks if a type is supported, reporting an error if not.
// Ported from: assemblyscript/src/program.ts Program.checkTypeSupported.
func (p *Program) CheckTypeSupported(typ *types.Type, reportNode ast.Node) bool {
	switch typ.Kind {
	case types.TypeKindV128:
		return p.checkFeatureEnabled(common.FeatureSimd, reportNode)
	case types.TypeKindFunc, types.TypeKindExtern:
		if !typ.Is(types.TypeFlagNullable) {
			return p.checkFeatureEnabled(common.FeatureGC, reportNode)
		}
		return p.checkFeatureEnabled(common.FeatureReferenceTypes, reportNode)
	case types.TypeKindAny, types.TypeKindEq, types.TypeKindStruct, types.TypeKindArray, types.TypeKindI31:
		return p.checkFeatureEnabled(common.FeatureReferenceTypes, reportNode) &&
			p.checkFeatureEnabled(common.FeatureGC, reportNode)
	case types.TypeKindString, types.TypeKindStringviewWTF8, types.TypeKindStringviewWTF16, types.TypeKindStringviewIter:
		return p.checkFeatureEnabled(common.FeatureReferenceTypes, reportNode) &&
			p.checkFeatureEnabled(common.FeatureStringref, reportNode)
	}

	if classReference := typ.GetClass(); classReference != nil {
		if classInstance, ok := classReference.(*Class); ok {
			for current := classInstance; current != nil; current = current.Base {
				for _, typeArgument := range current.TypeArguments {
					if !p.CheckTypeSupported(typeArgument, reportNode) {
						return false
					}
				}
			}
		}
	} else if signatureReference := typ.GetSignature(); signatureReference != nil {
		if thisType := signatureReference.ThisType; thisType != nil {
			if !p.CheckTypeSupported(thisType, reportNode) {
				return false
			}
		}
		for _, parameterType := range signatureReference.ParameterTypes {
			if !p.CheckTypeSupported(parameterType, reportNode) {
				return false
			}
		}
		if !p.CheckTypeSupported(signatureReference.ReturnType, reportNode) {
			return false
		}
	}
	return true
}

// Initialize initializes the program: sets up lookup maps, built-in types, etc.
// Ported from: assemblyscript/src/program.ts Program.initialize().
func (p *Program) Initialize() {
	if p.Initialized {
		return
	}
	p.Initialized = true

	options := p.Options

	p.registerNativeType(common.CommonNameI8, types.TypeI8)
	p.registerNativeType(common.CommonNameI16, types.TypeI16)
	p.registerNativeType(common.CommonNameI32, types.TypeI32)
	p.registerNativeType(common.CommonNameI64, types.TypeI64)
	p.registerNativeType(common.CommonNameIsize, options.IsizeType())
	p.registerNativeType(common.CommonNameU8, types.TypeU8)
	p.registerNativeType(common.CommonNameU16, types.TypeU16)
	p.registerNativeType(common.CommonNameU32, types.TypeU32)
	p.registerNativeType(common.CommonNameU64, types.TypeU64)
	p.registerNativeType(common.CommonNameUsize, options.UsizeType())
	p.registerNativeType(common.CommonNameBool, types.TypeBool)
	p.registerNativeType(common.CommonNameF32, types.TypeF32)
	p.registerNativeType(common.CommonNameF64, types.TypeF64)
	p.registerNativeType(common.CommonNameVoid, types.TypeVoid)
	p.registerNativeType(common.CommonNameNumber, types.TypeF64)
	p.registerNativeType(common.CommonNameBoolean, types.TypeBool)

	p.registerBuiltinGenericType(common.CommonNameNative)
	p.registerBuiltinGenericType(common.CommonNameIndexof)
	p.registerBuiltinGenericType(common.CommonNameValueof)
	p.registerBuiltinGenericType(common.CommonNameReturnof)
	p.registerBuiltinGenericType(common.CommonNameNonnull)

	p.registerNativeType(common.CommonNameV128, types.TypeV128)
	p.registerNativeType(common.CommonNameRefFunc, types.TypeFunc)
	p.registerNativeType(common.CommonNameRefExtern, types.TypeExtern)
	p.registerNativeType(common.CommonNameRefAny, types.TypeAnyRef)
	p.registerNativeType(common.CommonNameRefEq, types.TypeEq)
	p.registerNativeType(common.CommonNameRefStruct, types.TypeStructRef)
	p.registerNativeType(common.CommonNameRefArray, types.TypeArrayRef)
	p.registerNativeType(common.CommonNameRefI31, types.TypeI31)
	p.registerNativeType(common.CommonNameRefString, types.TypeStringRef)
	p.registerNativeType(common.CommonNameRefStringviewWtf8, types.TypeStringviewWTF8)
	p.registerNativeType(common.CommonNameRefStringviewWtf16, types.TypeStringviewWTF16)
	p.registerNativeType(common.CommonNameRefStringviewIter, types.TypeStringviewIter)

	target := int64(common.TargetWasm32)
	if options.IsWasm64() {
		target = int64(common.TargetWasm64)
	}
	p.registerConstantInteger(common.CommonNameASCTarget, types.TypeI32, target)
	p.registerConstantInteger(common.CommonNameASCRuntime, types.TypeI32, int64(options.Runtime))
	p.registerConstantInteger(common.CommonNameASCNoAssert, types.TypeBool, boolToI64(options.NoAssert))
	p.registerConstantInteger(common.CommonNameASCMemoryBase, types.TypeI32, int64(options.MemoryBase))
	p.registerConstantInteger(common.CommonNameASCTableBase, types.TypeI32, int64(options.TableBase))
	p.registerConstantInteger(common.CommonNameASCOptimizeLevel, types.TypeI32, int64(options.OptimizeLevelHint))
	p.registerConstantInteger(common.CommonNameASCShrinkLevel, types.TypeI32, int64(options.ShrinkLevelHint))
	p.registerConstantInteger(common.CommonNameASCLowMemoryLimit, types.TypeI32, int64(options.LowMemoryLimit))
	p.registerConstantInteger(common.CommonNameASCExportRuntime, types.TypeBool, boolToI64(options.ExportRuntime))
	p.registerConstantInteger(common.CommonNameASCVersionMajor, types.TypeI32, int64(options.BundleMajorVersion))
	p.registerConstantInteger(common.CommonNameASCVersionMinor, types.TypeI32, int64(options.BundleMinorVersion))
	p.registerConstantInteger(common.CommonNameASCVersionPatch, types.TypeI32, int64(options.BundlePatchVersion))

	p.registerConstantInteger(common.CommonNameASCFeatureSignExtension, types.TypeBool, boolToI64(options.HasFeature(common.FeatureSignExtension)))
	p.registerConstantInteger(common.CommonNameASCFeatureMutableGlobals, types.TypeBool, boolToI64(options.HasFeature(common.FeatureMutableGlobals)))
	p.registerConstantInteger(common.CommonNameASCFeatureNontrappingF2I, types.TypeBool, boolToI64(options.HasFeature(common.FeatureNontrappingF2I)))
	p.registerConstantInteger(common.CommonNameASCFeatureBulkMemory, types.TypeBool, boolToI64(options.HasFeature(common.FeatureBulkMemory)))
	p.registerConstantInteger(common.CommonNameASCFeatureSimd, types.TypeBool, boolToI64(options.HasFeature(common.FeatureSimd)))
	p.registerConstantInteger(common.CommonNameASCFeatureThreads, types.TypeBool, boolToI64(options.HasFeature(common.FeatureThreads)))
	p.registerConstantInteger(common.CommonNameASCFeatureExceptionHandling, types.TypeBool, boolToI64(options.HasFeature(common.FeatureExceptionHandling)))
	p.registerConstantInteger(common.CommonNameASCFeatureTailCalls, types.TypeBool, boolToI64(options.HasFeature(common.FeatureTailCalls)))
	p.registerConstantInteger(common.CommonNameASCFeatureReferenceTypes, types.TypeBool, boolToI64(options.HasFeature(common.FeatureReferenceTypes)))
	p.registerConstantInteger(common.CommonNameASCFeatureMultiValue, types.TypeBool, boolToI64(options.HasFeature(common.FeatureMultiValue)))
	p.registerConstantInteger(common.CommonNameASCFeatureGC, types.TypeBool, boolToI64(options.HasFeature(common.FeatureGC)))
	p.registerConstantInteger(common.CommonNameASCFeatureMemory64, types.TypeBool, boolToI64(options.HasFeature(common.FeatureMemory64)))
	p.registerConstantInteger(common.CommonNameASCFeatureRelaxedSimd, types.TypeBool, boolToI64(options.HasFeature(common.FeatureRelaxedSimd)))
	p.registerConstantInteger(common.CommonNameASCFeatureExtendedConst, types.TypeBool, boolToI64(options.HasFeature(common.FeatureExtendedConst)))
	p.registerConstantInteger(common.CommonNameASCFeatureStringref, types.TypeBool, boolToI64(options.HasFeature(common.FeatureStringref)))

	// remember deferred elements
	queuedImports := make([]*QueuedImport, 0)
	queuedExports := make(map[*File]map[string]*QueuedExport)
	queuedExportsStar := make(map[*File][]*QueuedExportStar)
	queuedExtends := make([]*ClassPrototype, 0)
	queuedImplements := make([]*ClassPrototype, 0)

	// initialize relevant declaration-like statements of the entire program
	for _, source := range p.Sources {
		file := NewFile(p, source)
		p.FilesByName[file.GetInternalName()] = file
		statements := source.Statements
		for _, statement := range statements {
			switch statement.GetKind() {
			case ast.NodeKindExport:
				p.initializeExports(statement.(*ast.ExportStatement), file, queuedExports, queuedExportsStar)
			case ast.NodeKindExportDefault:
				p.initializeExportDefault(statement.(*ast.ExportDefaultStatement), file, &queuedExtends, &queuedImplements)
			case ast.NodeKindImport:
				p.initializeImports(statement.(*ast.ImportStatement), file, &queuedImports, queuedExports)
			case ast.NodeKindVariable:
				p.initializeVariables(statement.(*ast.VariableStatement), file)
			case ast.NodeKindClassDeclaration:
				p.initializeClass(statement.(*ast.ClassDeclaration), file, &queuedExtends, &queuedImplements)
			case ast.NodeKindEnumDeclaration:
				p.initializeEnum(statement.(*ast.EnumDeclaration), file)
			case ast.NodeKindFunctionDeclaration:
				p.initializeFunction(statement.(*ast.FunctionDeclaration), file)
			case ast.NodeKindInterfaceDeclaration:
				p.initializeInterface(statement.(*ast.ClassDeclaration), file, &queuedExtends)
			case ast.NodeKindNamespaceDeclaration:
				p.initializeNamespace(statement.(*ast.NamespaceDeclaration), file, &queuedExtends, &queuedImplements)
			case ast.NodeKindTypeDeclaration:
				p.initializeTypeDefinition(statement.(*ast.TypeDeclaration), file)
			}
		}
	}

	// queued exports * should be linkable now that all files have been processed
	for file, starExports := range queuedExportsStar {
		for _, exportStar := range starExports {
			foreignFile := p.lookupForeignFile(exportStar.ForeignPath, exportStar.ForeignPathAlt)
			if foreignFile == nil {
				p.Error(
					diagnostics.DiagnosticCodeFile0NotFound,
					exportStar.PathLiteral.GetRange(),
					exportStar.PathLiteral.Value,
				)
				continue
			}
			file.EnsureExportStar(foreignFile)
		}
	}

	// queued imports should be resolvable now through traversing exports and queued exports.
	// note that imports may depend upon imports, so repeat until there's no more progress.
	for {
		i := 0
		madeProgress := false
		for i < len(queuedImports) {
			queuedImport := queuedImports[i]
			localIdentifier := queuedImport.LocalIdentifier
			foreignIdentifier := queuedImport.ForeignIdentifier
			// File must be found here, as it would otherwise already have been reported by the parser
			foreignFile := p.lookupForeignFile(queuedImport.ForeignPath, queuedImport.ForeignPathAlt)
			if foreignFile == nil {
				i++
				continue
			}
			if foreignIdentifier != nil { // i.e. import { foo [as bar] } from "./baz"
				element := p.lookupForeign(foreignIdentifier.Text, foreignFile, queuedExports)
				if element != nil {
					queuedImport.LocalFile.Add(localIdentifier.Text, element, localIdentifier)
					queuedImports = append(queuedImports[:i], queuedImports[i+1:]...)
					madeProgress = true
				} else {
					i++
				}
			} else { // i.e. import * as bar from "./bar"
				localFile := queuedImport.LocalFile
				localName := localIdentifier.Text
				localFile.Add(
					localName,
					foreignFile.AsAliasNamespace(localName, localFile, localIdentifier),
					localIdentifier,
				)
				queuedImports = append(queuedImports[:i], queuedImports[i+1:]...)
				madeProgress = true
			}
		}
		if !madeProgress {
			// report queued imports we were unable to resolve
			for _, queuedImport := range queuedImports {
				foreignIdentifier := queuedImport.ForeignIdentifier
				if foreignIdentifier != nil {
					p.Error(
						diagnostics.DiagnosticCodeModule0HasNoExportedMember1,
						foreignIdentifier.GetRange(),
						queuedImport.ForeignPath, foreignIdentifier.Text,
					)
				}
			}
			break
		}
	}

	// queued exports should be resolvable now that imports are finalized
	for file, exports := range queuedExports {
		for exportName, queuedExport := range exports {
			localName := queuedExport.LocalIdentifier.Text
			foreignPath := queuedExport.ForeignPath
			if foreignPath != "" { // i.e. export { foo [as bar] } from "./baz"
				foreignFile := p.lookupForeignFile(foreignPath, queuedExport.ForeignPathAlt)
				if foreignFile == nil {
					continue
				}
				element := p.lookupForeign(localName, foreignFile, queuedExports)
				if element != nil {
					file.EnsureExport(exportName, element)
				} else {
					p.Error(
						diagnostics.DiagnosticCodeModule0HasNoExportedMember1,
						queuedExport.LocalIdentifier.GetRange(),
						foreignPath, localName,
					)
				}
			} else { // i.e. export { foo [as bar] }
				element := file.GetMember(localName)
				if element != nil {
					file.EnsureExport(exportName, element)
				} else {
					globalElement := p.Lookup(localName)
					if globalElement != nil && IsDeclaredElement(globalElement.GetElementKind()) {
						file.EnsureExport(exportName, globalElement.(DeclaredElement))
					} else {
						p.Error(
							diagnostics.DiagnosticCodeModule0HasNoExportedMember1,
							queuedExport.ForeignIdentifier.GetRange(),
							file.GetInternalName(), queuedExport.ForeignIdentifier.Text,
						)
					}
				}
			}
		}
	}

	// register classes backing basic types
	p.registerWrapperClass(types.TypeI8, common.CommonNameCapI8)
	p.registerWrapperClass(types.TypeI16, common.CommonNameCapI16)
	p.registerWrapperClass(types.TypeI32, common.CommonNameCapI32)
	p.registerWrapperClass(types.TypeI64, common.CommonNameCapI64)
	p.registerWrapperClass(options.IsizeType(), common.CommonNameCapIsize)
	p.registerWrapperClass(types.TypeU8, common.CommonNameCapU8)
	p.registerWrapperClass(types.TypeU16, common.CommonNameCapU16)
	p.registerWrapperClass(types.TypeU32, common.CommonNameCapU32)
	p.registerWrapperClass(types.TypeU64, common.CommonNameCapU64)
	p.registerWrapperClass(options.UsizeType(), common.CommonNameCapUsize)
	p.registerWrapperClass(types.TypeBool, common.CommonNameCapBool)
	p.registerWrapperClass(types.TypeF32, common.CommonNameCapF32)
	p.registerWrapperClass(types.TypeF64, common.CommonNameCapF64)
	if options.HasFeature(common.FeatureSimd) {
		p.registerWrapperClass(types.TypeV128, common.CommonNameCapV128)
	}
	if options.HasFeature(common.FeatureReferenceTypes) {
		p.registerWrapperClass(types.TypeFunc, common.CommonNameCapRefFunc)
		p.registerWrapperClass(types.TypeExtern, common.CommonNameCapRefExtern)
		if options.HasFeature(common.FeatureGC) {
			p.registerWrapperClass(types.TypeAnyRef, common.CommonNameCapRefAny)
			p.registerWrapperClass(types.TypeEq, common.CommonNameCapRefEq)
			p.registerWrapperClass(types.TypeStructRef, common.CommonNameCapRefStruct)
			p.registerWrapperClass(types.TypeArrayRef, common.CommonNameCapRefArray)
			p.registerWrapperClass(types.TypeI31, common.CommonNameCapRefI31)
		}
		if options.HasFeature(common.FeatureStringref) {
			p.registerWrapperClass(types.TypeStringRef, common.CommonNameCapRefString)
		}
	}

	// resolve prototypes of extended classes or interfaces
	resolver := p.Resolver_
	for _, thisPrototype := range queuedExtends {
		extendsNode := thisPrototype.ExtendsNode()
		if extendsNode == nil {
			continue
		}
		baseElement := resolver.ResolveTypeName(extendsNode.Name, nil, thisPrototype.GetParent(), ReportModeReport)
		if baseElement == nil {
			continue
		}
		if thisPrototype.GetElementKind() == ElementKindClassPrototype {
			if baseElement.GetElementKind() == ElementKindClassPrototype {
				basePrototype := baseElement.(*ClassPrototype)
				if basePrototype.HasDecorator(DecoratorFlagsFinal) {
					p.Error(
						diagnostics.DiagnosticCodeClass0IsFinalAndCannotBeExtended,
						extendsNode.GetRange(),
						basePrototype.IdentifierNode().Text,
					)
				}
				if basePrototype.HasDecorator(DecoratorFlagsUnmanaged) != thisPrototype.HasDecorator(DecoratorFlagsUnmanaged) {
					p.Error(
						diagnostics.DiagnosticCodeUnmanagedClassesCannotExtendManagedClassesAndViceVersa,
						diagnostics.JoinRanges(thisPrototype.IdentifierNode().GetRange(), extendsNode.GetRange()),
					)
				}
				if !thisPrototype.Extends(basePrototype) {
					thisPrototype.BasePrototype = basePrototype
				} else {
					p.Error(
						diagnostics.DiagnosticCode0IsReferencedDirectlyOrIndirectlyInItsOwnBaseExpression,
						basePrototype.IdentifierNode().GetRange(),
						basePrototype.IdentifierNode().Text,
					)
				}
			} else {
				p.Error(
					diagnostics.DiagnosticCodeAClassMayOnlyExtendAnotherClass,
					extendsNode.GetRange(),
				)
			}
		} else if thisPrototype.GetElementKind() == ElementKindInterfacePrototype {
			if baseElement.GetElementKind() == ElementKindInterfacePrototype {
				basePrototype := baseElement.(*InterfacePrototype)
				if !thisPrototype.Extends(&basePrototype.ClassPrototype) {
					thisPrototype.BasePrototype = &basePrototype.ClassPrototype
				} else {
					p.Error(
						diagnostics.DiagnosticCode0IsReferencedDirectlyOrIndirectlyInItsOwnBaseExpression,
						basePrototype.IdentifierNode().GetRange(),
						basePrototype.IdentifierNode().Text,
					)
				}
			} else {
				p.Error(
					diagnostics.DiagnosticCodeAnInterfaceCanOnlyExtendAnInterface,
					extendsNode.GetRange(),
				)
			}
		}
	}

	// check override
	for _, prototype := range queuedExtends {
		instanceMembers := prototype.InstanceMembers
		if instanceMembers != nil {
			for _, member := range instanceMembers {
				declaration := member.GetDeclaration()
				if declaration != nil && (common.CommonFlags(getNodeFlags(declaration)) & common.CommonFlagsOverride) != 0 {
					basePrototype := prototype.BasePrototype
					hasOverride := false
					for basePrototype != nil {
						if basePrototype.InstanceMembers != nil {
							if _, ok := basePrototype.InstanceMembers[member.GetName()]; ok {
								hasOverride = true
								break
							}
						}
						basePrototype = basePrototype.BasePrototype
					}
					if !hasOverride {
						bp := prototype.BasePrototype
						if bp != nil {
							p.Error(
								diagnostics.DiagnosticCodeThisMemberCannotHaveAnOverrideModifierBecauseItIsNotDeclaredInTheBaseClass0,
								member.IdentifierNode().GetRange(),
								bp.GetName(),
							)
						}
					}
				}
			}
		}
	}

	// resolve prototypes of implemented interfaces
	for _, thisPrototype := range queuedImplements {
		implementsNodes := thisPrototype.ImplementsNodes()
		if implementsNodes == nil {
			continue
		}
		for _, implementsNode := range implementsNodes {
			interfaceElement := resolver.ResolveTypeName(implementsNode.Name, nil, thisPrototype.GetParent(), ReportModeReport)
			if interfaceElement == nil {
				continue
			}
			if interfaceElement.GetElementKind() == ElementKindInterfacePrototype {
				interfacePrototype := interfaceElement.(*InterfacePrototype)
				if thisPrototype.InterfacePrototypes == nil {
					thisPrototype.InterfacePrototypes = make([]*InterfacePrototype, 0)
				}
				thisPrototype.InterfacePrototypes = append(thisPrototype.InterfacePrototypes, interfacePrototype)
			} else {
				p.Error(
					diagnostics.DiagnosticCodeAClassCanOnlyImplementAnInterface,
					implementsNode.GetRange(),
				)
			}
		}
	}

	// process overrides in extended classes and implemented interfaces
	for _, thisPrototype := range queuedExtends {
		basePrototype := thisPrototype.BasePrototype
		if basePrototype != nil {
			p.processOverrides(thisPrototype, basePrototype)
		}
	}
	for _, thisPrototype := range queuedImplements {
		basePrototype := thisPrototype.BasePrototype
		interfacePrototypes := thisPrototype.InterfacePrototypes
		if basePrototype != nil {
			p.processOverrides(thisPrototype, basePrototype)
		}
		if interfacePrototypes != nil {
			for _, ifaceProto := range interfacePrototypes {
				p.processOverrides(thisPrototype, &ifaceProto.ClassPrototype)
			}
		}
	}

	// set up global aliases
	globalAliases := options.GlobalAliases
	if globalAliases == nil {
		globalAliases = make(map[string]string)
	}
	if _, ok := globalAliases[common.CommonNameAbort]; !ok {
		globalAliases[common.CommonNameAbort] = common.BuiltinNameAbort
	}
	if _, ok := globalAliases[common.CommonNameTrace]; !ok {
		globalAliases[common.CommonNameTrace] = common.BuiltinNameTrace
	}
	if _, ok := globalAliases[common.CommonNameSeed]; !ok {
		globalAliases[common.CommonNameSeed] = common.BuiltinNameSeed
	}
	if _, ok := globalAliases[common.CommonNameMath]; !ok {
		globalAliases[common.CommonNameMath] = common.CommonNameNativeMath
	}
	if _, ok := globalAliases[common.CommonNameMathf]; !ok {
		globalAliases[common.CommonNameMathf] = common.CommonNameNativeMathf
	}
	for alias, name := range globalAliases {
		if len(name) == 0 {
			delete(p.ElementsByNameMap, alias)
			continue
		}
		firstChar := name[0]
		if firstChar >= '0' && firstChar <= '9' {
			// Parse as integer
			val := int64(0)
			for _, ch := range name {
				if ch >= '0' && ch <= '9' {
					val = val*10 + int64(ch-'0')
				} else {
					break
				}
			}
			p.registerConstantInteger(alias, types.TypeI32, val)
		} else {
			if existing, ok := p.ElementsByNameMap[name]; ok {
				p.ElementsByNameMap[alias] = existing
			} else {
				p.Error(diagnostics.DiagnosticCodeElement0NotFound, nil, name)
			}
		}
	}

	// mark module exports
	for _, file := range p.FilesByName {
		if file.Source.SourceKind == ast.SourceKindUserEntry {
			p.markModuleExports(file)
		}
	}
}

func boolToI64(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

func (p *Program) checkFeatureEnabled(feature common.Feature, reportNode ast.Node) bool {
	if p.Options.HasFeature(feature) {
		return true
	}
	p.Error(
		diagnostics.DiagnosticCodeFeature0IsNotEnabled,
		reportNode.GetRange(),
		common.FeatureToString(feature),
	)
	return false
}

// ---------------------------------------------------------------------------
// flow.FlowProgramRef adapter
// ---------------------------------------------------------------------------

// flowProgramRef adapts Program to the flow.FlowProgramRef interface.
// The flow package uses int32/interface{} types to break circular dependencies
// between flow and diagnostics/program packages.
type flowProgramRef struct {
	program *Program
}

// Compile-time check.
var _ flow.FlowProgramRef = (*flowProgramRef)(nil)

// FlowProgramRef returns a flow.FlowProgramRef adapter for this program.
// The adapter is cached so repeated calls return the same object.
func (p *Program) FlowProgramRef() flow.FlowProgramRef {
	if p.flowRef == nil {
		p.flowRef = &flowProgramRef{program: p}
	}
	return p.flowRef
}

func (f *flowProgramRef) UncheckedBehaviorAlways() bool {
	return f.program.Options.UncheckedBehavior == 2 // UncheckedBehaviorAlways
}

func (f *flowProgramRef) Error(code int32, rng interface{}, args ...string) {
	var r *diagnostics.Range
	if rng != nil {
		r, _ = rng.(*diagnostics.Range)
	}
	f.program.Error(diagnostics.DiagnosticCode(code), r, args...)
}

func (f *flowProgramRef) ErrorRelated(code int32, rng1 interface{}, rng2 interface{}, args ...string) {
	var r1, r2 *diagnostics.Range
	if rng1 != nil {
		r1, _ = rng1.(*diagnostics.Range)
	}
	if rng2 != nil {
		r2, _ = rng2.(*diagnostics.Range)
	}
	f.program.ErrorRelated(diagnostics.DiagnosticCode(code), r1, r2, args...)
}

func (f *flowProgramRef) ElementsByName() map[string]flow.FlowElementRef {
	result := make(map[string]flow.FlowElementRef, len(f.program.ElementsByNameMap))
	for k, v := range f.program.ElementsByNameMap {
		result[k] = v
	}
	return result
}

func (f *flowProgramRef) InstancesByName() map[string]flow.FlowElementRef {
	result := make(map[string]flow.FlowElementRef, len(f.program.InstancesByNameMap))
	for k, v := range f.program.InstancesByNameMap {
		result[k] = v
	}
	return result
}

// ---------------------------------------------------------------------------
// types.ProgramReference implementation
// ---------------------------------------------------------------------------

// GetUsizeType returns the target's usize type.
func (p *Program) GetUsizeType() *types.Type {
	return p.Options.UsizeType()
}

// GetFunctionPrototype returns the Function class prototype, or nil.
func (p *Program) GetFunctionPrototype() interface{} {
	return p.FunctionPrototype()
}

// GetWrapperClasses returns the wrapper classes map.
func (p *Program) GetWrapperClasses() map[*types.Type]types.ClassReference {
	result := make(map[*types.Type]types.ClassReference, len(p.WrapperClasses))
	for k, v := range p.WrapperClasses {
		result[k] = v
	}
	return result
}

// GetUniqueSignatures returns the unique signatures map.
func (p *Program) GetUniqueSignatures() map[string]*types.Signature {
	return p.UniqueSignatures
}

// GetNextSignatureId returns the next available signature id.
func (p *Program) GetNextSignatureId() uint32 {
	return p.NextSignatureId
}

// SetNextSignatureId sets the next available signature id.
func (p *Program) SetNextSignatureId(id uint32) {
	p.NextSignatureId = id
}

// ResolveClass resolves a class prototype with the given type arguments.
func (p *Program) ResolveClass(prototype interface{}, typeArguments []*types.Type) types.ClassReference {
	if p.Resolver_ == nil || prototype == nil {
		return nil
	}
	switch typedPrototype := prototype.(type) {
	case *ClassPrototype:
		return p.Resolver_.ResolveClass(typedPrototype, typeArguments, make(map[string]*types.Type), ReportModeSwallow)
	case *InterfacePrototype:
		return p.Resolver_.ResolveClass(&typedPrototype.ClassPrototype, typeArguments, make(map[string]*types.Type), ReportModeSwallow)
	default:
		return nil
	}
}

// ---------------------------------------------------------------------------
// Element lookup
// ---------------------------------------------------------------------------

// Lookup looks up a program-level element by its internal name.
func (p *Program) Lookup(name string) Element {
	if elem, ok := p.ElementsByNameMap[name]; ok {
		return elem
	}
	return nil
}

// GetElementByDeclaration looks up an element by its declaration node.
func (p *Program) GetElementByDeclaration(declaration ast.Node) DeclaredElement {
	if elem, ok := p.ElementsByDeclaration[declaration]; ok {
		return elem
	}
	return nil
}

// EnsureGlobal ensures that an element is registered as a global.
// If a global with the same name already exists, merging is attempted.
func (p *Program) EnsureGlobal(name string, element DeclaredElement) DeclaredElement {
	if existing, ok := p.ElementsByNameMap[name]; ok {
		if existing == element {
			return element
		}
		merged := TryMerge(existing.(DeclaredElement), element)
		if merged != nil {
			p.ElementsByNameMap[name] = merged
			return merged
		}
		// If merge fails, the newer element wins.
		p.ElementsByNameMap[name] = element
		return element
	}
	p.ElementsByNameMap[name] = element
	return element
}

// SearchFunctionByRef searches for a function by its Binaryen function reference.
func (p *Program) SearchFunctionByRef(ref FunctionRef) *Function {
	if GetFunctionName == nil {
		return nil
	}
	name := GetFunctionName(ref)
	if elem, ok := p.InstancesByNameMap[name]; ok {
		if fn, ok := elem.(*Function); ok {
			return fn
		}
	}
	return nil
}

// MarkModuleImport marks an element as a module import.
func (p *Program) MarkModuleImport(moduleName, name string, element Element) {
	moduleMap, ok := p.ModuleImports[moduleName]
	if !ok {
		moduleMap = make(map[string]Element)
		p.ModuleImports[moduleName] = moduleMap
	}
	moduleMap[name] = element
}

// registerNativeType registers a native type definition with the program.
func (p *Program) registerNativeType(name string, typ *types.Type) {
	element := NewTypeDefinition(
		name,
		p.NativeFile,
		p.MakeNativeTypeDeclaration(name, common.CommonFlagsExport),
		DecoratorFlagsBuiltin,
	)
	element.SetType(typ)
	p.NativeFile.Add(name, element, nil)
}

// registerBuiltinGenericType registers a builtin generic type helper.
func (p *Program) registerBuiltinGenericType(name string) {
	element := NewTypeDefinition(
		name,
		p.NativeFile,
		p.MakeNativeTypeDeclaration(name, common.CommonFlagsExport|common.CommonFlagsGeneric),
		DecoratorFlagsBuiltin,
	)
	p.NativeFile.Add(name, element, nil)
}

// registerConstantInteger registers a constant integer value in the global scope.
func (p *Program) registerConstantInteger(name string, typ *types.Type, value int64) {
	global := NewGlobal(
		name,
		p.NativeFile,
		DecoratorFlagsLazy,
		p.MakeNativeVariableDeclaration(name, common.CommonFlagsConst|common.CommonFlagsExport),
	)
	global.SetConstantIntegerValue(value, typ)
	p.NativeFile.Add(name, global, nil)
}

// registerWrapperClass registers the wrapper class for a non-class type if present.
func (p *Program) registerWrapperClass(typ *types.Type, className string) {
	if typ == nil || typ.IsInternalReference() {
		return
	}
	if _, exists := p.WrapperClasses[typ]; exists {
		return
	}
	element := p.Lookup(className)
	if element == nil || element.GetElementKind() != ElementKindClassPrototype {
		return
	}
	classElement := p.Resolver_.ResolveClass(element.(*ClassPrototype), nil, make(map[string]*types.Type), ReportModeSwallow)
	if classElement == nil {
		return
	}
	classElement.WrappedType = typ
	p.WrapperClasses[typ] = classElement
}

// ---------------------------------------------------------------------------
// Native declaration factory methods
// ---------------------------------------------------------------------------

// nativeRange returns a zero-width range in the native source.
func nativeRange() diagnostics.Range {
	src := ast.NativeSource()
	return diagnostics.Range{
		Start:  0,
		End:    0,
		Source: src,
	}
}

// MakeNativeVariableDeclaration creates a native variable declaration.
func (p *Program) MakeNativeVariableDeclaration(name string, flags common.CommonFlags) *ast.VariableDeclaration {
	rng := nativeRange()
	ident := ast.NewIdentifierExpression(name, rng, false)
	return ast.NewVariableDeclaration(ident, nil, int32(flags), nil, nil, rng)
}

// MakeNativeFunctionDeclaration creates a native function declaration.
func (p *Program) MakeNativeFunctionDeclaration(name string, flags common.CommonFlags) *ast.FunctionDeclaration {
	rng := nativeRange()
	ident := ast.NewIdentifierExpression(name, rng, false)
	if p.nativeDummySignature == nil {
		voidType := ast.NewNamedTypeNode(
			ast.NewSimpleTypeName(common.CommonNameVoid, rng),
			nil,
			false,
			rng,
		)
		p.nativeDummySignature = ast.NewFunctionTypeNode(nil, voidType, nil, false, rng)
	}
	return ast.NewFunctionDeclaration(
		ident,
		nil,
		int32(flags|common.CommonFlagsAmbient),
		nil,
		p.nativeDummySignature,
		nil,
		ast.ArrowKindNone,
		rng,
	)
}

// MakeNativeNamespaceDeclaration creates a native namespace declaration.
func (p *Program) MakeNativeNamespaceDeclaration(name string, flags common.CommonFlags) *ast.NamespaceDeclaration {
	rng := nativeRange()
	ident := ast.NewIdentifierExpression(name, rng, false)
	return ast.NewNamespaceDeclaration(ident, nil, int32(flags|common.CommonFlagsAmbient), nil, rng)
}

// MakeNativeTypeDeclaration creates a native type declaration.
func (p *Program) MakeNativeTypeDeclaration(name string, flags common.CommonFlags) *ast.TypeDeclaration {
	rng := nativeRange()
	ident := ast.NewIdentifierExpression(name, rng, false)
	omittedType := ast.NewOmittedType(rng)
	return ast.NewTypeDeclaration(ident, nil, int32(flags), nil, omittedType, rng)
}

// MakeNativeFunction creates a native (ambient) function element.
func (p *Program) MakeNativeFunction(
	name string,
	signature *types.Signature,
	parent Element,
	flags common.CommonFlags,
	decoratorFlags DecoratorFlags,
) *Function {
	if parent == nil {
		parent = p.NativeFile
	}
	declaration := p.MakeNativeFunctionDeclaration(name, flags)
	prototype := NewFunctionPrototype(name, parent, declaration, decoratorFlags)
	return NewFunction(name, prototype, nil, signature, nil)
}

// ---------------------------------------------------------------------------
// Memory layout helpers (BLOCK / OBJECT overhead from ~lib/rt/common)
// ---------------------------------------------------------------------------

// BlockOverhead returns the size of a runtime BLOCK header.
// In AssemblyScript this is typically 16 bytes (mmInfo + gcInfo + rtId + rtSize).
func (p *Program) BlockOverhead() int32 {
	if blockInstance := p.RequireClass(common.CommonNameBlock); blockInstance != nil {
		return int32(blockInstance.NextMemoryOffset)
	}
	return 16
}

// ObjectOverhead returns the size of a runtime OBJECT header beyond the block.
// In AssemblyScript this is typically 4 bytes (gcInfo2) on wasm32, 8 on wasm64.
func (p *Program) ObjectOverhead() int32 {
	if objectInstance := p.RequireClass(common.CommonNameObject_); objectInstance != nil {
		return (int32(objectInstance.NextMemoryOffset) - p.BlockOverhead() + AlMask) & ^int32(AlMask)
	}
	if p.Options.IsWasm64() {
		return 8
	}
	return 4
}

// TotalOverhead returns BlockOverhead + ObjectOverhead.
func (p *Program) TotalOverhead() int32 {
	return p.BlockOverhead() + p.ObjectOverhead()
}

// ComputeBlockStart computes the aligned block start for a given current offset.
func (p *Program) ComputeBlockStart(currentOffset int32) int32 {
	blockOverhead := p.BlockOverhead()
	return ((currentOffset + blockOverhead + AlMask) & ^int32(AlMask)) - blockOverhead
}

// ---------------------------------------------------------------------------
// Cached stdlib element accessors (lazy)
// ---------------------------------------------------------------------------

// require looks up a program-level element by name and asserts its kind.
func (p *Program) require(name string, kind ElementKind) Element {
	elem := p.Lookup(name)
	if elem == nil {
		return nil
	}
	if elem.GetElementKind() != kind {
		return nil
	}
	return elem
}

// requirePrototype looks up a class prototype by name.
func (p *Program) requirePrototype(name string) *ClassPrototype {
	elem := p.require(name, ElementKindClassPrototype)
	if elem == nil {
		return nil
	}
	return elem.(*ClassPrototype)
}

// RequireClass resolves a non-generic class prototype to its concrete instance.
func (p *Program) RequireClass(name string) *Class {
	proto := p.requirePrototype(name)
	if proto == nil {
		return nil
	}
	if p.Resolver_ != nil {
		if resolved := p.Resolver_.ResolveClass(proto, nil, make(map[string]*types.Type), ReportModeSwallow); resolved != nil {
			return resolved
		}
	}
	return nil
}

// RequireFunction resolves a function prototype with optional type arguments.
func (p *Program) RequireFunction(name string, typeArguments []*types.Type) *Function {
	elem := p.require(name, ElementKindFunctionPrototype)
	if elem == nil {
		return nil
	}
	proto := elem.(*FunctionPrototype)
	if ResolveFunction != nil {
		return ResolveFunction(p.Resolver_, proto, typeArguments)
	}
	// Fallback: look for a default instance
	if proto.Instances != nil {
		for _, inst := range proto.Instances {
			return inst
		}
	}
	return nil
}

// RequireGlobal looks up a global variable by name.
func (p *Program) RequireGlobal(name string) *Global {
	elem := p.require(name, ElementKindGlobal)
	if elem == nil {
		return nil
	}
	return elem.(*Global)
}

// ArrayBufferViewInstance returns the cached ArrayBufferView class instance.
func (p *Program) ArrayBufferViewInstance() *Class {
	if p.cachedArrayBufferViewInstance == nil {
		p.cachedArrayBufferViewInstance = p.RequireClass(common.CommonNameArrayBufferView)
	}
	return p.cachedArrayBufferViewInstance
}

// ArrayBufferInstance returns the cached ArrayBuffer class instance.
func (p *Program) ArrayBufferInstance() *Class {
	if p.cachedArrayBufferInstance == nil {
		p.cachedArrayBufferInstance = p.RequireClass(common.CommonNameArrayBuffer)
	}
	return p.cachedArrayBufferInstance
}

// ArrayPrototype returns the cached Array class prototype.
func (p *Program) ArrayPrototype() *ClassPrototype {
	if p.cachedArrayPrototype == nil {
		p.cachedArrayPrototype = p.requirePrototype(common.CommonNameArray)
	}
	return p.cachedArrayPrototype
}

// StaticArrayPrototype returns the cached StaticArray class prototype.
func (p *Program) StaticArrayPrototype() *ClassPrototype {
	if p.cachedStaticArrayPrototype == nil {
		p.cachedStaticArrayPrototype = p.requirePrototype(common.CommonNameStaticArray)
	}
	return p.cachedStaticArrayPrototype
}

// SetPrototype returns the cached Set class prototype.
func (p *Program) SetPrototype() *ClassPrototype {
	if p.cachedSetPrototype == nil {
		p.cachedSetPrototype = p.requirePrototype(common.CommonNameSet)
	}
	return p.cachedSetPrototype
}

// MapPrototype returns the cached Map class prototype.
func (p *Program) MapPrototype() *ClassPrototype {
	if p.cachedMapPrototype == nil {
		p.cachedMapPrototype = p.requirePrototype(common.CommonNameMap)
	}
	return p.cachedMapPrototype
}

// FunctionPrototype returns the cached Function class prototype.
func (p *Program) FunctionPrototype() *ClassPrototype {
	if p.cachedFunctionPrototype == nil {
		p.cachedFunctionPrototype = p.requirePrototype(common.CommonNameFunction)
	}
	return p.cachedFunctionPrototype
}

// StringInstance returns the cached String class instance.
func (p *Program) StringInstance() *Class {
	if p.cachedStringInstance == nil {
		p.cachedStringInstance = p.RequireClass(common.CommonNameCapString)
	}
	return p.cachedStringInstance
}

// RegexpInstance returns the cached RegExp class instance.
func (p *Program) RegexpInstance() *Class {
	if p.cachedRegexpInstance == nil {
		p.cachedRegexpInstance = p.RequireClass(common.CommonNameRegExp)
	}
	return p.cachedRegexpInstance
}

// ObjectPrototype returns the cached Object class prototype.
func (p *Program) ObjectPrototype() *ClassPrototype {
	if p.cachedObjectPrototype == nil {
		p.cachedObjectPrototype = p.requirePrototype(common.CommonNameObject)
	}
	return p.cachedObjectPrototype
}

// ObjectInstance returns the cached Object class instance.
func (p *Program) ObjectInstance() *Class {
	if p.cachedObjectInstance == nil {
		p.cachedObjectInstance = p.RequireClass(common.CommonNameObject)
	}
	return p.cachedObjectInstance
}

// AbortInstance returns the cached abort function instance.
func (p *Program) AbortInstance() *Function {
	if p.cachedAbortInstance == nil {
		p.cachedAbortInstance = p.RequireFunction(common.CommonNameAbort, nil)
	}
	return p.cachedAbortInstance
}

// AllocInstance returns the cached __alloc runtime function instance.
func (p *Program) AllocInstance() *Function {
	if p.cachedAllocInstance == nil {
		p.cachedAllocInstance = p.RequireFunction(common.CommonNameAlloc, nil)
	}
	return p.cachedAllocInstance
}

// NewInstance returns the cached __new runtime function instance.
func (p *Program) NewInstance() *Function {
	if p.cachedNewInstance == nil {
		p.cachedNewInstance = p.RequireFunction(common.CommonNameNew, nil)
	}
	return p.cachedNewInstance
}

// VisitInstance returns the cached __visit runtime function instance.
func (p *Program) VisitInstance() *Function {
	if p.cachedVisitInstance == nil {
		if instance, ok := p.InstancesByNameMap[common.CommonNameVisit].(*Function); ok {
			p.cachedVisitInstance = instance
		} else {
			p.cachedVisitInstance = p.RequireFunction(common.CommonNameVisit, nil)
		}
	}
	return p.cachedVisitInstance
}

// LinkInstance returns the cached __link runtime function instance.
func (p *Program) LinkInstance() *Function {
	if p.cachedLinkInstance == nil {
		p.cachedLinkInstance = p.RequireFunction(common.CommonNameLink, nil)
	}
	return p.cachedLinkInstance
}

// Typed array prototype accessors

// Int8ArrayPrototype returns the cached Int8Array class prototype.
func (p *Program) Int8ArrayPrototype() *ClassPrototype {
	if p.cachedInt8ArrayPrototype == nil {
		p.cachedInt8ArrayPrototype = p.requirePrototype(common.CommonNameInt8Array)
	}
	return p.cachedInt8ArrayPrototype
}

// Int16ArrayPrototype returns the cached Int16Array class prototype.
func (p *Program) Int16ArrayPrototype() *ClassPrototype {
	if p.cachedInt16ArrayPrototype == nil {
		p.cachedInt16ArrayPrototype = p.requirePrototype(common.CommonNameInt16Array)
	}
	return p.cachedInt16ArrayPrototype
}

// Int32ArrayPrototype returns the cached Int32Array class prototype.
func (p *Program) Int32ArrayPrototype() *ClassPrototype {
	if p.cachedInt32ArrayPrototype == nil {
		p.cachedInt32ArrayPrototype = p.requirePrototype(common.CommonNameInt32Array)
	}
	return p.cachedInt32ArrayPrototype
}

// Int64ArrayPrototype returns the cached Int64Array class prototype.
func (p *Program) Int64ArrayPrototype() *ClassPrototype {
	if p.cachedInt64ArrayPrototype == nil {
		p.cachedInt64ArrayPrototype = p.requirePrototype(common.CommonNameInt64Array)
	}
	return p.cachedInt64ArrayPrototype
}

// Uint8ArrayPrototype returns the cached Uint8Array class prototype.
func (p *Program) Uint8ArrayPrototype() *ClassPrototype {
	if p.cachedUint8ArrayPrototype == nil {
		p.cachedUint8ArrayPrototype = p.requirePrototype(common.CommonNameUint8Array)
	}
	return p.cachedUint8ArrayPrototype
}

// Uint8ClampedArrayPrototype returns the cached Uint8ClampedArray class prototype.
func (p *Program) Uint8ClampedArrayPrototype() *ClassPrototype {
	if p.cachedUint8ClampedArrayPrototype == nil {
		p.cachedUint8ClampedArrayPrototype = p.requirePrototype(common.CommonNameUint8ClampedArray)
	}
	return p.cachedUint8ClampedArrayPrototype
}

// Uint16ArrayPrototype returns the cached Uint16Array class prototype.
func (p *Program) Uint16ArrayPrototype() *ClassPrototype {
	if p.cachedUint16ArrayPrototype == nil {
		p.cachedUint16ArrayPrototype = p.requirePrototype(common.CommonNameUint16Array)
	}
	return p.cachedUint16ArrayPrototype
}

// Uint32ArrayPrototype returns the cached Uint32Array class prototype.
func (p *Program) Uint32ArrayPrototype() *ClassPrototype {
	if p.cachedUint32ArrayPrototype == nil {
		p.cachedUint32ArrayPrototype = p.requirePrototype(common.CommonNameUint32Array)
	}
	return p.cachedUint32ArrayPrototype
}

// Uint64ArrayPrototype returns the cached Uint64Array class prototype.
func (p *Program) Uint64ArrayPrototype() *ClassPrototype {
	if p.cachedUint64ArrayPrototype == nil {
		p.cachedUint64ArrayPrototype = p.requirePrototype(common.CommonNameUint64Array)
	}
	return p.cachedUint64ArrayPrototype
}

// Float32ArrayPrototype returns the cached Float32Array class prototype.
func (p *Program) Float32ArrayPrototype() *ClassPrototype {
	if p.cachedFloat32ArrayPrototype == nil {
		p.cachedFloat32ArrayPrototype = p.requirePrototype(common.CommonNameFloat32Array)
	}
	return p.cachedFloat32ArrayPrototype
}

// Float64ArrayPrototype returns the cached Float64Array class prototype.
func (p *Program) Float64ArrayPrototype() *ClassPrototype {
	if p.cachedFloat64ArrayPrototype == nil {
		p.cachedFloat64ArrayPrototype = p.requirePrototype(common.CommonNameFloat64Array)
	}
	return p.cachedFloat64ArrayPrototype
}

// ---------------------------------------------------------------------------
// String representation
// ---------------------------------------------------------------------------

// String returns a debug string for the program.
func (p *Program) String() string {
	return fmt.Sprintf("Program[files=%d, elements=%d, instances=%d]",
		len(p.Sources),
		len(p.ElementsByNameMap),
		len(p.InstancesByNameMap),
	)
}
