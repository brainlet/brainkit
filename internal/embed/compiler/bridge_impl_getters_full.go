package asembed

import (
	"unsafe"

	quickjs "github.com/buke/quickjs-go"
)

// registerAllGetterImpls registers all expression getter (and related property accessor)
// bridge functions with real CGo implementations, overriding the stubs from RegisterBinaryenBridge.
func registerAllGetterImpls(ctx *quickjs.Context, lm *LinearMemory) {
	// =====================================================================
	// Block
	// =====================================================================
	setFunc(ctx, "_BinaryenBlockGetName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoBlockGetName(argU(a, 0))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})
	setFunc(ctx, "_BinaryenBlockGetNumChildren", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI(c, cgoBlockGetNumChildren(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenBlockGetChildAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoBlockGetChildAt(argU(a, 0), argI(a, 1)))
	})

	// =====================================================================
	// If
	// =====================================================================
	setFunc(ctx, "_BinaryenIfGetCondition", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoIfGetCondition(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenIfGetIfTrue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoIfGetIfTrue(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenIfGetIfFalse", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoIfGetIfFalse(argU(a, 0)))
	})

	// =====================================================================
	// Loop
	// =====================================================================
	setFunc(ctx, "_BinaryenLoopGetName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoLoopGetName(argU(a, 0))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})
	setFunc(ctx, "_BinaryenLoopGetBody", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoLoopGetBody(argU(a, 0)))
	})

	// =====================================================================
	// Break
	// =====================================================================
	setFunc(ctx, "_BinaryenBreakGetName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoBreakGetName(argU(a, 0))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})
	setFunc(ctx, "_BinaryenBreakGetCondition", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoBreakGetCondition(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenBreakGetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoBreakGetValue(argU(a, 0)))
	})

	// =====================================================================
	// Switch
	// =====================================================================
	setFunc(ctx, "_BinaryenSwitchGetNumNames", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI(c, cgoSwitchGetNumNames(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenSwitchGetNameAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoSwitchGetNameAt(argU(a, 0), argI(a, 1))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})
	setFunc(ctx, "_BinaryenSwitchGetDefaultName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoSwitchGetDefaultName(argU(a, 0))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})
	setFunc(ctx, "_BinaryenSwitchGetCondition", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoSwitchGetCondition(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenSwitchGetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoSwitchGetValue(argU(a, 0)))
	})

	// =====================================================================
	// Call
	// =====================================================================
	setFunc(ctx, "_BinaryenCallGetTarget", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoCallGetTarget(argU(a, 0))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})
	setFunc(ctx, "_BinaryenCallGetNumOperands", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI(c, cgoCallGetNumOperands(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenCallGetOperandAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoCallGetOperandAt(argU(a, 0), argI(a, 1)))
	})
	setFunc(ctx, "_BinaryenCallIsReturn", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retBool(c, cgoCallIsReturn(argU(a, 0)))
	})

	// =====================================================================
	// CallIndirect
	// =====================================================================
	setFunc(ctx, "_BinaryenCallIndirectGetTarget", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoCallIndirectGetTarget(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenCallIndirectGetTable", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoCallIndirectGetTable(argU(a, 0))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})
	setFunc(ctx, "_BinaryenCallIndirectGetNumOperands", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI(c, cgoCallIndirectGetNumOperands(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenCallIndirectGetOperandAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoCallIndirectGetOperandAt(argU(a, 0), argI(a, 1)))
	})
	setFunc(ctx, "_BinaryenCallIndirectIsReturn", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retBool(c, cgoCallIndirectIsReturn(argU(a, 0)))
	})

	// =====================================================================
	// LocalGet
	// =====================================================================
	setFunc(ctx, "_BinaryenLocalGetGetIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI(c, cgoLocalGetGetIndex(argU(a, 0)))
	})

	// =====================================================================
	// GlobalGet (expression)
	// =====================================================================
	setFunc(ctx, "_BinaryenGlobalGetGetName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoGlobalGetGetName(argU(a, 0))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})

	// =====================================================================
	// TableGet (expression)
	// =====================================================================
	setFunc(ctx, "_BinaryenTableGetGetTable", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoTableGetGetTable(argU(a, 0))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})
	setFunc(ctx, "_BinaryenTableGetGetIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoTableGetGetIndex(argU(a, 0)))
	})

	// =====================================================================
	// TableSize (expression)
	// =====================================================================
	setFunc(ctx, "_BinaryenTableSizeGetTable", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoTableSizeGetTable(argU(a, 0))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})

	// =====================================================================
	// TableGrow (expression)
	// =====================================================================
	setFunc(ctx, "_BinaryenTableGrowGetTable", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoTableGrowGetTable(argU(a, 0))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})
	setFunc(ctx, "_BinaryenTableGrowGetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoTableGrowGetValue(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenTableGrowGetDelta", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoTableGrowGetDelta(argU(a, 0)))
	})

	// =====================================================================
	// MemoryGrow
	// =====================================================================
	setFunc(ctx, "_BinaryenMemoryGrowGetDelta", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoMemoryGrowGetDelta(argU(a, 0)))
	})

	// =====================================================================
	// Load
	// =====================================================================
	setFunc(ctx, "_BinaryenLoadIsAtomic", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retBool(c, cgoLoadIsAtomic(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenLoadIsSigned", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retBool(c, cgoLoadIsSigned(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenLoadGetOffset", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retU32(c, cgoLoadGetOffset(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenLoadGetBytes", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retU32(c, cgoLoadGetBytes(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenLoadGetAlign", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retU32(c, cgoLoadGetAlign(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenLoadGetPtr", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoLoadGetPtr(argU(a, 0)))
	})

	// =====================================================================
	// Store
	// =====================================================================
	setFunc(ctx, "_BinaryenStoreIsAtomic", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retBool(c, cgoStoreIsAtomic(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenStoreGetBytes", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retU32(c, cgoStoreGetBytes(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenStoreGetOffset", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retU32(c, cgoStoreGetOffset(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenStoreGetAlign", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retU32(c, cgoStoreGetAlign(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenStoreGetPtr", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStoreGetPtr(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenStoreGetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStoreGetValue(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenStoreGetValueType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStoreGetValueType(argU(a, 0)))
	})

	// =====================================================================
	// Const
	// =====================================================================
	setFunc(ctx, "_BinaryenConstGetValueI32", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI32(c, cgoConstGetValueI32(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenConstGetValueI64Low", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI32(c, cgoConstGetValueI64Low(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenConstGetValueI64High", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI32(c, cgoConstGetValueI64High(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenConstGetValueF32", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return c.NewFloat64(float64(cgoConstGetValueF32(argU(a, 0))))
	})
	setFunc(ctx, "_BinaryenConstGetValueF64", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return c.NewFloat64(cgoConstGetValueF64(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenConstGetValueV128", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		outPtr := argI(a, 1)
		var v128 [16]byte
		cgoConstGetValueV128(expr, &v128)
		for i := 0; i < 16; i++ {
			lm.I32Store8(outPtr+i, v128[i])
		}
		return retVoid(c)
	})

	// =====================================================================
	// Unary
	// =====================================================================
	setFunc(ctx, "_BinaryenUnaryGetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI32(c, cgoUnaryGetOp(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenUnaryGetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoUnaryGetValue(argU(a, 0)))
	})

	// =====================================================================
	// Binary
	// =====================================================================
	setFunc(ctx, "_BinaryenBinaryGetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI32(c, cgoBinaryGetOp(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenBinaryGetLeft", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoBinaryGetLeft(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenBinaryGetRight", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoBinaryGetRight(argU(a, 0)))
	})

	// =====================================================================
	// Select
	// =====================================================================
	setFunc(ctx, "_BinaryenSelectGetIfTrue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoSelectGetIfTrue(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenSelectGetIfFalse", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoSelectGetIfFalse(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenSelectGetCondition", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoSelectGetCondition(argU(a, 0)))
	})

	// =====================================================================
	// Drop
	// =====================================================================
	setFunc(ctx, "_BinaryenDropGetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoDropGetValue(argU(a, 0)))
	})

	// =====================================================================
	// Return
	// =====================================================================
	setFunc(ctx, "_BinaryenReturnGetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoReturnGetValue(argU(a, 0)))
	})

	// =====================================================================
	// AtomicRMW
	// =====================================================================
	setFunc(ctx, "_BinaryenAtomicRMWGetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI32(c, cgoAtomicRMWGetOp(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenAtomicRMWGetBytes", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retU32(c, cgoAtomicRMWGetBytes(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenAtomicRMWGetOffset", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retU32(c, cgoAtomicRMWGetOffset(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenAtomicRMWGetPtr", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoAtomicRMWGetPtr(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenAtomicRMWGetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoAtomicRMWGetValue(argU(a, 0)))
	})

	// =====================================================================
	// AtomicCmpxchg
	// =====================================================================
	setFunc(ctx, "_BinaryenAtomicCmpxchgGetBytes", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retU32(c, cgoAtomicCmpxchgGetBytes(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenAtomicCmpxchgGetOffset", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retU32(c, cgoAtomicCmpxchgGetOffset(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenAtomicCmpxchgGetPtr", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoAtomicCmpxchgGetPtr(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenAtomicCmpxchgGetExpected", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoAtomicCmpxchgGetExpected(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenAtomicCmpxchgGetReplacement", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoAtomicCmpxchgGetReplacement(argU(a, 0)))
	})

	// =====================================================================
	// AtomicWait
	// =====================================================================
	setFunc(ctx, "_BinaryenAtomicWaitGetPtr", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoAtomicWaitGetPtr(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenAtomicWaitGetExpected", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoAtomicWaitGetExpected(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenAtomicWaitGetTimeout", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoAtomicWaitGetTimeout(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenAtomicWaitGetExpectedType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoAtomicWaitGetExpectedType(argU(a, 0)))
	})

	// =====================================================================
	// AtomicNotify
	// =====================================================================
	setFunc(ctx, "_BinaryenAtomicNotifyGetPtr", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoAtomicNotifyGetPtr(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenAtomicNotifyGetNotifyCount", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoAtomicNotifyGetNotifyCount(argU(a, 0)))
	})

	// =====================================================================
	// AtomicFence
	// =====================================================================
	setFunc(ctx, "_BinaryenAtomicFenceGetOrder", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI(c, cgoAtomicFenceGetOrder(argU(a, 0)))
	})

	// =====================================================================
	// SIMDExtract
	// =====================================================================
	setFunc(ctx, "_BinaryenSIMDExtractGetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI32(c, cgoSIMDExtractGetOp(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenSIMDExtractGetVec", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoSIMDExtractGetVec(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenSIMDExtractGetIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI(c, cgoSIMDExtractGetIndex(argU(a, 0)))
	})

	// =====================================================================
	// SIMDReplace
	// =====================================================================
	setFunc(ctx, "_BinaryenSIMDReplaceGetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI32(c, cgoSIMDReplaceGetOp(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenSIMDReplaceGetVec", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoSIMDReplaceGetVec(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenSIMDReplaceGetIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI(c, cgoSIMDReplaceGetIndex(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenSIMDReplaceGetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoSIMDReplaceGetValue(argU(a, 0)))
	})

	// =====================================================================
	// SIMDShuffle
	// =====================================================================
	setFunc(ctx, "_BinaryenSIMDShuffleGetLeft", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoSIMDShuffleGetLeft(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenSIMDShuffleGetRight", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoSIMDShuffleGetRight(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenSIMDShuffleGetMask", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		outPtr := argI(a, 1)
		var mask [16]byte
		cgoSIMDShuffleGetMask(expr, &mask)
		for i := 0; i < 16; i++ {
			lm.I32Store8(outPtr+i, mask[i])
		}
		return retVoid(c)
	})

	// =====================================================================
	// SIMDTernary
	// =====================================================================
	setFunc(ctx, "_BinaryenSIMDTernaryGetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI32(c, cgoSIMDTernaryGetOp(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenSIMDTernaryGetA", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoSIMDTernaryGetA(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenSIMDTernaryGetB", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoSIMDTernaryGetB(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenSIMDTernaryGetC", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoSIMDTernaryGetC(argU(a, 0)))
	})

	// =====================================================================
	// SIMDShift
	// =====================================================================
	setFunc(ctx, "_BinaryenSIMDShiftGetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI32(c, cgoSIMDShiftGetOp(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenSIMDShiftGetVec", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoSIMDShiftGetVec(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenSIMDShiftGetShift", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoSIMDShiftGetShift(argU(a, 0)))
	})

	// =====================================================================
	// SIMDLoad
	// =====================================================================
	setFunc(ctx, "_BinaryenSIMDLoadGetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI32(c, cgoSIMDLoadGetOp(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenSIMDLoadGetOffset", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retU32(c, cgoSIMDLoadGetOffset(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenSIMDLoadGetAlign", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retU32(c, cgoSIMDLoadGetAlign(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenSIMDLoadGetPtr", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoSIMDLoadGetPtr(argU(a, 0)))
	})

	// =====================================================================
	// SIMDLoadStoreLane
	// =====================================================================
	setFunc(ctx, "_BinaryenSIMDLoadStoreLaneGetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI32(c, cgoSIMDLoadStoreLaneGetOp(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenSIMDLoadStoreLaneGetOffset", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retU32(c, cgoSIMDLoadStoreLaneGetOffset(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenSIMDLoadStoreLaneGetAlign", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retU32(c, cgoSIMDLoadStoreLaneGetAlign(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenSIMDLoadStoreLaneGetIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI(c, cgoSIMDLoadStoreLaneGetIndex(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenSIMDLoadStoreLaneGetPtr", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoSIMDLoadStoreLaneGetPtr(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenSIMDLoadStoreLaneGetVec", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoSIMDLoadStoreLaneGetVec(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenSIMDLoadStoreLaneIsStore", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retBool(c, cgoSIMDLoadStoreLaneIsStore(argU(a, 0)))
	})

	// =====================================================================
	// MemoryInit
	// =====================================================================
	setFunc(ctx, "_BinaryenMemoryInitGetSegment", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoMemoryInitGetSegment(argU(a, 0))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})
	setFunc(ctx, "_BinaryenMemoryInitGetDest", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoMemoryInitGetDest(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenMemoryInitGetOffset", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoMemoryInitGetOffset(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenMemoryInitGetSize", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoMemoryInitGetSize(argU(a, 0)))
	})

	// =====================================================================
	// DataDrop
	// =====================================================================
	setFunc(ctx, "_BinaryenDataDropGetSegment", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoDataDropGetSegment(argU(a, 0))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})

	// =====================================================================
	// MemoryCopy
	// =====================================================================
	setFunc(ctx, "_BinaryenMemoryCopyGetDest", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoMemoryCopyGetDest(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenMemoryCopyGetSource", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoMemoryCopyGetSource(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenMemoryCopyGetSize", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoMemoryCopyGetSize(argU(a, 0)))
	})

	// =====================================================================
	// MemoryFill
	// =====================================================================
	setFunc(ctx, "_BinaryenMemoryFillGetDest", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoMemoryFillGetDest(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenMemoryFillGetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoMemoryFillGetValue(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenMemoryFillGetSize", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoMemoryFillGetSize(argU(a, 0)))
	})

	// =====================================================================
	// RefIsNull
	// =====================================================================
	setFunc(ctx, "_BinaryenRefIsNullGetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoRefIsNullGetValue(argU(a, 0)))
	})

	// =====================================================================
	// RefAs
	// =====================================================================
	setFunc(ctx, "_BinaryenRefAsGetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI32(c, cgoRefAsGetOp(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenRefAsGetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoRefAsGetValue(argU(a, 0)))
	})

	// =====================================================================
	// RefFunc
	// =====================================================================
	setFunc(ctx, "_BinaryenRefFuncGetFunc", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoRefFuncGetFunc(argU(a, 0))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})

	// =====================================================================
	// RefEq
	// =====================================================================
	setFunc(ctx, "_BinaryenRefEqGetLeft", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoRefEqGetLeft(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenRefEqGetRight", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoRefEqGetRight(argU(a, 0)))
	})

	// =====================================================================
	// Try
	// =====================================================================
	setFunc(ctx, "_BinaryenTryGetName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoTryGetName(argU(a, 0))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})
	setFunc(ctx, "_BinaryenTryGetBody", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoTryGetBody(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenTryGetNumCatchTags", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI(c, cgoTryGetNumCatchTags(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenTryGetNumCatchBodies", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI(c, cgoTryGetNumCatchBodies(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenTryGetCatchTagAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoTryGetCatchTagAt(argU(a, 0), argI(a, 1))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})
	setFunc(ctx, "_BinaryenTryGetCatchBodyAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoTryGetCatchBodyAt(argU(a, 0), argI(a, 1)))
	})
	setFunc(ctx, "_BinaryenTryGetDelegateTarget", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoTryGetDelegateTarget(argU(a, 0))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})
	setFunc(ctx, "_BinaryenTryIsDelegate", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retBool(c, cgoTryIsDelegate(argU(a, 0)))
	})

	// =====================================================================
	// Throw
	// =====================================================================
	setFunc(ctx, "_BinaryenThrowGetTag", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoThrowGetTag(argU(a, 0))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})
	setFunc(ctx, "_BinaryenThrowGetNumOperands", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI(c, cgoThrowGetNumOperands(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenThrowGetOperandAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoThrowGetOperandAt(argU(a, 0), argI(a, 1)))
	})

	// =====================================================================
	// Rethrow
	// =====================================================================
	setFunc(ctx, "_BinaryenRethrowGetTarget", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoRethrowGetTarget(argU(a, 0))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})

	// =====================================================================
	// TupleMake
	// =====================================================================
	setFunc(ctx, "_BinaryenTupleMakeGetNumOperands", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI(c, cgoTupleMakeGetNumOperands(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenTupleMakeGetOperandAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoTupleMakeGetOperandAt(argU(a, 0), argI(a, 1)))
	})

	// =====================================================================
	// TupleExtract
	// =====================================================================
	setFunc(ctx, "_BinaryenTupleExtractGetTuple", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoTupleExtractGetTuple(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenTupleExtractGetIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI(c, cgoTupleExtractGetIndex(argU(a, 0)))
	})

	// =====================================================================
	// RefI31
	// =====================================================================
	setFunc(ctx, "_BinaryenRefI31GetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoRefI31GetValue(argU(a, 0)))
	})

	// =====================================================================
	// I31Get
	// =====================================================================
	setFunc(ctx, "_BinaryenI31GetGetI31", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoI31GetGetI31(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenI31GetIsSigned", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retBool(c, cgoI31GetIsSigned(argU(a, 0)))
	})

	// =====================================================================
	// CallRef
	// =====================================================================
	setFunc(ctx, "_BinaryenCallRefGetNumOperands", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI(c, cgoCallRefGetNumOperands(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenCallRefGetOperandAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoCallRefGetOperandAt(argU(a, 0), argI(a, 1)))
	})
	setFunc(ctx, "_BinaryenCallRefGetTarget", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoCallRefGetTarget(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenCallRefIsReturn", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retBool(c, cgoCallRefIsReturn(argU(a, 0)))
	})

	// =====================================================================
	// RefTest
	// =====================================================================
	setFunc(ctx, "_BinaryenRefTestGetRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoRefTestGetRef(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenRefTestGetCastType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoRefTestGetCastType(argU(a, 0)))
	})

	// =====================================================================
	// RefCast
	// =====================================================================
	setFunc(ctx, "_BinaryenRefCastGetRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoRefCastGetRef(argU(a, 0)))
	})

	// =====================================================================
	// BrOn
	// =====================================================================
	setFunc(ctx, "_BinaryenBrOnGetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI32(c, cgoBrOnGetOp(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenBrOnGetName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoBrOnGetName(argU(a, 0))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})
	setFunc(ctx, "_BinaryenBrOnGetRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoBrOnGetRef(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenBrOnGetCastType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoBrOnGetCastType(argU(a, 0)))
	})

	// =====================================================================
	// StructNew
	// =====================================================================
	setFunc(ctx, "_BinaryenStructNewGetNumOperands", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI(c, cgoStructNewGetNumOperands(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenStructNewGetOperandAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStructNewGetOperandAt(argU(a, 0), argI(a, 1)))
	})

	// =====================================================================
	// StructGet
	// =====================================================================
	setFunc(ctx, "_BinaryenStructGetGetIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI(c, cgoStructGetGetIndex(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenStructGetGetRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStructGetGetRef(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenStructGetIsSigned", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retBool(c, cgoStructGetIsSigned(argU(a, 0)))
	})

	// =====================================================================
	// ArrayNew
	// =====================================================================
	setFunc(ctx, "_BinaryenArrayNewGetInit", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoArrayNewGetInit(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenArrayNewGetSize", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoArrayNewGetSize(argU(a, 0)))
	})

	// =====================================================================
	// ArrayNewFixed
	// =====================================================================
	setFunc(ctx, "_BinaryenArrayNewFixedGetNumValues", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI(c, cgoArrayNewFixedGetNumValues(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenArrayNewFixedGetValueAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoArrayNewFixedGetValueAt(argU(a, 0), argI(a, 1)))
	})

	// =====================================================================
	// ArrayGet
	// =====================================================================
	setFunc(ctx, "_BinaryenArrayGetGetRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoArrayGetGetRef(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenArrayGetGetIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoArrayGetGetIndex(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenArrayGetIsSigned", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retBool(c, cgoArrayGetIsSigned(argU(a, 0)))
	})

	// =====================================================================
	// ArrayLen
	// =====================================================================
	setFunc(ctx, "_BinaryenArrayLenGetRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoArrayLenGetRef(argU(a, 0)))
	})

	// =====================================================================
	// ArrayCopy
	// =====================================================================
	setFunc(ctx, "_BinaryenArrayCopyGetDestRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoArrayCopyGetDestRef(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenArrayCopyGetDestIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoArrayCopyGetDestIndex(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenArrayCopyGetSrcRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoArrayCopyGetSrcRef(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenArrayCopyGetSrcIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoArrayCopyGetSrcIndex(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenArrayCopyGetLength", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoArrayCopyGetLength(argU(a, 0)))
	})

	// =====================================================================
	// StringNew
	// =====================================================================
	setFunc(ctx, "_BinaryenStringNewGetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI32(c, cgoStringNewGetOp(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenStringNewGetRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStringNewGetRef(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenStringNewGetStart", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStringNewGetStart(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenStringNewGetEnd", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStringNewGetEnd(argU(a, 0)))
	})

	// =====================================================================
	// StringConst
	// =====================================================================
	setFunc(ctx, "_BinaryenStringConstGetString", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoStringConstGetString(argU(a, 0))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})

	// =====================================================================
	// StringMeasure
	// =====================================================================
	setFunc(ctx, "_BinaryenStringMeasureGetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI32(c, cgoStringMeasureGetOp(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenStringMeasureGetRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStringMeasureGetRef(argU(a, 0)))
	})

	// =====================================================================
	// StringEncode
	// =====================================================================
	setFunc(ctx, "_BinaryenStringEncodeGetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI32(c, cgoStringEncodeGetOp(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenStringEncodeGetStr", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStringEncodeGetStr(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenStringEncodeGetArray", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStringEncodeGetArray(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenStringEncodeGetStart", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStringEncodeGetStart(argU(a, 0)))
	})

	// =====================================================================
	// StringConcat
	// =====================================================================
	setFunc(ctx, "_BinaryenStringConcatGetLeft", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStringConcatGetLeft(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenStringConcatGetRight", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStringConcatGetRight(argU(a, 0)))
	})

	// =====================================================================
	// StringEq
	// =====================================================================
	setFunc(ctx, "_BinaryenStringEqGetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI32(c, cgoStringEqGetOp(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenStringEqGetLeft", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStringEqGetLeft(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenStringEqGetRight", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStringEqGetRight(argU(a, 0)))
	})

	// =====================================================================
	// StringWTF16Get
	// =====================================================================
	setFunc(ctx, "_BinaryenStringWTF16GetGetRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStringWTF16GetGetRef(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenStringWTF16GetGetPos", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStringWTF16GetGetPos(argU(a, 0)))
	})

	// =====================================================================
	// StringSliceWTF
	// =====================================================================
	setFunc(ctx, "_BinaryenStringSliceWTFGetRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStringSliceWTFGetRef(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenStringSliceWTFGetStart", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStringSliceWTFGetStart(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenStringSliceWTFGetEnd", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStringSliceWTFGetEnd(argU(a, 0)))
	})

	// =====================================================================
	// Global object getters (non-expression - takes GlobalRef)
	// =====================================================================
	setFunc(ctx, "_BinaryenGlobalGetName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoGlobalObjGetName(argU(a, 0))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})
	setFunc(ctx, "_BinaryenGlobalGetType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoGlobalObjGetType(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenGlobalIsMutable", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retBool(c, cgoGlobalObjIsMutable(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenGlobalGetInitExpr", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoGlobalObjGetInitExpr(argU(a, 0)))
	})

	// =====================================================================
	// Export getters
	// =====================================================================
	setFunc(ctx, "_BinaryenExportGetKind", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI(c, cgoExportGetKind(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenExportGetName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoExportGetName(argU(a, 0))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})
	setFunc(ctx, "_BinaryenExportGetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoExportGetValue(argU(a, 0))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})

	// =====================================================================
	// Function type getter
	// =====================================================================
	setFunc(ctx, "_BinaryenFunctionGetType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoFunctionObjGetType(argU(a, 0)))
	})

	// =====================================================================
	// Module-level indexed getters
	// =====================================================================
	setFunc(ctx, "_BinaryenGetExportByIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoGetExportByIndex(argU(a, 0), argI(a, 1)))
	})
	setFunc(ctx, "_BinaryenGetGlobalByIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoGetGlobalByIndex(argU(a, 0), argI(a, 1)))
	})
	setFunc(ctx, "_BinaryenGetTableByIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoGetTableByIndex(argU(a, 0), argI(a, 1)))
	})
	setFunc(ctx, "_BinaryenGetNumElementSegments", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI(c, cgoGetNumElementSegments(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenGetElementSegment", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		return retF(c, cgoGetElementSegment(module, cName))
	})
	setFunc(ctx, "_BinaryenGetElementSegmentByIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoGetElementSegmentByIndex(argU(a, 0), argI(a, 1)))
	})
	setFunc(ctx, "_BinaryenGetNumMemorySegments", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retU32(c, cgoGetNumMemorySegments(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenGetMemorySegmentByteOffset", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		return retU32(c, cgoGetMemorySegmentByteOffset(module, cName))
	})
	setFunc(ctx, "_BinaryenGetMemorySegmentByteLength", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		return retU32(c, cgoGetMemorySegmentByteLength(module, cName))
	})
	setFunc(ctx, "_BinaryenModuleGetDebugInfoFileName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoModuleGetDebugInfoFileName(argU(a, 0), argI(a, 1))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})

	// =====================================================================
	// Tag getters
	// =====================================================================
	setFunc(ctx, "_BinaryenTagGetName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoTagGetName(argU(a, 0))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})
	setFunc(ctx, "_BinaryenTagGetParams", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoTagGetParams(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenTagGetResults", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoTagGetResults(argU(a, 0)))
	})

	// =====================================================================
	// Table object getters (non-expression - takes TableRef)
	// =====================================================================
	setFunc(ctx, "_BinaryenTableGetName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cStr := cgoTableObjGetName(argU(a, 0))
		if cStr == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})
	setFunc(ctx, "_BinaryenTableGetInitial", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI(c, cgoTableObjGetInitial(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenTableGetMax", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retI(c, cgoTableObjGetMax(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenTableGetType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoTableObjGetType(argU(a, 0)))
	})
}

// ensure unsafe import is used
var _ = unsafe.Pointer(nil)
