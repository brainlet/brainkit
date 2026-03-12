package asembed

/*
#cgo CFLAGS: -I/Users/davidroman/Documents/code/clones/binaryen/src
#cgo LDFLAGS: -L/Users/davidroman/Documents/code/clones/binaryen/build/lib -lbinaryen -lstdc++ -lm
#include "binaryen-c.h"
#include <stdlib.h>
#include <string.h>
*/
import "C"
import "unsafe"

// CGo wrappers for Binaryen C API functions.
// Each wrapper is a thin Go function that converts types and calls the C API.
// Used by binaryen_bridge.go to implement the JS-to-C bridge.

// --- Type functions ---

func cgoTypeNone() uintptr        { return uintptr(C.BinaryenTypeNone()) }
func cgoTypeInt32() uintptr       { return uintptr(C.BinaryenTypeInt32()) }
func cgoTypeInt64() uintptr       { return uintptr(C.BinaryenTypeInt64()) }
func cgoTypeFloat32() uintptr     { return uintptr(C.BinaryenTypeFloat32()) }
func cgoTypeFloat64() uintptr     { return uintptr(C.BinaryenTypeFloat64()) }
func cgoTypeVec128() uintptr      { return uintptr(C.BinaryenTypeVec128()) }
func cgoTypeUnreachable() uintptr { return uintptr(C.BinaryenTypeUnreachable()) }
func cgoTypeAuto() uintptr        { return uintptr(C.BinaryenTypeAuto()) }

func cgoTypeFuncref() uintptr     { return uintptr(C.BinaryenTypeFuncref()) }
func cgoTypeExternref() uintptr   { return uintptr(C.BinaryenTypeExternref()) }
func cgoTypeAnyref() uintptr      { return uintptr(C.BinaryenTypeAnyref()) }
func cgoTypeEqref() uintptr       { return uintptr(C.BinaryenTypeEqref()) }
func cgoTypeI31ref() uintptr      { return uintptr(C.BinaryenTypeI31ref()) }
func cgoTypeStructref() uintptr   { return uintptr(C.BinaryenTypeStructref()) }
func cgoTypeArrayref() uintptr    { return uintptr(C.BinaryenTypeArrayref()) }
func cgoTypeStringref() uintptr   { return uintptr(C.BinaryenTypeStringref()) }
func cgoTypeNullref() uintptr     { return uintptr(C.BinaryenTypeNullref()) }
func cgoTypeNullExternref() uintptr { return uintptr(C.BinaryenTypeNullExternref()) }
func cgoTypeNullFuncref() uintptr { return uintptr(C.BinaryenTypeNullFuncref()) }

func cgoTypeCreate(types []uintptr) uintptr {
	if len(types) == 0 {
		return cgoTypeNone()
	}
	cTypes := make([]C.BinaryenType, len(types))
	for i, t := range types {
		cTypes[i] = C.BinaryenType(t)
	}
	return uintptr(C.BinaryenTypeCreate(&cTypes[0], C.BinaryenIndex(len(types))))
}

func cgoTypeArity(t uintptr) int {
	return int(C.BinaryenTypeArity(C.BinaryenType(t)))
}

func cgoTypeExpand(t uintptr, buf []uintptr) {
	if len(buf) == 0 {
		return
	}
	C.BinaryenTypeExpand(C.BinaryenType(t), (*C.BinaryenType)(unsafe.Pointer(&buf[0])))
}

func cgoTypeGetHeapType(t uintptr) uintptr {
	return uintptr(C.BinaryenTypeGetHeapType(C.BinaryenType(t)))
}

func cgoTypeFromHeapType(ht uintptr, nullable bool) uintptr {
	return uintptr(C.BinaryenTypeFromHeapType(C.BinaryenHeapType(ht), C.bool(nullable)))
}

func cgoTypeIsNullable(t uintptr) bool {
	return bool(C.BinaryenTypeIsNullable(C.BinaryenType(t)))
}

// --- Heap type functions ---

func cgoHeapTypeFunc() uintptr   { return uintptr(C.BinaryenHeapTypeFunc()) }
func cgoHeapTypeExt() uintptr    { return uintptr(C.BinaryenHeapTypeExt()) }
func cgoHeapTypeAny() uintptr    { return uintptr(C.BinaryenHeapTypeAny()) }
func cgoHeapTypeEq() uintptr     { return uintptr(C.BinaryenHeapTypeEq()) }
func cgoHeapTypeI31() uintptr    { return uintptr(C.BinaryenHeapTypeI31()) }
func cgoHeapTypeStruct() uintptr { return uintptr(C.BinaryenHeapTypeStruct()) }
func cgoHeapTypeArray() uintptr  { return uintptr(C.BinaryenHeapTypeArray()) }
func cgoHeapTypeString() uintptr { return uintptr(C.BinaryenHeapTypeString()) }
func cgoHeapTypeNone() uintptr   { return uintptr(C.BinaryenHeapTypeNone()) }
func cgoHeapTypeNoext() uintptr  { return uintptr(C.BinaryenHeapTypeNoext()) }
func cgoHeapTypeNofunc() uintptr { return uintptr(C.BinaryenHeapTypeNofunc()) }

func cgoHeapTypeIsBasic(ht uintptr) bool {
	return bool(C.BinaryenHeapTypeIsBasic(C.BinaryenHeapType(ht)))
}

func cgoHeapTypeIsSignature(ht uintptr) bool {
	return bool(C.BinaryenHeapTypeIsSignature(C.BinaryenHeapType(ht)))
}

func cgoHeapTypeIsStruct(ht uintptr) bool {
	return bool(C.BinaryenHeapTypeIsStruct(C.BinaryenHeapType(ht)))
}

func cgoHeapTypeIsArray(ht uintptr) bool {
	return bool(C.BinaryenHeapTypeIsArray(C.BinaryenHeapType(ht)))
}

func cgoHeapTypeIsBottom(ht uintptr) bool {
	return bool(C.BinaryenHeapTypeIsBottom(C.BinaryenHeapType(ht)))
}

func cgoHeapTypeGetBottom(ht uintptr) uintptr {
	return uintptr(C.BinaryenHeapTypeGetBottom(C.BinaryenHeapType(ht)))
}

func cgoHeapTypeIsSubType(left, right uintptr) bool {
	return bool(C.BinaryenHeapTypeIsSubType(C.BinaryenHeapType(left), C.BinaryenHeapType(right)))
}

// --- Struct/Array/Signature type info ---

func cgoStructTypeGetNumFields(ht uintptr) int {
	return int(C.BinaryenStructTypeGetNumFields(C.BinaryenHeapType(ht)))
}

func cgoStructTypeGetFieldType(ht uintptr, index int) uintptr {
	return uintptr(C.BinaryenStructTypeGetFieldType(C.BinaryenHeapType(ht), C.BinaryenIndex(index)))
}

func cgoStructTypeGetFieldPackedType(ht uintptr, index int) uint32 {
	return uint32(C.BinaryenStructTypeGetFieldPackedType(C.BinaryenHeapType(ht), C.BinaryenIndex(index)))
}

func cgoStructTypeIsFieldMutable(ht uintptr, index int) bool {
	return bool(C.BinaryenStructTypeIsFieldMutable(C.BinaryenHeapType(ht), C.BinaryenIndex(index)))
}

func cgoArrayTypeGetElementType(ht uintptr) uintptr {
	return uintptr(C.BinaryenArrayTypeGetElementType(C.BinaryenHeapType(ht)))
}

func cgoArrayTypeGetElementPackedType(ht uintptr) uint32 {
	return uint32(C.BinaryenArrayTypeGetElementPackedType(C.BinaryenHeapType(ht)))
}

func cgoArrayTypeIsElementMutable(ht uintptr) bool {
	return bool(C.BinaryenArrayTypeIsElementMutable(C.BinaryenHeapType(ht)))
}

func cgoSignatureTypeGetParams(ht uintptr) uintptr {
	return uintptr(C.BinaryenSignatureTypeGetParams(C.BinaryenHeapType(ht)))
}

func cgoSignatureTypeGetResults(ht uintptr) uintptr {
	return uintptr(C.BinaryenSignatureTypeGetResults(C.BinaryenHeapType(ht)))
}

// --- Packed types ---

func cgoPackedTypeNotPacked() uint32 { return uint32(C.BinaryenPackedTypeNotPacked()) }
func cgoPackedTypeInt8() uint32      { return uint32(C.BinaryenPackedTypeInt8()) }
func cgoPackedTypeInt16() uint32     { return uint32(C.BinaryenPackedTypeInt16()) }

// --- Module lifecycle ---

func cgoModuleCreate() uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenModuleCreate()))
}

func cgoModuleDispose(module uintptr) {
	C.BinaryenModuleDispose(C.BinaryenModuleRef(unsafe.Pointer(module)))
}

func cgoModuleValidate(module uintptr) bool {
	return bool(C.BinaryenModuleValidate(C.BinaryenModuleRef(unsafe.Pointer(module))))
}

func cgoModuleOptimize(module uintptr) {
	C.BinaryenModuleOptimize(C.BinaryenModuleRef(unsafe.Pointer(module)))
}

func cgoModulePrint(module uintptr) {
	C.BinaryenModulePrint(C.BinaryenModuleRef(unsafe.Pointer(module)))
}

func cgoModulePrintAsmjs(module uintptr) {
	C.BinaryenModulePrintAsmjs(C.BinaryenModuleRef(unsafe.Pointer(module)))
}

func cgoModulePrintStackIR(module uintptr) {
	C.BinaryenModulePrintStackIR(C.BinaryenModuleRef(unsafe.Pointer(module)))
}

func cgoModuleGetFeatures(module uintptr) uint32 {
	return uint32(C.BinaryenModuleGetFeatures(C.BinaryenModuleRef(unsafe.Pointer(module))))
}

func cgoModuleSetFeatures(module uintptr, features uint32) {
	C.BinaryenModuleSetFeatures(C.BinaryenModuleRef(unsafe.Pointer(module)), C.BinaryenFeatures(features))
}

func cgoModuleRunPasses(module uintptr, passes []string) {
	if len(passes) == 0 {
		return
	}
	cPasses := make([]*C.char, len(passes))
	for i, p := range passes {
		cPasses[i] = C.CString(p)
	}
	C.BinaryenModuleRunPasses(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		&cPasses[0],
		C.BinaryenIndex(len(passes)),
	)
	for _, cp := range cPasses {
		C.free(unsafe.Pointer(cp))
	}
}

// cgoModuleAutoDrop is not available in this binaryen version.
// func cgoModuleAutoDrop(module uintptr) { ... }

// BinaryCGoResult holds the result of BinaryenModuleAllocateAndWrite.
type BinaryCGoResult struct {
	Binary    []byte
	SourceMap string
}

