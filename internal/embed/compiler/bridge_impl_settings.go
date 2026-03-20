package asembed

import (
	"unsafe"

	quickjs "github.com/buke/quickjs-go"
)

func registerSettingsImpls(ctx *quickjs.Context, lm *LinearMemory) {
	setFunc(ctx, "_BinaryenGetOptimizeLevel", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retI(c, cgoGetOptimizeLevel())
	})
	setFunc(ctx, "_BinaryenSetOptimizeLevel", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		cgoSetOptimizeLevel(argI(args, 0))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenGetShrinkLevel", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retI(c, cgoGetShrinkLevel())
	})
	setFunc(ctx, "_BinaryenSetShrinkLevel", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		cgoSetShrinkLevel(argI(args, 0))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenGetDebugInfo", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retBool(c, cgoGetDebugInfo())
	})
	setFunc(ctx, "_BinaryenSetDebugInfo", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		cgoSetDebugInfo(argBool(args, 0))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenGetTrapsNeverHappen", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retBool(c, cgoGetTrapsNeverHappen())
	})
	setFunc(ctx, "_BinaryenSetTrapsNeverHappen", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		cgoSetTrapsNeverHappen(argBool(args, 0))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenGetClosedWorld", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retBool(c, cgoGetClosedWorld())
	})
	setFunc(ctx, "_BinaryenSetClosedWorld", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		cgoSetClosedWorld(argBool(args, 0))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenGetLowMemoryUnused", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retBool(c, cgoGetLowMemoryUnused())
	})
	setFunc(ctx, "_BinaryenSetLowMemoryUnused", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		cgoSetLowMemoryUnused(argBool(args, 0))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenGetZeroFilledMemory", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retBool(c, cgoGetZeroFilledMemory())
	})
	setFunc(ctx, "_BinaryenSetZeroFilledMemory", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		cgoSetZeroFilledMemory(argBool(args, 0))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenGetFastMath", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retBool(c, cgoGetFastMath())
	})
	setFunc(ctx, "_BinaryenSetFastMath", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		cgoSetFastMath(argBool(args, 0))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenGetGenerateStackIR", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retBool(c, cgoGetGenerateStackIR())
	})
	setFunc(ctx, "_BinaryenSetGenerateStackIR", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		cgoSetGenerateStackIR(argBool(args, 0))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenGetOptimizeStackIR", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retBool(c, cgoGetOptimizeStackIR())
	})
	setFunc(ctx, "_BinaryenSetOptimizeStackIR", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		cgoSetOptimizeStackIR(argBool(args, 0))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenGetPassArgument", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		namePtr := argI(a, 0)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		result := cgoGetPassArgument(name)
		if result == nil {
			return retI(c, 0)
		}
		s := cgoGoString(result)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})
	setFunc(ctx, "_BinaryenSetPassArgument", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		namePtr := argI(a, 0)
		valuePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		var value unsafe.Pointer
		if valuePtr != 0 {
			value = cgoCString(lm.ReadString(valuePtr))
			defer cgoFree(unsafe.Pointer(value))
		}
		cgoSetPassArgument(name, value)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenClearPassArguments", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		cgoClearPassArguments()
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenHasPassToSkip", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		namePtr := argI(a, 0)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retBool(c, cgoHasPassToSkip(name))
	})
	setFunc(ctx, "_BinaryenAddPassToSkip", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		namePtr := argI(a, 0)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		cgoAddPassToSkip(name)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenClearPassesToSkip", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		cgoClearPassesToSkip()
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenGetAlwaysInlineMaxSize", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retI(c, cgoGetAlwaysInlineMaxSize())
	})
	setFunc(ctx, "_BinaryenSetAlwaysInlineMaxSize", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		cgoSetAlwaysInlineMaxSize(argI(args, 0))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenGetFlexibleInlineMaxSize", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retI(c, cgoGetFlexibleInlineMaxSize())
	})
	setFunc(ctx, "_BinaryenSetFlexibleInlineMaxSize", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		cgoSetFlexibleInlineMaxSize(argI(args, 0))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenGetOneCallerInlineMaxSize", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retI(c, cgoGetOneCallerInlineMaxSize())
	})
	setFunc(ctx, "_BinaryenSetOneCallerInlineMaxSize", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		cgoSetOneCallerInlineMaxSize(argI(args, 0))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenGetAllowInliningFunctionsWithLoops", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retBool(c, cgoGetAllowInliningFunctionsWithLoops())
	})
	setFunc(ctx, "_BinaryenSetAllowInliningFunctionsWithLoops", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		cgoSetAllowInliningFunctionsWithLoops(argBool(args, 0))
		return retVoid(c)
	})
}

var _ = unsafe.Pointer(nil)
