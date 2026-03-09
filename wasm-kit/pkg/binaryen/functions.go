// Ported from: binaryen-c.h (functions, globals, tables, tags, exports, imports,
// element segments, and expression utilities)
package binaryen

/*
#include "binaryen-c.h"
#include <stdlib.h>
*/
import "C"
import "unsafe"

// ---------------------------------------------------------------------------
// Functions
// ---------------------------------------------------------------------------

// AddFunction adds a function to the module.
// varTypes are the types of function-local variables (not params).
func (m *Module) AddFunction(name string, params, results Type, varTypes []Type, body ExpressionRef) FunctionRef {
	var vt *C.BinaryenType
	if len(varTypes) > 0 {
		vt = (*C.BinaryenType)(unsafe.Pointer(&varTypes[0]))
	}
	ref := C.BinaryenAddFunction(
		m.ref,
		m.str(name),
		C.BinaryenType(params),
		C.BinaryenType(results),
		vt,
		C.BinaryenIndex(len(varTypes)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(body)),
	)
	return FunctionRef(unsafe.Pointer(ref))
}

// AddFunctionWithHeapType adds a function using an explicit HeapType for its
// signature instead of separate params/results types.
func (m *Module) AddFunctionWithHeapType(name string, ht HeapType, varTypes []Type, body ExpressionRef) FunctionRef {
	var vt *C.BinaryenType
	if len(varTypes) > 0 {
		vt = (*C.BinaryenType)(unsafe.Pointer(&varTypes[0]))
	}
	ref := C.BinaryenAddFunctionWithHeapType(
		m.ref,
		m.str(name),
		C.BinaryenHeapType(ht),
		vt,
		C.BinaryenIndex(len(varTypes)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(body)),
	)
	return FunctionRef(unsafe.Pointer(ref))
}

// GetFunction returns a function reference by name, or 0 if not found.
func (m *Module) GetFunction(name string) FunctionRef {
	ref := C.BinaryenGetFunction(m.ref, m.str(name))
	return FunctionRef(unsafe.Pointer(ref))
}

// RemoveFunction removes a function by name.
func (m *Module) RemoveFunction(name string) {
	C.BinaryenRemoveFunction(m.ref, m.str(name))
}

// GetNumFunctions returns the number of functions in the module.
func (m *Module) GetNumFunctions() Index {
	return Index(C.BinaryenGetNumFunctions(m.ref))
}

// GetFunctionByIndex returns the function at the specified index.
func (m *Module) GetFunctionByIndex(index Index) FunctionRef {
	ref := C.BinaryenGetFunctionByIndex(m.ref, C.BinaryenIndex(index))
	return FunctionRef(unsafe.Pointer(ref))
}

// ---------------------------------------------------------------------------
// Function property accessors (operate on FunctionRef, not Module)
// ---------------------------------------------------------------------------

// FunctionGetName returns the name of the function.
func FunctionGetName(fn FunctionRef) string {
	return goString(C.BinaryenFunctionGetName((C.BinaryenFunctionRef)(unsafe.Pointer(fn))))
}

// FunctionGetParams returns the params type of the function.
func FunctionGetParams(fn FunctionRef) Type {
	return Type(C.BinaryenFunctionGetParams((C.BinaryenFunctionRef)(unsafe.Pointer(fn))))
}

// FunctionGetResults returns the results type of the function.
func FunctionGetResults(fn FunctionRef) Type {
	return Type(C.BinaryenFunctionGetResults((C.BinaryenFunctionRef)(unsafe.Pointer(fn))))
}

// FunctionGetNumVars returns the number of additional local variables
// (not including params).
func FunctionGetNumVars(fn FunctionRef) Index {
	return Index(C.BinaryenFunctionGetNumVars((C.BinaryenFunctionRef)(unsafe.Pointer(fn))))
}

// FunctionGetVar returns the type of the additional local variable at the
// given index.
func FunctionGetVar(fn FunctionRef, index Index) Type {
	return Type(C.BinaryenFunctionGetVar(
		(C.BinaryenFunctionRef)(unsafe.Pointer(fn)),
		C.BinaryenIndex(index),
	))
}

// FunctionGetNumLocals returns the total number of locals (params + vars).
func FunctionGetNumLocals(fn FunctionRef) Index {
	return Index(C.BinaryenFunctionGetNumLocals((C.BinaryenFunctionRef)(unsafe.Pointer(fn))))
}

