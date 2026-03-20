package asembed

import (
	"unsafe"

	quickjs "github.com/buke/quickjs-go"
)

func registerTypeImpls(ctx *quickjs.Context, lm *LinearMemory) {
	setFunc(ctx, "_BinaryenTypeCreate", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		typesPtr := argI(args, 0)
		numTypes := argI(args, 1)
		types := readPtrArray(lm, typesPtr, numTypes)
		return retF(c, cgoTypeCreate(types))
	})
	setFunc(ctx, "_BinaryenTypeArity", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retI(c, cgoTypeArity(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenTypeExpand", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		t := argU(args, 0)
		outPtr := argI(args, 1)
		arity := cgoTypeArity(t)
		buf := make([]uintptr, arity)
		cgoTypeExpand(t, buf)
		for i := 0; i < arity; i++ {
			lm.I32StorePtr(outPtr+i*4, uint64(buf[i]))
		}
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenTypeGetHeapType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoTypeGetHeapType(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenTypeFromHeapType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoTypeFromHeapType(argU(args, 0), argBool(args, 1)))
	})
	setFunc(ctx, "_BinaryenTypeIsNullable", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retBool(c, cgoTypeIsNullable(argU(args, 0)))
	})
	// No-arg type constants
	setFunc(ctx, "_BinaryenTypeFuncref", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoTypeFuncref())
	})
	setFunc(ctx, "_BinaryenTypeExternref", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoTypeExternref())
	})
	setFunc(ctx, "_BinaryenTypeAnyref", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoTypeAnyref())
	})
	setFunc(ctx, "_BinaryenTypeEqref", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoTypeEqref())
	})
	setFunc(ctx, "_BinaryenTypeI31ref", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoTypeI31ref())
	})
	setFunc(ctx, "_BinaryenTypeStructref", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoTypeStructref())
	})
	setFunc(ctx, "_BinaryenTypeArrayref", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoTypeArrayref())
	})
	setFunc(ctx, "_BinaryenTypeStringref", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoTypeStringref())
	})
	setFunc(ctx, "_BinaryenTypeNullref", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoTypeNullref())
	})
	setFunc(ctx, "_BinaryenTypeNullExternref", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoTypeNullExternref())
	})
	setFunc(ctx, "_BinaryenTypeNullFuncref", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoTypeNullFuncref())
	})
}

func registerHeapTypeImpls(ctx *quickjs.Context, lm *LinearMemory) {
	setFunc(ctx, "_BinaryenHeapTypeFunc", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoHeapTypeFunc())
	})
	setFunc(ctx, "_BinaryenHeapTypeExt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoHeapTypeExt())
	})
	setFunc(ctx, "_BinaryenHeapTypeAny", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoHeapTypeAny())
	})
	setFunc(ctx, "_BinaryenHeapTypeEq", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoHeapTypeEq())
	})
	setFunc(ctx, "_BinaryenHeapTypeI31", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoHeapTypeI31())
	})
	setFunc(ctx, "_BinaryenHeapTypeStruct", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoHeapTypeStruct())
	})
	setFunc(ctx, "_BinaryenHeapTypeArray", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoHeapTypeArray())
	})
	setFunc(ctx, "_BinaryenHeapTypeString", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoHeapTypeString())
	})
	setFunc(ctx, "_BinaryenHeapTypeNone", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoHeapTypeNone())
	})
	setFunc(ctx, "_BinaryenHeapTypeNoext", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoHeapTypeNoext())
	})
	setFunc(ctx, "_BinaryenHeapTypeNofunc", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoHeapTypeNofunc())
	})
	setFunc(ctx, "_BinaryenHeapTypeIsBasic", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retBool(c, cgoHeapTypeIsBasic(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenHeapTypeIsSignature", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retBool(c, cgoHeapTypeIsSignature(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenHeapTypeIsStruct", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retBool(c, cgoHeapTypeIsStruct(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenHeapTypeIsArray", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retBool(c, cgoHeapTypeIsArray(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenHeapTypeIsBottom", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retBool(c, cgoHeapTypeIsBottom(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenHeapTypeGetBottom", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoHeapTypeGetBottom(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenHeapTypeIsSubType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retBool(c, cgoHeapTypeIsSubType(argU(a, 0), argU(a, 1)))
	})
}

func registerStructArraySigTypeImpls(ctx *quickjs.Context, lm *LinearMemory) {
	setFunc(ctx, "_BinaryenStructTypeGetNumFields", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retI(c, cgoStructTypeGetNumFields(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenStructTypeGetFieldType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStructTypeGetFieldType(argU(a, 0), argI(a, 1)))
	})
	setFunc(ctx, "_BinaryenStructTypeGetFieldPackedType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retU32(c, cgoStructTypeGetFieldPackedType(argU(a, 0), argI(a, 1)))
	})
	setFunc(ctx, "_BinaryenStructTypeIsFieldMutable", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retBool(c, cgoStructTypeIsFieldMutable(argU(a, 0), argI(a, 1)))
	})
	setFunc(ctx, "_BinaryenArrayTypeGetElementType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoArrayTypeGetElementType(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenArrayTypeGetElementPackedType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retU32(c, cgoArrayTypeGetElementPackedType(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenArrayTypeIsElementMutable", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retBool(c, cgoArrayTypeIsElementMutable(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenSignatureTypeGetParams", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoSignatureTypeGetParams(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenSignatureTypeGetResults", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoSignatureTypeGetResults(argU(args, 0)))
	})
}

// suppress unused import warning
var _ = unsafe.Pointer(nil)
