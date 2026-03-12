package asembed

import (
	"unsafe"

	"github.com/fastschema/qjs"
)

// arg helpers — keep the bridge code compact
func argU(args []*qjs.Value, i int) uintptr {
	if i >= len(args) {
		return 0
	}
	return uintptr(uint64(args[i].Float64()))
}

func argI(args []*qjs.Value, i int) int {
	if i >= len(args) {
		return 0
	}
	return int(args[i].Int32())
}

func argI32(args []*qjs.Value, i int) int32 {
	if i >= len(args) {
		return 0
	}
	return args[i].Int32()
}

func argU32(args []*qjs.Value, i int) uint32 {
	if i >= len(args) {
		return 0
	}
	return uint32(args[i].Int32())
}

func argF32(args []*qjs.Value, i int) float32 {
	if i >= len(args) {
		return 0
	}
	return float32(args[i].Float64())
}

func argF64(args []*qjs.Value, i int) float64 {
	if i >= len(args) {
		return 0
	}
	return args[i].Float64()
}

func argBool(args []*qjs.Value, i int) bool {
	if i >= len(args) {
		return false
	}
	return args[i].Int32() != 0
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
func readPtrArray(lm *LinearMemory, ptr, count int) []uintptr {
	if ptr == 0 || count <= 0 {
		return nil
	}
	result := make([]uintptr, count)
	for i := 0; i < count; i++ {
		result[i] = uintptr(uint32(lm.I32Load(ptr + i*4)))
	}
	return result
}

// retF returns a float64 qjs value.
func retF(ctx *qjs.Context, v uintptr) (*qjs.Value, error) {
	return ctx.NewFloat64(float64(v)), nil
}

// retI returns an int as float64 qjs value.
func retI(ctx *qjs.Context, v int) (*qjs.Value, error) {
	return ctx.NewFloat64(float64(v)), nil
}

// retI32 returns an int32 as float64 qjs value.
func retI32(ctx *qjs.Context, v int32) (*qjs.Value, error) {
	return ctx.NewFloat64(float64(v)), nil
}

// retU32 returns a uint32 as float64 qjs value.
func retU32(ctx *qjs.Context, v uint32) (*qjs.Value, error) {
	return ctx.NewFloat64(float64(v)), nil
}

// retBool returns a bool as float64 (0 or 1).
func retBool(ctx *qjs.Context, v bool) (*qjs.Value, error) {
	if v {
		return ctx.NewFloat64(1), nil
	}
	return ctx.NewFloat64(0), nil
}

// retVoid returns 0 for void functions.
func retVoid(ctx *qjs.Context) (*qjs.Value, error) {
	return ctx.NewFloat64(0), nil
}

// RegisterBinaryenBridgeImpl re-registers all Binaryen bridge functions
// with real CGo implementations, overriding the stubs from RegisterBinaryenBridge.
func RegisterBinaryenBridgeImpl(ctx *qjs.Context, lm *LinearMemory) {
	registerTypeImpls(ctx, lm)
	registerHeapTypeImpls(ctx, lm)
	registerStructArraySigTypeImpls(ctx, lm)
	registerModuleImpls(ctx, lm)
	registerLiteralImpls(ctx, lm)
	registerExpressionInfoImpls(ctx, lm)
	registerExpressionConstructorImpls(ctx, lm)
	registerExpressionGetterSetterImpls(ctx, lm)
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
}
