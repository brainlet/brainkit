// Ported from: assemblyscript/src/passes/rtrace.ts (RtraceMemory class, 80 lines)
//
// A lightweight store instrumentation pass. Can be used to find rogue stores to
// protected memory addresses like object headers or similar, without going
// overboard with instrumentation. Also passes a flag whether a store originates
// within the runtime or other code.
package passes

import (
	"strings"

	"github.com/brainlet/brainkit/wasm-kit/compiler"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/pkg/binaryen"
)

// RtraceMemory instruments stores to also call an import.
type RtraceMemory struct {
	Pass

	// Whether we've seen any stores.
	seenStores bool
	// Target pointer type.
	ptrType module.TypeRef
}

// NewRtraceMemory constructs a new RtraceMemory pass for the given compiler.
func NewRtraceMemory(c *compiler.Compiler) *RtraceMemory {
	p := &RtraceMemory{
		Pass:    NewPass(c.Module()),
		ptrType: c.Options().SizeTypeRef(),
	}
	p.InitVisitor(p)
	return p
}

// checkRT checks if the current function is within the runtime library.
func (p *RtraceMemory) checkRT() bool {
	functionName := binaryen.FunctionGetName(p.CurrentFunction())
	return strings.HasPrefix(functionName, "~lib/rt/")
}

// VisitStore overrides the base visitor to instrument store expressions.
func (p *RtraceMemory) VisitStore(store module.ExpressionRef) {
	mod := p.Module
	ptr := binaryen.StoreGetPtr(store)
	offset := binaryen.StoreGetOffset(store)
	bytes := binaryen.StoreGetBytes(store)
	// onstore(ptr: usize, offset: i32, bytes: i32, isRT: bool) -> ptr
	var isRT int32
	if p.checkRT() {
		isRT = 1
	}
	binaryen.StoreSetPtr(store,
		mod.Call("~onstore", []module.ExpressionRef{
			ptr,
			mod.I32(int32(offset)),
			mod.I32(int32(bytes)),
			mod.I32(isRT),
		}, p.ptrType),
	)
	p.seenStores = true
}

// TODO: MemoryFill, Atomics

// WalkModule overrides the base pass to walk the module and add the import if needed.
func (p *RtraceMemory) WalkModule() {
	p.Pass.WalkModule()
	if p.seenStores {
		p.Module.AddFunctionImport("~onstore", "rtrace", "onstore",
			module.CreateType([]module.TypeRef{p.ptrType, module.TypeRefI32, module.TypeRefI32, module.TypeRefI32}),
			p.ptrType,
		)
	}
}
