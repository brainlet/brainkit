package asembed

import (
	"unsafe"

	quickjs "github.com/buke/quickjs-go"
)

func registerAllConstructorImpls(ctx *quickjs.Context, lm *LinearMemory) {
	// --- Atomic expression constructors ---

	setFunc(ctx, "_BinaryenAtomicRMW", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		op := argI(a, 1)
		bytes := argI(a, 2)
		offset := argI(a, 3)
		ptr := argU(a, 4)
		value := argU(a, 5)
		typ := argU(a, 6)
		memNamePtr := argI(a, 7)
		var memName unsafe.Pointer
		if memNamePtr != 0 {
			memName = cgoCString(lm.ReadString(memNamePtr))
			defer cgoFree(unsafe.Pointer(memName))
		}
		return retF(c, cgoAtomicRMW(module, op, bytes, offset, ptr, value, typ, memName))
	})

	setFunc(ctx, "_BinaryenAtomicCmpxchg", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		bytes := argI(a, 1)
		offset := argI(a, 2)
		ptr := argU(a, 3)
		expected := argU(a, 4)
		replacement := argU(a, 5)
		typ := argU(a, 6)
		memNamePtr := argI(a, 7)
		var memName unsafe.Pointer
		if memNamePtr != 0 {
			memName = cgoCString(lm.ReadString(memNamePtr))
			defer cgoFree(unsafe.Pointer(memName))
		}
		return retF(c, cgoAtomicCmpxchg(module, bytes, offset, ptr, expected, replacement, typ, memName))
	})

	setFunc(ctx, "_BinaryenAtomicWait", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		ptr := argU(a, 1)
		expected := argU(a, 2)
		timeout := argU(a, 3)
		typ := argU(a, 4)
		memNamePtr := argI(a, 5)
		var memName unsafe.Pointer
		if memNamePtr != 0 {
			memName = cgoCString(lm.ReadString(memNamePtr))
			defer cgoFree(unsafe.Pointer(memName))
		}
		return retF(c, cgoAtomicWait(module, ptr, expected, timeout, typ, memName))
	})

	setFunc(ctx, "_BinaryenAtomicNotify", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		ptr := argU(a, 1)
		notifyCount := argU(a, 2)
		memNamePtr := argI(a, 3)
		var memName unsafe.Pointer
		if memNamePtr != 0 {
			memName = cgoCString(lm.ReadString(memNamePtr))
			defer cgoFree(unsafe.Pointer(memName))
		}
		return retF(c, cgoAtomicNotify(module, ptr, notifyCount, memName))
	})

	setFunc(ctx, "_BinaryenAtomicFence", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoAtomicFence(argU(args, 0)))
	})

	// --- SIMD expression constructors ---

	setFunc(ctx, "_BinaryenSIMDExtract", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		op := argI32(a, 1)
		vec := argU(a, 2)
		index := uint8(argI32(a, 3))
		return retF(c, cgoSIMDExtract(module, op, vec, index))
	})

	setFunc(ctx, "_BinaryenSIMDReplace", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		op := argI32(a, 1)
		vec := argU(a, 2)
		index := uint8(argI32(a, 3))
		value := argU(a, 4)
		return retF(c, cgoSIMDReplace(module, op, vec, index, value))
	})

	setFunc(ctx, "_BinaryenSIMDShuffle", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		left := argU(a, 1)
		right := argU(a, 2)
		maskPtr := argI(a, 3)
		var mask [16]byte
		if maskPtr != 0 {
			maskBytes := lm.ReadBytes(maskPtr, 16)
			copy(mask[:], maskBytes)
		}
		return retF(c, cgoSIMDShuffle(module, left, right, mask))
	})

	setFunc(ctx, "_BinaryenSIMDTernary", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		op := argI32(a, 1)
		va := argU(a, 2)
		vb := argU(a, 3)
		vc := argU(a, 4)
		return retF(c, cgoSIMDTernary(module, op, va, vb, vc))
	})

	setFunc(ctx, "_BinaryenSIMDShift", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		op := argI32(a, 1)
		vec := argU(a, 2)
		shift := argU(a, 3)
		return retF(c, cgoSIMDShift(module, op, vec, shift))
	})

	setFunc(ctx, "_BinaryenSIMDLoad", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		op := argI32(a, 1)
		offset := argU32(a, 2)
		align := argU32(a, 3)
		ptr := argU(a, 4)
		memNamePtr := argI(a, 5)
		var memName unsafe.Pointer
		if memNamePtr != 0 {
			memName = cgoCString(lm.ReadString(memNamePtr))
			defer cgoFree(unsafe.Pointer(memName))
		}
		return retF(c, cgoSIMDLoad(module, op, offset, align, ptr, memName))
	})

	setFunc(ctx, "_BinaryenSIMDLoadStoreLane", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		op := argI32(a, 1)
		offset := argU32(a, 2)
		align := argU32(a, 3)
		index := uint8(argI32(a, 4))
		ptr := argU(a, 5)
		vec := argU(a, 6)
		memNamePtr := argI(a, 7)
		var memName unsafe.Pointer
		if memNamePtr != 0 {
			memName = cgoCString(lm.ReadString(memNamePtr))
			defer cgoFree(unsafe.Pointer(memName))
		}
		return retF(c, cgoSIMDLoadStoreLane(module, op, offset, align, index, ptr, vec, memName))
	})

	// --- Bulk memory expression constructors ---

	setFunc(ctx, "_BinaryenDataDrop", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		segmentPtr := argI(a, 1)
		segment := cgoCString(lm.ReadString(segmentPtr))
		defer cgoFree(unsafe.Pointer(segment))
		return retF(c, cgoDataDrop(module, segment))
	})

	setFunc(ctx, "_BinaryenMemoryInit", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		segmentPtr := argI(a, 1)
		dest := argU(a, 2)
		offset := argU(a, 3)
		size := argU(a, 4)
		memNamePtr := argI(a, 5)
		segment := cgoCString(lm.ReadString(segmentPtr))
		defer cgoFree(unsafe.Pointer(segment))
		var memName unsafe.Pointer
		if memNamePtr != 0 {
			memName = cgoCString(lm.ReadString(memNamePtr))
			defer cgoFree(unsafe.Pointer(memName))
		}
		return retF(c, cgoMemoryInit(module, segment, dest, offset, size, memName))
	})

	// --- GC array expression constructors ---

	setFunc(ctx, "_BinaryenArrayNewData", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		typ := argU(a, 1)
		namePtr := argI(a, 2)
		offset := argU(a, 3)
		size := argU(a, 4)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(c, cgoArrayNewData(module, typ, name, offset, size))
	})

	// ArrayNewElem, ArrayFill, ArrayInitData, ArrayInitElem are not available in this
	// binaryen version. They remain as stubs (returning 0) from RegisterBinaryenBridge.
	// When the binaryen version is updated, these can be implemented.

	// --- Mutation operations: Block ---

	setFunc(ctx, "_BinaryenBlockAppendChild", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		idx := cgoBlockAppendChild(argU(a, 0), argU(a, 1))
		return retU32(c, idx)
	})

	setFunc(ctx, "_BinaryenBlockInsertChildAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoBlockInsertChildAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(c)
	})

	setFunc(ctx, "_BinaryenBlockRemoveChildAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoBlockRemoveChildAt(argU(a, 0), argU32(a, 1)))
	})

	// --- Mutation operations: Call ---

	setFunc(ctx, "_BinaryenCallAppendOperand", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		idx := cgoCallAppendOperand(argU(a, 0), argU(a, 1))
		return retU32(c, idx)
	})

	setFunc(ctx, "_BinaryenCallInsertOperandAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoCallInsertOperandAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(c)
	})

	setFunc(ctx, "_BinaryenCallRemoveOperandAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoCallRemoveOperandAt(argU(a, 0), argU32(a, 1)))
	})

	// --- Mutation operations: CallIndirect ---

	setFunc(ctx, "_BinaryenCallIndirectAppendOperand", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		idx := cgoCallIndirectAppendOperand(argU(a, 0), argU(a, 1))
		return retU32(c, idx)
	})

	setFunc(ctx, "_BinaryenCallIndirectInsertOperandAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoCallIndirectInsertOperandAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(c)
	})

	setFunc(ctx, "_BinaryenCallIndirectRemoveOperandAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoCallIndirectRemoveOperandAt(argU(a, 0), argU32(a, 1)))
	})

	// --- Mutation operations: CallRef ---

	setFunc(ctx, "_BinaryenCallRefAppendOperand", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		idx := cgoCallRefAppendOperand(argU(a, 0), argU(a, 1))
		return retU32(c, idx)
	})

	setFunc(ctx, "_BinaryenCallRefInsertOperandAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoCallRefInsertOperandAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(c)
	})

	setFunc(ctx, "_BinaryenCallRefRemoveOperandAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoCallRefRemoveOperandAt(argU(a, 0), argU32(a, 1)))
	})

	// --- Mutation operations: Switch ---

	setFunc(ctx, "_BinaryenSwitchAppendName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		idx := cgoSwitchAppendName(expr, name)
		return retU32(c, idx)
	})

	setFunc(ctx, "_BinaryenSwitchInsertNameAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		index := argU32(a, 1)
		namePtr := argI(a, 2)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		cgoSwitchInsertNameAt(expr, index, name)
		return retVoid(c)
	})

	setFunc(ctx, "_BinaryenSwitchRemoveNameAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cName := cgoSwitchRemoveNameAt(argU(a, 0), argU32(a, 1))
		if cName == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cName)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})

	// --- Mutation operations: Throw ---

	setFunc(ctx, "_BinaryenThrowAppendOperand", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		idx := cgoThrowAppendOperand(argU(a, 0), argU(a, 1))
		return retU32(c, idx)
	})

	setFunc(ctx, "_BinaryenThrowInsertOperandAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoThrowInsertOperandAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(c)
	})

	setFunc(ctx, "_BinaryenThrowRemoveOperandAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoThrowRemoveOperandAt(argU(a, 0), argU32(a, 1)))
	})

	// --- Mutation operations: Try ---

	setFunc(ctx, "_BinaryenTryAppendCatchTag", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		tagPtr := argI(a, 1)
		tag := cgoCString(lm.ReadString(tagPtr))
		defer cgoFree(unsafe.Pointer(tag))
		idx := cgoTryAppendCatchTag(expr, tag)
		return retU32(c, idx)
	})

	setFunc(ctx, "_BinaryenTryInsertCatchTagAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		index := argU32(a, 1)
		tagPtr := argI(a, 2)
		tag := cgoCString(lm.ReadString(tagPtr))
		defer cgoFree(unsafe.Pointer(tag))
		cgoTryInsertCatchTagAt(expr, index, tag)
		return retVoid(c)
	})

	setFunc(ctx, "_BinaryenTryRemoveCatchTagAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cName := cgoTryRemoveCatchTagAt(argU(a, 0), argU32(a, 1))
		if cName == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cName)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})

	setFunc(ctx, "_BinaryenTryAppendCatchBody", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		idx := cgoTryAppendCatchBody(argU(a, 0), argU(a, 1))
		return retU32(c, idx)
	})

	setFunc(ctx, "_BinaryenTryInsertCatchBodyAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoTryInsertCatchBodyAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(c)
	})

	setFunc(ctx, "_BinaryenTryRemoveCatchBodyAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoTryRemoveCatchBodyAt(argU(a, 0), argU32(a, 1)))
	})

	setFunc(ctx, "_BinaryenTryHasCatchAll", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retBool(c, cgoTryHasCatchAll(argU(args, 0)))
	})

	// --- Mutation operations: TupleMake ---

	setFunc(ctx, "_BinaryenTupleMakeAppendOperand", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		idx := cgoTupleMakeAppendOperand(argU(a, 0), argU(a, 1))
		return retU32(c, idx)
	})

	setFunc(ctx, "_BinaryenTupleMakeInsertOperandAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoTupleMakeInsertOperandAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(c)
	})

	setFunc(ctx, "_BinaryenTupleMakeRemoveOperandAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoTupleMakeRemoveOperandAt(argU(a, 0), argU32(a, 1)))
	})

	// --- Mutation operations: StructNew ---

	setFunc(ctx, "_BinaryenStructNewAppendOperand", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		idx := cgoStructNewAppendOperand(argU(a, 0), argU(a, 1))
		return retU32(c, idx)
	})

	setFunc(ctx, "_BinaryenStructNewInsertOperandAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStructNewInsertOperandAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(c)
	})

	setFunc(ctx, "_BinaryenStructNewRemoveOperandAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStructNewRemoveOperandAt(argU(a, 0), argU32(a, 1)))
	})

	// --- Mutation operations: ArrayNewFixed ---

	setFunc(ctx, "_BinaryenArrayNewFixedAppendValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		idx := cgoArrayNewFixedAppendValue(argU(a, 0), argU(a, 1))
		return retU32(c, idx)
	})

	setFunc(ctx, "_BinaryenArrayNewFixedInsertValueAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoArrayNewFixedInsertValueAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(c)
	})

	setFunc(ctx, "_BinaryenArrayNewFixedRemoveValueAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoArrayNewFixedRemoveValueAt(argU(a, 0), argU32(a, 1)))
	})

	// --- Module operations ---

	setFunc(ctx, "_BinaryenModuleParse", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		textPtr := argI(a, 0)
		text := cgoCString(lm.ReadString(textPtr))
		defer cgoFree(unsafe.Pointer(text))
		return retF(c, cgoModuleParse(text))
	})

	setFunc(ctx, "_BinaryenModuleRead", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		dataPtr := argI(a, 0)
		size := argI(a, 1)
		data := lm.ReadBytes(dataPtr, size)
		var cData unsafe.Pointer
		if len(data) > 0 {
			cData = unsafe.Pointer(&data[0])
		}
		return retF(c, cgoModuleRead(cData, size))
	})

	setFunc(ctx, "_BinaryenModuleReadWithFeatures", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		dataPtr := argI(a, 0)
		size := argI(a, 1)
		features := argU32(a, 2)
		data := lm.ReadBytes(dataPtr, size)
		var cData unsafe.Pointer
		if len(data) > 0 {
			cData = unsafe.Pointer(&data[0])
		}
		return retF(c, cgoModuleReadWithFeatures(cData, size, features))
	})

	setFunc(ctx, "_BinaryenModuleInterpret", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		cgoModuleInterpret(argU(args, 0))
		return retVoid(c)
	})

	setFunc(ctx, "_BinaryenModuleAddDebugInfoFileName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		filenamePtr := argI(a, 1)
		filename := cgoCString(lm.ReadString(filenamePtr))
		defer cgoFree(unsafe.Pointer(filename))
		idx := cgoModuleAddDebugInfoFileName(module, filename)
		return retU32(c, idx)
	})

	// --- Function operations ---

	setFunc(ctx, "_BinaryenFunctionAddVar", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		fn := argU(a, 0)
		typ := argU(a, 1)
		idx := cgoFunctionAddVar(fn, typ)
		return retU32(c, idx)
	})

	setFunc(ctx, "_BinaryenFunctionOptimize", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		fn := argU(a, 0)
		module := argU(a, 1)
		cgoFunctionOptimize(fn, module)
		return retVoid(c)
	})

	setFunc(ctx, "_BinaryenFunctionRunPasses", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		fn := argU(a, 0)
		module := argU(a, 1)
		passesPtr := argI(a, 2)
		numPasses := argI(a, 3)
		passes := make([]string, numPasses)
		for i := 0; i < numPasses; i++ {
			sp := lm.I32Load(passesPtr + i*4)
			passes[i] = lm.ReadString(sp)
		}
		cgoFunctionRunPasses(fn, module, passes)
		return retVoid(c)
	})

	// --- Misc operations ---

	setFunc(ctx, "_BinaryenLiteralVec128", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		xPtr := argI(a, 0)
		outPtr := argI(a, 1)
		var x [16]byte
		if xPtr != 0 {
			xBytes := lm.ReadBytes(xPtr, 16)
			copy(x[:], xBytes)
		}
		litSize := cgoSizeofLiteral()
		out := make([]byte, litSize)
		cgoLiteralVec128(x, out)
		if outPtr != 0 {
			lm.WriteBytes(outPtr, out)
		}
		return retVoid(c)
	})

	setFunc(ctx, "_BinaryenRemoveElementSegment", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		cgoRemoveElementSegment(module, name)
		return retVoid(c)
	})

	setFunc(ctx, "_BinaryenTableHasMax", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retBool(c, cgoTableHasMax(argU(args, 0)))
	})

	setFunc(ctx, "_BinaryenCopyMemorySegmentData", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		segNamePtr := argI(a, 1)
		bufferPtr := argI(a, 2)
		segName := cgoCString(lm.ReadString(segNamePtr))
		defer cgoFree(unsafe.Pointer(segName))
		// We need to allocate a temporary buffer, copy data into it, then write to linear memory.
		// But we don't know the size. Use BinaryenGetMemorySegmentByteLength first.
		// For now, write directly using the buffer pointer as destination in linear memory.
		// The caller has already allocated enough space at bufferPtr.
		// We'll use a temp Go buffer and then copy to linear memory.
		// Actually, the C API writes to a char* buffer, so we need a Go-side buffer.
		// We'll get the segment byte length first.
		// For safety, use a large buffer and let C write into it.
		// The proper approach: the caller knows the size (they called GetMemorySegmentByteLength).
		// We just need to provide a buffer and copy results back.
		// Use a reasonable max or query the length.
		// Since this is rarely called, use a conservative approach:
		// allocate a temp buffer of the segment size, let C fill it, copy to linear memory.

		// Get segment byte length via C API
		segLen := int(cgoGetMemorySegmentByteLength(module, segName))
		if segLen > 0 {
			buf := make([]byte, segLen)
			cgoCopyMemorySegmentData(module, segName, unsafe.Pointer(&buf[0]))
			lm.WriteBytes(bufferPtr, buf)
		}
		return retVoid(c)
	})

	// ArrayNewElem — not in this binaryen C API version, stays as stub
	// ArrayFill — not in this binaryen C API version, stays as stub
	// ArrayInitData — not in this binaryen C API version, stays as stub
	// ArrayInitElem — not in this binaryen C API version, stays as stub
}

var _ = unsafe.Pointer(nil)
