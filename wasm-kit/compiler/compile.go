// Ported from: assemblyscript/src/compiler.ts compile(), initDefaultMemory(), initDefaultTable()
// (lines 530-914)
package compiler

import (
	"fmt"

	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/flow"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/program"
	"github.com/brainlet/brainkit/wasm-kit/types"
	"github.com/brainlet/brainkit/wasm-kit/util"
)

// CompileProgram performs compilation of the underlying Program to a Module.
// This is the main compilation entry point.
// Ported from: assemblyscript/src/compiler.ts Compiler.compile() (lines 530-763).
func (c *Compiler) CompileProgram() *module.Module {
	options := c.Options()
	mod := c.Module()
	prog := c.Program
	resolver := c.Resolver()
	hasShadowStack := options.StackSize > 0 // implies runtime=incremental

	// initialize lookup maps, built-ins, imports, exports, etc.
	prog.Initialize()

	// Mark the module as closed-world for better optimization.
	// Binaryen treats all function references as leaked when not closed-world.
	mod.SetClosedWorld(true)

	// obtain the main start function
	startFunctionRef := c.CurrentFlow.TargetFunction
	startFunctionInstance := startFunctionRef.(*program.Function)

	// compile entry file(s) while traversing reachable elements
	for _, file := range prog.FilesByName {
		if file.Source.SourceKind == ast.SourceKindUserEntry {
			c.CompileFile(file)
			c.CompileModuleExports(file)
		}
	}

	// compile and export runtime if requested or necessary
	if options.ExportRuntime || (options.BindingsHint && c.DesiresExportRuntime) {
		for _, name := range RuntimeFunctionNames {
			instance := prog.RequireFunction(name, nil)
			if instance != nil && c.CompileFunction(instance) && !mod.HasExport(name) {
				mod.AddFunctionExport(instance.GetInternalName(), name)
			}
		}
		for _, name := range RuntimeGlobalNames {
			instance := prog.RequireGlobal(name)
			if instance != nil && c.CompileGlobal(instance) && !mod.HasExport(name) {
				mod.AddGlobalExport(instance.GetInternalName(), name)
			}
		}
	}

	// compile lazy functions
	lazyFunctions := c.LazyFunctions
	for {
		functionsToCompile := make([]*program.Function, 0, len(lazyFunctions))
		for instance := range lazyFunctions {
			functionsToCompile = append(functionsToCompile, instance)
		}
		// clear lazy set
		for k := range lazyFunctions {
			delete(lazyFunctions, k)
		}
		for _, instance := range functionsToCompile {
			c.CompileFunctionForced(instance)
		}
		if len(lazyFunctions) == 0 {
			break
		}
	}

	// set up override stubs
	functionTable := c.FunctionTable
	overrideStubs := c.OverrideStubs
	for i, instance := range functionTable {
		if instance.Is(uint32(common.CommonFlagsOverridden)) {
			functionTable[i] = c.EnsureOverrideStub(instance) // includes varargs stub
		} else if instance.Signature.RequiredParameters < int32(len(instance.Signature.ParameterTypes)) {
			functionTable[i] = c.EnsureVarargsStub(instance)
		}
	}
	overrideStubsSeen := make(map[*program.Function]struct{})
	for {
		// override stubs and overrides have cross-dependencies on each other, in that compiling
		// either may discover the respective other. do this in a loop until no more are found.
		resolver.DiscoveredOverride = false
		for instance := range overrideStubs {
			overrideInstances := resolver.ResolveOverrides(instance)
			if overrideInstances != nil {
				for _, oi := range overrideInstances {
					c.CompileFunction(oi)
				}
			}
			overrideStubsSeen[instance] = struct{}{}
		}
		if !(len(overrideStubs) > len(overrideStubsSeen) || resolver.DiscoveredOverride) {
			break
		}
	}
	for instance := range overrideStubs {
		c.FinalizeOverrideStub(instance)
	}

	// compile pending instanceof helpers
	for elem, name := range c.PendingInstanceOf {
		switch elem.GetElementKind() {
		case program.ElementKindClass, program.ElementKindInterface:
			c.FinalizeInstanceOf(elem.(*program.Class), name)
		case program.ElementKindClassPrototype, program.ElementKindInterfacePrototype:
			c.FinalizeAnyInstanceOf(elem.(*program.ClassPrototype), name)
		}
	}

	// finalize runtime features
	mod.RemoveGlobal(common.BuiltinNameRttiBase)
	if c.RuntimeFeatures&RuntimeFeaturesRtti != 0 {
		compileRTTI(c)
	}
	if c.RuntimeFeatures&RuntimeFeaturesVisitGlobals != 0 {
		compileVisitGlobals(c)
	}
	if c.RuntimeFeatures&RuntimeFeaturesVisitMembers != 0 {
		compileVisitMembers(c)
	}

	memoryOffset := i64Align(c.MemoryOffset, int64(options.UsizeType().ByteSize()))

	// finalize data
	mod.RemoveGlobal(common.BuiltinNameDataEnd)
	if (c.RuntimeFeatures&RuntimeFeaturesData != 0) || hasShadowStack {
		if options.IsWasm64() {
			mod.AddGlobal(common.BuiltinNameDataEnd, module.TypeRefI64, false,
				mod.I64(memoryOffset),
			)
		} else {
			mod.AddGlobal(common.BuiltinNameDataEnd, module.TypeRefI32, false,
				mod.I32(int32(memoryOffset)),
			)
		}
	}

	// finalize stack (grows down from __heap_base to __data_end)
	mod.RemoveGlobal(common.BuiltinNameStackPointer)
	if (c.RuntimeFeatures&RuntimeFeaturesStack != 0) || hasShadowStack {
		memoryOffset = i64Align(
			memoryOffset+int64(options.StackSize),
			int64(options.UsizeType().ByteSize()),
		)
		if options.IsWasm64() {
			mod.AddGlobal(common.BuiltinNameStackPointer, module.TypeRefI64, true,
				mod.I64(memoryOffset),
			)
		} else {
			mod.AddGlobal(common.BuiltinNameStackPointer, module.TypeRefI32, true,
				mod.I32(int32(memoryOffset)),
			)
		}
	}

	// finalize heap
	mod.RemoveGlobal(common.BuiltinNameHeapBase)
	if (c.RuntimeFeatures&RuntimeFeaturesHeap != 0) || hasShadowStack {
		if options.IsWasm64() {
			mod.AddGlobal(common.BuiltinNameHeapBase, module.TypeRefI64, false,
				mod.I64(memoryOffset),
			)
		} else {
			mod.AddGlobal(common.BuiltinNameHeapBase, module.TypeRefI32, false,
				mod.I32(int32(memoryOffset)),
			)
		}
	}

	// setup default memory & table
	c.initDefaultMemory(memoryOffset)
	c.initDefaultTable()

	// expose the arguments length helper if there are varargs exports
	if c.RuntimeFeatures&RuntimeFeaturesSetArgumentsLength != 0 {
		mod.AddFunction(common.BuiltinNameSetArgumentsLength, module.TypeRefI32, module.TypeRefNone, nil,
			mod.GlobalSet(c.EnsureArgumentsLength(), mod.LocalGet(0, module.TypeRefI32)),
		)
		mod.AddFunctionExport(common.BuiltinNameSetArgumentsLength, ExportNameSetArgumentsLength)
	}

	// NOTE: no more element compiles from here. may go to the start function!

	// compile the start function if not empty or if explicitly requested
	startIsEmpty := len(c.CurrentBody) == 0
	exportStart := options.ExportStart
	hasExportStart := options.ExportStartSet
	if !startIsEmpty || hasExportStart {
		signature := startFunctionInstance.Signature
		if !startIsEmpty && hasExportStart {
			mod.AddGlobal(common.BuiltinNameStarted, module.TypeRefI32, true, mod.I32(0))
			// prepend: if (__started) return;
			c.CurrentBody = append([]module.ExpressionRef{
				mod.If(
					mod.GlobalGet(common.BuiltinNameStarted, module.TypeRefI32),
					mod.Return(0),
					0,
				),
				mod.GlobalSet(common.BuiltinNameStarted, mod.I32(1)),
			}, c.CurrentBody...)
		}
		funcRef := mod.AddFunction(
			startFunctionInstance.GetInternalName(),
			signature.ParamRefs(),
			signature.ResultRefs(),
			typesToRefs(startFunctionInstance.GetNonParameterLocalTypes()),
			mod.Flatten(c.CurrentBody, module.TypeRefNone),
		)
		startFunctionInstance.Finalize(mod, funcRef)
		if !hasExportStart {
			mod.SetStart(funcRef)
		} else {
			if !util.IsIdentifier(exportStart) || mod.HasExport(exportStart) {
				c.Error(
					diagnostics.DiagnosticCodeStartFunctionName0IsInvalidOrConflictsWithAnotherExport,
					ast.NativeSource().GetRange(),
					exportStart, "", "",
				)
			} else {
				mod.AddFunctionExport(startFunctionInstance.GetInternalName(), exportStart)
			}
		}
	}

	// Run custom passes
	if hasShadowStack {
		c.ShadowStack.WalkModule()
	}
	if prog.Lookup("ASC_RTRACE") != nil {
		// RtraceMemory pass would be instantiated and run here.
		// Stubbed until passes are ported.
	}

	return mod
}

// initDefaultMemory sets up the module's default memory.
// Ported from: assemblyscript/src/compiler.ts initDefaultMemory (lines 765-854).
func (c *Compiler) initDefaultMemory(memoryOffset int64) {
	c.MemoryOffset = memoryOffset

	options := c.Options()
	mod := c.Module()
	memorySegments := c.MemorySegments

	var initialPages uint32
	maximumPages := module.UnlimitedMemory
	isSharedMemory := false

	if options.MemoryBase != 0 || len(memorySegments) > 0 {
		aligned := i64Align(memoryOffset, 0x10000)
		initialPages = uint32(aligned >> 16)
	}

	if options.InitialMemory != 0 {
		if options.InitialMemory < initialPages {
			c.Error(
				diagnostics.DiagnosticCodeModuleRequiresAtLeast0PagesOfInitialMemory,
				nil,
				fmt.Sprintf("%d", initialPages), "", "",
			)
		} else {
			initialPages = options.InitialMemory
		}
	}

	if options.MaximumMemory != 0 {
		if options.MaximumMemory < initialPages {
			c.Error(
				diagnostics.DiagnosticCodeModuleRequiresAtLeast0PagesOfMaximumMemory,
				nil,
				fmt.Sprintf("%d", initialPages), "", "",
			)
		} else {
			maximumPages = options.MaximumMemory
		}
	}

	if options.SharedMemory {
		isSharedMemory = true
		if options.MaximumMemory == 0 {
			c.Error(
				diagnostics.DiagnosticCodeSharedMemoryRequiresMaximumMemoryToBeDefined,
				nil,
				"", "", "",
			)
			isSharedMemory = false
		}
		if !options.HasFeature(common.FeatureThreads) {
			c.Error(
				diagnostics.DiagnosticCodeSharedMemoryRequiresFeatureThreadsToBeEnabled,
				nil,
				"", "", "",
			)
			isSharedMemory = false
		}
	}

	// check that we didn't exceed lowMemoryLimit already
	if options.LowMemoryLimit != 0 {
		lowMemoryLimit := int64(options.LowMemoryLimit & ^uint32(15))
		if memoryOffset > lowMemoryLimit {
			c.Error(
				diagnostics.DiagnosticCodeLowMemoryLimitExceededByStaticData01,
				nil,
				fmt.Sprintf("%d", memoryOffset),
				fmt.Sprintf("%d", lowMemoryLimit),
				"",
			)
		}
	}

	// Setup internal memory with default name "0"
	exportName := ""
	if options.ExportMemory {
		exportName = ExportNameMemory
	}
	segments := make([]module.MemorySegment, len(memorySegments))
	for i, seg := range memorySegments {
		segments[i] = *seg
	}
	mod.SetMemory(
		initialPages,
		maximumPages,
		segments,
		module.Target(options.Target),
		exportName,
		common.CommonNameDefaultMemory,
		isSharedMemory,
	)

	// import memory if requested
	if options.ImportMemory {
		mod.AddMemoryImport(
			common.CommonNameDefaultMemory,
			ImportNameDefaultNamespace,
			ImportNameMemory,
			isSharedMemory,
		)
	}
}

