// Ported from: packages/ai/src/test/mock-speech-model-v2.ts
package testutil

import (
	sm "github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
)

// MockSpeechModelV2 is a test double for a SpeechModel with specificationVersion "v2".
type MockSpeechModelV2 struct {
	ProviderID string
	ModelIDVal string

	DoGenerateFunc func(options sm.CallOptions) (sm.GenerateResult, error)
}

// MockSpeechModelV2Options configures a MockSpeechModelV2.
type MockSpeechModelV2Options struct {
	Provider   string
	ModelID    string
	DoGenerate func(options sm.CallOptions) (sm.GenerateResult, error)
}

// NewMockSpeechModelV2 creates a new MockSpeechModelV2 with the given options.
func NewMockSpeechModelV2(opts ...MockSpeechModelV2Options) *MockSpeechModelV2 {
	var o MockSpeechModelV2Options
	if len(opts) > 0 {
		o = opts[0]
	}

	m := &MockSpeechModelV2{
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

func (m *MockSpeechModelV2) SpecificationVersion() string { return "v2" }
func (m *MockSpeechModelV2) Provider() string             { return m.ProviderID }
func (m *MockSpeechModelV2) ModelID() string              { return m.ModelIDVal }

func (m *MockSpeechModelV2) DoGenerate(options sm.CallOptions) (sm.GenerateResult, error) {
	return m.DoGenerateFunc(options)
}
