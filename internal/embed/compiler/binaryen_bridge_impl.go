package asembed

import (
	"unsafe"

	quickjs "github.com/buke/quickjs-go"
)

// arg helpers — keep the bridge code compact
func argU(args []*quickjs.Value, i int) uintptr {
	if i >= len(args) {
		return 0
	}
	return uintptr(uint64(args[i].ToFloat64()))
}

func argI(args []*quickjs.Value, i int) int {
	if i >= len(args) {
		return 0
	}
	return int(args[i].ToInt32())
}

func argI32(args []*quickjs.Value, i int) int32 {
	if i >= len(args) {
		return 0
	}
	return args[i].ToInt32()
}

func argU32(args []*quickjs.Value, i int) uint32 {
	if i >= len(args) {
		return 0
	}
	return uint32(args[i].ToInt32())
}

func argF32(args []*quickjs.Value, i int) float32 {
	if i >= len(args) {
		return 0
	}
	return float32(args[i].ToFloat64())
}

func argF64(args []*quickjs.Value, i int) float64 {
	if i >= len(args) {
		return 0
	}
	return args[i].ToFloat64()
}

func argBool(args []*quickjs.Value, i int) bool {
	if i >= len(args) {
		return false
	}
	return args[i].ToInt32() != 0
}

// readCStr reads a null-terminated string from linear memory at the given pointer.
// Returns a C string (as unsafe.Pointer) that must be freed by the caller, or nil if ptr==0.
func readCStr(lm *LinearMemory, ptr int) unsafe.Pointer {
	if ptr == 0 {
		return nil
	}
	s := lm.ReadString(ptr)
	return cgoCString(s)
}

// readPtrArray reads count pointers from linear memory starting at ptr.
// Uses I32LoadPtr to retrieve full 64-bit pointer values on ARM64.
func readPtrArray(lm *LinearMemory, ptr, count int) []uintptr {
	if ptr == 0 || count <= 0 {
		return nil
	}
	result := make([]uintptr, count)
	for i := 0; i < count; i++ {
		result[i] = lm.I32LoadPtr(ptr + i*4)
	}
	return result
}

// retF returns a float64 value.
func retF(ctx *quickjs.Context, v uintptr) *quickjs.Value {
	return ctx.NewFloat64(float64(v))
}

// retI returns an int as float64 value.
func retI(ctx *quickjs.Context, v int) *quickjs.Value {
	return ctx.NewFloat64(float64(v))
}

// retI32 returns an int32 as float64 value.
func retI32(ctx *quickjs.Context, v int32) *quickjs.Value {
	return ctx.NewFloat64(float64(v))
}

// retU32 returns a uint32 as float64 value.
func retU32(ctx *quickjs.Context, v uint32) *quickjs.Value {
	return ctx.NewFloat64(float64(v))
}

// retBool returns a bool as float64 (0 or 1).
func retBool(ctx *quickjs.Context, v bool) *quickjs.Value {
	if v {
		return ctx.NewFloat64(1)
	}
	return ctx.NewFloat64(0)
}

// retVoid returns 0 for void functions.
func retVoid(ctx *quickjs.Context) *quickjs.Value {
	return ctx.NewFloat64(0)
}

// setFunc registers a bridge function as a global JS function.
// This helper standardizes the registration pattern for all bridge functions.
func setFunc(ctx *quickjs.Context, name string, fn func(ctx *quickjs.Context, args []*quickjs.Value) *quickjs.Value) {
	ctx.Globals().Set(name, ctx.NewFunction(func(c *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return fn(c, args)
	}))
}

// RegisterBinaryenBridgeImpl re-registers all Binaryen bridge functions
// with real CGo implementations, overriding the stubs from RegisterBinaryenBridge.
func RegisterBinaryenBridgeImpl(ctx *quickjs.Context, lm *LinearMemory) {
	registerTypeImpls(ctx, lm)
	registerHeapTypeImpls(ctx, lm)
	registerStructArraySigTypeImpls(ctx, lm)
	registerModuleImpls(ctx, lm)
	registerLiteralImpls(ctx, lm)
	registerExpressionInfoImpls(ctx, lm)
	registerExpressionConstructorImpls(ctx, lm)
	registerExpressionGetterSetterImpls(ctx, lm)
	registerAllSetterImpls(ctx, lm)
	registerFunctionImpls(ctx, lm)
	registerGlobalImpls(ctx, lm)
	registerExportImpls(ctx, lm)
	registerImportImpls(ctx, lm)
	registerTagImpls(ctx, lm)
	registerTableImpls(ctx, lm)
	registerMemoryImpls(ctx, lm)
	registerElementSegmentImpls(ctx, lm)
	registerSettingsImpls(ctx, lm)
	registerOpcodeImpls(ctx, lm)
	registerFeatureImpls(ctx, lm)
	registerExprIdImpls(ctx, lm)
	registerRelooperImpls(ctx, lm)
	registerExpressionRunnerImpls(ctx, lm)
	registerTypeBuilderImpls(ctx, lm)
	registerGCImpls(ctx, lm)
	registerStringImpls(ctx, lm)
	registerMiscImpls(ctx, lm)
	registerAllConstructorImpls(ctx, lm)
	registerAllGetterImpls(ctx, lm)
}