// initDefaultTable sets up the module's default function table.
// Ported from: assemblyscript/src/compiler.ts initDefaultTable (lines 856-914).
func (c *Compiler) initDefaultTable() {
	options := c.Options()
	mod := c.Module()

	// import and/or export table if requested
	if options.ImportTable {
		mod.AddTableImport(
			common.CommonNameDefaultTable,
			ImportNameDefaultNamespace,
			ImportNameTable,
		)
		mod.SetClosedWorld(false)
		if options.Pedantic && options.WillOptimize() {
			c.Pedantic(
				diagnostics.DiagnosticCodeImportingTheTableDisablesSomeIndirectCallOptimizations,
				nil,
				"", "", "",
			)
		}
	}
	if options.ExportTable {
		mod.AddTableExport(common.CommonNameDefaultTable, ExportNameTable)
		mod.SetClosedWorld(false)
		if options.Pedantic && options.WillOptimize() {
			c.Pedantic(
				diagnostics.DiagnosticCodeExportingTheTableDisablesSomeIndirectCallOptimizations,
				nil,
				"", "", "",
			)
		}
	}

	// set up function table (first elem is blank)
	tableBase := options.TableBase
	if tableBase == 0 {
		tableBase = 1 // leave first elem blank
	}
	functionTable := c.FunctionTable
	functionTableNames := make([]string, len(functionTable))
	for i, fn := range functionTable {
		functionTableNames[i] = fn.GetInternalName()
	}

	initialTableSize := tableBase + uint32(len(functionTable))
	maximumTableSize := module.UnlimitedTable

	if !(options.ImportTable || options.ExportTable) {
		// use fixed size for non-imported and non-exported tables
		maximumTableSize = initialTableSize
		if options.WillOptimize() {
			// Hint for directize pass
			mod.SetPassArgument("directize-initial-contents-immutable", "true")
		}
	}
	mod.AddFunctionTable(
		common.CommonNameDefaultTable,
		initialTableSize,
		maximumTableSize,
		functionTableNames,
		mod.I32(int32(tableBase)),
	)
}

// --- Stub methods for compilation phases not yet ported ---

// CompileModuleExports compiles module-level exports for a file.
// Ported from: assemblyscript/src/compiler.ts compileModuleExports (lines 916-935).
func (c *Compiler) CompileModuleExports(file *program.File) {
	exports := file.Exports
	if exports != nil {
		for exportName, element := range exports {
			c.compileModuleExport(exportName, element, "")
		}
	}
	// Handle re-exports (export * from)
	exportsStar := file.ExportsStar
	if exportsStar != nil {
		for _, reexportedFile := range exportsStar {
			c.CompileModuleExports(reexportedFile)
		}
	}
}

// compileModuleExport compiles the respective module export(s) for the specified element.
// Ported from: assemblyscript/src/compiler.ts compileModuleExport (lines 938-1073).
func (c *Compiler) compileModuleExport(name string, element program.DeclaredElement, prefix string) {
	mod := c.Module()

	switch element.GetElementKind() {
	case program.ElementKindFunctionPrototype:
		// obtain the default instance
		functionPrototype := element.(*program.FunctionPrototype)
		if !functionPrototype.Is(common.CommonFlagsGeneric) {
			functionInstance := c.Resolver().ResolveFunction(functionPrototype, nil, nil, program.ReportModeReport)
			if functionInstance != nil {
				c.compileModuleExport(name, functionInstance, prefix)
			}
			return
		}

	case program.ElementKindFunction:
		functionInstance := element.(*program.Function)
		if !functionInstance.HasDecorator(program.DecoratorFlagsBuiltin) {
			signature := functionInstance.Signature
			if signature.RequiredParameters < int32(len(signature.ParameterTypes)) {
				// utilize varargs stub to fill in omitted arguments
				functionInstance = c.EnsureVarargsStub(functionInstance)
				c.RuntimeFeatures |= RuntimeFeaturesSetArgumentsLength
			}
			c.CompileFunction(functionInstance)
			if functionInstance.Is(common.CommonFlagsCompiled) {
				exportName := prefix + name
				if !mod.HasExport(exportName) {
					mod.AddFunctionExport(functionInstance.GetInternalName(), exportName)
					c.HasCustomFunctionExports = true
					hasManagedOperands := signature.HasManagedOperands()
					if hasManagedOperands {
						if c.ShadowStack != nil {
							c.ShadowStack.NoteExport(exportName, signature.GetManagedOperandIndices())
						}
					}
					if !c.DesiresExportRuntime {
						thisType := signature.ThisType
						if (thisType != nil && lowerRequiresExportRuntime(thisType)) ||
							liftRequiresExportRuntime(signature.ReturnType) {
							c.DesiresExportRuntime = true
						} else {
							parameterTypes := signature.ParameterTypes
							for _, pt := range parameterTypes {
								if lowerRequiresExportRuntime(pt) {
									c.DesiresExportRuntime = true
									break
								}
							}
						}
					}
					if functionInstance.Signature.ReturnType.Kind == types.TypeKindFunc {
						mod.SetClosedWorld(false)
					}
				}
				return
			}
		}

	case program.ElementKindGlobal:
		global := element.(*program.Global)
		isConst := global.Is(common.CommonFlagsConst) || global.Is(common.CommonFlagsStatic|common.CommonFlagsReadonly)
		if !isConst && !c.Options().HasFeature(common.FeatureMutableGlobals) {
			c.Warning(
				diagnostics.DiagnosticCodeFeature0IsNotEnabled,
				global.IdentifierNode().GetRange(),
				"mutable-globals", "", "",
			)
			return
		}
		c.CompileGlobal(global)
		if global.Is(common.CommonFlagsCompiled) {
			exportName := prefix + name
			if !mod.HasExport(exportName) {
				mod.AddGlobalExport(element.GetInternalName(), exportName)
				if !c.DesiresExportRuntime {
					globalType := global.GetResolvedType()
					if liftRequiresExportRuntime(globalType) ||
						(!global.Is(common.CommonFlagsConst) && lowerRequiresExportRuntime(globalType)) {
						c.DesiresExportRuntime = true
					}
				}
				if global.GetResolvedType().Kind == types.TypeKindFunc {
					mod.SetClosedWorld(false)
				}
			}
			if global.GetResolvedType() == types.TypeV128 {
				typeNode := global.TypeNode()
				var rng *diagnostics.Range
				if typeNode != nil {
					rng = typeNode.GetRange()
				} else {
					rng = global.IdentifierNode().GetRange()
				}
				c.Warning(
					diagnostics.DiagnosticCodeExchangeOf0ValuesIsNotSupportedByAllEmbeddings,
					rng,
					"v128", "", "",
				)
			}
			return
		}

	case program.ElementKindEnum:
		c.CompileEnum(element.(*program.Enum))
		members := element.GetMembers()
		if members != nil {
			subPrefix := prefix + name + common.STATIC_DELIMITER
			for memberName, member := range members {
				if !member.Is(common.CommonFlagsPrivate) {
					c.compileModuleExport(memberName, member, subPrefix)
				}
			}
		}
		return

	case program.ElementKindEnumValue:
		enumValue := element.(*program.EnumValue)
		if !enumValue.IsImmutable && !c.Options().HasFeature(common.FeatureMutableGlobals) {
			c.Error(
				diagnostics.DiagnosticCodeFeature0IsNotEnabled,
				enumValue.IdentifierNode().GetRange(),
				"mutable-globals", "", "",
			)
			return
		}
		if enumValue.Is(common.CommonFlagsCompiled) {
			exportName := prefix + name
			if !mod.HasExport(exportName) {
				mod.AddGlobalExport(element.GetInternalName(), exportName)
			}
			return
		}
	}

	// Fallthrough: element kind not handled or not compiled
	c.Warning(
		diagnostics.DiagnosticCodeOnlyVariablesFunctionsAndEnumsBecomeWebassemblyModuleExports,
		element.IdentifierNode().GetRange(),
		"", "", "",
	)
}

// CompileGlobal is now in compile_global.go

// EnsureOverrideStub creates an override stub for the given function.
// An override stub redirects virtual calls to the actual override targeted by
// the call. It utilizes varargs stubs where necessary. Only a placeholder is
// created here; actual code is generated in FinalizeOverrideStub.
// Ported from: assemblyscript/src/compiler.ts ensureOverrideStub (lines 6768-6880).
func (c *Compiler) EnsureOverrideStub(original *program.Function) *program.Function {
	stub := original.OverrideStub
	if stub != nil {
		return stub
	}
	stub = original.NewStub("override", -1)
	original.OverrideStub = stub
	mod := c.Module()
	stub.Ref = mod.AddFunction(
		stub.GetInternalName(),
		stub.Signature.ParamRefs(),
		stub.Signature.ResultRefs(),
		nil,
		mod.Unreachable(),
	)
	c.OverrideStubs[original] = struct{}{}
	return stub
}

