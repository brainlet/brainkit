package asembed

import (
	"unsafe"

	quickjs "github.com/buke/quickjs-go"
)

func registerRelooperImpls(ctx *quickjs.Context, lm *LinearMemory) {
	setFunc(ctx, "_RelooperCreate", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoRelooperCreate(argU(args, 0)))
	})
	setFunc(ctx, "_RelooperAddBlock", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoRelooperAddBlock(argU(a, 0), argU(a, 1)))
	})
	setFunc(ctx, "_RelooperAddBranch", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoRelooperAddBranch(argU(a, 0), argU(a, 1), argU(a, 2), argU(a, 3))
		return retVoid(c)
	})
	setFunc(ctx, "_RelooperAddBlockWithSwitch", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoRelooperAddBlockWithSwitch(argU(a, 0), argU(a, 1), argU(a, 2)))
	})
	setFunc(ctx, "_RelooperAddBranchForSwitch", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
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
		return retVoid(c)
	})
	setFunc(ctx, "_RelooperRenderAndDispose", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoRelooperRenderAndDispose(argU(a, 0), argU(a, 1), argU32(a, 2)))
	})
}

func registerExpressionRunnerImpls(ctx *quickjs.Context, lm *LinearMemory) {
	setFunc(ctx, "_ExpressionRunnerCreate", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoExpressionRunnerCreate(argU(a, 0), argU32(a, 1), argU32(a, 2), argU32(a, 3)))
	})
	setFunc(ctx, "_ExpressionRunnerSetLocalValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retBool(c, cgoExpressionRunnerSetLocalValue(argU(a, 0), argU32(a, 1), argU(a, 2)))
	})
	setFunc(ctx, "_ExpressionRunnerSetGlobalValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		runner := argU(a, 0)
		namePtr := argI(a, 1)
		value := argU(a, 2)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retBool(c, cgoExpressionRunnerSetGlobalValue(runner, name, value))
	})
	setFunc(ctx, "_ExpressionRunnerRunAndDispose", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoExpressionRunnerRunAndDispose(argU(a, 0), argU(a, 1)))
	})
}

func registerTypeBuilderImpls(ctx *quickjs.Context, lm *LinearMemory) {
	setFunc(ctx, "_TypeBuilderCreate", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoTypeBuilderCreate(argI(args, 0)))
	})
	setFunc(ctx, "_TypeBuilderGrow", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoTypeBuilderGrow(argU(a, 0), argI(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_TypeBuilderGetSize", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retI(c, cgoTypeBuilderGetSize(argU(args, 0)))
	})
	setFunc(ctx, "_TypeBuilderSetSignatureType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoTypeBuilderSetSignatureType(argU(a, 0), argI(a, 1), argU(a, 2), argU(a, 3))
		return retVoid(c)
	})
	setFunc(ctx, "_TypeBuilderSetStructType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
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
		return retVoid(c)
	})
	setFunc(ctx, "_TypeBuilderSetArrayType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoTypeBuilderSetArrayType(argU(a, 0), argI(a, 1), argU(a, 2), argU32(a, 3), argBool(a, 4))
		return retVoid(c)
	})
	setFunc(ctx, "_TypeBuilderGetTempHeapType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoTypeBuilderGetTempHeapType(argU(a, 0), argI(a, 1)))
	})
	setFunc(ctx, "_TypeBuilderGetTempTupleType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		builder := argU(a, 0)
		typesPtr := argI(a, 1)
		numTypes := argI(a, 2)
		types := readPtrArray(lm, typesPtr, numTypes)
		return retF(c, cgoTypeBuilderGetTempTupleType(builder, types))
	})
	setFunc(ctx, "_TypeBuilderGetTempRefType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoTypeBuilderGetTempRefType(argU(a, 0), argU(a, 1), argBool(a, 2)))
	})
	setFunc(ctx, "_TypeBuilderSetSubType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoTypeBuilderSetSubType(argU(a, 0), argI(a, 1), argU(a, 2))
		return retVoid(c)
	})
	setFunc(ctx, "_TypeBuilderSetOpen", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoTypeBuilderSetOpen(argU(a, 0), argI(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_TypeBuilderCreateRecGroup", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoTypeBuilderCreateRecGroup(argU(a, 0), argI(a, 1), argI(a, 2))
		return retVoid(c)
	})
	setFunc(ctx, "_TypeBuilderBuildAndDispose", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
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
		return retBool(c, ok)
	})
}

