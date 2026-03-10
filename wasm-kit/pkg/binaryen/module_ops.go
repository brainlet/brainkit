// Ported from: src/glue/binaryen.d.ts (module operations section)
package binaryen

/*
#include "binaryen-c.h"
#include <stdlib.h>
*/
import "C"
import "unsafe"

// ---------------------------------------------------------------------------
// Module validation, optimization, and output
// ---------------------------------------------------------------------------

// Validate checks whether the module is valid Wasm.
func (m *Module) Validate() bool {
	return goBool(C.BinaryenModuleValidate(m.ref))
}

// Optimize runs the standard Binaryen optimization passes on the module.
func (m *Module) Optimize() {
	C.BinaryenModuleOptimize(m.ref)
}

// RunPasses runs specific named optimization passes.
func (m *Module) RunPasses(passes []string) {
	if len(passes) == 0 {
		return
	}
	cPasses := make([]*C.char, len(passes))
	for i, p := range passes {
		cPasses[i] = C.CString(p)
	}
	C.BinaryenModuleRunPasses(m.ref, &cPasses[0], C.BinaryenIndex(len(passes)))
	for _, cp := range cPasses {
		C.free(unsafe.Pointer(cp))
	}
}

// Print prints the module to stdout in WAT format.
func (m *Module) Print() {
	C.BinaryenModulePrint(m.ref)
}

// PrintStackIR prints the module to stdout in stack IR format.
func (m *Module) PrintStackIR() {
	C.BinaryenModulePrintStackIR(m.ref)
}

// EmitBinary emits the module as a Wasm binary and returns the bytes.
func (m *Module) EmitBinary() []byte {
	var result C.BinaryenModuleAllocateAndWriteResult
	result = C.BinaryenModuleAllocateAndWrite(m.ref, nil)
	defer C.free(result.binary)
	return C.GoBytes(result.binary, C.int(result.binaryBytes))
}

// EmitText emits the module as WAT text and returns the string.
func (m *Module) EmitText() string {
	cs := C.BinaryenModuleAllocateAndWriteText(m.ref)
	defer C.free(unsafe.Pointer(cs))
	return C.GoString(cs)
}

// EmitStackIR emits the module as stack IR text.
func (m *Module) EmitStackIR() string {
	cs := C.BinaryenModuleAllocateAndWriteStackIR(m.ref)
	if cs == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(cs))
	return C.GoString(cs)
}

// ---------------------------------------------------------------------------
// Module feature flags
// ---------------------------------------------------------------------------

// SetFeatures sets the enabled Wasm features for this module.
func (m *Module) SetFeatures(features Features) {
	C.BinaryenModuleSetFeatures(m.ref, C.BinaryenFeatures(features))
}

// GetFeatures returns the enabled Wasm features.
func (m *Module) GetFeatures() Features {
	return Features(C.BinaryenModuleGetFeatures(m.ref))
}

// ---------------------------------------------------------------------------
// Module memory
// ---------------------------------------------------------------------------

// SetMemory configures the module's memory.
func (m *Module) SetMemory(initial, maximum Index, exportName string, shared bool) {
	var cExport *C.char
	if exportName != "" {
		cExport = m.str(exportName)
	}
	C.BinaryenSetMemory(m.ref, C.BinaryenIndex(initial), C.BinaryenIndex(maximum),
		cExport, nil, nil, nil, nil, nil, 0, cBool(shared), cBool(false), m.str("0"))
}

// HasMemory returns whether the module has a memory defined.
func (m *Module) HasMemory() bool {
	return goBool(C.BinaryenHasMemory(m.ref))
}

// GetNumMemorySegments returns the number of memory segments.
func (m *Module) GetNumMemorySegments() Index {
	return Index(C.BinaryenGetNumMemorySegments(m.ref))
}

// ---------------------------------------------------------------------------
// Module data segments
// ---------------------------------------------------------------------------

// AddDataSegment adds a passive data segment.
func (m *Module) AddDataSegment(name string, data []byte) {
	var dataPtr *C.char
	if len(data) > 0 {
		dataPtr = (*C.char)(unsafe.Pointer(&data[0]))
	}
	C.BinaryenAddDataSegment(m.ref, m.str(name), m.str("0"), cBool(true),
		nil, dataPtr, C.BinaryenIndex(len(data)))
}