// EnsureVarargsStub creates a varargs stub for the given function.
// A varargs stub is a function called with omitted arguments being zeroed,
// reading the `argumentsLength` helper global to decide which initializers
// to inject before calling the original function. It is typically attempted
// to circumvent the varargs stub where possible, for example where omitted
// arguments are constants and can be inlined into the original call.
// Ported from: assemblyscript/src/compiler.ts ensureVarargsStub (lines 6538-6668).
func (c *Compiler) EnsureVarargsStub(original *program.Function) *program.Function {
	stub := original.VarargsStub
	if stub != nil {
		return stub
	}

	originalSignature := original.Signature
	originalParameterTypes := originalSignature.ParameterTypes
	originalParameterDeclarations := original.Prototype.FunctionTypeNode().Parameters
	returnType := originalSignature.ReturnType
	isInstance := original.Is(common.CommonFlagsInstance)

	// arguments excl. `this`, operands incl. `this`
	minArguments := originalSignature.RequiredParameters
	minOperands := minArguments
	maxArguments := int32(len(originalParameterTypes))
	maxOperands := maxArguments
	if isInstance {
		minOperands++
		maxOperands++
	}
	numOptional := maxOperands - minOperands
	if numOptional <= 0 {
		panic("ensureVarargsStub: numOptional must be > 0")
	}

	forwardedOperands := make([]module.ExpressionRef, minOperands)
	operandIndex := int32(0)
	stmts := make([]module.ExpressionRef, 0)

	// forward `this` if applicable
	mod := c.Module()
	thisType := originalSignature.ThisType
	if thisType != nil {
		forwardedOperands[0] = mod.LocalGet(0, thisType.ToRef())
		operandIndex = 1
	}

	// forward required arguments
	for i := int32(0); i < minArguments; i++ {
		paramType := originalParameterTypes[i]
		forwardedOperands[operandIndex] = mod.LocalGet(operandIndex, paramType.ToRef())
		operandIndex++
	}
	if operandIndex != minOperands {
		panic("ensureVarargsStub: operandIndex != minOperands")
	}

	// create the varargs stub
	stub = original.NewStub("varargs", maxArguments)
	original.VarargsStub = stub

	// compile initializers of omitted arguments in the scope of the stub,
	// accounting for additional locals and a proper `this` context.
	previousFlow := c.CurrentFlow
	fl := stub.Flow
	if original.Is(common.CommonFlagsConstructor) {
		fl.SetFlag(flow.FlowFlagCtorParamContext)
	}
	c.CurrentFlow = fl

	// create a br_table switching over the number of optional parameters provided
	numNames := numOptional + 1 // incl. outer block
	names := make([]string, numNames)
	ofN := fmt.Sprintf("of%d", numOptional)
	for i := int32(0); i < numNames; i++ {
		names[i] = fmt.Sprintf("%d%s", i, ofN)
	}
	argumentsLength := c.EnsureArgumentsLength()

	// condition is number of provided optional arguments, so subtract required arguments
	var switchCondition module.ExpressionRef
	if minArguments != 0 {
		switchCondition = mod.Binary(module.BinaryOpSubI32,
			mod.GlobalGet(argumentsLength, module.TypeRefI32),
			mod.I32(minArguments),
		)
	} else {
		switchCondition = mod.GlobalGet(argumentsLength, module.TypeRefI32)
	}

	table := mod.Block(names[0], []module.ExpressionRef{
		mod.Block("outOfRange", []module.ExpressionRef{
			mod.Switch(names, "outOfRange", switchCondition, 0),
		}, module.TypeRefNone),
		mod.Unreachable(),
	}, module.TypeRefNone)

	for i := int32(0); i < numOptional; i++ {
		paramType := originalParameterTypes[minArguments+i]
		declaration := originalParameterDeclarations[minArguments+i]
		var initExpr module.ExpressionRef

		if declaration.ParameterKind == ast.ParameterKindRest {
			// Rest parameters get an empty array literal
			arrExpr := ast.NewArrayLiteralExpression(nil, *declaration.GetRange().AtEnd())
			initExpr = c.compileArrayLiteral(arrExpr, paramType, ConstraintsConvExplicit)
			initExpr = mod.LocalSet(operandIndex, initExpr, paramType.IsManaged())
		} else if declaration.Initializer != nil {
			initExpr = c.CompileExpression(
				declaration.Initializer,
				paramType,
				ConstraintsConvImplicit,
			)
			initExpr = mod.LocalSet(operandIndex, initExpr, paramType.IsManaged())
		} else {
			c.Error(
				diagnostics.DiagnosticCodeOptionalParameterMustHaveAnInitializer,
				declaration.GetRange(),
				"", "", "",
			)
			initExpr = mod.Unreachable()
		}

		table = mod.Block(names[i+1], []module.ExpressionRef{
			table,
			initExpr,
		}, module.TypeRefNone)

		forwardedOperands = append(forwardedOperands, mod.LocalGet(operandIndex, paramType.ToRef()))
		operandIndex++
	}
	if operandIndex != maxOperands {
		panic("ensureVarargsStub: operandIndex != maxOperands")
	}

	stmts = append(stmts, table)
	// assume this will always succeed (can just use name as the reportNode)
	stmts = append(stmts, c.makeCallDirect(original, forwardedOperands, original.GetDeclaration(), false))

	c.CurrentFlow = previousFlow

	funcRef := mod.AddFunction(
		stub.GetInternalName(),
		stub.Signature.ParamRefs(),
		stub.Signature.ResultRefs(),
		typesToRefs(stub.GetNonParameterLocalTypes()),
		mod.Flatten(stmts, returnType.ToRef()),
	)
	stub.Set(common.CommonFlagsCompiled)
	stub.Finalize(mod, funcRef)
	return stub
}

// needToStack checks whether a managed operand expression needs to go through
// the shadow stack's ~tostack runtime call. Returns false for zero constants
// and known static GC object offsets (they don't need stack protection).
// Ported from: assemblyscript/src/compiler.ts needToStack (lines 6834-6847).
func (c *Compiler) needToStack(expr module.ExpressionRef) bool {
	mod := c.Module()
	precomp := mod.RunExpression(expr, module.ExpressionRunnerFlagsDefault, 8, 1)
	// cannot precompute, so must go to stack
	if precomp == 0 {
		return true
	}
	value := module.GetConstValueInteger(precomp, c.Options().IsWasm64())
	// zero constant doesn't need to go to stack
	if value == 0 {
		return false
	}
	// static GC objects don't need to go to stack
	hi := int32(value >> 32)
	lo := int32(value)
	if inner, ok := c.StaticGcObjectOffsets[hi]; ok {
		if _, ok2 := inner[lo]; ok2 {
			return false
		}
	}
	return true
}

// operandsTostack marks managed call operands for the shadow stack.
// For each operand whose type is managed and that needs stacking,
// wraps it through the ~tostack runtime call.
// Ported from: assemblyscript/src/compiler.ts operandsTostack (lines 6850-6878).
func (c *Compiler) operandsTostack(signature *types.Signature, operands []module.ExpressionRef) {
	if c.Options().StackSize <= 0 {
		return
	}
	mod := c.Module()
	operandIndex := 0
	thisType := signature.ThisType
	if thisType != nil {
		if thisType.IsManaged() {
			operand := operands[0]
			if c.needToStack(operand) {
				operands[operandIndex] = mod.Tostack(operand)
			}
		}
		operandIndex++
	}
	parameterIndex := 0
	parameterTypes := signature.ParameterTypes
	for operandIndex < len(operands) {
		paramType := parameterTypes[parameterIndex]
		if paramType.IsManaged() {
			operand := operands[operandIndex]
			if c.needToStack(operand) {
				operands[operandIndex] = mod.Tostack(operand)
			}
		}
		operandIndex++
		parameterIndex++
	}
}

// checkUnsafe checks if an unsafe operation is allowed. If noUnsafe is enabled
// and the source is not a library file, reports an error.
// Ported from: assemblyscript/src/compiler.ts checkUnsafe (lines 6319-6334).
func (c *Compiler) checkUnsafe(reportNode ast.Node, relatedReportNode ast.Node) {
	if c.Options().NoUnsafe {
		rng := reportNode.GetRange()
		if rng != nil && rng.Source != nil {
			if src, ok := rng.Source.(*ast.Source); ok && src.IsLibrary() {
				return // Library files may always use unsafe features
			}
		}
		if relatedReportNode != nil {
			c.ErrorRelated(
				diagnostics.DiagnosticCodeOperationIsUnsafe,
				reportNode.GetRange(),
				relatedReportNode.GetRange(),
				"", "", "",
			)
		} else {
			c.Error(
				diagnostics.DiagnosticCodeOperationIsUnsafe,
				reportNode.GetRange(),
				"", "", "",
			)
		}
	}
}

// adjustArgumentsForRestParams adjusts argument expressions for rest parameters.
// If the signature has rest params and more args than params, the trailing args
// are collected into an ArrayLiteralExpression at the rest position.
// Ported from: assemblyscript/src/compiler.ts adjustArgumentsForRestParams (lines 6336-6365).
func (c *Compiler) adjustArgumentsForRestParams(
	argumentExpressions []ast.Node,
	signature *types.Signature,
	reportNode ast.Node,
) []ast.Node {
	// if no rest args, return the original args
	if !signature.HasRest {
		return argumentExpressions
	}

	// if there are fewer args than params, then the rest args were not provided
	// so return the original args
	numArguments := len(argumentExpressions)
	numParams := len(signature.ParameterTypes)
	if numArguments < numParams {
		return argumentExpressions
	}

	// make an array literal expression from the rest args
	elements := make([]ast.Node, numArguments-numParams+1)
	copy(elements, argumentExpressions[numParams-1:])
	startRange := elements[0].GetRange()
	endRange := elements[len(elements)-1].GetRange()
	rng := diagnostics.Range{
		Start:  startRange.Start,
		End:    endRange.End,
		Source: reportNode.GetRange().Source,
	}
	arrExpr := ast.NewArrayLiteralExpression(elements, rng)

	// return the original args, but replace the rest args with the array
	exprs := make([]ast.Node, numParams)
	copy(exprs, argumentExpressions[:numParams-1])
	exprs[numParams-1] = arrExpr
	return exprs
}

// makeToString converts an expression to a string representation by looking up
// the toString() method on the expression's class or wrapper type.
// Ported from: assemblyscript/src/compiler.ts makeToString (lines 10287-10348).
func (c *Compiler) makeToString(expr module.ExpressionRef, typ *types.Type, reportNode ast.Node) module.ExpressionRef {
	mod := c.Module()
	stringInstance := c.Program.StringInstance()
	stringType := stringInstance.GetResolvedType()

	// If already a string, return as-is
	if typ == stringType {
		return expr
	}

	// Look up toString() on the class or wrapper type
	classType := typ.GetClassOrWrapper(c.Program)
	if classType != nil {
		classInstance := classType.(*program.Class)
		toStringInstance := classInstance.GetMethod("toString", nil)
		if toStringInstance != nil {
			toStringSignature := toStringInstance.Signature

			// Validate call signature (0 args, has this)
			if !c.checkCallSignature(toStringSignature, 0, true, reportNode) {
				c.CurrentType = stringType
				return mod.Unreachable()
			}

			// Check this-type compatibility
			thisType := toStringSignature.ThisType
			if !typ.IsStrictlyAssignableTo(thisType, false) {
				if !typ.Is(types.TypeFlagNullable) {
					toStringRange := toStringInstance.IdentifierAndSignatureRange()
					c.ErrorRelated(
						diagnostics.DiagnosticCodeTheThisTypesOfEachSignatureAreIncompatible,
						reportNode.GetRange(),
						&toStringRange,
						"", "", "",
					)
					c.CurrentType = stringType
					return mod.Unreachable()
				}

				// Attempt to retry on the non-nullable form of the type, wrapped in a ternary:
				// `expr ? expr.toString() : "null"`
				tempLocal := c.CurrentFlow.GetTempLocal(typ)
				return mod.If(
					mod.LocalTee(tempLocal.FlowIndex(), expr, typ.IsManaged(), typ.ToRef()),
					c.makeToString(
						mod.LocalGet(tempLocal.FlowIndex(), typ.ToRef()),
						typ.NonNullableType(),
						reportNode,
					),
					c.EnsureStaticString("null"),
				)
			}

			// Check return type compatibility
			toStringReturnType := toStringSignature.ReturnType
			if !toStringReturnType.IsStrictlyAssignableTo(stringType, false) {
				toStringRange := toStringInstance.IdentifierAndSignatureRange()
				c.ErrorRelated(
					diagnostics.DiagnosticCodeType0IsNotAssignableToType1,
					reportNode.GetRange(),
					&toStringRange,
					toStringReturnType.String(), stringType.String(), "",
				)
				c.CurrentType = stringType
				return mod.Unreachable()
			}

			return c.makeCallDirect(toStringInstance, []module.ExpressionRef{expr}, reportNode, false)
		}
	}

	// No toString available
	c.Error(
		diagnostics.DiagnosticCodeType0IsNotAssignableToType1,
		reportNode.GetRange(),
		typ.String(), stringType.String(), "",
	)
	c.CurrentType = stringType
	return mod.Unreachable()
}

