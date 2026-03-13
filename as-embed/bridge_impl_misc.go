package asembed

import (
	"unsafe"

	"github.com/fastschema/qjs"
)

func registerRelooperImpls(ctx *qjs.Context, lm *LinearMemory) {
	ctx.SetFunc("_RelooperCreate", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoRelooperCreate(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_RelooperAddBlock", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoRelooperAddBlock(argU(a, 0), argU(a, 1)))
	})
	ctx.SetFunc("_RelooperAddBranch", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoRelooperAddBranch(argU(a, 0), argU(a, 1), argU(a, 2), argU(a, 3))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_RelooperAddBlockWithSwitch", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoRelooperAddBlockWithSwitch(argU(a, 0), argU(a, 1), argU(a, 2)))
	})
	ctx.SetFunc("_RelooperAddBranchForSwitch", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		from := argU(a, 0)
		to := argU(a, 1)
		indexesPtr := argI(a, 2)
		numIndexes := argI(a, 3)
		code := argU(a, 4)
		indexes := make([]uint32, numIndexes)
		for i := 0; i < numIndexes; i++ {
			indexes[i] = uint32(lm.I32Load(indexesPtr + i*4))
		}
		cgoRelooperAddBranchForSwitch(from, to, indexes, code)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_RelooperRenderAndDispose", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoRelooperRenderAndDispose(argU(a, 0), argU(a, 1), argU32(a, 2)))
	})
}

func registerExpressionRunnerImpls(ctx *qjs.Context, lm *LinearMemory) {
	ctx.SetFunc("_ExpressionRunnerCreate", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoExpressionRunnerCreate(argU(a, 0), argU32(a, 1), argU32(a, 2), argU32(a, 3)))
	})
	ctx.SetFunc("_ExpressionRunnerSetLocalValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retBool(this.Context(), cgoExpressionRunnerSetLocalValue(argU(a, 0), argU32(a, 1), argU(a, 2)))
	})
	ctx.SetFunc("_ExpressionRunnerSetGlobalValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		runner := argU(a, 0)
		namePtr := argI(a, 1)
		value := argU(a, 2)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retBool(this.Context(), cgoExpressionRunnerSetGlobalValue(runner, name, value))
	})
	ctx.SetFunc("_ExpressionRunnerRunAndDispose", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoExpressionRunnerRunAndDispose(argU(a, 0), argU(a, 1)))
	})
}

func registerTypeBuilderImpls(ctx *qjs.Context, lm *LinearMemory) {
	ctx.SetFunc("_TypeBuilderCreate", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoTypeBuilderCreate(argI(this.Args(), 0)))
	})
	ctx.SetFunc("_TypeBuilderGrow", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoTypeBuilderGrow(argU(a, 0), argI(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_TypeBuilderGetSize", func(this *qjs.This) (*qjs.Value, error) {
		return retI(this.Context(), cgoTypeBuilderGetSize(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_TypeBuilderSetSignatureType", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoTypeBuilderSetSignatureType(argU(a, 0), argI(a, 1), argU(a, 2), argU(a, 3))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_TypeBuilderSetStructType", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		builder := argU(a, 0)
		index := argI(a, 1)
		fieldTypesPtr := argI(a, 2)
		fieldPackedTypesPtr := argI(a, 3)
		fieldMutablesPtr := argI(a, 4)
		numFields := argI(a, 5)
		fieldTypes := readPtrArray(lm, fieldTypesPtr, numFields)
		fieldPackedTypes := make([]uint32, numFields)
		fieldMutables := make([]bool, numFields)
		for i := 0; i < numFields; i++ {
			fieldPackedTypes[i] = uint32(lm.I32Load(fieldPackedTypesPtr + i*4))
			fieldMutables[i] = lm.I32Load(fieldMutablesPtr+i*4) != 0
		}
		cgoTypeBuilderSetStructType(builder, index, fieldTypes, fieldPackedTypes, fieldMutables)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_TypeBuilderSetArrayType", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoTypeBuilderSetArrayType(argU(a, 0), argI(a, 1), argU(a, 2), argU32(a, 3), argBool(a, 4))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_TypeBuilderGetTempHeapType", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoTypeBuilderGetTempHeapType(argU(a, 0), argI(a, 1)))
	})
	ctx.SetFunc("_TypeBuilderGetTempTupleType", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		builder := argU(a, 0)
		typesPtr := argI(a, 1)
		numTypes := argI(a, 2)
		types := readPtrArray(lm, typesPtr, numTypes)
		return retF(this.Context(), cgoTypeBuilderGetTempTupleType(builder, types))
	})
	ctx.SetFunc("_TypeBuilderGetTempRefType", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoTypeBuilderGetTempRefType(argU(a, 0), argU(a, 1), argBool(a, 2)))
	})
	ctx.SetFunc("_TypeBuilderSetSubType", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoTypeBuilderSetSubType(argU(a, 0), argI(a, 1), argU(a, 2))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_TypeBuilderSetOpen", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoTypeBuilderSetOpen(argU(a, 0), argI(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_TypeBuilderCreateRecGroup", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoTypeBuilderCreateRecGroup(argU(a, 0), argI(a, 1), argI(a, 2))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_TypeBuilderBuildAndDispose", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		builder := argU(a, 0)
		outPtr := argI(a, 1)
		// errorIndexPtr := argI(a, 2) // not used for now
		size := cgoTypeBuilderGetSize(builder)
		heapTypes := make([]uintptr, size)
		ok := cgoTypeBuilderBuildAndDispose(builder, heapTypes)
		if ok && outPtr != 0 {
			for i := 0; i < size; i++ {
				lm.I32StorePtr(outPtr+i*4, uint64(heapTypes[i]))
			}
		}
		return retBool(this.Context(), ok)
	})
}

