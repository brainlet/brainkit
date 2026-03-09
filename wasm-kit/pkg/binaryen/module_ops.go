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
