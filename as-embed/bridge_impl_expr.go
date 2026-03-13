package asembed

import (
	"unsafe"

	"github.com/fastschema/qjs"
)

func registerExpressionConstructorImpls(ctx *qjs.Context, lm *LinearMemory) {
	ctx.SetFunc("_BinaryenBlock", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		childrenPtr := argI(a, 2)
		numChildren := argI(a, 3)
		typ := argU(a, 4)
		var name unsafe.Pointer
		if namePtr != 0 {
			s := lm.ReadString(namePtr)
			name = cgoCString(s)
			defer cgoFree(unsafe.Pointer(name))
		}
		children := readPtrArray(lm, childrenPtr, numChildren)
		return retF(this.Context(), cgoBlock(module, name, children, typ))
	})
	ctx.SetFunc("_BinaryenIf", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoIf(argU(a, 0), argU(a, 1), argU(a, 2), argU(a, 3)))
	})
	ctx.SetFunc("_BinaryenLoop", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		body := argU(a, 2)
		var name unsafe.Pointer
		if namePtr != 0 {
			s := lm.ReadString(namePtr)
			name = cgoCString(s)
			defer cgoFree(unsafe.Pointer(name))
		}
		return retF(this.Context(), cgoLoop(module, name, body))
	})
	ctx.SetFunc("_BinaryenBreak", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		condition := argU(a, 2)
		value := argU(a, 3)
		var name unsafe.Pointer
		if namePtr != 0 {
			s := lm.ReadString(namePtr)
			name = cgoCString(s)
			defer cgoFree(unsafe.Pointer(name))
		}
		return retF(this.Context(), cgoBreak(module, name, condition, value))
	})
	ctx.SetFunc("_BinaryenSwitch", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namesPtr := argI(a, 1)
		numNames := argI(a, 2)
		defaultNamePtr := argI(a, 3)
		condition := argU(a, 4)
		value := argU(a, 5)
		names := make([]unsafe.Pointer, numNames)
		for i := 0; i < numNames; i++ {
			sp := lm.I32Load(namesPtr + i*4)
			names[i] = cgoCString(lm.ReadString(sp))
		}
		var defaultName unsafe.Pointer
		if defaultNamePtr != 0 {
			defaultName = cgoCString(lm.ReadString(defaultNamePtr))
		}
		result := cgoSwitch(module, names, defaultName, condition, value)
		for _, n := range names {
			cgoFree(unsafe.Pointer(n))
		}
		if defaultName != nil {
			cgoFree(unsafe.Pointer(defaultName))
		}
		return retF(this.Context(), result)
	})
	ctx.SetFunc("_BinaryenCall", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		targetPtr := argI(a, 1)
		operandsPtr := argI(a, 2)
		numOperands := argI(a, 3)
		returnType := argU(a, 4)
		target := cgoCString(lm.ReadString(targetPtr))
		defer cgoFree(unsafe.Pointer(target))
		operands := readPtrArray(lm, operandsPtr, numOperands)
		return retF(this.Context(), cgoCall(module, target, operands, returnType))
	})
	ctx.SetFunc("_BinaryenReturnCall", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		targetPtr := argI(a, 1)
		operandsPtr := argI(a, 2)
		numOperands := argI(a, 3)
		returnType := argU(a, 4)
		target := cgoCString(lm.ReadString(targetPtr))
		defer cgoFree(unsafe.Pointer(target))
		operands := readPtrArray(lm, operandsPtr, numOperands)
		return retF(this.Context(), cgoReturnCall(module, target, operands, returnType))
	})
	ctx.SetFunc("_BinaryenCallIndirect", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		tablePtr := argI(a, 1)
		target := argU(a, 2)
		operandsPtr := argI(a, 3)
		numOperands := argI(a, 4)
		params := argU(a, 5)
		results := argU(a, 6)
		table := cgoCString(lm.ReadString(tablePtr))
		defer cgoFree(unsafe.Pointer(table))
		operands := readPtrArray(lm, operandsPtr, numOperands)
		return retF(this.Context(), cgoCallIndirect(module, table, target, operands, params, results))
	})
	ctx.SetFunc("_BinaryenReturnCallIndirect", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		tablePtr := argI(a, 1)
		target := argU(a, 2)
		operandsPtr := argI(a, 3)
		numOperands := argI(a, 4)
		params := argU(a, 5)
		results := argU(a, 6)
		table := cgoCString(lm.ReadString(tablePtr))
		defer cgoFree(unsafe.Pointer(table))
		operands := readPtrArray(lm, operandsPtr, numOperands)
		return retF(this.Context(), cgoReturnCallIndirect(module, table, target, operands, params, results))
	})
	ctx.SetFunc("_BinaryenLocalGet", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoLocalGet(argU(a, 0), argI(a, 1), argU(a, 2)))
	})
	ctx.SetFunc("_BinaryenLocalSet", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoLocalSet(argU(a, 0), argI(a, 1), argU(a, 2)))
	})
	ctx.SetFunc("_BinaryenLocalTee", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoLocalTee(argU(a, 0), argI(a, 1), argU(a, 2), argU(a, 3)))
	})
	ctx.SetFunc("_BinaryenGlobalGet", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		typ := argU(a, 2)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(this.Context(), cgoGlobalGet(module, name, typ))
	})
	ctx.SetFunc("_BinaryenGlobalSet", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		value := argU(a, 2)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(this.Context(), cgoGlobalSet(module, name, value))
	})
	ctx.SetFunc("_BinaryenLoad", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		bytes := argU32(a, 1)
		signed := argBool(a, 2)
		offset := argU32(a, 3)
		align := argU32(a, 4)
		typ := argU(a, 5)
		ptr := argU(a, 6)
		memNamePtr := argI(a, 7)
		var memName unsafe.Pointer
		if memNamePtr != 0 {
			memName = cgoCString(lm.ReadString(memNamePtr))
			defer cgoFree(unsafe.Pointer(memName))
		}
		return retF(this.Context(), cgoLoad(module, bytes, signed, offset, align, typ, ptr, memName))
	})
	ctx.SetFunc("_BinaryenStore", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		bytes := argU32(a, 1)
		offset := argU32(a, 2)
		align := argU32(a, 3)
		ptr := argU(a, 4)
		value := argU(a, 5)
		typ := argU(a, 6)
		memNamePtr := argI(a, 7)
		var memName unsafe.Pointer
		if memNamePtr != 0 {
			memName = cgoCString(lm.ReadString(memNamePtr))
			defer cgoFree(unsafe.Pointer(memName))
		}
		return retF(this.Context(), cgoStore(module, bytes, offset, align, ptr, value, typ, memName))
	})
	ctx.SetFunc("_BinaryenAtomicLoad", func(this *qjs.This) (*qjs.Value, error) {
		// AtomicLoad(module, bytes, offset, type, ptr, memoryName) — same as Load but atomic=true
		a := this.Args()
		module := argU(a, 0)
		bytes := argU32(a, 1)
		offset := argU32(a, 2)
		typ := argU(a, 3)
		ptr := argU(a, 4)
		memNamePtr := argI(a, 5)
		var memName unsafe.Pointer
		if memNamePtr != 0 {
			memName = cgoCString(lm.ReadString(memNamePtr))
			defer cgoFree(unsafe.Pointer(memName))
		}
		// Use Load with signed=false, align=bytes (natural alignment for atomics)
		return retF(this.Context(), cgoLoad(module, bytes, false, offset, bytes, typ, ptr, memName))
	})
	ctx.SetFunc("_BinaryenAtomicStore", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		bytes := argU32(a, 1)
		offset := argU32(a, 2)
		ptr := argU(a, 3)
		value := argU(a, 4)
		typ := argU(a, 5)
		memNamePtr := argI(a, 6)
		var memName unsafe.Pointer
		if memNamePtr != 0 {
			memName = cgoCString(lm.ReadString(memNamePtr))
			defer cgoFree(unsafe.Pointer(memName))
		}
		return retF(this.Context(), cgoStore(module, bytes, offset, bytes, ptr, value, typ, memName))
	})
	ctx.SetFunc("_BinaryenConst", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		litPtr := argI(a, 1)
		// Read literal struct from linear memory
		litSize := cgoSizeofLiteral()
		litBytes := lm.ReadBytes(litPtr, litSize)
		return retF(this.Context(), cgoConstBytes(module, litBytes))
	})
	ctx.SetFunc("_BinaryenUnary", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoUnary(argU(a, 0), argI32(a, 1), argU(a, 2)))
	})
	ctx.SetFunc("_BinaryenBinary", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoBinary(argU(a, 0), argI32(a, 1), argU(a, 2), argU(a, 3)))
	})
	ctx.SetFunc("_BinaryenSelect", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoSelect(argU(a, 0), argU(a, 1), argU(a, 2), argU(a, 3)))
	})
	ctx.SetFunc("_BinaryenDrop", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoDrop(argU(a, 0), argU(a, 1)))
	})
	ctx.SetFunc("_BinaryenReturn", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoReturn(argU(a, 0), argU(a, 1)))
	})
	ctx.SetFunc("_BinaryenNop", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoNop(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_BinaryenUnreachable", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoUnreachable(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_BinaryenMemorySize", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		memNamePtr := argI(a, 1)
		memIs64 := argBool(a, 2)
		var memName unsafe.Pointer
		if memNamePtr != 0 {
			memName = cgoCString(lm.ReadString(memNamePtr))
			defer cgoFree(unsafe.Pointer(memName))
		}
		return retF(this.Context(), cgoMemorySize(module, memName, memIs64))
	})
	ctx.SetFunc("_BinaryenMemoryGrow", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		delta := argU(a, 1)
		memNamePtr := argI(a, 2)
		memIs64 := argBool(a, 3)
		var memName unsafe.Pointer
		if memNamePtr != 0 {
			memName = cgoCString(lm.ReadString(memNamePtr))
			defer cgoFree(unsafe.Pointer(memName))
		}
		return retF(this.Context(), cgoMemoryGrow(module, delta, memName, memIs64))
	})
	ctx.SetFunc("_BinaryenTry", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		body := argU(a, 2)
		catchTagsPtr := argI(a, 3)
		numCatchTags := argI(a, 4)
		catchBodiesPtr := argI(a, 5)
		numCatchBodies := argI(a, 6)
		delegateTargetPtr := argI(a, 7)
		var name unsafe.Pointer
		if namePtr != 0 {
			name = cgoCString(lm.ReadString(namePtr))
			defer cgoFree(unsafe.Pointer(name))
		}
		catchTags := make([]unsafe.Pointer, numCatchTags)
		for i := 0; i < numCatchTags; i++ {
			sp := lm.I32Load(catchTagsPtr + i*4)
			catchTags[i] = cgoCString(lm.ReadString(sp))
		}
		catchBodies := readPtrArray(lm, catchBodiesPtr, numCatchBodies)
		var delegateTarget unsafe.Pointer
		if delegateTargetPtr != 0 {
			delegateTarget = cgoCString(lm.ReadString(delegateTargetPtr))
		}
		result := cgoTry(module, name, body, catchTags, catchBodies, delegateTarget)
		for _, t := range catchTags {
			cgoFree(unsafe.Pointer(t))
		}
		if delegateTarget != nil {
			cgoFree(unsafe.Pointer(delegateTarget))
		}
		return retF(this.Context(), result)
	})
	ctx.SetFunc("_BinaryenThrow", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		tagPtr := argI(a, 1)
		operandsPtr := argI(a, 2)
		numOperands := argI(a, 3)
		tag := cgoCString(lm.ReadString(tagPtr))
		defer cgoFree(unsafe.Pointer(tag))
		operands := readPtrArray(lm, operandsPtr, numOperands)
		return retF(this.Context(), cgoThrow(module, tag, operands))
	})
	ctx.SetFunc("_BinaryenRethrow", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		targetPtr := argI(a, 1)
		target := cgoCString(lm.ReadString(targetPtr))
		defer cgoFree(unsafe.Pointer(target))
		return retF(this.Context(), cgoRethrow(module, target))
	})
	ctx.SetFunc("_BinaryenRefNull", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoRefNull(argU(a, 0), argU(a, 1)))
	})
	ctx.SetFunc("_BinaryenRefIsNull", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoRefIsNull(argU(a, 0), argU(a, 1)))
	})
	ctx.SetFunc("_BinaryenRefAs", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoRefAs(argU(a, 0), argI32(a, 1), argU(a, 2)))
	})
	ctx.SetFunc("_BinaryenRefFunc", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		funcNamePtr := argI(a, 1)
		typ := argU(a, 2)
		funcName := cgoCString(lm.ReadString(funcNamePtr))
		defer cgoFree(unsafe.Pointer(funcName))
		return retF(this.Context(), cgoRefFunc(module, funcName, typ))
	})
	ctx.SetFunc("_BinaryenRefEq", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoRefEq(argU(a, 0), argU(a, 1), argU(a, 2)))
	})
	ctx.SetFunc("_BinaryenTableGet", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		index := argU(a, 2)
		typ := argU(a, 3)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(this.Context(), cgoTableGet(module, name, index, typ))
	})
	ctx.SetFunc("_BinaryenTableSet", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		index := argU(a, 2)
		value := argU(a, 3)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(this.Context(), cgoTableSet(module, name, index, value))
	})
	ctx.SetFunc("_BinaryenTableSize", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(this.Context(), cgoTableSize(module, name))
	})
	ctx.SetFunc("_BinaryenTableGrow", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		value := argU(a, 2)
		delta := argU(a, 3)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(this.Context(), cgoTableGrow(module, name, value, delta))
	})
	ctx.SetFunc("_BinaryenTupleMake", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		operandsPtr := argI(a, 1)
		numOperands := argI(a, 2)
		operands := readPtrArray(lm, operandsPtr, numOperands)
		return retF(this.Context(), cgoTupleMake(module, operands))
	})
	ctx.SetFunc("_BinaryenTupleExtract", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoTupleExtract(argU(a, 0), argU(a, 1), argI(a, 2)))
	})
	ctx.SetFunc("_BinaryenPop", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoPop(argU(a, 0), argU(a, 1)))
	})
	ctx.SetFunc("_BinaryenMemoryCopy", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		dest := argU(a, 1)
		source := argU(a, 2)
		size := argU(a, 3)
		destMemPtr := argI(a, 4)
		srcMemPtr := argI(a, 5)
		var destMem, srcMem unsafe.Pointer
		if destMemPtr != 0 {
			destMem = cgoCString(lm.ReadString(destMemPtr))
			defer cgoFree(unsafe.Pointer(destMem))
		}
		if srcMemPtr != 0 {
			srcMem = cgoCString(lm.ReadString(srcMemPtr))
			defer cgoFree(unsafe.Pointer(srcMem))
		}
		return retF(this.Context(), cgoMemoryCopy(module, dest, source, size, destMem, srcMem))
	})
	ctx.SetFunc("_BinaryenMemoryFill", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		dest := argU(a, 1)
		value := argU(a, 2)
		size := argU(a, 3)
		memNamePtr := argI(a, 4)
		var memName unsafe.Pointer
		if memNamePtr != 0 {
			memName = cgoCString(lm.ReadString(memNamePtr))
			defer cgoFree(unsafe.Pointer(memName))
		}
		return retF(this.Context(), cgoMemoryFill(module, dest, value, size, memName))
	})
}

var _ = unsafe.Pointer(nil)
