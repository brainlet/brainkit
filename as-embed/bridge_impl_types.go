package asembed

import (
	"unsafe"

	"github.com/fastschema/qjs"
)

func registerTypeImpls(ctx *qjs.Context, lm *LinearMemory) {
	ctx.SetFunc("_BinaryenTypeCreate", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		typesPtr := argI(args, 0)
		numTypes := argI(args, 1)
		types := readPtrArray(lm, typesPtr, numTypes)
		return retF(this.Context(), cgoTypeCreate(types))
	})
	ctx.SetFunc("_BinaryenTypeArity", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		return retI(this.Context(), cgoTypeArity(argU(args, 0)))
	})
	ctx.SetFunc("_BinaryenTypeExpand", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		t := argU(args, 0)
		outPtr := argI(args, 1)
		arity := cgoTypeArity(t)
		buf := make([]uintptr, arity)
		cgoTypeExpand(t, buf)
		for i := 0; i < arity; i++ {
			lm.I32Store(outPtr+i*4, int(buf[i]))
		}
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenTypeGetHeapType", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		return retF(this.Context(), cgoTypeGetHeapType(argU(args, 0)))
	})
	ctx.SetFunc("_BinaryenTypeFromHeapType", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		return retF(this.Context(), cgoTypeFromHeapType(argU(args, 0), argBool(args, 1)))
	})
	ctx.SetFunc("_BinaryenTypeIsNullable", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		return retBool(this.Context(), cgoTypeIsNullable(argU(args, 0)))
	})
	// No-arg type constants
	ctx.SetFunc("_BinaryenTypeFuncref", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoTypeFuncref())
	})
	ctx.SetFunc("_BinaryenTypeExternref", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoTypeExternref())
	})
	ctx.SetFunc("_BinaryenTypeAnyref", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoTypeAnyref())
	})
	ctx.SetFunc("_BinaryenTypeEqref", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoTypeEqref())
	})
	ctx.SetFunc("_BinaryenTypeI31ref", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoTypeI31ref())
	})
	ctx.SetFunc("_BinaryenTypeStructref", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoTypeStructref())
	})
	ctx.SetFunc("_BinaryenTypeArrayref", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoTypeArrayref())
	})
	ctx.SetFunc("_BinaryenTypeStringref", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoTypeStringref())
	})
	ctx.SetFunc("_BinaryenTypeNullref", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoTypeNullref())
	})
	ctx.SetFunc("_BinaryenTypeNullExternref", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoTypeNullExternref())
	})
	ctx.SetFunc("_BinaryenTypeNullFuncref", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoTypeNullFuncref())
	})
}

func registerHeapTypeImpls(ctx *qjs.Context, lm *LinearMemory) {
	ctx.SetFunc("_BinaryenHeapTypeFunc", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoHeapTypeFunc())
	})
	ctx.SetFunc("_BinaryenHeapTypeExt", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoHeapTypeExt())
	})
	ctx.SetFunc("_BinaryenHeapTypeAny", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoHeapTypeAny())
	})
	ctx.SetFunc("_BinaryenHeapTypeEq", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoHeapTypeEq())
	})
	ctx.SetFunc("_BinaryenHeapTypeI31", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoHeapTypeI31())
	})
	ctx.SetFunc("_BinaryenHeapTypeStruct", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoHeapTypeStruct())
	})
	ctx.SetFunc("_BinaryenHeapTypeArray", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoHeapTypeArray())
	})
	ctx.SetFunc("_BinaryenHeapTypeString", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoHeapTypeString())
	})
	ctx.SetFunc("_BinaryenHeapTypeNone", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoHeapTypeNone())
	})
	ctx.SetFunc("_BinaryenHeapTypeNoext", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoHeapTypeNoext())
	})
	ctx.SetFunc("_BinaryenHeapTypeNofunc", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoHeapTypeNofunc())
	})
	ctx.SetFunc("_BinaryenHeapTypeIsBasic", func(this *qjs.This) (*qjs.Value, error) {
		return retBool(this.Context(), cgoHeapTypeIsBasic(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_BinaryenHeapTypeIsSignature", func(this *qjs.This) (*qjs.Value, error) {
		return retBool(this.Context(), cgoHeapTypeIsSignature(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_BinaryenHeapTypeIsStruct", func(this *qjs.This) (*qjs.Value, error) {
		return retBool(this.Context(), cgoHeapTypeIsStruct(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_BinaryenHeapTypeIsArray", func(this *qjs.This) (*qjs.Value, error) {
		return retBool(this.Context(), cgoHeapTypeIsArray(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_BinaryenHeapTypeIsBottom", func(this *qjs.This) (*qjs.Value, error) {
		return retBool(this.Context(), cgoHeapTypeIsBottom(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_BinaryenHeapTypeGetBottom", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoHeapTypeGetBottom(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_BinaryenHeapTypeIsSubType", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retBool(this.Context(), cgoHeapTypeIsSubType(argU(a, 0), argU(a, 1)))
	})
}

func registerStructArraySigTypeImpls(ctx *qjs.Context, lm *LinearMemory) {
	ctx.SetFunc("_BinaryenStructTypeGetNumFields", func(this *qjs.This) (*qjs.Value, error) {
		return retI(this.Context(), cgoStructTypeGetNumFields(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_BinaryenStructTypeGetFieldType", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStructTypeGetFieldType(argU(a, 0), argI(a, 1)))
	})
	ctx.SetFunc("_BinaryenStructTypeGetFieldPackedType", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retU32(this.Context(), cgoStructTypeGetFieldPackedType(argU(a, 0), argI(a, 1)))
	})
	ctx.SetFunc("_BinaryenStructTypeIsFieldMutable", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retBool(this.Context(), cgoStructTypeIsFieldMutable(argU(a, 0), argI(a, 1)))
	})
	ctx.SetFunc("_BinaryenArrayTypeGetElementType", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoArrayTypeGetElementType(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_BinaryenArrayTypeGetElementPackedType", func(this *qjs.This) (*qjs.Value, error) {
		return retU32(this.Context(), cgoArrayTypeGetElementPackedType(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_BinaryenArrayTypeIsElementMutable", func(this *qjs.This) (*qjs.Value, error) {
		return retBool(this.Context(), cgoArrayTypeIsElementMutable(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_BinaryenSignatureTypeGetParams", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoSignatureTypeGetParams(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_BinaryenSignatureTypeGetResults", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoSignatureTypeGetResults(argU(this.Args(), 0)))
	})
}

// suppress unused import warning
var _ = unsafe.Pointer(nil)
