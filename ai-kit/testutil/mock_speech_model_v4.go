// Ported from: packages/ai/src/test/mock-speech-model-v4.ts
package testutil

import (
	sm "github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
)

// MockSpeechModelV4 is a test double for a SpeechModel with specificationVersion "v4".
type MockSpeechModelV4 struct {
	ProviderID string
	ModelIDVal string

	DoGenerateFunc func(options sm.CallOptions) (sm.GenerateResult, error)
}

// MockSpeechModelV4Options configures a MockSpeechModelV4.
type MockSpeechModelV4Options struct {
	Provider   string
	ModelID    string
	DoGenerate func(options sm.CallOptions) (sm.GenerateResult, error)
}

// NewMockSpeechModelV4 creates a new MockSpeechModelV4 with the given options.
func NewMockSpeechModelV4(opts ...MockSpeechModelV4Options) *MockSpeechModelV4 {
	var o MockSpeechModelV4Options
	if len(opts) > 0 {
		o = opts[0]
	}

	m := &MockSpeechModelV4{
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

func (m *MockSpeechModelV4) SpecificationVersion() string { return "v4" }
func (m *MockSpeechModelV4) Provider() string             { return m.ProviderID }
func (m *MockSpeechModelV4) ModelID() string              { return m.ModelIDVal }

func (m *MockSpeechModelV4) DoGenerate(options sm.CallOptions) (sm.GenerateResult, error) {
	return m.DoGenerateFunc(options)
}
