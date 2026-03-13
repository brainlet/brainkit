package asembed

import (
	"unsafe"

	quickjs "github.com/buke/quickjs-go"
)

func registerModuleImpls(ctx *quickjs.Context, lm *LinearMemory) {
	setFunc(ctx, "_BinaryenModuleCreate", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoModuleCreate())
	})
	setFunc(ctx, "_BinaryenModuleDispose", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		cgoModuleDispose(argU(args, 0))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenModuleValidate", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retBool(c, cgoModuleValidate(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenModuleOptimize", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		cgoModuleOptimize(argU(args, 0))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenModulePrint", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		cgoModulePrint(argU(args, 0))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenModulePrintAsmjs", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		cgoModulePrintAsmjs(argU(args, 0))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenModuleGetFeatures", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retU32(c, cgoModuleGetFeatures(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenModuleSetFeatures", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoModuleSetFeatures(argU(a, 0), argU32(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenModuleRunPasses", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		passesPtr := argI(a, 1)
		numPasses := argI(a, 2)
		passes := make([]string, numPasses)
		for i := 0; i < numPasses; i++ {
			strPtr := lm.I32Load(passesPtr + i*4)
			passes[i] = lm.ReadString(strPtr)
		}
		cgoModuleRunPasses(module, passes)
		return retVoid(c)
	})
	// _BinaryenModuleAllocateAndWrite — writes result struct to linear memory
	setFunc(ctx, "_BinaryenModuleAllocateAndWrite", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
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
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenModuleAllocateAndWriteText", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		text := cgoModuleAllocateAndWriteText(argU(args, 0))
		ptr := lm.Malloc(len(text) + 1)
		lm.WriteString(ptr, text)
		return retI(c, ptr)
	})
	setFunc(ctx, "_BinaryenModuleAllocateAndWriteStackIR", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		text := cgoModuleAllocateAndWriteStackIR(argU(args, 0))
		if text == "" {
			return retI(c, 0)
		}
		ptr := lm.Malloc(len(text) + 1)
		lm.WriteString(ptr, text)
		return retI(c, ptr)
	})
	setFunc(ctx, "_BinaryenSizeofAllocateAndWriteResult", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retI(c, 12) // {binary: i32, binaryBytes: i32, sourceMap: i32}
	})
	setFunc(ctx, "_BinaryenModulePrintStackIR", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		cgoModulePrintStackIR(argU(args, 0))
		return retVoid(c)
	})
	// ModuleParse, ModuleRead, ModuleReadWithFeatures, ModuleInterpret — keep as stubs
	// ModuleAddDebugInfoFileName, ModuleGetDebugInfoFileName — keep as stubs
	// ModuleSetTypeName, ModuleSetFieldName — keep as stubs
	setFunc(ctx, "_BinaryenSetStart", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoSetStart(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	// _BinaryenGetStart — not in this binaryen version, stays as stub
}

func registerLiteralImpls(ctx *quickjs.Context, lm *LinearMemory) {
	setFunc(ctx, "_BinaryenSizeofLiteral", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retI(c, cgoSizeofLiteral())
	})
	setFunc(ctx, "_BinaryenLiteralInt32", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		outPtr := argI(a, 0)
		value := argI32(a, 1)
		litBytes := make([]byte, cgoSizeofLiteral())
		cgoLiteralInt32(value, litBytes)
		lm.WriteBytes(outPtr, litBytes)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenLiteralInt64", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		outPtr := argI(a, 0)
		lo := argI32(a, 1)
		hi := argI32(a, 2)
		litBytes := make([]byte, cgoSizeofLiteral())
		cgoLiteralInt64(lo, hi, litBytes)
		lm.WriteBytes(outPtr, litBytes)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenLiteralFloat32", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		outPtr := argI(a, 0)
		value := argF32(a, 1)
		litBytes := make([]byte, cgoSizeofLiteral())
		cgoLiteralFloat32(value, litBytes)
		lm.WriteBytes(outPtr, litBytes)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenLiteralFloat64", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		outPtr := argI(a, 0)
		value := argF64(a, 1)
		litBytes := make([]byte, cgoSizeofLiteral())
		cgoLiteralFloat64(value, litBytes)
		lm.WriteBytes(outPtr, litBytes)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenLiteralFloat32Bits", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		outPtr := argI(a, 0)
		value := argI32(a, 1)
		litBytes := make([]byte, cgoSizeofLiteral())
		cgoLiteralFloat32Bits(value, litBytes)
		lm.WriteBytes(outPtr, litBytes)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenLiteralFloat64Bits", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		outPtr := argI(a, 0)
		lo := argI32(a, 1)
		hi := argI32(a, 2)
		litBytes := make([]byte, cgoSizeofLiteral())
		cgoLiteralFloat64Bits(lo, hi, litBytes)
		lm.WriteBytes(outPtr, litBytes)
		return retVoid(c)
	})
	// _BinaryenLiteralVec128 — stays as stub for now
}

func registerExpressionInfoImpls(ctx *quickjs.Context, lm *LinearMemory) {
	setFunc(ctx, "_BinaryenExpressionGetId", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retI(c, cgoExpressionGetId(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenExpressionGetType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoExpressionGetType(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenExpressionSetType", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoExpressionSetType(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenExpressionPrint", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		cgoExpressionPrint(argU(args, 0))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenExpressionCopy", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoExpressionCopy(argU(a, 0), argU(a, 1)))
	})
	setFunc(ctx, "_BinaryenExpressionFinalize", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		cgoExpressionFinalize(argU(args, 0))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenExpressionGetSideEffects", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retU32(c, cgoExpressionGetSideEffects(argU(a, 0), argU(a, 1)))
	})
}

var _ = unsafe.Pointer(nil)
