// Ported from: packages/ai/src/test/mock-embedding-model-v4.ts
package testutil

import (
	em "github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
)

// MockEmbeddingModelV4 is a test double for an EmbeddingModel with specificationVersion "v4".
type MockEmbeddingModelV4 struct {
	ProviderID              string
	ModelIDVal              string
	MaxEmbeddingsPerCallVal *int
	ParallelCalls           bool

	DoEmbedFunc  func(options em.CallOptions) (em.Result, error)
	DoEmbedCalls []em.CallOptions
}

// MockEmbeddingModelV4Options configures a MockEmbeddingModelV4.
type MockEmbeddingModelV4Options struct {
	Provider              string
	ModelID               string
	MaxEmbeddingsPerCall  *int
	SupportsParallelCalls bool
	DoEmbed               interface{} // func(em.CallOptions)(em.Result,error) | em.Result | []em.Result
}

// NewMockEmbeddingModelV4 creates a new MockEmbeddingModelV4 with the given options.
func NewMockEmbeddingModelV4(opts ...MockEmbeddingModelV4Options) *MockEmbeddingModelV4 {
	var o MockEmbeddingModelV4Options
	if len(opts) > 0 {
		o = opts[0]
	}

	defaultMax := 1
	m := &MockEmbeddingModelV4{
		ProviderID:              "mock-provider",
		ModelIDVal:              "mock-model-id",
		MaxEmbeddingsPerCallVal: &defaultMax,
		ParallelCalls:           false,
		DoEmbedCalls:            []em.CallOptions{},
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

	switch v := o.DoEmbed.(type) {
	case func(em.CallOptions) (em.Result, error):
		m.DoEmbedFunc = v
	case em.Result:
		m.DoEmbedFunc = func(_ em.CallOptions) (em.Result, error) { return v, nil }
	case []em.Result:
		m.DoEmbedFunc = func(_ em.CallOptions) (em.Result, error) {
			idx := len(m.DoEmbedCalls)
			if idx < len(v) {
				return v[idx], nil
			}
			return v[len(v)-1], nil
		}
	default:
		m.DoEmbedFunc = func(_ em.CallOptions) (em.Result, error) {
			panic("not implemented")
		}
	}

	return m
}

func (m *MockEmbeddingModelV4) SpecificationVersion() string         { return "v4" }
func (m *MockEmbeddingModelV4) Provider() string                     { return m.ProviderID }
func (m *MockEmbeddingModelV4) ModelID() string                      { return m.ModelIDVal }
func (m *MockEmbeddingModelV4) MaxEmbeddingsPerCall() (*int, error)  { return m.MaxEmbeddingsPerCallVal, nil }
func (m *MockEmbeddingModelV4) SupportsParallelCalls() (bool, error) { return m.ParallelCalls, nil }

func (m *MockEmbeddingModelV4) DoEmbed(options em.CallOptions) (em.Result, error) {
	m.DoEmbedCalls = append(m.DoEmbedCalls, options)
	return m.DoEmbedFunc(options)
}