// FunctionHasLocalName returns whether the local at the given index has a name.
func FunctionHasLocalName(fn FunctionRef, index Index) bool {
	return goBool(C.BinaryenFunctionHasLocalName(
		(C.BinaryenFunctionRef)(unsafe.Pointer(fn)),
		C.BinaryenIndex(index),
	))
}

// FunctionGetLocalName returns the name of the local at the given index.
func FunctionGetLocalName(fn FunctionRef, index Index) string {
	return goString(C.BinaryenFunctionGetLocalName(
		(C.BinaryenFunctionRef)(unsafe.Pointer(fn)),
		C.BinaryenIndex(index),
	))
}

// FunctionSetLocalName sets the name of the local at the given index.
func FunctionSetLocalName(fn FunctionRef, index Index, name string) {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	C.BinaryenFunctionSetLocalName(
		(C.BinaryenFunctionRef)(unsafe.Pointer(fn)),
		C.BinaryenIndex(index),
		cs,
	)
}

// FunctionGetBody returns the body expression of the function.
func FunctionGetBody(fn FunctionRef) ExpressionRef {
	ref := C.BinaryenFunctionGetBody((C.BinaryenFunctionRef)(unsafe.Pointer(fn)))
	return ExpressionRef(unsafe.Pointer(ref))
}

// FunctionSetBody sets the body expression of the function.
func FunctionSetBody(fn FunctionRef, body ExpressionRef) {
	C.BinaryenFunctionSetBody(
		(C.BinaryenFunctionRef)(unsafe.Pointer(fn)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(body)),
	)
}

// ---------------------------------------------------------------------------
// Globals
// ---------------------------------------------------------------------------

// AddGlobal adds a global to the module.
func (m *Module) AddGlobal(name string, typ Type, mutable bool, init ExpressionRef) GlobalRef {
	ref := C.BinaryenAddGlobal(
		m.ref,
		m.str(name),
		C.BinaryenType(typ),
		cBool(mutable),
		(C.BinaryenExpressionRef)(unsafe.Pointer(init)),
	)
	return GlobalRef(unsafe.Pointer(ref))
}

// GetGlobal returns a global reference by name, or 0 if not found.
func (m *Module) GetGlobal(name string) GlobalRef {
	ref := C.BinaryenGetGlobal(m.ref, m.str(name))
	return GlobalRef(unsafe.Pointer(ref))
}

// RemoveGlobal removes a global by name.
func (m *Module) RemoveGlobal(name string) {
	C.BinaryenRemoveGlobal(m.ref, m.str(name))
}

// GetNumGlobals returns the number of globals in the module.
func (m *Module) GetNumGlobals() Index {
	return Index(C.BinaryenGetNumGlobals(m.ref))
}

// GetGlobalByIndex returns the global at the specified index.
func (m *Module) GetGlobalByIndex(index Index) GlobalRef {
	ref := C.BinaryenGetGlobalByIndex(m.ref, C.BinaryenIndex(index))
	return GlobalRef(unsafe.Pointer(ref))
}

// ---------------------------------------------------------------------------
// Global property accessors
// ---------------------------------------------------------------------------

// GlobalGetName returns the name of the global.
func GlobalGetName(g GlobalRef) string {
	return goString(C.BinaryenGlobalGetName((C.BinaryenGlobalRef)(unsafe.Pointer(g))))
}

// GlobalGetType returns the value type of the global.
func GlobalGetType(g GlobalRef) Type {
	return Type(C.BinaryenGlobalGetType((C.BinaryenGlobalRef)(unsafe.Pointer(g))))
}

// GlobalIsMutable returns whether the global is mutable.
func GlobalIsMutable(g GlobalRef) bool {
	return goBool(C.BinaryenGlobalIsMutable((C.BinaryenGlobalRef)(unsafe.Pointer(g))))
}

// GlobalGetInitExpr returns the initialization expression of the global.
func GlobalGetInitExpr(g GlobalRef) ExpressionRef {
	ref := C.BinaryenGlobalGetInitExpr((C.BinaryenGlobalRef)(unsafe.Pointer(g)))
	return ExpressionRef(unsafe.Pointer(ref))
}

// ---------------------------------------------------------------------------
// Tables
// ---------------------------------------------------------------------------

// AddTable adds a table to the module.
func (m *Module) AddTable(name string, initial, maximum Index, tableType Type) TableRef {
	ref := C.BinaryenAddTable(
		m.ref,
		m.str(name),
		C.BinaryenIndex(initial),
		C.BinaryenIndex(maximum),
		C.BinaryenType(tableType),
	)
	return TableRef(unsafe.Pointer(ref))
}

