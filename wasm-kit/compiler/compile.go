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
	startFunctionBody := c.CurrentBody

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
	startIsEmpty := len(startFunctionBody) == 0
	exportStart := options.ExportStart
	hasExportStart := options.ExportStartSet
	if !startIsEmpty || hasExportStart {
		signature := startFunctionInstance.Signature
		if !startIsEmpty && hasExportStart {
			mod.AddGlobal(common.BuiltinNameStarted, module.TypeRefI32, true, mod.I32(0))
			// prepend: if (__started) return;
			startFunctionBody = append([]module.ExpressionRef{
				mod.If(
					mod.GlobalGet(common.BuiltinNameStarted, module.TypeRefI32),
					mod.Return(0),
					0,
				),
				mod.GlobalSet(common.BuiltinNameStarted, mod.I32(1)),
			}, startFunctionBody...)
		}
		funcRef := mod.AddFunction(
			startFunctionInstance.GetInternalName(),
			signature.ParamRefs(),
			signature.ResultRefs(),
			typesToRefs(startFunctionInstance.GetNonParameterLocalTypes()),
			mod.Flatten(startFunctionBody, module.TypeRefNone),
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
// Ported from: assemblyscript/src/compiler.ts compileModuleExports (lines 916-1076).
func (c *Compiler) CompileModuleExports(file *program.File) {
	mod := c.Module()
	exports := file.Exports
	if exports == nil {
		return
	}

	for exportName, element := range exports {
		// Skip already-exported elements
		if mod.HasExport(exportName) {
			continue
		}
		c.compileModuleExport(exportName, element)
	}

	// Handle re-exports (export * from)
	for _, reexportedFile := range file.ExportsStar {
		c.CompileModuleExports(reexportedFile)
	}
}

// compileModuleExport compiles a single module export.
func (c *Compiler) compileModuleExport(name string, element program.Element) {
	mod := c.Module()

	switch element.GetElementKind() {
	case program.ElementKindFunctionPrototype:
		// Resolve the function prototype to its default instance and compile it
		prototype := element.(*program.FunctionPrototype)
		instances := prototype.Instances
		if instances != nil {
			for _, instance := range instances {
				if c.CompileFunction(instance) {
					internalName := instance.GetInternalName()
					if !mod.HasExport(name) {
						mod.AddFunctionExport(internalName, name)
					}
				}
				break // Export the first (default) instance
			}
		} else {
			// Try to resolve the default (no type args) instance
			resolver := c.Resolver()
			instance := resolver.ResolveFunction(prototype, nil, nil, program.ReportModeReport)
			if instance != nil && c.CompileFunction(instance) {
				internalName := instance.GetInternalName()
				if !mod.HasExport(name) {
					mod.AddFunctionExport(internalName, name)
				}
			}
		}

	case program.ElementKindFunction:
		instance := element.(*program.Function)
		if c.CompileFunction(instance) {
			internalName := instance.GetInternalName()
			if !mod.HasExport(name) {
				mod.AddFunctionExport(internalName, name)
			}
		}

	case program.ElementKindGlobal:
		global := element.(*program.Global)
		if c.CompileGlobal(global) {
			internalName := global.GetInternalName()
			if !mod.HasExport(name) {
				mod.AddGlobalExport(internalName, name)
			}
		}

	case program.ElementKindEnum:
		enum := element.(*program.Enum)
		c.CompileEnum(enum)
		// Export enum values as individual globals
		members := enum.GetMembers()
		if members != nil {
			for memberName, member := range members {
				if member.GetElementKind() == program.ElementKindEnumValue {
					enumValue := member.(*program.EnumValue)
					if !enumValue.IsImmutable || !enumValue.Is(common.CommonFlagsConst) {
						memberExportName := name + "." + memberName
						memberInternalName := enumValue.GetInternalName()
						if !mod.HasExport(memberExportName) {
							mod.AddGlobalExport(memberInternalName, memberExportName)
						}
					}
				}
			}
		}

	case program.ElementKindClassPrototype:
		// Classes export their constructors and static members
		prototype := element.(*program.ClassPrototype)
		resolver := c.Resolver()
		instance := resolver.ResolveClass(prototype, nil, nil, program.ReportModeReport)
		if instance != nil {
			c.compileClassExports(name, instance)
		}

	case program.ElementKindClass:
		instance := element.(*program.Class)
		c.compileClassExports(name, instance)

	case program.ElementKindNamespace:
		// Export namespace members
		ns := element.(*program.Namespace)
		members := ns.GetMembers()
		if members != nil {
			for memberName, member := range members {
				if member.Is(common.CommonFlagsExport) || member.Is(common.CommonFlagsModuleExport) {
					exportName := name + "." + memberName
					if !mod.HasExport(exportName) {
						c.compileModuleExport(exportName, member)
					}
				}
			}
		}
	}
}

// compileClassExports compiles exports for a class instance (constructor + static members).
func (c *Compiler) compileClassExports(name string, instance *program.Class) {
	mod := c.Module()

	// Export the constructor if present
	ctorInstance := instance.ConstructorInstance
	if ctorInstance != nil && c.CompileFunction(ctorInstance) {
		ctorExportName := name + "#constructor"
		if !mod.HasExport(ctorExportName) {
			mod.AddFunctionExport(ctorInstance.GetInternalName(), ctorExportName)
		}
	}

	// Export static members
	members := instance.GetMembers()
	if members == nil {
		return
	}
	for memberName, member := range members {
		if !member.Is(common.CommonFlagsStatic) {
			continue
		}
		memberExportName := name + "." + memberName
		switch member.GetElementKind() {
		case program.ElementKindFunctionPrototype:
			prototype := member.(*program.FunctionPrototype)
			resolver := c.Resolver()
			fnInstance := resolver.ResolveFunction(prototype, nil, nil, program.ReportModeReport)
			if fnInstance != nil && c.CompileFunction(fnInstance) {
				if !mod.HasExport(memberExportName) {
					mod.AddFunctionExport(fnInstance.GetInternalName(), memberExportName)
				}
			}
		case program.ElementKindFunction:
			fnInstance := member.(*program.Function)
			if c.CompileFunction(fnInstance) {
				if !mod.HasExport(memberExportName) {
					mod.AddFunctionExport(fnInstance.GetInternalName(), memberExportName)
				}
			}
		case program.ElementKindGlobal:
			global := member.(*program.Global)
			if c.CompileGlobal(global) {
				if !mod.HasExport(memberExportName) {
					mod.AddGlobalExport(global.GetInternalName(), memberExportName)
				}
			}
		}
	}
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
// A varargs stub is called with omitted arguments being zeroed, reading the
// __argumentsLength global to decide which initializers to inject before
// calling the original function.
// Ported from: assemblyscript/src/compiler.ts ensureVarargsStub (lines 6528-6630).
func (c *Compiler) EnsureVarargsStub(original *program.Function) *program.Function {
	stub := original.VarargsStub
	if stub != nil {
		return stub
	}

	originalSignature := original.Signature
	originalParameterTypes := originalSignature.ParameterTypes
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
		return original
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

	// create the varargs stub function
	stub = original.NewStub("varargs", maxArguments)
	original.VarargsStub = stub

	// compile initializers of omitted arguments in the scope of the stub
	previousFlow := c.CurrentFlow
	fl := stub.Flow
	if original.Is(common.CommonFlagsConstructor) {
		fl.SetFlag(flow.FlowFlagCtorParamContext)
	}
	c.CurrentFlow = fl

	// Get parameter declarations for default initializers
	var parameterDeclarations []*ast.ParameterNode
	funcTypeNode := original.Prototype.FunctionTypeNode()
	if funcTypeNode != nil {
		parameterDeclarations = funcTypeNode.Parameters
	}

	// create a br_table switching over the number of optional parameters provided
	numNames := numOptional + 1 // incl. outer block
	names := make([]string, numNames)
	ofN := fmt.Sprintf("of%d", numOptional)
	for i := int32(0); i < numNames; i++ {
		names[i] = fmt.Sprintf("%d%s", i, ofN)
	}
	argumentsLength := c.EnsureArgumentsLength()

	// condition is number of provided optional arguments
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
		paramIdx := minArguments + i
		paramType := originalParameterTypes[paramIdx]
		var initExpr module.ExpressionRef

		if parameterDeclarations != nil && int(paramIdx) < len(parameterDeclarations) {
			declaration := parameterDeclarations[paramIdx]
			if declaration.ParameterKind == ast.ParameterKindRest {
				// Rest parameters get an empty array literal
				arrExpr := ast.NewArrayLiteralExpression(nil, *declaration.GetRange())
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
		} else {
			// No declaration available, use zero value
			initExpr = mod.LocalSet(operandIndex, c.makeZeroOfType(paramType), paramType.IsManaged())
		}

		table = mod.Block(names[i+1], []module.ExpressionRef{
			table,
			initExpr,
		}, module.TypeRefNone)

		// Extend forwardedOperands
		forwardedOperands = append(forwardedOperands, mod.LocalGet(operandIndex, paramType.ToRef()))
		operandIndex++
	}

	stmts = append(stmts, table)
	stmts = append(stmts, c.makeCallDirect(original, forwardedOperands, original.GetDeclaration()))

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

// makeCallDirect compiles a direct call to a function, ensuring it is compiled first.
// Ported from: assemblyscript/src/compiler.ts makeCallDirect (lines 6632-6768).
func (c *Compiler) makeCallDirect(instance *program.Function, operands []module.ExpressionRef, reportNode ast.Node) module.ExpressionRef {
	mod := c.Module()
	// Ensure the function is compiled
	c.CompileFunction(instance)
	// Just do a direct call for now. The full implementation handles
	// varargs forwarding, arguments length setting, and return type matching.
	numParams := int32(len(instance.Signature.ParameterTypes))
	if instance.Signature.ThisType != nil {
		numParams++
	}
	// If the target has optional params and we're calling with fewer operands,
	// route through the varargs stub instead
	if int32(len(operands)) < numParams {
		stub := c.EnsureVarargsStub(instance)
		if stub != instance {
			c.RuntimeFeatures |= RuntimeFeaturesSetArgumentsLength
			argumentsLength := c.EnsureArgumentsLength()
			return mod.Block("", []module.ExpressionRef{
				mod.GlobalSet(argumentsLength, mod.I32(int32(len(operands)))),
				mod.Call(stub.GetInternalName(), operands, instance.Signature.ReturnType.ToRef()),
			}, instance.Signature.ReturnType.ToRef())
		}
	}
	return mod.Call(instance.GetInternalName(), operands, instance.Signature.ReturnType.ToRef())
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
	sizeTypeRef := c.Options().UsizeType().ToRef()

	stmts := make([]module.ExpressionRef, 0)

	// Compute rtId offset: totalOverhead - OBJECT.offsetof("rtId")
	objectInstance := prog.ObjectInstance()
	rtIdOffset := prog.TotalOverhead() - int32(objectInstance.Offsetof("rtId"))

	// local.set $1, load(4, false, this - rtIdOffset, i32)
	var subExpr module.ExpressionRef
	if sizeTypeRef == module.TypeRefI64 {
		subExpr = mod.Binary(module.BinaryOpSubI64,
			mod.LocalGet(0, sizeTypeRef),
			mod.I64(int64(rtIdOffset)),
		)
	} else {
		subExpr = mod.Binary(module.BinaryOpSubI32,
			mod.LocalGet(0, sizeTypeRef),
			mod.I32(rtIdOffset),
		)
	}
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

		var subExpr module.ExpressionRef
		if sizeTypeRef == module.TypeRefI64 {
			subExpr = mod.Binary(module.BinaryOpSubI64,
				mod.LocalGet(0, sizeTypeRef),
				mod.I64(int64(rtIdOffset)),
			)
		} else {
			subExpr = mod.Binary(module.BinaryOpSubI32,
				mod.LocalGet(0, sizeTypeRef),
				mod.I32(rtIdOffset),
			)
		}
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

// compileRTTI compiles runtime type information.
func compileRTTI(c *Compiler) {
	// TODO: Implement RTTI compilation.
}

// compileVisitGlobals compiles the __visit_globals function.
func compileVisitGlobals(c *Compiler) {
	// TODO: Implement visitGlobals compilation.
}

// compileVisitMembers compiles the __visit_members function.
func compileVisitMembers(c *Compiler) {
	// TODO: Implement visitMembers compilation.
}

// compileCallExpressionBuiltin compiles a call to a builtin function.
// Ported from: assemblyscript/src/compiler.ts compileCallExpressionBuiltin (lines 6215-6252).
// TODO: Port builtins.ts (11,394 lines) for full implementation.
func (c *Compiler) compileCallExpressionBuiltin(
	prototype *program.FunctionPrototype,
	expression *ast.CallExpression,
	contextualType *types.Type,
) module.ExpressionRef {
	// TODO: Implement builtin function compilation.
	// This requires porting builtins.ts which handles all built-in types and functions.
	c.Error(
		diagnostics.DiagnosticCodeNotImplemented0,
		expression.GetRange(),
		"Builtin function calls", "", "",
	)
	return c.Module().Unreachable()
}

// checkFieldInitialization checks that fields are properly initialized.
// Ported from: assemblyscript/src/compiler.ts checkFieldInitialization (lines 6932-6985).
func (c *Compiler) checkFieldInitialization(classInstance *program.Class, reportNode ast.Node) {
	// TODO: Port field initialization check.
	// This checks that all fields in the class are initialized by the constructor.
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
// TODO: Full implementation needs __visit_globals / __rtti infrastructure.
func (c *Compiler) makeRuntimeNonNullCheck(expr module.ExpressionRef, typ *types.Type, reportNode ast.Node) module.ExpressionRef {
	// Stub: return expr as-is. Runtime assertions not yet implemented.
	return expr
}

// makeRuntimeDowncastCheck inserts a runtime downcast type check.
// Ported from: assemblyscript/src/compiler.ts makeRuntimeDowncastCheck.
// TODO: Full implementation needs RTTI infrastructure.
func (c *Compiler) makeRuntimeDowncastCheck(expr module.ExpressionRef, fromType, toType *types.Type, reportNode ast.Node) module.ExpressionRef {
	// Stub: return expr as-is. Runtime type checks not yet implemented.
	return expr
}

// --- Utility functions ---

// i64Align aligns a value to the given alignment.
func i64Align(value, alignment int64) int64 {
	mask := alignment - 1
	return (value + mask) & ^mask
}