// makeCallDirect creates a direct call to the specified function.
// Handles inline expansion, default parameter filling, varargs stubs,
// override stub dispatch, and managed operand stacking.
// Ported from: assemblyscript/src/compiler.ts makeCallDirect (lines 6881-7004).
func (c *Compiler) makeCallDirect(instance *program.Function, operands []module.ExpressionRef, reportNode ast.Node, immediatelyDropped bool) module.ExpressionRef {
	// Try to inline if decorated with @inline
	if instance.HasDecorator(program.DecoratorFlagsInline) {
		if !instance.Is(common.CommonFlagsOverridden) {
			if instance.Is(common.CommonFlagsStub) {
				panic("@inline on stub doesn't make sense")
			}
			inlineStack := c.InlineStack
			if inlineStackContains(inlineStack, instance) {
				c.Warning(
					diagnostics.DiagnosticCodeFunction0CannotBeInlinedIntoItself,
					reportNode.GetRange(),
					instance.GetInternalName(), "", "",
				)
			} else {
				inlineStack = append(inlineStack, instance)
				c.InlineStack = inlineStack
				var expr module.ExpressionRef
				if instance.Is(common.CommonFlagsInstance) {
					theOperands := operands
					if len(theOperands) == 0 {
						panic("instance method inline requires operands with this")
					}
					expr = c.makeCallInline(instance, theOperands[1:], theOperands[0], immediatelyDropped)
				} else {
					expr = c.makeCallInline(instance, operands, 0, immediatelyDropped)
				}
				c.InlineStack = c.InlineStack[:len(c.InlineStack)-1]
				return expr
			}
		} else {
			c.Warning(
				diagnostics.DiagnosticCodeFunction0IsVirtualAndWillNotBeInlined,
				reportNode.GetRange(),
				instance.GetInternalName(), "", "",
			)
		}
	}

	mod := c.Module()
	numOperands := len(operands)
	numArguments := numOperands
	minArguments := int(instance.Signature.RequiredParameters)
	minOperands := minArguments
	parameterTypes := instance.Signature.ParameterTypes
	maxArguments := len(parameterTypes)
	maxOperands := maxArguments
	if instance.Is(common.CommonFlagsInstance) {
		minOperands++
		maxOperands++
		numArguments--
	}
	if numOperands < minOperands {
		panic(fmt.Sprintf("makeCallDirect: numOperands %d < minOperands %d", numOperands, minOperands))
	}

	if !c.CompileFunction(instance) {
		return mod.Unreachable()
	}
	returnType := instance.Signature.ReturnType

	// fill up omitted arguments with their initializers, if constant, otherwise with zeroes.
	if numOperands < maxOperands {
		if operands == nil {
			operands = make([]module.ExpressionRef, 0, maxOperands)
		}
		parameterNodes := instance.Prototype.FunctionTypeNode().Parameters
		if len(parameterNodes) != len(parameterTypes) {
			panic("parameterNodes.length != parameterTypes.length")
		}
		allOptionalsAreConstant := true
		for i := numArguments; i < maxArguments; i++ {
			initializer := parameterNodes[i].Initializer
			if initializer != nil {
				if ast.CompilesToConst(initializer) {
					operands = append(operands, c.CompileExpression(
						initializer,
						parameterTypes[i],
						ConstraintsConvImplicit,
					))
					continue
				}
				resolved := c.Resolver().LookupExpression(initializer, instance.Flow, parameterTypes[i], program.ReportModeSwallow)
				if resolved != nil && resolved.GetElementKind() == program.ElementKindGlobal {
					global := resolved.(*program.Global)
					if c.CompileGlobalLazy(global, initializer) && global.Is(common.CommonFlagsInlined) {
						operands = append(operands, c.compileInlineConstant(global, parameterTypes[i], ConstraintsConvImplicit))
						continue
					}
				}
			}
			operands = append(operands, c.makeZeroOfType(parameterTypes[i]))
			allOptionalsAreConstant = false
		}
		if !allOptionalsAreConstant && !instance.Is(common.CommonFlagsModuleImport) {
			original := instance
			instance = c.EnsureVarargsStub(instance)
			if !c.CompileFunction(instance) {
				return mod.Unreachable()
			}
			instance.Flow.Flags = original.Flow.Flags
			returnTypeRef := returnType.ToRef()
			// We know the last operand is optional and omitted, so inject setting
			// ~argumentsLength into that operand, which is always safe.
			lastOperand := operands[maxOperands-1]
			if module.GetSideEffects(lastOperand, mod.BinaryenModule())&module.SideEffectWritesGlobal != 0 {
				panic("last operand writes global unexpectedly")
			}
			lastOperandType := parameterTypes[maxArguments-1]
			operands[maxOperands-1] = mod.Block("", []module.ExpressionRef{
				mod.GlobalSet(c.EnsureArgumentsLength(), mod.I32(int32(numArguments))),
				lastOperand,
			}, lastOperandType.ToRef())
			c.operandsTostack(instance.Signature, operands)
			expr := mod.Call(instance.GetInternalName(), operands, returnTypeRef)
			if returnType != types.TypeVoid && immediatelyDropped {
				expr = mod.Drop(expr)
				c.CurrentType = types.TypeVoid
			} else {
				c.CurrentType = returnType
			}
			return expr
		}
	}

	// Call the override stub if the function has overloads
	if instance.Is(common.CommonFlagsOverridden) && !ast.IsAccessOnSuper(reportNode) {
		instance = c.EnsureOverrideStub(instance)
	}

	if operands != nil {
		c.operandsTostack(instance.Signature, operands)
	}
	expr := mod.Call(instance.GetInternalName(), operands, returnType.ToRef())
	c.CurrentType = returnType
	return expr
}

// makeCallIndirect creates an indirect call to a first-class function.
// Handles zero-filling omitted arguments, injecting argumentsLength global set,
// side-effect handling for functionArg, shadow stack, and call_indirect emission.
// Ported from: assemblyscript/src/compiler.ts makeCallIndirect (lines 7047-7112).
func (c *Compiler) makeCallIndirect(
	signature *types.Signature,
	functionArg module.ExpressionRef,
	reportNode ast.Node,
	operands []module.ExpressionRef,
	immediatelyDropped bool,
) module.ExpressionRef {
	mod := c.Module()
	numOperands := len(operands)
	numArguments := numOperands
	minArguments := int(signature.RequiredParameters)
	minOperands := minArguments
	parameterTypes := signature.ParameterTypes
	returnType := signature.ReturnType
	maxArguments := len(parameterTypes)
	maxOperands := maxArguments
	if signature.ThisType != nil {
		minOperands++
		maxOperands++
		numArguments--
	}
	if numOperands < minOperands {
		panic(fmt.Sprintf("makeCallIndirect: numOperands %d < minOperands %d", numOperands, minOperands))
	}

	// fill up omitted arguments with zeroes
	if numOperands < maxOperands {
		if operands == nil {
			operands = make([]module.ExpressionRef, 0, maxOperands)
		}
		for i := numArguments; i < maxArguments; i++ {
			operands = append(operands, c.makeZeroOfType(parameterTypes[i]))
		}
	}

	// We might be calling a varargs stub here, even if all operands have been
	// provided, so we must set `argumentsLength` in any case. Inject setting it
	// into the index argument, which becomes executed last after any operands.
	argumentsLength := c.EnsureArgumentsLength()
	sizeTypeRef := module.TypeRef(c.Options().SizeTypeRef())
	if module.GetSideEffects(functionArg, mod.BinaryenModule())&module.SideEffectWritesGlobal != 0 {
		fl := c.CurrentFlow
		temp := fl.GetTempLocal(c.Options().UsizeType())
		tempIndex := temp.FlowIndex()
		functionArg = mod.Block("", []module.ExpressionRef{
			mod.LocalSet(tempIndex, functionArg, true), // Function
			mod.GlobalSet(argumentsLength, mod.I32(int32(numArguments))),
			mod.LocalGet(tempIndex, sizeTypeRef),
		}, sizeTypeRef)
	} else { // simplify
		functionArg = mod.Block("", []module.ExpressionRef{
			mod.GlobalSet(argumentsLength, mod.I32(int32(numArguments))),
			functionArg,
		}, sizeTypeRef)
	}
	if operands != nil {
		c.operandsTostack(signature, operands)
	}
	expr := mod.CallIndirect(
		"", // TODO: handle multiple tables
		mod.Load(4, false, functionArg, module.TypeRefI32, 0, 4, module.DefaultMemory), // ._index
		operands,
		signature.ParamRefs(),
		signature.ResultRefs(),
	)
	c.CurrentType = returnType
	return expr
}

// inlineStackContains checks if a function is already in the inline stack.
func inlineStackContains(stack []*program.Function, instance *program.Function) bool {
	for _, f := range stack {
		if f == instance {
			return true
		}
	}
	return false
}

