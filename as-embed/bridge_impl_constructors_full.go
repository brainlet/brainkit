package asembed

import (
	"unsafe"

	"github.com/fastschema/qjs"
)

func registerAllConstructorImpls(ctx *qjs.Context, lm *LinearMemory) {
	// --- Atomic expression constructors ---

	ctx.SetFunc("_BinaryenAtomicRMW", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
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
		return retF(this.Context(), cgoAtomicRMW(module, op, bytes, offset, ptr, value, typ, memName))
	})

	ctx.SetFunc("_BinaryenAtomicCmpxchg", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
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
		return retF(this.Context(), cgoAtomicCmpxchg(module, bytes, offset, ptr, expected, replacement, typ, memName))
	})

	ctx.SetFunc("_BinaryenAtomicWait", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
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
		return retF(this.Context(), cgoAtomicWait(module, ptr, expected, timeout, typ, memName))
	})

	ctx.SetFunc("_BinaryenAtomicNotify", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		ptr := argU(a, 1)
		notifyCount := argU(a, 2)
		memNamePtr := argI(a, 3)
		var memName unsafe.Pointer
		if memNamePtr != 0 {
			memName = cgoCString(lm.ReadString(memNamePtr))
			defer cgoFree(unsafe.Pointer(memName))
		}
		return retF(this.Context(), cgoAtomicNotify(module, ptr, notifyCount, memName))
	})

	ctx.SetFunc("_BinaryenAtomicFence", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoAtomicFence(argU(this.Args(), 0)))
	})

	// --- SIMD expression constructors ---

	ctx.SetFunc("_BinaryenSIMDExtract", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		op := argI32(a, 1)
		vec := argU(a, 2)
		index := uint8(argI32(a, 3))
		return retF(this.Context(), cgoSIMDExtract(module, op, vec, index))
	})

	ctx.SetFunc("_BinaryenSIMDReplace", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		op := argI32(a, 1)
		vec := argU(a, 2)
		index := uint8(argI32(a, 3))
		value := argU(a, 4)
		return retF(this.Context(), cgoSIMDReplace(module, op, vec, index, value))
	})

	ctx.SetFunc("_BinaryenSIMDShuffle", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		left := argU(a, 1)
		right := argU(a, 2)
		maskPtr := argI(a, 3)
		var mask [16]byte
		if maskPtr != 0 {
			maskBytes := lm.ReadBytes(maskPtr, 16)
			copy(mask[:], maskBytes)
		}
		return retF(this.Context(), cgoSIMDShuffle(module, left, right, mask))
	})

	ctx.SetFunc("_BinaryenSIMDTernary", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		op := argI32(a, 1)
		va := argU(a, 2)
		vb := argU(a, 3)
		vc := argU(a, 4)
		return retF(this.Context(), cgoSIMDTernary(module, op, va, vb, vc))
	})

	ctx.SetFunc("_BinaryenSIMDShift", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		op := argI32(a, 1)
		vec := argU(a, 2)
		shift := argU(a, 3)
		return retF(this.Context(), cgoSIMDShift(module, op, vec, shift))
	})

	ctx.SetFunc("_BinaryenSIMDLoad", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
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
		return retF(this.Context(), cgoSIMDLoad(module, op, offset, align, ptr, memName))
	})

	ctx.SetFunc("_BinaryenSIMDLoadStoreLane", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
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
		return retF(this.Context(), cgoSIMDLoadStoreLane(module, op, offset, align, index, ptr, vec, memName))
	})

	// --- Bulk memory expression constructors ---

	ctx.SetFunc("_BinaryenDataDrop", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		segmentPtr := argI(a, 1)
		segment := cgoCString(lm.ReadString(segmentPtr))
		defer cgoFree(unsafe.Pointer(segment))
		return retF(this.Context(), cgoDataDrop(module, segment))
	})

	ctx.SetFunc("_BinaryenMemoryInit", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
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
		return retF(this.Context(), cgoMemoryInit(module, segment, dest, offset, size, memName))
	})

	// --- GC array expression constructors ---

	ctx.SetFunc("_BinaryenArrayNewData", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		typ := argU(a, 1)
		namePtr := argI(a, 2)
		offset := argU(a, 3)
		size := argU(a, 4)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(this.Context(), cgoArrayNewData(module, typ, name, offset, size))
	})

	// ArrayNewElem, ArrayFill, ArrayInitData, ArrayInitElem are not available in this
	// binaryen version. They remain as stubs (returning 0) from RegisterBinaryenBridge.
	// When the binaryen version is updated, these can be implemented.

	// --- Mutation operations: Block ---

	ctx.SetFunc("_BinaryenBlockAppendChild", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		idx := cgoBlockAppendChild(argU(a, 0), argU(a, 1))
		return retU32(this.Context(), idx)
	})

	ctx.SetFunc("_BinaryenBlockInsertChildAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoBlockInsertChildAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(this.Context())
	})

	ctx.SetFunc("_BinaryenBlockRemoveChildAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoBlockRemoveChildAt(argU(a, 0), argU32(a, 1)))
	})

	// --- Mutation operations: Call ---

	ctx.SetFunc("_BinaryenCallAppendOperand", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		idx := cgoCallAppendOperand(argU(a, 0), argU(a, 1))
		return retU32(this.Context(), idx)
	})

	ctx.SetFunc("_BinaryenCallInsertOperandAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoCallInsertOperandAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(this.Context())
	})

	ctx.SetFunc("_BinaryenCallRemoveOperandAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoCallRemoveOperandAt(argU(a, 0), argU32(a, 1)))
	})

	// --- Mutation operations: CallIndirect ---

	ctx.SetFunc("_BinaryenCallIndirectAppendOperand", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		idx := cgoCallIndirectAppendOperand(argU(a, 0), argU(a, 1))
		return retU32(this.Context(), idx)
	})

	ctx.SetFunc("_BinaryenCallIndirectInsertOperandAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoCallIndirectInsertOperandAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(this.Context())
	})

	ctx.SetFunc("_BinaryenCallIndirectRemoveOperandAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoCallIndirectRemoveOperandAt(argU(a, 0), argU32(a, 1)))
	})

	// --- Mutation operations: CallRef ---

	ctx.SetFunc("_BinaryenCallRefAppendOperand", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		idx := cgoCallRefAppendOperand(argU(a, 0), argU(a, 1))
		return retU32(this.Context(), idx)
	})

	ctx.SetFunc("_BinaryenCallRefInsertOperandAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoCallRefInsertOperandAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(this.Context())
	})

	ctx.SetFunc("_BinaryenCallRefRemoveOperandAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoCallRefRemoveOperandAt(argU(a, 0), argU32(a, 1)))
	})

	// --- Mutation operations: Switch ---

	ctx.SetFunc("_BinaryenSwitchAppendName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		idx := cgoSwitchAppendName(expr, name)
		return retU32(this.Context(), idx)
	})

	ctx.SetFunc("_BinaryenSwitchInsertNameAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		index := argU32(a, 1)
		namePtr := argI(a, 2)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		cgoSwitchInsertNameAt(expr, index, name)
		return retVoid(this.Context())
	})

	ctx.SetFunc("_BinaryenSwitchRemoveNameAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cName := cgoSwitchRemoveNameAt(argU(a, 0), argU32(a, 1))
		if cName == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cName)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})

	// --- Mutation operations: Throw ---

	ctx.SetFunc("_BinaryenThrowAppendOperand", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		idx := cgoThrowAppendOperand(argU(a, 0), argU(a, 1))
		return retU32(this.Context(), idx)
	})

	ctx.SetFunc("_BinaryenThrowInsertOperandAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoThrowInsertOperandAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(this.Context())
	})

	ctx.SetFunc("_BinaryenThrowRemoveOperandAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoThrowRemoveOperandAt(argU(a, 0), argU32(a, 1)))
	})

	// --- Mutation operations: Try ---

	ctx.SetFunc("_BinaryenTryAppendCatchTag", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		tagPtr := argI(a, 1)
		tag := cgoCString(lm.ReadString(tagPtr))
		defer cgoFree(unsafe.Pointer(tag))
		idx := cgoTryAppendCatchTag(expr, tag)
		return retU32(this.Context(), idx)
	})

	ctx.SetFunc("_BinaryenTryInsertCatchTagAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		index := argU32(a, 1)
		tagPtr := argI(a, 2)
		tag := cgoCString(lm.ReadString(tagPtr))
		defer cgoFree(unsafe.Pointer(tag))
		cgoTryInsertCatchTagAt(expr, index, tag)
		return retVoid(this.Context())
	})

	ctx.SetFunc("_BinaryenTryRemoveCatchTagAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cName := cgoTryRemoveCatchTagAt(argU(a, 0), argU32(a, 1))
		if cName == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cName)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})

	ctx.SetFunc("_BinaryenTryAppendCatchBody", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		idx := cgoTryAppendCatchBody(argU(a, 0), argU(a, 1))
		return retU32(this.Context(), idx)
	})

	ctx.SetFunc("_BinaryenTryInsertCatchBodyAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoTryInsertCatchBodyAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(this.Context())
	})

	ctx.SetFunc("_BinaryenTryRemoveCatchBodyAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoTryRemoveCatchBodyAt(argU(a, 0), argU32(a, 1)))
	})

	ctx.SetFunc("_BinaryenTryHasCatchAll", func(this *qjs.This) (*qjs.Value, error) {
		return retBool(this.Context(), cgoTryHasCatchAll(argU(this.Args(), 0)))
	})

	// --- Mutation operations: TupleMake ---

	ctx.SetFunc("_BinaryenTupleMakeAppendOperand", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		idx := cgoTupleMakeAppendOperand(argU(a, 0), argU(a, 1))
		return retU32(this.Context(), idx)
	})

	ctx.SetFunc("_BinaryenTupleMakeInsertOperandAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoTupleMakeInsertOperandAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(this.Context())
	})

	ctx.SetFunc("_BinaryenTupleMakeRemoveOperandAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoTupleMakeRemoveOperandAt(argU(a, 0), argU32(a, 1)))
	})

	// --- Mutation operations: StructNew ---

	ctx.SetFunc("_BinaryenStructNewAppendOperand", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		idx := cgoStructNewAppendOperand(argU(a, 0), argU(a, 1))
		return retU32(this.Context(), idx)
	})

	ctx.SetFunc("_BinaryenStructNewInsertOperandAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStructNewInsertOperandAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(this.Context())
	})

	ctx.SetFunc("_BinaryenStructNewRemoveOperandAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStructNewRemoveOperandAt(argU(a, 0), argU32(a, 1)))
	})

	// --- Mutation operations: ArrayNewFixed ---

	ctx.SetFunc("_BinaryenArrayNewFixedAppendValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		idx := cgoArrayNewFixedAppendValue(argU(a, 0), argU(a, 1))
		return retU32(this.Context(), idx)
	})

	ctx.SetFunc("_BinaryenArrayNewFixedInsertValueAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoArrayNewFixedInsertValueAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(this.Context())
	})

	ctx.SetFunc("_BinaryenArrayNewFixedRemoveValueAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoArrayNewFixedRemoveValueAt(argU(a, 0), argU32(a, 1)))
	})

	// --- Module operations ---

	ctx.SetFunc("_BinaryenModuleParse", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		textPtr := argI(a, 0)
		text := cgoCString(lm.ReadString(textPtr))
		defer cgoFree(unsafe.Pointer(text))
		return retF(this.Context(), cgoModuleParse(text))
	})

	ctx.SetFunc("_BinaryenModuleRead", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		dataPtr := argI(a, 0)
		size := argI(a, 1)
		data := lm.ReadBytes(dataPtr, size)
		var cData unsafe.Pointer
		if len(data) > 0 {
			cData = unsafe.Pointer(&data[0])
		}
		return retF(this.Context(), cgoModuleRead(cData, size))
	})

	ctx.SetFunc("_BinaryenModuleReadWithFeatures", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		dataPtr := argI(a, 0)
		size := argI(a, 1)
		features := argU32(a, 2)
		data := lm.ReadBytes(dataPtr, size)
		var cData unsafe.Pointer
		if len(data) > 0 {
			cData = unsafe.Pointer(&data[0])
		}
		return retF(this.Context(), cgoModuleReadWithFeatures(cData, size, features))
	})

	ctx.SetFunc("_BinaryenModuleInterpret", func(this *qjs.This) (*qjs.Value, error) {
		cgoModuleInterpret(argU(this.Args(), 0))
		return retVoid(this.Context())
	})

	ctx.SetFunc("_BinaryenModuleAddDebugInfoFileName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		filenamePtr := argI(a, 1)
		filename := cgoCString(lm.ReadString(filenamePtr))
		defer cgoFree(unsafe.Pointer(filename))
		idx := cgoModuleAddDebugInfoFileName(module, filename)
		return retU32(this.Context(), idx)
	})

	// --- Function operations ---

	ctx.SetFunc("_BinaryenFunctionAddVar", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		fn := argU(a, 0)
		typ := argU(a, 1)
		idx := cgoFunctionAddVar(fn, typ)
		return retU32(this.Context(), idx)
	})

	ctx.SetFunc("_BinaryenFunctionOptimize", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		fn := argU(a, 0)
		module := argU(a, 1)
		cgoFunctionOptimize(fn, module)
		return retVoid(this.Context())
	})

	ctx.SetFunc("_BinaryenFunctionRunPasses", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
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
		return retVoid(this.Context())
	})

	// --- Misc operations ---

	ctx.SetFunc("_BinaryenLiteralVec128", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
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
		return retVoid(this.Context())
	})

	ctx.SetFunc("_BinaryenRemoveElementSegment", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		cgoRemoveElementSegment(module, name)
		return retVoid(this.Context())
	})

	ctx.SetFunc("_BinaryenTableHasMax", func(this *qjs.This) (*qjs.Value, error) {
		return retBool(this.Context(), cgoTableHasMax(argU(this.Args(), 0)))
	})

	ctx.SetFunc("_BinaryenCopyMemorySegmentData", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
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
		return retVoid(this.Context())
	})

	// ArrayNewElem — not in this binaryen C API version, stays as stub
	// ArrayFill — not in this binaryen C API version, stays as stub
	// ArrayInitData — not in this binaryen C API version, stays as stub
	// ArrayInitElem — not in this binaryen C API version, stays as stub
}

var _ = unsafe.Pointer(nil)