// GetTable returns a table reference by name, or 0 if not found.
func (m *Module) GetTable(name string) TableRef {
	ref := C.BinaryenGetTable(m.ref, m.str(name))
	return TableRef(unsafe.Pointer(ref))
}

// RemoveTable removes a table by name.
func (m *Module) RemoveTable(name string) {
	C.BinaryenRemoveTable(m.ref, m.str(name))
}

// GetNumTables returns the number of tables in the module.
func (m *Module) GetNumTables() Index {
	return Index(C.BinaryenGetNumTables(m.ref))
}

// GetTableByIndex returns the table at the specified index.
func (m *Module) GetTableByIndex(index Index) TableRef {
	ref := C.BinaryenGetTableByIndex(m.ref, C.BinaryenIndex(index))
	return TableRef(unsafe.Pointer(ref))
}

// ---------------------------------------------------------------------------
// Table property accessors
// ---------------------------------------------------------------------------

// TableGetName returns the name of the table.
func TableGetName(t TableRef) string {
	return goString(C.BinaryenTableGetName((C.BinaryenTableRef)(unsafe.Pointer(t))))
}

// TableGetInitial returns the initial size of the table.
func TableGetInitial(t TableRef) Index {
	return Index(C.BinaryenTableGetInitial((C.BinaryenTableRef)(unsafe.Pointer(t))))
}

// TableHasMax returns whether the table has a maximum size.
func TableHasMax(t TableRef) bool {
	return goBool(C.BinaryenTableHasMax((C.BinaryenTableRef)(unsafe.Pointer(t))))
}

// TableGetMax returns the maximum size of the table.
func TableGetMax(t TableRef) Index {
	return Index(C.BinaryenTableGetMax((C.BinaryenTableRef)(unsafe.Pointer(t))))
}

// TableGetType returns the element type of the table.
func TableGetType(t TableRef) Type {
	return Type(C.BinaryenTableGetType((C.BinaryenTableRef)(unsafe.Pointer(t))))
}

// ---------------------------------------------------------------------------
// Tags
// ---------------------------------------------------------------------------

// AddTag adds a tag to the module.
func (m *Module) AddTag(name string, params, results Type) TagRef {
	ref := C.BinaryenAddTag(
		m.ref,
		m.str(name),
		C.BinaryenType(params),
		C.BinaryenType(results),
	)
	return TagRef(unsafe.Pointer(ref))
}

// GetTag returns a tag reference by name, or 0 if not found.
func (m *Module) GetTag(name string) TagRef {
	ref := C.BinaryenGetTag(m.ref, m.str(name))
	return TagRef(unsafe.Pointer(ref))
}

// RemoveTag removes a tag by name.
func (m *Module) RemoveTag(name string) {
	C.BinaryenRemoveTag(m.ref, m.str(name))
}

// ---------------------------------------------------------------------------
// Tag property accessors
// ---------------------------------------------------------------------------

// TagGetName returns the name of the tag.
func TagGetName(t TagRef) string {
	return goString(C.BinaryenTagGetName((C.BinaryenTagRef)(unsafe.Pointer(t))))
}

// TagGetParams returns the params type of the tag.
func TagGetParams(t TagRef) Type {
	return Type(C.BinaryenTagGetParams((C.BinaryenTagRef)(unsafe.Pointer(t))))
}

// TagGetResults returns the results type of the tag.
func TagGetResults(t TagRef) Type {
	return Type(C.BinaryenTagGetResults((C.BinaryenTagRef)(unsafe.Pointer(t))))
}

// ---------------------------------------------------------------------------
// Exports
// ---------------------------------------------------------------------------

// AddFunctionExport adds a function export to the module.
func (m *Module) AddFunctionExport(internalName, externalName string) ExportRef {
	ref := C.BinaryenAddFunctionExport(m.ref, m.str(internalName), m.str(externalName))
	return ExportRef(unsafe.Pointer(ref))
}

// AddTableExport adds a table export to the module.
func (m *Module) AddTableExport(internalName, externalName string) ExportRef {
	ref := C.BinaryenAddTableExport(m.ref, m.str(internalName), m.str(externalName))
	return ExportRef(unsafe.Pointer(ref))
}

// AddMemoryExport adds a memory export to the module.
func (m *Module) AddMemoryExport(internalName, externalName string) ExportRef {
	ref := C.BinaryenAddMemoryExport(m.ref, m.str(internalName), m.str(externalName))
	return ExportRef(unsafe.Pointer(ref))
}

