// Ported from: packages/ai/src/test/mock-transcription-model-v2.ts
package testutil

import (
	tm "github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
)

// MockTranscriptionModelV2 is a test double for a TranscriptionModel with specificationVersion "v2".
type MockTranscriptionModelV2 struct {
	ProviderID string
	ModelIDVal string

	DoGenerateFunc func(options tm.CallOptions) (tm.GenerateResult, error)
}

// MockTranscriptionModelV2Options configures a MockTranscriptionModelV2.
type MockTranscriptionModelV2Options struct {
	Provider   string
	ModelID    string
	DoGenerate func(options tm.CallOptions) (tm.GenerateResult, error)
}

// NewMockTranscriptionModelV2 creates a new MockTranscriptionModelV2 with the given options.
func NewMockTranscriptionModelV2(opts ...MockTranscriptionModelV2Options) *MockTranscriptionModelV2 {
	var o MockTranscriptionModelV2Options
	if len(opts) > 0 {
		o = opts[0]
	}

	m := &MockTranscriptionModelV2{
		ProviderID: "mock-provider",
		ModelIDVal: "mock-model-id",
	}

	if o.Provider != "" {
		m.ProviderID = o.Provider
	}
	if o.ModelID != "" {
		m.ModelIDVal = o.ModelID
	}

	if o.DoGenerate != nil {
		m.DoGenerateFunc = o.DoGenerate
	} else {
		m.DoGenerateFunc = func(_ tm.CallOptions) (tm.GenerateResult, error) {
			panic("not implemented")
		}
	}

	return m
}

func (m *MockTranscriptionModelV2) SpecificationVersion() string { return "v2" }
func (m *MockTranscriptionModelV2) Provider() string             { return m.ProviderID }
func (m *MockTranscriptionModelV2) ModelID() string              { return m.ModelIDVal }

func (m *MockTranscriptionModelV2) DoGenerate(options tm.CallOptions) (tm.GenerateResult, error) {
	return m.DoGenerateFunc(options)
}