func cgoModuleAllocateAndWrite(module uintptr, sourceMapURL string) BinaryCGoResult {
	var cURL *C.char
	if sourceMapURL != "" {
		cURL = C.CString(sourceMapURL)
		defer C.free(unsafe.Pointer(cURL))
	}

	result := C.BinaryenModuleAllocateAndWrite(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		cURL,
	)

	binaryLen := int(result.binaryBytes)
	binary := C.GoBytes(result.binary, C.int(binaryLen))

	var sourceMap string
	if result.sourceMap != nil {
		sourceMap = C.GoString(result.sourceMap)
		C.free(unsafe.Pointer(result.sourceMap))
	}

	C.free(result.binary)

	return BinaryCGoResult{
		Binary:    binary,
		SourceMap: sourceMap,
	}
}

func cgoModuleAllocateAndWriteText(module uintptr) string {
	cStr := C.BinaryenModuleAllocateAndWriteText(C.BinaryenModuleRef(unsafe.Pointer(module)))
	if cStr == nil {
		return ""
	}
	s := C.GoString(cStr)
	C.free(unsafe.Pointer(cStr))
	return s
}

func cgoModuleAllocateAndWriteStackIR(module uintptr) string {
	cStr := C.BinaryenModuleAllocateAndWriteStackIR(C.BinaryenModuleRef(unsafe.Pointer(module)))
	if cStr == nil {
		return ""
	}
	s := C.GoString(cStr)
	C.free(unsafe.Pointer(cStr))
	return s
}

// --- Literal operations ---

func cgoSizeofLiteral() int {
	return int(C.sizeof_struct_BinaryenLiteral)
}

func cgoLiteralInt32(value int32, out []byte) {
	lit := C.BinaryenLiteralInt32(C.int32_t(value))
	copy(out, (*[256]byte)(unsafe.Pointer(&lit))[:len(out)])
}

func cgoLiteralInt64(lo, hi int32, out []byte) {
	val := int64(uint32(lo)) | (int64(hi) << 32)
	lit := C.BinaryenLiteralInt64(C.int64_t(val))
	copy(out, (*[256]byte)(unsafe.Pointer(&lit))[:len(out)])
}

func cgoLiteralFloat32(value float32, out []byte) {
	lit := C.BinaryenLiteralFloat32(C.float(value))
	copy(out, (*[256]byte)(unsafe.Pointer(&lit))[:len(out)])
}

func cgoLiteralFloat64(value float64, out []byte) {
	lit := C.BinaryenLiteralFloat64(C.double(value))
	copy(out, (*[256]byte)(unsafe.Pointer(&lit))[:len(out)])
}

func cgoLiteralFloat32Bits(value int32, out []byte) {
	lit := C.BinaryenLiteralFloat32Bits(C.int32_t(value))
	copy(out, (*[256]byte)(unsafe.Pointer(&lit))[:len(out)])
}

func cgoLiteralFloat64Bits(lo, hi int32, out []byte) {
	val := int64(uint32(lo)) | (int64(hi) << 32)
	lit := C.BinaryenLiteralFloat64Bits(C.int64_t(val))
	copy(out, (*[256]byte)(unsafe.Pointer(&lit))[:len(out)])
}

// --- Expression info ---