// makeCallInline compiles an inlined call to the given function.
// Creates an inline flow, binds parameters as scoped locals, compiles the function
// body inline, and manages the return label for inline returns.
// Ported from: assemblyscript/src/compiler.ts makeCallInline (lines 6444-6525).
func (c *Compiler) makeCallInline(
	instance *program.Function,
	operands []module.ExpressionRef,
	thisArg module.ExpressionRef,
	immediatelyDropped bool,
) module.ExpressionRef {
	mod := c.Module()
	numArguments := len(operands)
	signature := instance.Signature
	parameterTypes := signature.ParameterTypes
	numParameters := len(parameterTypes)

	// Create a new inline flow and use it to compile the function as a block
	previousFlow := c.CurrentFlow
	fl := flow.CreateInline(previousFlow.TargetFunction, instance)
	body := make([]module.ExpressionRef, 0)

	if thisArg != 0 {
		parent := instance.GetParent()
		if parent == nil {
			panic("makeCallInline: parent must not be nil")
		}
		if parent.GetElementKind() != program.ElementKindClass {
			panic("makeCallInline: parent must be a class")
		}
		classInstance := parent.(*program.Class)
		thisType := instance.Signature.ThisType
		if thisType == nil {
			panic("makeCallInline: thisType must not be nil")
		}
		thisLocal := fl.AddScopedLocal(common.CommonNameThis, thisType)
		body = append(body,
			mod.LocalSet(thisLocal.FlowIndex(), thisArg, thisType.IsManaged()),
		)
		fl.SetLocalFlag(thisLocal.FlowIndex(), flow.LocalFlagInitialized)
		base := classInstance.Base
		if base != nil {
			fl.AddScopedAlias(common.CommonNameSuper, base.GetType(), thisLocal.FlowIndex(), nil)
		}
	} else {
		if instance.Signature.ThisType != nil {
			panic("makeCallInline: thisType must be nil for non-this calls")
		}
	}
	for i := 0; i < numArguments; i++ {
		paramExpr := operands[i]
		paramType := parameterTypes[i]
		argumentLocal := fl.AddScopedLocal(instance.GetParameterName(int32(i)), paramType)
		// inlining is aware of wrap/nonnull states:
		if !previousFlow.CanOverflow(paramExpr, paramType) {
			fl.SetLocalFlag(argumentLocal.FlowIndex(), flow.LocalFlagWrapped)
		}
		if fl.IsNonnull(paramExpr, paramType) {
			fl.SetLocalFlag(argumentLocal.FlowIndex(), flow.LocalFlagNonNull)
		}
		body = append(body,
			mod.LocalSet(argumentLocal.FlowIndex(), paramExpr, paramType.IsManaged()),
		)
		fl.SetLocalFlag(argumentLocal.FlowIndex(), flow.LocalFlagInitialized)
	}

	// Compile omitted arguments with final argument locals blocked. Doesn't need to take care of
	// side-effects within earlier expressions because these already happened on set.
	c.CurrentFlow = fl
	isConstructor := instance.Is(common.CommonFlagsConstructor)
	if isConstructor {
		fl.SetFlag(flow.FlowFlagCtorParamContext)
	}
	for i := numArguments; i < numParameters; i++ {
		initType := parameterTypes[i]
		funcTypeNode := instance.Prototype.FunctionTypeNode()
		if funcTypeNode == nil {
			panic("makeCallInline: function type node must not be nil")
		}
		paramNode := funcTypeNode.Parameters[i]
		if paramNode.Initializer == nil {
			panic("makeCallInline: parameter initializer must not be nil")
		}
		initExpr := c.CompileExpression(
			paramNode.Initializer,
			initType,
			ConstraintsConvImplicit,
		)
		argumentLocal := fl.AddScopedLocal(instance.GetParameterName(int32(i)), initType)
		body = append(body,
			c.makeLocalAssignment(argumentLocal.(*program.Local), initExpr, initType, false),
		)
	}
	fl.UnsetFlag(flow.FlowFlagCtorParamContext)

	// Compile the called function's body in the scope of the inlined flow
	c.compileFunctionBody(instance, &body)

	// If a constructor, perform field init checks on its flow directly
	if isConstructor {
		parent := instance.GetParent()
		if parent == nil || parent.GetElementKind() != program.ElementKindClass {
			panic("makeCallInline: inline constructor parent must be a class")
		}
		c.checkFieldInitializationInFlow(parent.(*program.Class), fl, nil)
	}

	// Free any new scoped locals and reset to the original flow
	returnType := fl.ReturnType()
	c.CurrentFlow = previousFlow

	// Create an outer block that we can break to when returning a value out of order
	c.CurrentType = returnType
	return mod.Block(fl.InlineReturnLabel, body, returnType.ToRef())
}

// FinalizeOverrideStub finalizes an override stub by building a switch over
// the runtime type ID to dispatch to the correct override.
// Ported from: assemblyscript/src/compiler.ts finalizeOverrideStub (lines 6882-6962).
func (c *Compiler) FinalizeOverrideStub(instance *program.Function) {
	stub := c.EnsureOverrideStub(instance)
	if stub.Is(common.CommonFlagsCompiled) {
		return
	}

	mod := c.Module()
	usizeType := c.Options().UsizeType()
	sizeTypeRef := usizeType.ToRef()
	parameterTypes := instance.Signature.ParameterTypes
	returnType := instance.Signature.ReturnType
	numParameters := len(parameterTypes)
	tempIndex := int32(1 + numParameters) // incl. `this`

	// Switch over this's rtId: load(4, false, this - 8, i32)
	var subExpr module.ExpressionRef
	if sizeTypeRef == module.TypeRefI64 {
		subExpr = mod.Binary(module.BinaryOpSubI64,
			mod.LocalGet(0, sizeTypeRef),
			mod.I64(8),
		)
	} else {
		subExpr = mod.Binary(module.BinaryOpSubI32,
			mod.LocalGet(0, sizeTypeRef),
			mod.I32(8),
		)
	}
	condition := mod.Load(4, false, subExpr, module.TypeRefI32, 0, 4, "")

	builder := module.NewSwitchBuilder(mod.BinaryenModule(), condition)

	overrideInstances := c.Resolver().ResolveOverrides(instance)
	if overrideInstances != nil {
		mostRecentInheritanceMapping := make(map[*program.Class]*program.Class)
		for _, overrideInstance := range overrideInstances {
			if !overrideInstance.Is(common.CommonFlagsCompiled) {
				continue // errored
			}

			overrideSignature := overrideInstance.Signature
			originalSignature := instance.Signature

			if !overrideSignature.IsAssignableTo(originalSignature, true) {
				identNode := overrideInstance.IdentifierNode()
				var identRange *diagnostics.Range
				if identNode != nil {
					identRange = identNode.GetRange()
				}
				c.Error(
					diagnostics.DiagnosticCodeType0IsNotAssignableToType1,
					identRange,
					overrideSignature.ToString(false),
					originalSignature.ToString(false),
					"",
				)
				continue
			}

			overrideParameterTypes := overrideSignature.ParameterTypes
			overrideNumParameters := len(overrideParameterTypes)
			paramExprs := make([]module.ExpressionRef, 1+overrideNumParameters)
			paramExprs[0] = mod.LocalGet(0, sizeTypeRef) // this
			for n := 1; n <= numParameters; n++ {
				paramExprs[n] = mod.LocalGet(int32(n), parameterTypes[n-1].ToRef())
			}
			needsVarargsStub := false
			for n := numParameters; n < overrideNumParameters; n++ {
				paramExprs[1+n] = c.makeZeroOfType(overrideParameterTypes[n])
				needsVarargsStub = true
			}

			calledName := overrideInstance.GetInternalName()
			if needsVarargsStub {
				calledName = c.EnsureVarargsStub(overrideInstance).GetInternalName()
			}
			returnTypeRef := overrideSignature.ReturnType.ToRef()
			stmts := make([]module.ExpressionRef, 0)
			if needsVarargsStub {
				stmts = append(stmts, mod.GlobalSet(c.EnsureArgumentsLength(), mod.I32(int32(numParameters))))
			}
			if returnType == types.TypeVoid {
				stmts = append(stmts,
					mod.Call(calledName, paramExprs, returnTypeRef),
					mod.Return(0),
				)
			} else {
				stmts = append(stmts,
					mod.Return(mod.Call(calledName, paramExprs, returnTypeRef)),
				)
			}

			classInstance := overrideInstance.GetBoundClassOrInterface()
			if classInstance != nil {
				builder.AddCase(int32(classInstance.Id()), stmts)
				// Also alias each extender inheriting this exact overload
				if classInstance.Extenders != nil {
					for extender := range classInstance.Extenders {
						if extender.Prototype.InstanceMembers != nil {
							decl := instance.GetDeclaration()
							if decl != nil {
								identNode := instance.IdentifierNode()
								if identNode != nil {
									if _, has := extender.Prototype.InstanceMembers[identNode.Text]; has {
										continue // skip those not inheriting
									}
								}
							}
						}
						prev, hasPrev := mostRecentInheritanceMapping[extender]
						if !hasPrev || !prev.ExtendsClass(classInstance) {
							mostRecentInheritanceMapping[extender] = classInstance
							builder.AddOrReplaceCase(int32(extender.Id()), stmts)
						}
					}
				}
			}
		}
	}

	// Call the original function if no other id matches and the method is not
	// abstract or part of an interface.
	var body module.ExpressionRef
	instanceClass := instance.GetBoundClassOrInterface()
	if !instance.Is(common.CommonFlagsAbstract) && !(instanceClass != nil && instanceClass.GetElementKind() == program.ElementKindInterface) {
		paramExprs := make([]module.ExpressionRef, 1+numParameters)
		paramExprs[0] = mod.LocalGet(0, sizeTypeRef) // this
		for i := 0; i < numParameters; i++ {
			paramExprs[1+i] = mod.LocalGet(int32(1+i), parameterTypes[i].ToRef())
		}
		body = mod.Call(instance.GetInternalName(), paramExprs, returnType.ToRef())
	} else {
		body = mod.Unreachable()
	}

	// Replace the placeholder stub function
	ref := stub.Ref
	if ref != 0 {
		mod.RemoveFunction(stub.GetInternalName())
	}
	stub.Ref = mod.AddFunction(
		stub.GetInternalName(),
		stub.Signature.ParamRefs(),
		stub.Signature.ResultRefs(),
		[]module.TypeRef{module.TypeRefI32},
		mod.Block("", []module.ExpressionRef{
			builder.Render(tempIndex, ""),
			body,
		}, returnType.ToRef()),
	)
	stub.Set(common.CommonFlagsCompiled)
}

// FinalizeInstanceOf compiles an instanceof helper for a class.
// Creates a function that checks if the runtime type ID of an object matches
// the given class or any of its extenders/implementers.
// Ported from: assemblyscript/src/compiler.ts finalizeInstanceOf (lines 10082-10128).
func (c *Compiler) FinalizeInstanceOf(classInstance *program.Class, name string) {
	prog := c.Program
	mod := c.Module()
	sizeTypeRef := c.Options().SizeTypeRef()

	stmts := make([]module.ExpressionRef, 0)

	// Compute rtId offset: totalOverhead - OBJECT.offsetof("rtId")
	objectInstance := prog.ObjectInstance()
	rtIdOffset := prog.TotalOverhead() - int32(objectInstance.Offsetof("rtId"))

	// local.set $1, load(4, false, this - rtIdOffset, i32)
	var subOp module.Op
	if sizeTypeRef == module.TypeRefI64 {
		subOp = module.BinaryOpSubI64
	} else {
		subOp = module.BinaryOpSubI32
	}
	subExpr := mod.Binary(subOp,
		mod.LocalGet(0, sizeTypeRef),
		mod.I32(rtIdOffset),
	)
	stmts = append(stmts, mod.LocalSet(1,
		mod.Load(4, false, subExpr, module.TypeRefI32, 0, 4, ""),
		false,
	))

	// Collect all matching instances
	if classInstance.IsInterface() {
		// Interface: check all implementers
		if classInstance.Implementers != nil {
			for impl := range classInstance.Implementers {
				stmts = append(stmts, mod.Br("is_instance",
					mod.Binary(module.BinaryOpEqI32,
						mod.LocalGet(1, module.TypeRefI32),
						mod.I32(int32(impl.Id())),
					),
					0,
				))
			}
		}
	} else {
		// Class: check self and all extenders
		stmts = append(stmts, mod.Br("is_instance",
			mod.Binary(module.BinaryOpEqI32,
				mod.LocalGet(1, module.TypeRefI32),
				mod.I32(int32(classInstance.Id())),
			),
			0,
		))
		if classInstance.Extenders != nil {
			for extender := range classInstance.Extenders {
				stmts = append(stmts, mod.Br("is_instance",
					mod.Binary(module.BinaryOpEqI32,
						mod.LocalGet(1, module.TypeRefI32),
						mod.I32(int32(extender.Id())),
					),
					0,
				))
			}
		}
	}

	stmts = append(stmts, mod.Return(mod.I32(0)))

	// Wrap in is_instance block
	isInstanceBlock := mod.Block("is_instance", stmts, module.TypeRefNone)

	mod.RemoveFunction(name)
	mod.AddFunction(name, sizeTypeRef, module.TypeRefI32,
		[]module.TypeRef{module.TypeRefI32},
		mod.Block("", []module.ExpressionRef{isInstanceBlock, mod.I32(1)}, module.TypeRefI32),
	)
}

