// Ported from: assemblyscript/src/passes/pass.ts (Visitor abstract class, lines 270-1235)
//
// Base class of custom Binaryen visitors. Each visit method corresponds to a
// Binaryen expression kind. The Visit method dispatches to the appropriate
// visit method based on expression ID.
package passes

import (
	"fmt"

	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/pkg/binaryen"
)

// Visitor is an interface that defines visit methods for all Binaryen expression types.
// The default implementation of each method is a no-op. Implementations embed
// BaseVisitor and override only the methods they need.
type Visitor interface {
	// Expression visitors
	VisitBlock(expr module.ExpressionRef)
	VisitIf(expr module.ExpressionRef)
	VisitLoop(expr module.ExpressionRef)
	VisitBreak(expr module.ExpressionRef)
	VisitSwitch(expr module.ExpressionRef)
	VisitCallPre(expr module.ExpressionRef)
	VisitCall(expr module.ExpressionRef)
	VisitCallIndirectPre(expr module.ExpressionRef)
	VisitCallIndirect(expr module.ExpressionRef)
	VisitLocalGet(expr module.ExpressionRef)
	VisitLocalSet(expr module.ExpressionRef)
	VisitGlobalGet(expr module.ExpressionRef)
	VisitGlobalSet(expr module.ExpressionRef)
	VisitLoad(expr module.ExpressionRef)
	VisitStore(expr module.ExpressionRef)
	VisitConst(expr module.ExpressionRef)
	VisitUnary(expr module.ExpressionRef)
	VisitBinary(expr module.ExpressionRef)
	VisitSelect(expr module.ExpressionRef)
	VisitDrop(expr module.ExpressionRef)
	VisitReturn(expr module.ExpressionRef)
	VisitMemorySize(expr module.ExpressionRef)
	VisitMemoryGrow(expr module.ExpressionRef)
	VisitNop(expr module.ExpressionRef)
	VisitUnreachable(expr module.ExpressionRef)
	VisitAtomicRMW(expr module.ExpressionRef)
	VisitAtomicCmpxchg(expr module.ExpressionRef)
	VisitAtomicWait(expr module.ExpressionRef)
	VisitAtomicNotify(expr module.ExpressionRef)
	VisitAtomicFence(expr module.ExpressionRef)
	VisitSIMDExtract(expr module.ExpressionRef)
	VisitSIMDReplace(expr module.ExpressionRef)
	VisitSIMDShuffle(expr module.ExpressionRef)
	VisitSIMDTernary(expr module.ExpressionRef)
	VisitSIMDShift(expr module.ExpressionRef)
	VisitSIMDLoad(expr module.ExpressionRef)
	VisitSIMDLoadStoreLane(expr module.ExpressionRef)
	VisitMemoryInit(expr module.ExpressionRef)
	VisitDataDrop(expr module.ExpressionRef)
	VisitMemoryCopy(expr module.ExpressionRef)
	VisitMemoryFill(expr module.ExpressionRef)
	VisitPop(expr module.ExpressionRef)
	VisitRefNull(expr module.ExpressionRef)
	VisitRefIsNull(expr module.ExpressionRef)
	VisitRefFunc(expr module.ExpressionRef)
	VisitRefEq(expr module.ExpressionRef)
	VisitTry(expr module.ExpressionRef)
	VisitThrow(expr module.ExpressionRef)
	VisitRethrow(expr module.ExpressionRef)
	VisitTupleMake(expr module.ExpressionRef)
	VisitTupleExtract(expr module.ExpressionRef)
	VisitRefI31(expr module.ExpressionRef)
	VisitI31Get(expr module.ExpressionRef)
	VisitCallRef(expr module.ExpressionRef)
	VisitRefTest(expr module.ExpressionRef)
	VisitRefCast(expr module.ExpressionRef)
	VisitBrOn(expr module.ExpressionRef)
	VisitStructNew(expr module.ExpressionRef)
	VisitStructGet(expr module.ExpressionRef)
	VisitStructSet(expr module.ExpressionRef)
	VisitArrayNew(expr module.ExpressionRef)
	VisitArrayNewFixed(expr module.ExpressionRef)
	VisitArrayGet(expr module.ExpressionRef)
	VisitArraySet(expr module.ExpressionRef)
	VisitArrayLen(expr module.ExpressionRef)
	VisitArrayCopy(expr module.ExpressionRef)
	VisitRefAs(expr module.ExpressionRef)
	VisitStringNew(expr module.ExpressionRef)
	VisitStringConst(expr module.ExpressionRef)
	VisitStringMeasure(expr module.ExpressionRef)
	VisitStringEncode(expr module.ExpressionRef)
	VisitStringConcat(expr module.ExpressionRef)
	VisitStringEq(expr module.ExpressionRef)
	VisitStringAs(expr module.ExpressionRef)
	VisitStringWTF8Advance(expr module.ExpressionRef)
	VisitStringWTF16Get(expr module.ExpressionRef)
	VisitStringIterNext(expr module.ExpressionRef)
	VisitStringIterMove(expr module.ExpressionRef)
	VisitStringSliceWTF(expr module.ExpressionRef)
	VisitStringSliceIter(expr module.ExpressionRef)

	// Immediate visitors
	VisitName(name string)
	VisitLabel(name string)
	VisitIndex(index module.Index)
	VisitTag(name string)

	// Visit dispatches to the appropriate visitor method.
	Visit(expr module.ExpressionRef)

	// Accessors
	CurrentExpression() module.ExpressionRef
	ParentExpressionOrNull() module.ExpressionRef
}

