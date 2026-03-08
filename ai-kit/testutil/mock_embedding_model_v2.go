// Ported from: packages/ai/src/test/mock-embedding-model-v2.ts
package testutil

import (
	em "github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
)

// MockEmbeddingModelV2 is a test double for an EmbeddingModel with specificationVersion "v2".
// The TS version is generic over VALUE; in Go, EmbeddingModel is always string-based.
type MockEmbeddingModelV2 struct {
	ProviderID             string
	ModelIDVal             string
	MaxEmbeddingsPerCallVal *int
	ParallelCalls          bool

	DoEmbedFunc func(options em.CallOptions) (em.Result, error)
}

// MockEmbeddingModelV2Options configures a MockEmbeddingModelV2.
type MockEmbeddingModelV2Options struct {
	Provider              string
	ModelID               string
	MaxEmbeddingsPerCall  *int
	SupportsParallelCalls bool
	DoEmbed               func(options em.CallOptions) (em.Result, error)
}

// NewMockEmbeddingModelV2 creates a new MockEmbeddingModelV2 with the given options.
func NewMockEmbeddingModelV2(opts ...MockEmbeddingModelV2Options) *MockEmbeddingModelV2 {
	var o MockEmbeddingModelV2Options
	if len(opts) > 0 {
		o = opts[0]
	}

	defaultMax := 1
	m := &MockEmbeddingModelV2{
		ProviderID:              "mock-provider",
		ModelIDVal:              "mock-model-id",
		MaxEmbeddingsPerCallVal: &defaultMax,
		ParallelCalls:           false,
	}

	if o.Provider != "" {
		m.ProviderID = o.Provider
	}
	if o.ModelID != "" {
		m.ModelIDVal = o.ModelID
	}
	if o.MaxEmbeddingsPerCall != nil {
		m.MaxEmbeddingsPerCallVal = o.MaxEmbeddingsPerCall
	}
	m.ParallelCalls = o.SupportsParallelCalls

	if o.DoEmbed != nil {
		m.DoEmbedFunc = o.DoEmbed
	} else {
		m.DoEmbedFunc = func(_ em.CallOptions) (em.Result, error) {
			panic("not implemented")
		}
	}

	return m
}

func (m *MockEmbeddingModelV2) SpecificationVersion() string            { return "v2" }
func (m *MockEmbeddingModelV2) Provider() string                        { return m.ProviderID }
func (m *MockEmbeddingModelV2) ModelID() string                         { return m.ModelIDVal }
func (m *MockEmbeddingModelV2) MaxEmbeddingsPerCall() (*int, error)     { return m.MaxEmbeddingsPerCallVal, nil }
func (m *MockEmbeddingModelV2) SupportsParallelCalls() (bool, error)    { return m.ParallelCalls, nil }

func (m *MockEmbeddingModelV2) DoEmbed(options em.CallOptions) (em.Result, error) {
	return m.DoEmbedFunc(options)
}