// ---------------------------------------------------------------------------
// Pass configuration
// ---------------------------------------------------------------------------

// SetOptimizeLevel sets the optimization level (0-4).
func SetOptimizeLevel(level int) {
	C.BinaryenSetOptimizeLevel(C.int(level))
}

// SetShrinkLevel sets the shrink level (0-2).
func SetShrinkLevel(level int) {
	C.BinaryenSetShrinkLevel(C.int(level))
}

// GetOptimizeLevel returns the current optimization level.
func GetOptimizeLevel() int {
	return int(C.BinaryenGetOptimizeLevel())
}

// GetShrinkLevel returns the current shrink level.
func GetShrinkLevel() int {
	return int(C.BinaryenGetShrinkLevel())
}

// SetDebugInfo enables or disables debug info generation.
func SetDebugInfo(on bool) {
	C.BinaryenSetDebugInfo(cBool(on))
}

// SetLowMemoryUnused marks low memory as unused for optimization.
func SetLowMemoryUnused(on bool) {
	C.BinaryenSetLowMemoryUnused(cBool(on))
}

// GetLowMemoryUnused returns whether low memory is marked as unused.
func GetLowMemoryUnused() bool {
	return goBool(C.BinaryenGetLowMemoryUnused())
}

// SetZeroFilledMemory marks memory as zero-filled.
func SetZeroFilledMemory(on bool) {
	C.BinaryenSetZeroFilledMemory(cBool(on))
}

// SetFastMath enables or disables fast math optimizations.
func SetFastMath(on bool) {
	C.BinaryenSetFastMath(cBool(on))
}

// AddPassToSkip registers a pass name to be skipped by the optimizer.
func AddPassToSkip(pass string) {
	cs := C.CString(pass)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenAddPassToSkip(cs)
}

// ClearPassesToSkip clears the list of passes to skip.
func ClearPassesToSkip() {
	C.BinaryenClearPassesToSkip()
}

// ---------------------------------------------------------------------------
// Side effects
// ---------------------------------------------------------------------------

// GetSideEffects returns the side effects of an expression.
func GetSideEffects(expr ExpressionRef, mod *Module) SideEffects {
	return SideEffects(C.BinaryenExpressionGetSideEffects(
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)), mod.ref))
}

// ---------------------------------------------------------------------------
// Start function
// ---------------------------------------------------------------------------

// SetStart sets the start function for the module.
func (m *Module) SetStart(fn FunctionRef) {
	C.BinaryenSetStart(m.ref, (C.BinaryenFunctionRef)(unsafe.Pointer(fn)))
}

// ---------------------------------------------------------------------------
// Debug info
// ---------------------------------------------------------------------------

// AddDebugInfoFileName adds a debug info file name and returns its index.
func (m *Module) AddDebugInfoFileName(filename string) Index {
	cs := C.CString(filename)
	defer C.free(unsafe.Pointer(cs))
	return Index(C.BinaryenModuleAddDebugInfoFileName(m.ref, cs))
}

// GetDebugInfoFileName returns the debug info file name at the given index.
func (m *Module) GetDebugInfoFileName(index Index) string {
	cs := C.BinaryenModuleGetDebugInfoFileName(m.ref, C.BinaryenIndex(index))
	if cs == nil {
		return ""
	}
	return C.GoString(cs)
}

