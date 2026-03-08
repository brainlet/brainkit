// Ported from: packages/ai/src/test/mock-image-model-v3.ts
package testutil

import (
	im "github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
)

// MockImageModelV3 is a test double for an ImageModel with specificationVersion "v3".
type MockImageModelV3 struct {
	ProviderID          string
	ModelIDVal          string
	MaxImagesPerCallVal *int

	DoGenerateFunc func(options im.CallOptions) (im.GenerateResult, error)
}

// MockImageModelV3Options configures a MockImageModelV3.
type MockImageModelV3Options struct {
	Provider         string
	ModelID          string
	MaxImagesPerCall *int
	DoGenerate       func(options im.CallOptions) (im.GenerateResult, error)
}

// NewMockImageModelV3 creates a new MockImageModelV3 with the given options.
func NewMockImageModelV3(opts ...MockImageModelV3Options) *MockImageModelV3 {
	var o MockImageModelV3Options
	if len(opts) > 0 {
		o = opts[0]
	}

	defaultMax := 1
	m := &MockImageModelV3{
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

func (m *MockImageModelV3) SpecificationVersion() string    { return "v3" }
func (m *MockImageModelV3) Provider() string                { return m.ProviderID }
func (m *MockImageModelV3) ModelID() string                 { return m.ModelIDVal }
func (m *MockImageModelV3) MaxImagesPerCall() (*int, error) { return m.MaxImagesPerCallVal, nil }

func (m *MockImageModelV3) DoGenerate(options im.CallOptions) (im.GenerateResult, error) {
	return m.DoGenerateFunc(options)
}
