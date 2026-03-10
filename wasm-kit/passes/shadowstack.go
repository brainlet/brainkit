// Ported from: assemblyscript/src/passes/shadowstack.ts (ShadowStackPass + InstrumentReturns, 696 lines)
//
// Instruments a module with a shadow stack for precise GC. Marks function
// arguments and local assignments that go through __tostack with stores to a
// shadow stack of managed values only.
package passes

import (
	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/compiler"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/pkg/binaryen"
	"github.com/brainlet/brainkit/wasm-kit/program"
)

// Type aliases matching the TS source.
type LocalIndex = module.Index
type SlotIndex = module.Index
type SlotMap = map[LocalIndex]SlotIndex
type TempMap = map[module.TypeRef]LocalIndex

// matchPattern attempts to match the `__tostack(value)` pattern.
// Returns `value` if a match, otherwise 0.
func matchPattern(mod *module.Module, expr module.ExpressionRef) module.ExpressionRef {
	isFound := false
	for binaryen.ExpressionGetId(expr) == module.ExpressionIdCall &&
		binaryen.CallGetTarget(expr) == common.BuiltinNameTostack {
		if binaryen.CallGetNumOperands(expr) != 1 {
			panic("expected 1 operand for __tostack")
		}
		expr = binaryen.CallGetOperandAt(expr, 0)
		isFound = true
	}
	if !isFound {
		return 0
	}
	return expr
}

// needsSlot tests whether a value matched by matchPattern needs a slot.
func needsSlot(value module.ExpressionRef) bool {
	switch binaryen.ExpressionGetId(value) {
	// no need to stack null pointers
	case module.ExpressionIdConst:
		return !module.IsConstZero(value)
	// note: can't omit a slot when assigning from another local since the other
	// local might have shorter lifetime and become reassigned, say in a loop,
	// then no longer holding on to the previous value in its stack slot.
	}
	return true
}

// ShadowStackPass instruments a module with a shadow stack for precise GC.
type ShadowStackPass struct {
	Pass

	// Stack frame slots, per function.
	slotMaps map[module.FunctionRef]SlotMap
	// Temporary locals, per function.
	tempMaps map[module.FunctionRef]TempMap
	// Exports (with managed operands) map.
	exportMap map[string][]int32
	// Compiler reference.
	Compiler *compiler.Compiler

	// hasStackCheckFunction tracks whether ~stack_check has been emitted.
	hasStackCheckFunction bool
	// Slot offset accounting for nested calls.
	callSlotOffset int32
	// Slot offset stack in nested calls.
	callSlotStack []int32
}

// NewShadowStackPass constructs a new ShadowStackPass for the given compiler.
func NewShadowStackPass(c *compiler.Compiler) *ShadowStackPass {
	p := &ShadowStackPass{
		Pass:          NewPass(c.Module()),
		slotMaps:      make(map[module.FunctionRef]SlotMap),
		tempMaps:      make(map[module.FunctionRef]TempMap),
		exportMap:     make(map[string][]int32),
		Compiler:      c,
		callSlotStack: make([]int32, 0),
	}
	p.InitVisitor(p)
	return p
}

// options returns the compiler options.
func (p *ShadowStackPass) options() *program.Options {
	return p.Compiler.Options()
}

// ptrType returns the target pointer type.
func (p *ShadowStackPass) ptrType() module.TypeRef {
	return module.TypeRef(p.options().SizeTypeRef())
}

// ptrSize returns the target pointer size.
func (p *ShadowStackPass) ptrSize() int32 {
	if p.ptrType() == module.TypeRefI64 {
		return 8
	}
	return 4
}

// ptrBinaryAdd returns the target pointer addition operation.
func (p *ShadowStackPass) ptrBinaryAdd() module.Op {
	if p.ptrType() == module.TypeRefI64 {
		return module.BinaryOpAddI64
	}
	return module.BinaryOpAddI32
}