// FinalizeAnyInstanceOf compiles a generic instanceof helper for a class prototype.
// Unlike FinalizeInstanceOf which targets a specific class, this targets all
// resolved instances of a prototype (generic class with different type args).
// Ported from: assemblyscript/src/compiler.ts finalizeAnyInstanceOf (lines 10130-10200).
func (c *Compiler) FinalizeAnyInstanceOf(prototype *program.ClassPrototype, name string) {
	mod := c.Module()
	sizeTypeRef := c.Options().UsizeType().ToRef()

	stmts := make([]module.ExpressionRef, 0)
	instances := prototype.Instances

	if instances != nil && len(instances) > 0 {
		prog := c.Program
		objectInstance := prog.ObjectInstance()
		rtIdOffset := prog.TotalOverhead() - int32(objectInstance.Offsetof("rtId"))

		var subOp module.Op
		if sizeTypeRef == module.TypeRefI64 {
			subOp = module.BinaryOpSubI64
		} else {
			subOp = module.BinaryOpSubI32
		}
		subExpr := mod.Binary(subOp,
			mod.LocalGet(0, sizeTypeRef),
			mod.I32(rtIdOffset),
		)
		stmts = append(stmts, mod.LocalSet(1,
			mod.Load(4, false, subExpr, module.TypeRefI32, 0, 4, ""),
			false,
		))

		// Collect all class instances (self + extenders for classes, implementers for interfaces)
		allInstances := make(map[*program.Class]struct{})
		for _, instance := range instances {
			if instance.IsInterface() {
				if instance.Implementers != nil {
					for impl := range instance.Implementers {
						allInstances[impl] = struct{}{}
					}
				}
			} else {
				allInstances[instance] = struct{}{}
				if instance.Extenders != nil {
					for extender := range instance.Extenders {
						allInstances[extender] = struct{}{}
					}
				}
			}
		}

		for inst := range allInstances {
			stmts = append(stmts, mod.Br("is_instance",
				mod.Binary(module.BinaryOpEqI32,
					mod.LocalGet(1, module.TypeRefI32),
					mod.I32(int32(inst.Id())),
				),
				0,
			))
		}
	}

	stmts = append(stmts, mod.Return(mod.I32(0)))

	isInstanceBlock := mod.Block("is_instance", stmts, module.TypeRefNone)

	mod.RemoveFunction(name)
	mod.AddFunction(name, sizeTypeRef, module.TypeRefI32,
		[]module.TypeRef{module.TypeRefI32},
		mod.Block("", []module.ExpressionRef{isInstanceBlock, mod.I32(1)}, module.TypeRefI32),
	)
}

// EnsureArgumentsLength ensures the __argumentsLength global exists and returns its name.
// Ported from: assemblyscript/src/compiler.ts ensureArgumentsLength.
func (c *Compiler) EnsureArgumentsLength() string {
	name := common.BuiltinNameArgumentsLength
	if c.BuiltinArgumentsLength == 0 {
		mod := c.Module()
		mod.AddGlobal(name, module.TypeRefI32, true, mod.I32(0))
		c.BuiltinArgumentsLength = 1 // mark as created
	}
	return name
}

// prepareInstanceOf ensures an instanceof helper exists for the given class and returns its name.
// Ported from: assemblyscript/src/compiler.ts prepareInstanceOf (lines 7789-7799).
func (c *Compiler) prepareInstanceOf(classInstance *program.Class) string {
	name := "~instanceof|" + classInstance.GetInternalName()
	if existing, ok := c.PendingInstanceOf[classInstance]; ok {
		return existing
	}
	c.PendingInstanceOf[classInstance] = name
	mod := c.Module()
	sizeTypeRef := c.Options().UsizeType().ToRef()
	mod.AddFunction(name, sizeTypeRef, module.TypeRefI32, nil, mod.Unreachable())
	return name
}

// prepareAnyInstanceOf ensures an instanceof helper exists for the given class prototype and returns its name.
// Ported from: assemblyscript/src/compiler.ts prepareAnyInstanceOf (lines 7932-7941).
func (c *Compiler) prepareAnyInstanceOf(prototype *program.ClassPrototype) string {
	name := "~anyinstanceof|" + prototype.GetInternalName()
	if existing, ok := c.PendingInstanceOf[prototype]; ok {
		return existing
	}
	c.PendingInstanceOf[prototype] = name
	mod := c.Module()
	sizeTypeRef := c.Options().UsizeType().ToRef()
	mod.AddFunction(name, sizeTypeRef, module.TypeRefI32, nil, mod.Unreachable())
	return name
}

// makeAbort makes a call to abort, if present, otherwise creates a trap.
// Compiles the message expression (if any) and delegates to makeStaticAbort.
// Ported from: assemblyscript/src/compiler.ts makeAbort (lines 10481-10500).
func (c *Compiler) makeAbort(message ast.Node, codeLocation ast.Node) module.ExpressionRef {
	prog := c.Program
	abortInstance := prog.AbortInstance()
	if abortInstance == nil || !c.CompileFunction(abortInstance) {
		return c.Module().Unreachable()
	}

	stringInstance := prog.StringInstance()
	var messageArg module.ExpressionRef
	if message != nil {
		messageArg = c.CompileExpression(message, stringInstance.GetType(), ConstraintsConvImplicit)
	} else {
		messageArg = c.makeZeroOfType(stringInstance.GetType())
	}

	return c.MakeStaticAbort(messageArg, codeLocation)
}

// MakeStaticAbort makes a call to abort, if present, otherwise creates a trap.
// Ported from: assemblyscript/src/compiler.ts makeStaticAbort.
func (c *Compiler) MakeStaticAbort(messageExpr module.ExpressionRef, codeLocation ast.Node) module.ExpressionRef {
	mod := c.Module()
	abortInstance := c.Program.AbortInstance()
	if abortInstance == nil || !c.CompileFunction(abortInstance) {
		return mod.Unreachable()
	}

	rng := codeLocation.GetRange()
	if rng == nil || rng.Source == nil {
		return mod.Block("", []module.ExpressionRef{
			mod.Call(abortInstance.GetInternalName(), []module.ExpressionRef{
				messageExpr,
				c.EnsureStaticString(""),
				mod.I32(0),
				mod.I32(0),
			}, module.TypeRefNone),
			mod.Unreachable(),
		}, module.TypeRefUnreachable)
	}

	filenameExpr := c.EnsureStaticString(rng.Source.SourceNormalizedPath())
	line := rng.Source.LineAt(rng.Start)
	col := rng.Source.ColumnAt()
	return mod.Block("", []module.ExpressionRef{
		mod.Call(abortInstance.GetInternalName(), []module.ExpressionRef{
			messageExpr,
			filenameExpr,
			mod.I32(line),
			mod.I32(col),
		}, module.TypeRefNone),
		mod.Unreachable(),
	}, module.TypeRefUnreachable)
}

// compileRTTI compiles runtime type information.
func compileRTTI(c *Compiler) {
	prog := c.Program
	mod := c.Module()
	managedClasses := prog.ManagedClasses
	count := len(managedClasses)
	data := make([]byte, 4+4*count) // count | TypeInfo*
	util.WriteI32(int32(count), data, 0)

	abvInstance := prog.ArrayBufferViewInstance()
	var abvPrototype *program.ClassPrototype
	if abvInstance != nil {
		abvPrototype = abvInstance.Prototype
	}
	arrayPrototype := prog.ArrayPrototype()
	setPrototype := prog.SetPrototype()
	mapPrototype := prog.MapPrototype()
	staticArrayPrototype := prog.StaticArrayPrototype()

	off := 4
	for instanceID := int32(0); instanceID < int32(count); instanceID++ {
		instance, ok := managedClasses[instanceID]
		if !ok || instance == nil {
			panic("missing managed class for runtime id")
		}

		var flags common.TypeinfoFlags
		if instance.IsPointerfree() {
			flags |= common.TypeinfoFlagsPOINTERFREE
		}

		switch {
		case abvPrototype != nil && instance != abvInstance && instance.ExtendsPrototype(abvPrototype):
			valueType := instance.GetArrayValueType()
			flags |= common.TypeinfoFlagsARRAYBUFFERVIEW
			flags |= common.TypeinfoFlagsVALUE_ALIGN_0 * typeToRuntimeFlags(valueType)

		case arrayPrototype != nil && instance.ExtendsPrototype(arrayPrototype):
			valueType := instance.GetArrayValueType()
			flags |= common.TypeinfoFlagsARRAY
			flags |= common.TypeinfoFlagsVALUE_ALIGN_0 * typeToRuntimeFlags(valueType)

		case setPrototype != nil && instance.ExtendsPrototype(setPrototype):
			typeArguments := instance.GetTypeArgumentsTo(setPrototype)
			if len(typeArguments) == 1 {
				flags |= common.TypeinfoFlagsSET
				flags |= common.TypeinfoFlagsVALUE_ALIGN_0 * typeToRuntimeFlags(typeArguments[0])
			}

		case mapPrototype != nil && instance.ExtendsPrototype(mapPrototype):
			typeArguments := instance.GetTypeArgumentsTo(mapPrototype)
			if len(typeArguments) == 2 {
				flags |= common.TypeinfoFlagsMAP
				flags |= common.TypeinfoFlagsKEY_ALIGN_0 * typeToRuntimeFlags(typeArguments[0])
				flags |= common.TypeinfoFlagsVALUE_ALIGN_0 * typeToRuntimeFlags(typeArguments[1])
			}

		case staticArrayPrototype != nil && instance.ExtendsPrototype(staticArrayPrototype):
			valueType := instance.GetArrayValueType()
			flags |= common.TypeinfoFlagsSTATICARRAY
			flags |= common.TypeinfoFlagsVALUE_ALIGN_0 * typeToRuntimeFlags(valueType)
		}

		util.WriteI32(int32(flags), data, off)
		off += 4
		instance.RttiFlags = uint32(flags)
	}

	segment := c.addAlignedMemorySegment(data, 16)
	offset := module.GetConstValueInteger(segment.Offset, c.Options().IsWasm64())
	if c.Options().IsWasm64() {
		mod.AddGlobal(common.BuiltinNameRttiBase, module.TypeRefI64, false, mod.I64(offset))
	} else {
		mod.AddGlobal(common.BuiltinNameRttiBase, module.TypeRefI32, false, mod.I32(int32(offset)))
	}
}

