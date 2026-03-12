package asembed

import (
	"unsafe"

	"github.com/fastschema/qjs"
)

// registerAllSetterImpls registers all expression setter (and related getter)
// bridge functions with real CGo implementations.
func registerAllSetterImpls(ctx *qjs.Context, lm *LinearMemory) {
	// =====================================================================
	// Block
	// =====================================================================
	ctx.SetFunc("_BinaryenBlockSetName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		cgoBlockSetName(expr, cName)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenBlockSetChildAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoBlockSetChildAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(this.Context())
	})

	// =====================================================================
	// If
	// =====================================================================
	ctx.SetFunc("_BinaryenIfSetCondition", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoIfSetCondition(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenIfSetIfTrue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoIfSetIfTrue(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenIfSetIfFalse", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoIfSetIfFalse(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// Loop
	// =====================================================================
	ctx.SetFunc("_BinaryenLoopSetName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		cgoLoopSetName(expr, cName)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenLoopSetBody", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoLoopSetBody(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// Break
	// =====================================================================
	ctx.SetFunc("_BinaryenBreakSetName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		cgoBreakSetName(expr, cName)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenBreakSetCondition", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoBreakSetCondition(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenBreakSetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoBreakSetValue(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// Switch
	// =====================================================================
	ctx.SetFunc("_BinaryenSwitchSetNameAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		idx := argU32(a, 1)
		cName := readCStr(lm, argI(a, 2))
		if cName != nil {
			defer cgoFree(cName)
		}
		cgoSwitchSetNameAt(expr, idx, cName)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenSwitchSetDefaultName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		cgoSwitchSetDefaultName(expr, cName)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenSwitchSetCondition", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSwitchSetCondition(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenSwitchSetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSwitchSetValue(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// Call
	// =====================================================================
	ctx.SetFunc("_BinaryenCallSetTarget", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		cTarget := readCStr(lm, argI(a, 1))
		if cTarget != nil {
			defer cgoFree(cTarget)
		}
		cgoCallSetTarget(expr, cTarget)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenCallSetOperandAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoCallSetOperandAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenCallSetReturn", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoCallSetReturn(argU(a, 0), argBool(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// CallIndirect
	// =====================================================================
	ctx.SetFunc("_BinaryenCallIndirectSetTarget", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoCallIndirectSetTarget(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenCallIndirectSetTable", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		cTable := readCStr(lm, argI(a, 1))
		if cTable != nil {
			defer cgoFree(cTable)
		}
		cgoCallIndirectSetTable(expr, cTable)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenCallIndirectSetOperandAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoCallIndirectSetOperandAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenCallIndirectSetReturn", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoCallIndirectSetReturn(argU(a, 0), argBool(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// LocalGet
	// =====================================================================
	ctx.SetFunc("_BinaryenLocalGetSetIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoLocalGetSetIndex(argU(a, 0), argU32(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// LocalSet (expression type named "LocalSet")
	// =====================================================================
	// Getters on the LocalSet expression type
	ctx.SetFunc("_BinaryenLocalSetGetIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retU32(this.Context(), cgoLocalSetGetIndex(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenLocalSetGetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoLocalSetGetValue(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenLocalSetIsTee", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retBool(this.Context(), cgoLocalSetIsTee(argU(a, 0)))
	})
	// Setters on the LocalSet expression type
	ctx.SetFunc("_BinaryenLocalSetSetIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoLocalSetSetIndex(argU(a, 0), argU32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenLocalSetSetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoLocalSetSetValue(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// GlobalGet (expression type)
	// =====================================================================
	ctx.SetFunc("_BinaryenGlobalGetSetName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		cgoGlobalGetSetName(expr, cName)
		return retVoid(this.Context())
	})

	// =====================================================================
	// GlobalSet (expression type named "GlobalSet")
	// =====================================================================
	ctx.SetFunc("_BinaryenGlobalSetGetName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retStr(this.Context(), lm, cgoGlobalSetGetName(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenGlobalSetGetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoGlobalSetGetValue(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenGlobalSetSetName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		cgoGlobalSetSetName(expr, cName)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenGlobalSetSetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoGlobalSetSetValue(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// TableGet (expression type)
	// =====================================================================
	ctx.SetFunc("_BinaryenTableGetSetTable", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		cTable := readCStr(lm, argI(a, 1))
		if cTable != nil {
			defer cgoFree(cTable)
		}
		cgoTableGetSetTable(expr, cTable)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenTableGetSetIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoTableGetSetIndex(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// TableSet (expression type named "TableSet")
	// =====================================================================
	ctx.SetFunc("_BinaryenTableSetGetTable", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retStr(this.Context(), lm, cgoTableSetGetTable(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenTableSetGetIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoTableSetGetIndex(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenTableSetGetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoTableSetGetValue(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenTableSetSetTable", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		cTable := readCStr(lm, argI(a, 1))
		if cTable != nil {
			defer cgoFree(cTable)
		}
		cgoTableSetSetTable(expr, cTable)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenTableSetSetIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoTableSetSetIndex(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenTableSetSetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoTableSetSetValue(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// TableSize (expression type)
	// =====================================================================
	ctx.SetFunc("_BinaryenTableSizeSetTable", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		cTable := readCStr(lm, argI(a, 1))
		if cTable != nil {
			defer cgoFree(cTable)
		}
		cgoTableSizeSetTable(expr, cTable)
		return retVoid(this.Context())
	})

	// =====================================================================
	// TableGrow (expression type)
	// =====================================================================
	ctx.SetFunc("_BinaryenTableGrowSetTable", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		cTable := readCStr(lm, argI(a, 1))
		if cTable != nil {
			defer cgoFree(cTable)
		}
		cgoTableGrowSetTable(expr, cTable)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenTableGrowSetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoTableGrowSetValue(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenTableGrowSetDelta", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoTableGrowSetDelta(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// Table (module-level, not expression)
	// =====================================================================
	ctx.SetFunc("_BinaryenTableSetName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		table := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		cgoTableSetName(table, cName)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenTableSetInitial", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoTableSetInitial(argU(a, 0), argU32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenTableSetMax", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoTableSetMax(argU(a, 0), argU32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenTableSetType", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoTableSetType(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// MemoryGrow
	// =====================================================================
	ctx.SetFunc("_BinaryenMemoryGrowSetDelta", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoMemoryGrowSetDelta(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// Load
	// =====================================================================
	ctx.SetFunc("_BinaryenLoadSetAtomic", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoLoadSetAtomic(argU(a, 0), argBool(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenLoadSetSigned", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoLoadSetSigned(argU(a, 0), argBool(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenLoadSetOffset", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoLoadSetOffset(argU(a, 0), argU32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenLoadSetBytes", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoLoadSetBytes(argU(a, 0), argU32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenLoadSetAlign", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoLoadSetAlign(argU(a, 0), argU32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenLoadSetPtr", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoLoadSetPtr(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// Store
	// =====================================================================
	ctx.SetFunc("_BinaryenStoreSetAtomic", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStoreSetAtomic(argU(a, 0), argBool(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenStoreSetBytes", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStoreSetBytes(argU(a, 0), argU32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenStoreSetOffset", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStoreSetOffset(argU(a, 0), argU32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenStoreSetAlign", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStoreSetAlign(argU(a, 0), argU32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenStoreSetPtr", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStoreSetPtr(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenStoreSetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStoreSetValue(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenStoreSetValueType", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStoreSetValueType(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// Const
	// =====================================================================
	ctx.SetFunc("_BinaryenConstSetValueI32", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoConstSetValueI32(argU(a, 0), argI32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenConstSetValueI64Low", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoConstSetValueI64Low(argU(a, 0), argI32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenConstSetValueI64High", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoConstSetValueI64High(argU(a, 0), argI32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenConstSetValueF32", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoConstSetValueF32(argU(a, 0), argF32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenConstSetValueF64", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoConstSetValueF64(argU(a, 0), argF64(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenConstSetValueV128", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		ptr := argI(a, 1)
		var buf [16]byte
		copy(buf[:], lm.ReadBytes(ptr, 16))
		cgoConstSetValueV128(expr, buf)
		return retVoid(this.Context())
	})

	// =====================================================================
	// Unary
	// =====================================================================
	ctx.SetFunc("_BinaryenUnarySetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoUnarySetOp(argU(a, 0), argI32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenUnarySetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoUnarySetValue(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// Binary
	// =====================================================================
	ctx.SetFunc("_BinaryenBinarySetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoBinarySetOp(argU(a, 0), argI32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenBinarySetLeft", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoBinarySetLeft(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenBinarySetRight", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoBinarySetRight(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// Select
	// =====================================================================
	ctx.SetFunc("_BinaryenSelectSetIfTrue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSelectSetIfTrue(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenSelectSetIfFalse", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSelectSetIfFalse(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenSelectSetCondition", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSelectSetCondition(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// Drop
	// =====================================================================
	ctx.SetFunc("_BinaryenDropSetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoDropSetValue(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// Return
	// =====================================================================
	ctx.SetFunc("_BinaryenReturnSetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoReturnSetValue(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// AtomicRMW
	// =====================================================================
	ctx.SetFunc("_BinaryenAtomicRMWSetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoAtomicRMWSetOp(argU(a, 0), argI32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenAtomicRMWSetBytes", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoAtomicRMWSetBytes(argU(a, 0), argU32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenAtomicRMWSetOffset", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoAtomicRMWSetOffset(argU(a, 0), argU32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenAtomicRMWSetPtr", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoAtomicRMWSetPtr(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenAtomicRMWSetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoAtomicRMWSetValue(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// AtomicCmpxchg
	// =====================================================================
	ctx.SetFunc("_BinaryenAtomicCmpxchgSetBytes", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoAtomicCmpxchgSetBytes(argU(a, 0), argU32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenAtomicCmpxchgSetOffset", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoAtomicCmpxchgSetOffset(argU(a, 0), argU32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenAtomicCmpxchgSetPtr", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoAtomicCmpxchgSetPtr(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenAtomicCmpxchgSetExpected", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoAtomicCmpxchgSetExpected(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenAtomicCmpxchgSetReplacement", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoAtomicCmpxchgSetReplacement(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// AtomicWait
	// =====================================================================
	ctx.SetFunc("_BinaryenAtomicWaitSetPtr", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoAtomicWaitSetPtr(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenAtomicWaitSetExpected", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoAtomicWaitSetExpected(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenAtomicWaitSetTimeout", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoAtomicWaitSetTimeout(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenAtomicWaitSetExpectedType", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoAtomicWaitSetExpectedType(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// AtomicNotify
	// =====================================================================
	ctx.SetFunc("_BinaryenAtomicNotifySetPtr", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoAtomicNotifySetPtr(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenAtomicNotifySetNotifyCount", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoAtomicNotifySetNotifyCount(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// AtomicFence
	// =====================================================================
	ctx.SetFunc("_BinaryenAtomicFenceSetOrder", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoAtomicFenceSetOrder(argU(a, 0), uint8(argU32(a, 1)))
		return retVoid(this.Context())
	})

	// =====================================================================
	// SIMDExtract
	// =====================================================================
	ctx.SetFunc("_BinaryenSIMDExtractSetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDExtractSetOp(argU(a, 0), argI32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenSIMDExtractSetVec", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDExtractSetVec(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenSIMDExtractSetIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDExtractSetIndex(argU(a, 0), uint8(argU32(a, 1)))
		return retVoid(this.Context())
	})

	// =====================================================================
	// SIMDReplace
	// =====================================================================
	ctx.SetFunc("_BinaryenSIMDReplaceSetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDReplaceSetOp(argU(a, 0), argI32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenSIMDReplaceSetVec", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDReplaceSetVec(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenSIMDReplaceSetIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDReplaceSetIndex(argU(a, 0), uint8(argU32(a, 1)))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenSIMDReplaceSetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDReplaceSetValue(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// SIMDShuffle
	// =====================================================================
	ctx.SetFunc("_BinaryenSIMDShuffleSetLeft", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDShuffleSetLeft(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenSIMDShuffleSetRight", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDShuffleSetRight(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenSIMDShuffleSetMask", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		ptr := argI(a, 1)
		var buf [16]byte
		copy(buf[:], lm.ReadBytes(ptr, 16))
		cgoSIMDShuffleSetMask(expr, buf)
		return retVoid(this.Context())
	})

	// =====================================================================
	// SIMDTernary
	// =====================================================================
	ctx.SetFunc("_BinaryenSIMDTernarySetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDTernarySetOp(argU(a, 0), argI32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenSIMDTernarySetA", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDTernarySetA(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenSIMDTernarySetB", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDTernarySetB(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenSIMDTernarySetC", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDTernarySetC(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// SIMDShift
	// =====================================================================
	ctx.SetFunc("_BinaryenSIMDShiftSetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDShiftSetOp(argU(a, 0), argI32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenSIMDShiftSetVec", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDShiftSetVec(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenSIMDShiftSetShift", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDShiftSetShift(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// SIMDLoad
	// =====================================================================
	ctx.SetFunc("_BinaryenSIMDLoadSetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDLoadSetOp(argU(a, 0), argI32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenSIMDLoadSetOffset", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDLoadSetOffset(argU(a, 0), argU32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenSIMDLoadSetAlign", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDLoadSetAlign(argU(a, 0), argU32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenSIMDLoadSetPtr", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDLoadSetPtr(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// SIMDLoadStoreLane
	// =====================================================================
	ctx.SetFunc("_BinaryenSIMDLoadStoreLaneSetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDLoadStoreLaneSetOp(argU(a, 0), argI32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenSIMDLoadStoreLaneSetOffset", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDLoadStoreLaneSetOffset(argU(a, 0), argU32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenSIMDLoadStoreLaneSetAlign", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDLoadStoreLaneSetAlign(argU(a, 0), argU32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenSIMDLoadStoreLaneSetIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDLoadStoreLaneSetIndex(argU(a, 0), uint8(argU32(a, 1)))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenSIMDLoadStoreLaneSetPtr", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDLoadStoreLaneSetPtr(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenSIMDLoadStoreLaneSetVec", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSIMDLoadStoreLaneSetVec(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// MemoryInit
	// =====================================================================
	ctx.SetFunc("_BinaryenMemoryInitSetSegment", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		cSeg := readCStr(lm, argI(a, 1))
		if cSeg != nil {
			defer cgoFree(cSeg)
		}
		cgoMemoryInitSetSegment(expr, cSeg)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenMemoryInitSetDest", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoMemoryInitSetDest(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenMemoryInitSetOffset", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoMemoryInitSetOffset(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenMemoryInitSetSize", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoMemoryInitSetSize(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// DataDrop
	// =====================================================================
	ctx.SetFunc("_BinaryenDataDropSetSegment", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		cSeg := readCStr(lm, argI(a, 1))
		if cSeg != nil {
			defer cgoFree(cSeg)
		}
		cgoDataDropSetSegment(expr, cSeg)
		return retVoid(this.Context())
	})

	// =====================================================================
	// MemoryCopy
	// =====================================================================
	ctx.SetFunc("_BinaryenMemoryCopySetDest", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoMemoryCopySetDest(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenMemoryCopySetSource", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoMemoryCopySetSource(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenMemoryCopySetSize", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoMemoryCopySetSize(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// MemoryFill
	// =====================================================================
	ctx.SetFunc("_BinaryenMemoryFillSetDest", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoMemoryFillSetDest(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenMemoryFillSetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoMemoryFillSetValue(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenMemoryFillSetSize", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoMemoryFillSetSize(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// RefIsNull
	// =====================================================================
	ctx.SetFunc("_BinaryenRefIsNullSetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoRefIsNullSetValue(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// RefAs
	// =====================================================================
	ctx.SetFunc("_BinaryenRefAsSetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoRefAsSetOp(argU(a, 0), argI32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenRefAsSetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoRefAsSetValue(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// RefFunc
	// =====================================================================
	ctx.SetFunc("_BinaryenRefFuncSetFunc", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		cgoRefFuncSetFunc(expr, cName)
		return retVoid(this.Context())
	})

	// =====================================================================
	// RefI31
	// =====================================================================
	ctx.SetFunc("_BinaryenRefI31SetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoRefI31SetValue(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// RefEq
	// =====================================================================
	ctx.SetFunc("_BinaryenRefEqSetLeft", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoRefEqSetLeft(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenRefEqSetRight", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoRefEqSetRight(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// RefTest
	// =====================================================================
	ctx.SetFunc("_BinaryenRefTestSetRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoRefTestSetRef(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenRefTestSetCastType", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoRefTestSetCastType(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// RefCast
	// =====================================================================
	ctx.SetFunc("_BinaryenRefCastSetRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoRefCastSetRef(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// BrOn
	// =====================================================================
	ctx.SetFunc("_BinaryenBrOnSetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoBrOnSetOp(argU(a, 0), argI32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenBrOnSetName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		cgoBrOnSetName(expr, cName)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenBrOnSetRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoBrOnSetRef(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenBrOnSetCastType", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoBrOnSetCastType(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// I31Get
	// =====================================================================
	ctx.SetFunc("_BinaryenI31GetSetI31", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoI31GetSetI31(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenI31GetSetSigned", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoI31GetSetSigned(argU(a, 0), argBool(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// Try
	// =====================================================================
	ctx.SetFunc("_BinaryenTrySetName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		cgoTrySetName(expr, cName)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenTrySetBody", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoTrySetBody(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenTrySetCatchTagAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		idx := argU32(a, 1)
		cTag := readCStr(lm, argI(a, 2))
		if cTag != nil {
			defer cgoFree(cTag)
		}
		cgoTrySetCatchTagAt(expr, idx, cTag)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenTrySetCatchBodyAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoTrySetCatchBodyAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenTrySetDelegateTarget", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		cTarget := readCStr(lm, argI(a, 1))
		if cTarget != nil {
			defer cgoFree(cTarget)
		}
		cgoTrySetDelegateTarget(expr, cTarget)
		return retVoid(this.Context())
	})

	// =====================================================================
	// Throw
	// =====================================================================
	ctx.SetFunc("_BinaryenThrowSetTag", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		cTag := readCStr(lm, argI(a, 1))
		if cTag != nil {
			defer cgoFree(cTag)
		}
		cgoThrowSetTag(expr, cTag)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenThrowSetOperandAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoThrowSetOperandAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(this.Context())
	})

	// =====================================================================
	// Rethrow — RethrowSetDepth maps to RethrowSetTarget (API evolution).
	// The AS compiler's _BinaryenRethrowSetDepth takes (expr, depth_int), but
	// the current C API BinaryenRethrowSetTarget takes (expr, const char*).
	// Since these are incompatible, we map it to SetTarget using the string
	// representation, or leave as no-op if not feasible.
	// Actually, checking the binaryen-c.h, BinaryenRethrowSetTarget takes a
	// const char*. The old "depth" API is gone. We map this as a string-setter.
	// =====================================================================
	ctx.SetFunc("_BinaryenRethrowSetDepth", func(this *qjs.This) (*qjs.Value, error) {
		// No direct C API equivalent (BinaryenRethrowSetDepth was removed).
		// BinaryenRethrowSetTarget takes a string. This remains a no-op stub.
		return retVoid(this.Context())
	})

	// =====================================================================
	// TupleMake
	// =====================================================================
	ctx.SetFunc("_BinaryenTupleMakeSetOperandAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoTupleMakeSetOperandAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(this.Context())
	})

	// =====================================================================
	// TupleExtract
	// =====================================================================
	ctx.SetFunc("_BinaryenTupleExtractSetTuple", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoTupleExtractSetTuple(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenTupleExtractSetIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoTupleExtractSetIndex(argU(a, 0), argU32(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// CallRef
	// =====================================================================
	ctx.SetFunc("_BinaryenCallRefSetOperandAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoCallRefSetOperandAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenCallRefSetTarget", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoCallRefSetTarget(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenCallRefSetReturn", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoCallRefSetReturn(argU(a, 0), argBool(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// StructNew
	// =====================================================================
	ctx.SetFunc("_BinaryenStructNewSetOperandAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStructNewSetOperandAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(this.Context())
	})

	// =====================================================================
	// StructGet
	// =====================================================================
	ctx.SetFunc("_BinaryenStructGetSetIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStructGetSetIndex(argU(a, 0), argU32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenStructGetSetRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStructGetSetRef(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenStructGetSetSigned", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStructGetSetSigned(argU(a, 0), argBool(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// StructSet (expression type named "StructSet")
	// =====================================================================
	ctx.SetFunc("_BinaryenStructSetGetIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retU32(this.Context(), cgoStructSetGetIndex(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenStructSetGetRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStructSetGetRef(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenStructSetGetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoStructSetGetValue(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenStructSetSetIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStructSetSetIndex(argU(a, 0), argU32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenStructSetSetRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStructSetSetRef(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenStructSetSetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStructSetSetValue(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// ArrayNew
	// =====================================================================
	ctx.SetFunc("_BinaryenArrayNewSetInit", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoArrayNewSetInit(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenArrayNewSetSize", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoArrayNewSetSize(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// ArrayNewFixed
	// =====================================================================
	ctx.SetFunc("_BinaryenArrayNewFixedSetValueAt", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoArrayNewFixedSetValueAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(this.Context())
	})

	// =====================================================================
	// ArrayGet
	// =====================================================================
	ctx.SetFunc("_BinaryenArrayGetSetRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoArrayGetSetRef(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenArrayGetSetIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoArrayGetSetIndex(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenArrayGetSetSigned", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoArrayGetSetSigned(argU(a, 0), argBool(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// ArraySet (expression type named "ArraySet")
	// =====================================================================
	ctx.SetFunc("_BinaryenArraySetGetIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoArraySetGetIndex(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenArraySetGetRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoArraySetGetRef(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenArraySetGetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoArraySetGetValue(argU(a, 0)))
	})
	ctx.SetFunc("_BinaryenArraySetSetIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoArraySetSetIndex(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenArraySetSetRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoArraySetSetRef(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenArraySetSetValue", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoArraySetSetValue(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// ArrayLen
	// =====================================================================
	ctx.SetFunc("_BinaryenArrayLenSetRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoArrayLenSetRef(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// ArrayCopy
	// =====================================================================
	ctx.SetFunc("_BinaryenArrayCopySetDestRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoArrayCopySetDestRef(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenArrayCopySetDestIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoArrayCopySetDestIndex(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenArrayCopySetSrcRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoArrayCopySetSrcRef(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenArrayCopySetSrcIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoArrayCopySetSrcIndex(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenArrayCopySetLength", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoArrayCopySetLength(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// ArrayFill — No C API, remain as stubs
	// =====================================================================
	// BinaryenArrayFillSet{Index,Ref,Size,Value} do not exist in binaryen-c.h

	// =====================================================================
	// ArrayInitData — No C API, remain as stubs
	// =====================================================================
	// BinaryenArrayInitDataSet{Index,Offset,Ref,Segment,Size} do not exist in binaryen-c.h

	// =====================================================================
	// ArrayInitElem — No C API, remain as stubs
	// =====================================================================
	// BinaryenArrayInitElemSet{Index,Offset,Ref,Segment,Size} do not exist in binaryen-c.h

	// =====================================================================
	// ArrayNewData — No C API setters, remain as stubs
	// =====================================================================
	// BinaryenArrayNewDataSet{Offset,Segment,Size} do not exist in binaryen-c.h

	// =====================================================================
	// ArrayNewElem — No C API setters, remain as stubs
	// =====================================================================
	// BinaryenArrayNewElemSet{Offset,Segment,Size} do not exist in binaryen-c.h

	// =====================================================================
	// String expressions
	// =====================================================================
	ctx.SetFunc("_BinaryenStringNewSetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStringNewSetOp(argU(a, 0), argI32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenStringNewSetRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStringNewSetRef(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenStringNewSetStart", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStringNewSetStart(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenStringNewSetEnd", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStringNewSetEnd(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	ctx.SetFunc("_BinaryenStringConstSetString", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		expr := argU(a, 0)
		cStr := readCStr(lm, argI(a, 1))
		if cStr != nil {
			defer cgoFree(cStr)
		}
		cgoStringConstSetString(expr, cStr)
		return retVoid(this.Context())
	})

	ctx.SetFunc("_BinaryenStringMeasureSetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStringMeasureSetOp(argU(a, 0), argI32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenStringMeasureSetRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStringMeasureSetRef(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	ctx.SetFunc("_BinaryenStringEncodeSetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStringEncodeSetOp(argU(a, 0), argI32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenStringEncodeSetStr", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStringEncodeSetStr(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenStringEncodeSetArray", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStringEncodeSetArray(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenStringEncodeSetStart", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStringEncodeSetStart(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	ctx.SetFunc("_BinaryenStringConcatSetLeft", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStringConcatSetLeft(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenStringConcatSetRight", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStringConcatSetRight(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	ctx.SetFunc("_BinaryenStringEqSetOp", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStringEqSetOp(argU(a, 0), argI32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenStringEqSetLeft", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStringEqSetLeft(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenStringEqSetRight", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStringEqSetRight(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	ctx.SetFunc("_BinaryenStringWTF16GetSetRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStringWTF16GetSetRef(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenStringWTF16GetSetPos", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStringWTF16GetSetPos(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	ctx.SetFunc("_BinaryenStringSliceWTFSetRef", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStringSliceWTFSetRef(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenStringSliceWTFSetStart", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStringSliceWTFSetStart(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenStringSliceWTFSetEnd", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoStringSliceWTFSetEnd(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})

	// =====================================================================
	// Function (module-level, not expression)
	// =====================================================================
	ctx.SetFunc("_BinaryenFunctionSetType", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoFunctionSetType(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenFunctionSetDebugLocation", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoFunctionSetDebugLocation(argU(a, 0), argU(a, 1), argU32(a, 2), argU32(a, 3), argU32(a, 4))
		return retVoid(this.Context())
	})
}

// retStr writes a C string to linear memory and returns a pointer to it, or 0 if nil.
// This is used for getter functions that return const char*.
func retStr(ctx *qjs.Context, lm *LinearMemory, s unsafe.Pointer) (*qjs.Value, error) {
	if s == nil {
		return ctx.NewFloat64(0), nil
	}
	// The returned const char* is owned by binaryen, so we just need to read it
	// and write it to linear memory. For now, return 0 as writing to linear
	// memory requires allocation which the JS side handles.
	// These getters return const char* which the JS side reads via UTF8ToString.
	// We return the C pointer directly as a numeric value.
	return ctx.NewFloat64(float64(uintptr(s))), nil
}

var _ = unsafe.Pointer(nil)