// ptrBinarySub returns the target pointer subtraction operation.
func (p *ShadowStackPass) ptrBinarySub() module.Op {
	if p.ptrType() == module.TypeRefI64 {
		return module.BinaryOpSubI64
	}
	return module.BinaryOpSubI32
}

// ptrConst gets a constant with the specified value of the target pointer type.
func (p *ShadowStackPass) ptrConst(value int32) module.ExpressionRef {
	if p.ptrType() == module.TypeRefI64 {
		return p.Module.I64(int64(value))
	}
	return p.Module.I32(value)
}

// NoteSlot notes the presence of a slot for the specified (imaginary) local,
// returning the slot index.
func (p *ShadowStackPass) NoteSlot(fn module.FunctionRef, localIndex module.Index) int32 {
	slotMap, ok := p.slotMaps[fn]
	if ok {
		if slotIdx, exists := slotMap[localIndex]; exists {
			return int32(slotIdx)
		}
	} else {
		slotMap = make(SlotMap)
		p.slotMaps[fn] = slotMap
	}
	slotIndex := module.Index(len(slotMap))
	slotMap[localIndex] = slotIndex
	return int32(slotIndex)
}

// NoteExport notes the presence of an exported function taking managed operands.
func (p *ShadowStackPass) NoteExport(name string, managedOperandIndices []int32) {
	if len(managedOperandIndices) == 0 {
		return
	}
	p.exportMap[name] = managedOperandIndices
}

// getSharedTemp gets a shared temporary local of the given type in the specified function.
func (p *ShadowStackPass) getSharedTemp(fn module.FunctionRef, typ module.TypeRef) module.Index {
	tempMap, ok := p.tempMaps[fn]
	if ok {
		if localIdx, exists := tempMap[typ]; exists {
			return localIdx
		}
	} else {
		tempMap = make(TempMap)
		p.tempMaps[fn] = tempMap
	}
	numLocals := binaryen.FunctionGetNumLocals(fn)
	localIndex := numLocals + module.Index(len(tempMap))
	tempMap[typ] = localIndex
	return localIndex
}

// makeStackOffset makes an expression modifying the stack pointer by the given offset.
func (p *ShadowStackPass) makeStackOffset(offset int32) module.ExpressionRef {
	if offset == 0 {
		panic("offset must not be 0")
	}
	mod := p.Module
	var op module.Op
	var absOffset int32
	if offset >= 0 {
		op = p.ptrBinaryAdd()
		absOffset = offset
	} else {
		op = p.ptrBinarySub()
		absOffset = -offset
	}
	expr := mod.GlobalSet(common.BuiltinNameStackPointer,
		mod.Binary(op,
			mod.GlobalGet(common.BuiltinNameStackPointer, p.ptrType()),
			p.ptrConst(absOffset),
		),
	)
	if offset > 0 {
		return expr
	}
	return mod.Block("", []module.ExpressionRef{
		expr,
		p.makeStackCheck(),
	}, module.TypeRefNone)
}

// makeStackFill makes a sequence of expressions zeroing the stack frame.
func (p *ShadowStackPass) makeStackFill(frameSize int32, stmts *[]module.ExpressionRef) {
	if frameSize <= 0 {
		panic("frameSize must be > 0")
	}
	mod := p.Module
	if p.options().HasFeature(common.FeatureBulkMemory) && frameSize > 16 {
		*stmts = append(*stmts,
			mod.MemoryFill(
				mod.GlobalGet(common.BuiltinNameStackPointer, p.ptrType()),
				mod.I32(0), // TODO: Wasm64 also i32?
				p.ptrConst(frameSize),
				module.DefaultMemory,
			),
		)
	} else {
		remain := frameSize
		for remain >= 8 {
			// store<i64>(__stack_pointer, 0, frameSize - remain)
			*stmts = append(*stmts,
				mod.Store(8,
					mod.GlobalGet(common.BuiltinNameStackPointer, p.ptrType()),
					mod.I64(0),
					module.TypeRefI64,
					uint32(frameSize-remain),
					0, module.DefaultMemory,
				),
			)
			remain -= 8
		}
		if remain > 0 {
			if remain != 4 {
				panic("remaining frame size must be 4")
			}
			// store<i32>(__stack_pointer, 0, frameSize - remain)
			*stmts = append(*stmts,
				mod.Store(4,
					mod.GlobalGet(common.BuiltinNameStackPointer, p.ptrType()),
					mod.I32(0),
					module.TypeRefI32,
					uint32(frameSize-remain),
					0, module.DefaultMemory,
				),
			)
		}
	}
}