// BaseVisitor provides default no-op implementations for all Visitor methods.
// It also maintains the expression stack and current expression tracking.
type BaseVisitor struct {
	stack              []module.ExpressionRef
	currentExpression  module.ExpressionRef
	// self holds the concrete Visitor implementation for virtual dispatch.
	self Visitor
}

// InitVisitor sets the self pointer for virtual dispatch.
// Must be called before using the visitor.
func (v *BaseVisitor) InitVisitor(self Visitor) {
	v.self = self
}

// CurrentExpression returns the current expression being walked.
func (v *BaseVisitor) CurrentExpression() module.ExpressionRef {
	if v.currentExpression == 0 {
		panic("not walking expressions")
	}
	return v.currentExpression
}

// ParentExpressionOrNull returns the parent expression of the current expression
// being walked. Returns zero if already the top-most expression.
func (v *BaseVisitor) ParentExpressionOrNull() module.ExpressionRef {
	length := len(v.stack)
	if length > 0 {
		return v.stack[length-1]
	}
	return 0
}

// Default no-op visitor methods

func (v *BaseVisitor) VisitBlock(expr module.ExpressionRef)            {}
func (v *BaseVisitor) VisitIf(expr module.ExpressionRef)               {}
func (v *BaseVisitor) VisitLoop(expr module.ExpressionRef)             {}
func (v *BaseVisitor) VisitBreak(expr module.ExpressionRef)            {}
func (v *BaseVisitor) VisitSwitch(expr module.ExpressionRef)           {}
func (v *BaseVisitor) VisitCallPre(expr module.ExpressionRef)          {}
func (v *BaseVisitor) VisitCall(expr module.ExpressionRef)             {}
func (v *BaseVisitor) VisitCallIndirectPre(expr module.ExpressionRef)  {}
func (v *BaseVisitor) VisitCallIndirect(expr module.ExpressionRef)     {}
func (v *BaseVisitor) VisitLocalGet(expr module.ExpressionRef)         {}
func (v *BaseVisitor) VisitLocalSet(expr module.ExpressionRef)         {}
func (v *BaseVisitor) VisitGlobalGet(expr module.ExpressionRef)        {}
func (v *BaseVisitor) VisitGlobalSet(expr module.ExpressionRef)        {}
func (v *BaseVisitor) VisitLoad(expr module.ExpressionRef)             {}
func (v *BaseVisitor) VisitStore(expr module.ExpressionRef)            {}
func (v *BaseVisitor) VisitConst(expr module.ExpressionRef)            {}
func (v *BaseVisitor) VisitUnary(expr module.ExpressionRef)            {}
func (v *BaseVisitor) VisitBinary(expr module.ExpressionRef)           {}
func (v *BaseVisitor) VisitSelect(expr module.ExpressionRef)           {}
func (v *BaseVisitor) VisitDrop(expr module.ExpressionRef)             {}
func (v *BaseVisitor) VisitReturn(expr module.ExpressionRef)           {}
func (v *BaseVisitor) VisitMemorySize(expr module.ExpressionRef)       {}
func (v *BaseVisitor) VisitMemoryGrow(expr module.ExpressionRef)       {}
func (v *BaseVisitor) VisitNop(expr module.ExpressionRef)              {}
func (v *BaseVisitor) VisitUnreachable(expr module.ExpressionRef)      {}
func (v *BaseVisitor) VisitAtomicRMW(expr module.ExpressionRef)        {}
func (v *BaseVisitor) VisitAtomicCmpxchg(expr module.ExpressionRef)    {}
func (v *BaseVisitor) VisitAtomicWait(expr module.ExpressionRef)       {}
func (v *BaseVisitor) VisitAtomicNotify(expr module.ExpressionRef)     {}
func (v *BaseVisitor) VisitAtomicFence(expr module.ExpressionRef)      {}
func (v *BaseVisitor) VisitSIMDExtract(expr module.ExpressionRef)      {}
func (v *BaseVisitor) VisitSIMDReplace(expr module.ExpressionRef)      {}
func (v *BaseVisitor) VisitSIMDShuffle(expr module.ExpressionRef)      {}
func (v *BaseVisitor) VisitSIMDTernary(expr module.ExpressionRef)      {}
func (v *BaseVisitor) VisitSIMDShift(expr module.ExpressionRef)        {}
func (v *BaseVisitor) VisitSIMDLoad(expr module.ExpressionRef)         {}
func (v *BaseVisitor) VisitSIMDLoadStoreLane(expr module.ExpressionRef) {}
func (v *BaseVisitor) VisitMemoryInit(expr module.ExpressionRef)       {}
func (v *BaseVisitor) VisitDataDrop(expr module.ExpressionRef)         {}
func (v *BaseVisitor) VisitMemoryCopy(expr module.ExpressionRef)       {}
func (v *BaseVisitor) VisitMemoryFill(expr module.ExpressionRef)       {}
func (v *BaseVisitor) VisitPop(expr module.ExpressionRef)              {}
func (v *BaseVisitor) VisitRefNull(expr module.ExpressionRef)          {}
func (v *BaseVisitor) VisitRefIsNull(expr module.ExpressionRef)        {}
func (v *BaseVisitor) VisitRefFunc(expr module.ExpressionRef)          {}
func (v *BaseVisitor) VisitRefEq(expr module.ExpressionRef)            {}
func (v *BaseVisitor) VisitTry(expr module.ExpressionRef)              {}
func (v *BaseVisitor) VisitThrow(expr module.ExpressionRef)            {}
func (v *BaseVisitor) VisitRethrow(expr module.ExpressionRef)          {}
func (v *BaseVisitor) VisitTupleMake(expr module.ExpressionRef)        {}
func (v *BaseVisitor) VisitTupleExtract(expr module.ExpressionRef)     {}
func (v *BaseVisitor) VisitRefI31(expr module.ExpressionRef)           {}
func (v *BaseVisitor) VisitI31Get(expr module.ExpressionRef)           {}
func (v *BaseVisitor) VisitCallRef(expr module.ExpressionRef)          {}
func (v *BaseVisitor) VisitRefTest(expr module.ExpressionRef)          {}
func (v *BaseVisitor) VisitRefCast(expr module.ExpressionRef)          {}
func (v *BaseVisitor) VisitBrOn(expr module.ExpressionRef)             {}
func (v *BaseVisitor) VisitStructNew(expr module.ExpressionRef)        {}
func (v *BaseVisitor) VisitStructGet(expr module.ExpressionRef)        {}
func (v *BaseVisitor) VisitStructSet(expr module.ExpressionRef)        {}
func (v *BaseVisitor) VisitArrayNew(expr module.ExpressionRef)         {}
func (v *BaseVisitor) VisitArrayNewFixed(expr module.ExpressionRef)    {}
func (v *BaseVisitor) VisitArrayGet(expr module.ExpressionRef)         {}
func (v *BaseVisitor) VisitArraySet(expr module.ExpressionRef)         {}
func (v *BaseVisitor) VisitArrayLen(expr module.ExpressionRef)         {}
func (v *BaseVisitor) VisitArrayCopy(expr module.ExpressionRef)        {}
func (v *BaseVisitor) VisitRefAs(expr module.ExpressionRef)            {}
func (v *BaseVisitor) VisitStringNew(expr module.ExpressionRef)        {}
func (v *BaseVisitor) VisitStringConst(expr module.ExpressionRef)      {}
func (v *BaseVisitor) VisitStringMeasure(expr module.ExpressionRef)    {}
func (v *BaseVisitor) VisitStringEncode(expr module.ExpressionRef)     {}
func (v *BaseVisitor) VisitStringConcat(expr module.ExpressionRef)     {}
func (v *BaseVisitor) VisitStringEq(expr module.ExpressionRef)         {}
func (v *BaseVisitor) VisitStringAs(expr module.ExpressionRef)         {}
func (v *BaseVisitor) VisitStringWTF8Advance(expr module.ExpressionRef) {}
func (v *BaseVisitor) VisitStringWTF16Get(expr module.ExpressionRef)   {}
func (v *BaseVisitor) VisitStringIterNext(expr module.ExpressionRef)   {}
func (v *BaseVisitor) VisitStringIterMove(expr module.ExpressionRef)   {}
func (v *BaseVisitor) VisitStringSliceWTF(expr module.ExpressionRef)   {}
func (v *BaseVisitor) VisitStringSliceIter(expr module.ExpressionRef)  {}