// AddGlobalExport adds a global export to the module.
func (m *Module) AddGlobalExport(internalName, externalName string) ExportRef {
	ref := C.BinaryenAddGlobalExport(m.ref, m.str(internalName), m.str(externalName))
	return ExportRef(unsafe.Pointer(ref))
}

// AddTagExport adds a tag export to the module.
func (m *Module) AddTagExport(internalName, externalName string) ExportRef {
	ref := C.BinaryenAddTagExport(m.ref, m.str(internalName), m.str(externalName))
	return ExportRef(unsafe.Pointer(ref))
}

// GetExport returns an export reference by external name, or 0 if not found.
func (m *Module) GetExport(externalName string) ExportRef {
	ref := C.BinaryenGetExport(m.ref, m.str(externalName))
	return ExportRef(unsafe.Pointer(ref))
}

// RemoveExport removes an export by external name.
func (m *Module) RemoveExport(externalName string) {
	C.BinaryenRemoveExport(m.ref, m.str(externalName))
}

// GetNumExports returns the number of exports in the module.
func (m *Module) GetNumExports() Index {
	return Index(C.BinaryenGetNumExports(m.ref))
}

// GetExportByIndex returns the export at the specified index.
func (m *Module) GetExportByIndex(index Index) ExportRef {
	ref := C.BinaryenGetExportByIndex(m.ref, C.BinaryenIndex(index))
	return ExportRef(unsafe.Pointer(ref))
}

// ---------------------------------------------------------------------------
// Export property accessors
// ---------------------------------------------------------------------------

// ExportGetKind returns the external kind of the export.
func ExportGetKind(e ExportRef) ExternalKind {
	return ExternalKind(C.BinaryenExportGetKind((C.BinaryenExportRef)(unsafe.Pointer(e))))
}

// ExportGetName returns the external name of the export.
func ExportGetName(e ExportRef) string {
	return goString(C.BinaryenExportGetName((C.BinaryenExportRef)(unsafe.Pointer(e))))
}

// ExportGetValue returns the internal name (value) of the export.
func ExportGetValue(e ExportRef) string {
	return goString(C.BinaryenExportGetValue((C.BinaryenExportRef)(unsafe.Pointer(e))))
}

// ---------------------------------------------------------------------------
// Imports
// ---------------------------------------------------------------------------

// AddFunctionImport adds (or marks) a function as an import.
func (m *Module) AddFunctionImport(internalName, externalModuleName, externalBaseName string, params, results Type) {
	C.BinaryenAddFunctionImport(
		m.ref,
		m.str(internalName),
		m.str(externalModuleName),
		m.str(externalBaseName),
		C.BinaryenType(params),
		C.BinaryenType(results),
	)
}

// AddTableImport adds (or marks) a table as an import.
func (m *Module) AddTableImport(internalName, externalModuleName, externalBaseName string) {
	C.BinaryenAddTableImport(
		m.ref,
		m.str(internalName),
		m.str(externalModuleName),
		m.str(externalBaseName),
	)
}

// AddMemoryImport adds (or marks) a memory as an import.
func (m *Module) AddMemoryImport(internalName, externalModuleName, externalBaseName string, shared bool) {
	C.BinaryenAddMemoryImport(
		m.ref,
		m.str(internalName),
		m.str(externalModuleName),
		m.str(externalBaseName),
		C.uint8_t(boolToU8(shared)),
	)
}

// AddGlobalImport adds (or marks) a global as an import.
func (m *Module) AddGlobalImport(internalName, externalModuleName, externalBaseName string, globalType Type, mutable bool) {
	C.BinaryenAddGlobalImport(
		m.ref,
		m.str(internalName),
		m.str(externalModuleName),
		m.str(externalBaseName),
		C.BinaryenType(globalType),
		cBool(mutable),
	)
}

// AddTagImport adds (or marks) a tag as an import.
func (m *Module) AddTagImport(internalName, externalModuleName, externalBaseName string, params, results Type) {
	C.BinaryenAddTagImport(
		m.ref,
		m.str(internalName),
		m.str(externalModuleName),
		m.str(externalBaseName),
		C.BinaryenType(params),
		C.BinaryenType(results),
	)
}

// ---------------------------------------------------------------------------
// Import property accessors
// ---------------------------------------------------------------------------

