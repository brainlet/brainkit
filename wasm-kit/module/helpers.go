// Ported from: src/module.ts (free functions and utility types, lines ~2988-3694)
package module

import (
	"fmt"
	"math"

	"github.com/brainlet/brainkit/wasm-kit/pkg/binaryen"
)

// ---------------------------------------------------------------------------
// Type aliases for convenience. These map to the corresponding binaryen types.
// ---------------------------------------------------------------------------

type TypeRef = binaryen.Type
type ExpressionRef = binaryen.ExpressionRef
type FunctionRef = binaryen.FunctionRef
type GlobalRef = binaryen.GlobalRef
type TagRef = binaryen.TagRef
// ExpressionID, Op, Index, and other auxiliary type aliases are declared in
// constants.go to avoid redeclaration.

// ---------------------------------------------------------------------------
// Side effects constants
// ---------------------------------------------------------------------------

const (
	SideEffectNone             uint32 = 0
	SideEffectBranches         uint32 = 1
	SideEffectCalls            uint32 = 2
	SideEffectReadsLocal       uint32 = 4
	SideEffectWritesLocal      uint32 = 8
	SideEffectReadsGlobal      uint32 = 16
	SideEffectWritesGlobal     uint32 = 32
	SideEffectReadsMemory      uint32 = 64
	SideEffectWritesMemory     uint32 = 128
	SideEffectReadsTable       uint32 = 256
	SideEffectWritesTable      uint32 = 512
	SideEffectImplicitTrap     uint32 = 1024
	SideEffectIsAtomic         uint32 = 2048
	SideEffectThrows           uint32 = 4096
	SideEffectDanglingPop      uint32 = 8192
	SideEffectTrapsNeverHappen uint32 = 16384
	SideEffectAny              uint32 = 32767
)

// ---------------------------------------------------------------------------
// Type helpers
// ---------------------------------------------------------------------------

// CreateType creates a tuple type from multiple type refs.
// Returns TypeNone if nil/empty, the single type if length 1.
func CreateType(types []TypeRef) TypeRef {
	if len(types) == 0 {
		return binaryen.TypeNone()
	}
	if len(types) == 1 {
		return types[0]
	}
	return binaryen.TypeCreate(types)
}

// ExpandType expands a tuple type into its component types.
func ExpandType(t TypeRef) []TypeRef {
	return binaryen.TypeExpand(t)
}

// IsNullableType returns whether a reference type is nullable.
func IsNullableType(t TypeRef) bool {
	return binaryen.TypeIsNullable(t)
}

// ---------------------------------------------------------------------------
// Expression (generic)
// ---------------------------------------------------------------------------

// GetExpressionId returns the expression kind ID.
func GetExpressionId(expr ExpressionRef) ExpressionID {
	return binaryen.ExpressionGetId(expr)
}

// GetExpressionType returns the type of an expression.
func GetExpressionType(expr ExpressionRef) TypeRef {
	return binaryen.ExpressionGetType(expr)
}

// ---------------------------------------------------------------------------
// Const value accessors
// ---------------------------------------------------------------------------

// GetConstValueI32 returns the i32 value of a const expression.
func GetConstValueI32(expr ExpressionRef) int32 {
	return binaryen.ConstGetValueI32(expr)
}

// GetConstValueI64Low returns the low 32 bits of an i64 const.
func GetConstValueI64Low(expr ExpressionRef) int32 {
	return binaryen.ConstGetValueI64Low(expr)
}

// GetConstValueI64High returns the high 32 bits of an i64 const.
func GetConstValueI64High(expr ExpressionRef) int32 {
	return binaryen.ConstGetValueI64High(expr)
}

// GetConstValueI64 returns the full i64 value of a const expression.
func GetConstValueI64(expr ExpressionRef) int64 {
	return binaryen.ConstGetValueI64(expr)
}

// GetConstValueInteger returns the integer value of a const expression,
// handling both i32 and i64 (wasm64) cases. For i32, it zero-extends to i64.
// For i64, it combines the low and high 32-bit halves.
func GetConstValueInteger(expr ExpressionRef, isWasm64 bool) int64 {
	if isWasm64 {
		lo := int64(uint32(GetConstValueI64Low(expr)))
		hi := int64(GetConstValueI64High(expr))
		return (hi << 32) | lo
	}
	return int64(uint32(GetConstValueI32(expr)))
}