func registerGCImpls(ctx *qjs.Context, lm *LinearMemory) {
	ctx.SetFunc("_BinaryenRefI31", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoRefI31(argU(a, 0), argU(a, 1)))
	})
	ctx.SetFunc("_BinaryenI31Get", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoI31Get(argU(a, 0), argU(a, 1), argBool(a, 2)))
	})
	ctx.SetFunc("_BinaryenRefTest", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoRefTest(argU(a, 0), argU(a, 1), argU(a, 2)))
	})
	ctx.SetFunc("_BinaryenRefCast", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoRefCast(argU(a, 0), argU(a, 1), argU(a, 2)))
	})
	ctx.SetFunc("_BinaryenBrOn", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		op := argI32(a, 1)
		namePtr := argI(a, 2)
		ref := argU(a, 3)
		castType := argU(a, 4)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(this.Context(), cgoBrOn(module, op, name, ref, castType))
	})
	ctx.SetFunc("_BinaryenCallRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		target := argU(a, 1)
		operandsPtr := argI(a, 2)
		numOperands := argI(a, 3)
		typ := argU(a, 4)
		isReturn := argBool(a, 5)
		operands := readPtrArray(lm, operandsPtr, numOperands)
		return retF(this.Context(), cgoCallRef(module, target, operands, typ, isReturn))
	})
	ctx.SetFunc("_BinaryenStructNew", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		operandsPtr := argI(a, 1)
		numOperands := argI(a, 2)
		typ := argU(a, 3)
		operands := readPtrArray(lm, operandsPtr, numOperands)
		return retF(this.Context(), cgoStructNew(module, operands, typ))
	})
	ctx.SetFunc("_BinaryenStructGet", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStructGet(argU(a, 0), argU32(a, 1), argU(a, 2), argU(a, 3), argBool(a, 4)))
	})
	ctx.SetFunc("_BinaryenStructSet", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStructSet(argU(a, 0), argU32(a, 1), argU(a, 2), argU(a, 3)))
	})
	ctx.SetFunc("_BinaryenArrayNew", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoArrayNew(argU(a, 0), argU(a, 1), argU(a, 2), argU(a, 3)))
	})
	ctx.SetFunc("_BinaryenArrayNewFixed", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		typ := argU(a, 1)
		valuesPtr := argI(a, 2)
		numValues := argI(a, 3)
		values := readPtrArray(lm, valuesPtr, numValues)
		return retF(this.Context(), cgoArrayNewFixed(module, typ, values))
	})
	ctx.SetFunc("_BinaryenArrayGet", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoArrayGet(argU(a, 0), argU(a, 1), argU(a, 2), argU(a, 3), argBool(a, 4)))
	})
	ctx.SetFunc("_BinaryenArraySet", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoArraySet(argU(a, 0), argU(a, 1), argU(a, 2), argU(a, 3)))
	})
	ctx.SetFunc("_BinaryenArrayLen", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoArrayLen(argU(a, 0), argU(a, 1)))
	})
	ctx.SetFunc("_BinaryenArrayCopy", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoArrayCopy(argU(a, 0), argU(a, 1), argU(a, 2), argU(a, 3), argU(a, 4), argU(a, 5)))
	})
	// ArrayFill, ArrayInitData, ArrayInitElem — not in this binaryen version, stay as stubs
}

