// Ported from: assemblyscript/src/passes/pass.ts (replaceChild function, lines 1321-2120)
//
// Utility function for replacing a child expression within a parent expression.
package passes

import (
	"fmt"

	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/pkg/binaryen"
)

// ReplaceChild replaces an expression within a parent expression.
// Returns the replaced expression on success, otherwise 0.
func ReplaceChild(parent, search, replacement module.ExpressionRef) module.ExpressionRef {
	switch binaryen.ExpressionGetId(parent) {
	case module.ExpressionIdBlock:
		numChildren := binaryen.BlockGetNumChildren(parent)
		for i := module.Index(0); i < numChildren; i++ {
			child := binaryen.BlockGetChildAt(parent, i)
			if child == search {
				binaryen.BlockSetChildAt(parent, i, replacement)
				return child
			}
		}

	case module.ExpressionIdIf:
		condition := binaryen.IfGetCondition(parent)
		if condition == search {
			binaryen.IfSetCondition(parent, replacement)
			return condition
		}
		ifTrue := binaryen.IfGetIfTrue(parent)
		if ifTrue == search {
			binaryen.IfSetIfTrue(parent, replacement)
			return ifTrue
		}
		ifFalse := binaryen.IfGetIfFalse(parent)
		if ifFalse == search {
			binaryen.IfSetIfFalse(parent, replacement)
			return ifFalse
		}

	case module.ExpressionIdLoop:
		body := binaryen.LoopGetBody(parent)
		if body == search {
			binaryen.LoopSetBody(parent, replacement)
			return body
		}

	case module.ExpressionIdBreak:
		condition := binaryen.BreakGetCondition(parent)
		if condition == search {
			binaryen.BreakSetCondition(parent, replacement)
			return condition
		}
		value := binaryen.BreakGetValue(parent)
		if value == search {
			binaryen.BreakSetValue(parent, replacement)
			return value
		}

	case module.ExpressionIdSwitch:
		condition := binaryen.SwitchGetCondition(parent)
		if condition == search {
			binaryen.SwitchSetCondition(parent, replacement)
			return condition
		}
		value := binaryen.SwitchGetValue(parent)
		if value == search {
			binaryen.SwitchSetValue(parent, replacement)
			return value
		}

	case module.ExpressionIdCall:
		numOperands := binaryen.CallGetNumOperands(parent)
		for i := module.Index(0); i < numOperands; i++ {
			operand := binaryen.CallGetOperandAt(parent, i)
			if operand == search {
				binaryen.CallSetOperandAt(parent, i, replacement)
				return operand
			}
		}

	case module.ExpressionIdCallIndirect:
		target := binaryen.CallIndirectGetTarget(parent)
		if target == search {
			binaryen.CallIndirectSetTarget(parent, replacement)
			return target
		}
		numOperands := binaryen.CallIndirectGetNumOperands(parent)
		for i := module.Index(0); i < numOperands; i++ {
			operand := binaryen.CallIndirectGetOperandAt(parent, i)
			if operand == search {
				binaryen.CallIndirectSetOperandAt(parent, i, replacement)
				return operand
			}
		}

	case module.ExpressionIdLocalGet:
		// no children

	case module.ExpressionIdLocalSet:
		value := binaryen.LocalSetGetValue(parent)
		if value == search {
			binaryen.LocalSetSetValue(parent, replacement)
			return value
		}

	case module.ExpressionIdGlobalGet:
		// no children

	case module.ExpressionIdGlobalSet:
		value := binaryen.GlobalSetGetValue(parent)
		if value == search {
			binaryen.GlobalSetSetValue(parent, replacement)
			return value
		}

	case module.ExpressionIdLoad:
		ptr := binaryen.LoadGetPtr(parent)
		if ptr == search {
			binaryen.LoadSetPtr(parent, replacement)
			return ptr
		}

	case module.ExpressionIdStore:
		ptr := binaryen.StoreGetPtr(parent)
		if ptr == search {
			binaryen.StoreSetPtr(parent, replacement)
			return ptr
		}
		value := binaryen.StoreGetValue(parent)
		if value == search {
			binaryen.StoreSetValue(parent, replacement)
			return value
		}

	case module.ExpressionIdConst:
		// no children

	case module.ExpressionIdUnary:
		value := binaryen.UnaryGetValue(parent)
		if value == search {
			binaryen.UnarySetValue(parent, replacement)
			return value
		}

	case module.ExpressionIdBinary:
		left := binaryen.BinaryGetLeft(parent)
		if left == search {
			binaryen.BinarySetLeft(parent, replacement)
			return left
		}
		right := binaryen.BinaryGetRight(parent)
		if right == search {
			binaryen.BinarySetRight(parent, replacement)
			return right
		}

	case module.ExpressionIdSelect:
		ifTrue := binaryen.SelectGetIfTrue(parent)
		if ifTrue == search {
			binaryen.SelectSetIfTrue(parent, replacement)
			return ifTrue
		}
		ifFalse := binaryen.SelectGetIfFalse(parent)
		if ifFalse == search {
			binaryen.SelectSetIfFalse(parent, replacement)
			return ifFalse
		}
		condition := binaryen.SelectGetCondition(parent)
		if condition == search {
			binaryen.SelectSetCondition(parent, replacement)
			return condition
		}

	case module.ExpressionIdDrop:
		value := binaryen.DropGetValue(parent)
		if value == search {
			binaryen.DropSetValue(parent, replacement)
			return value
		}

	case module.ExpressionIdReturn:
		value := binaryen.ReturnGetValue(parent)
		if value == search {
			binaryen.ReturnSetValue(parent, replacement)
			return value
		}

	case module.ExpressionIdMemorySize:
		// no children

	case module.ExpressionIdMemoryGrow:
		delta := binaryen.MemoryGrowGetDelta(parent)
		if delta == search {
			binaryen.MemoryGrowSetDelta(parent, replacement)
			return delta
		}

	case module.ExpressionIdNop:
		// no children

	case module.ExpressionIdUnreachable:
		// no children

	case module.ExpressionIdAtomicRMW:
		ptr := binaryen.AtomicRMWGetPtr(parent)
		if ptr == search {
			binaryen.AtomicRMWSetPtr(parent, replacement)
			return ptr
		}
		value := binaryen.AtomicRMWGetValue(parent)
		if value == search {
			binaryen.AtomicRMWSetValue(parent, replacement)
			return value
		}

	case module.ExpressionIdAtomicCmpxchg:
		ptr := binaryen.AtomicCmpxchgGetPtr(parent)
		if ptr == search {
			binaryen.AtomicCmpxchgSetPtr(parent, replacement)
			return ptr
		}
		expected := binaryen.AtomicCmpxchgGetExpected(parent)
		if expected == search {
			binaryen.AtomicCmpxchgSetExpected(parent, replacement)
			return expected
		}
		repl := binaryen.AtomicCmpxchgGetReplacement(parent)
		if repl == search {
			binaryen.AtomicCmpxchgSetReplacement(parent, replacement)
			return repl
		}

	case module.ExpressionIdAtomicWait:
		ptr := binaryen.AtomicWaitGetPtr(parent)
		if ptr == search {
			binaryen.AtomicWaitSetPtr(parent, replacement)
			return ptr
		}
		expected := binaryen.AtomicWaitGetExpected(parent)
		if expected == search {
			binaryen.AtomicWaitSetExpected(parent, replacement)
			return expected
		}
		timeout := binaryen.AtomicWaitGetTimeout(parent)
		if timeout == search {
			binaryen.AtomicWaitSetTimeout(parent, replacement)
			return timeout
		}

	case module.ExpressionIdAtomicNotify:
		ptr := binaryen.AtomicNotifyGetPtr(parent)
		if ptr == search {
			binaryen.AtomicNotifySetPtr(parent, replacement)
			return ptr
		}
		notifyCount := binaryen.AtomicNotifyGetNotifyCount(parent)
		if notifyCount == search {
			binaryen.AtomicNotifySetNotifyCount(parent, replacement)
			return notifyCount
		}

	case module.ExpressionIdAtomicFence:
		// no children

	case module.ExpressionIdSIMDExtract:
		vec := binaryen.SIMDExtractGetVec(parent)
		if vec == search {
			binaryen.SIMDExtractSetVec(parent, replacement)
			return vec
		}

	case module.ExpressionIdSIMDReplace:
		vec := binaryen.SIMDReplaceGetVec(parent)
		if vec == search {
			binaryen.SIMDReplaceSetVec(parent, replacement)
			return vec
		}
		value := binaryen.SIMDReplaceGetValue(parent)
		if value == search {
			binaryen.SIMDReplaceSetValue(parent, replacement)
			return value
		}

	case module.ExpressionIdSIMDShuffle:
		left := binaryen.SIMDShuffleGetLeft(parent)
		if left == search {
			binaryen.SIMDShuffleSetLeft(parent, replacement)
			return left
		}
		right := binaryen.SIMDShuffleGetRight(parent)
		if right == search {
			binaryen.SIMDShuffleSetRight(parent, replacement)
			return right
		}

	case module.ExpressionIdSIMDTernary:
		a := binaryen.SIMDTernaryGetA(parent)
		if a == search {
			binaryen.SIMDTernarySetA(parent, replacement)
			return a
		}
		b := binaryen.SIMDTernaryGetB(parent)
		if b == search {
			binaryen.SIMDTernarySetB(parent, replacement)
			return b
		}
		c := binaryen.SIMDTernaryGetC(parent)
		if c == search {
			binaryen.SIMDTernarySetC(parent, replacement)
			return c
		}

	case module.ExpressionIdSIMDShift:
		vec := binaryen.SIMDShiftGetVec(parent)
		if vec == search {
			binaryen.SIMDShiftSetVec(parent, replacement)
			return vec
		}
		shift := binaryen.SIMDShiftGetShift(parent)
		if shift == search {
			binaryen.SIMDShiftSetShift(parent, replacement)
			return shift
		}

	case module.ExpressionIdSIMDLoad:
		ptr := binaryen.SIMDLoadGetPtr(parent)
		if ptr == search {
			binaryen.SIMDLoadSetPtr(parent, replacement)
			return ptr
		}

	case module.ExpressionIdSIMDLoadStoreLane:
		ptr := binaryen.SIMDLoadStoreLaneGetPtr(parent)
		if ptr == search {
			binaryen.SIMDLoadStoreLaneSetPtr(parent, replacement)
			return ptr
		}
		vec := binaryen.SIMDLoadStoreLaneGetVec(parent)
		if vec == search {
			binaryen.SIMDLoadStoreLaneSetVec(parent, replacement)
			return ptr // Note: TS returns ptr here (bug in TS source)
		}

	case module.ExpressionIdMemoryInit:
		dest := binaryen.MemoryInitGetDest(parent)
		if dest == search {
			binaryen.MemoryInitSetDest(parent, replacement)
			return dest
		}
		offset := binaryen.MemoryInitGetOffset(parent)
		if offset == search {
			binaryen.MemoryInitSetOffset(parent, replacement)
			return offset
		}
		size := binaryen.MemoryInitGetSize(parent)
		if size == search {
			binaryen.MemoryInitSetSize(parent, replacement)
			return size
		}

	case module.ExpressionIdDataDrop:
		// no children

	case module.ExpressionIdMemoryCopy:
		dest := binaryen.MemoryCopyGetDest(parent)
		if dest == search {
			binaryen.MemoryCopySetDest(parent, replacement)
			return dest
		}
		source := binaryen.MemoryCopyGetSource(parent)
		if source == search {
			binaryen.MemoryCopySetSource(parent, replacement)
			return source
		}
		size := binaryen.MemoryCopyGetSize(parent)
		if size == search {
			binaryen.MemoryCopySetSize(parent, replacement)
			return size
		}

	case module.ExpressionIdMemoryFill:
		dest := binaryen.MemoryFillGetDest(parent)
		if dest == search {
			binaryen.MemoryFillSetDest(parent, replacement)
			return dest
		}
		value := binaryen.MemoryFillGetValue(parent)
		if value == search {
			binaryen.MemoryFillSetValue(parent, replacement)
			return value
		}
		size := binaryen.MemoryFillGetSize(parent)
		if size == search {
			binaryen.MemoryFillSetSize(parent, replacement)
			return size
		}

	case module.ExpressionIdPop:
		// no children

	case module.ExpressionIdRefNull:
		// no children

	case module.ExpressionIdRefIsNull:
		value := binaryen.RefIsNullGetValue(parent)
		if value == search {
			binaryen.RefIsNullSetValue(parent, replacement)
			return value
		}

	case module.ExpressionIdRefFunc:
		// no expression children

	case module.ExpressionIdRefEq:
		left := binaryen.RefEqGetLeft(parent)
		if left == search {
			binaryen.RefEqSetLeft(parent, replacement)
			return left
		}
		right := binaryen.RefEqGetRight(parent)
		if right == search {
			binaryen.RefEqSetRight(parent, replacement)
			return right
		}

	case module.ExpressionIdTry:
		body := binaryen.TryGetBody(parent)
		if body == search {
			binaryen.TrySetBody(parent, replacement)
			return body
		}
		numCatchBodies := binaryen.TryGetNumCatchBodies(parent)
		for i := module.Index(0); i < numCatchBodies; i++ {
			catchBody := binaryen.TryGetCatchBodyAt(parent, i)
			if catchBody == search {
				binaryen.TrySetCatchBodyAt(parent, i, replacement)
				return catchBody
			}
		}

	case module.ExpressionIdThrow:
		numOperands := binaryen.ThrowGetNumOperands(parent)
		for i := module.Index(0); i < numOperands; i++ {
			operand := binaryen.ThrowGetOperandAt(parent, i)
			if operand == search {
				binaryen.ThrowSetOperandAt(parent, i, replacement)
				return operand
			}
		}

	case module.ExpressionIdRethrow:
		// no children

	case module.ExpressionIdTupleMake:
		numOperands := binaryen.TupleMakeGetNumOperands(parent)
		for i := module.Index(0); i < numOperands; i++ {
			operand := binaryen.TupleMakeGetOperandAt(parent, i)
			if operand == search {
				binaryen.TupleMakeSetOperandAt(parent, i, replacement)
				return operand
			}
		}

	case module.ExpressionIdTupleExtract:
		tuple := binaryen.TupleExtractGetTuple(parent)
		if tuple == search {
			binaryen.TupleExtractSetTuple(parent, replacement)
			return tuple
		}

	case module.ExpressionIdRefI31:
		value := binaryen.RefI31GetValue(parent)
		if value == search {
			binaryen.RefI31SetValue(parent, replacement)
			return value
		}

	case module.ExpressionIdI31Get:
		i31Expr := binaryen.I31GetGetI31(parent)
		if i31Expr == search {
			binaryen.I31GetSetI31(parent, replacement)
			return i31Expr
		}

	case module.ExpressionIdCallRef:
		numOperands := binaryen.CallRefGetNumOperands(parent)
		for i := module.Index(0); i < numOperands; i++ {
			operand := binaryen.CallRefGetOperandAt(parent, i)
			if operand == search {
				binaryen.CallRefSetOperandAt(parent, i, replacement)
				return operand
			}
		}
		target := binaryen.CallRefGetTarget(parent)
		if target == search {
			binaryen.CallRefSetTarget(parent, replacement)
			return target
		}

	case module.ExpressionIdRefTest:
		ref := binaryen.RefTestGetRef(parent)
		if ref == search {
			binaryen.RefTestSetRef(parent, replacement)
			return ref
		}

	case module.ExpressionIdRefCast:
		ref := binaryen.RefCastGetRef(parent)
		if ref == search {
			binaryen.RefCastSetRef(parent, replacement)
			return ref
		}

	case module.ExpressionIdBrOn:
		ref := binaryen.BrOnGetRef(parent)
		if ref == search {
			binaryen.BrOnSetRef(parent, replacement)
			return ref
		}

	case module.ExpressionIdStructNew:
		numOperands := binaryen.StructNewGetNumOperands(parent)
		for i := module.Index(0); i < numOperands; i++ {
			operand := binaryen.StructNewGetOperandAt(parent, i)
			if operand == search {
				binaryen.StructNewSetOperandAt(parent, i, replacement)
				return operand
			}
		}

	case module.ExpressionIdStructGet:
		ref := binaryen.StructGetGetRef(parent)
		if ref == search {
			binaryen.StructGetSetRef(parent, replacement)
			return ref
		}

	case module.ExpressionIdStructSet:
		ref := binaryen.StructSetGetRef(parent)
		if ref == search {
			binaryen.StructSetSetRef(parent, replacement)
			return ref
		}
		value := binaryen.StructSetGetValue(parent)
		if value == search {
			binaryen.StructSetSetValue(parent, replacement)
			return value
		}

	case module.ExpressionIdArrayNew:
		size := binaryen.ArrayNewGetSize(parent)
		if size == search {
			binaryen.ArrayNewSetSize(parent, replacement)
			return size
		}
		init := binaryen.ArrayNewGetInit(parent)
		if init == search {
			binaryen.ArrayNewSetInit(parent, replacement)
			return init
		}

	case module.ExpressionIdArrayNewFixed:
		numValues := binaryen.ArrayNewFixedGetNumValues(parent)
		for i := module.Index(0); i < numValues; i++ {
			value := binaryen.ArrayNewFixedGetValueAt(parent, i)
			if value == search {
				binaryen.ArrayNewFixedSetValueAt(parent, i, replacement)
				return value
			}
		}

	case module.ExpressionIdArrayGet:
		ref := binaryen.ArrayGetGetRef(parent)
		if ref == search {
			binaryen.ArrayGetSetRef(parent, replacement)
			return ref
		}
		index := binaryen.ArrayGetGetIndex(parent)
		if index == search {
			binaryen.ArrayGetSetIndex(parent, replacement)
			return index
		}

	case module.ExpressionIdArraySet:
		ref := binaryen.ArraySetGetRef(parent)
		if ref == search {
			binaryen.ArraySetSetRef(parent, replacement)
			return ref
		}
		index := binaryen.ArraySetGetIndex(parent)
		if index == search {
			binaryen.ArraySetSetIndex(parent, replacement)
			return index
		}
		value := binaryen.ArraySetGetValue(parent)
		if value == search {
			binaryen.ArraySetSetValue(parent, replacement)
			return value
		}

	case module.ExpressionIdArrayLen:
		ref := binaryen.ArrayLenGetRef(parent)
		if ref == search {
			binaryen.ArrayLenSetRef(parent, replacement)
			return ref
		}

	case module.ExpressionIdArrayCopy:
		destRef := binaryen.ArrayCopyGetDestRef(parent)
		if destRef == search {
			binaryen.ArrayCopySetDestRef(parent, replacement)
			return destRef
		}
		destIndex := binaryen.ArrayCopyGetDestIndex(parent)
		if destIndex == search {
			binaryen.ArrayCopySetDestIndex(parent, replacement)
			return destIndex
		}
		srcRef := binaryen.ArrayCopyGetSrcRef(parent)
		if srcRef == search {
			binaryen.ArrayCopySetSrcRef(parent, replacement)
			return srcRef
		}
		srcIndex := binaryen.ArrayCopyGetSrcIndex(parent)
		if srcIndex == search {
			binaryen.ArrayCopySetSrcIndex(parent, replacement)
			return srcIndex
		}
		length := binaryen.ArrayCopyGetLength(parent)
		if length == search {
			binaryen.ArrayCopySetLength(parent, replacement)
			return length
		}

	case module.ExpressionIdRefAs:
		value := binaryen.RefAsGetValue(parent)
		if value == search {
			binaryen.RefAsSetValue(parent, replacement)
			return value
		}

	case module.ExpressionIdStringNew:
		ptr := binaryen.StringNewGetRef(parent)
		if ptr == search {
			binaryen.StringNewSetRef(parent, replacement)
			return ptr
		}
		start := binaryen.StringNewGetStart(parent)
		if start == search {
			binaryen.StringNewSetStart(parent, replacement)
			return start
		}
		end := binaryen.StringNewGetEnd(parent)
		if end == search {
			binaryen.StringNewSetEnd(parent, replacement)
			return end
		}

	case module.ExpressionIdStringConst:
		// no children

	case module.ExpressionIdStringMeasure:
		ref := binaryen.StringMeasureGetRef(parent)
		if ref == search {
			binaryen.StringMeasureSetRef(parent, replacement)
			return ref
		}

	case module.ExpressionIdStringEncode:
		ref := binaryen.StringEncodeGetStr(parent)
		if ref == search {
			binaryen.StringEncodeSetStr(parent, replacement)
			return ref
		}
		ptr := binaryen.StringEncodeGetArray(parent)
		if ptr == search {
			binaryen.StringEncodeSetArray(parent, replacement)
			return ptr
		}
		start := binaryen.StringEncodeGetStart(parent)
		if start == search {
			binaryen.StringEncodeSetStart(parent, replacement)
			return start
		}

	case module.ExpressionIdStringConcat:
		left := binaryen.StringConcatGetLeft(parent)
		if left == search {
			binaryen.StringConcatSetLeft(parent, replacement)
			return left
		}
		right := binaryen.StringConcatGetRight(parent)
		if right == search {
			binaryen.StringConcatSetRight(parent, replacement)
			return right
		}

	case module.ExpressionIdStringEq:
		left := binaryen.StringEqGetLeft(parent)
		if left == search {
			binaryen.StringEqSetLeft(parent, replacement)
			return left
		}
		right := binaryen.StringEqGetRight(parent)
		if right == search {
			binaryen.StringEqSetRight(parent, replacement)
			return right
		}

	case module.ExpressionIdStringWTF16Get:
		ref := binaryen.StringWTF16GetGetRef(parent)
		if ref == search {
			binaryen.StringWTF16GetSetRef(parent, replacement)
			return ref
		}
		pos := binaryen.StringWTF16GetGetPos(parent)
		if pos == search {
			binaryen.StringWTF16GetSetPos(parent, replacement)
			return pos
		}

	case module.ExpressionIdStringSliceWTF:
		ref := binaryen.StringSliceWTFGetRef(parent)
		if ref == search {
			binaryen.StringSliceWTFSetRef(parent, replacement)
			return ref
		}
		start := binaryen.StringSliceWTFGetStart(parent)
		if start == search {
			binaryen.StringSliceWTFSetStart(parent, replacement)
			return start
		}
		end := binaryen.StringSliceWTFGetEnd(parent)
		if end == search {
			binaryen.StringSliceWTFSetEnd(parent, replacement)
			return end
		}

	default:
		panic(fmt.Sprintf("unexpected expression id: %d", binaryen.ExpressionGetId(parent)))
	}
	return 0
}
