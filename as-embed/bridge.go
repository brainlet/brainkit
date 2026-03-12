package asembed

import "github.com/fastschema/qjs"

func RegisterMemoryBridge(ctx *qjs.Context, lm *LinearMemory) {
	ctx.SetFunc("_malloc", func(this *qjs.This) (*qjs.Value, error) {
		size := int(this.Args()[0].Int32())
		return this.Context().NewInt32(int32(lm.Malloc(size))), nil
	})

	ctx.SetFunc("_free", func(this *qjs.This) (*qjs.Value, error) {
		ptr := int(this.Args()[0].Int32())
		lm.Free(ptr)
		return this.Context().NewInt32(0), nil
	})

	ctx.SetFunc("__i32_store", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		addr := int(args[0].Int32())
		// Use Float64 to preserve full 64-bit pointer values on ARM64.
		// Int32() would truncate Binaryen pointers that exceed 32 bits.
		val := uint64(args[1].Float64())
		lm.I32StorePtr(addr, val)
		return this.Context().NewInt32(0), nil
	})

	ctx.SetFunc("__i32_store8", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		lm.I32Store8(int(args[0].Int32()), byte(args[1].Int32()))
		return this.Context().NewInt32(0), nil
	})

	ctx.SetFunc("__i32_store16", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		lm.I32Store16(int(args[0].Int32()), uint16(args[1].Int32()))
		return this.Context().NewInt32(0), nil
	})

	ctx.SetFunc("__i32_load", func(this *qjs.This) (*qjs.Value, error) {
		addr := int(this.Args()[0].Int32())
		// Use I32LoadPtr to return full 64-bit pointer values on ARM64.
		return this.Context().NewFloat64(float64(lm.I32LoadPtr(addr))), nil
	})

	ctx.SetFunc("__i32_load8_u", func(this *qjs.This) (*qjs.Value, error) {
		return this.Context().NewInt32(int32(lm.I32Load8U(int(this.Args()[0].Int32())))), nil
	})

	ctx.SetFunc("__i32_load8_s", func(this *qjs.This) (*qjs.Value, error) {
		return this.Context().NewInt32(int32(lm.I32Load8S(int(this.Args()[0].Int32())))), nil
	})

	ctx.SetFunc("__i32_load16_u", func(this *qjs.This) (*qjs.Value, error) {
		return this.Context().NewInt32(int32(lm.I32Load16U(int(this.Args()[0].Int32())))), nil
	})

	ctx.SetFunc("__i32_load16_s", func(this *qjs.This) (*qjs.Value, error) {
		return this.Context().NewInt32(int32(lm.I32Load16S(int(this.Args()[0].Int32())))), nil
	})

	ctx.SetFunc("__f32_store", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		lm.F32Store(int(args[0].Int32()), float32(args[1].Float64()))
		return this.Context().NewInt32(0), nil
	})

	ctx.SetFunc("__f64_store", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		lm.F64Store(int(args[0].Int32()), args[1].Float64())
		return this.Context().NewInt32(0), nil
	})

	ctx.SetFunc("__f32_load", func(this *qjs.This) (*qjs.Value, error) {
		return this.Context().NewFloat64(float64(lm.F32Load(int(this.Args()[0].Int32())))), nil
	})

	ctx.SetFunc("__f64_load", func(this *qjs.This) (*qjs.Value, error) {
		return this.Context().NewFloat64(lm.F64Load(int(this.Args()[0].Int32()))), nil
	})
}
