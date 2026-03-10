// Ported from: assemblyscript/src/passes/pass.ts (Pass class, lines 1237-1319)
//
// Base class of custom Binaryen passes. Extends the Visitor with module walking
// capabilities and expression replacement utilities.
package passes

import (
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/pkg/binaryen"
)

// Pass is the base class for custom Binaryen passes. It embeds BaseVisitor
// and adds module-walking and expression replacement methods.
type Pass struct {
	BaseVisitor

	// Module is the module being walked.
	Module *module.Module

	currentFunction module.FunctionRef
	currentGlobal   module.GlobalRef
}

// NewPass constructs a new Pass for the given module.
func NewPass(mod *module.Module) Pass {
	p := Pass{
		Module: mod,
	}
	return p
}

// CurrentFunction returns the current function being walked.
func (p *Pass) CurrentFunction() module.FunctionRef {
	if p.currentFunction == 0 {
		panic("not walking a function")
	}
	return p.currentFunction
}

// CurrentGlobal returns the current global being walked.
func (p *Pass) CurrentGlobal() module.GlobalRef {
	if p.currentGlobal == 0 {
		panic("not walking a global")
	}
	return p.currentGlobal
}

// WalkModule walks the entire module: all functions, then all globals.
func (p *Pass) WalkModule() {
	p.WalkFunctions()
	p.WalkGlobals()
}

// WalkFunctions walks all functions in the module.
func (p *Pass) WalkFunctions() {
	bmod := p.Module.BinaryenModule()
	numFunctions := bmod.GetNumFunctions()
	for i := module.Index(0); i < numFunctions; i++ {
		p.WalkFunction(bmod.GetFunctionByIndex(i))
	}
}

// WalkFunction walks a specific function.
func (p *Pass) WalkFunction(fn module.FunctionRef) {
	body := binaryen.FunctionGetBody(fn)
	if body != 0 {
		p.currentFunction = fn
		p.self.Visit(body)
		p.currentFunction = 0
	}
}

// WalkGlobals walks all global variables in the module.
func (p *Pass) WalkGlobals() {
	bmod := p.Module.BinaryenModule()
	numGlobals := bmod.GetNumGlobals()
	for i := module.Index(0); i < numGlobals; i++ {
		p.WalkGlobal(bmod.GetGlobalByIndex(i))
	}
}

// WalkGlobal walks a specific global variable.
func (p *Pass) WalkGlobal(global module.GlobalRef) {
	p.currentGlobal = global
	init := binaryen.GlobalGetInitExpr(global)
	if init != 0 {
		p.self.Visit(init)
	}
	p.currentGlobal = 0
}

// ReplaceCurrent replaces the current expression with the specified replacement.
func (p *Pass) ReplaceCurrent(replacement module.ExpressionRef) {
	search := p.CurrentExpression()
	fn := p.CurrentFunction()
	body := binaryen.FunctionGetBody(fn)
	if body == search {
		binaryen.FunctionSetBody(fn, replacement)
	} else {
		parent := p.ParentExpressionOrNull()
		if parent == 0 {
			panic("no parent expression to replace in")
		}
		replaced := ReplaceChild(parent, search, replacement)
		if replaced == 0 {
			panic("failed to replace expression")
		}
		binaryen.ExpressionFinalize(parent)
	}
}