// makeStackCheck makes a check that the current stack pointer is valid.
func (p *ShadowStackPass) makeStackCheck() module.ExpressionRef {
	mod := p.Module
	if !p.hasStackCheckFunction {
		p.hasStackCheckFunction = true
		mod.AddFunction("~stack_check", module.TypeRefNone, module.TypeRefNone, nil,
			mod.If(
				mod.Binary(module.BinaryOpLtI32,
					mod.GlobalGet(common.BuiltinNameStackPointer, p.ptrType()),
					mod.GlobalGet(common.BuiltinNameDataEnd, p.ptrType()),
				),
				p.Compiler.MakeStaticAbort(
					p.Compiler.EnsureStaticString("stack overflow"),
					ast.NativeSource(),
				),
				0,
			),
		)
	}
	return mod.Call("~stack_check", nil, module.TypeRefNone)
}

// updateCallOperands processes call operands, replacing tostack patterns with
// shadow stack store sequences. Returns the number of slots used.
func (p *ShadowStackPass) updateCallOperands(operands []module.ExpressionRef) int32 {
	mod := p.Module
	numSlots := int32(0)
	for i := 0; i < len(operands); i++ {
		operand := operands[i]
		match := matchPattern(mod, operand)
		if match == 0 {
			continue
		}
		if !needsSlot(match) {
			operands[i] = match
			continue
		}
		currentFunction := p.CurrentFunction()
		numLocals := binaryen.FunctionGetNumLocals(currentFunction)
		slotIndex := p.NoteSlot(currentFunction, numLocals+module.Index(p.callSlotOffset+numSlots))
		temp := p.getSharedTemp(currentFunction, p.ptrType())
		stmts := make([]module.ExpressionRef, 0, 3)
		// t = value
		stmts = append(stmts,
			mod.LocalSet(int32(temp), match, false),
		)
		// store<usize>(__stack_pointer, t, slotIndex * ptrSize)
		stmts = append(stmts,
			mod.Store(uint32(p.ptrSize()),
				mod.GlobalGet(common.BuiltinNameStackPointer, p.ptrType()),
				mod.LocalGet(int32(temp), p.ptrType()),
				p.ptrType(), uint32(slotIndex*p.ptrSize()),
				0, module.DefaultMemory,
			),
		)
		// -> t
		stmts = append(stmts,
			mod.LocalGet(int32(temp), p.ptrType()),
		)
		operands[i] = mod.Block("", stmts, p.ptrType())
		numSlots++
	}
	return numSlots
}

// VisitCallPre overrides the base visitor to instrument call operands.
func (p *ShadowStackPass) VisitCallPre(call module.ExpressionRef) {
	numOperands := binaryen.CallGetNumOperands(call)
	operands := make([]module.ExpressionRef, numOperands)
	for i := module.Index(0); i < numOperands; i++ {
		operands[i] = binaryen.CallGetOperandAt(call, i)
	}
	numSlots := p.updateCallOperands(operands)
	for i := 0; i < len(operands); i++ {
		binaryen.CallSetOperandAt(call, module.Index(i), operands[i])
	}
	if numSlots > 0 {
		// Reserve these slots for us so nested calls use their own
		p.callSlotOffset += numSlots
	}
	p.callSlotStack = append(p.callSlotStack, numSlots)
}

