// Emulated linear memory for the Binaryen memory bridge.
//
// In the original Emscripten model, JS and C share a WebAssembly.Memory (a
// contiguous byte array). In our architecture, JS runs in QuickJS and Binaryen
// runs via CGo — there is no shared linear memory. This package provides a
// Go-side []byte that JS reads/writes through bridge functions, emulating the
// Emscripten memory model.

package main

import (
	"encoding/binary"
	"math"
	"sync"
)

// LinearMemory emulates the Emscripten linear memory as a Go byte slice.
// JS interacts with it via _malloc, __i32_store, __i32_load, etc.
// Go reads from it when forwarding calls to real Binaryen C functions.
type LinearMemory struct {
	data   []byte     // the "linear memory" (64MB)
	offset int        // bump allocator pointer
	mu     sync.Mutex // protects concurrent access
}

// NewLinearMemory allocates a 64MB linear memory region.
// The first 8 bytes are reserved (null pointer region), so offset starts at 8.
func NewLinearMemory() *LinearMemory {
	const size = 64 * 1024 * 1024 // 64MB
	return &LinearMemory{
		data:   make([]byte, size),
		offset: 8, // reserve first 8 bytes (ptr 0 = null)
	}
}

// Malloc allocates `size` bytes from the bump allocator, 8-byte aligned.
// Returns the offset into the linear memory (the "pointer" JS will use).
func (lm *LinearMemory) Malloc(size int) int {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Align to 8 bytes
	aligned := (lm.offset + 7) &^ 7
	if aligned+size > len(lm.data) {
		panic("LinearMemory: out of memory")
	}
	ptr := aligned
	lm.offset = aligned + size
	return ptr
}

// Free is a no-op for the bump allocator. Memory is freed when the entire
// LinearMemory is discarded (acceptable for compilation lifecycle).
func (lm *LinearMemory) Free(ptr int) {
	// no-op
}

// Reset resets the allocator to the initial state, effectively freeing all
// allocated memory. Useful between test runs.
func (lm *LinearMemory) Reset() {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.offset = 8
	// Zero out the data to avoid stale reads
	for i := range lm.data {
		lm.data[i] = 0
	}
}

// --- 32-bit integer operations ---

// I32Store writes a 32-bit integer at addr in little-endian format.
func (lm *LinearMemory) I32Store(addr, value int) {
	binary.LittleEndian.PutUint32(lm.data[addr:], uint32(value))
}

// I32Load reads a 32-bit integer from addr in little-endian format.
func (lm *LinearMemory) I32Load(addr int) int {
	return int(int32(binary.LittleEndian.Uint32(lm.data[addr:])))
}

// --- 8-bit operations ---

// I32Store8 writes a single byte at addr.
func (lm *LinearMemory) I32Store8(addr int, value byte) {
	lm.data[addr] = value
}

// I32Load8U reads an unsigned byte from addr.
func (lm *LinearMemory) I32Load8U(addr int) int {
	return int(lm.data[addr])
}

// I32Load8S reads a signed byte from addr.
func (lm *LinearMemory) I32Load8S(addr int) int {
	return int(int8(lm.data[addr]))
}

// --- 16-bit operations ---

// I32Store16 writes a 16-bit integer at addr in little-endian format.
func (lm *LinearMemory) I32Store16(addr int, value uint16) {
	binary.LittleEndian.PutUint16(lm.data[addr:], value)
}

// I32Load16U reads an unsigned 16-bit integer from addr.
func (lm *LinearMemory) I32Load16U(addr int) int {
	return int(binary.LittleEndian.Uint16(lm.data[addr:]))
}

// I32Load16S reads a signed 16-bit integer from addr.
func (lm *LinearMemory) I32Load16S(addr int) int {
	return int(int16(binary.LittleEndian.Uint16(lm.data[addr:])))
}

// --- Float operations ---

// F32Store writes a 32-bit float at addr.
func (lm *LinearMemory) F32Store(addr int, value float32) {
	binary.LittleEndian.PutUint32(lm.data[addr:], math.Float32bits(value))
}

// F32Load reads a 32-bit float from addr.
func (lm *LinearMemory) F32Load(addr int) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(lm.data[addr:]))
}

// F64Store writes a 64-bit float at addr.
func (lm *LinearMemory) F64Store(addr int, value float64) {
	binary.LittleEndian.PutUint64(lm.data[addr:], math.Float64bits(value))
}

// F64Load reads a 64-bit float from addr.
func (lm *LinearMemory) F64Load(addr int) float64 {
	return math.Float64frombits(binary.LittleEndian.Uint64(lm.data[addr:]))
}

// --- String and byte operations ---

// ReadString reads a null-terminated UTF-8 string from the linear memory
// starting at ptr.
func (lm *LinearMemory) ReadString(ptr int) string {
	end := ptr
	for end < len(lm.data) && lm.data[end] != 0 {
		end++
	}
	return string(lm.data[ptr:end])
}

// WriteString writes a string plus null terminator to the linear memory at ptr.
func (lm *LinearMemory) WriteString(ptr int, s string) {
	copy(lm.data[ptr:], s)
	lm.data[ptr+len(s)] = 0
}

// ReadBytes reads `length` raw bytes from the linear memory at ptr.
func (lm *LinearMemory) ReadBytes(ptr, length int) []byte {
	result := make([]byte, length)
	copy(result, lm.data[ptr:ptr+length])
	return result
}

// WriteBytes writes raw bytes to the linear memory at ptr.
func (lm *LinearMemory) WriteBytes(ptr int, data []byte) {
	copy(lm.data[ptr:], data)
}

// DataSlice returns a direct slice into the linear memory at [ptr:ptr+length].
// Use with caution — the returned slice aliases the internal buffer.
func (lm *LinearMemory) DataSlice(ptr, length int) []byte {
	return lm.data[ptr : ptr+length]
}