func registerGCImpls(ctx *quickjs.Context, lm *LinearMemory) {
	setFunc(ctx, "_BinaryenRefI31", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoRefI31(argU(a, 0), argU(a, 1)))
	})
	setFunc(ctx, "_BinaryenI31Get", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoI31Get(argU(a, 0), argU(a, 1), argBool(a, 2)))
	})
	setFunc(ctx, "_BinaryenRefTest", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoRefTest(argU(a, 0), argU(a, 1), argU(a, 2)))
	})
	setFunc(ctx, "_BinaryenRefCast", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoRefCast(argU(a, 0), argU(a, 1), argU(a, 2)))
	})
	setFunc(ctx, "_BinaryenBrOn", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		op := argI32(a, 1)
		namePtr := argI(a, 2)
		ref := argU(a, 3)
		castType := argU(a, 4)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(c, cgoBrOn(module, op, name, ref, castType))
	})
	setFunc(ctx, "_BinaryenCallRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		target := argU(a, 1)
		operandsPtr := argI(a, 2)
		numOperands := argI(a, 3)
		typ := argU(a, 4)
		isReturn := argBool(a, 5)
		operands := readPtrArray(lm, operandsPtr, numOperands)
		return retF(c, cgoCallRef(module, target, operands, typ, isReturn))
	})
	setFunc(ctx, "_BinaryenStructNew", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		operandsPtr := argI(a, 1)
		numOperands := argI(a, 2)
		typ := argU(a, 3)
		operands := readPtrArray(lm, operandsPtr, numOperands)
		return retF(c, cgoStructNew(module, operands, typ))
	})
	setFunc(ctx, "_BinaryenStructGet", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStructGet(argU(a, 0), argU32(a, 1), argU(a, 2), argU(a, 3), argBool(a, 4)))
	})
	setFunc(ctx, "_BinaryenStructSet", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStructSet(argU(a, 0), argU32(a, 1), argU(a, 2), argU(a, 3)))
	})
	setFunc(ctx, "_BinaryenArrayNew", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoArrayNew(argU(a, 0), argU(a, 1), argU(a, 2), argU(a, 3)))
	})
	setFunc(ctx, "_BinaryenArrayNewFixed", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		typ := argU(a, 1)
		valuesPtr := argI(a, 2)
		numValues := argI(a, 3)
		values := readPtrArray(lm, valuesPtr, numValues)
		return retF(c, cgoArrayNewFixed(module, typ, values))
	})
	setFunc(ctx, "_BinaryenArrayGet", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoArrayGet(argU(a, 0), argU(a, 1), argU(a, 2), argU(a, 3), argBool(a, 4)))
	})
	setFunc(ctx, "_BinaryenArraySet", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoArraySet(argU(a, 0), argU(a, 1), argU(a, 2), argU(a, 3)))
	})
	setFunc(ctx, "_BinaryenArrayLen", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoArrayLen(argU(a, 0), argU(a, 1)))
	})
	setFunc(ctx, "_BinaryenArrayCopy", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoArrayCopy(argU(a, 0), argU(a, 1), argU(a, 2), argU(a, 3), argU(a, 4), argU(a, 5)))
	})
	// ArrayFill, ArrayInitData, ArrayInitElem — not in this binaryen version, stay as stubs
}

func registerStringImpls(ctx *quickjs.Context, lm *LinearMemory) {
	setFunc(ctx, "_BinaryenStringNew", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStringNew(argU(a, 0), argI32(a, 1), argU(a, 2), argU(a, 3), argU(a, 4)))
	})
	setFunc(ctx, "_BinaryenStringConst", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(c, cgoStringConst(module, name))
	})
	setFunc(ctx, "_BinaryenStringMeasure", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStringMeasure(argU(a, 0), argI32(a, 1), argU(a, 2)))
	})
	setFunc(ctx, "_BinaryenStringEncode", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStringEncode(argU(a, 0), argI32(a, 1), argU(a, 2), argU(a, 3), argU(a, 4)))
	})
	setFunc(ctx, "_BinaryenStringConcat", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStringConcat(argU(a, 0), argU(a, 1), argU(a, 2)))
	})
	setFunc(ctx, "_BinaryenStringEq", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStringEq(argU(a, 0), argI32(a, 1), argU(a, 2), argU(a, 3)))
	})
	setFunc(ctx, "_BinaryenStringWTF16Get", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStringWTF16Get(argU(a, 0), argU(a, 1), argU(a, 2)))
	})
	setFunc(ctx, "_BinaryenStringSliceWTF", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStringSliceWTF(argU(a, 0), argU(a, 1), argU(a, 2), argU(a, 3)))
	})
}

func registerExpressionGetterSetterImpls(ctx *quickjs.Context, lm *LinearMemory) {
	// Expression getter/setters are numerous (300+) but not needed for basic compilation.
	// The AS compiler creates expressions via constructors and doesn't read back properties
	// during the compile() -> optimize() -> validate() flow.
	// These all remain as stubs (returning 0) from RegisterBinaryenBridge.
	// We'll implement them on-demand as needed.
}

func registerMiscImpls(ctx *quickjs.Context, lm *LinearMemory) {
	// ModuleParse, ModuleRead, ModuleReadWithFeatures, ModuleInterpret,
	// ModuleAddDebugInfoFileName, ModuleGetDebugInfoFileName,
	// ModuleSetTypeName, ModuleSetFieldName,
	// GetStart, SizeofAllocateAndWriteResult,
	// GetNumMemorySegments, GetMemorySegmentByteOffset, GetMemorySegmentByteLength,
	// CopyMemorySegmentData, GetExportByIndex, GetGlobalByIndex, GetTableByIndex,
	// etc. — all remain as stubs from RegisterBinaryenBridge.
	// Implement on-demand as the compilation flow requires them.
	setFunc(ctx, "_BinaryenModuleSetTypeName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		// Used by AS compiler for debug info — safe to ignore
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenModuleSetFieldName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retVoid(c)
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
