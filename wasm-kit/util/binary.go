package util

import (
	"encoding/binary"
	"math"
)

func ReadI8(buffer []byte, offset int) int32 {
	return int32(buffer[offset])
}

func WriteI8(value int32, buffer []byte, offset int) {
	buffer[offset] = byte(value)
}

func ReadI16(buffer []byte, offset int) int32 {
	return int32(binary.LittleEndian.Uint16(buffer[offset:]))
}

func WriteI16(value int32, buffer []byte, offset int) {
	binary.LittleEndian.PutUint16(buffer[offset:], uint16(value))
}

func ReadI32(buffer []byte, offset int) int32 {
	return int32(binary.LittleEndian.Uint32(buffer[offset:]))
}

func WriteI32(value int32, buffer []byte, offset int) {
	binary.LittleEndian.PutUint32(buffer[offset:], uint32(value))
}

func WriteI32AsI64(value int32, buffer []byte, offset int, unsigned bool) {
	WriteI32(value, buffer, offset)
	if unsigned || value >= 0 {
		WriteI32(0, buffer, offset+4)
	} else {
		WriteI32(-1, buffer, offset+4)
	}
}

func ReadI64(buffer []byte, offset int) int64 {
	return int64(binary.LittleEndian.Uint64(buffer[offset:]))
}

func WriteI64(value int64, buffer []byte, offset int) {
	binary.LittleEndian.PutUint64(buffer[offset:], uint64(value))
}

func WriteI64AsI32(value int64, buffer []byte, offset int, unsigned bool) {
	if unsigned {
		if value < 0 || value > 0xFFFFFFFF {
			panic("i64 value does not fit in unsigned i32")
		}
	} else {
		if value < -0x80000000 || value > 0x7FFFFFFF {
			panic("i64 value does not fit in signed i32")
		}
	}
	WriteI32(int32(value), buffer, offset)
}

func ReadF32(buffer []byte, offset int) float32 {
	bits := binary.LittleEndian.Uint32(buffer[offset:])
	return math.Float32frombits(bits)
}

func WriteF32(value float32, buffer []byte, offset int) {
	bits := math.Float32bits(value)
	binary.LittleEndian.PutUint32(buffer[offset:], bits)
}

func ReadF64(buffer []byte, offset int) float64 {
	bits := binary.LittleEndian.Uint64(buffer[offset:])
	return math.Float64frombits(bits)
}

func WriteF64(value float64, buffer []byte, offset int) {
	bits := math.Float64bits(value)
	binary.LittleEndian.PutUint64(buffer[offset:], bits)
}

func ReadV128(buffer []byte, offset int) [16]byte {
	var result [16]byte
	copy(result[:], buffer[offset:offset+16])
	return result
}

func WriteV128(value [16]byte, buffer []byte, offset int) {
	copy(buffer[offset:offset+16], value[:])
}