// compileVisitGlobals compiles the __visit_globals function.
func compileVisitGlobals(c *Compiler) {
	mod := c.Module()
	sizeTypeRef := c.Options().SizeTypeRef()
	visitInstance := c.Program.VisitInstance()
	if visitInstance == nil {
		panic("missing __visit runtime function")
	}

	c.CompileFunctionForced(visitInstance)

	exprs := make([]module.ExpressionRef, 0)
	for _, element := range c.Program.ElementsByNameMap {
		global, ok := element.(*program.Global)
		if !ok {
			continue
		}

		globalType := global.GetResolvedType()
		if globalType == nil {
			continue
		}
		classReference := globalType.GetClass()
		if classReference == nil ||
			classReference.HasDecorator(uint32(program.DecoratorFlagsUnmanaged)) ||
			!global.Is(common.CommonFlagsCompiled) {
			continue
		}

		if global.Is(common.CommonFlagsInlined) {
			value := global.GetConstantIntegerValue()
			if value == 0 {
				continue
			}
			valueExpr := mod.I32(int32(value))
			if c.Options().IsWasm64() {
				valueExpr = mod.I64(value)
			}
			exprs = append(exprs, mod.Call(visitInstance.GetInternalName(), []module.ExpressionRef{
				valueExpr,
				mod.LocalGet(0, module.TypeRefI32),
			}, module.TypeRefNone))
			continue
		}

		exprs = append(exprs, mod.If(
			mod.LocalTee(1, mod.GlobalGet(global.GetInternalName(), sizeTypeRef), false, sizeTypeRef),
			mod.Call(visitInstance.GetInternalName(), []module.ExpressionRef{
				mod.LocalGet(1, sizeTypeRef),
				mod.LocalGet(0, module.TypeRefI32),
			}, module.TypeRefNone),
			0,
		))
	}

	body := mod.Nop()
	if len(exprs) != 0 {
		body = mod.Block("", exprs, module.TypeRefNone)
	}
	mod.AddFunction(
		common.BuiltinNameVisitGlobals,
		module.TypeRefI32,
		module.TypeRefNone,
		[]module.TypeRef{sizeTypeRef},
		body,
	)
}

// compileVisitMembers compiles the __visit_members function.
func compileVisitMembers(c *Compiler) {
	prog := c.Program
	mod := c.Module()
	usizeType := c.Options().UsizeType()
	sizeTypeRef := usizeType.ToRef()
	visitInstance := prog.VisitInstance()
	if visitInstance == nil {
		panic("missing __visit runtime function")
	}
	c.CompileFunctionForced(visitInstance)

	count := len(prog.ManagedClasses)
	params := module.CreateType([]module.TypeRef{sizeTypeRef, module.TypeRefI32})
	if count == 0 {
		mod.AddFunction(common.BuiltinNameVisitMembers, params, module.TypeRefNone, nil, mod.Unreachable())
		return
	}

	names := make([]string, count)
	cases := make([]module.ExpressionRef, count)
	for i := 0; i < count; i++ {
		instance := prog.ManagedClasses[int32(i)]
		if instance == nil {
			panic("missing managed class for runtime id")
		}
		names[i] = instance.GetInternalName()
		if instance.IsPointerfree() {
			cases[i] = mod.Return(0)
			continue
		}
		cases[i] = mod.Block("", []module.ExpressionRef{
			mod.Call(instance.GetInternalName()+"~visit", []module.ExpressionRef{
				mod.LocalGet(0, sizeTypeRef),
				mod.LocalGet(1, module.TypeRefI32),
			}, module.TypeRefNone),
			mod.Return(0),
		}, module.TypeRefNone)
		ensureVisitMembersOf(c, instance)
	}

	subExpr := mod.Binary(module.BinaryOpSubI32,
		mod.LocalGet(0, sizeTypeRef),
		mod.I32(8),
	)
	if sizeTypeRef == module.TypeRefI64 {
		subExpr = mod.Binary(module.BinaryOpSubI64,
			mod.LocalGet(0, sizeTypeRef),
			mod.I64(8),
		)
	}

	current := mod.Block(names[0], []module.ExpressionRef{
		mod.Switch(
			names,
			"invalid",
			mod.Load(4, false, subExpr, module.TypeRefI32, 0, 4, common.CommonNameDefaultMemory),
			0,
		),
	}, module.TypeRefNone)
	for i := 0; i < count-1; i++ {
		current = mod.Block(names[i+1], []module.ExpressionRef{
			current,
			cases[i],
		}, module.TypeRefNone)
	}
	current = mod.Block("invalid", []module.ExpressionRef{
		current,
		cases[count-1],
	}, module.TypeRefNone)

	mod.AddFunction(
		common.BuiltinNameVisitMembers,
		params,
		module.TypeRefNone,
		nil,
		mod.Flatten([]module.ExpressionRef{
			current,
			mod.Unreachable(),
		}, module.TypeRefNone),
	)
}

// compileCallExpressionBuiltin compiles a call to a builtin function.
// Ported from: assemblyscript/src/compiler.ts compileCallExpressionBuiltin (lines 6215-6268).
func (c *Compiler) compileCallExpressionBuiltin(
	prototype *program.FunctionPrototype,
	expression *ast.CallExpression,
	contextualType *types.Type,
) module.ExpressionRef {
	// Check @unsafe decorator
	if prototype.HasDecorator(program.DecoratorFlagsUnsafe) {
		c.checkUnsafe(expression, expression)
	}

	var typeArguments []*types.Type

	// Builtins handle omitted type arguments on their own. If present,
	// resolve them here and pass them to the builtin, even if it's still
	// up to the builtin how to handle them.
	typeParameterNodes := prototype.TypeParameterNodes()
	typeArgumentNodes := expression.TypeArguments
	if typeArgumentNodes != nil {
		if !prototype.Is(common.CommonFlagsGeneric) {
			c.Error(
				diagnostics.DiagnosticCodeType0IsNotGeneric,
				expression.GetRange(),
				prototype.GetInternalName(), "", "",
			)
		}
		if typeParameterNodes != nil {
			ctxTypeArgs := util.CloneMap(c.CurrentFlow.ContextualTypeArguments())
			parent, _ := c.CurrentFlow.SourceFunction().FlowParent().(program.Element)
			typeArguments = c.Resolver().ResolveTypeArguments(
				typeParameterNodes,
				typeArgumentNodes,
				c.CurrentFlow,
				parent,
				ctxTypeArgs,
				expression,
				program.ReportModeReport,
			)
		}
	}

	// Build context
	callee := expression.Expression
	var thisOperand ast.Node
	if callee.GetKind() == ast.NodeKindPropertyAccess {
		thisOperand = callee.(*ast.PropertyAccessExpression).Expression
	}
	ctx := &BuiltinFunctionContext{
		Compiler:       c,
		Prototype:      prototype,
		TypeArguments:  typeArguments,
		Operands:       expression.Args,
		ThisOperand:    thisOperand,
		ContextualType: contextualType,
		ReportNode:     expression,
		ContextIsExact: false,
	}

	// Compute internal name for dispatch
	var internalName string
	if prototype.Is(common.CommonFlagsInstance) {
		// Omit generic name components, e.g. in Function<...>#call
		parent := prototype.GetBoundClassOrInterface()
		if parent != nil {
			internalName = parent.Prototype.GetInternalName() + "#" + prototype.GetName()
		} else {
			internalName = prototype.GetInternalName()
		}
	} else {
		internalName = prototype.GetInternalName()
	}

	// Dispatch to handler
	fn := GetBuiltinHandler(internalName)
	if fn == nil {
		panic(fmt.Sprintf("missing builtin handler for: %s", internalName))
	}
	return fn(ctx)
}

// checkFieldInitialization checks that fields are properly initialized.
// Ported from: assemblyscript/src/compiler.ts checkFieldInitialization (lines 6932-6985).
func (c *Compiler) checkFieldInitialization(classInstance *program.Class, reportNode ast.Node) {
	if classInstance == nil || classInstance.DidCheckFieldInitialization {
		return
	}
	classInstance.DidCheckFieldInitialization = true

	ctor := classInstance.ConstructorInstance
	if ctor == nil {
		ctor = c.ensureConstructor(classInstance, reportNode)
	}
	if ctor == nil || ctor.Flow == nil {
		return
	}

	c.checkFieldInitializationInFlow(classInstance, ctor.Flow, reportNode)
}

// checkFieldInitializationInFlow checks that all own class fields are initialized in the specified flow.
// Ported from: assemblyscript/src/compiler.ts checkFieldInitializationInFlow (lines 9006-9044).
func (c *Compiler) checkFieldInitializationInFlow(classInstance *program.Class, fl *flow.Flow, relatedNode ast.Node) {
	if classInstance == nil || fl == nil {
		return
	}

	members := classInstance.OrderedOwnMembers()
	if len(members) == 0 {
		return
	}

	for _, member := range members {
		propertyPrototype, ok := member.(*program.PropertyPrototype)
		if !ok {
			continue
		}

		property := propertyPrototype.PropertyInstance
		if property == nil || !property.IsField() {
			continue
		}

		propertyRange := property.GetDeclaration().GetRange()
		if ident := property.IdentifierNode(); ident != nil {
			propertyRange = ident.GetRange()
		}

		if property.InitializerNode() == nil && !fl.IsThisFieldFlag(property, flow.FieldFlagInitialized) {
			if !property.Is(common.CommonFlagsDefinitelyAssigned) {
				if relatedNode != nil {
					c.ErrorRelated(
						diagnostics.DiagnosticCodeProperty0HasNoInitializerAndIsNotAssignedInTheConstructorBeforeThisIsUsedOrReturned,
						propertyRange,
						relatedNode.GetRange(),
						property.GetInternalName(),
						"",
						"",
					)
				} else {
					c.Error(
						diagnostics.DiagnosticCodeProperty0HasNoInitializerAndIsNotAssignedInTheConstructorBeforeThisIsUsedOrReturned,
						propertyRange,
						property.GetInternalName(),
						"",
						"",
					)
				}
			}
		} else if property.Is(common.CommonFlagsDefinitelyAssigned) {
			propertyType := property.GetResolvedType()
			if propertyType != nil && propertyType.IsReference() {
				c.Warning(
					diagnostics.DiagnosticCodeProperty0IsAlwaysAssignedBeforeBeingUsed,
					propertyRange,
					property.GetInternalName(),
					"",
					"",
				)
			} else {
				c.Pedantic(
					diagnostics.DiagnosticCodeUnnecessaryDefiniteAssignment,
					propertyRange,
					"",
					"",
					"",
				)
			}
		}
	}
}