func registerStringImpls(ctx *qjs.Context, lm *LinearMemory) {
	ctx.SetFunc("_BinaryenStringNew", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStringNew(argU(a, 0), argI32(a, 1), argU(a, 2), argU(a, 3), argU(a, 4)))
	})
	ctx.SetFunc("_BinaryenStringConst", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(this.Context(), cgoStringConst(module, name))
	})
	ctx.SetFunc("_BinaryenStringMeasure", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStringMeasure(argU(a, 0), argI32(a, 1), argU(a, 2)))
	})
	ctx.SetFunc("_BinaryenStringEncode", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStringEncode(argU(a, 0), argI32(a, 1), argU(a, 2), argU(a, 3), argU(a, 4)))
	})
	ctx.SetFunc("_BinaryenStringConcat", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStringConcat(argU(a, 0), argU(a, 1), argU(a, 2)))
	})
	ctx.SetFunc("_BinaryenStringEq", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStringEq(argU(a, 0), argI32(a, 1), argU(a, 2), argU(a, 3)))
	})
	ctx.SetFunc("_BinaryenStringWTF16Get", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStringWTF16Get(argU(a, 0), argU(a, 1), argU(a, 2)))
	})
	ctx.SetFunc("_BinaryenStringSliceWTF", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStringSliceWTF(argU(a, 0), argU(a, 1), argU(a, 2), argU(a, 3)))
	})
}

func registerExpressionGetterSetterImpls(ctx *qjs.Context, lm *LinearMemory) {
	// Expression getter/setters are numerous (300+) but not needed for basic compilation.
	// The AS compiler creates expressions via constructors and doesn't read back properties
	// during the compile() -> optimize() -> validate() flow.
	// These all remain as stubs (returning 0) from RegisterBinaryenBridge.
	// We'll implement them on-demand as needed.
}

func registerMiscImpls(ctx *qjs.Context, lm *LinearMemory) {
	// ModuleParse, ModuleRead, ModuleReadWithFeatures, ModuleInterpret,
	// ModuleAddDebugInfoFileName, ModuleGetDebugInfoFileName,
	// ModuleSetTypeName, ModuleSetFieldName,
	// GetStart, SizeofAllocateAndWriteResult,
	// GetNumMemorySegments, GetMemorySegmentByteOffset, GetMemorySegmentByteLength,
	// CopyMemorySegmentData, GetExportByIndex, GetGlobalByIndex, GetTableByIndex,
	// etc. — all remain as stubs from RegisterBinaryenBridge.
	// Implement on-demand as the compilation flow requires them.
	ctx.SetFunc("_BinaryenModuleSetTypeName", func(this *qjs.This) (*qjs.Value, error) {
		// Used by AS compiler for debug info — safe to ignore
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenModuleSetFieldName", func(this *qjs.This) (*qjs.Value, error) {
		return retVoid(this.Context())
	})
}

// cgoGoString converts a C *char to a Go string without importing C.
// It reads bytes until null terminator.
func cgoGoString(p unsafe.Pointer) string {
	if p == nil {
		return ""
	}
	var buf []byte
	for ptr := unsafe.Pointer(p); ; {
		b := *(*byte)(ptr)
		if b == 0 {
			break
		}
		buf = append(buf, b)
		ptr = unsafe.Add(ptr, 1)
	}
	return string(buf)
}

var _ = unsafe.Pointer(nil)
