package bench_test

import (
	"testing"

	"github.com/brainlet/brainkit/sdk"
)

// BenchmarkEnvelopeEncode exercises EncodeEnvelope — the wire
// format every reply on the shared-inbox path traverses.
func BenchmarkEnvelopeEncode(b *testing.B) {
	env := sdk.EnvelopeOK(map[string]any{
		"ok":    true,
		"count": 42,
		"items": []string{"a", "b", "c"},
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := sdk.EncodeEnvelope(env); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEnvelopeDecode exercises the reverse path. Uses a
// pre-encoded payload so the measurement isolates decode cost
// from encode.
func BenchmarkEnvelopeDecode(b *testing.B) {
	payload, err := sdk.EncodeEnvelope(sdk.EnvelopeOK(map[string]any{
		"ok":    true,
		"count": 42,
		"items": []string{"a", "b", "c"},
	}))
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := sdk.DecodeEnvelope(payload); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEnvelopeRoundTrip measures encode+decode end-to-end —
// the combined cost per reply on the Caller path.
func BenchmarkEnvelopeRoundTrip(b *testing.B) {
	env := sdk.EnvelopeOK(map[string]any{
		"ok":    true,
		"count": 42,
		"items": []string{"a", "b", "c"},
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		payload, err := sdk.EncodeEnvelope(env)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := sdk.DecodeEnvelope(payload); err != nil {
			b.Fatal(err)
		}
	}
}
