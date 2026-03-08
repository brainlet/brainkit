// Ported from: packages/ai/src/test/mock-transcription-model-v4.ts
package testutil

import (
	tm "github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
)

// MockTranscriptionModelV4 is a test double for a TranscriptionModel with specificationVersion "v4".
type MockTranscriptionModelV4 struct {
	ProviderID string
	ModelIDVal string

	DoGenerateFunc func(options tm.CallOptions) (tm.GenerateResult, error)
}

// MockTranscriptionModelV4Options configures a MockTranscriptionModelV4.
type MockTranscriptionModelV4Options struct {
	Provider   string
	ModelID    string
	DoGenerate func(options tm.CallOptions) (tm.GenerateResult, error)
}

// NewMockTranscriptionModelV4 creates a new MockTranscriptionModelV4 with the given options.
func NewMockTranscriptionModelV4(opts ...MockTranscriptionModelV4Options) *MockTranscriptionModelV4 {
	var o MockTranscriptionModelV4Options
	if len(opts) > 0 {
		o = opts[0]
	}

	m := &MockTranscriptionModelV4{
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

func (m *MockTranscriptionModelV4) SpecificationVersion() string { return "v4" }
func (m *MockTranscriptionModelV4) Provider() string             { return m.ProviderID }
func (m *MockTranscriptionModelV4) ModelID() string              { return m.ModelIDVal }

func (m *MockTranscriptionModelV4) DoGenerate(options tm.CallOptions) (tm.GenerateResult, error) {
	return m.DoGenerateFunc(options)
}
