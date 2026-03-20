package asembed

import quickjs "github.com/buke/quickjs-go"

func RegisterMemoryBridge(ctx *quickjs.Context, lm *LinearMemory) {
	setFunc(ctx, "_malloc", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		size := int(args[0].ToInt32())
		return c.NewInt32(int32(lm.Malloc(size)))
	})

	setFunc(ctx, "_free", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		ptr := int(args[0].ToInt32())
		lm.Free(ptr)
		return c.NewInt32(0)
	})

	setFunc(ctx, "__i32_store", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		addr := int(args[0].ToInt32())
		// Use Float64 to preserve full 64-bit pointer values on ARM64.
		// Int32() would truncate Binaryen pointers that exceed 32 bits.
		val := uint64(args[1].ToFloat64())
		lm.I32StorePtr(addr, val)
		return c.NewInt32(0)
	})

	setFunc(ctx, "__i32_store8", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		lm.I32Store8(int(args[0].ToInt32()), byte(args[1].ToInt32()))
		return c.NewInt32(0)
	})

	setFunc(ctx, "__i32_store16", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		lm.I32Store16(int(args[0].ToInt32()), uint16(args[1].ToInt32()))
		return c.NewInt32(0)
	})

	setFunc(ctx, "__i32_load", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		addr := int(args[0].ToInt32())
		// Use I32LoadPtr to return full 64-bit pointer values on ARM64.
		return c.NewFloat64(float64(lm.I32LoadPtr(addr)))
	})

	setFunc(ctx, "__i32_load8_u", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return c.NewInt32(int32(lm.I32Load8U(int(args[0].ToInt32()))))
	})

	setFunc(ctx, "__i32_load8_s", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return c.NewInt32(int32(lm.I32Load8S(int(args[0].ToInt32()))))
	})

	setFunc(ctx, "__i32_load16_u", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return c.NewInt32(int32(lm.I32Load16U(int(args[0].ToInt32()))))
	})

	setFunc(ctx, "__i32_load16_s", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return c.NewInt32(int32(lm.I32Load16S(int(args[0].ToInt32()))))
	})

	setFunc(ctx, "__f32_store", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		lm.F32Store(int(args[0].ToInt32()), float32(args[1].ToFloat64()))
		return c.NewInt32(0)
	})

	setFunc(ctx, "__f64_store", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		lm.F64Store(int(args[0].ToInt32()), args[1].ToFloat64())
		return c.NewInt32(0)
	})

	setFunc(ctx, "__f32_load", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return c.NewFloat64(float64(lm.F32Load(int(args[0].ToInt32()))))
	})

	setFunc(ctx, "__f64_load", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return c.NewFloat64(lm.F64Load(int(args[0].ToInt32())))
	})
}