// VisitCall overrides the base visitor to pop the call slot stack.
func (p *ShadowStackPass) VisitCall(call module.ExpressionRef) {
	numSlots := p.callSlotStack[len(p.callSlotStack)-1]
	p.callSlotStack = p.callSlotStack[:len(p.callSlotStack)-1]
	if numSlots > 0 {
		p.callSlotOffset -= numSlots
	}
}

// VisitCallIndirectPre overrides the base visitor to instrument call_indirect operands.
func (p *ShadowStackPass) VisitCallIndirectPre(callIndirect module.ExpressionRef) {
	numOperands := binaryen.CallIndirectGetNumOperands(callIndirect)
	operands := make([]module.ExpressionRef, numOperands)
	for i := module.Index(0); i < numOperands; i++ {
		operands[i] = binaryen.CallIndirectGetOperandAt(callIndirect, i)
	}
	numSlots := p.updateCallOperands(operands)
	for i := 0; i < len(operands); i++ {
		binaryen.CallIndirectSetOperandAt(callIndirect, module.Index(i), operands[i])
	}
	if numSlots > 0 {
		// Reserve these slots for us so nested calls use their own
		p.callSlotOffset += numSlots
	}
	p.callSlotStack = append(p.callSlotStack, numSlots)
}

// VisitCallIndirect overrides the base visitor to pop the call slot stack.
func (p *ShadowStackPass) VisitCallIndirect(callIndirect module.ExpressionRef) {
	numSlots := p.callSlotStack[len(p.callSlotStack)-1]
	p.callSlotStack = p.callSlotStack[:len(p.callSlotStack)-1]
	if numSlots > 0 {
		p.callSlotOffset -= numSlots
	}
}

// VisitLocalSet overrides the base visitor to instrument local_set with tostack pattern.
func (p *ShadowStackPass) VisitLocalSet(localSet module.ExpressionRef) {
	mod := p.Module
	value := binaryen.LocalSetGetValue(localSet)
	match := matchPattern(mod, value)
	if match == 0 {
		return
	}
	if !needsSlot(match) {
		binaryen.LocalSetSetValue(localSet, match)
		return
	}
	index := binaryen.LocalSetGetIndex(localSet)
	slotIndex := p.NoteSlot(p.currentFunction, index)
	stmts := make([]module.ExpressionRef, 0, 2)
	// store<usize>(__stack_pointer, local = match, slotIndex * ptrSize)
	stmts = append(stmts,
		mod.Store(uint32(p.ptrSize()),
			mod.GlobalGet(common.BuiltinNameStackPointer, p.ptrType()),
			mod.LocalTee(int32(index), match, false, p.ptrType()),
			p.ptrType(), uint32(slotIndex*p.ptrSize()),
			0, module.DefaultMemory,
		),
	)
	if binaryen.LocalSetIsTee(localSet) {
		// -> local
		stmts = append(stmts,
			mod.LocalGet(int32(index), p.ptrType()),
		)
		p.ReplaceCurrent(mod.Flatten(stmts, p.ptrType()))
	} else {
		p.ReplaceCurrent(mod.Flatten(stmts, module.TypeRefNone))
	}
}