func (v *BaseVisitor) VisitName(name string)      {}
func (v *BaseVisitor) VisitLabel(name string)      {}
func (v *BaseVisitor) VisitIndex(index module.Index) {}
func (v *BaseVisitor) VisitTag(name string)        {}

// Visit dispatches to the appropriate visitor method based on expression ID.
// This is a post-order traversal: children are visited before the parent's
// visit method is called.
func (v *BaseVisitor) Visit(expr module.ExpressionRef) {
	if expr == 0 {
		panic("Visit called with null expression")
	}
	previousExpression := v.currentExpression
	v.currentExpression = expr

	self := v.self

	switch binaryen.ExpressionGetId(expr) {
	case module.ExpressionIdBlock:
		v.stack = append(v.stack, expr)
		name := binaryen.BlockGetName(expr)
		if name != "" {
			self.VisitLabel(name)
		}
		for i := module.Index(0); i < binaryen.BlockGetNumChildren(expr); i++ {
			self.Visit(binaryen.BlockGetChildAt(expr, i))
		}
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitBlock(expr)

	case module.ExpressionIdIf:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.IfGetCondition(expr))
		self.Visit(binaryen.IfGetIfTrue(expr))
		ifFalse := binaryen.IfGetIfFalse(expr)
		if ifFalse != 0 {
			self.Visit(ifFalse)
		}
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitIf(expr)

	case module.ExpressionIdLoop:
		v.stack = append(v.stack, expr)
		name := binaryen.LoopGetName(expr)
		if name != "" {
			self.VisitLabel(name)
		}
		self.Visit(binaryen.LoopGetBody(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitLoop(expr)

	case module.ExpressionIdBreak:
		v.stack = append(v.stack, expr)
		self.VisitLabel(binaryen.BreakGetName(expr))
		condition := binaryen.BreakGetCondition(expr)
		if condition != 0 {
			self.Visit(condition)
		}
		value := binaryen.BreakGetValue(expr)
		if value != 0 {
			self.Visit(value)
		}
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitBreak(expr)

	case module.ExpressionIdSwitch:
		v.stack = append(v.stack, expr)
		defaultName := binaryen.SwitchGetDefaultName(expr)
		if defaultName != "" {
			self.VisitLabel(defaultName)
		}
		numNames := binaryen.SwitchGetNumNames(expr)
		for i := module.Index(0); i < numNames; i++ {
			self.VisitLabel(binaryen.SwitchGetNameAt(expr, i))
		}
		self.Visit(binaryen.SwitchGetCondition(expr))
		value := binaryen.SwitchGetValue(expr)
		if value != 0 {
			self.Visit(value)
		}
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitSwitch(expr)

	case module.ExpressionIdCall:
		self.VisitCallPre(expr)
		v.stack = append(v.stack, expr)
		self.VisitName(binaryen.CallGetTarget(expr))
		numOperands := binaryen.CallGetNumOperands(expr)
		for i := module.Index(0); i < numOperands; i++ {
			self.Visit(binaryen.CallGetOperandAt(expr, i))
		}
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitCall(expr)

	case module.ExpressionIdCallIndirect:
		self.VisitCallIndirectPre(expr)
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.CallIndirectGetTarget(expr))
		numOperands := binaryen.CallIndirectGetNumOperands(expr)
		for i := module.Index(0); i < numOperands; i++ {
			self.Visit(binaryen.CallIndirectGetOperandAt(expr, i))
		}
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitCallIndirect(expr)

	case module.ExpressionIdLocalGet:
		v.stack = append(v.stack, expr)
		self.VisitIndex(binaryen.LocalGetGetIndex(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitLocalGet(expr)

	case module.ExpressionIdLocalSet:
		v.stack = append(v.stack, expr)
		self.VisitIndex(binaryen.LocalSetGetIndex(expr))
		self.Visit(binaryen.LocalSetGetValue(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitLocalSet(expr)

	case module.ExpressionIdGlobalGet:
		v.stack = append(v.stack, expr)
		self.VisitName(binaryen.GlobalGetGetName(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitGlobalGet(expr)

	case module.ExpressionIdGlobalSet:
		v.stack = append(v.stack, expr)
		self.VisitName(binaryen.GlobalSetGetName(expr))
		self.Visit(binaryen.GlobalSetGetValue(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitGlobalSet(expr)

	case module.ExpressionIdLoad:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.LoadGetPtr(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitLoad(expr)

	case module.ExpressionIdStore:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.StoreGetPtr(expr))
		self.Visit(binaryen.StoreGetValue(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitStore(expr)

	case module.ExpressionIdConst:
		self.VisitConst(expr)

	case module.ExpressionIdUnary:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.UnaryGetValue(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitUnary(expr)

	case module.ExpressionIdBinary:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.BinaryGetLeft(expr))
		self.Visit(binaryen.BinaryGetRight(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitBinary(expr)

	case module.ExpressionIdSelect:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.SelectGetIfTrue(expr))
		self.Visit(binaryen.SelectGetIfFalse(expr))
		self.Visit(binaryen.SelectGetCondition(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitSelect(expr)

	case module.ExpressionIdDrop:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.DropGetValue(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitDrop(expr)

	case module.ExpressionIdReturn:
		value := binaryen.ReturnGetValue(expr)
		if value != 0 {
			v.stack = append(v.stack, expr)
			self.Visit(value)
			v.stack = v.stack[:len(v.stack)-1]
		}
		self.VisitReturn(expr)

	case module.ExpressionIdMemorySize:
		self.VisitMemorySize(expr)

	case module.ExpressionIdMemoryGrow:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.MemoryGrowGetDelta(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitMemoryGrow(expr)

	case module.ExpressionIdNop:
		self.VisitNop(expr)

	case module.ExpressionIdUnreachable:
		self.VisitUnreachable(expr)

	case module.ExpressionIdAtomicRMW:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.AtomicRMWGetPtr(expr))
		self.Visit(binaryen.AtomicRMWGetValue(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitAtomicRMW(expr)

	case module.ExpressionIdAtomicCmpxchg:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.AtomicCmpxchgGetPtr(expr))
		self.Visit(binaryen.AtomicCmpxchgGetExpected(expr))
		self.Visit(binaryen.AtomicCmpxchgGetReplacement(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitAtomicCmpxchg(expr)

	case module.ExpressionIdAtomicWait:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.AtomicWaitGetPtr(expr))
		self.Visit(binaryen.AtomicWaitGetExpected(expr))
		self.Visit(binaryen.AtomicWaitGetTimeout(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitAtomicWait(expr)

	case module.ExpressionIdAtomicNotify:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.AtomicNotifyGetPtr(expr))
		self.Visit(binaryen.AtomicNotifyGetNotifyCount(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitAtomicNotify(expr)

	case module.ExpressionIdAtomicFence:
		self.VisitAtomicFence(expr)

	case module.ExpressionIdSIMDExtract:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.SIMDExtractGetVec(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitSIMDExtract(expr)

	case module.ExpressionIdSIMDReplace:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.SIMDReplaceGetVec(expr))
		self.Visit(binaryen.SIMDReplaceGetValue(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitSIMDReplace(expr)

	case module.ExpressionIdSIMDShuffle:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.SIMDShuffleGetLeft(expr))
		self.Visit(binaryen.SIMDShuffleGetRight(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitSIMDShuffle(expr)

	case module.ExpressionIdSIMDTernary:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.SIMDTernaryGetA(expr))
		self.Visit(binaryen.SIMDTernaryGetB(expr))
		self.Visit(binaryen.SIMDTernaryGetC(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitSIMDTernary(expr)

	case module.ExpressionIdSIMDShift:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.SIMDShiftGetVec(expr))
		self.Visit(binaryen.SIMDShiftGetShift(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitSIMDShift(expr)

	case module.ExpressionIdSIMDLoad:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.SIMDLoadGetPtr(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitSIMDLoad(expr)

	case module.ExpressionIdSIMDLoadStoreLane:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.SIMDLoadStoreLaneGetPtr(expr))
		self.Visit(binaryen.SIMDLoadStoreLaneGetVec(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitSIMDLoadStoreLane(expr)

	case module.ExpressionIdMemoryInit:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.MemoryInitGetDest(expr))
		self.Visit(binaryen.MemoryInitGetOffset(expr))
		self.Visit(binaryen.MemoryInitGetSize(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitMemoryInit(expr)

	case module.ExpressionIdDataDrop:
		self.VisitDataDrop(expr)

	case module.ExpressionIdMemoryCopy:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.MemoryCopyGetDest(expr))
		self.Visit(binaryen.MemoryCopyGetSource(expr))
		self.Visit(binaryen.MemoryCopyGetSize(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitMemoryCopy(expr)

	case module.ExpressionIdMemoryFill:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.MemoryFillGetDest(expr))
		self.Visit(binaryen.MemoryFillGetValue(expr))
		self.Visit(binaryen.MemoryFillGetSize(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitMemoryFill(expr)

	case module.ExpressionIdPop:
		self.VisitPop(expr)

	case module.ExpressionIdRefNull:
		self.VisitRefNull(expr)

	case module.ExpressionIdRefIsNull:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.RefIsNullGetValue(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitRefIsNull(expr)

	case module.ExpressionIdRefFunc:
		v.stack = append(v.stack, expr)
		self.VisitName(binaryen.RefFuncGetFunc(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitRefFunc(expr)

	case module.ExpressionIdRefEq:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.RefEqGetLeft(expr))
		self.Visit(binaryen.RefEqGetRight(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitRefEq(expr)

	case module.ExpressionIdTry:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.TryGetBody(expr))
		numCatchBodies := binaryen.TryGetNumCatchBodies(expr)
		for i := module.Index(0); i < numCatchBodies; i++ {
			self.Visit(binaryen.TryGetCatchBodyAt(expr, i))
		}
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitTry(expr)

	case module.ExpressionIdThrow:
		v.stack = append(v.stack, expr)
		self.VisitTag(binaryen.ThrowGetTag(expr))
		numOperands := binaryen.ThrowGetNumOperands(expr)
		for i := module.Index(0); i < numOperands; i++ {
			self.Visit(binaryen.ThrowGetOperandAt(expr, i))
		}
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitThrow(expr)

	case module.ExpressionIdRethrow:
		self.VisitRethrow(expr)

	case module.ExpressionIdTupleMake:
		numOperands := binaryen.TupleMakeGetNumOperands(expr)
		if numOperands > 0 {
			v.stack = append(v.stack, expr)
			for i := module.Index(0); i < numOperands; i++ {
				self.Visit(binaryen.TupleMakeGetOperandAt(expr, i))
			}
			v.stack = v.stack[:len(v.stack)-1]
		}
		self.VisitTupleMake(expr)

	case module.ExpressionIdTupleExtract:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.TupleExtractGetTuple(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitTupleExtract(expr)

	case module.ExpressionIdRefI31:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.RefI31GetValue(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitRefI31(expr)

	case module.ExpressionIdI31Get:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.I31GetGetI31(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitI31Get(expr)

	case module.ExpressionIdCallRef:
		v.stack = append(v.stack, expr)
		numOperands := binaryen.CallRefGetNumOperands(expr)
		if numOperands > 0 {
			for i := module.Index(0); i < numOperands; i++ {
				self.Visit(binaryen.CallRefGetOperandAt(expr, i))
			}
		}
		self.Visit(binaryen.CallRefGetTarget(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitCallRef(expr)

	case module.ExpressionIdRefTest:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.RefTestGetRef(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitRefTest(expr)

	case module.ExpressionIdRefCast:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.RefCastGetRef(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitRefCast(expr)

	case module.ExpressionIdBrOn:
		v.stack = append(v.stack, expr)
		self.VisitLabel(binaryen.BrOnGetName(expr))
		self.Visit(binaryen.BrOnGetRef(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitBrOn(expr)

	case module.ExpressionIdStructNew:
		numOperands := binaryen.StructNewGetNumOperands(expr)
		if numOperands > 0 {
			v.stack = append(v.stack, expr)
			for i := module.Index(0); i < numOperands; i++ {
				self.Visit(binaryen.StructNewGetOperandAt(expr, i))
			}
			v.stack = v.stack[:len(v.stack)-1]
		}
		self.VisitStructNew(expr)

	case module.ExpressionIdStructGet:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.StructGetGetRef(expr))
		self.VisitIndex(binaryen.StructGetGetIndex(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitStructGet(expr)

	case module.ExpressionIdStructSet:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.StructSetGetRef(expr))
		self.VisitIndex(binaryen.StructSetGetIndex(expr))
		self.Visit(binaryen.StructSetGetValue(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitStructSet(expr)

	case module.ExpressionIdArrayNew:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.ArrayNewGetSize(expr))
		init := binaryen.ArrayNewGetInit(expr)
		if init != 0 {
			self.Visit(init)
		}
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitArrayNew(expr)

	case module.ExpressionIdArrayNewFixed:
		numValues := binaryen.ArrayNewFixedGetNumValues(expr)
		if numValues > 0 {
			v.stack = append(v.stack, expr)
			for i := module.Index(0); i < numValues; i++ {
				self.Visit(binaryen.ArrayNewFixedGetValueAt(expr, i))
			}
			v.stack = v.stack[:len(v.stack)-1]
		}
		self.VisitArrayNewFixed(expr)

	case module.ExpressionIdArrayGet:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.ArrayGetGetRef(expr))
		self.Visit(binaryen.ArrayGetGetIndex(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitArrayGet(expr)

	case module.ExpressionIdArraySet:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.ArraySetGetRef(expr))
		self.Visit(binaryen.ArraySetGetIndex(expr))
		self.Visit(binaryen.ArraySetGetValue(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitArraySet(expr)

	case module.ExpressionIdArrayLen:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.ArrayLenGetRef(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitArrayLen(expr)

	case module.ExpressionIdArrayCopy:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.ArrayCopyGetDestRef(expr))
		self.Visit(binaryen.ArrayCopyGetDestIndex(expr))
		self.Visit(binaryen.ArrayCopyGetSrcRef(expr))
		self.Visit(binaryen.ArrayCopyGetSrcIndex(expr))
		self.Visit(binaryen.ArrayCopyGetLength(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitArrayCopy(expr)

	case module.ExpressionIdRefAs:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.RefAsGetValue(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitRefAs(expr)

	case module.ExpressionIdStringNew:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.StringNewGetRef(expr))
		start := binaryen.StringNewGetStart(expr) // GC only
		if start != 0 {
			self.Visit(start)
		}
		end := binaryen.StringNewGetEnd(expr) // GC only
		if end != 0 {
			self.Visit(end)
		}
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitStringNew(expr)

	case module.ExpressionIdStringConst:
		v.stack = append(v.stack, expr)
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitStringConst(expr)

	case module.ExpressionIdStringMeasure:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.StringMeasureGetRef(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitStringMeasure(expr)

	case module.ExpressionIdStringEncode:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.StringEncodeGetStr(expr))
		self.Visit(binaryen.StringEncodeGetArray(expr))
		start := binaryen.StringEncodeGetStart(expr) // GC only
		if start != 0 {
			self.Visit(start)
		}
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitStringEncode(expr)

	case module.ExpressionIdStringConcat:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.StringConcatGetLeft(expr))
		self.Visit(binaryen.StringConcatGetRight(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitStringConcat(expr)

	case module.ExpressionIdStringEq:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.StringEqGetLeft(expr))
		self.Visit(binaryen.StringEqGetRight(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitStringEq(expr)

	case module.ExpressionIdStringWTF16Get:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.StringWTF16GetGetRef(expr))
		self.Visit(binaryen.StringWTF16GetGetPos(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitStringWTF16Get(expr)

	case module.ExpressionIdStringSliceWTF:
		v.stack = append(v.stack, expr)
		self.Visit(binaryen.StringSliceWTFGetRef(expr))
		self.Visit(binaryen.StringSliceWTFGetStart(expr))
		self.Visit(binaryen.StringSliceWTFGetEnd(expr))
		v.stack = v.stack[:len(v.stack)-1]
		self.VisitStringSliceWTF(expr)

	default:
		panic(fmt.Sprintf("unexpected expression kind: %d", binaryen.ExpressionGetId(expr)))
	}

	v.currentExpression = previousExpression
}
