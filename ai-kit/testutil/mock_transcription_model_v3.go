// Ported from: packages/ai/src/test/mock-transcription-model-v3.ts
package testutil

import (
	tm "github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
)

// MockTranscriptionModelV3 is a test double for a TranscriptionModel with specificationVersion "v3".
type MockTranscriptionModelV3 struct {
	ProviderID string
	ModelIDVal string

	DoGenerateFunc func(options tm.CallOptions) (tm.GenerateResult, error)
}

// MockTranscriptionModelV3Options configures a MockTranscriptionModelV3.
type MockTranscriptionModelV3Options struct {
	Provider   string
	ModelID    string
	DoGenerate func(options tm.CallOptions) (tm.GenerateResult, error)
}

// NewMockTranscriptionModelV3 creates a new MockTranscriptionModelV3 with the given options.
func NewMockTranscriptionModelV3(opts ...MockTranscriptionModelV3Options) *MockTranscriptionModelV3 {
	var o MockTranscriptionModelV3Options
	if len(opts) > 0 {
		o = opts[0]
	}

	m := &MockTranscriptionModelV3{
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

func (m *MockTranscriptionModelV3) SpecificationVersion() string { return "v3" }
func (m *MockTranscriptionModelV3) Provider() string             { return m.ProviderID }
func (m *MockTranscriptionModelV3) ModelID() string              { return m.ModelIDVal }

func (m *MockTranscriptionModelV3) DoGenerate(options tm.CallOptions) (tm.GenerateResult, error) {
	return m.DoGenerateFunc(options)
}
