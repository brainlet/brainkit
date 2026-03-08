// Ported from: packages/ai/src/test/mock-image-model-v2.ts
package testutil

import (
	im "github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
)

// MockImageModelV2 is a test double for an ImageModel with specificationVersion "v2".
type MockImageModelV2 struct {
	ProviderID          string
	ModelIDVal          string
	MaxImagesPerCallVal *int

	DoGenerateFunc func(options im.CallOptions) (im.GenerateResult, error)
}

// MockImageModelV2Options configures a MockImageModelV2.
type MockImageModelV2Options struct {
	Provider         string
	ModelID          string
	MaxImagesPerCall *int
	DoGenerate       func(options im.CallOptions) (im.GenerateResult, error)
}

// NewMockImageModelV2 creates a new MockImageModelV2 with the given options.
func NewMockImageModelV2(opts ...MockImageModelV2Options) *MockImageModelV2 {
	var o MockImageModelV2Options
	if len(opts) > 0 {
		o = opts[0]
	}

	defaultMax := 1
	m := &MockImageModelV2{
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

func (m *MockImageModelV2) SpecificationVersion() string        { return "v2" }
func (m *MockImageModelV2) Provider() string                    { return m.ProviderID }
func (m *MockImageModelV2) ModelID() string                     { return m.ModelIDVal }
func (m *MockImageModelV2) MaxImagesPerCall() (*int, error)     { return m.MaxImagesPerCallVal, nil }

func (m *MockImageModelV2) DoGenerate(options im.CallOptions) (im.GenerateResult, error) {
	return m.DoGenerateFunc(options)
}
