package jsbridge

import (
	"testing"
)

func TestWebAssemblyInstantiate(t *testing.T) {
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
	defer val.Free()

	if val.String() != "42" {
		t.Errorf("add(40, 2) = %s, want 42", val.String())
	}
	t.Logf("WebAssembly add(40, 2) = %s", val.String())
}
