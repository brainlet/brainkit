// Ported from: src/glue/binaryen.d.ts (Relooper section)
package binaryen

/*
#include "binaryen-c.h"
*/
import "C"
import "unsafe"

// ---------------------------------------------------------------------------
// Relooper — CFG to structured control flow
// ---------------------------------------------------------------------------

// Relooper wraps the Binaryen Relooper for converting CFG into structured Wasm.
// Usage: Create → AddBlock(s) → AddBranch(es) → RenderAndDispose.
type Relooper struct {
	ref C.RelooperRef
}

// RelooperBlock wraps a Relooper basic block.
type RelooperBlock struct {
	ref C.RelooperBlockRef
}

// NewRelooper creates a new Relooper instance for the given module.
func (m *Module) NewRelooper() *Relooper {
	return &Relooper{ref: C.RelooperCreate(m.ref)}
}

// AddBlock adds a basic block with the given code expression.
func (r *Relooper) AddBlock(code ExpressionRef) *RelooperBlock {
	return &RelooperBlock{
		ref: C.RelooperAddBlock(r.ref, (C.BinaryenExpressionRef)(unsafe.Pointer(code))),
	}
}

// AddBranch adds a branch from one block to another.
// condition can be 0 for unconditional branches.
// code can be 0 for no code on the branch (used for phis).
func RelooperAddBranch(from, to *RelooperBlock, condition, code ExpressionRef) {
	C.RelooperAddBranch(
		from.ref,
		to.ref,
		(C.BinaryenExpressionRef)(unsafe.Pointer(condition)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(code)),
	)
}

// AddBlockWithSwitch adds a block that ends with a switch on a condition.
func (r *Relooper) AddBlockWithSwitch(code, condition ExpressionRef) *RelooperBlock {
	return &RelooperBlock{
		ref: C.RelooperAddBlockWithSwitch(
			r.ref,
			(C.BinaryenExpressionRef)(unsafe.Pointer(code)),
			(C.BinaryenExpressionRef)(unsafe.Pointer(condition)),
		),
	}
}

// AddBranchForSwitch adds a switch-style branch to another block.
// indexes specifies which switch table entries go to the target.
func RelooperAddBranchForSwitch(from, to *RelooperBlock, indexes []Index, code ExpressionRef) {
	var idxPtr *C.BinaryenIndex
	if len(indexes) > 0 {
		idxPtr = (*C.BinaryenIndex)(unsafe.Pointer(&indexes[0]))
	}
	C.RelooperAddBranchForSwitch(
		from.ref,
		to.ref,
		idxPtr,
		C.BinaryenIndex(len(indexes)),
		(C.BinaryenExpressionRef)(unsafe.Pointer(code)),
	)
}

// RenderAndDispose generates structured control flow from the CFG and disposes the Relooper.
// entry is the entry block. labelHelper is the index of a free i32 local for irreducible control flow.
// After this call, the Relooper and its blocks/branches are invalid.
func (r *Relooper) RenderAndDispose(entry *RelooperBlock, labelHelper Index) ExpressionRef {
	ref := C.RelooperRenderAndDispose(r.ref, entry.ref, C.BinaryenIndex(labelHelper))
	return ExpressionRef(unsafe.Pointer(ref))
}
