// Ported from: packages/ai/src/test/mock-video-model-v4.ts
package testutil

import (
	vm "github.com/brainlet/brainkit/ai-kit/provider/videomodel"
)

// MockVideoModelV4 is a test double for a VideoModel (Experimental_VideoModelV4)
// with specificationVersion "v4".
type MockVideoModelV4 struct {
	ProviderID          string
	ModelIDVal          string
	MaxVideosPerCallVal *int

	DoGenerateFunc func(options vm.CallOptions) (vm.GenerateResult, error)
}

// MockVideoModelV4Options configures a MockVideoModelV4.
type MockVideoModelV4Options struct {
	Provider         string
	ModelID          string
	MaxVideosPerCall *int
	DoGenerate       func(options vm.CallOptions) (vm.GenerateResult, error)
}

// NewMockVideoModelV4 creates a new MockVideoModelV4 with the given options.
func NewMockVideoModelV4(opts ...MockVideoModelV4Options) *MockVideoModelV4 {
	var o MockVideoModelV4Options
	if len(opts) > 0 {
		o = opts[0]
	}

	defaultMax := 1
	m := &MockVideoModelV4{
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

func (m *MockVideoModelV4) SpecificationVersion() string    { return "v4" }
func (m *MockVideoModelV4) Provider() string                { return m.ProviderID }
func (m *MockVideoModelV4) ModelID() string                 { return m.ModelIDVal }
func (m *MockVideoModelV4) MaxVideosPerCall() (*int, error) { return m.MaxVideosPerCallVal, nil }

func (m *MockVideoModelV4) DoGenerate(options vm.CallOptions) (vm.GenerateResult, error) {
	return m.DoGenerateFunc(options)
}
