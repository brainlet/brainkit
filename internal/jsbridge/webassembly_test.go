package jsbridge

import (
	"context"
	"testing"
	"time"
)

func TestWASMWebAssemblyInstantiate(t *testing.T) {
	b := newTestBridge(t, WebAssembly())

	// Minimal WASM: (module (func (export "add") (param i32 i32) (result i32) local.get 0 local.get 1 i32.add))
	val, err := b.EvalAsync("wasm-test.js", `(async () => {
		const wasmBytes = new Uint8Array([
			0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00,
			0x01, 0x07, 0x01, 0x60, 0x02, 0x7f, 0x7f, 0x01, 0x7f,
			0x03, 0x02, 0x01, 0x00,
			0x07, 0x07, 0x01, 0x03, 0x61, 0x64, 0x64, 0x00, 0x00,
			0x0a, 0x09, 0x01, 0x07, 0x00, 0x20, 0x00, 0x20, 0x01, 0x6a, 0x0b,
		]);
		const { instance } = await WebAssembly.instantiate(wasmBytes);
		const result = instance.exports.add(40, 2);
		return String(result);
	})()`)

	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}

	result := val.String()
	val.Free()

	if result != "42" {
		t.Errorf("add(40, 2) = %s, want 42", result)
	}
	if got := resourceCount(b, "wasm.modules"); got != 1 {
		t.Fatalf("wasm module resources = %d, want 1 snapshot=%+v", got, b.DebugSnapshot())
	}
	closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := b.CloseContext(closeCtx); err != nil {
		t.Fatalf("CloseContext: %v snapshot=%+v", err, b.DebugSnapshot())
	}
	if got := resourceCount(b, "wasm.modules"); got != 0 {
		t.Fatalf("wasm module resources after close = %d, want 0 snapshot=%+v", got, b.DebugSnapshot())
	}
	if got := resourceCount(b, "wasm.runtimes"); got != 0 {
		t.Fatalf("wasm runtime resources after close = %d, want 0 snapshot=%+v", got, b.DebugSnapshot())
	}
	t.Logf("WebAssembly add(40, 2) = %s", result)
}
