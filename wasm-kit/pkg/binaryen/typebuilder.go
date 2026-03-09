// Ported from: src/glue/binaryen.d.ts (TypeBuilder section)
package binaryen

/*
#include "binaryen-c.h"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// ---------------------------------------------------------------------------
// TypeBuilder — recursive type construction
// ---------------------------------------------------------------------------

// TypeBuilderErrorReason identifies why a TypeBuilder build failed.
type TypeBuilderErrorReason = uint32

// TypeBuilder error reason constants.
func TypeBuilderErrorReasonSelfSupertype() TypeBuilderErrorReason {
	return TypeBuilderErrorReason(C.TypeBuilderErrorReasonSelfSupertype())
}
func TypeBuilderErrorReasonInvalidSupertype() TypeBuilderErrorReason {
	return TypeBuilderErrorReason(C.TypeBuilderErrorReasonInvalidSupertype())
}
func TypeBuilderErrorReasonForwardSupertypeReference() TypeBuilderErrorReason {
	return TypeBuilderErrorReason(C.TypeBuilderErrorReasonForwardSupertypeReference())
}
func TypeBuilderErrorReasonForwardChildReference() TypeBuilderErrorReason {
	return TypeBuilderErrorReason(C.TypeBuilderErrorReasonForwardChildReference())
}

// TypeBuilder wraps the Binaryen TypeBuilder for constructing recursive heap types.
type TypeBuilder struct {
	ref C.TypeBuilderRef
}

// NewTypeBuilder creates a type builder with the given initial table size.
func NewTypeBuilder(size Index) *TypeBuilder {
	return &TypeBuilder{ref: C.TypeBuilderCreate(C.BinaryenIndex(size))}
}

// Grow grows the builder's backing table by count slots.
func (tb *TypeBuilder) Grow(count Index) {
	C.TypeBuilderGrow(tb.ref, C.BinaryenIndex(count))
}

// GetSize returns the current size of the builder's table.
func (tb *TypeBuilder) GetSize() Index {
	return Index(C.TypeBuilderGetSize(tb.ref))
}

// SetSignatureType sets the heap type at index to a concrete signature type.
func (tb *TypeBuilder) SetSignatureType(index Index, paramTypes, resultTypes Type) {
	C.TypeBuilderSetSignatureType(tb.ref, C.BinaryenIndex(index),
		C.BinaryenType(paramTypes), C.BinaryenType(resultTypes))
}

// SetStructType sets the heap type at index to a concrete struct type.
func (tb *TypeBuilder) SetStructType(index Index, fieldTypes []Type, fieldPackedTypes []PackedType, fieldMutables []bool) {
	n := len(fieldTypes)
	if n == 0 {
		C.TypeBuilderSetStructType(tb.ref, C.BinaryenIndex(index), nil, nil, nil, 0)
		return
	}
	cTypes := make([]C.BinaryenType, n)
	for i, t := range fieldTypes {
		cTypes[i] = C.BinaryenType(t)
	}
	cPacked := make([]C.BinaryenPackedType, n)
	for i, p := range fieldPackedTypes {
		cPacked[i] = C.BinaryenPackedType(p)
	}
	cMut := make([]C.bool, n)
	for i, m := range fieldMutables {
		cMut[i] = cBool(m)
	}
	C.TypeBuilderSetStructType(tb.ref, C.BinaryenIndex(index),
		&cTypes[0], &cPacked[0], &cMut[0], C.int(n))
}

// SetArrayType sets the heap type at index to a concrete array type.
func (tb *TypeBuilder) SetArrayType(index Index, elementType Type, elementPackedType PackedType, elementMutable bool) {
	mut := C.int(0)
	if elementMutable {
		mut = 1
	}
	C.TypeBuilderSetArrayType(tb.ref, C.BinaryenIndex(index),
		C.BinaryenType(elementType), C.BinaryenPackedType(elementPackedType), mut)
}

// GetTempHeapType returns a temporary heap type at the given index.
// Temporary types may only be used within the type builder.
func (tb *TypeBuilder) GetTempHeapType(index Index) HeapType {
	return HeapType(C.TypeBuilderGetTempHeapType(tb.ref, C.BinaryenIndex(index)))
}

// GetTempTupleType returns a temporary tuple type owned by the builder.
func (tb *TypeBuilder) GetTempTupleType(types []Type) Type {
	if len(types) == 0 {
		return TypeNone()
	}
	cTypes := make([]C.BinaryenType, len(types))
	for i, t := range types {
		cTypes[i] = C.BinaryenType(t)
	}
	return Type(C.TypeBuilderGetTempTupleType(tb.ref, &cTypes[0], C.BinaryenIndex(len(types))))
}

// GetTempRefType returns a temporary reference type owned by the builder.
func (tb *TypeBuilder) GetTempRefType(heapType HeapType, nullable bool) Type {
	n := C.int(0)
	if nullable {
		n = 1
	}
	return Type(C.TypeBuilderGetTempRefType(tb.ref, C.BinaryenHeapType(heapType), n))
}

// SetSubType sets the type at index to be a subtype of the given super type.
func (tb *TypeBuilder) SetSubType(index Index, superType HeapType) {
	C.TypeBuilderSetSubType(tb.ref, C.BinaryenIndex(index), C.BinaryenHeapType(superType))
}

// SetOpen sets the type at index to be open (non-final).
func (tb *TypeBuilder) SetOpen(index Index) {
	C.TypeBuilderSetOpen(tb.ref, C.BinaryenIndex(index))
}

// CreateRecGroup creates a recursion group in the range [index, index+length).
func (tb *TypeBuilder) CreateRecGroup(index, length Index) {
	C.TypeBuilderCreateRecGroup(tb.ref, C.BinaryenIndex(index), C.BinaryenIndex(length))
}

// TypeBuilderResult holds the result of a TypeBuilder build.
type TypeBuilderResult struct {
	HeapTypes  []HeapType
	ErrorIndex Index
	ErrorReason TypeBuilderErrorReason
}

// BuildAndDispose builds the type hierarchy and disposes the builder.
// Returns the resulting heap types and nil error on success.
// On failure, returns an error with the index and reason.
func (tb *TypeBuilder) BuildAndDispose() ([]HeapType, error) {
	size := tb.GetSize()
	heapTypes := make([]C.BinaryenHeapType, size)
	var errorIndex C.BinaryenIndex
	var errorReason C.TypeBuilderErrorReason

	ok := C.TypeBuilderBuildAndDispose(tb.ref, &heapTypes[0], &errorIndex, &errorReason)
	if !goBool(ok) {
		return nil, fmt.Errorf("TypeBuilder error at index %d: reason %d", uint32(errorIndex), uint32(errorReason))
	}

	result := make([]HeapType, size)
	for i := range heapTypes {
		result[i] = HeapType(heapTypes[i])
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Module type naming
// ---------------------------------------------------------------------------

// SetTypeName sets the textual name for a compound heap type.
func (m *Module) SetTypeName(heapType HeapType, name string) {
	C.BinaryenModuleSetTypeName(m.ref, C.BinaryenHeapType(heapType), m.str(name))
}

// SetFieldName sets the field name for a struct heap type at the given index.
func (m *Module) SetFieldName(heapType HeapType, index Index, name string) {
	C.BinaryenModuleSetFieldName(m.ref, C.BinaryenHeapType(heapType), C.BinaryenIndex(index), m.str(name))
}

// keep unsafe import used
var _ = unsafe.Pointer(nil)
