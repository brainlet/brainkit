package asembed

import (
	"unsafe"

	quickjs "github.com/buke/quickjs-go"
)

// registerAllSetterImpls registers all expression setter (and related getter)
// bridge functions with real CGo implementations.
func registerAllSetterImpls(ctx *quickjs.Context, lm *LinearMemory) {
	// =====================================================================
	// Block
	// =====================================================================
	setFunc(ctx, "_BinaryenBlockSetName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		cgoBlockSetName(expr, cName)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenBlockSetChildAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoBlockSetChildAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(c)
	})

	// =====================================================================
	// If
	// =====================================================================
	setFunc(ctx, "_BinaryenIfSetCondition", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoIfSetCondition(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenIfSetIfTrue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoIfSetIfTrue(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenIfSetIfFalse", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoIfSetIfFalse(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// Loop
	// =====================================================================
	setFunc(ctx, "_BinaryenLoopSetName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		cgoLoopSetName(expr, cName)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenLoopSetBody", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoLoopSetBody(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// Break
	// =====================================================================
	setFunc(ctx, "_BinaryenBreakSetName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		cgoBreakSetName(expr, cName)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenBreakSetCondition", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoBreakSetCondition(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenBreakSetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoBreakSetValue(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// Switch
	// =====================================================================
	setFunc(ctx, "_BinaryenSwitchSetNameAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		idx := argU32(a, 1)
		cName := readCStr(lm, argI(a, 2))
		if cName != nil {
			defer cgoFree(cName)
		}
		cgoSwitchSetNameAt(expr, idx, cName)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenSwitchSetDefaultName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		cgoSwitchSetDefaultName(expr, cName)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenSwitchSetCondition", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSwitchSetCondition(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenSwitchSetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSwitchSetValue(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// Call
	// =====================================================================
	setFunc(ctx, "_BinaryenCallSetTarget", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		cTarget := readCStr(lm, argI(a, 1))
		if cTarget != nil {
			defer cgoFree(cTarget)
		}
		cgoCallSetTarget(expr, cTarget)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenCallSetOperandAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoCallSetOperandAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenCallSetReturn", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoCallSetReturn(argU(a, 0), argBool(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// CallIndirect
	// =====================================================================
	setFunc(ctx, "_BinaryenCallIndirectSetTarget", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoCallIndirectSetTarget(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenCallIndirectSetTable", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		cTable := readCStr(lm, argI(a, 1))
		if cTable != nil {
			defer cgoFree(cTable)
		}
		cgoCallIndirectSetTable(expr, cTable)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenCallIndirectSetOperandAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoCallIndirectSetOperandAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenCallIndirectSetReturn", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoCallIndirectSetReturn(argU(a, 0), argBool(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// LocalGet
	// =====================================================================
	setFunc(ctx, "_BinaryenLocalGetSetIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoLocalGetSetIndex(argU(a, 0), argU32(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// LocalSet (expression type named "LocalSet")
	// =====================================================================
	// Getters on the LocalSet expression type
	setFunc(ctx, "_BinaryenLocalSetGetIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retU32(c, cgoLocalSetGetIndex(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenLocalSetGetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoLocalSetGetValue(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenLocalSetIsTee", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retBool(c, cgoLocalSetIsTee(argU(a, 0)))
	})
	// Setters on the LocalSet expression type
	setFunc(ctx, "_BinaryenLocalSetSetIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoLocalSetSetIndex(argU(a, 0), argU32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenLocalSetSetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoLocalSetSetValue(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// GlobalGet (expression type)
	// =====================================================================
	setFunc(ctx, "_BinaryenGlobalGetSetName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		cgoGlobalGetSetName(expr, cName)
		return retVoid(c)
	})

	// =====================================================================
	// GlobalSet (expression type named "GlobalSet")
	// =====================================================================
	setFunc(ctx, "_BinaryenGlobalSetGetName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retStr(c, lm, cgoGlobalSetGetName(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenGlobalSetGetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoGlobalSetGetValue(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenGlobalSetSetName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		cgoGlobalSetSetName(expr, cName)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenGlobalSetSetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoGlobalSetSetValue(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// TableGet (expression type)
	// =====================================================================
	setFunc(ctx, "_BinaryenTableGetSetTable", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		cTable := readCStr(lm, argI(a, 1))
		if cTable != nil {
			defer cgoFree(cTable)
		}
		cgoTableGetSetTable(expr, cTable)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenTableGetSetIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoTableGetSetIndex(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// TableSet (expression type named "TableSet")
	// =====================================================================
	setFunc(ctx, "_BinaryenTableSetGetTable", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retStr(c, lm, cgoTableSetGetTable(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenTableSetGetIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoTableSetGetIndex(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenTableSetGetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoTableSetGetValue(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenTableSetSetTable", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		cTable := readCStr(lm, argI(a, 1))
		if cTable != nil {
			defer cgoFree(cTable)
		}
		cgoTableSetSetTable(expr, cTable)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenTableSetSetIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoTableSetSetIndex(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenTableSetSetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoTableSetSetValue(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// TableSize (expression type)
	// =====================================================================
	setFunc(ctx, "_BinaryenTableSizeSetTable", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		cTable := readCStr(lm, argI(a, 1))
		if cTable != nil {
			defer cgoFree(cTable)
		}
		cgoTableSizeSetTable(expr, cTable)
		return retVoid(c)
	})

	// =====================================================================
	// TableGrow (expression type)
	// =====================================================================
	setFunc(ctx, "_BinaryenTableGrowSetTable", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		cTable := readCStr(lm, argI(a, 1))
		if cTable != nil {
			defer cgoFree(cTable)
		}
		cgoTableGrowSetTable(expr, cTable)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenTableGrowSetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoTableGrowSetValue(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenTableGrowSetDelta", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoTableGrowSetDelta(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// Table (module-level, not expression)
	// =====================================================================
	setFunc(ctx, "_BinaryenTableSetName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		table := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		cgoTableSetName(table, cName)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenTableSetInitial", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoTableSetInitial(argU(a, 0), argU32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenTableSetMax", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoTableSetMax(argU(a, 0), argU32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenTableSetType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoTableSetType(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// MemoryGrow
	// =====================================================================
	setFunc(ctx, "_BinaryenMemoryGrowSetDelta", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoMemoryGrowSetDelta(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// Load
	// =====================================================================
	setFunc(ctx, "_BinaryenLoadSetAtomic", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoLoadSetAtomic(argU(a, 0), argBool(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenLoadSetSigned", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoLoadSetSigned(argU(a, 0), argBool(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenLoadSetOffset", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoLoadSetOffset(argU(a, 0), argU32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenLoadSetBytes", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoLoadSetBytes(argU(a, 0), argU32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenLoadSetAlign", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoLoadSetAlign(argU(a, 0), argU32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenLoadSetPtr", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoLoadSetPtr(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// Store
	// =====================================================================
	setFunc(ctx, "_BinaryenStoreSetAtomic", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStoreSetAtomic(argU(a, 0), argBool(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenStoreSetBytes", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStoreSetBytes(argU(a, 0), argU32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenStoreSetOffset", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStoreSetOffset(argU(a, 0), argU32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenStoreSetAlign", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStoreSetAlign(argU(a, 0), argU32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenStoreSetPtr", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStoreSetPtr(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenStoreSetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStoreSetValue(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenStoreSetValueType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStoreSetValueType(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// Const
	// =====================================================================
	setFunc(ctx, "_BinaryenConstSetValueI32", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoConstSetValueI32(argU(a, 0), argI32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenConstSetValueI64Low", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoConstSetValueI64Low(argU(a, 0), argI32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenConstSetValueI64High", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoConstSetValueI64High(argU(a, 0), argI32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenConstSetValueF32", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoConstSetValueF32(argU(a, 0), argF32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenConstSetValueF64", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoConstSetValueF64(argU(a, 0), argF64(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenConstSetValueV128", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		ptr := argI(a, 1)
		var buf [16]byte
		copy(buf[:], lm.ReadBytes(ptr, 16))
		cgoConstSetValueV128(expr, buf)
		return retVoid(c)
	})

	// =====================================================================
	// Unary
	// =====================================================================
	setFunc(ctx, "_BinaryenUnarySetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoUnarySetOp(argU(a, 0), argI32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenUnarySetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoUnarySetValue(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// Binary
	// =====================================================================
	setFunc(ctx, "_BinaryenBinarySetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoBinarySetOp(argU(a, 0), argI32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenBinarySetLeft", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoBinarySetLeft(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenBinarySetRight", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoBinarySetRight(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// Select
	// =====================================================================
	setFunc(ctx, "_BinaryenSelectSetIfTrue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSelectSetIfTrue(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenSelectSetIfFalse", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSelectSetIfFalse(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenSelectSetCondition", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSelectSetCondition(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// Drop
	// =====================================================================
	setFunc(ctx, "_BinaryenDropSetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoDropSetValue(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// Return
	// =====================================================================
	setFunc(ctx, "_BinaryenReturnSetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoReturnSetValue(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// AtomicRMW
	// =====================================================================
	setFunc(ctx, "_BinaryenAtomicRMWSetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoAtomicRMWSetOp(argU(a, 0), argI32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenAtomicRMWSetBytes", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoAtomicRMWSetBytes(argU(a, 0), argU32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenAtomicRMWSetOffset", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoAtomicRMWSetOffset(argU(a, 0), argU32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenAtomicRMWSetPtr", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoAtomicRMWSetPtr(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenAtomicRMWSetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoAtomicRMWSetValue(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// AtomicCmpxchg
	// =====================================================================
	setFunc(ctx, "_BinaryenAtomicCmpxchgSetBytes", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoAtomicCmpxchgSetBytes(argU(a, 0), argU32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenAtomicCmpxchgSetOffset", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoAtomicCmpxchgSetOffset(argU(a, 0), argU32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenAtomicCmpxchgSetPtr", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoAtomicCmpxchgSetPtr(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenAtomicCmpxchgSetExpected", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoAtomicCmpxchgSetExpected(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenAtomicCmpxchgSetReplacement", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoAtomicCmpxchgSetReplacement(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// AtomicWait
	// =====================================================================
	setFunc(ctx, "_BinaryenAtomicWaitSetPtr", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoAtomicWaitSetPtr(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenAtomicWaitSetExpected", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoAtomicWaitSetExpected(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenAtomicWaitSetTimeout", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoAtomicWaitSetTimeout(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenAtomicWaitSetExpectedType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoAtomicWaitSetExpectedType(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// AtomicNotify
	// =====================================================================
	setFunc(ctx, "_BinaryenAtomicNotifySetPtr", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoAtomicNotifySetPtr(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenAtomicNotifySetNotifyCount", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoAtomicNotifySetNotifyCount(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// AtomicFence
	// =====================================================================
	setFunc(ctx, "_BinaryenAtomicFenceSetOrder", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoAtomicFenceSetOrder(argU(a, 0), uint8(argU32(a, 1)))
		return retVoid(c)
	})

	// =====================================================================
	// SIMDExtract
	// =====================================================================
	setFunc(ctx, "_BinaryenSIMDExtractSetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDExtractSetOp(argU(a, 0), argI32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenSIMDExtractSetVec", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDExtractSetVec(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenSIMDExtractSetIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDExtractSetIndex(argU(a, 0), uint8(argU32(a, 1)))
		return retVoid(c)
	})

	// =====================================================================
	// SIMDReplace
	// =====================================================================
	setFunc(ctx, "_BinaryenSIMDReplaceSetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDReplaceSetOp(argU(a, 0), argI32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenSIMDReplaceSetVec", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDReplaceSetVec(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenSIMDReplaceSetIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDReplaceSetIndex(argU(a, 0), uint8(argU32(a, 1)))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenSIMDReplaceSetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDReplaceSetValue(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// SIMDShuffle
	// =====================================================================
	setFunc(ctx, "_BinaryenSIMDShuffleSetLeft", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDShuffleSetLeft(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenSIMDShuffleSetRight", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDShuffleSetRight(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenSIMDShuffleSetMask", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		ptr := argI(a, 1)
		var buf [16]byte
		copy(buf[:], lm.ReadBytes(ptr, 16))
		cgoSIMDShuffleSetMask(expr, buf)
		return retVoid(c)
	})

	// =====================================================================
	// SIMDTernary
	// =====================================================================
	setFunc(ctx, "_BinaryenSIMDTernarySetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDTernarySetOp(argU(a, 0), argI32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenSIMDTernarySetA", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDTernarySetA(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenSIMDTernarySetB", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDTernarySetB(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenSIMDTernarySetC", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDTernarySetC(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// SIMDShift
	// =====================================================================
	setFunc(ctx, "_BinaryenSIMDShiftSetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDShiftSetOp(argU(a, 0), argI32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenSIMDShiftSetVec", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDShiftSetVec(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenSIMDShiftSetShift", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDShiftSetShift(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// SIMDLoad
	// =====================================================================
	setFunc(ctx, "_BinaryenSIMDLoadSetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDLoadSetOp(argU(a, 0), argI32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenSIMDLoadSetOffset", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDLoadSetOffset(argU(a, 0), argU32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenSIMDLoadSetAlign", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDLoadSetAlign(argU(a, 0), argU32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenSIMDLoadSetPtr", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDLoadSetPtr(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// SIMDLoadStoreLane
	// =====================================================================
	setFunc(ctx, "_BinaryenSIMDLoadStoreLaneSetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDLoadStoreLaneSetOp(argU(a, 0), argI32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenSIMDLoadStoreLaneSetOffset", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDLoadStoreLaneSetOffset(argU(a, 0), argU32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenSIMDLoadStoreLaneSetAlign", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDLoadStoreLaneSetAlign(argU(a, 0), argU32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenSIMDLoadStoreLaneSetIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDLoadStoreLaneSetIndex(argU(a, 0), uint8(argU32(a, 1)))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenSIMDLoadStoreLaneSetPtr", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDLoadStoreLaneSetPtr(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenSIMDLoadStoreLaneSetVec", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSIMDLoadStoreLaneSetVec(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// MemoryInit
	// =====================================================================
	setFunc(ctx, "_BinaryenMemoryInitSetSegment", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		cSeg := readCStr(lm, argI(a, 1))
		if cSeg != nil {
			defer cgoFree(cSeg)
		}
		cgoMemoryInitSetSegment(expr, cSeg)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenMemoryInitSetDest", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoMemoryInitSetDest(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenMemoryInitSetOffset", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoMemoryInitSetOffset(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenMemoryInitSetSize", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoMemoryInitSetSize(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// DataDrop
	// =====================================================================
	setFunc(ctx, "_BinaryenDataDropSetSegment", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		cSeg := readCStr(lm, argI(a, 1))
		if cSeg != nil {
			defer cgoFree(cSeg)
		}
		cgoDataDropSetSegment(expr, cSeg)
		return retVoid(c)
	})

	// =====================================================================
	// MemoryCopy
	// =====================================================================
	setFunc(ctx, "_BinaryenMemoryCopySetDest", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoMemoryCopySetDest(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenMemoryCopySetSource", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoMemoryCopySetSource(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenMemoryCopySetSize", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoMemoryCopySetSize(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// MemoryFill
	// =====================================================================
	setFunc(ctx, "_BinaryenMemoryFillSetDest", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoMemoryFillSetDest(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenMemoryFillSetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoMemoryFillSetValue(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenMemoryFillSetSize", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoMemoryFillSetSize(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// RefIsNull
	// =====================================================================
	setFunc(ctx, "_BinaryenRefIsNullSetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoRefIsNullSetValue(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// RefAs
	// =====================================================================
	setFunc(ctx, "_BinaryenRefAsSetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoRefAsSetOp(argU(a, 0), argI32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenRefAsSetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoRefAsSetValue(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// RefFunc
	// =====================================================================
	setFunc(ctx, "_BinaryenRefFuncSetFunc", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		cgoRefFuncSetFunc(expr, cName)
		return retVoid(c)
	})

	// =====================================================================
	// RefI31
	// =====================================================================
	setFunc(ctx, "_BinaryenRefI31SetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoRefI31SetValue(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// RefEq
	// =====================================================================
	setFunc(ctx, "_BinaryenRefEqSetLeft", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoRefEqSetLeft(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenRefEqSetRight", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoRefEqSetRight(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// RefTest
	// =====================================================================
	setFunc(ctx, "_BinaryenRefTestSetRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoRefTestSetRef(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenRefTestSetCastType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoRefTestSetCastType(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// RefCast
	// =====================================================================
	setFunc(ctx, "_BinaryenRefCastSetRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoRefCastSetRef(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// BrOn
	// =====================================================================
	setFunc(ctx, "_BinaryenBrOnSetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoBrOnSetOp(argU(a, 0), argI32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenBrOnSetName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		cgoBrOnSetName(expr, cName)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenBrOnSetRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoBrOnSetRef(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenBrOnSetCastType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoBrOnSetCastType(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// I31Get
	// =====================================================================
	setFunc(ctx, "_BinaryenI31GetSetI31", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoI31GetSetI31(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenI31GetSetSigned", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoI31GetSetSigned(argU(a, 0), argBool(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// Try
	// =====================================================================
	setFunc(ctx, "_BinaryenTrySetName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		cName := readCStr(lm, argI(a, 1))
		if cName != nil {
			defer cgoFree(cName)
		}
		cgoTrySetName(expr, cName)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenTrySetBody", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoTrySetBody(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenTrySetCatchTagAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		idx := argU32(a, 1)
		cTag := readCStr(lm, argI(a, 2))
		if cTag != nil {
			defer cgoFree(cTag)
		}
		cgoTrySetCatchTagAt(expr, idx, cTag)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenTrySetCatchBodyAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoTrySetCatchBodyAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenTrySetDelegateTarget", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		cTarget := readCStr(lm, argI(a, 1))
		if cTarget != nil {
			defer cgoFree(cTarget)
		}
		cgoTrySetDelegateTarget(expr, cTarget)
		return retVoid(c)
	})

	// =====================================================================
	// Throw
	// =====================================================================
	setFunc(ctx, "_BinaryenThrowSetTag", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		cTag := readCStr(lm, argI(a, 1))
		if cTag != nil {
			defer cgoFree(cTag)
		}
		cgoThrowSetTag(expr, cTag)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenThrowSetOperandAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoThrowSetOperandAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(c)
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
	setFunc(ctx, "_BinaryenRethrowSetDepth", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		// No direct C API equivalent (BinaryenRethrowSetDepth was removed).
		// BinaryenRethrowSetTarget takes a string. This remains a no-op stub.
		return retVoid(c)
	})

	// =====================================================================
	// TupleMake
	// =====================================================================
	setFunc(ctx, "_BinaryenTupleMakeSetOperandAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoTupleMakeSetOperandAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(c)
	})

	// =====================================================================
	// TupleExtract
	// =====================================================================
	setFunc(ctx, "_BinaryenTupleExtractSetTuple", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoTupleExtractSetTuple(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenTupleExtractSetIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoTupleExtractSetIndex(argU(a, 0), argU32(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// CallRef
	// =====================================================================
	setFunc(ctx, "_BinaryenCallRefSetOperandAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoCallRefSetOperandAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenCallRefSetTarget", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoCallRefSetTarget(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenCallRefSetReturn", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoCallRefSetReturn(argU(a, 0), argBool(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// StructNew
	// =====================================================================
	setFunc(ctx, "_BinaryenStructNewSetOperandAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStructNewSetOperandAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(c)
	})

	// =====================================================================
	// StructGet
	// =====================================================================
	setFunc(ctx, "_BinaryenStructGetSetIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStructGetSetIndex(argU(a, 0), argU32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenStructGetSetRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStructGetSetRef(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenStructGetSetSigned", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStructGetSetSigned(argU(a, 0), argBool(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// StructSet (expression type named "StructSet")
	// =====================================================================
	setFunc(ctx, "_BinaryenStructSetGetIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retU32(c, cgoStructSetGetIndex(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenStructSetGetRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStructSetGetRef(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenStructSetGetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoStructSetGetValue(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenStructSetSetIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStructSetSetIndex(argU(a, 0), argU32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenStructSetSetRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStructSetSetRef(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenStructSetSetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStructSetSetValue(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// ArrayNew
	// =====================================================================
	setFunc(ctx, "_BinaryenArrayNewSetInit", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoArrayNewSetInit(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenArrayNewSetSize", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoArrayNewSetSize(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// ArrayNewFixed
	// =====================================================================
	setFunc(ctx, "_BinaryenArrayNewFixedSetValueAt", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoArrayNewFixedSetValueAt(argU(a, 0), argU32(a, 1), argU(a, 2))
		return retVoid(c)
	})

	// =====================================================================
	// ArrayGet
	// =====================================================================
	setFunc(ctx, "_BinaryenArrayGetSetRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoArrayGetSetRef(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenArrayGetSetIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoArrayGetSetIndex(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenArrayGetSetSigned", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoArrayGetSetSigned(argU(a, 0), argBool(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// ArraySet (expression type named "ArraySet")
	// =====================================================================
	setFunc(ctx, "_BinaryenArraySetGetIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoArraySetGetIndex(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenArraySetGetRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoArraySetGetRef(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenArraySetGetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoArraySetGetValue(argU(a, 0)))
	})
	setFunc(ctx, "_BinaryenArraySetSetIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoArraySetSetIndex(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenArraySetSetRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoArraySetSetRef(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenArraySetSetValue", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoArraySetSetValue(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// ArrayLen
	// =====================================================================
	setFunc(ctx, "_BinaryenArrayLenSetRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoArrayLenSetRef(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// ArrayCopy
	// =====================================================================
	setFunc(ctx, "_BinaryenArrayCopySetDestRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoArrayCopySetDestRef(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenArrayCopySetDestIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoArrayCopySetDestIndex(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenArrayCopySetSrcRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoArrayCopySetSrcRef(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenArrayCopySetSrcIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoArrayCopySetSrcIndex(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenArrayCopySetLength", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoArrayCopySetLength(argU(a, 0), argU(a, 1))
		return retVoid(c)
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
	setFunc(ctx, "_BinaryenStringNewSetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStringNewSetOp(argU(a, 0), argI32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenStringNewSetRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStringNewSetRef(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenStringNewSetStart", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStringNewSetStart(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenStringNewSetEnd", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStringNewSetEnd(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	setFunc(ctx, "_BinaryenStringConstSetString", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		expr := argU(a, 0)
		cStr := readCStr(lm, argI(a, 1))
		if cStr != nil {
			defer cgoFree(cStr)
		}
		cgoStringConstSetString(expr, cStr)
		return retVoid(c)
	})

	setFunc(ctx, "_BinaryenStringMeasureSetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStringMeasureSetOp(argU(a, 0), argI32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenStringMeasureSetRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStringMeasureSetRef(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	setFunc(ctx, "_BinaryenStringEncodeSetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStringEncodeSetOp(argU(a, 0), argI32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenStringEncodeSetStr", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStringEncodeSetStr(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenStringEncodeSetArray", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStringEncodeSetArray(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenStringEncodeSetStart", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStringEncodeSetStart(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	setFunc(ctx, "_BinaryenStringConcatSetLeft", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStringConcatSetLeft(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenStringConcatSetRight", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStringConcatSetRight(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	setFunc(ctx, "_BinaryenStringEqSetOp", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStringEqSetOp(argU(a, 0), argI32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenStringEqSetLeft", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStringEqSetLeft(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenStringEqSetRight", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStringEqSetRight(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	setFunc(ctx, "_BinaryenStringWTF16GetSetRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStringWTF16GetSetRef(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenStringWTF16GetSetPos", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStringWTF16GetSetPos(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	setFunc(ctx, "_BinaryenStringSliceWTFSetRef", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStringSliceWTFSetRef(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenStringSliceWTFSetStart", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStringSliceWTFSetStart(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenStringSliceWTFSetEnd", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoStringSliceWTFSetEnd(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})

	// =====================================================================
	// Function (module-level, not expression)
	// =====================================================================
	setFunc(ctx, "_BinaryenFunctionSetType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoFunctionSetType(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenFunctionSetDebugLocation", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoFunctionSetDebugLocation(argU(a, 0), argU(a, 1), argU32(a, 2), argU32(a, 3), argU32(a, 4))
		return retVoid(c)
	})
}

// retStr writes a C string to linear memory and returns a pointer to it, or 0 if nil.
// This is used for getter functions that return const char*.
func retStr(ctx *quickjs.Context, lm *LinearMemory, s unsafe.Pointer) *quickjs.Value {
	if s == nil {
		return ctx.NewFloat64(0)
	}
	// The returned const char* is owned by binaryen, so we just need to read it
	// and write it to linear memory. For now, return 0 as writing to linear
	// memory requires allocation which the JS side handles.
	// These getters return const char* which the JS side reads via UTF8ToString.
	// We return the C pointer directly as a numeric value.
	return ctx.NewFloat64(float64(uintptr(s)))
}

var _ = unsafe.Pointer(nil)