func cgoExpressionGetId(expr uintptr) int {
	return int(C.BinaryenExpressionGetId(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoExpressionGetType(expr uintptr) uintptr {
	return uintptr(C.BinaryenExpressionGetType(C.BinaryenExpressionRef(unsafe.Pointer(expr))))
}

func cgoExpressionSetType(expr uintptr, typ uintptr) {
	C.BinaryenExpressionSetType(C.BinaryenExpressionRef(unsafe.Pointer(expr)), C.BinaryenType(typ))
}

func cgoExpressionPrint(expr uintptr) {
	C.BinaryenExpressionPrint(C.BinaryenExpressionRef(unsafe.Pointer(expr)))
}

func cgoExpressionCopy(expr uintptr, module uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenExpressionCopy(
		C.BinaryenExpressionRef(unsafe.Pointer(expr)),
		C.BinaryenModuleRef(unsafe.Pointer(module)),
	)))
}

func cgoExpressionFinalize(expr uintptr) {
	C.BinaryenExpressionFinalize(C.BinaryenExpressionRef(unsafe.Pointer(expr)))
}

// --- Expression constructors ---

func cgoBlock(module uintptr, name unsafe.Pointer, children []uintptr, typ uintptr) uintptr {
	var cChildren *C.BinaryenExpressionRef
	if len(children) > 0 {
		cArr := make([]C.BinaryenExpressionRef, len(children))
		for i, c := range children {
			cArr[i] = C.BinaryenExpressionRef(unsafe.Pointer(c))
		}
		cChildren = &cArr[0]
	}
	ref := C.BinaryenBlock(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
		cChildren,
		C.BinaryenIndex(len(children)),
		C.BinaryenType(typ),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoIf(module uintptr, condition, ifTrue, ifFalse uintptr) uintptr {
	ref := C.BinaryenIf(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenExpressionRef(unsafe.Pointer(condition)),
		C.BinaryenExpressionRef(unsafe.Pointer(ifTrue)),
		C.BinaryenExpressionRef(unsafe.Pointer(ifFalse)),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoLoop(module uintptr, name unsafe.Pointer, body uintptr) uintptr {
	ref := C.BinaryenLoop(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
		C.BinaryenExpressionRef(unsafe.Pointer(body)),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoBreak(module uintptr, name unsafe.Pointer, condition, value uintptr) uintptr {
	ref := C.BinaryenBreak(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
		C.BinaryenExpressionRef(unsafe.Pointer(condition)),
		C.BinaryenExpressionRef(unsafe.Pointer(value)),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoSwitch(module uintptr, names []unsafe.Pointer, defaultName unsafe.Pointer, condition, value uintptr) uintptr {
	var cNames **C.char
	if len(names) > 0 {
		cArr := make([]*C.char, len(names))
		for i, n := range names {
			cArr[i] = (*C.char)(n)
		}
		cNames = &cArr[0]
	}
	ref := C.BinaryenSwitch(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		cNames,
		C.BinaryenIndex(len(names)),
		(*C.char)(defaultName),
		C.BinaryenExpressionRef(unsafe.Pointer(condition)),
		C.BinaryenExpressionRef(unsafe.Pointer(value)),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoCall(module uintptr, target unsafe.Pointer, operands []uintptr, returnType uintptr) uintptr {
	var cOperands *C.BinaryenExpressionRef
	if len(operands) > 0 {
		cArr := make([]C.BinaryenExpressionRef, len(operands))
		for i, o := range operands {
			cArr[i] = C.BinaryenExpressionRef(unsafe.Pointer(o))
		}
		cOperands = &cArr[0]
	}
	ref := C.BinaryenCall(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(target),
		cOperands,
		C.BinaryenIndex(len(operands)),
		C.BinaryenType(returnType),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoCallIndirect(module uintptr, table unsafe.Pointer, target uintptr, operands []uintptr, params, results uintptr) uintptr {
	var cOperands *C.BinaryenExpressionRef
	if len(operands) > 0 {
		cArr := make([]C.BinaryenExpressionRef, len(operands))
		for i, o := range operands {
			cArr[i] = C.BinaryenExpressionRef(unsafe.Pointer(o))
		}
		cOperands = &cArr[0]
	}
	ref := C.BinaryenCallIndirect(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(table),
		C.BinaryenExpressionRef(unsafe.Pointer(target)),
		cOperands,
		C.BinaryenIndex(len(operands)),
		C.BinaryenType(params),
		C.BinaryenType(results),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoLocalGet(module uintptr, index int, typ uintptr) uintptr {
	ref := C.BinaryenLocalGet(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenIndex(index),
		C.BinaryenType(typ),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoLocalSet(module uintptr, index int, value uintptr) uintptr {
	ref := C.BinaryenLocalSet(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenIndex(index),
		C.BinaryenExpressionRef(unsafe.Pointer(value)),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoLocalTee(module uintptr, index int, value uintptr, typ uintptr) uintptr {
	ref := C.BinaryenLocalTee(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenIndex(index),
		C.BinaryenExpressionRef(unsafe.Pointer(value)),
		C.BinaryenType(typ),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoGlobalGet(module uintptr, name unsafe.Pointer, typ uintptr) uintptr {
	ref := C.BinaryenGlobalGet(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
		C.BinaryenType(typ),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoGlobalSet(module uintptr, name unsafe.Pointer, value uintptr) uintptr {
	ref := C.BinaryenGlobalSet(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
		C.BinaryenExpressionRef(unsafe.Pointer(value)),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoLoad(module uintptr, bytes uint32, signed bool, offset, align uint32, typ uintptr, ptr uintptr, memoryName unsafe.Pointer) uintptr {
	ref := C.BinaryenLoad(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.uint32_t(bytes),
		C.bool(signed),
		C.uint32_t(offset),
		C.uint32_t(align),
		C.BinaryenType(typ),
		C.BinaryenExpressionRef(unsafe.Pointer(ptr)),
		(*C.char)(memoryName),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoStore(module uintptr, bytes, offset, align uint32, ptr, value uintptr, typ uintptr, memoryName unsafe.Pointer) uintptr {
	ref := C.BinaryenStore(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.uint32_t(bytes),
		C.uint32_t(offset),
		C.uint32_t(align),
		C.BinaryenExpressionRef(unsafe.Pointer(ptr)),
		C.BinaryenExpressionRef(unsafe.Pointer(value)),
		C.BinaryenType(typ),
		(*C.char)(memoryName),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoConst(module uintptr, literalPtr unsafe.Pointer) uintptr {
	ref := C.BinaryenConst(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		*(*C.struct_BinaryenLiteral)(literalPtr),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoUnary(module uintptr, op int32, value uintptr) uintptr {
	ref := C.BinaryenUnary(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenOp(op),
		C.BinaryenExpressionRef(unsafe.Pointer(value)),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoBinary(module uintptr, op int32, left, right uintptr) uintptr {
	ref := C.BinaryenBinary(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenOp(op),
		C.BinaryenExpressionRef(unsafe.Pointer(left)),
		C.BinaryenExpressionRef(unsafe.Pointer(right)),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoSelect(module uintptr, condition, ifTrue, ifFalse uintptr) uintptr {
	ref := C.BinaryenSelect(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenExpressionRef(unsafe.Pointer(condition)),
		C.BinaryenExpressionRef(unsafe.Pointer(ifTrue)),
		C.BinaryenExpressionRef(unsafe.Pointer(ifFalse)),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoDrop(module uintptr, value uintptr) uintptr {
	ref := C.BinaryenDrop(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenExpressionRef(unsafe.Pointer(value)),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoReturn(module uintptr, value uintptr) uintptr {
	ref := C.BinaryenReturn(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenExpressionRef(unsafe.Pointer(value)),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoNop(module uintptr) uintptr {
	ref := C.BinaryenNop(C.BinaryenModuleRef(unsafe.Pointer(module)))
	return uintptr(unsafe.Pointer(ref))
}

func cgoUnreachable(module uintptr) uintptr {
	ref := C.BinaryenUnreachable(C.BinaryenModuleRef(unsafe.Pointer(module)))
	return uintptr(unsafe.Pointer(ref))
}

func cgoMemorySize(module uintptr, memoryName unsafe.Pointer, memoryIs64 bool) uintptr {
	ref := C.BinaryenMemorySize(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(memoryName),
		C.bool(memoryIs64),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoMemoryGrow(module uintptr, delta uintptr, memoryName unsafe.Pointer, memoryIs64 bool) uintptr {
	ref := C.BinaryenMemoryGrow(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenExpressionRef(unsafe.Pointer(delta)),
		(*C.char)(memoryName),
		C.bool(memoryIs64),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoTry(module uintptr, name unsafe.Pointer, body uintptr, catchTags []unsafe.Pointer, catchBodies []uintptr, delegateTarget unsafe.Pointer) uintptr {
	var cTags **C.char
	if len(catchTags) > 0 {
		cTagArr := make([]*C.char, len(catchTags))
		for i, t := range catchTags {
			cTagArr[i] = (*C.char)(t)
		}
		cTags = &cTagArr[0]
	}
	var cBodies *C.BinaryenExpressionRef
	if len(catchBodies) > 0 {
		cArr := make([]C.BinaryenExpressionRef, len(catchBodies))
		for i, b := range catchBodies {
			cArr[i] = C.BinaryenExpressionRef(unsafe.Pointer(b))
		}
		cBodies = &cArr[0]
	}
	ref := C.BinaryenTry(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
		C.BinaryenExpressionRef(unsafe.Pointer(body)),
		cTags,
		C.BinaryenIndex(len(catchTags)),
		cBodies,
		C.BinaryenIndex(len(catchBodies)),
		(*C.char)(delegateTarget),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoThrow(module uintptr, tag unsafe.Pointer, operands []uintptr) uintptr {
	var cOperands *C.BinaryenExpressionRef
	if len(operands) > 0 {
		cArr := make([]C.BinaryenExpressionRef, len(operands))
		for i, o := range operands {
			cArr[i] = C.BinaryenExpressionRef(unsafe.Pointer(o))
		}
		cOperands = &cArr[0]
	}
	ref := C.BinaryenThrow(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(tag),
		cOperands,
		C.BinaryenIndex(len(operands)),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoRethrow(module uintptr, target unsafe.Pointer) uintptr {
	ref := C.BinaryenRethrow(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(target),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoRefNull(module uintptr, typ uintptr) uintptr {
	ref := C.BinaryenRefNull(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenType(typ),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoRefIsNull(module uintptr, value uintptr) uintptr {
	ref := C.BinaryenRefIsNull(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenExpressionRef(unsafe.Pointer(value)),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoRefAs(module uintptr, op int32, value uintptr) uintptr {
	ref := C.BinaryenRefAs(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenOp(op),
		C.BinaryenExpressionRef(unsafe.Pointer(value)),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoRefFunc(module uintptr, funcName unsafe.Pointer, typ uintptr) uintptr {
	ref := C.BinaryenRefFunc(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(funcName),
		C.BinaryenType(typ),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoRefEq(module uintptr, left, right uintptr) uintptr {
	ref := C.BinaryenRefEq(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenExpressionRef(unsafe.Pointer(left)),
		C.BinaryenExpressionRef(unsafe.Pointer(right)),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoTableGet(module uintptr, name unsafe.Pointer, index uintptr, typ uintptr) uintptr {
	ref := C.BinaryenTableGet(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
		C.BinaryenExpressionRef(unsafe.Pointer(index)),
		C.BinaryenType(typ),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoTableSet(module uintptr, name unsafe.Pointer, index, value uintptr) uintptr {
	ref := C.BinaryenTableSet(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
		C.BinaryenExpressionRef(unsafe.Pointer(index)),
		C.BinaryenExpressionRef(unsafe.Pointer(value)),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoTableSize(module uintptr, name unsafe.Pointer) uintptr {
	ref := C.BinaryenTableSize(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoTableGrow(module uintptr, name unsafe.Pointer, value, delta uintptr) uintptr {
	ref := C.BinaryenTableGrow(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
		C.BinaryenExpressionRef(unsafe.Pointer(value)),
		C.BinaryenExpressionRef(unsafe.Pointer(delta)),
	)
	return uintptr(unsafe.Pointer(ref))
}

// --- Function operations ---

func cgoAddFunction(module uintptr, name unsafe.Pointer, params, results uintptr, varTypes []uintptr, body uintptr) uintptr {
	var cVarTypes *C.BinaryenType
	if len(varTypes) > 0 {
		cArr := make([]C.BinaryenType, len(varTypes))
		for i, t := range varTypes {
			cArr[i] = C.BinaryenType(t)
		}
		cVarTypes = &cArr[0]
	}
	ref := C.BinaryenAddFunction(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
		C.BinaryenType(params),
		C.BinaryenType(results),
		cVarTypes,
		C.BinaryenIndex(len(varTypes)),
		C.BinaryenExpressionRef(unsafe.Pointer(body)),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoAddFunctionExport(module uintptr, internalName, externalName unsafe.Pointer) uintptr {
	ref := C.BinaryenAddFunctionExport(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(internalName),
		(*C.char)(externalName),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoAddFunctionImport(module uintptr, internalName, externalModuleName, externalBaseName unsafe.Pointer, params, results uintptr) {
	C.BinaryenAddFunctionImport(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(internalName),
		(*C.char)(externalModuleName),
		(*C.char)(externalBaseName),
		C.BinaryenType(params),
		C.BinaryenType(results),
	)
}

func cgoAddGlobal(module uintptr, name unsafe.Pointer, typ uintptr, mutable bool, init uintptr) uintptr {
	ref := C.BinaryenAddGlobal(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
		C.BinaryenType(typ),
		C.bool(mutable),
		C.BinaryenExpressionRef(unsafe.Pointer(init)),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoAddGlobalExport(module uintptr, internalName, externalName unsafe.Pointer) uintptr {
	ref := C.BinaryenAddGlobalExport(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(internalName),
		(*C.char)(externalName),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoAddGlobalImport(module uintptr, internalName, externalModuleName, externalBaseName unsafe.Pointer, globalType uintptr, mutable bool) {
	C.BinaryenAddGlobalImport(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(internalName),
		(*C.char)(externalModuleName),
		(*C.char)(externalBaseName),
		C.BinaryenType(globalType),
		C.bool(mutable),
	)
}

func cgoAddMemoryExport(module uintptr, internalName, externalName unsafe.Pointer) uintptr {
	ref := C.BinaryenAddMemoryExport(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(internalName),
		(*C.char)(externalName),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoAddMemoryImport(module uintptr, internalName, externalModuleName, externalBaseName unsafe.Pointer, shared bool) {
	var s C.uint8_t
	if shared {
		s = 1
	}
	C.BinaryenAddMemoryImport(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(internalName),
		(*C.char)(externalModuleName),
		(*C.char)(externalBaseName),
		s,
	)
}

func cgoAddTable(module uintptr, name unsafe.Pointer, initial, maximum uint32, tableType uintptr) uintptr {
	ref := C.BinaryenAddTable(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
		C.BinaryenIndex(initial),
		C.BinaryenIndex(maximum),
		C.BinaryenType(tableType),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoAddTableExport(module uintptr, internalName, externalName unsafe.Pointer) uintptr {
	ref := C.BinaryenAddTableExport(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(internalName),
		(*C.char)(externalName),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoAddTableImport(module uintptr, internalName, externalModuleName, externalBaseName unsafe.Pointer) {
	C.BinaryenAddTableImport(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(internalName),
		(*C.char)(externalModuleName),
		(*C.char)(externalBaseName),
	)
}

func cgoAddTag(module uintptr, name unsafe.Pointer, params, results uintptr) uintptr {
	ref := C.BinaryenAddTag(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
		C.BinaryenType(params),
		C.BinaryenType(results),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoAddTagExport(module uintptr, internalName, externalName unsafe.Pointer) uintptr {
	ref := C.BinaryenAddTagExport(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(internalName),
		(*C.char)(externalName),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoAddTagImport(module uintptr, internalName, externalModuleName, externalBaseName unsafe.Pointer, params, results uintptr) {
	C.BinaryenAddTagImport(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(internalName),
		(*C.char)(externalModuleName),
		(*C.char)(externalBaseName),
		C.BinaryenType(params),
		C.BinaryenType(results),
	)
}

func cgoSetMemory(module uintptr, initial, maximum uint32, exportName unsafe.Pointer, segmentNames []unsafe.Pointer, segmentDatas [][]byte, segmentPassives []bool, segmentOffsets []uintptr, segmentSizes []uint32, numSegments int, shared, memory64 bool, memName unsafe.Pointer) {
	var cSegNames **C.char
	var cSegDatas **C.char
	var cSegPassives *C.bool
	var cSegOffsets *C.BinaryenExpressionRef
	var cSegSizes *C.BinaryenIndex

	if numSegments > 0 {
		segNameArr := make([]*C.char, numSegments)
		segDataArr := make([]*C.char, numSegments)
		segPassiveArr := make([]C.bool, numSegments)
		segOffsetArr := make([]C.BinaryenExpressionRef, numSegments)
		segSizeArr := make([]C.BinaryenIndex, numSegments)

		for i := 0; i < numSegments; i++ {
			if i < len(segmentNames) {
				segNameArr[i] = (*C.char)(segmentNames[i])
			}
			if i < len(segmentDatas) && len(segmentDatas[i]) > 0 {
				segDataArr[i] = (*C.char)(unsafe.Pointer(&segmentDatas[i][0]))
			}
			if i < len(segmentPassives) {
				segPassiveArr[i] = C.bool(segmentPassives[i])
			}
			if i < len(segmentOffsets) {
				segOffsetArr[i] = C.BinaryenExpressionRef(unsafe.Pointer(segmentOffsets[i]))
			}
			if i < len(segmentSizes) {
				segSizeArr[i] = C.BinaryenIndex(segmentSizes[i])
			}
		}
		cSegNames = &segNameArr[0]
		cSegDatas = &segDataArr[0]
		cSegPassives = &segPassiveArr[0]
		cSegOffsets = &segOffsetArr[0]
		cSegSizes = &segSizeArr[0]
	}

	C.BinaryenSetMemory(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenIndex(initial),
		C.BinaryenIndex(maximum),
		(*C.char)(exportName),
		cSegNames,
		cSegDatas,
		cSegPassives,
		cSegOffsets,
		cSegSizes,
		C.BinaryenIndex(numSegments),
		C.bool(shared),
		C.bool(memory64),
		(*C.char)(memName),
	)
}

func cgoHasMemory(module uintptr) bool {
	return bool(C.BinaryenHasMemory(C.BinaryenModuleRef(unsafe.Pointer(module))))
}

func cgoSetStart(module uintptr, start uintptr) {
	C.BinaryenSetStart(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenFunctionRef(unsafe.Pointer(start)),
	)
}

// cgoGetStart is not available in this binaryen version — the bridge stub returns 0.
// func cgoGetStart(module uintptr) uintptr { ... }

// --- Global settings ---

func cgoGetOptimizeLevel() int  { return int(C.BinaryenGetOptimizeLevel()) }
func cgoSetOptimizeLevel(l int) { C.BinaryenSetOptimizeLevel(C.int(l)) }
func cgoGetShrinkLevel() int    { return int(C.BinaryenGetShrinkLevel()) }
func cgoSetShrinkLevel(l int)   { C.BinaryenSetShrinkLevel(C.int(l)) }
func cgoGetDebugInfo() bool     { return bool(C.BinaryenGetDebugInfo()) }
func cgoSetDebugInfo(on bool)   { C.BinaryenSetDebugInfo(C.bool(on)) }

func cgoGetLowMemoryUnused() bool     { return bool(C.BinaryenGetLowMemoryUnused()) }
func cgoSetLowMemoryUnused(on bool)   { C.BinaryenSetLowMemoryUnused(C.bool(on)) }
func cgoGetZeroFilledMemory() bool    { return bool(C.BinaryenGetZeroFilledMemory()) }
func cgoSetZeroFilledMemory(on bool)  { C.BinaryenSetZeroFilledMemory(C.bool(on)) }
func cgoGetFastMath() bool            { return bool(C.BinaryenGetFastMath()) }
func cgoSetFastMath(on bool)          { C.BinaryenSetFastMath(C.bool(on)) }
func cgoGetTrapsNeverHappen() bool    { return bool(C.BinaryenGetTrapsNeverHappen()) }
func cgoSetTrapsNeverHappen(on bool)  { C.BinaryenSetTrapsNeverHappen(C.bool(on)) }
func cgoGetClosedWorld() bool         { return bool(C.BinaryenGetClosedWorld()) }
func cgoSetClosedWorld(on bool)       { C.BinaryenSetClosedWorld(C.bool(on)) }

func cgoGetGenerateStackIR() bool     { return bool(C.BinaryenGetGenerateStackIR()) }
func cgoSetGenerateStackIR(on bool)   { C.BinaryenSetGenerateStackIR(C.bool(on)) }
func cgoGetOptimizeStackIR() bool     { return bool(C.BinaryenGetOptimizeStackIR()) }
func cgoSetOptimizeStackIR(on bool)   { C.BinaryenSetOptimizeStackIR(C.bool(on)) }

func cgoGetAlwaysInlineMaxSize() int          { return int(C.BinaryenGetAlwaysInlineMaxSize()) }
func cgoSetAlwaysInlineMaxSize(size int)      { C.BinaryenSetAlwaysInlineMaxSize(C.BinaryenIndex(size)) }
func cgoGetFlexibleInlineMaxSize() int        { return int(C.BinaryenGetFlexibleInlineMaxSize()) }
func cgoSetFlexibleInlineMaxSize(size int)    { C.BinaryenSetFlexibleInlineMaxSize(C.BinaryenIndex(size)) }
func cgoGetOneCallerInlineMaxSize() int       { return int(C.BinaryenGetOneCallerInlineMaxSize()) }
func cgoSetOneCallerInlineMaxSize(size int)   { C.BinaryenSetOneCallerInlineMaxSize(C.BinaryenIndex(size)) }
func cgoGetAllowInliningFunctionsWithLoops() bool {
	return bool(C.BinaryenGetAllowInliningFunctionsWithLoops())
}
func cgoSetAllowInliningFunctionsWithLoops(on bool) {
	C.BinaryenSetAllowInliningFunctionsWithLoops(C.bool(on))
}

func cgoGetPassArgument(name unsafe.Pointer) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenGetPassArgument((*C.char)(name)))
}

func cgoSetPassArgument(name, value unsafe.Pointer) {
	C.BinaryenSetPassArgument((*C.char)(name), (*C.char)(value))
}

func cgoClearPassArguments() {
	C.BinaryenClearPassArguments()
}

func cgoHasPassToSkip(name unsafe.Pointer) bool {
	return bool(C.BinaryenHasPassToSkip((*C.char)(name)))
}

func cgoAddPassToSkip(name unsafe.Pointer) {
	C.BinaryenAddPassToSkip((*C.char)(name))
}

func cgoClearPassesToSkip() {
	C.BinaryenClearPassesToSkip()
}

// --- Relooper ---

func cgoRelooperCreate(module uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.RelooperCreate(C.BinaryenModuleRef(unsafe.Pointer(module)))))
}

func cgoRelooperAddBlock(relooper uintptr, code uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.RelooperAddBlock(
		C.RelooperRef(unsafe.Pointer(relooper)),
		C.BinaryenExpressionRef(unsafe.Pointer(code)),
	)))
}

func cgoRelooperAddBranch(from, to uintptr, condition, code uintptr) {
	C.RelooperAddBranch(
		C.RelooperBlockRef(unsafe.Pointer(from)),
		C.RelooperBlockRef(unsafe.Pointer(to)),
		C.BinaryenExpressionRef(unsafe.Pointer(condition)),
		C.BinaryenExpressionRef(unsafe.Pointer(code)),
	)
}

func cgoRelooperAddBlockWithSwitch(relooper uintptr, code, condition uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.RelooperAddBlockWithSwitch(
		C.RelooperRef(unsafe.Pointer(relooper)),
		C.BinaryenExpressionRef(unsafe.Pointer(code)),
		C.BinaryenExpressionRef(unsafe.Pointer(condition)),
	)))
}

func cgoRelooperAddBranchForSwitch(from, to uintptr, indexes []uint32, code uintptr) {
	var cIndexes *C.BinaryenIndex
	if len(indexes) > 0 {
		cArr := make([]C.BinaryenIndex, len(indexes))
		for i, idx := range indexes {
			cArr[i] = C.BinaryenIndex(idx)
		}
		cIndexes = &cArr[0]
	}
	C.RelooperAddBranchForSwitch(
		C.RelooperBlockRef(unsafe.Pointer(from)),
		C.RelooperBlockRef(unsafe.Pointer(to)),
		cIndexes,
		C.BinaryenIndex(len(indexes)),
		C.BinaryenExpressionRef(unsafe.Pointer(code)),
	)
}

func cgoRelooperRenderAndDispose(relooper uintptr, entry uintptr, labelHelper uint32) uintptr {
	ref := C.RelooperRenderAndDispose(
		C.RelooperRef(unsafe.Pointer(relooper)),
		C.RelooperBlockRef(unsafe.Pointer(entry)),
		C.BinaryenIndex(labelHelper),
	)
	return uintptr(unsafe.Pointer(ref))
}

// --- ExpressionRunner ---

func cgoExpressionRunnerCreate(module uintptr, flags uint32, maxDepth, maxLoopIterations uint32) uintptr {
	return uintptr(unsafe.Pointer(C.ExpressionRunnerCreate(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.ExpressionRunnerFlags(flags),
		C.BinaryenIndex(maxDepth),
		C.BinaryenIndex(maxLoopIterations),
	)))
}

func cgoExpressionRunnerSetLocalValue(runner uintptr, index uint32, value uintptr) bool {
	return bool(C.ExpressionRunnerSetLocalValue(
		C.ExpressionRunnerRef(unsafe.Pointer(runner)),
		C.BinaryenIndex(index),
		C.BinaryenExpressionRef(unsafe.Pointer(value)),
	))
}

func cgoExpressionRunnerSetGlobalValue(runner uintptr, name unsafe.Pointer, value uintptr) bool {
	return bool(C.ExpressionRunnerSetGlobalValue(
		C.ExpressionRunnerRef(unsafe.Pointer(runner)),
		(*C.char)(name),
		C.BinaryenExpressionRef(unsafe.Pointer(value)),
	))
}

func cgoExpressionRunnerRunAndDispose(runner uintptr, expr uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.ExpressionRunnerRunAndDispose(
		C.ExpressionRunnerRef(unsafe.Pointer(runner)),
		C.BinaryenExpressionRef(unsafe.Pointer(expr)),
	)))
}

// --- TypeBuilder ---

func cgoTypeBuilderCreate(size int) uintptr {
	return uintptr(unsafe.Pointer(C.TypeBuilderCreate(C.BinaryenIndex(size))))
}

func cgoTypeBuilderGrow(builder uintptr, count int) {
	C.TypeBuilderGrow(C.TypeBuilderRef(unsafe.Pointer(builder)), C.BinaryenIndex(count))
}

func cgoTypeBuilderGetSize(builder uintptr) int {
	return int(C.TypeBuilderGetSize(C.TypeBuilderRef(unsafe.Pointer(builder))))
}

func cgoTypeBuilderSetSignatureType(builder uintptr, index int, params, results uintptr) {
	C.TypeBuilderSetSignatureType(
		C.TypeBuilderRef(unsafe.Pointer(builder)),
		C.BinaryenIndex(index),
		C.BinaryenType(params),
		C.BinaryenType(results),
	)
}

func cgoTypeBuilderSetStructType(builder uintptr, index int, fieldTypes []uintptr, fieldPackedTypes []uint32, fieldMutables []bool) {
	nFields := len(fieldTypes)
	cFieldTypes := make([]C.BinaryenType, nFields)
	cPackedTypes := make([]C.BinaryenPackedType, nFields)
	cMutables := make([]C.bool, nFields)
	for i := 0; i < nFields; i++ {
		cFieldTypes[i] = C.BinaryenType(fieldTypes[i])
		if i < len(fieldPackedTypes) {
			cPackedTypes[i] = C.BinaryenPackedType(fieldPackedTypes[i])
		}
		if i < len(fieldMutables) {
			cMutables[i] = C.bool(fieldMutables[i])
		}
	}

	var pFieldTypes *C.BinaryenType
	var pPackedTypes *C.BinaryenPackedType
	var pMutables *C.bool
	if nFields > 0 {
		pFieldTypes = &cFieldTypes[0]
		pPackedTypes = &cPackedTypes[0]
		pMutables = &cMutables[0]
	}

	C.TypeBuilderSetStructType(
		C.TypeBuilderRef(unsafe.Pointer(builder)),
		C.BinaryenIndex(index),
		pFieldTypes,
		pPackedTypes,
		pMutables,
		C.int(nFields),
	)
}

func cgoTypeBuilderSetArrayType(builder uintptr, index int, elementType uintptr, elementPackedType uint32, elementMutable bool) {
	var m C.int
	if elementMutable {
		m = 1
	}
	C.TypeBuilderSetArrayType(
		C.TypeBuilderRef(unsafe.Pointer(builder)),
		C.BinaryenIndex(index),
		C.BinaryenType(elementType),
		C.BinaryenPackedType(elementPackedType),
		m,
	)
}

func cgoTypeBuilderGetTempHeapType(builder uintptr, index int) uintptr {
	return uintptr(C.TypeBuilderGetTempHeapType(
		C.TypeBuilderRef(unsafe.Pointer(builder)),
		C.BinaryenIndex(index),
	))
}

func cgoTypeBuilderGetTempRefType(builder uintptr, ht uintptr, nullable bool) uintptr {
	var n C.int
	if nullable {
		n = 1
	}
	return uintptr(C.TypeBuilderGetTempRefType(
		C.TypeBuilderRef(unsafe.Pointer(builder)),
		C.BinaryenHeapType(ht),
		n,
	))
}

func cgoTypeBuilderGetTempTupleType(builder uintptr, types []uintptr) uintptr {
	if len(types) == 0 {
		return cgoTypeNone()
	}
	cTypes := make([]C.BinaryenType, len(types))
	for i, t := range types {
		cTypes[i] = C.BinaryenType(t)
	}
	return uintptr(C.TypeBuilderGetTempTupleType(
		C.TypeBuilderRef(unsafe.Pointer(builder)),
		&cTypes[0],
		C.BinaryenIndex(len(types)),
	))
}

func cgoTypeBuilderSetSubType(builder uintptr, index int, superType uintptr) {
	C.TypeBuilderSetSubType(
		C.TypeBuilderRef(unsafe.Pointer(builder)),
		C.BinaryenIndex(index),
		C.BinaryenHeapType(superType),
	)
}

func cgoTypeBuilderSetOpen(builder uintptr, index int) {
	C.TypeBuilderSetOpen(
		C.TypeBuilderRef(unsafe.Pointer(builder)),
		C.BinaryenIndex(index),
	)
}

func cgoTypeBuilderCreateRecGroup(builder uintptr, index, length int) {
	C.TypeBuilderCreateRecGroup(
		C.TypeBuilderRef(unsafe.Pointer(builder)),
		C.BinaryenIndex(index),
		C.BinaryenIndex(length),
	)
}

func cgoTypeBuilderBuildAndDispose(builder uintptr, heapTypes []uintptr) bool {
	var cHeapTypes *C.BinaryenHeapType
	if len(heapTypes) > 0 {
		cArr := make([]C.BinaryenHeapType, len(heapTypes))
		cHeapTypes = &cArr[0]
		var errorIndex C.BinaryenIndex
		var errorReason C.TypeBuilderErrorReason
		ok := bool(C.TypeBuilderBuildAndDispose(
			C.TypeBuilderRef(unsafe.Pointer(builder)),
			cHeapTypes,
			&errorIndex,
			&errorReason,
		))
		if ok {
			for i := range heapTypes {
				heapTypes[i] = uintptr(cArr[i])
			}
		}
		return ok
	}
	return false
}

// --- DataSegment operations ---

func cgoAddDataSegment(module uintptr, segmentName, memoryName unsafe.Pointer, passive bool, offset uintptr, data []byte) {
	var cData *C.char
	if len(data) > 0 {
		cData = (*C.char)(unsafe.Pointer(&data[0]))
	}
	C.BinaryenAddDataSegment(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(segmentName),
		(*C.char)(memoryName),
		C.bool(passive),
		C.BinaryenExpressionRef(unsafe.Pointer(offset)),
		cData,
		C.BinaryenIndex(len(data)),
	)
}

// --- Element segment operations ---

func cgoAddActiveElementSegment(module uintptr, table unsafe.Pointer, name unsafe.Pointer, funcNames []unsafe.Pointer, offset uintptr) uintptr {
	var cFuncNames **C.char
	if len(funcNames) > 0 {
		cFNArr := make([]*C.char, len(funcNames))
		for i, fn := range funcNames {
			cFNArr[i] = (*C.char)(fn)
		}
		cFuncNames = &cFNArr[0]
	}
	ref := C.BinaryenAddActiveElementSegment(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(table),
		(*C.char)(name),
		cFuncNames,
		C.BinaryenIndex(len(funcNames)),
		C.BinaryenExpressionRef(unsafe.Pointer(offset)),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoAddPassiveElementSegment(module uintptr, name unsafe.Pointer, funcNames []unsafe.Pointer) uintptr {
	var cFuncNames **C.char
	if len(funcNames) > 0 {
		cFNArr := make([]*C.char, len(funcNames))
		for i, fn := range funcNames {
			cFNArr[i] = (*C.char)(fn)
		}
		cFuncNames = &cFNArr[0]
	}
	ref := C.BinaryenAddPassiveElementSegment(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
		cFuncNames,
		C.BinaryenIndex(len(funcNames)),
	)
	return uintptr(unsafe.Pointer(ref))
}

// --- Add custom section ---

func cgoAddCustomSection(module uintptr, name unsafe.Pointer, contents []byte) {
	var cContents *C.char
	if len(contents) > 0 {
		cContents = (*C.char)(unsafe.Pointer(&contents[0]))
	}
	C.BinaryenAddCustomSection(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
		cContents,
		C.BinaryenIndex(len(contents)),
	)
}

// --- GC / struct / array / ref expression constructors ---

func cgoStructNew(module uintptr, operands []uintptr, typ uintptr) uintptr {
	var cOperands *C.BinaryenExpressionRef
	if len(operands) > 0 {
		cArr := make([]C.BinaryenExpressionRef, len(operands))
		for i, o := range operands {
			cArr[i] = C.BinaryenExpressionRef(unsafe.Pointer(o))
		}
		cOperands = &cArr[0]
	}
	ref := C.BinaryenStructNew(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		cOperands,
		C.BinaryenIndex(len(operands)),
		C.BinaryenHeapType(typ),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoStructGet(module uintptr, index uint32, ref uintptr, typ uintptr, signed bool) uintptr {
	r := C.BinaryenStructGet(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenIndex(index),
		C.BinaryenExpressionRef(unsafe.Pointer(ref)),
		C.BinaryenType(typ),
		C.bool(signed),
	)
	return uintptr(unsafe.Pointer(r))
}

func cgoStructSet(module uintptr, index uint32, ref, value uintptr) uintptr {
	r := C.BinaryenStructSet(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenIndex(index),
		C.BinaryenExpressionRef(unsafe.Pointer(ref)),
		C.BinaryenExpressionRef(unsafe.Pointer(value)),
	)
	return uintptr(unsafe.Pointer(r))
}

func cgoArrayNew(module uintptr, typ uintptr, size, init uintptr) uintptr {
	ref := C.BinaryenArrayNew(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenHeapType(typ),
		C.BinaryenExpressionRef(unsafe.Pointer(size)),
		C.BinaryenExpressionRef(unsafe.Pointer(init)),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoArrayNewFixed(module uintptr, typ uintptr, values []uintptr) uintptr {
	var cValues *C.BinaryenExpressionRef
	if len(values) > 0 {
		cArr := make([]C.BinaryenExpressionRef, len(values))
		for i, v := range values {
			cArr[i] = C.BinaryenExpressionRef(unsafe.Pointer(v))
		}
		cValues = &cArr[0]
	}
	ref := C.BinaryenArrayNewFixed(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenHeapType(typ),
		cValues,
		C.BinaryenIndex(len(values)),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoArrayGet(module uintptr, ref, index uintptr, typ uintptr, signed bool) uintptr {
	r := C.BinaryenArrayGet(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenExpressionRef(unsafe.Pointer(ref)),
		C.BinaryenExpressionRef(unsafe.Pointer(index)),
		C.BinaryenType(typ),
		C.bool(signed),
	)
	return uintptr(unsafe.Pointer(r))
}

func cgoArraySet(module uintptr, ref, index, value uintptr) uintptr {
	r := C.BinaryenArraySet(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenExpressionRef(unsafe.Pointer(ref)),
		C.BinaryenExpressionRef(unsafe.Pointer(index)),
		C.BinaryenExpressionRef(unsafe.Pointer(value)),
	)
	return uintptr(unsafe.Pointer(r))
}

func cgoArrayLen(module uintptr, ref uintptr) uintptr {
	r := C.BinaryenArrayLen(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenExpressionRef(unsafe.Pointer(ref)),
	)
	return uintptr(unsafe.Pointer(r))
}

func cgoArrayCopy(module uintptr, destRef, destIndex, srcRef, srcIndex, length uintptr) uintptr {
	r := C.BinaryenArrayCopy(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenExpressionRef(unsafe.Pointer(destRef)),
		C.BinaryenExpressionRef(unsafe.Pointer(destIndex)),
		C.BinaryenExpressionRef(unsafe.Pointer(srcRef)),
		C.BinaryenExpressionRef(unsafe.Pointer(srcIndex)),
		C.BinaryenExpressionRef(unsafe.Pointer(length)),
	)
	return uintptr(unsafe.Pointer(r))
}

func cgoRefI31(module uintptr, value uintptr) uintptr {
	ref := C.BinaryenRefI31(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenExpressionRef(unsafe.Pointer(value)),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoI31Get(module uintptr, i31 uintptr, signed bool) uintptr {
	ref := C.BinaryenI31Get(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenExpressionRef(unsafe.Pointer(i31)),
		C.bool(signed),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoRefTest(module uintptr, ref uintptr, castType uintptr) uintptr {
	r := C.BinaryenRefTest(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenExpressionRef(unsafe.Pointer(ref)),
		C.BinaryenType(castType),
	)
	return uintptr(unsafe.Pointer(r))
}

func cgoRefCast(module uintptr, ref uintptr, typ uintptr) uintptr {
	r := C.BinaryenRefCast(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenExpressionRef(unsafe.Pointer(ref)),
		C.BinaryenType(typ),
	)
	return uintptr(unsafe.Pointer(r))
}

func cgoBrOn(module uintptr, op int32, name unsafe.Pointer, ref uintptr, castType uintptr) uintptr {
	r := C.BinaryenBrOn(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenOp(op),
		(*C.char)(name),
		C.BinaryenExpressionRef(unsafe.Pointer(ref)),
		C.BinaryenType(castType),
	)
	return uintptr(unsafe.Pointer(r))
}

func cgoCallRef(module uintptr, target uintptr, operands []uintptr, typ uintptr, isReturn bool) uintptr {
	var cOperands *C.BinaryenExpressionRef
	if len(operands) > 0 {
		cArr := make([]C.BinaryenExpressionRef, len(operands))
		for i, o := range operands {
			cArr[i] = C.BinaryenExpressionRef(unsafe.Pointer(o))
		}
		cOperands = &cArr[0]
	}
	ref := C.BinaryenCallRef(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenExpressionRef(unsafe.Pointer(target)),
		cOperands,
		C.BinaryenIndex(len(operands)),
		C.BinaryenType(typ),
		C.bool(isReturn),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoRefCastNop(module uintptr, ref uintptr) uintptr {
	// Not all binaryen versions have RefCastNop — use RefCast with same type as fallback
	return cgoRefCast(module, ref, cgoExpressionGetType(ref))
}

// --- String operations (stringref) ---

func cgoStringNew(module uintptr, op int32, ref, start, end uintptr) uintptr {
	r := C.BinaryenStringNew(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenOp(op),
		C.BinaryenExpressionRef(unsafe.Pointer(ref)),
		C.BinaryenExpressionRef(unsafe.Pointer(start)),
		C.BinaryenExpressionRef(unsafe.Pointer(end)),
	)
	return uintptr(unsafe.Pointer(r))
}

func cgoStringConst(module uintptr, name unsafe.Pointer) uintptr {
	ref := C.BinaryenStringConst(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoStringMeasure(module uintptr, op int32, ref uintptr) uintptr {
	r := C.BinaryenStringMeasure(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenOp(op),
		C.BinaryenExpressionRef(unsafe.Pointer(ref)),
	)
	return uintptr(unsafe.Pointer(r))
}

func cgoStringEncode(module uintptr, op int32, ref, ptr, start uintptr) uintptr {
	r := C.BinaryenStringEncode(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenOp(op),
		C.BinaryenExpressionRef(unsafe.Pointer(ref)),
		C.BinaryenExpressionRef(unsafe.Pointer(ptr)),
		C.BinaryenExpressionRef(unsafe.Pointer(start)),
	)
	return uintptr(unsafe.Pointer(r))
}

func cgoStringConcat(module uintptr, left, right uintptr) uintptr {
	ref := C.BinaryenStringConcat(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenExpressionRef(unsafe.Pointer(left)),
		C.BinaryenExpressionRef(unsafe.Pointer(right)),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoStringEq(module uintptr, op int32, left, right uintptr) uintptr {
	ref := C.BinaryenStringEq(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenOp(op),
		C.BinaryenExpressionRef(unsafe.Pointer(left)),
		C.BinaryenExpressionRef(unsafe.Pointer(right)),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoStringWTF16Get(module uintptr, ref, pos uintptr) uintptr {
	r := C.BinaryenStringWTF16Get(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenExpressionRef(unsafe.Pointer(ref)),
		C.BinaryenExpressionRef(unsafe.Pointer(pos)),
	)
	return uintptr(unsafe.Pointer(r))
}

func cgoStringSliceWTF(module uintptr, ref, start, end uintptr) uintptr {
	r := C.BinaryenStringSliceWTF(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenExpressionRef(unsafe.Pointer(ref)),
		C.BinaryenExpressionRef(unsafe.Pointer(start)),
		C.BinaryenExpressionRef(unsafe.Pointer(end)),
	)
	return uintptr(unsafe.Pointer(r))
}

// --- SideEffects ---

func cgoSideEffectsNone() uint32 {
	return uint32(C.BinaryenSideEffectNone())
}

func cgoSideEffectsAny() uint32 {
	return uint32(C.BinaryenSideEffectAny())
}

func cgoExpressionGetSideEffects(expr uintptr, module uintptr) uint32 {
	return uint32(C.BinaryenExpressionGetSideEffects(
		C.BinaryenExpressionRef(unsafe.Pointer(expr)),
		C.BinaryenModuleRef(unsafe.Pointer(module)),
	))
}

// --- Feature flags ---

func cgoFeatureAll() uint32     { return uint32(C.BinaryenFeatureAll()) }
func cgoFeatureMVP() uint32     { return uint32(C.BinaryenFeatureMVP()) }
func cgoFeatureAtomics() uint32 { return uint32(C.BinaryenFeatureAtomics()) }
func cgoFeatureBulkMemory() uint32 { return uint32(C.BinaryenFeatureBulkMemory()) }
func cgoFeatureMutableGlobals() uint32 { return uint32(C.BinaryenFeatureMutableGlobals()) }
func cgoFeatureNontrappingFPToInt() uint32 { return uint32(C.BinaryenFeatureNontrappingFPToInt()) }
func cgoFeatureSignExt() uint32 { return uint32(C.BinaryenFeatureSignExt()) }
func cgoFeatureSIMD128() uint32 { return uint32(C.BinaryenFeatureSIMD128()) }
func cgoFeatureExceptionHandling() uint32 { return uint32(C.BinaryenFeatureExceptionHandling()) }
func cgoFeatureTailCall() uint32 { return uint32(C.BinaryenFeatureTailCall()) }
func cgoFeatureReferenceTypes() uint32 { return uint32(C.BinaryenFeatureReferenceTypes()) }
func cgoFeatureMultivalue() uint32 { return uint32(C.BinaryenFeatureMultivalue()) }
func cgoFeatureGC() uint32      { return uint32(C.BinaryenFeatureGC()) }
func cgoFeatureMemory64() uint32 { return uint32(C.BinaryenFeatureMemory64()) }
func cgoFeatureStrings() uint32  { return uint32(C.BinaryenFeatureStrings()) }
func cgoFeatureMultiMemory() uint32 { return uint32(C.BinaryenFeatureMultiMemory()) }

// --- Opcode functions ---
// These return BinaryenOp constants.

func cgoClzInt32() int32    { return int32(C.BinaryenClzInt32()) }
func cgoCtzInt32() int32    { return int32(C.BinaryenCtzInt32()) }
func cgoPopcntInt32() int32 { return int32(C.BinaryenPopcntInt32()) }
func cgoNegFloat32() int32  { return int32(C.BinaryenNegFloat32()) }
func cgoAbsFloat32() int32  { return int32(C.BinaryenAbsFloat32()) }
func cgoCeilFloat32() int32 { return int32(C.BinaryenCeilFloat32()) }
func cgoFloorFloat32() int32 { return int32(C.BinaryenFloorFloat32()) }
func cgoTruncFloat32() int32 { return int32(C.BinaryenTruncFloat32()) }
func cgoNearestFloat32() int32 { return int32(C.BinaryenNearestFloat32()) }
func cgoSqrtFloat32() int32 { return int32(C.BinaryenSqrtFloat32()) }
func cgoEqZInt32() int32    { return int32(C.BinaryenEqZInt32()) }
func cgoClzInt64() int32    { return int32(C.BinaryenClzInt64()) }
func cgoCtzInt64() int32    { return int32(C.BinaryenCtzInt64()) }
func cgoPopcntInt64() int32 { return int32(C.BinaryenPopcntInt64()) }
func cgoNegFloat64() int32  { return int32(C.BinaryenNegFloat64()) }
func cgoAbsFloat64() int32  { return int32(C.BinaryenAbsFloat64()) }
func cgoCeilFloat64() int32 { return int32(C.BinaryenCeilFloat64()) }
func cgoFloorFloat64() int32 { return int32(C.BinaryenFloorFloat64()) }
func cgoTruncFloat64() int32 { return int32(C.BinaryenTruncFloat64()) }
func cgoNearestFloat64() int32 { return int32(C.BinaryenNearestFloat64()) }
func cgoSqrtFloat64() int32 { return int32(C.BinaryenSqrtFloat64()) }
func cgoEqZInt64() int32    { return int32(C.BinaryenEqZInt64()) }

func cgoAddInt32() int32 { return int32(C.BinaryenAddInt32()) }
func cgoSubInt32() int32 { return int32(C.BinaryenSubInt32()) }
func cgoMulInt32() int32 { return int32(C.BinaryenMulInt32()) }
func cgoDivSInt32() int32 { return int32(C.BinaryenDivSInt32()) }
func cgoDivUInt32() int32 { return int32(C.BinaryenDivUInt32()) }
func cgoRemSInt32() int32 { return int32(C.BinaryenRemSInt32()) }
func cgoRemUInt32() int32 { return int32(C.BinaryenRemUInt32()) }
func cgoAndInt32() int32  { return int32(C.BinaryenAndInt32()) }
func cgoOrInt32() int32   { return int32(C.BinaryenOrInt32()) }
func cgoXorInt32() int32  { return int32(C.BinaryenXorInt32()) }
func cgoShlInt32() int32  { return int32(C.BinaryenShlInt32()) }
func cgoShrSInt32() int32 { return int32(C.BinaryenShrSInt32()) }
func cgoShrUInt32() int32 { return int32(C.BinaryenShrUInt32()) }
func cgoRotLInt32() int32 { return int32(C.BinaryenRotLInt32()) }
func cgoRotRInt32() int32 { return int32(C.BinaryenRotRInt32()) }
func cgoEqInt32() int32   { return int32(C.BinaryenEqInt32()) }
func cgoNeInt32() int32   { return int32(C.BinaryenNeInt32()) }
func cgoLtSInt32() int32  { return int32(C.BinaryenLtSInt32()) }
func cgoLtUInt32() int32  { return int32(C.BinaryenLtUInt32()) }
func cgoLeSInt32() int32  { return int32(C.BinaryenLeSInt32()) }
func cgoLeUInt32() int32  { return int32(C.BinaryenLeUInt32()) }
func cgoGtSInt32() int32  { return int32(C.BinaryenGtSInt32()) }
func cgoGtUInt32() int32  { return int32(C.BinaryenGtUInt32()) }
func cgoGeSInt32() int32  { return int32(C.BinaryenGeSInt32()) }
func cgoGeUInt32() int32  { return int32(C.BinaryenGeUInt32()) }

func cgoAddInt64() int32 { return int32(C.BinaryenAddInt64()) }
func cgoSubInt64() int32 { return int32(C.BinaryenSubInt64()) }
func cgoMulInt64() int32 { return int32(C.BinaryenMulInt64()) }
func cgoDivSInt64() int32 { return int32(C.BinaryenDivSInt64()) }
func cgoDivUInt64() int32 { return int32(C.BinaryenDivUInt64()) }
func cgoRemSInt64() int32 { return int32(C.BinaryenRemSInt64()) }
func cgoRemUInt64() int32 { return int32(C.BinaryenRemUInt64()) }
func cgoAndInt64() int32  { return int32(C.BinaryenAndInt64()) }
func cgoOrInt64() int32   { return int32(C.BinaryenOrInt64()) }
func cgoXorInt64() int32  { return int32(C.BinaryenXorInt64()) }
func cgoShlInt64() int32  { return int32(C.BinaryenShlInt64()) }
func cgoShrSInt64() int32 { return int32(C.BinaryenShrSInt64()) }
func cgoShrUInt64() int32 { return int32(C.BinaryenShrUInt64()) }
func cgoRotLInt64() int32 { return int32(C.BinaryenRotLInt64()) }
func cgoRotRInt64() int32 { return int32(C.BinaryenRotRInt64()) }
func cgoEqInt64() int32   { return int32(C.BinaryenEqInt64()) }
func cgoNeInt64() int32   { return int32(C.BinaryenNeInt64()) }
func cgoLtSInt64() int32  { return int32(C.BinaryenLtSInt64()) }
func cgoLtUInt64() int32  { return int32(C.BinaryenLtUInt64()) }
func cgoLeSInt64() int32  { return int32(C.BinaryenLeSInt64()) }
func cgoLeUInt64() int32  { return int32(C.BinaryenLeUInt64()) }
func cgoGtSInt64() int32  { return int32(C.BinaryenGtSInt64()) }
func cgoGtUInt64() int32  { return int32(C.BinaryenGtUInt64()) }
func cgoGeSInt64() int32  { return int32(C.BinaryenGeSInt64()) }
func cgoGeUInt64() int32  { return int32(C.BinaryenGeUInt64()) }

func cgoAddFloat32() int32 { return int32(C.BinaryenAddFloat32()) }
func cgoSubFloat32() int32 { return int32(C.BinaryenSubFloat32()) }
func cgoMulFloat32() int32 { return int32(C.BinaryenMulFloat32()) }
func cgoDivFloat32() int32 { return int32(C.BinaryenDivFloat32()) }
func cgoCopySignFloat32() int32 { return int32(C.BinaryenCopySignFloat32()) }
func cgoMinFloat32() int32 { return int32(C.BinaryenMinFloat32()) }
func cgoMaxFloat32() int32 { return int32(C.BinaryenMaxFloat32()) }
func cgoEqFloat32() int32  { return int32(C.BinaryenEqFloat32()) }
func cgoNeFloat32() int32  { return int32(C.BinaryenNeFloat32()) }
func cgoLtFloat32() int32  { return int32(C.BinaryenLtFloat32()) }
func cgoLeFloat32() int32  { return int32(C.BinaryenLeFloat32()) }
func cgoGtFloat32() int32  { return int32(C.BinaryenGtFloat32()) }
func cgoGeFloat32() int32  { return int32(C.BinaryenGeFloat32()) }

func cgoAddFloat64() int32 { return int32(C.BinaryenAddFloat64()) }
func cgoSubFloat64() int32 { return int32(C.BinaryenSubFloat64()) }
func cgoMulFloat64() int32 { return int32(C.BinaryenMulFloat64()) }
func cgoDivFloat64() int32 { return int32(C.BinaryenDivFloat64()) }
func cgoCopySignFloat64() int32 { return int32(C.BinaryenCopySignFloat64()) }
func cgoMinFloat64() int32 { return int32(C.BinaryenMinFloat64()) }
func cgoMaxFloat64() int32 { return int32(C.BinaryenMaxFloat64()) }
func cgoEqFloat64() int32  { return int32(C.BinaryenEqFloat64()) }
func cgoNeFloat64() int32  { return int32(C.BinaryenNeFloat64()) }
func cgoLtFloat64() int32  { return int32(C.BinaryenLtFloat64()) }
func cgoLeFloat64() int32  { return int32(C.BinaryenLeFloat64()) }
func cgoGtFloat64() int32  { return int32(C.BinaryenGtFloat64()) }
func cgoGeFloat64() int32  { return int32(C.BinaryenGeFloat64()) }

// Conversion ops
func cgoTruncSFloat32ToInt32() int32 { return int32(C.BinaryenTruncSFloat32ToInt32()) }
func cgoTruncSFloat32ToInt64() int32 { return int32(C.BinaryenTruncSFloat32ToInt64()) }
func cgoTruncUFloat32ToInt32() int32 { return int32(C.BinaryenTruncUFloat32ToInt32()) }
func cgoTruncUFloat32ToInt64() int32 { return int32(C.BinaryenTruncUFloat32ToInt64()) }
func cgoTruncSFloat64ToInt32() int32 { return int32(C.BinaryenTruncSFloat64ToInt32()) }
func cgoTruncSFloat64ToInt64() int32 { return int32(C.BinaryenTruncSFloat64ToInt64()) }
func cgoTruncUFloat64ToInt32() int32 { return int32(C.BinaryenTruncUFloat64ToInt32()) }
func cgoTruncUFloat64ToInt64() int32 { return int32(C.BinaryenTruncUFloat64ToInt64()) }
func cgoReinterpretFloat32() int32   { return int32(C.BinaryenReinterpretFloat32()) }
func cgoReinterpretFloat64() int32   { return int32(C.BinaryenReinterpretFloat64()) }
func cgoConvertSInt32ToFloat32() int32 { return int32(C.BinaryenConvertSInt32ToFloat32()) }
func cgoConvertSInt32ToFloat64() int32 { return int32(C.BinaryenConvertSInt32ToFloat64()) }
func cgoConvertUInt32ToFloat32() int32 { return int32(C.BinaryenConvertUInt32ToFloat32()) }
func cgoConvertUInt32ToFloat64() int32 { return int32(C.BinaryenConvertUInt32ToFloat64()) }
func cgoConvertSInt64ToFloat32() int32 { return int32(C.BinaryenConvertSInt64ToFloat32()) }
func cgoConvertSInt64ToFloat64() int32 { return int32(C.BinaryenConvertSInt64ToFloat64()) }
func cgoConvertUInt64ToFloat32() int32 { return int32(C.BinaryenConvertUInt64ToFloat32()) }
func cgoConvertUInt64ToFloat64() int32 { return int32(C.BinaryenConvertUInt64ToFloat64()) }
func cgoPromoteFloat32() int32       { return int32(C.BinaryenPromoteFloat32()) }
func cgoDemoteFloat64() int32        { return int32(C.BinaryenDemoteFloat64()) }
func cgoReinterpretInt32() int32     { return int32(C.BinaryenReinterpretInt32()) }
func cgoReinterpretInt64() int32     { return int32(C.BinaryenReinterpretInt64()) }
func cgoExtendSInt32() int32         { return int32(C.BinaryenExtendSInt32()) }
func cgoExtendUInt32() int32         { return int32(C.BinaryenExtendUInt32()) }
func cgoWrapInt64() int32            { return int32(C.BinaryenWrapInt64()) }
func cgoTruncSatSFloat32ToInt32() int32 { return int32(C.BinaryenTruncSatSFloat32ToInt32()) }
func cgoTruncSatSFloat32ToInt64() int32 { return int32(C.BinaryenTruncSatSFloat32ToInt64()) }
func cgoTruncSatUFloat32ToInt32() int32 { return int32(C.BinaryenTruncSatUFloat32ToInt32()) }
func cgoTruncSatUFloat32ToInt64() int32 { return int32(C.BinaryenTruncSatUFloat32ToInt64()) }
func cgoTruncSatSFloat64ToInt32() int32 { return int32(C.BinaryenTruncSatSFloat64ToInt32()) }
func cgoTruncSatSFloat64ToInt64() int32 { return int32(C.BinaryenTruncSatSFloat64ToInt64()) }
func cgoTruncSatUFloat64ToInt32() int32 { return int32(C.BinaryenTruncSatUFloat64ToInt32()) }
func cgoTruncSatUFloat64ToInt64() int32 { return int32(C.BinaryenTruncSatUFloat64ToInt64()) }
func cgoExtendS8Int32() int32       { return int32(C.BinaryenExtendS8Int32()) }
func cgoExtendS16Int32() int32      { return int32(C.BinaryenExtendS16Int32()) }
func cgoExtendS8Int64() int32       { return int32(C.BinaryenExtendS8Int64()) }
func cgoExtendS16Int64() int32      { return int32(C.BinaryenExtendS16Int64()) }
func cgoExtendS32Int64() int32      { return int32(C.BinaryenExtendS32Int64()) }

// --- GC ref ops ---
func cgoBrOnCast() int32       { return int32(C.BinaryenBrOnCast()) }
func cgoBrOnCastFail() int32   { return int32(C.BinaryenBrOnCastFail()) }
func cgoRefAsNonNull() int32   { return int32(C.BinaryenRefAsNonNull()) }
func cgoRefAsExternInternalize() int32 { return int32(C.BinaryenRefAsExternInternalize()) }
func cgoRefAsExternExternalize() int32 { return int32(C.BinaryenRefAsExternExternalize()) }

// --- String ops (only those available in this binaryen version) ---
func cgoStringNewLossyUTF8Array() int32 { return int32(C.BinaryenStringNewLossyUTF8Array()) }
func cgoStringNewWTF16Array() int32 { return int32(C.BinaryenStringNewWTF16Array()) }
func cgoStringNewFromCodePoint() int32 { return int32(C.BinaryenStringNewFromCodePoint()) }
func cgoStringMeasureUTF8() int32   { return int32(C.BinaryenStringMeasureUTF8()) }
func cgoStringMeasureWTF16() int32  { return int32(C.BinaryenStringMeasureWTF16()) }
func cgoStringEncodeLossyUTF8Array() int32 { return int32(C.BinaryenStringEncodeLossyUTF8Array()) }
func cgoStringEncodeWTF16Array() int32 { return int32(C.BinaryenStringEncodeWTF16Array()) }
func cgoStringEqEqual() int32      { return int32(C.BinaryenStringEqEqual()) }
func cgoStringEqCompare() int32    { return int32(C.BinaryenStringEqCompare()) }

// ArrayFill, ArrayInitData, ArrayInitElem are not available in this binaryen version.
// They remain as stubs (return 0) in the bridge.

// --- ExpressionId constants ---

func cgoInvalidId() int  { return int(C.BinaryenInvalidId()) }
func cgoNopId() int      { return int(C.BinaryenNopId()) }
func cgoBlockId() int    { return int(C.BinaryenBlockId()) }
func cgoIfId() int       { return int(C.BinaryenIfId()) }
func cgoLoopId() int     { return int(C.BinaryenLoopId()) }
func cgoBreakId() int    { return int(C.BinaryenBreakId()) }
func cgoSwitchId() int   { return int(C.BinaryenSwitchId()) }
func cgoCallId() int     { return int(C.BinaryenCallId()) }
func cgoCallIndirectId() int { return int(C.BinaryenCallIndirectId()) }
func cgoLocalGetId() int { return int(C.BinaryenLocalGetId()) }
func cgoLocalSetId() int { return int(C.BinaryenLocalSetId()) }
func cgoGlobalGetId() int { return int(C.BinaryenGlobalGetId()) }
func cgoGlobalSetId() int { return int(C.BinaryenGlobalSetId()) }
func cgoLoadId() int     { return int(C.BinaryenLoadId()) }
func cgoStoreId() int    { return int(C.BinaryenStoreId()) }
func cgoConstId() int    { return int(C.BinaryenConstId()) }
func cgoUnaryId() int    { return int(C.BinaryenUnaryId()) }
func cgoBinaryId() int   { return int(C.BinaryenBinaryId()) }
func cgoSelectId() int   { return int(C.BinaryenSelectId()) }
func cgoDropId() int     { return int(C.BinaryenDropId()) }
func cgoReturnId() int   { return int(C.BinaryenReturnId()) }
func cgoMemorySizeId() int { return int(C.BinaryenMemorySizeId()) }
func cgoMemoryGrowId() int { return int(C.BinaryenMemoryGrowId()) }
func cgoUnreachableId() int { return int(C.BinaryenUnreachableId()) }

// --- Function getters ---

func cgoGetFunction(module uintptr, name unsafe.Pointer) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenGetFunction(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
	)))
}

func cgoRemoveFunction(module uintptr, name unsafe.Pointer) {
	C.BinaryenRemoveFunction(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
	)
}

func cgoGetNumFunctions(module uintptr) int {
	return int(C.BinaryenGetNumFunctions(C.BinaryenModuleRef(unsafe.Pointer(module))))
}

func cgoGetFunctionByIndex(module uintptr, index int) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenGetFunctionByIndex(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenIndex(index),
	)))
}

func cgoFunctionGetName(fn uintptr) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenFunctionGetName(C.BinaryenFunctionRef(unsafe.Pointer(fn))))
}

func cgoFunctionGetParams(fn uintptr) uintptr {
	return uintptr(C.BinaryenFunctionGetParams(C.BinaryenFunctionRef(unsafe.Pointer(fn))))
}

func cgoFunctionGetResults(fn uintptr) uintptr {
	return uintptr(C.BinaryenFunctionGetResults(C.BinaryenFunctionRef(unsafe.Pointer(fn))))
}

func cgoFunctionGetNumVars(fn uintptr) int {
	return int(C.BinaryenFunctionGetNumVars(C.BinaryenFunctionRef(unsafe.Pointer(fn))))
}

func cgoFunctionGetVar(fn uintptr, index int) uintptr {
	return uintptr(C.BinaryenFunctionGetVar(C.BinaryenFunctionRef(unsafe.Pointer(fn)), C.BinaryenIndex(index)))
}

func cgoFunctionGetBody(fn uintptr) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenFunctionGetBody(C.BinaryenFunctionRef(unsafe.Pointer(fn)))))
}

func cgoFunctionSetBody(fn uintptr, body uintptr) {
	C.BinaryenFunctionSetBody(
		C.BinaryenFunctionRef(unsafe.Pointer(fn)),
		C.BinaryenExpressionRef(unsafe.Pointer(body)),
	)
}

func cgoFunctionGetNumLocals(fn uintptr) int {
	return int(C.BinaryenFunctionGetNumLocals(C.BinaryenFunctionRef(unsafe.Pointer(fn))))
}

func cgoFunctionHasLocalName(fn uintptr, index int) bool {
	return bool(C.BinaryenFunctionHasLocalName(C.BinaryenFunctionRef(unsafe.Pointer(fn)), C.BinaryenIndex(index)))
}

func cgoFunctionGetLocalName(fn uintptr, index int) unsafe.Pointer {
	return unsafe.Pointer(C.BinaryenFunctionGetLocalName(C.BinaryenFunctionRef(unsafe.Pointer(fn)), C.BinaryenIndex(index)))
}

func cgoFunctionSetLocalName(fn uintptr, index int, name unsafe.Pointer) {
	C.BinaryenFunctionSetLocalName(C.BinaryenFunctionRef(unsafe.Pointer(fn)), C.BinaryenIndex(index), (*C.char)(name))
}

// --- Global getters ---

func cgoGetGlobal(module uintptr, name unsafe.Pointer) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenGetGlobal(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
	)))
}

func cgoRemoveGlobal(module uintptr, name unsafe.Pointer) {
	C.BinaryenRemoveGlobal(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
	)
}

func cgoGetNumGlobals(module uintptr) int {
	return int(C.BinaryenGetNumGlobals(C.BinaryenModuleRef(unsafe.Pointer(module))))
}

// --- Export getters ---

func cgoGetExport(module uintptr, externalName unsafe.Pointer) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenGetExport(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(externalName),
	)))
}

func cgoRemoveExport(module uintptr, externalName unsafe.Pointer) {
	C.BinaryenRemoveExport(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(externalName),
	)
}

func cgoGetNumExports(module uintptr) int {
	return int(C.BinaryenGetNumExports(C.BinaryenModuleRef(unsafe.Pointer(module))))
}

// --- Tag getters ---

func cgoGetTag(module uintptr, name unsafe.Pointer) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenGetTag(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
	)))
}

func cgoRemoveTag(module uintptr, name unsafe.Pointer) {
	C.BinaryenRemoveTag(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
	)
}

// --- Table getters ---

func cgoGetTable(module uintptr, name unsafe.Pointer) uintptr {
	return uintptr(unsafe.Pointer(C.BinaryenGetTable(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
	)))
}

func cgoRemoveTable(module uintptr, name unsafe.Pointer) {
	C.BinaryenRemoveTable(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(name),
	)
}

func cgoGetNumTables(module uintptr) int {
	return int(C.BinaryenGetNumTables(C.BinaryenModuleRef(unsafe.Pointer(module))))
}

// --- Memory getters ---

func cgoMemoryGetInitial(module uintptr, name unsafe.Pointer) uint32 {
	return uint32(C.BinaryenMemoryGetInitial(C.BinaryenModuleRef(unsafe.Pointer(module)), (*C.char)(name)))
}

func cgoMemoryHasMax(module uintptr, name unsafe.Pointer) bool {
	return bool(C.BinaryenMemoryHasMax(C.BinaryenModuleRef(unsafe.Pointer(module)), (*C.char)(name)))
}

func cgoMemoryGetMax(module uintptr, name unsafe.Pointer) uint32 {
	return uint32(C.BinaryenMemoryGetMax(C.BinaryenModuleRef(unsafe.Pointer(module)), (*C.char)(name)))
}

func cgoMemoryIsShared(module uintptr, name unsafe.Pointer) bool {
	return bool(C.BinaryenMemoryIsShared(C.BinaryenModuleRef(unsafe.Pointer(module)), (*C.char)(name)))
}

func cgoMemoryIs64(module uintptr, name unsafe.Pointer) bool {
	return bool(C.BinaryenMemoryIs64(C.BinaryenModuleRef(unsafe.Pointer(module)), (*C.char)(name)))
}

// --- TupleMake / TupleExtract ---

func cgoTupleMake(module uintptr, operands []uintptr) uintptr {
	if len(operands) == 0 {
		return 0
	}
	cArr := make([]C.BinaryenExpressionRef, len(operands))
	for i, o := range operands {
		cArr[i] = C.BinaryenExpressionRef(unsafe.Pointer(o))
	}
	ref := C.BinaryenTupleMake(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		&cArr[0],
		C.BinaryenIndex(len(operands)),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoTupleExtract(module uintptr, tuple uintptr, index int) uintptr {
	ref := C.BinaryenTupleExtract(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenExpressionRef(unsafe.Pointer(tuple)),
		C.BinaryenIndex(index),
	)
	return uintptr(unsafe.Pointer(ref))
}

// --- Pop ---

func cgoPop(module uintptr, typ uintptr) uintptr {
	ref := C.BinaryenPop(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenType(typ),
	)
	return uintptr(unsafe.Pointer(ref))
}

// ThrowRef is not available in this binaryen version.
// It remains as a stub (return 0) in the bridge.

// --- ReturnCall / ReturnCallIndirect ---

func cgoReturnCall(module uintptr, target unsafe.Pointer, operands []uintptr, returnType uintptr) uintptr {
	var cOperands *C.BinaryenExpressionRef
	if len(operands) > 0 {
		cArr := make([]C.BinaryenExpressionRef, len(operands))
		for i, o := range operands {
			cArr[i] = C.BinaryenExpressionRef(unsafe.Pointer(o))
		}
		cOperands = &cArr[0]
	}
	ref := C.BinaryenReturnCall(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(target),
		cOperands,
		C.BinaryenIndex(len(operands)),
		C.BinaryenType(returnType),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoReturnCallIndirect(module uintptr, table unsafe.Pointer, target uintptr, operands []uintptr, params, results uintptr) uintptr {
	var cOperands *C.BinaryenExpressionRef
	if len(operands) > 0 {
		cArr := make([]C.BinaryenExpressionRef, len(operands))
		for i, o := range operands {
			cArr[i] = C.BinaryenExpressionRef(unsafe.Pointer(o))
		}
		cOperands = &cArr[0]
	}
	ref := C.BinaryenReturnCallIndirect(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		(*C.char)(table),
		C.BinaryenExpressionRef(unsafe.Pointer(target)),
		cOperands,
		C.BinaryenIndex(len(operands)),
		C.BinaryenType(params),
		C.BinaryenType(results),
	)
	return uintptr(unsafe.Pointer(ref))
}

// --- MemoryCopy / MemoryFill ---

func cgoMemoryCopy(module uintptr, dest, source, size uintptr, destMemory, sourceMemory unsafe.Pointer) uintptr {
	ref := C.BinaryenMemoryCopy(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenExpressionRef(unsafe.Pointer(dest)),
		C.BinaryenExpressionRef(unsafe.Pointer(source)),
		C.BinaryenExpressionRef(unsafe.Pointer(size)),
		(*C.char)(destMemory),
		(*C.char)(sourceMemory),
	)
	return uintptr(unsafe.Pointer(ref))
}

func cgoMemoryFill(module uintptr, dest, value, size uintptr, memoryName unsafe.Pointer) uintptr {
	ref := C.BinaryenMemoryFill(
		C.BinaryenModuleRef(unsafe.Pointer(module)),
		C.BinaryenExpressionRef(unsafe.Pointer(dest)),
		C.BinaryenExpressionRef(unsafe.Pointer(value)),
		C.BinaryenExpressionRef(unsafe.Pointer(size)),
		(*C.char)(memoryName),
	)
	return uintptr(unsafe.Pointer(ref))
}

// Utility: CString helper (caller must free)
func cgoCString(s string) unsafe.Pointer {
	return unsafe.Pointer(C.CString(s))
}

func cgoFree(ptr unsafe.Pointer) {
	C.free(ptr)
}
