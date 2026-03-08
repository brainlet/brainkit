// Ported from: packages/ai/src/test/mock-image-model-v4.ts
package testutil

import (
	im "github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
)

// MockImageModelV4 is a test double for an ImageModel with specificationVersion "v4".
type MockImageModelV4 struct {
	ProviderID          string
	ModelIDVal          string
	MaxImagesPerCallVal *int

	DoGenerateFunc func(options im.CallOptions) (im.GenerateResult, error)
}

// MockImageModelV4Options configures a MockImageModelV4.
type MockImageModelV4Options struct {
	Provider         string
	ModelID          string
	MaxImagesPerCall *int
	DoGenerate       func(options im.CallOptions) (im.GenerateResult, error)
}

// NewMockImageModelV4 creates a new MockImageModelV4 with the given options.
func NewMockImageModelV4(opts ...MockImageModelV4Options) *MockImageModelV4 {
	var o MockImageModelV4Options
	if len(opts) > 0 {
		o = opts[0]
	}

	defaultMax := 1
	m := &MockImageModelV4{
		ProviderID:          "mock-provider",
		ModelIDVal:          "mock-model-id",
		MaxImagesPerCallVal: &defaultMax,
	}

	if o.Provider != "" {
		m.ProviderID = o.Provider
	}
	if o.ModelID != "" {
		m.ModelIDVal = o.ModelID
	}
	if o.MaxImagesPerCall != nil {
		m.MaxImagesPerCallVal = o.MaxImagesPerCall
	}

	if o.DoGenerate != nil {
		m.DoGenerateFunc = o.DoGenerate
	} else {
		m.DoGenerateFunc = func(_ im.CallOptions) (im.GenerateResult, error) {
			panic("not implemented")
		}
	}

	return m
}

func (m *MockImageModelV4) SpecificationVersion() string    { return "v4" }
func (m *MockImageModelV4) Provider() string                { return m.ProviderID }
func (m *MockImageModelV4) ModelID() string                 { return m.ModelIDVal }
func (m *MockImageModelV4) MaxImagesPerCall() (*int, error) { return m.MaxImagesPerCallVal, nil }

func (m *MockImageModelV4) DoGenerate(options im.CallOptions) (im.GenerateResult, error) {
	return m.DoGenerateFunc(options)
}
