// Ported from: assemblyscript/src/compiler.ts compile(), initDefaultMemory(), initDefaultTable()
// (lines 530-914)
package compiler

import (
	"fmt"

	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/program"
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
	// TODO: Implement module exports compilation.
}

// CompileGlobal is now in compile_global.go

// EnsureOverrideStub creates an override stub for the given function.
// Ported from: assemblyscript/src/compiler.ts ensureOverrideStub (lines 6768-6880).
func (c *Compiler) EnsureOverrideStub(instance *program.Function) *program.Function {
	// TODO: Implement override stub creation.
	return instance
}

// EnsureVarargsStub creates a varargs stub for the given function.
// Ported from: assemblyscript/src/compiler.ts ensureVarargsStub (lines 6528-6630).
func (c *Compiler) EnsureVarargsStub(instance *program.Function) *program.Function {
	// TODO: Implement varargs stub creation.
	return instance
}

// FinalizeOverrideStub finalizes an override stub.
// Ported from: assemblyscript/src/compiler.ts finalizeOverrideStub (lines 6882-6962).
func (c *Compiler) FinalizeOverrideStub(instance *program.Function) {
	// TODO: Implement override stub finalization.
}

// FinalizeInstanceOf compiles an instanceof helper for a class.
// Ported from: assemblyscript/src/compiler.ts finalizeInstanceOf (lines 10082-10128).
func (c *Compiler) FinalizeInstanceOf(classInstance *program.Class, name string) {
	// TODO: Implement instanceof helper compilation.
}

// FinalizeAnyInstanceOf compiles a generic instanceof helper for a class prototype.
// Ported from: assemblyscript/src/compiler.ts finalizeAnyInstanceOf (lines 10130-10200).
func (c *Compiler) FinalizeAnyInstanceOf(prototype *program.ClassPrototype, name string) {
	// TODO: Implement any-instanceof helper compilation.
}

// EnsureArgumentsLength ensures the __argumentsLength global exists and returns its name.
// Ported from: assemblyscript/src/compiler.ts ensureArgumentsLength.
func (c *Compiler) EnsureArgumentsLength() string {
	// TODO: Implement arguments length global creation.
	return common.BuiltinNameArgumentsLength
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

// --- Utility functions ---

// i64Align aligns a value to the given alignment.
func i64Align(value, alignment int64) int64 {
	mask := alignment - 1
	return (value + mask) & ^mask
}
