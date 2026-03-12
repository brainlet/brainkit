package asembed

import "testing"

func TestLinearMemoryMalloc(t *testing.T) {
	lm := NewLinearMemory()
	ptr1 := lm.Malloc(16)
	if ptr1 != 8 {
		t.Errorf("first alloc = %d, want 8", ptr1)
	}
	ptr2 := lm.Malloc(16)
	if ptr2 != 24 {
		t.Errorf("second alloc = %d, want 24", ptr2)
	}
}

func TestLinearMemoryI32(t *testing.T) {
	lm := NewLinearMemory()
	ptr := lm.Malloc(8)
	lm.I32Store(ptr, 42)
	got := lm.I32Load(ptr)
	if got != 42 {
		t.Errorf("I32 round-trip: got %d, want 42", got)
	}
	lm.I32Store(ptr, -1)
	got = lm.I32Load(ptr)
	if got != -1 {
		t.Errorf("I32 negative: got %d, want -1", got)
	}
}

func TestLinearMemoryFloat(t *testing.T) {
	lm := NewLinearMemory()
	ptr := lm.Malloc(16)
	lm.F32Store(ptr, 3.14)
	f32 := lm.F32Load(ptr)
	if f32 < 3.13 || f32 > 3.15 {
		t.Errorf("F32 round-trip: got %f, want ~3.14", f32)
	}
	lm.F64Store(ptr+8, 2.718281828)
	f64 := lm.F64Load(ptr + 8)
	if f64 != 2.718281828 {
		t.Errorf("F64 round-trip: got %f, want 2.718281828", f64)
	}
}

func TestLinearMemoryString(t *testing.T) {
	lm := NewLinearMemory()
	ptr := lm.Malloc(32)
	lm.WriteString(ptr, "hello")
	got := lm.ReadString(ptr)
	if got != "hello" {
		t.Errorf("String round-trip: got %q, want %q", got, "hello")
	}
}

func TestLinearMemoryReset(t *testing.T) {
	lm := NewLinearMemory()
	lm.Malloc(1024)
	lm.Reset()
	ptr := lm.Malloc(16)
	if ptr != 8 {
		t.Errorf("after reset: got %d, want 8", ptr)
	}
}
