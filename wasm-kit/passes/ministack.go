// Ported from: assemblyscript/src/passes/ministack.ts (MiniStack class, 182 lines)
//
// A potential minimalistic shadow stack. Currently not used.
//
// Instruments a module's exports to track when the execution stack is fully
// unwound, and injects a call to `__autocollect` to be invoked when it is.
// Accounts for the currently in-flight managed return value from Wasm to the
// host by pushing it to a mini stack, essentially a stack of only one value,
// while `__autocollect` is executing.
package passes

import (
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/pkg/binaryen"
	"github.com/brainlet/brainkit/wasm-kit/program"
)

const (
	MINISTACK   = "~lib/rt/__ministack"
	STACK_DEPTH = "~stack_depth"
	AUTOCOLLECT = "__autocollect"
)

// MiniStack instruments a module with a minimalistic shadow stack for precise GC.
type MiniStack struct {
	Pass

	// Program reference.
	Program *program.Program
	// Exported functions returning managed values.
	managedReturns map[string]struct{}
}

// NewMiniStack constructs a new MiniStack pass for the given module and program.
func NewMiniStack(mod *module.Module, prog *program.Program) *MiniStack {
	p := &MiniStack{
		Pass:           NewPass(mod),
		Program:        prog,
		managedReturns: make(map[string]struct{}),
	}
	p.InitVisitor(p)
	return p
}

// NoteManagedReturn notes the presence of an exported function with a managed return value.
func (p *MiniStack) NoteManagedReturn(exportName string) {
	p.managedReturns[exportName] = struct{}{}
}

// instrumentFunctionExport instruments a function export to also maintain stack depth.
func (p *MiniStack) instrumentFunctionExport(ref module.ExportRef) {
	if binaryen.ExportGetKind(ref) != module.ExternalKindFunction {
		panic("expected function export")
	}
	mod := p.Module
	bmod := mod.BinaryenModule()
	internalName := binaryen.ExportGetValue(ref)
	externalName := binaryen.ExportGetName(ref)
	functionRef := bmod.GetFunction(internalName)
	originalName := binaryen.FunctionGetName(functionRef)

	wrapperName := "export:" + originalName
	if !mod.HasFunction(wrapperName) {
		params := binaryen.FunctionGetParams(functionRef)
		results := binaryen.FunctionGetResults(functionRef)
		numLocals := binaryen.FunctionGetNumLocals(functionRef)
		vars := make([]module.TypeRef, 0)

		// Prepare a call to the original function
		paramTypes := module.ExpandType(params)
		numParams := len(paramTypes)
		operands := make([]module.ExpressionRef, numParams)
		for i := 0; i < numParams; i++ {
			operands[i] = mod.LocalGet(int32(i), paramTypes[i])
		}
		call := mod.Call(originalName, operands, results)

		// Create a wrapper function also maintaining stack depth
		stmts := make([]module.ExpressionRef, 0)
		if numLocals > 0 {
			stmts = append(stmts,
				mod.GlobalSet(STACK_DEPTH,
					mod.Binary(module.BinaryOpAddI32,
						mod.GlobalGet(STACK_DEPTH, module.TypeRefI32),
						mod.I32(1), // only need to know > 0
					),
				),
			)
		}
		if results == module.TypeRefNone {
			stmts = append(stmts, call)
		} else {
			vars = append(vars, results)
			stmts = append(stmts,
				mod.LocalSet(int32(numParams), call, false), // no shadow stack here
			)
		}
		if numLocals > 0 {
			stmts = append(stmts,
				mod.GlobalSet(STACK_DEPTH,
					mod.Binary(module.BinaryOpSubI32,
						mod.GlobalGet(STACK_DEPTH, module.TypeRefI32),
						mod.I32(1), // only need to know > 0
					),
				),
			)
		}
		// Push managed return value (or zero) to ministack
		var ministackValue module.ExpressionRef
		if _, ok := p.managedReturns[externalName]; ok {
			ministackValue = mod.LocalGet(int32(numParams), results)
		} else {
			ministackValue = mod.I32(0)
		}
		stmts = append(stmts,
			mod.GlobalSet(MINISTACK, ministackValue),
		)
		stmts = append(stmts,
			mod.If(
				mod.Unary(module.UnaryOpEqzI32,
					mod.GlobalGet(STACK_DEPTH, module.TypeRefI32),
				),
				mod.Call(AUTOCOLLECT, nil, module.TypeRefNone),
				0,
			),
		)
		if results != module.TypeRefNone {
			stmts = append(stmts,
				mod.LocalGet(int32(numParams), results),
			)
		}
		mod.AddFunction(wrapperName, params, results, vars,
			mod.Block("", stmts, results),
		)
	}

	// Replace the original export with the wrapped one
	mod.RemoveExport(externalName)
	mod.AddFunctionExport(wrapperName, externalName)
}

// Run runs the pass. Returns true if the mini stack has been added.
func (p *MiniStack) Run() bool {
	mod := p.Module
	bmod := mod.BinaryenModule()
	numExports := bmod.GetNumExports()
	if numExports > 0 {
		functionExportRefs := make([]module.ExportRef, 0)
		// We are going to modify the list of exports, so do this in two steps
		for i := module.Index(0); i < numExports; i++ {
			exportRef := bmod.GetExportByIndex(i)
			if binaryen.ExportGetKind(exportRef) == module.ExternalKindFunction {
				functionExportRefs = append(functionExportRefs, exportRef)
			}
		}
		numFunctionExports := len(functionExportRefs)
		if numFunctionExports > 0 {
			for i := 0; i < numFunctionExports; i++ {
				p.instrumentFunctionExport(functionExportRefs[i])
			}
			mod.AddGlobal(STACK_DEPTH, module.TypeRefI32, true, mod.I32(0))
			return true
		}
	}
	return false
}