// SetDebugLocation sets a debug location for an expression within a function.
func (m *Module) SetDebugLocation(fn FunctionRef, expr ExpressionRef, fileIndex, line, col Index) {
	C.BinaryenFunctionSetDebugLocation(
		(C.BinaryenFunctionRef)(unsafe.Pointer(fn)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
		C.BinaryenIndex(fileIndex),
		C.BinaryenIndex(line),
		C.BinaryenIndex(col),
	)
}

// ---------------------------------------------------------------------------
// Memory (full API with segments and offsets)
// ---------------------------------------------------------------------------

// DataSegment describes a data segment for SetMemoryFull.
type DataSegment struct {
	Name     string
	Data     []byte
	Passive  bool
	Offset   ExpressionRef // offset expression, 0 for passive segments
}

// SetMemoryFull configures the module's memory with data segments and offset expressions.
func (m *Module) SetMemoryFull(initial, maximum Index, exportName string, segments []DataSegment, shared bool, is64 bool, memoryName string) {
	var cExport *C.char
	if exportName != "" {
		cExport = m.str(exportName)
	}
	n := len(segments)
	if n == 0 {
		C.BinaryenSetMemory(m.ref, C.BinaryenIndex(initial), C.BinaryenIndex(maximum),
			cExport, nil, nil, nil, nil, nil, 0, cBool(shared), cBool(is64), m.str(memoryName))
		return
	}
	cNames := make([]*C.char, n)
	cData := make([]*C.char, n)
	cPassive := make([]C.bool, n)
	cOffsets := make([]C.BinaryenExpressionRef, n)
	cSizes := make([]C.BinaryenIndex, n)
	for i, seg := range segments {
		cNames[i] = m.str(seg.Name)
		if len(seg.Data) > 0 {
			cData[i] = (*C.char)(unsafe.Pointer(&seg.Data[0]))
		}
		cPassive[i] = cBool(seg.Passive)
		cOffsets[i] = (C.BinaryenExpressionRef)(unsafe.Pointer(seg.Offset))
		cSizes[i] = C.BinaryenIndex(len(seg.Data))
	}
	C.BinaryenSetMemory(m.ref, C.BinaryenIndex(initial), C.BinaryenIndex(maximum),
		cExport, &cNames[0], &cData[0], &cPassive[0], &cOffsets[0], &cSizes[0],
		C.BinaryenIndex(n), cBool(shared), cBool(is64), m.str(memoryName))
}

// ---------------------------------------------------------------------------
// Binary output with source map
// ---------------------------------------------------------------------------

// EmitBinaryWithSourceMap emits the module as a Wasm binary with an optional source map.
func (m *Module) EmitBinaryWithSourceMap(sourceMapURL string) (binary []byte, sourceMap string) {
	var cURL *C.char
	if sourceMapURL != "" {
		cURL = C.CString(sourceMapURL)
		defer C.free(unsafe.Pointer(cURL))
	}
	result := C.BinaryenModuleAllocateAndWrite(m.ref, cURL)
	defer C.free(result.binary)
	binary = C.GoBytes(result.binary, C.int(result.binaryBytes))
	if result.sourceMap != nil {
		defer C.free(unsafe.Pointer(result.sourceMap))
		sourceMap = C.GoString(result.sourceMap)
	}
	return
}

// ---------------------------------------------------------------------------
// Module queries
// ---------------------------------------------------------------------------

// HasExport returns true if an export with the given name exists.
func HasExport(mod *Module, name string) bool {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	ref := C.BinaryenGetExport(mod.ref, cs)
	return ref != nil
}

// HasFunction returns true if a function with the given name exists.
func HasFunction(mod *Module, name string) bool {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	ref := C.BinaryenGetFunction(mod.ref, cs)
	return ref != nil
}

// ---------------------------------------------------------------------------
// Pass arguments
// ---------------------------------------------------------------------------

// SetPassArgument sets a string argument for an optimization pass.
func SetPassArgument(name, value string) {
	cn := C.CString(name)
	defer C.free(unsafe.Pointer(cn))
	cv := C.CString(value)
	defer C.free(unsafe.Pointer(cv))
	C.BinaryenSetPassArgument(cn, cv)
}

// ClearPassArguments clears all pass arguments.
func ClearPassArguments() {
	C.BinaryenClearPassArguments()
}

// ---------------------------------------------------------------------------
// Inline size configuration
// ---------------------------------------------------------------------------

// SetAlwaysInlineMaxSize sets the max function size to always inline.
func SetAlwaysInlineMaxSize(size Index) {
	C.BinaryenSetAlwaysInlineMaxSize(C.BinaryenIndex(size))
}

// SetFlexibleInlineMaxSize sets the max function size for flexible inlining.
func SetFlexibleInlineMaxSize(size Index) {
	C.BinaryenSetFlexibleInlineMaxSize(C.BinaryenIndex(size))
}

// SetOneCallerInlineMaxSize sets the max function size to inline when it has one caller.
func SetOneCallerInlineMaxSize(size Index) {
	C.BinaryenSetOneCallerInlineMaxSize(C.BinaryenIndex(size))
}

// ---------------------------------------------------------------------------
// Interpret
// ---------------------------------------------------------------------------

// Interpret calls the Binaryen interpreter on the module.
func (m *Module) Interpret() {
	C.BinaryenModuleInterpret(m.ref)
}

// ---------------------------------------------------------------------------
// Module closed-world and stack IR configuration
// ---------------------------------------------------------------------------

// SetClosedWorld marks the module as closed-world for optimizations.
func SetClosedWorld(on bool) {
	C.BinaryenSetClosedWorld(cBool(on))
}

// SetGenerateStackIR controls StackIR generation during optimization.
func SetGenerateStackIR(on bool) {
	C.BinaryenSetGenerateStackIR(cBool(on))
}

// SetOptimizeStackIR controls StackIR optimization.
func SetOptimizeStackIR(on bool) {
	C.BinaryenSetOptimizeStackIR(cBool(on))
}

// SetAllowInliningFunctionsWithLoops enables or disables inlining of functions
// containing loops.
func SetAllowInliningFunctionsWithLoops(enabled bool) {
	C.BinaryenSetAllowInliningFunctionsWithLoops(cBool(enabled))
}

// FunctionRunPasses runs specific named optimization passes on a single function.
func FunctionRunPasses(fn FunctionRef, mod *Module, passes []string) {
	if len(passes) == 0 {
		return
	}
	cPasses := make([]*C.char, len(passes))
	for i, p := range passes {
		cPasses[i] = C.CString(p)
	}
	C.BinaryenFunctionRunPasses(
		(C.BinaryenFunctionRef)(unsafe.Pointer(fn)),
		mod.ref,
		&cPasses[0],
		C.BinaryenIndex(len(passes)),
	)
	for _, cp := range cPasses {
		C.free(unsafe.Pointer(cp))
	}
}

// ---------------------------------------------------------------------------
// Expression runner
// ---------------------------------------------------------------------------

// ExpressionRunnerCreate creates an expression runner (const evaluator).
func ExpressionRunnerCreate(mod *Module, flags ExpressionRunnerFlags, maxDepth, maxLoopIterations Index) uintptr {
	return uintptr(unsafe.Pointer(C.ExpressionRunnerCreate(
		mod.ref,
		C.ExpressionRunnerFlags(flags),
		C.BinaryenIndex(maxDepth),
		C.BinaryenIndex(maxLoopIterations),
	)))
}

// ExpressionRunnerSetLocalValue sets a local value in the expression runner.
func ExpressionRunnerSetLocalValue(runner uintptr, index Index, value ExpressionRef) bool {
	return goBool(C.ExpressionRunnerSetLocalValue(
		(C.ExpressionRunnerRef)(unsafe.Pointer(runner)),
		C.BinaryenIndex(index),
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	))
}

// ExpressionRunnerSetGlobalValue sets a global value in the expression runner.
func ExpressionRunnerSetGlobalValue(runner uintptr, name string, value ExpressionRef) bool {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	return goBool(C.ExpressionRunnerSetGlobalValue(
		(C.ExpressionRunnerRef)(unsafe.Pointer(runner)),
		cs,
		(C.BinaryenExpressionRef)(unsafe.Pointer(value)),
	))
}

// ExpressionRunnerRunAndDispose runs the expression runner and disposes it.
func ExpressionRunnerRunAndDispose(runner uintptr, expr ExpressionRef) ExpressionRef {
	return ExpressionRef(unsafe.Pointer(C.ExpressionRunnerRunAndDispose(
		(C.ExpressionRunnerRef)(unsafe.Pointer(runner)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(expr)),
	)))
}

// ---------------------------------------------------------------------------
// Module read from binary
// ---------------------------------------------------------------------------

// ModuleRead creates a module from a Wasm binary buffer.
func ModuleRead(data []byte) *Module {
	if len(data) == 0 {
		return nil
	}
	ref := C.BinaryenModuleRead((*C.char)(unsafe.Pointer(&data[0])), C.size_t(len(data)))
	if ref == nil {
		return nil
	}
	return &Module{ref: ref}
}