// FunctionImportGetModule returns the external module name of a function import.
func FunctionImportGetModule(fn FunctionRef) string {
	return goString(C.BinaryenFunctionImportGetModule((C.BinaryenFunctionRef)(unsafe.Pointer(fn))))
}

// FunctionImportGetBase returns the external base name of a function import.
func FunctionImportGetBase(fn FunctionRef) string {
	return goString(C.BinaryenFunctionImportGetBase((C.BinaryenFunctionRef)(unsafe.Pointer(fn))))
}

// TableImportGetModule returns the external module name of a table import.
func TableImportGetModule(t TableRef) string {
	return goString(C.BinaryenTableImportGetModule((C.BinaryenTableRef)(unsafe.Pointer(t))))
}

// TableImportGetBase returns the external base name of a table import.
func TableImportGetBase(t TableRef) string {
	return goString(C.BinaryenTableImportGetBase((C.BinaryenTableRef)(unsafe.Pointer(t))))
}

// GlobalImportGetModule returns the external module name of a global import.
func GlobalImportGetModule(g GlobalRef) string {
	return goString(C.BinaryenGlobalImportGetModule((C.BinaryenGlobalRef)(unsafe.Pointer(g))))
}

// GlobalImportGetBase returns the external base name of a global import.
func GlobalImportGetBase(g GlobalRef) string {
	return goString(C.BinaryenGlobalImportGetBase((C.BinaryenGlobalRef)(unsafe.Pointer(g))))
}

// TagImportGetModule returns the external module name of a tag import.
func TagImportGetModule(t TagRef) string {
	return goString(C.BinaryenTagImportGetModule((C.BinaryenTagRef)(unsafe.Pointer(t))))
}

// TagImportGetBase returns the external base name of a tag import.
func TagImportGetBase(t TagRef) string {
	return goString(C.BinaryenTagImportGetBase((C.BinaryenTagRef)(unsafe.Pointer(t))))
}

// ---------------------------------------------------------------------------
// Element segments
// ---------------------------------------------------------------------------

// AddActiveElementSegment adds an active element segment to the module.
// The segment is associated with the named table and starts at the given offset.
func (m *Module) AddActiveElementSegment(table, name string, funcNames []string, offset ExpressionRef) ElementSegmentRef {
	cNames := make([]*C.char, len(funcNames))
	for i, fn := range funcNames {
		cNames[i] = m.str(fn)
	}
	var cNamesPtr **C.char
	if len(cNames) > 0 {
		cNamesPtr = &cNames[0]
	}
	ref := C.BinaryenAddActiveElementSegment(
		m.ref,
		m.str(table),
		m.str(name),
		cNamesPtr,
		C.BinaryenIndex(len(funcNames)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(offset)),
	)
	return ElementSegmentRef(unsafe.Pointer(ref))
}

// AddPassiveElementSegment adds a passive element segment to the module.
func (m *Module) AddPassiveElementSegment(name string, funcNames []string) ElementSegmentRef {
	cNames := make([]*C.char, len(funcNames))
	for i, fn := range funcNames {
		cNames[i] = m.str(fn)
	}
	var cNamesPtr **C.char
	if len(cNames) > 0 {
		cNamesPtr = &cNames[0]
	}
	ref := C.BinaryenAddPassiveElementSegment(
		m.ref,
		m.str(name),
		cNamesPtr,
		C.BinaryenIndex(len(funcNames)),
	)
	return ElementSegmentRef(unsafe.Pointer(ref))
}

// RemoveElementSegment removes an element segment by name.
func (m *Module) RemoveElementSegment(name string) {
	C.BinaryenRemoveElementSegment(m.ref, m.str(name))
}

// GetNumElementSegments returns the number of element segments.
func (m *Module) GetNumElementSegments() Index {
	return Index(C.BinaryenGetNumElementSegments(m.ref))
}

// GetElementSegment returns an element segment by name, or 0 if not found.
func (m *Module) GetElementSegment(name string) ElementSegmentRef {
	ref := C.BinaryenGetElementSegment(m.ref, m.str(name))
	return ElementSegmentRef(unsafe.Pointer(ref))
}

// GetElementSegmentByIndex returns the element segment at the specified index.
func (m *Module) GetElementSegmentByIndex(index Index) ElementSegmentRef {
	ref := C.BinaryenGetElementSegmentByIndex(m.ref, C.BinaryenIndex(index))
	return ElementSegmentRef(unsafe.Pointer(ref))
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// boolToU8 converts a Go bool to uint8 (for C uint8_t parameters).
func boolToU8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}
