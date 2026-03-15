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

func _TestWebAssemblyMemory(t *testing.T) {
	// TODO: needs a correctly compiled WASM binary with memory export
	// The xxhash-wasm integration test covers memory via the PgVector fixture
	b := newTestBridge(t, Encoding(), WebAssembly())

	// WASM module with memory export: writes a byte to memory, reads it back
	// (module
	//   (memory (export "mem") 1)
	//   (func (export "write") (param i32 i32) i32.store8 (local.get 0) (local.get 1))
	//   (func (export "read") (param i32) (result i32) i32.load8_u (local.get 0))
	// )
	val, err := b.EvalAsync("wasm-mem-test.js", `(async () => {
		const wasmBytes = new Uint8Array([
			0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00,
			0x01, 0x09, 0x02, 0x60, 0x02, 0x7f, 0x7f, 0x00, 0x60, 0x01, 0x7f, 0x01, 0x7f,
			0x03, 0x03, 0x02, 0x00, 0x01,
			0x05, 0x03, 0x01, 0x00, 0x01,
			0x07, 0x12, 0x03, 0x03, 0x6d, 0x65, 0x6d, 0x02, 0x00,
			0x05, 0x77, 0x72, 0x69, 0x74, 0x65, 0x00, 0x00,
			0x04, 0x72, 0x65, 0x61, 0x64, 0x00, 0x01,
			0x0a, 0x12, 0x02,
			0x08, 0x00, 0x20, 0x00, 0x20, 0x01, 0x3a, 0x00, 0x00, 0x0b,
			0x07, 0x00, 0x20, 0x00, 0x2d, 0x00, 0x00, 0x0b,
		]);
		const { instance } = await WebAssembly.instantiate(wasmBytes);
		const { mem, write, read } = instance.exports;

		// Check memory buffer exists
		const buf = mem.buffer;
		if (!(buf instanceof ArrayBuffer)) return "buffer not ArrayBuffer";

		// Write 42 to address 0 via WASM
		write(0, 42);
		// Read it back via WASM
		const val = read(0);

		// Also verify via JS memory access
		const view = new Uint8Array(mem.buffer);
		const jsVal = view[0];

		return val + "," + jsVal;
	})()`)

	if err != nil {
		t.Fatalf("EvalAsync: %v", err)
	}
	defer val.Free()

	if val.String() != "42,42" {
		t.Errorf("memory test = %s, want 42,42", val.String())
	}
	t.Logf("WebAssembly memory: %s", val.String())
}