// updateFunction updates a function with additional locals etc.
func (p *ShadowStackPass) updateFunction(funcRef module.FunctionRef) {
	name := binaryen.FunctionGetName(funcRef)
	params := binaryen.FunctionGetParams(funcRef)
	results := binaryen.FunctionGetResults(funcRef)
	body := binaryen.FunctionGetBody(funcRef)
	if body == 0 {
		panic("function body is null")
	}
	numVars := binaryen.FunctionGetNumVars(funcRef)
	vars := make([]module.TypeRef, numVars)
	for i := module.Index(0); i < numVars; i++ {
		vars[i] = binaryen.FunctionGetVar(funcRef, i)
	}
	if tempMap, ok := p.tempMaps[funcRef]; ok {
		for typ := range tempMap {
			vars = append(vars, typ)
		}
	}
	bmod := p.Module.BinaryenModule()
	bmod.RemoveFunction(name)
	newFuncRef := bmod.AddFunction(name, params, results, vars, body)
	opts := p.options()
	if opts.SourceMap || opts.DebugInfo {
		fn := p.Compiler.Program.SearchFunctionByRef(newFuncRef)
		if fn != nil {
			fn.AddDebugInfo(p.Module, newFuncRef)
		}
	}
}

// updateExport updates a function export taking managed arguments.
func (p *ShadowStackPass) updateExport(exportRef module.ExportRef, managedOperandIndices []int32) {
	mod := p.Module
	bmod := mod.BinaryenModule()
	if binaryen.ExportGetKind(exportRef) != module.ExternalKindFunction {
		panic("expected function export")
	}

	internalName := binaryen.ExportGetValue(exportRef)
	externalName := binaryen.ExportGetName(exportRef)
	funcRef := bmod.GetFunction(internalName)
	params := binaryen.FunctionGetParams(funcRef)
	paramTypes := module.ExpandType(params)
	numParams := len(paramTypes)
	results := binaryen.FunctionGetResults(funcRef)
	numLocals := numParams
	vars := make([]module.TypeRef, 0)
	numSlots := len(managedOperandIndices)
	if numSlots == 0 {
		panic("expected managed operand indices")
	}
	frameSize := int32(numSlots) * p.ptrSize()
	wrapperName := "export:" + internalName

	if !mod.HasFunction(wrapperName) {
		stmts := make([]module.ExpressionRef, 0)
		// __stack_pointer -= frameSize
		stmts = append(stmts,
			p.makeStackOffset(-frameSize),
		)
		for slotIndex := 0; slotIndex < numSlots; slotIndex++ {
			// store<usize>(__stack_pointer, $local, slotIndex * ptrSize)
			stmts = append(stmts,
				mod.Store(uint32(p.ptrSize()),
					mod.GlobalGet(common.BuiltinNameStackPointer, p.ptrType()),
					mod.LocalGet(int32(managedOperandIndices[slotIndex]), p.ptrType()),
					p.ptrType(), uint32(int32(slotIndex)*p.ptrSize()),
					0, module.DefaultMemory,
				),
			)
		}
		forwardedOperands := make([]module.ExpressionRef, numParams)
		for i := 0; i < numParams; i++ {
			forwardedOperands[i] = mod.LocalGet(int32(i), paramTypes[i])
		}
		if results != module.TypeRefNone {
			tempIndex := numLocals
			numLocals++
			vars = append(vars, results)
			// t = original(...)
			stmts = append(stmts,
				mod.LocalSet(int32(tempIndex),
					mod.Call(internalName, forwardedOperands, results),
					false, // internal
				),
			)
			// __stack_pointer += frameSize
			stmts = append(stmts,
				p.makeStackOffset(+frameSize),
			)
			// -> t
			stmts = append(stmts,
				mod.LocalGet(int32(tempIndex), results),
			)
		} else {
			// original(...)
			stmts = append(stmts,
				mod.Call(internalName, forwardedOperands, results),
			)
			// __stack_pointer += frameSize
			stmts = append(stmts,
				p.makeStackOffset(+frameSize),
			)
		}
		mod.AddFunction(wrapperName, params, results, vars,
			mod.Block("", stmts, results),
		)
	}
	mod.RemoveExport(externalName)
	mod.AddFunctionExport(wrapperName, externalName)
}

