package asembed

import (
	"unsafe"

	"github.com/fastschema/qjs"
)

// registerAllGetterImpls registers all expression getter (and related property accessor)
// bridge functions with real CGo implementations, overriding the stubs from RegisterBinaryenBridge.
func registerAllGetterImpls(ctx *qjs.Context, lm *LinearMemory) {
	// =====================================================================
	// Block
	// =====================================================================
	ctx.SetFunc("_BinaryenBlockGetName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoBlockGetName(argU(a, 0))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})
	ctx.SetFunc("_BinaryenBlockGetNumChildren", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI(this.Context(), cgoBlockGetNumChildren(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenBlockGetChildAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoBlockGetChildAt(argU(a, 0), argI(a, 1)))
	})

	// =====================================================================
	// If
	// =====================================================================
	ctx.SetFunc("_BinaryenIfGetCondition", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoIfGetCondition(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenIfGetIfTrue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoIfGetIfTrue(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenIfGetIfFalse", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoIfGetIfFalse(argU(a, 0)))
	})

	// =====================================================================
	// Loop
	// =====================================================================
	ctx.SetFunc("_BinaryenLoopGetName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoLoopGetName(argU(a, 0))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})
	ctx.SetFunc("_BinaryenLoopGetBody", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoLoopGetBody(argU(a, 0)))
	})

	// =====================================================================
	// Break
	// =====================================================================
	ctx.SetFunc("_BinaryenBreakGetName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoBreakGetName(argU(a, 0))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})
	ctx.SetFunc("_BinaryenBreakGetCondition", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoBreakGetCondition(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenBreakGetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoBreakGetValue(argU(a, 0)))
	})

	// =====================================================================
	// Switch
	// =====================================================================
	ctx.SetFunc("_BinaryenSwitchGetNumNames", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI(this.Context(), cgoSwitchGetNumNames(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenSwitchGetNameAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoSwitchGetNameAt(argU(a, 0), argI(a, 1))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})
	ctx.SetFunc("_BinaryenSwitchGetDefaultName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoSwitchGetDefaultName(argU(a, 0))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})
	ctx.SetFunc("_BinaryenSwitchGetCondition", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoSwitchGetCondition(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenSwitchGetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoSwitchGetValue(argU(a, 0)))
	})

	// =====================================================================
	// Call
	// =====================================================================
	ctx.SetFunc("_BinaryenCallGetTarget", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoCallGetTarget(argU(a, 0))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})
	ctx.SetFunc("_BinaryenCallGetNumOperands", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI(this.Context(), cgoCallGetNumOperands(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenCallGetOperandAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoCallGetOperandAt(argU(a, 0), argI(a, 1)))
	})
	ctx.SetFunc("_BinaryenCallIsReturn", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retBool(this.Context(), cgoCallIsReturn(argU(a, 0)))
	})

	// =====================================================================
	// CallIndirect
	// =====================================================================
	ctx.SetFunc("_BinaryenCallIndirectGetTarget", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoCallIndirectGetTarget(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenCallIndirectGetTable", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoCallIndirectGetTable(argU(a, 0))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})
	ctx.SetFunc("_BinaryenCallIndirectGetNumOperands", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI(this.Context(), cgoCallIndirectGetNumOperands(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenCallIndirectGetOperandAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoCallIndirectGetOperandAt(argU(a, 0), argI(a, 1)))
	})
	ctx.SetFunc("_BinaryenCallIndirectIsReturn", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retBool(this.Context(), cgoCallIndirectIsReturn(argU(a, 0)))
	})

	// =====================================================================
	// LocalGet
	// =====================================================================
	ctx.SetFunc("_BinaryenLocalGetGetIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI(this.Context(), cgoLocalGetGetIndex(argU(a, 0)))
	})

	// =====================================================================
	// GlobalGet (expression)
	// =====================================================================
	ctx.SetFunc("_BinaryenGlobalGetGetName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoGlobalGetGetName(argU(a, 0))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})

	// =====================================================================
	// TableGet (expression)
	// =====================================================================
	ctx.SetFunc("_BinaryenTableGetGetTable", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoTableGetGetTable(argU(a, 0))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})
	ctx.SetFunc("_BinaryenTableGetGetIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoTableGetGetIndex(argU(a, 0)))
	})

	// =====================================================================
	// TableSize (expression)
	// =====================================================================
	ctx.SetFunc("_BinaryenTableSizeGetTable", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoTableSizeGetTable(argU(a, 0))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})

	// =====================================================================
	// TableGrow (expression)
	// =====================================================================
	ctx.SetFunc("_BinaryenTableGrowGetTable", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoTableGrowGetTable(argU(a, 0))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})
	ctx.SetFunc("_BinaryenTableGrowGetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoTableGrowGetValue(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenTableGrowGetDelta", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoTableGrowGetDelta(argU(a, 0)))
	})

	// =====================================================================
	// MemoryGrow
	// =====================================================================
	ctx.SetFunc("_BinaryenMemoryGrowGetDelta", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoMemoryGrowGetDelta(argU(a, 0)))
	})

	// =====================================================================
	// Load
	// =====================================================================
	ctx.SetFunc("_BinaryenLoadIsAtomic", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retBool(this.Context(), cgoLoadIsAtomic(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenLoadIsSigned", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retBool(this.Context(), cgoLoadIsSigned(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenLoadGetOffset", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retU32(this.Context(), cgoLoadGetOffset(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenLoadGetBytes", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retU32(this.Context(), cgoLoadGetBytes(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenLoadGetAlign", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retU32(this.Context(), cgoLoadGetAlign(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenLoadGetPtr", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoLoadGetPtr(argU(a, 0)))
	})

	// =====================================================================
	// Store
	// =====================================================================
	ctx.SetFunc("_BinaryenStoreIsAtomic", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retBool(this.Context(), cgoStoreIsAtomic(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenStoreGetBytes", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retU32(this.Context(), cgoStoreGetBytes(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenStoreGetOffset", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retU32(this.Context(), cgoStoreGetOffset(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenStoreGetAlign", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retU32(this.Context(), cgoStoreGetAlign(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenStoreGetPtr", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStoreGetPtr(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenStoreGetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStoreGetValue(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenStoreGetValueType", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStoreGetValueType(argU(a, 0)))
	})

	// =====================================================================
	// Const
	// =====================================================================
	ctx.SetFunc("_BinaryenConstGetValueI32", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI32(this.Context(), cgoConstGetValueI32(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenConstGetValueI64Low", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI32(this.Context(), cgoConstGetValueI64Low(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenConstGetValueI64High", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI32(this.Context(), cgoConstGetValueI64High(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenConstGetValueF32", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return this.Context().NewFloat64(float64(cgoConstGetValueF32(argU(a, 0)))), nil
	})
	ctx.SetFunc("_BinaryenConstGetValueF64", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return this.Context().NewFloat64(cgoConstGetValueF64(argU(a, 0))), nil
	})
	ctx.SetFunc("_BinaryenConstGetValueV128", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		outPtr := argI(a, 1)
		var v128 [16]byte
		cgoConstGetValueV128(expr, &v128)
		for i := 0; i < 16; i++ {
			lm.I32Store8(outPtr+i, v128[i])
		}
		return retVoid(this.Context())
	})

	// =====================================================================
	// Unary
	// =====================================================================
	ctx.SetFunc("_BinaryenUnaryGetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI32(this.Context(), cgoUnaryGetOp(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenUnaryGetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoUnaryGetValue(argU(a, 0)))
	})

	// =====================================================================
	// Binary
	// =====================================================================
	ctx.SetFunc("_BinaryenBinaryGetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI32(this.Context(), cgoBinaryGetOp(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenBinaryGetLeft", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoBinaryGetLeft(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenBinaryGetRight", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoBinaryGetRight(argU(a, 0)))
	})

	// =====================================================================
	// Select
	// =====================================================================
	ctx.SetFunc("_BinaryenSelectGetIfTrue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoSelectGetIfTrue(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenSelectGetIfFalse", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoSelectGetIfFalse(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenSelectGetCondition", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoSelectGetCondition(argU(a, 0)))
	})

	// =====================================================================
	// Drop
	// =====================================================================
	ctx.SetFunc("_BinaryenDropGetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoDropGetValue(argU(a, 0)))
	})

	// =====================================================================
	// Return
	// =====================================================================
	ctx.SetFunc("_BinaryenReturnGetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoReturnGetValue(argU(a, 0)))
	})

	// =====================================================================
	// AtomicRMW
	// =====================================================================
	ctx.SetFunc("_BinaryenAtomicRMWGetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI32(this.Context(), cgoAtomicRMWGetOp(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenAtomicRMWGetBytes", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retU32(this.Context(), cgoAtomicRMWGetBytes(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenAtomicRMWGetOffset", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retU32(this.Context(), cgoAtomicRMWGetOffset(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenAtomicRMWGetPtr", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoAtomicRMWGetPtr(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenAtomicRMWGetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoAtomicRMWGetValue(argU(a, 0)))
	})

	// =====================================================================
	// AtomicCmpxchg
	// =====================================================================
	ctx.SetFunc("_BinaryenAtomicCmpxchgGetBytes", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retU32(this.Context(), cgoAtomicCmpxchgGetBytes(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenAtomicCmpxchgGetOffset", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retU32(this.Context(), cgoAtomicCmpxchgGetOffset(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenAtomicCmpxchgGetPtr", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoAtomicCmpxchgGetPtr(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenAtomicCmpxchgGetExpected", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoAtomicCmpxchgGetExpected(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenAtomicCmpxchgGetReplacement", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoAtomicCmpxchgGetReplacement(argU(a, 0)))
	})

	// =====================================================================
	// AtomicWait
	// =====================================================================
	ctx.SetFunc("_BinaryenAtomicWaitGetPtr", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoAtomicWaitGetPtr(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenAtomicWaitGetExpected", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoAtomicWaitGetExpected(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenAtomicWaitGetTimeout", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoAtomicWaitGetTimeout(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenAtomicWaitGetExpectedType", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoAtomicWaitGetExpectedType(argU(a, 0)))
	})

	// =====================================================================
	// AtomicNotify
	// =====================================================================
	ctx.SetFunc("_BinaryenAtomicNotifyGetPtr", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoAtomicNotifyGetPtr(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenAtomicNotifyGetNotifyCount", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoAtomicNotifyGetNotifyCount(argU(a, 0)))
	})

	// =====================================================================
	// AtomicFence
	// =====================================================================
	ctx.SetFunc("_BinaryenAtomicFenceGetOrder", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI(this.Context(), cgoAtomicFenceGetOrder(argU(a, 0)))
	})

	// =====================================================================
	// SIMDExtract
	// =====================================================================
	ctx.SetFunc("_BinaryenSIMDExtractGetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI32(this.Context(), cgoSIMDExtractGetOp(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenSIMDExtractGetVec", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoSIMDExtractGetVec(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenSIMDExtractGetIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI(this.Context(), cgoSIMDExtractGetIndex(argU(a, 0)))
	})

	// =====================================================================
	// SIMDReplace
	// =====================================================================
	ctx.SetFunc("_BinaryenSIMDReplaceGetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI32(this.Context(), cgoSIMDReplaceGetOp(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenSIMDReplaceGetVec", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoSIMDReplaceGetVec(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenSIMDReplaceGetIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI(this.Context(), cgoSIMDReplaceGetIndex(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenSIMDReplaceGetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoSIMDReplaceGetValue(argU(a, 0)))
	})

	// =====================================================================
	// SIMDShuffle
	// =====================================================================
	ctx.SetFunc("_BinaryenSIMDShuffleGetLeft", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoSIMDShuffleGetLeft(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenSIMDShuffleGetRight", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoSIMDShuffleGetRight(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenSIMDShuffleGetMask", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		outPtr := argI(a, 1)
		var mask [16]byte
		cgoSIMDShuffleGetMask(expr, &mask)
		for i := 0; i < 16; i++ {
			lm.I32Store8(outPtr+i, mask[i])
		}
		return retVoid(this.Context())
	})

	// =====================================================================
	// SIMDTernary
	// =====================================================================
	ctx.SetFunc("_BinaryenSIMDTernaryGetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI32(this.Context(), cgoSIMDTernaryGetOp(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenSIMDTernaryGetA", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoSIMDTernaryGetA(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenSIMDTernaryGetB", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoSIMDTernaryGetB(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenSIMDTernaryGetC", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoSIMDTernaryGetC(argU(a, 0)))
	})

	// =====================================================================
	// SIMDShift
	// =====================================================================
	ctx.SetFunc("_BinaryenSIMDShiftGetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI32(this.Context(), cgoSIMDShiftGetOp(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenSIMDShiftGetVec", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoSIMDShiftGetVec(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenSIMDShiftGetShift", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoSIMDShiftGetShift(argU(a, 0)))
	})

	// =====================================================================
	// SIMDLoad
	// =====================================================================
	ctx.SetFunc("_BinaryenSIMDLoadGetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI32(this.Context(), cgoSIMDLoadGetOp(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenSIMDLoadGetOffset", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retU32(this.Context(), cgoSIMDLoadGetOffset(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenSIMDLoadGetAlign", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retU32(this.Context(), cgoSIMDLoadGetAlign(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenSIMDLoadGetPtr", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoSIMDLoadGetPtr(argU(a, 0)))
	})

	// =====================================================================
	// SIMDLoadStoreLane
	// =====================================================================
	ctx.SetFunc("_BinaryenSIMDLoadStoreLaneGetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI32(this.Context(), cgoSIMDLoadStoreLaneGetOp(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenSIMDLoadStoreLaneGetOffset", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retU32(this.Context(), cgoSIMDLoadStoreLaneGetOffset(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenSIMDLoadStoreLaneGetAlign", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retU32(this.Context(), cgoSIMDLoadStoreLaneGetAlign(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenSIMDLoadStoreLaneGetIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI(this.Context(), cgoSIMDLoadStoreLaneGetIndex(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenSIMDLoadStoreLaneGetPtr", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoSIMDLoadStoreLaneGetPtr(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenSIMDLoadStoreLaneGetVec", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoSIMDLoadStoreLaneGetVec(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenSIMDLoadStoreLaneIsStore", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retBool(this.Context(), cgoSIMDLoadStoreLaneIsStore(argU(a, 0)))
	})

	// =====================================================================
	// MemoryInit
	// =====================================================================
	ctx.SetFunc("_BinaryenMemoryInitGetSegment", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoMemoryInitGetSegment(argU(a, 0))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})
	ctx.SetFunc("_BinaryenMemoryInitGetDest", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoMemoryInitGetDest(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenMemoryInitGetOffset", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoMemoryInitGetOffset(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenMemoryInitGetSize", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoMemoryInitGetSize(argU(a, 0)))
	})

	// =====================================================================
	// DataDrop
	// =====================================================================
	ctx.SetFunc("_BinaryenDataDropGetSegment", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoDataDropGetSegment(argU(a, 0))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})

	// =====================================================================
	// MemoryCopy
	// =====================================================================
	ctx.SetFunc("_BinaryenMemoryCopyGetDest", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoMemoryCopyGetDest(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenMemoryCopyGetSource", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoMemoryCopyGetSource(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenMemoryCopyGetSize", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoMemoryCopyGetSize(argU(a, 0)))
	})

	// =====================================================================
	// MemoryFill
	// =====================================================================
	ctx.SetFunc("_BinaryenMemoryFillGetDest", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoMemoryFillGetDest(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenMemoryFillGetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoMemoryFillGetValue(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenMemoryFillGetSize", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoMemoryFillGetSize(argU(a, 0)))
	})

	// =====================================================================
	// RefIsNull
	// =====================================================================
	ctx.SetFunc("_BinaryenRefIsNullGetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoRefIsNullGetValue(argU(a, 0)))
	})

	// =====================================================================
	// RefAs
	// =====================================================================
	ctx.SetFunc("_BinaryenRefAsGetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI32(this.Context(), cgoRefAsGetOp(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenRefAsGetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoRefAsGetValue(argU(a, 0)))
	})

	// =====================================================================
	// RefFunc
	// =====================================================================
	ctx.SetFunc("_BinaryenRefFuncGetFunc", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoRefFuncGetFunc(argU(a, 0))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})

	// =====================================================================
	// RefEq
	// =====================================================================
	ctx.SetFunc("_BinaryenRefEqGetLeft", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoRefEqGetLeft(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenRefEqGetRight", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoRefEqGetRight(argU(a, 0)))
	})

	// =====================================================================
	// Try
	// =====================================================================
	ctx.SetFunc("_BinaryenTryGetName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoTryGetName(argU(a, 0))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})
	ctx.SetFunc("_BinaryenTryGetBody", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoTryGetBody(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenTryGetNumCatchTags", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI(this.Context(), cgoTryGetNumCatchTags(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenTryGetNumCatchBodies", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI(this.Context(), cgoTryGetNumCatchBodies(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenTryGetCatchTagAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoTryGetCatchTagAt(argU(a, 0), argI(a, 1))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})
	ctx.SetFunc("_BinaryenTryGetCatchBodyAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoTryGetCatchBodyAt(argU(a, 0), argI(a, 1)))
	})
	ctx.SetFunc("_BinaryenTryGetDelegateTarget", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoTryGetDelegateTarget(argU(a, 0))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})
	ctx.SetFunc("_BinaryenTryIsDelegate", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retBool(this.Context(), cgoTryIsDelegate(argU(a, 0)))
	})

	// =====================================================================
	// Throw
	// =====================================================================
	ctx.SetFunc("_BinaryenThrowGetTag", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoThrowGetTag(argU(a, 0))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})
	ctx.SetFunc("_BinaryenThrowGetNumOperands", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI(this.Context(), cgoThrowGetNumOperands(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenThrowGetOperandAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoThrowGetOperandAt(argU(a, 0), argI(a, 1)))
	})

	// =====================================================================
	// Rethrow
	// =====================================================================
	ctx.SetFunc("_BinaryenRethrowGetTarget", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoRethrowGetTarget(argU(a, 0))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})

	// =====================================================================
	// TupleMake
	// =====================================================================
	ctx.SetFunc("_BinaryenTupleMakeGetNumOperands", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI(this.Context(), cgoTupleMakeGetNumOperands(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenTupleMakeGetOperandAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoTupleMakeGetOperandAt(argU(a, 0), argI(a, 1)))
	})

	// =====================================================================
	// TupleExtract
	// =====================================================================
	ctx.SetFunc("_BinaryenTupleExtractGetTuple", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoTupleExtractGetTuple(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenTupleExtractGetIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI(this.Context(), cgoTupleExtractGetIndex(argU(a, 0)))
	})

	// =====================================================================
	// RefI31
	// =====================================================================
	ctx.SetFunc("_BinaryenRefI31GetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoRefI31GetValue(argU(a, 0)))
	})

	// =====================================================================
	// I31Get
	// =====================================================================
	ctx.SetFunc("_BinaryenI31GetGetI31", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoI31GetGetI31(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenI31GetIsSigned", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retBool(this.Context(), cgoI31GetIsSigned(argU(a, 0)))
	})

	// =====================================================================
	// CallRef
	// =====================================================================
	ctx.SetFunc("_BinaryenCallRefGetNumOperands", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI(this.Context(), cgoCallRefGetNumOperands(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenCallRefGetOperandAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoCallRefGetOperandAt(argU(a, 0), argI(a, 1)))
	})
	ctx.SetFunc("_BinaryenCallRefGetTarget", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoCallRefGetTarget(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenCallRefIsReturn", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retBool(this.Context(), cgoCallRefIsReturn(argU(a, 0)))
	})

	// =====================================================================
	// RefTest
	// =====================================================================
	ctx.SetFunc("_BinaryenRefTestGetRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoRefTestGetRef(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenRefTestGetCastType", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoRefTestGetCastType(argU(a, 0)))
	})

	// =====================================================================
	// RefCast
	// =====================================================================
	ctx.SetFunc("_BinaryenRefCastGetRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoRefCastGetRef(argU(a, 0)))
	})

	// =====================================================================
	// BrOn
	// =====================================================================
	ctx.SetFunc("_BinaryenBrOnGetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI32(this.Context(), cgoBrOnGetOp(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenBrOnGetName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoBrOnGetName(argU(a, 0))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})
	ctx.SetFunc("_BinaryenBrOnGetRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoBrOnGetRef(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenBrOnGetCastType", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoBrOnGetCastType(argU(a, 0)))
	})

	// =====================================================================
	// StructNew
	// =====================================================================
	ctx.SetFunc("_BinaryenStructNewGetNumOperands", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI(this.Context(), cgoStructNewGetNumOperands(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenStructNewGetOperandAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStructNewGetOperandAt(argU(a, 0), argI(a, 1)))
	})

	// =====================================================================
	// StructGet
	// =====================================================================
	ctx.SetFunc("_BinaryenStructGetGetIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI(this.Context(), cgoStructGetGetIndex(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenStructGetGetRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStructGetGetRef(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenStructGetIsSigned", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retBool(this.Context(), cgoStructGetIsSigned(argU(a, 0)))
	})

	// =====================================================================
	// ArrayNew
	// =====================================================================
	ctx.SetFunc("_BinaryenArrayNewGetInit", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoArrayNewGetInit(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenArrayNewGetSize", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoArrayNewGetSize(argU(a, 0)))
	})

	// =====================================================================
	// ArrayNewFixed
	// =====================================================================
	ctx.SetFunc("_BinaryenArrayNewFixedGetNumValues", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI(this.Context(), cgoArrayNewFixedGetNumValues(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenArrayNewFixedGetValueAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoArrayNewFixedGetValueAt(argU(a, 0), argI(a, 1)))
	})

	// =====================================================================
	// ArrayGet
	// =====================================================================
	ctx.SetFunc("_BinaryenArrayGetGetRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoArrayGetGetRef(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenArrayGetGetIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoArrayGetGetIndex(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenArrayGetIsSigned", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retBool(this.Context(), cgoArrayGetIsSigned(argU(a, 0)))
	})

	// =====================================================================
	// ArrayLen
	// =====================================================================
	ctx.SetFunc("_BinaryenArrayLenGetRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoArrayLenGetRef(argU(a, 0)))
	})

	// =====================================================================
	// ArrayCopy
	// =====================================================================
	ctx.SetFunc("_BinaryenArrayCopyGetDestRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoArrayCopyGetDestRef(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenArrayCopyGetDestIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoArrayCopyGetDestIndex(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenArrayCopyGetSrcRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoArrayCopyGetSrcRef(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenArrayCopyGetSrcIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoArrayCopyGetSrcIndex(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenArrayCopyGetLength", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoArrayCopyGetLength(argU(a, 0)))
	})

	// =====================================================================
	// StringNew
	// =====================================================================
	ctx.SetFunc("_BinaryenStringNewGetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI32(this.Context(), cgoStringNewGetOp(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenStringNewGetRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStringNewGetRef(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenStringNewGetStart", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStringNewGetStart(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenStringNewGetEnd", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStringNewGetEnd(argU(a, 0)))
	})

	// =====================================================================
	// StringConst
	// =====================================================================
	ctx.SetFunc("_BinaryenStringConstGetString", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoStringConstGetString(argU(a, 0))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})

	// =====================================================================
	// StringMeasure
	// =====================================================================
	ctx.SetFunc("_BinaryenStringMeasureGetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI32(this.Context(), cgoStringMeasureGetOp(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenStringMeasureGetRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStringMeasureGetRef(argU(a, 0)))
	})

	// =====================================================================
	// StringEncode
	// =====================================================================
	ctx.SetFunc("_BinaryenStringEncodeGetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI32(this.Context(), cgoStringEncodeGetOp(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenStringEncodeGetStr", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStringEncodeGetStr(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenStringEncodeGetArray", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStringEncodeGetArray(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenStringEncodeGetStart", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStringEncodeGetStart(argU(a, 0)))
	})

	// =====================================================================
	// StringConcat
	// =====================================================================
	ctx.SetFunc("_BinaryenStringConcatGetLeft", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStringConcatGetLeft(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenStringConcatGetRight", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStringConcatGetRight(argU(a, 0)))
	})

	// =====================================================================
	// StringEq
	// =====================================================================
	ctx.SetFunc("_BinaryenStringEqGetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI32(this.Context(), cgoStringEqGetOp(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenStringEqGetLeft", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStringEqGetLeft(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenStringEqGetRight", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStringEqGetRight(argU(a, 0)))
	})

	// =====================================================================
	// StringWTF16Get
	// =====================================================================
	ctx.SetFunc("_BinaryenStringWTF16GetGetRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStringWTF16GetGetRef(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenStringWTF16GetGetPos", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStringWTF16GetGetPos(argU(a, 0)))
	})

	// =====================================================================
	// StringSliceWTF
	// =====================================================================
	ctx.SetFunc("_BinaryenStringSliceWTFGetRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStringSliceWTFGetRef(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenStringSliceWTFGetStart", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStringSliceWTFGetStart(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenStringSliceWTFGetEnd", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStringSliceWTFGetEnd(argU(a, 0)))
	})

	// =====================================================================
	// Global object getters (non-expression - takes GlobalRef)
	// =====================================================================
	ctx.SetFunc("_BinaryenGlobalGetName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoGlobalObjGetName(argU(a, 0))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})
	ctx.SetFunc("_BinaryenGlobalGetType", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoGlobalObjGetType(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenGlobalIsMutable", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retBool(this.Context(), cgoGlobalObjIsMutable(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenGlobalGetInitExpr", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoGlobalObjGetInitExpr(argU(a, 0)))
	})

	// =====================================================================
	// Export getters
	// =====================================================================
	ctx.SetFunc("_BinaryenExportGetKind", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI(this.Context(), cgoExportGetKind(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenExportGetName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoExportGetName(argU(a, 0))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})
	ctx.SetFunc("_BinaryenExportGetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoExportGetValue(argU(a, 0))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})

	// =====================================================================
	// Function type getter
	// =====================================================================
	ctx.SetFunc("_BinaryenFunctionGetType", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoFunctionObjGetType(argU(a, 0)))
	})

	// =====================================================================
	// Module-level indexed getters
	// =====================================================================
	ctx.SetFunc("_BinaryenGetExportByIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoGetExportByIndex(argU(a, 0), argI(a, 1)))
	})
	ctx.SetFunc("_BinaryenGetGlobalByIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoGetGlobalByIndex(argU(a, 0), argI(a, 1)))
	})
	ctx.SetFunc("_BinaryenGetTableByIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoGetTableByIndex(argU(a, 0), argI(a, 1)))
	})
	ctx.SetFunc("_BinaryenGetNumElementSegments", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI(this.Context(), cgoGetNumElementSegments(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenGetElementSegment", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		return retF(this.Context(), cgoGetElementSegment(module, cName))
	})
	ctx.SetFunc("_BinaryenGetElementSegmentByIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoGetElementSegmentByIndex(argU(a, 0), argI(a, 1)))
	})
	ctx.SetFunc("_BinaryenGetNumMemorySegments", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retU32(this.Context(), cgoGetNumMemorySegments(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenGetMemorySegmentByteOffset", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		return retU32(this.Context(), cgoGetMemorySegmentByteOffset(module, cName))
	})
	ctx.SetFunc("_BinaryenGetMemorySegmentByteLength", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		return retU32(this.Context(), cgoGetMemorySegmentByteLength(module, cName))
	})
	ctx.SetFunc("_BinaryenModuleGetDebugInfoFileName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoModuleGetDebugInfoFileName(argU(a, 0), argI(a, 1))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})

	// =====================================================================
	// Tag getters
	// =====================================================================
	ctx.SetFunc("_BinaryenTagGetName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoTagGetName(argU(a, 0))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})
	ctx.SetFunc("_BinaryenTagGetParams", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoTagGetParams(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenTagGetResults", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoTagGetResults(argU(a, 0)))
	})

	// =====================================================================
	// Table object getters (non-expression - takes TableRef)
	// =====================================================================
	ctx.SetFunc("_BinaryenTableGetName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cStr := cgoTableObjGetName(argU(a, 0))
		if cStr == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cStr)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})
	ctx.SetFunc("_BinaryenTableGetInitial", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI(this.Context(), cgoTableObjGetInitial(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenTableGetMax", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retI(this.Context(), cgoTableObjGetMax(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenTableGetType", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoTableObjGetType(argU(a, 0)))
	})
}

// ensure unsafe import is used
var _ = unsafe.Pointer(nil)
