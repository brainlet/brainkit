// Ported from: packages/ai/src/test/mock-video-model-v3.ts
package testutil

import (
	vm "github.com/brainlet/brainkit/ai-kit/provider/videomodel"
)

// MockVideoModelV3 is a test double for a VideoModel (Experimental_VideoModelV3)
// with specificationVersion "v3".
type MockVideoModelV3 struct {
	ProviderID          string
	ModelIDVal          string
	MaxVideosPerCallVal *int

	DoGenerateFunc func(options vm.CallOptions) (vm.GenerateResult, error)
}

// MockVideoModelV3Options configures a MockVideoModelV3.
type MockVideoModelV3Options struct {
	Provider         string
	ModelID          string
	MaxVideosPerCall *int
	DoGenerate       func(options vm.CallOptions) (vm.GenerateResult, error)
}

// NewMockVideoModelV3 creates a new MockVideoModelV3 with the given options.
func NewMockVideoModelV3(opts ...MockVideoModelV3Options) *MockVideoModelV3 {
	var o MockVideoModelV3Options
	if len(opts) > 0 {
		o = opts[0]
	}

	defaultMax := 1
	m := &MockVideoModelV3{
		ProviderID:          "mock-provider",
		ModelIDVal:          "mock-model-id",
		MaxVideosPerCallVal: &defaultMax,
	}

	if o.Provider != "" {
		m.ProviderID = o.Provider
	}
	if o.ModelID != "" {
		m.ModelIDVal = o.ModelID
	}
	if o.MaxVideosPerCall != nil {
		m.MaxVideosPerCallVal = o.MaxVideosPerCall
	}

	if o.DoGenerate != nil {
		m.DoGenerateFunc = o.DoGenerate
	} else {
		m.DoGenerateFunc = func(_ vm.CallOptions) (vm.GenerateResult, error) {
			panic("not implemented")
		}
	}

	return m
}

func (m *MockVideoModelV3) SpecificationVersion() string    { return "v3" }
func (m *MockVideoModelV3) Provider() string                { return m.ProviderID }
func (m *MockVideoModelV3) ModelID() string                 { return m.ModelIDVal }
func (m *MockVideoModelV3) MaxVideosPerCall() (*int, error) { return m.MaxVideosPerCallVal, nil }

func (m *MockVideoModelV3) DoGenerate(options vm.CallOptions) (vm.GenerateResult, error) {
	return m.DoGenerateFunc(options)
}
