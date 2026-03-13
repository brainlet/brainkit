package asembed

import (
	"unsafe"

	quickjs "github.com/buke/quickjs-go"
)

func registerExpressionConstructorImpls(ctx *quickjs.Context, lm *LinearMemory) {
	setFunc(ctx, "_BinaryenBlock", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
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
		return retF(c, cgoBlock(module, name, children, typ))
	})
	setFunc(ctx, "_BinaryenIf", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoIf(argU(a, 0), argU(a, 1), argU(a, 2), argU(a, 3)))
	})
	setFunc(ctx, "_BinaryenLoop", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		body := argU(a, 2)
		var name unsafe.Pointer
		if namePtr != 0 {
			s := lm.ReadString(namePtr)
			name = cgoCString(s)
			defer cgoFree(unsafe.Pointer(name))
		}
		return retF(c, cgoLoop(module, name, body))
	})
	setFunc(ctx, "_BinaryenBreak", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
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
		return retF(c, cgoBreak(module, name, condition, value))
	})
	setFunc(ctx, "_BinaryenSwitch", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
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
		return retF(c, result)
	})
	setFunc(ctx, "_BinaryenCall", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		targetPtr := argI(a, 1)
		operandsPtr := argI(a, 2)
		numOperands := argI(a, 3)
		returnType := argU(a, 4)
		target := cgoCString(lm.ReadString(targetPtr))
		defer cgoFree(unsafe.Pointer(target))
		operands := readPtrArray(lm, operandsPtr, numOperands)
		return retF(c, cgoCall(module, target, operands, returnType))
	})
	setFunc(ctx, "_BinaryenReturnCall", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		targetPtr := argI(a, 1)
		operandsPtr := argI(a, 2)
		numOperands := argI(a, 3)
		returnType := argU(a, 4)
		target := cgoCString(lm.ReadString(targetPtr))
		defer cgoFree(unsafe.Pointer(target))
		operands := readPtrArray(lm, operandsPtr, numOperands)
		return retF(c, cgoReturnCall(module, target, operands, returnType))
	})
	setFunc(ctx, "_BinaryenCallIndirect", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
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
		return retF(c, cgoCallIndirect(module, table, target, operands, params, results))
	})
	setFunc(ctx, "_BinaryenReturnCallIndirect", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
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
		return retF(c, cgoReturnCallIndirect(module, table, target, operands, params, results))
	})
	setFunc(ctx, "_BinaryenLocalGet", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoLocalGet(argU(a, 0), argI(a, 1), argU(a, 2)))
	})
	setFunc(ctx, "_BinaryenLocalSet", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoLocalSet(argU(a, 0), argI(a, 1), argU(a, 2)))
	})
	setFunc(ctx, "_BinaryenLocalTee", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoLocalTee(argU(a, 0), argI(a, 1), argU(a, 2), argU(a, 3)))
	})
	setFunc(ctx, "_BinaryenGlobalGet", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		typ := argU(a, 2)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(c, cgoGlobalGet(module, name, typ))
	})
	setFunc(ctx, "_BinaryenGlobalSet", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		value := argU(a, 2)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(c, cgoGlobalSet(module, name, value))
	})
	setFunc(ctx, "_BinaryenLoad", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
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
		return retF(c, cgoLoad(module, bytes, signed, offset, align, typ, ptr, memName))
	})
	setFunc(ctx, "_BinaryenStore", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
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
		return retF(c, cgoStore(module, bytes, offset, align, ptr, value, typ, memName))
	})
	setFunc(ctx, "_BinaryenAtomicLoad", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		// AtomicLoad(module, bytes, offset, type, ptr, memoryName) — same as Load but atomic=true
		a := args
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
		return retF(c, cgoLoad(module, bytes, false, offset, bytes, typ, ptr, memName))
	})
	setFunc(ctx, "_BinaryenAtomicStore", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
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
		return retF(c, cgoStore(module, bytes, offset, bytes, ptr, value, typ, memName))
	})
	setFunc(ctx, "_BinaryenConst", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		litPtr := argI(a, 1)
		// Read literal struct from linear memory
		litSize := cgoSizeofLiteral()
		litBytes := lm.ReadBytes(litPtr, litSize)
		return retF(c, cgoConstBytes(module, litBytes))
	})
	setFunc(ctx, "_BinaryenUnary", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoUnary(argU(a, 0), argI32(a, 1), argU(a, 2)))
	})
	setFunc(ctx, "_BinaryenBinary", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoBinary(argU(a, 0), argI32(a, 1), argU(a, 2), argU(a, 3)))
	})
	setFunc(ctx, "_BinaryenSelect", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoSelect(argU(a, 0), argU(a, 1), argU(a, 2), argU(a, 3)))
	})
	setFunc(ctx, "_BinaryenDrop", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoDrop(argU(a, 0), argU(a, 1)))
	})
	setFunc(ctx, "_BinaryenReturn", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoReturn(argU(a, 0), argU(a, 1)))
	})
	setFunc(ctx, "_BinaryenNop", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoNop(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenUnreachable", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoUnreachable(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenMemorySize", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		memNamePtr := argI(a, 1)
		memIs64 := argBool(a, 2)
		var memName unsafe.Pointer
		if memNamePtr != 0 {
			memName = cgoCString(lm.ReadString(memNamePtr))
			defer cgoFree(unsafe.Pointer(memName))
		}
		return retF(c, cgoMemorySize(module, memName, memIs64))
	})
	setFunc(ctx, "_BinaryenMemoryGrow", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		delta := argU(a, 1)
		memNamePtr := argI(a, 2)
		memIs64 := argBool(a, 3)
		var memName unsafe.Pointer
		if memNamePtr != 0 {
			memName = cgoCString(lm.ReadString(memNamePtr))
			defer cgoFree(unsafe.Pointer(memName))
		}
		return retF(c, cgoMemoryGrow(module, delta, memName, memIs64))
	})
	setFunc(ctx, "_BinaryenTry", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
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
		return retF(c, result)
	})
	setFunc(ctx, "_BinaryenThrow", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		tagPtr := argI(a, 1)
		operandsPtr := argI(a, 2)
		numOperands := argI(a, 3)
		tag := cgoCString(lm.ReadString(tagPtr))
		defer cgoFree(unsafe.Pointer(tag))
		operands := readPtrArray(lm, operandsPtr, numOperands)
		return retF(c, cgoThrow(module, tag, operands))
	})
	setFunc(ctx, "_BinaryenRethrow", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		targetPtr := argI(a, 1)
		target := cgoCString(lm.ReadString(targetPtr))
		defer cgoFree(unsafe.Pointer(target))
		return retF(c, cgoRethrow(module, target))
	})
	setFunc(ctx, "_BinaryenRefNull", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoRefNull(argU(a, 0), argU(a, 1)))
	})
	setFunc(ctx, "_BinaryenRefIsNull", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoRefIsNull(argU(a, 0), argU(a, 1)))
	})
	setFunc(ctx, "_BinaryenRefAs", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoRefAs(argU(a, 0), argI32(a, 1), argU(a, 2)))
	})
	setFunc(ctx, "_BinaryenRefFunc", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		funcNamePtr := argI(a, 1)
		typ := argU(a, 2)
		funcName := cgoCString(lm.ReadString(funcNamePtr))
		defer cgoFree(unsafe.Pointer(funcName))
		return retF(c, cgoRefFunc(module, funcName, typ))
	})
	setFunc(ctx, "_BinaryenRefEq", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoRefEq(argU(a, 0), argU(a, 1), argU(a, 2)))
	})
	setFunc(ctx, "_BinaryenTableGet", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		index := argU(a, 2)
		typ := argU(a, 3)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(c, cgoTableGet(module, name, index, typ))
	})
	setFunc(ctx, "_BinaryenTableSet", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		index := argU(a, 2)
		value := argU(a, 3)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(c, cgoTableSet(module, name, index, value))
	})
	setFunc(ctx, "_BinaryenTableSize", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(c, cgoTableSize(module, name))
	})
	setFunc(ctx, "_BinaryenTableGrow", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		value := argU(a, 2)
		delta := argU(a, 3)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(c, cgoTableGrow(module, name, value, delta))
	})
	setFunc(ctx, "_BinaryenTupleMake", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		operandsPtr := argI(a, 1)
		numOperands := argI(a, 2)
		operands := readPtrArray(lm, operandsPtr, numOperands)
		return retF(c, cgoTupleMake(module, operands))
	})
	setFunc(ctx, "_BinaryenTupleExtract", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoTupleExtract(argU(a, 0), argU(a, 1), argI(a, 2)))
	})
	setFunc(ctx, "_BinaryenPop", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoPop(argU(a, 0), argU(a, 1)))
	})
	setFunc(ctx, "_BinaryenMemoryCopy", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
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
		return retF(c, cgoMemoryCopy(module, dest, source, size, destMem, srcMem))
	})
	setFunc(ctx, "_BinaryenMemoryFill", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
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
		return retF(c, cgoMemoryFill(module, dest, value, size, memName))
	})
}

var _ = unsafe.Pointer(nil)