// GetConstValueF32 returns the f32 value of a const expression.
func GetConstValueF32(expr ExpressionRef) float32 {
	return binaryen.ConstGetValueF32(expr)
}

// GetConstValueF64 returns the f64 value of a const expression.
func GetConstValueF64(expr ExpressionRef) float64 {
	return binaryen.ConstGetValueF64(expr)
}

// GetConstValueV128 returns the v128 value of a const expression as 16 bytes.
func GetConstValueV128(expr ExpressionRef) [16]byte {
	var out [16]byte
	binaryen.ConstGetValueV128(expr, &out)
	return out
}

// ---------------------------------------------------------------------------
// Const predicates
// ---------------------------------------------------------------------------

// IsConstZero checks if expr is a const with value 0 for i32/i64/f32/f64.
func IsConstZero(expr ExpressionRef) bool {
	if GetExpressionId(expr) != binaryen.ConstId() {
		return false
	}
	t := GetExpressionType(expr)
	switch t {
	case binaryen.TypeInt32():
		return GetConstValueI32(expr) == 0
	case binaryen.TypeInt64():
		return (GetConstValueI64Low(expr) | GetConstValueI64High(expr)) == 0
	case binaryen.TypeFloat32():
		return GetConstValueF32(expr) == 0
	case binaryen.TypeFloat64():
		return GetConstValueF64(expr) == 0
	}
	return false
}

// IsConstNonZero checks if expr is a const with a non-zero value.
func IsConstNonZero(expr ExpressionRef) bool {
	if GetExpressionId(expr) != binaryen.ConstId() {
		return false
	}
	t := GetExpressionType(expr)
	switch t {
	case binaryen.TypeInt32():
		return GetConstValueI32(expr) != 0
	case binaryen.TypeInt64():
		return (GetConstValueI64Low(expr) | GetConstValueI64High(expr)) != 0
	case binaryen.TypeFloat32():
		return GetConstValueF32(expr) != 0
	case binaryen.TypeFloat64():
		return GetConstValueF64(expr) != 0
	}
	return false
}

// IsConstNegZero checks if expr is a -0.0 float const.
func IsConstNegZero(expr ExpressionRef) bool {
	if GetExpressionId(expr) != binaryen.ConstId() {
		return false
	}
	t := GetExpressionType(expr)
	switch t {
	case binaryen.TypeFloat32():
		d := GetConstValueF32(expr)
		return d == 0 && math.Float32bits(d)&(1<<31) != 0
	case binaryen.TypeFloat64():
		d := GetConstValueF64(expr)
		return d == 0 && math.Float64bits(d)&(1<<63) != 0
	}
	return false
}

// IsConstNaN checks if expr is a NaN float const.
func IsConstNaN(expr ExpressionRef) bool {
	if GetExpressionId(expr) != binaryen.ConstId() {
		return false
	}
	t := GetExpressionType(expr)
	switch t {
	case binaryen.TypeFloat32():
		return math.IsNaN(float64(GetConstValueF32(expr)))
	case binaryen.TypeFloat64():
		return math.IsNaN(GetConstValueF64(expr))
	}
	return false
}

