package asembed

import (
	"unsafe"

	"github.com/fastschema/qjs"
)

func registerSettingsImpls(ctx *qjs.Context, lm *LinearMemory) {
	ctx.SetFunc("_BinaryenGetOptimizeLevel", func(this *qjs.This) (*qjs.Value, error) {
		return retI(this.Context(), cgoGetOptimizeLevel())
	})
	ctx.SetFunc("_BinaryenSetOptimizeLevel", func(this *qjs.This) (*qjs.Value, error) {
		cgoSetOptimizeLevel(argI(this.Args(), 0))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenGetShrinkLevel", func(this *qjs.This) (*qjs.Value, error) {
		return retI(this.Context(), cgoGetShrinkLevel())
	})
	ctx.SetFunc("_BinaryenSetShrinkLevel", func(this *qjs.This) (*qjs.Value, error) {
		cgoSetShrinkLevel(argI(this.Args(), 0))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenGetDebugInfo", func(this *qjs.This) (*qjs.Value, error) {
		return retBool(this.Context(), cgoGetDebugInfo())
	})
	ctx.SetFunc("_BinaryenSetDebugInfo", func(this *qjs.This) (*qjs.Value, error) {
		cgoSetDebugInfo(argBool(this.Args(), 0))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenGetTrapsNeverHappen", func(this *qjs.This) (*qjs.Value, error) {
		return retBool(this.Context(), cgoGetTrapsNeverHappen())
	})
	ctx.SetFunc("_BinaryenSetTrapsNeverHappen", func(this *qjs.This) (*qjs.Value, error) {
		cgoSetTrapsNeverHappen(argBool(this.Args(), 0))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenGetClosedWorld", func(this *qjs.This) (*qjs.Value, error) {
		return retBool(this.Context(), cgoGetClosedWorld())
	})
	ctx.SetFunc("_BinaryenSetClosedWorld", func(this *qjs.This) (*qjs.Value, error) {
		cgoSetClosedWorld(argBool(this.Args(), 0))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenGetLowMemoryUnused", func(this *qjs.This) (*qjs.Value, error) {
		return retBool(this.Context(), cgoGetLowMemoryUnused())
	})
	ctx.SetFunc("_BinaryenSetLowMemoryUnused", func(this *qjs.This) (*qjs.Value, error) {
		cgoSetLowMemoryUnused(argBool(this.Args(), 0))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenGetZeroFilledMemory", func(this *qjs.This) (*qjs.Value, error) {
		return retBool(this.Context(), cgoGetZeroFilledMemory())
	})
	ctx.SetFunc("_BinaryenSetZeroFilledMemory", func(this *qjs.This) (*qjs.Value, error) {
		cgoSetZeroFilledMemory(argBool(this.Args(), 0))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenGetFastMath", func(this *qjs.This) (*qjs.Value, error) {
		return retBool(this.Context(), cgoGetFastMath())
	})
	ctx.SetFunc("_BinaryenSetFastMath", func(this *qjs.This) (*qjs.Value, error) {
		cgoSetFastMath(argBool(this.Args(), 0))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenGetGenerateStackIR", func(this *qjs.This) (*qjs.Value, error) {
		return retBool(this.Context(), cgoGetGenerateStackIR())
	})
	ctx.SetFunc("_BinaryenSetGenerateStackIR", func(this *qjs.This) (*qjs.Value, error) {
		cgoSetGenerateStackIR(argBool(this.Args(), 0))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenGetOptimizeStackIR", func(this *qjs.This) (*qjs.Value, error) {
		return retBool(this.Context(), cgoGetOptimizeStackIR())
	})
	ctx.SetFunc("_BinaryenSetOptimizeStackIR", func(this *qjs.This) (*qjs.Value, error) {
		cgoSetOptimizeStackIR(argBool(this.Args(), 0))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenGetPassArgument", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		namePtr := argI(a, 0)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		result := cgoGetPassArgument(name)
		if result == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(result)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})
	ctx.SetFunc("_BinaryenSetPassArgument", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
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
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenClearPassArguments", func(this *qjs.This) (*qjs.Value, error) {
		cgoClearPassArguments()
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenHasPassToSkip", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		namePtr := argI(a, 0)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retBool(this.Context(), cgoHasPassToSkip(name))
	})
	ctx.SetFunc("_BinaryenAddPassToSkip", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		namePtr := argI(a, 0)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		cgoAddPassToSkip(name)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenClearPassesToSkip", func(this *qjs.This) (*qjs.Value, error) {
		cgoClearPassesToSkip()
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenGetAlwaysInlineMaxSize", func(this *qjs.This) (*qjs.Value, error) {
		return retI(this.Context(), cgoGetAlwaysInlineMaxSize())
	})
	ctx.SetFunc("_BinaryenSetAlwaysInlineMaxSize", func(this *qjs.This) (*qjs.Value, error) {
		cgoSetAlwaysInlineMaxSize(argI(this.Args(), 0))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenGetFlexibleInlineMaxSize", func(this *qjs.This) (*qjs.Value, error) {
		return retI(this.Context(), cgoGetFlexibleInlineMaxSize())
	})
	ctx.SetFunc("_BinaryenSetFlexibleInlineMaxSize", func(this *qjs.This) (*qjs.Value, error) {
		cgoSetFlexibleInlineMaxSize(argI(this.Args(), 0))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenGetOneCallerInlineMaxSize", func(this *qjs.This) (*qjs.Value, error) {
		return retI(this.Context(), cgoGetOneCallerInlineMaxSize())
	})
	ctx.SetFunc("_BinaryenSetOneCallerInlineMaxSize", func(this *qjs.This) (*qjs.Value, error) {
		cgoSetOneCallerInlineMaxSize(argI(this.Args(), 0))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenGetAllowInliningFunctionsWithLoops", func(this *qjs.This) (*qjs.Value, error) {
		return retBool(this.Context(), cgoGetAllowInliningFunctionsWithLoops())
	})
	ctx.SetFunc("_BinaryenSetAllowInliningFunctionsWithLoops", func(this *qjs.This) (*qjs.Value, error) {
		cgoSetAllowInliningFunctionsWithLoops(argBool(this.Args(), 0))
		return retVoid(this.Context())
	})
}

var _ = unsafe.Pointer(nil)