// ensureSmallIntegerWrap ensures a small integer value is properly wrapped/sign-extended.
// Ported from: assemblyscript/src/compiler.ts ensureSmallIntegerWrap (lines 9962-10024).
func (c *Compiler) ensureSmallIntegerWrap(expr module.ExpressionRef, typ *types.Type) module.ExpressionRef {
	mod := c.Module()
	fl := c.CurrentFlow
	switch typ.Kind {
	case types.TypeKindBool:
		if fl.CanOverflow(expr, typ) {
			expr = mod.Binary(module.BinaryOpNeI32, expr, mod.I32(0))
		}
	case types.TypeKindI8:
		if fl.CanOverflow(expr, typ) {
			if c.Options().HasFeature(common.FeatureSignExtension) {
				expr = mod.Unary(module.UnaryOpExtend8I32, expr)
			} else {
				expr = mod.Binary(module.BinaryOpShrI32,
					mod.Binary(module.BinaryOpShlI32, expr, mod.I32(24)),
					mod.I32(24),
				)
			}
		}
	case types.TypeKindI16:
		if fl.CanOverflow(expr, typ) {
			if c.Options().HasFeature(common.FeatureSignExtension) {
				expr = mod.Unary(module.UnaryOpExtend16I32, expr)
			} else {
				expr = mod.Binary(module.BinaryOpShrI32,
					mod.Binary(module.BinaryOpShlI32, expr, mod.I32(16)),
					mod.I32(16),
				)
			}
		}
	case types.TypeKindU8:
		if fl.CanOverflow(expr, typ) {
			expr = mod.Binary(module.BinaryOpAndI32, expr, mod.I32(0xff))
		}
	case types.TypeKindU16:
		if fl.CanOverflow(expr, typ) {
			expr = mod.Binary(module.BinaryOpAndI32, expr, mod.I32(0xffff))
		}
	}
	return expr
}

// makeRuntimeNonNullCheck inserts a runtime non-null assertion check.
// Ported from: assemblyscript/src/compiler.ts makeRuntimeNonNullCheck.
func (c *Compiler) makeRuntimeNonNullCheck(expr module.ExpressionRef, typ *types.Type, reportNode ast.Node) module.ExpressionRef {
	mod := c.Module()
	fl := c.CurrentFlow
	temp := fl.GetTempLocal(typ)
	tempIndex := temp.FlowIndex()
	if !fl.CanOverflow(expr, typ) {
		fl.SetLocalFlag(tempIndex, flow.LocalFlagWrapped)
	}
	fl.SetLocalFlag(tempIndex, flow.LocalFlagNonNull)

	staticAbortCallExpr := c.MakeStaticAbort(
		c.EnsureStaticString("Unexpected 'null' (not assigned or failed cast)"),
		reportNode,
	)

	if typ.IsExternalReference() {
		nonNullExpr := mod.LocalGet(tempIndex, typ.ToRef())
		if c.Options().HasFeature(common.FeatureGC) {
			nonNullExpr = mod.RefAsNonNull(nonNullExpr)
		}
		expr = mod.If(
			mod.RefIsNull(mod.LocalTee(tempIndex, expr, false, typ.ToRef())),
			staticAbortCallExpr,
			nonNullExpr,
		)
	} else {
		expr = mod.If(
			mod.LocalTee(tempIndex, expr, typ.IsManaged(), typ.ToRef()),
			mod.LocalGet(tempIndex, typ.ToRef()),
			staticAbortCallExpr,
		)
	}
	c.CurrentType = typ.NonNullableType()
	return expr
}

// makeRuntimeDowncastCheck inserts a runtime downcast type check.
// Ported from: assemblyscript/src/compiler.ts makeRuntimeDowncastCheck.
func (c *Compiler) makeRuntimeDowncastCheck(expr module.ExpressionRef, fromType, toType *types.Type, reportNode ast.Node) module.ExpressionRef {
	if !toType.IsReference() || !toType.NonNullableType().IsAssignableTo(fromType, false) {
		panic("invalid runtime downcast")
	}

	mod := c.Module()
	fl := c.CurrentFlow
	temp := fl.GetTempLocal(fromType)
	tempIndex := temp.FlowIndex()

	staticAbortCallExpr := c.MakeStaticAbort(
		c.EnsureStaticString("invalid downcast"),
		reportNode,
	)

	classRef := toType.GetClass()
	classInstance, ok := classRef.(*program.Class)
	if !ok || classInstance == nil {
		return mod.Unreachable()
	}
	instanceofName := c.prepareInstanceOf(classInstance)

	if !toType.IsNullableReference() || fl.IsNonnull(expr, fromType) {
		expr = mod.If(
			mod.Call(instanceofName, []module.ExpressionRef{
				mod.LocalTee(tempIndex, expr, fromType.IsManaged(), fromType.ToRef()),
			}, module.TypeRefI32),
			mod.LocalGet(tempIndex, fromType.ToRef()),
			staticAbortCallExpr,
		)
	} else {
		expr = mod.If(
			mod.Unary(module.UnaryOpEqzI32, mod.LocalTee(tempIndex, expr, fromType.IsManaged(), fromType.ToRef())),
			c.makeZeroOfType(toType),
			mod.If(
				mod.Call(instanceofName, []module.ExpressionRef{
					mod.LocalGet(tempIndex, fromType.ToRef()),
				}, module.TypeRefI32),
				mod.LocalGet(tempIndex, fromType.ToRef()),
				staticAbortCallExpr,
			),
		)
	}
	c.CurrentType = toType
	return expr
}

// --- Utility functions ---

// i64Align aligns a value to the given alignment.
func i64Align(value, alignment int64) int64 {
	mask := alignment - 1
	return (value + mask) & ^mask
}

func typeToRuntimeFlags(typ *types.Type) common.TypeinfoFlags {
	flags := common.TypeinfoFlagsVALUE_ALIGN_0 * common.TypeinfoFlags(1<<typ.AlignLog2())
	if typ.Is(types.TypeFlagSigned) {
		flags |= common.TypeinfoFlagsVALUE_SIGNED
	}
	if typ.Is(types.TypeFlagFloat) {
		flags |= common.TypeinfoFlagsVALUE_FLOAT
	}
	if typ.Is(types.TypeFlagNullable) {
		flags |= common.TypeinfoFlagsVALUE_NULLABLE
	}
	if typ.IsManaged() {
		flags |= common.TypeinfoFlagsVALUE_MANAGED
	}
	return flags / common.TypeinfoFlagsVALUE_ALIGN_0
}

func (c *Compiler) addAlignedMemorySegment(buffer []byte, alignment int32) *module.MemorySegment {
	if !util.IsPowerOf2(alignment) {
		panic("alignment must be a power of two")
	}

	alignedOffset := i64Align(c.MemoryOffset, int64(alignment))
	var offsetExpr module.ExpressionRef
	if c.Options().IsWasm64() {
		offsetExpr = c.Module().I64(alignedOffset)
	} else {
		offsetExpr = c.Module().I32(int32(alignedOffset))
	}

	segment := &module.MemorySegment{
		Buffer:    buffer,
		Offset:    offsetExpr,
		RawOffset: alignedOffset,
	}
	c.MemorySegments = append(c.MemorySegments, segment)
	c.MemoryOffset = alignedOffset + int64(len(buffer))
	return segment
}

// addRuntimeMemorySegment adds a static memory segment representing a runtime object.
// Ported from: assemblyscript/src/compiler.ts addRuntimeMemorySegment (lines 1953-1959).
func (c *Compiler) addRuntimeMemorySegment(buffer []byte) *module.MemorySegment {
	memoryOffset := c.Program.ComputeBlockStart64(c.MemoryOffset)
	var offsetExpr module.ExpressionRef
	if c.Options().IsWasm64() {
		offsetExpr = c.Module().I64(memoryOffset)
	} else {
		offsetExpr = c.Module().I32(int32(memoryOffset))
	}
	segment := &module.MemorySegment{
		Buffer:    buffer,
		Offset:    offsetExpr,
		RawOffset: memoryOffset,
	}
	c.MemorySegments = append(c.MemorySegments, segment)
	c.MemoryOffset = memoryOffset + int64(len(buffer))
	return segment
}

func ensureVisitMembersOf(c *Compiler, instance *program.Class) {
	if !instance.GetResolvedType().IsManaged() {
		panic("visitor requested for unmanaged class")
	}
	if instance.VisitRef != 0 {
		return
	}

	prog := c.Program
	mod := c.Module()
	usizeType := c.Options().UsizeType()
	sizeTypeRef := usizeType.ToRef()
	sizeTypeSize := uint32(usizeType.ByteSize())
	visitInstance := prog.VisitInstance()
	if visitInstance == nil {
		panic("missing __visit runtime function")
	}

	body := make([]module.ExpressionRef, 0)
	base := instance.Base
	if base != nil {
		body = append(body, mod.Call(base.GetInternalName()+"~visit", []module.ExpressionRef{
			mod.LocalGet(0, sizeTypeRef),
			mod.LocalGet(1, module.TypeRefI32),
		}, module.TypeRefNone))
	}

	hasVisitImpl := false
	if instance.IsDeclaredInLibrary() {
		if visitPrototype, ok := instance.GetMember(common.CommonNameVisit).(*program.FunctionPrototype); ok {
			visitMethod := c.Resolver().ResolveFunction(visitPrototype, nil, nil, program.ReportModeReport)
			if visitMethod == nil || !c.CompileFunction(visitMethod) {
				body = append(body, mod.Unreachable())
			} else {
				body = append(body, mod.Call(visitMethod.GetInternalName(), []module.ExpressionRef{
					mod.LocalGet(0, sizeTypeRef),
					mod.LocalGet(1, module.TypeRefI32),
				}, module.TypeRefNone))
			}
			hasVisitImpl = true
		}
	}

	needsTempValue := false
	if !hasVisitImpl {
		for _, member := range instance.OrderedOwnMembers() {
			propertyPrototype, ok := member.(*program.PropertyPrototype)
			if !ok {
				continue
			}
			property := propertyPrototype.PropertyInstance
			if property == nil || !property.IsField() || property.GetBoundClassOrInterface() != instance {
				continue
			}
			fieldType := property.GetResolvedType()
			if fieldType == nil || !fieldType.IsManaged() {
				continue
			}
			fieldOffset := property.MemoryOffset
			if fieldOffset < 0 {
				panic("managed field without offset")
			}
			needsTempValue = true
			body = append(body, mod.Call(visitInstance.GetInternalName(), []module.ExpressionRef{
				mod.Load(sizeTypeSize, false,
					mod.LocalGet(0, sizeTypeRef),
					sizeTypeRef,
					uint32(fieldOffset),
					sizeTypeSize,
					common.CommonNameDefaultMemory,
				),
				mod.LocalGet(1, module.TypeRefI32),
			}, module.TypeRefNone))
		}
	}

	varTypes := []module.TypeRef(nil)
	if needsTempValue {
		varTypes = []module.TypeRef{sizeTypeRef}
	}
	instance.VisitRef = mod.AddFunction(
		instance.GetInternalName()+"~visit",
		module.CreateType([]module.TypeRef{sizeTypeRef, module.TypeRefI32}),
		module.TypeRefNone,
		varTypes,
		mod.Flatten(body, module.TypeRefNone),
	)

	if base != nil && base.GetResolvedType().IsManaged() {
		ensureVisitMembersOf(c, base)
	}
}