// IsConstExpressionNaN checks if a float expression evaluates to NaN.
// For Const expressions it checks the value directly.
// For GlobalGet expressions it evaluates using the expression runner.
func IsConstExpressionNaN(mod *Module, expr ExpressionRef) bool {
	id := GetExpressionId(expr)
	t := GetExpressionType(expr)
	if t != binaryen.TypeFloat32() && t != binaryen.TypeFloat64() {
		return false
	}
	if id == binaryen.ConstId() {
		if t == binaryen.TypeFloat32() {
			return math.IsNaN(float64(GetConstValueF32(expr)))
		}
		return math.IsNaN(GetConstValueF64(expr))
	}
	if id == binaryen.GlobalGetId() {
		precomp := mod.RunExpression(expr, binaryen.ExpressionRunnerFlagsDefault(), 8, 1)
		if precomp != 0 {
			if t == binaryen.TypeFloat32() {
				return math.IsNaN(float64(GetConstValueF32(precomp)))
			}
			return math.IsNaN(GetConstValueF64(precomp))
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// LocalGet
// ---------------------------------------------------------------------------

// GetLocalGetIndex returns the local index of a local.get expression.
func GetLocalGetIndex(expr ExpressionRef) Index {
	return binaryen.LocalGetGetIndex(expr)
}

// ---------------------------------------------------------------------------
// LocalSet
// ---------------------------------------------------------------------------

// GetLocalSetIndex returns the local index of a local.set expression.
func GetLocalSetIndex(expr ExpressionRef) Index {
	return binaryen.LocalSetGetIndex(expr)
}

// GetLocalSetValue returns the value of a local.set expression.
func GetLocalSetValue(expr ExpressionRef) ExpressionRef {
	return binaryen.LocalSetGetValue(expr)
}

// IsLocalTee returns whether a local.set expression is a tee (returns the set value).
func IsLocalTee(expr ExpressionRef) bool {
	return binaryen.LocalSetIsTee(expr)
}

// ---------------------------------------------------------------------------
// GlobalGet
// ---------------------------------------------------------------------------

// GetGlobalGetName returns the global name of a global.get expression.
func GetGlobalGetName(expr ExpressionRef) string {
	return binaryen.GlobalGetGetName(expr)
}

// ---------------------------------------------------------------------------
// Binary
// ---------------------------------------------------------------------------

// GetBinaryOp returns the operator of a binary expression.
func GetBinaryOp(expr ExpressionRef) Op {
	return binaryen.BinaryGetOp(expr)
}

// GetBinaryLeft returns the left operand of a binary expression.
func GetBinaryLeft(expr ExpressionRef) ExpressionRef {
	return binaryen.BinaryGetLeft(expr)
}

// GetBinaryRight returns the right operand of a binary expression.
func GetBinaryRight(expr ExpressionRef) ExpressionRef {
	return binaryen.BinaryGetRight(expr)
}

// ---------------------------------------------------------------------------
// Unary
// ---------------------------------------------------------------------------

// GetUnaryOp returns the operator of a unary expression.
func GetUnaryOp(expr ExpressionRef) Op {
	return binaryen.UnaryGetOp(expr)
}

// GetUnaryValue returns the operand of a unary expression.
func GetUnaryValue(expr ExpressionRef) ExpressionRef {
	return binaryen.UnaryGetValue(expr)
}

// ---------------------------------------------------------------------------
// Load
// ---------------------------------------------------------------------------

// GetLoadBytes returns the number of bytes loaded.
func GetLoadBytes(expr ExpressionRef) uint32 {
	return binaryen.LoadGetBytes(expr)
}

// GetLoadOffset returns the memory offset of a load expression.
func GetLoadOffset(expr ExpressionRef) uint32 {
	return binaryen.LoadGetOffset(expr)
}

// GetLoadPtr returns the pointer expression of a load.
func GetLoadPtr(expr ExpressionRef) ExpressionRef {
	return binaryen.LoadGetPtr(expr)
}

// IsLoadSigned returns whether the load is sign-extending.
func IsLoadSigned(expr ExpressionRef) bool {
	return binaryen.LoadIsSigned(expr)
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

// GetStoreBytes returns the number of bytes stored.
func GetStoreBytes(expr ExpressionRef) uint32 {
	return binaryen.StoreGetBytes(expr)
}

// GetStoreOffset returns the memory offset of a store expression.
func GetStoreOffset(expr ExpressionRef) uint32 {
	return binaryen.StoreGetOffset(expr)
}

// GetStorePtr returns the pointer expression of a store.
func GetStorePtr(expr ExpressionRef) ExpressionRef {
	return binaryen.StoreGetPtr(expr)
}

// GetStoreValue returns the value expression of a store.
func GetStoreValue(expr ExpressionRef) ExpressionRef {
	return binaryen.StoreGetValue(expr)
}

// ---------------------------------------------------------------------------
// Block
// ---------------------------------------------------------------------------

// GetBlockName returns the label of a block expression.
func GetBlockName(expr ExpressionRef) string {
	return binaryen.BlockGetName(expr)
}

// GetBlockChildCount returns the number of children in a block.
func GetBlockChildCount(expr ExpressionRef) Index {
	return binaryen.BlockGetNumChildren(expr)
}

// GetBlockChildAt returns the child expression at the given index in a block.
func GetBlockChildAt(expr ExpressionRef, index Index) ExpressionRef {
	return binaryen.BlockGetChildAt(expr, index)
}

// ---------------------------------------------------------------------------
// If
// ---------------------------------------------------------------------------

// GetIfCondition returns the condition expression of an if.
func GetIfCondition(expr ExpressionRef) ExpressionRef {
	return binaryen.IfGetCondition(expr)
}

// GetIfTrue returns the true branch of an if expression.
func GetIfTrue(expr ExpressionRef) ExpressionRef {
	return binaryen.IfGetIfTrue(expr)
}

// GetIfFalse returns the false branch of an if expression.
func GetIfFalse(expr ExpressionRef) ExpressionRef {
	return binaryen.IfGetIfFalse(expr)
}

// ---------------------------------------------------------------------------
// Loop
// ---------------------------------------------------------------------------

// GetLoopName returns the label of a loop expression.
func GetLoopName(expr ExpressionRef) string {
	return binaryen.LoopGetName(expr)
}

// GetLoopBody returns the body expression of a loop.
func GetLoopBody(expr ExpressionRef) ExpressionRef {
	return binaryen.LoopGetBody(expr)
}

// ---------------------------------------------------------------------------
// Break
// ---------------------------------------------------------------------------

// GetBreakName returns the target label of a break expression.
func GetBreakName(expr ExpressionRef) string {
	return binaryen.BreakGetName(expr)
}

// GetBreakCondition returns the condition of a conditional break, or 0 if unconditional.
func GetBreakCondition(expr ExpressionRef) ExpressionRef {
	return binaryen.BreakGetCondition(expr)
}

// ---------------------------------------------------------------------------
// Select
// ---------------------------------------------------------------------------

// GetSelectThen returns the "then" (ifTrue) value of a select expression.
func GetSelectThen(expr ExpressionRef) ExpressionRef {
	return binaryen.SelectGetIfTrue(expr)
}

// GetSelectElse returns the "else" (ifFalse) value of a select expression.
func GetSelectElse(expr ExpressionRef) ExpressionRef {
	return binaryen.SelectGetIfFalse(expr)
}

// GetSelectCondition returns the condition of a select expression.
func GetSelectCondition(expr ExpressionRef) ExpressionRef {
	return binaryen.SelectGetCondition(expr)
}

// ---------------------------------------------------------------------------
// Drop
// ---------------------------------------------------------------------------

// GetDropValue returns the value expression being dropped.
func GetDropValue(expr ExpressionRef) ExpressionRef {
	return binaryen.DropGetValue(expr)
}

// ---------------------------------------------------------------------------
// Return
// ---------------------------------------------------------------------------

// GetReturnValue returns the value expression of a return, or 0 if void.
func GetReturnValue(expr ExpressionRef) ExpressionRef {
	return binaryen.ReturnGetValue(expr)
}

// ---------------------------------------------------------------------------
// Call
// ---------------------------------------------------------------------------

// GetCallTarget returns the function name targeted by a call expression.
func GetCallTarget(expr ExpressionRef) string {
	return binaryen.CallGetTarget(expr)
}

// GetCallOperandCount returns the number of operands in a call expression.
func GetCallOperandCount(expr ExpressionRef) Index {
	return binaryen.CallGetNumOperands(expr)
}

// GetCallOperandAt returns the operand at the given index in a call expression.
func GetCallOperandAt(expr ExpressionRef, index Index) ExpressionRef {
	return binaryen.CallGetOperandAt(expr, index)
}

// ---------------------------------------------------------------------------
// MemoryGrow
// ---------------------------------------------------------------------------

// GetMemoryGrowDelta returns the delta expression of a memory.grow.
func GetMemoryGrowDelta(expr ExpressionRef) ExpressionRef {
	return binaryen.MemoryGrowGetDelta(expr)
}

// ---------------------------------------------------------------------------
// Function accessors
// ---------------------------------------------------------------------------

// GetFunctionBody returns the body expression of a function.
func GetFunctionBody(fn FunctionRef) ExpressionRef {
	return binaryen.FunctionGetBody(fn)
}

// GetFunctionName returns the name of a function.
func GetFunctionName(fn FunctionRef) string {
	return binaryen.FunctionGetName(fn)
}

// GetFunctionParams returns the parameter type of a function.
func GetFunctionParams(fn FunctionRef) TypeRef {
	return binaryen.FunctionGetParams(fn)
}

// GetFunctionResults returns the result type of a function.
func GetFunctionResults(fn FunctionRef) TypeRef {
	return binaryen.FunctionGetResults(fn)
}

// GetFunctionVars returns the local variable types of a function.
func GetFunctionVars(fn FunctionRef) []TypeRef {
	n := binaryen.FunctionGetNumVars(fn)
	if n == 0 {
		return nil
	}
	vars := make([]TypeRef, n)
	for i := uint32(0); i < n; i++ {
		vars[i] = binaryen.FunctionGetVar(fn, i)
	}
	return vars
}

// ---------------------------------------------------------------------------
// Global accessors
// ---------------------------------------------------------------------------

// GetGlobalName returns the name of a global.
func GetGlobalName(g GlobalRef) string {
	return binaryen.GlobalGetName(g)
}

// GetGlobalType returns the type of a global.
func GetGlobalType(g GlobalRef) TypeRef {
	return binaryen.GlobalGetType(g)
}

// IsGlobalMutable returns whether a global is mutable.
func IsGlobalMutable(g GlobalRef) bool {
	return binaryen.GlobalIsMutable(g)
}

// GetGlobalInit returns the initializer expression of a global.
func GetGlobalInit(g GlobalRef) ExpressionRef {
	return binaryen.GlobalGetInitExpr(g)
}

// ---------------------------------------------------------------------------
// Tag accessors
// ---------------------------------------------------------------------------

// GetTagName returns the name of a tag.
func GetTagName(t TagRef) string {
	return binaryen.TagGetName(t)
}

// GetTagParams returns the parameter type of a tag.
func GetTagParams(t TagRef) TypeRef {
	return binaryen.TagGetParams(t)
}

// GetTagResults returns the result type of a tag.
func GetTagResults(t TagRef) TypeRef {
	return binaryen.TagGetResults(t)
}

// ---------------------------------------------------------------------------
// Side effects
// ---------------------------------------------------------------------------

// GetSideEffects returns the side effect flags for an expression.
func GetSideEffects(expr ExpressionRef, mod *binaryen.Module) uint32 {
	return binaryen.GetSideEffects(expr, mod)
}

// MustPreserveSideEffects returns whether the expression has any side effects
// that must be preserved. Read-only local and global accesses are excluded
// since they do not produce observable effects.
func MustPreserveSideEffects(expr ExpressionRef, mod *binaryen.Module) bool {
	effects := GetSideEffects(expr, mod)
	return (effects & ^(SideEffectReadsLocal | SideEffectReadsGlobal)) != SideEffectNone
}

// ---------------------------------------------------------------------------
// SwitchBuilder
// ---------------------------------------------------------------------------

// SwitchBuilder constructs switch-like branching using a sequence of br_if
// instructions. Binaryen understands sequences of br_if and knows how to
// convert them into a br_table if the switched-over values are dense enough,
// or a size-efficient sequence of if-else otherwise, depending on the
// optimization level.
type SwitchBuilder struct {
	mod          *binaryen.Module
	condition    ExpressionRef
	values       []int32
	indexes      []int32
	cases        [][]ExpressionRef
	defaultIndex int32
}

// NewSwitchBuilder creates a new builder using the specified i32 condition.
func NewSwitchBuilder(mod *binaryen.Module, condition ExpressionRef) *SwitchBuilder {
	return &SwitchBuilder{
		mod:          mod,
		condition:    condition,
		defaultIndex: -1,
	}
}

// AddCase links a case value to the specified branch code.
func (sb *SwitchBuilder) AddCase(value int32, code []ExpressionRef) {
	sb.values = append(sb.values, value)
	sb.indexes = append(sb.indexes, sb.addCode(code))
}

// AddOrReplaceCase links a case to the specified branch. If the value already
// exists, the old case is replaced.
func (sb *SwitchBuilder) AddOrReplaceCase(value int32, code []ExpressionRef) {
	codeIndex := sb.addCode(code)
	for i, v := range sb.values {
		if v == value {
			sb.indexes[i] = codeIndex
			return
		}
	}
	sb.values = append(sb.values, value)
	sb.indexes = append(sb.indexes, codeIndex)
}

// addCode registers a case body and returns its index. If the same slice
// reference was already added, returns the existing index.
func (sb *SwitchBuilder) addCode(code []ExpressionRef) int32 {
	// Note: In Go we cannot compare slices by identity like TS does with
	// indexOf. Each AddCase call creates a distinct entry.
	idx := int32(len(sb.cases))
	sb.cases = append(sb.cases, code)
	return idx
}

// AddDefault links the default branch.
func (sb *SwitchBuilder) AddDefault(code []ExpressionRef) {
	if sb.defaultIndex != -1 {
		panic("SwitchBuilder: default already set")
	}
	sb.defaultIndex = int32(len(sb.cases))
	sb.cases = append(sb.cases, code)
}

// Render renders the switch to a block expression. localIndex is a free i32
// local used for holding the condition value. labelPostfix can be used to
// disambiguate labels when multiple switches exist in the same scope.
func (sb *SwitchBuilder) Render(localIndex int32, labelPostfix string) ExpressionRef {
	mod := sb.mod
	cases := sb.cases
	numCases := len(cases)
	if numCases == 0 {
		return mod.Drop(sb.condition)
	}

	values := sb.values
	numValues := len(values)
	indexes := sb.indexes

	// Build label names for each case
	labels := make([]string, numCases)
	for i := 0; i < numCases; i++ {
		labels[i] = fmt.Sprintf("case%d%s", i, labelPostfix)
	}

	// Build the entry block contents:
	// [0] = local.set localIndex, condition
	// [1..numValues] = br_if caseN, (i32.eq (local.get localIndex) (i32.const value))
	// [numValues+1] = br default (unconditional)
	entry := make([]ExpressionRef, 1+numValues+1)
	entry[0] = mod.LocalSet(uint32(localIndex), sb.condition)

	for i := 0; i < numValues; i++ {
		index := indexes[i]
		entry[1+i] = mod.Break(
			labels[index],
			mod.Binary(
				binaryen.EqInt32(),
				mod.LocalGet(uint32(localIndex), binaryen.TypeInt32()),
				mod.ConstInt32(values[i]),
			),
			0,
		)
	}

	// Unconditional branch to default or the default case label
	defaultLabel := fmt.Sprintf("default%s", labelPostfix)
	if sb.defaultIndex >= 0 {
		entry[1+numValues] = mod.Break(labels[sb.defaultIndex], 0, 0)
	} else {
		entry[1+numValues] = mod.Break(defaultLabel, 0, 0)
	}

	// Wrap everything in nested blocks, one per case
	current := mod.Block(labels[0], entry, binaryen.TypeAuto())
	for i := 1; i < numCases; i++ {
		block := make([]ExpressionRef, 0, 1+len(cases[i-1]))
		block = append(block, current)
		block = append(block, cases[i-1]...)
		current = mod.Block(labels[i], block, binaryen.TypeAuto())
	}

	// The outermost block wraps the last case
	lastCase := make([]ExpressionRef, 0, 1+len(cases[numCases-1]))
	lastCase = append(lastCase, current)
	lastCase = append(lastCase, cases[numCases-1]...)

	if sb.defaultIndex >= 0 {
		// All cases are labeled, outer block has no name
		return mod.Block("", lastCase, binaryen.TypeAuto())
	}
	return mod.Block(defaultLabel, lastCase, binaryen.TypeAuto())
}

// ---------------------------------------------------------------------------
// BinaryModule is declared in module.go.
