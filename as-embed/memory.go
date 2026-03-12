package asembed

import (
	"encoding/binary"
	"math"
	"sync"
)

type LinearMemory struct {
	data   []byte
	offset int
	mu     sync.Mutex
}

func NewLinearMemory() *LinearMemory {
	const size = 64 * 1024 * 1024
	return &LinearMemory{
		data:   make([]byte, size),
		offset: 8,
	}
}

func (lm *LinearMemory) Malloc(size int) int {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	aligned := (lm.offset + 7) &^ 7
	if aligned+size > len(lm.data) {
		panic("LinearMemory: out of memory")
	}
	ptr := aligned
	lm.offset = aligned + size
	return ptr
}

func (lm *LinearMemory) Free(ptr int) {}

func (lm *LinearMemory) Reset() {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.offset = 8
	for i := range lm.data {
		lm.data[i] = 0
	}
}

func (lm *LinearMemory) I32Store(addr, value int) {
	binary.LittleEndian.PutUint32(lm.data[addr:], uint32(value))
}

func (lm *LinearMemory) I32Load(addr int) int {
	return int(int32(binary.LittleEndian.Uint32(lm.data[addr:])))
}

func (lm *LinearMemory) I32Store8(addr int, value byte) {
	lm.data[addr] = value
}

func (lm *LinearMemory) I32Load8U(addr int) int {
	return int(lm.data[addr])
}

func (lm *LinearMemory) I32Load8S(addr int) int {
	return int(int8(lm.data[addr]))
}

func (lm *LinearMemory) I32Store16(addr int, value uint16) {
	binary.LittleEndian.PutUint16(lm.data[addr:], value)
}

func (lm *LinearMemory) I32Load16U(addr int) int {
	return int(binary.LittleEndian.Uint16(lm.data[addr:]))
}

func (lm *LinearMemory) I32Load16S(addr int) int {
	return int(int16(binary.LittleEndian.Uint16(lm.data[addr:])))
}

func (lm *LinearMemory) F32Store(addr int, value float32) {
	binary.LittleEndian.PutUint32(lm.data[addr:], math.Float32bits(value))
}

func (lm *LinearMemory) F32Load(addr int) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(lm.data[addr:]))
}

func (lm *LinearMemory) F64Store(addr int, value float64) {
	binary.LittleEndian.PutUint64(lm.data[addr:], math.Float64bits(value))
}

func (lm *LinearMemory) F64Load(addr int) float64 {
	return math.Float64frombits(binary.LittleEndian.Uint64(lm.data[addr:]))
}

func (lm *LinearMemory) ReadString(ptr int) string {
	end := ptr
	for end < len(lm.data) && lm.data[end] != 0 {
		end++
	}
	return string(lm.data[ptr:end])
}

func (lm *LinearMemory) WriteString(ptr int, s string) {
	copy(lm.data[ptr:], s)
	lm.data[ptr+len(s)] = 0
}

func (lm *LinearMemory) ReadBytes(ptr, length int) []byte {
	result := make([]byte, length)
	copy(result, lm.data[ptr:ptr+length])
	return result
}

func (lm *LinearMemory) WriteBytes(ptr int, data []byte) {
	copy(lm.data[ptr:], data)
}

func (lm *LinearMemory) DataSlice(ptr, length int) []byte {
	return lm.data[ptr : ptr+length]
}
