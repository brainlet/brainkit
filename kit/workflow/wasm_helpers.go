package workflow

import (
	"context"
	"encoding/binary"
	"fmt"
	"unicode/utf16"

	"github.com/tetratelabs/wazero/api"
)

// readASString reads an AssemblyScript string from WASM linear memory.
// AS strings are UTF-16LE encoded. rtSize at offset -4.
func readASString(m api.Module, ptr uint32) string {
	if ptr == 0 {
		return ""
	}
	mem := m.Memory()
	if mem == nil {
		return ""
	}
	rtSize, ok := mem.ReadUint32Le(ptr - 4)
	if !ok || rtSize == 0 {
		return ""
	}
	data, ok := mem.Read(ptr, rtSize)
	if !ok {
		return ""
	}
	if len(data) < 2 {
		return ""
	}
	u16s := make([]uint16, len(data)/2)
	for i := range u16s {
		u16s[i] = binary.LittleEndian.Uint16(data[i*2:])
	}
	return string(utf16.Decode(u16s))
}

// writeASString allocates an AS string in WASM memory and returns its pointer.
// Requires __new export (AS runtime). String class ID = 2.
func writeASString(ctx context.Context, m api.Module, s string) (uint32, error) {
	newFn := m.ExportedFunction("__new")
	if newFn == nil {
		return 0, fmt.Errorf("module does not export __new")
	}
	u16s := utf16.Encode([]rune(s))
	byteLen := len(u16s) * 2
	results, err := newFn.Call(ctx, uint64(byteLen), 2)
	if err != nil {
		return 0, fmt.Errorf("__new: %w", err)
	}
	ptr := uint32(results[0])
	data := make([]byte, byteLen)
	for i, c := range u16s {
		binary.LittleEndian.PutUint16(data[i*2:], c)
	}
	m.Memory().Write(ptr, data)
	return ptr, nil
}
