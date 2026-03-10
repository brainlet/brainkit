// Ported from: assemblyscript/src/builtins.ts
// Builtin dispatch map, registration hub, and shared helpers.
// Type/context declarations and type-query builtins are in builtins_types.go.
// Math builtins are in builtins_math.go.
// Memory builtins are in builtins_memory.go.
// Atomics builtins are in builtins_atomics.go.
// SIMD builtins are in builtins_simd.go.
// Type-specific SIMD aliases are in builtins_aliases.go.
// Additional context types (BuiltinTypesContext, BuiltinVariableContext) are in builtins_context.go.
package compiler

import (
	"strconv"

	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/types"
	"github.com/brainlet/brainkit/wasm-kit/util"
)

func init() {
	registerMemoryBuiltins()
	registerAtomicsBuiltins()
	registerControlBuiltins()
	registerSIMDBuiltins()
	registerAliasBuiltins()
}

// GetBuiltinHandler returns the builtin handler for the given internal name, or nil.
func GetBuiltinHandler(internalName string) BuiltinFunction {
	if h, ok := builtinFunctions[internalName]; ok {
		return h
	}
	return nil
}

// typeArgsRange is a helper that returns a pointer to the type arguments range.
func typeArgsRange(ctx *BuiltinFunctionContext) *diagnostics.Range {
	rng := ctx.ReportNode.TypeArgumentsRange()
	return &rng
}

// argsRange is a helper that returns a pointer to the arguments range.
func argsRange(ctx *BuiltinFunctionContext) *diagnostics.Range {
	rng := ctx.ReportNode.ArgumentsRange()
	return &rng
}

// boolToInt converts a bool to an int for bitwise-or error checking patterns.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// === Check helpers (not in builtins_types.go) ===

// checkFeatureEnabled checks that the specified feature is enabled. Returns true on error.
// Ported from: assemblyscript/src/builtins.ts checkFeatureEnabled (line 11257).
func checkFeatureEnabled(ctx *BuiltinFunctionContext, feature common.Feature) bool {
	if !ctx.Compiler.Options().HasFeature(feature) {
		ctx.Compiler.Error(
			diagnostics.DiagnosticCodeFeature0IsNotEnabled,
			ctx.ReportNode.GetRange(),
			common.FeatureToString(feature), "", "",
		)
		return true
	}
	return false
}

// === Immediate evaluation helpers ===

// evaluateImmediateOffset evaluates a compile-time constant immediate offset argument.
// Returns a non-negative offset on success, or -1 on error (already reported).
// Ported from: assemblyscript/src/builtins.ts evaluateImmediateOffset (line 11203).
func evaluateImmediateOffset(expression ast.Node, c *Compiler) int32 {
	mod := c.Module()
	var value int32
	if c.Options().IsWasm64() {
		expr := c.CompileExpression(expression, types.TypeUsize64, ConstraintsConvImplicit)
		precomp := mod.RunExpression(expr, module.ExpressionRunnerFlagsPreserveSideeffects, 8, 1)
		if precomp != 0 {
			// assert high == 0 (TODO: support full 64-bit offsets)
			value = module.GetConstValueI64Low(precomp)
		} else {
			c.Error(
				diagnostics.DiagnosticCodeExpressionMustBeACompileTimeConstant,
				expression.GetRange(),
				"", "", "",
			)
			value = -1
		}
	} else {
		expr := c.CompileExpression(expression, types.TypeUsize32, ConstraintsConvImplicit)
		precomp := mod.RunExpression(expr, module.ExpressionRunnerFlagsPreserveSideeffects, 8, 1)
		if precomp != 0 {
			value = module.GetConstValueI32(precomp)
		} else {
			c.Error(
				diagnostics.DiagnosticCodeExpressionMustBeACompileTimeConstant,
				expression.GetRange(),
				"", "", "",
			)
			value = -1
		}
	}
	return value
}

// evaluateImmediateAlign evaluates a compile-time constant immediate align argument.
// Returns a positive alignment on success, or -1 on error (already reported).
// Ported from: assemblyscript/src/builtins.ts evaluateImmediateAlign (line 11236).
func evaluateImmediateAlign(expression ast.Node, naturalAlign int32, c *Compiler) int32 {
	align := evaluateImmediateOffset(expression, c)
	if align < 0 {
		return align
	}
	if align < 1 || naturalAlign > 16 {
		c.Error(
			diagnostics.DiagnosticCode0MustBeAValueBetween1And2Inclusive,
			expression.GetRange(),
			"Alignment", "1", strconv.Itoa(int(naturalAlign)),
		)
		return -1
	}
	if !util.IsPowerOf2(align) {
		c.Error(
			diagnostics.DiagnosticCode0MustBeAPowerOfTwo,
			expression.GetRange(),
			"Alignment", "", "",
		)
		return -1
	}
	return align
}
