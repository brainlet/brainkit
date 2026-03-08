// Ported from: packages/ai/src/test/mock-speech-model-v3.ts
package testutil

import (
	sm "github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
)

// MockSpeechModelV3 is a test double for a SpeechModel with specificationVersion "v3".
type MockSpeechModelV3 struct {
	ProviderID string
	ModelIDVal string

	DoGenerateFunc func(options sm.CallOptions) (sm.GenerateResult, error)
}

// MockSpeechModelV3Options configures a MockSpeechModelV3.
type MockSpeechModelV3Options struct {
	Provider   string
	ModelID    string
	DoGenerate func(options sm.CallOptions) (sm.GenerateResult, error)
}

// NewMockSpeechModelV3 creates a new MockSpeechModelV3 with the given options.
func NewMockSpeechModelV3(opts ...MockSpeechModelV3Options) *MockSpeechModelV3 {
	var o MockSpeechModelV3Options
	if len(opts) > 0 {
		o = opts[0]
	}

	m := &MockSpeechModelV3{
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
		m.DoGenerateFunc = func(_ sm.CallOptions) (sm.GenerateResult, error) {
			panic("not implemented")
		}
	}

	return m
}

func (m *MockSpeechModelV3) SpecificationVersion() string { return "v3" }
func (m *MockSpeechModelV3) Provider() string             { return m.ProviderID }
func (m *MockSpeechModelV3) ModelID() string              { return m.ModelIDVal }

func (m *MockSpeechModelV3) DoGenerate(options sm.CallOptions) (sm.GenerateResult, error) {
	return m.DoGenerateFunc(options)
}
