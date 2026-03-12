package asembed

import (
	"unsafe"

	"github.com/fastschema/qjs"
)

func registerModuleImpls(ctx *qjs.Context, lm *LinearMemory) {
	ctx.SetFunc("_BinaryenModuleCreate", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoModuleCreate())
	})
	ctx.SetFunc("_BinaryenModuleDispose", func(this *qjs.This) (*qjs.Value, error) {
		cgoModuleDispose(argU(this.Args(), 0))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenModuleValidate", func(this *qjs.This) (*qjs.Value, error) {
		return retBool(this.Context(), cgoModuleValidate(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_BinaryenModuleOptimize", func(this *qjs.This) (*qjs.Value, error) {
		cgoModuleOptimize(argU(this.Args(), 0))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenModulePrint", func(this *qjs.This) (*qjs.Value, error) {
		cgoModulePrint(argU(this.Args(), 0))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenModulePrintAsmjs", func(this *qjs.This) (*qjs.Value, error) {
		cgoModulePrintAsmjs(argU(this.Args(), 0))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenModuleGetFeatures", func(this *qjs.This) (*qjs.Value, error) {
		return retU32(this.Context(), cgoModuleGetFeatures(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_BinaryenModuleSetFeatures", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoModuleSetFeatures(argU(a, 0), argU32(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenModuleRunPasses", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		passesPtr := argI(a, 1)
		numPasses := argI(a, 2)
		passes := make([]string, numPasses)
		for i := 0; i < numPasses; i++ {
			strPtr := lm.I32Load(passesPtr + i*4)
			passes[i] = lm.ReadString(strPtr)
		}
		cgoModuleRunPasses(module, passes)
		return retVoid(this.Context())
	})
	// _BinaryenModuleAllocateAndWrite — writes result struct to linear memory
	ctx.SetFunc("_BinaryenModuleAllocateAndWrite", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		outputPtr := argI(a, 1)
		sourceMapURLPtr := argI(a, 2)
		var smURL string
		if sourceMapURLPtr != 0 {
			smURL = lm.ReadString(sourceMapURLPtr)
		}
		result := cgoModuleAllocateAndWrite(module, smURL)
		// Write result to linear memory: {binary: ptr, binaryBytes: i32, sourceMap: ptr}
		binaryPtr := lm.Malloc(len(result.Binary))
		lm.WriteBytes(binaryPtr, result.Binary)
		lm.I32Store(outputPtr, binaryPtr)
		lm.I32Store(outputPtr+4, len(result.Binary))
		if result.SourceMap != "" {
			smPtr := lm.Malloc(len(result.SourceMap) + 1)
			lm.WriteString(smPtr, result.SourceMap)
			lm.I32Store(outputPtr+8, smPtr)
		} else {
			lm.I32Store(outputPtr+8, 0)
		}
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenModuleAllocateAndWriteText", func(this *qjs.This) (*qjs.Value, error) {
		text := cgoModuleAllocateAndWriteText(argU(this.Args(), 0))
		ptr := lm.Malloc(len(text) + 1)
		lm.WriteString(ptr, text)
		return retI(this.Context(), ptr)
	})
	ctx.SetFunc("_BinaryenModuleAllocateAndWriteStackIR", func(this *qjs.This) (*qjs.Value, error) {
		text := cgoModuleAllocateAndWriteStackIR(argU(this.Args(), 0))
		if text == "" {
			return retI(this.Context(), 0)
		}
		ptr := lm.Malloc(len(text) + 1)
		lm.WriteString(ptr, text)
		return retI(this.Context(), ptr)
	})
	ctx.SetFunc("_BinaryenSizeofAllocateAndWriteResult", func(this *qjs.This) (*qjs.Value, error) {
		return retI(this.Context(), 12) // {binary: i32, binaryBytes: i32, sourceMap: i32}
	})
	ctx.SetFunc("_BinaryenModulePrintStackIR", func(this *qjs.This) (*qjs.Value, error) {
		cgoModulePrintStackIR(argU(this.Args(), 0))
		return retVoid(this.Context())
	})
	// ModuleParse, ModuleRead, ModuleReadWithFeatures, ModuleInterpret — keep as stubs
	// ModuleAddDebugInfoFileName, ModuleGetDebugInfoFileName — keep as stubs
	// ModuleSetTypeName, ModuleSetFieldName — keep as stubs
	ctx.SetFunc("_BinaryenSetStart", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoSetStart(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	// _BinaryenGetStart — not in this binaryen version, stays as stub
}

func registerLiteralImpls(ctx *qjs.Context, lm *LinearMemory) {
	ctx.SetFunc("_BinaryenSizeofLiteral", func(this *qjs.This) (*qjs.Value, error) {
		return retI(this.Context(), cgoSizeofLiteral())
	})
	ctx.SetFunc("_BinaryenLiteralInt32", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		outPtr := argI(a, 0)
		value := argI32(a, 1)
		litBytes := make([]byte, cgoSizeofLiteral())
		cgoLiteralInt32(value, litBytes)
		lm.WriteBytes(outPtr, litBytes)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenLiteralInt64", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		outPtr := argI(a, 0)
		lo := argI32(a, 1)
		hi := argI32(a, 2)
		litBytes := make([]byte, cgoSizeofLiteral())
		cgoLiteralInt64(lo, hi, litBytes)
		lm.WriteBytes(outPtr, litBytes)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenLiteralFloat32", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		outPtr := argI(a, 0)
		value := argF32(a, 1)
		litBytes := make([]byte, cgoSizeofLiteral())
		cgoLiteralFloat32(value, litBytes)
		lm.WriteBytes(outPtr, litBytes)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenLiteralFloat64", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		outPtr := argI(a, 0)
		value := argF64(a, 1)
		litBytes := make([]byte, cgoSizeofLiteral())
		cgoLiteralFloat64(value, litBytes)
		lm.WriteBytes(outPtr, litBytes)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenLiteralFloat32Bits", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		outPtr := argI(a, 0)
		value := argI32(a, 1)
		litBytes := make([]byte, cgoSizeofLiteral())
		cgoLiteralFloat32Bits(value, litBytes)
		lm.WriteBytes(outPtr, litBytes)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenLiteralFloat64Bits", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		outPtr := argI(a, 0)
		lo := argI32(a, 1)
		hi := argI32(a, 2)
		litBytes := make([]byte, cgoSizeofLiteral())
		cgoLiteralFloat64Bits(lo, hi, litBytes)
		lm.WriteBytes(outPtr, litBytes)
		return retVoid(this.Context())
	})
	// _BinaryenLiteralVec128 — stays as stub for now
}

func registerExpressionInfoImpls(ctx *qjs.Context, lm *LinearMemory) {
	ctx.SetFunc("_BinaryenExpressionGetId", func(this *qjs.This) (*qjs.Value, error) {
		return retI(this.Context(), cgoExpressionGetId(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_BinaryenExpressionGetType", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoExpressionGetType(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_BinaryenExpressionSetType", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoExpressionSetType(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenExpressionPrint", func(this *qjs.This) (*qjs.Value, error) {
		cgoExpressionPrint(argU(this.Args(), 0))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenExpressionCopy", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoExpressionCopy(argU(a, 0), argU(a, 1)))
	})
	ctx.SetFunc("_BinaryenExpressionFinalize", func(this *qjs.This) (*qjs.Value, error) {
		cgoExpressionFinalize(argU(this.Args(), 0))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenExpressionGetSideEffects", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retU32(this.Context(), cgoExpressionGetSideEffects(argU(a, 0), argU(a, 1)))
	})
}

var _ = unsafe.Pointer(nil)