// WalkModule overrides the base Pass to add shadow stack instrumentation.
func (p *ShadowStackPass) WalkModule() {
	// Run the pass normally
	p.Pass.WalkModule()

	// Instrument returns in functions utilizing stack slots
	mod := p.Module
	instrumentReturns := newInstrumentReturns(p)
	for fn, slotMap := range p.slotMaps {
		frameSize := int32(len(slotMap)) * p.ptrSize()

		// Instrument function returns
		instrumentReturns.frameSize = frameSize
		instrumentReturns.WalkFunction(fn)

		// Instrument function entry
		stmts := make([]module.ExpressionRef, 0)
		// __stack_pointer -= frameSize
		stmts = append(stmts,
			p.makeStackOffset(-frameSize),
		)
		// memory.fill(__stack_pointer, 0, frameSize)
		p.makeStackFill(frameSize, &stmts)

		// Handle implicit return
		body := binaryen.FunctionGetBody(fn)
		bodyType := binaryen.ExpressionGetType(body)
		if bodyType == module.TypeRefUnreachable {
			// body
			stmts = append(stmts, body)
		} else if bodyType == module.TypeRefNone {
			// body
			stmts = append(stmts, body)
			// __stack_pointer += frameSize
			stmts = append(stmts,
				p.makeStackOffset(+frameSize),
			)
		} else {
			temp := p.getSharedTemp(fn, bodyType)
			// t = body
			stmts = append(stmts,
				mod.LocalSet(int32(temp), body, false),
			)
			// __stack_pointer += frameSize
			stmts = append(stmts,
				p.makeStackOffset(+frameSize),
			)
			// -> t
			stmts = append(stmts,
				mod.LocalGet(int32(temp), bodyType),
			)
		}
		binaryen.FunctionSetBody(fn, mod.Flatten(stmts, bodyType))
	}

	// Update functions we added more locals to
	for fn := range p.tempMaps {
		p.updateFunction(fn)
	}

	// Update exports taking managed arguments
	for exportName, managedOperandIndices := range p.exportMap {
		exportRef := mod.GetExport(exportName)
		p.updateExport(exportRef, managedOperandIndices)
	}
}

// InstrumentReturns is a companion pass that instruments `return` statements
// to restore the stack frame.
type InstrumentReturns struct {
	Pass

	// parentPass is the parent ShadowStackPass.
	parentPass *ShadowStackPass
	// frameSize is the frame size of the current function being processed.
	frameSize int32
}

// newInstrumentReturns creates a new InstrumentReturns pass.
func newInstrumentReturns(shadowStack *ShadowStackPass) *InstrumentReturns {
	ir := &InstrumentReturns{
		Pass:       NewPass(shadowStack.Module),
		parentPass: shadowStack,
	}
	ir.InitVisitor(ir)
	return ir
}

// VisitReturn overrides the base visitor to instrument return statements.
func (ir *InstrumentReturns) VisitReturn(ret module.ExpressionRef) {
	if ir.frameSize == 0 {
		panic("frameSize must be set")
	}
	mod := ir.Module
	value := binaryen.ReturnGetValue(ret)
	stmts := make([]module.ExpressionRef, 0)
	if value != 0 {
		returnType := binaryen.ExpressionGetType(value)
		if returnType == module.TypeRefUnreachable {
			return
		}
		temp := ir.parentPass.getSharedTemp(ir.CurrentFunction(), returnType)
		// t = value
		stmts = append(stmts,
			mod.LocalSet(int32(temp), value, false),
		)
		// __stack_pointer += frameSize
		stmts = append(stmts,
			ir.parentPass.makeStackOffset(+ir.frameSize),
		)
		// return t
		binaryen.ReturnSetValue(ret, mod.LocalGet(int32(temp), returnType))
	} else {
		// __stack_pointer += frameSize
		stmts = append(stmts,
			ir.parentPass.makeStackOffset(+ir.frameSize),
		)
		// return
	}
	stmts = append(stmts, ret)
	ir.ReplaceCurrent(mod.Flatten(stmts, module.TypeRefUnreachable))
}
