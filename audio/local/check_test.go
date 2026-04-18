package local

import (
	"encoding/binary"
	"testing"
)

func TestSynthesizeSineWAVShape(t *testing.T) {
	wav := synthesizeSineWAV(440.0, 1_000_000_000, 24000, 2) // 1s at 24 kHz stereo
	if string(wav[0:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
		t.Fatalf("bad RIFF/WAVE header: %q / %q", wav[0:4], wav[8:12])
	}
	channels := binary.LittleEndian.Uint16(wav[22:24])
	rate := binary.LittleEndian.Uint32(wav[24:28])
	bits := binary.LittleEndian.Uint16(wav[34:36])
	dataSize := binary.LittleEndian.Uint32(wav[40:44])
	if channels != 2 || rate != 24000 || bits != 16 {
		t.Errorf("fmt chunk: ch=%d rate=%d bits=%d, want 2/24000/16", channels, rate, bits)
	}
	// 1 second of stereo PCM16 at 24 kHz = 24000 * 2 * 2 = 96000 bytes.
	if dataSize != 96000 {
		t.Errorf("data chunk size = %d, want 96000", dataSize)
	}
	if len(wav) != 44+96000 {
		t.Errorf("total wav bytes = %d, want %d", len(wav), 44+96000)
	}
}

func TestPeakPCM16(t *testing.T) {
	// Silence → peak 0.
	if got := peakPCM16(make([]byte, 2000)); got != 0 {
		t.Errorf("silent peak = %d, want 0", got)
	}

	// Single non-zero sample near the top of range.
	buf := make([]byte, 4)
	s1 := int16(-20000)
	s2 := int16(12345)
	binary.LittleEndian.PutUint16(buf[0:], uint16(s1))
	binary.LittleEndian.PutUint16(buf[2:], uint16(s2))
	if got := peakPCM16(buf); got != 20000 {
		t.Errorf("peak = %d, want 20000 (abs)", got)
	}

	// Odd-length trailing byte should be ignored, not crash.
	if got := peakPCM16([]byte{0, 0, 0x00, 0x7F, 0xAA}); got != 0x7F00 {
		t.Errorf("peak with odd tail = %d, want %d", got, 0x7F00)
	}
}

func TestCheckResultOK(t *testing.T) {
	// Empty → OK.
	if !(CheckResult{}).OK() {
		t.Error("zero CheckResult should be OK")
	}
	// Error fails.
	if (CheckResult{Err: errExample("x")}).OK() {
		t.Error("CheckResult with Err should not be OK")
	}
	// Warning fails.
	if (CheckResult{Warnings: []string{"system muted"}}).OK() {
		t.Error("CheckResult with warnings should not be OK")
	}
}

type errExample string

func (e errExample) Error() string { return string(e) }
