// Ported from: packages/ai/src/test/mock-reranking-model-v4.ts
package testutil

import (
	rm "github.com/brainlet/brainkit/ai-kit/provider/rerankingmodel"
)

// MockRerankingModelV4 is a test double for a RerankingModel with specificationVersion "v4".
type MockRerankingModelV4 struct {
	ProviderID string
	ModelIDVal string

	DoRerankFunc func(options rm.CallOptions) (rm.RerankResult, error)
}

// MockRerankingModelV4Options configures a MockRerankingModelV4.
type MockRerankingModelV4Options struct {
	Provider string
	ModelID  string
	DoRerank func(options rm.CallOptions) (rm.RerankResult, error)
}

// NewMockRerankingModelV4 creates a new MockRerankingModelV4 with the given options.
func NewMockRerankingModelV4(opts ...MockRerankingModelV4Options) *MockRerankingModelV4 {
	var o MockRerankingModelV4Options
	if len(opts) > 0 {
		o = opts[0]
	}

	m := &MockRerankingModelV4{
		ProviderID: "mock-provider",
		ModelIDVal: "mock-model-id",
	}

	if o.Provider != "" {
		m.ProviderID = o.Provider
	}
	if o.ModelID != "" {
		m.ModelIDVal = o.ModelID
	}

	if o.DoRerank != nil {
		m.DoRerankFunc = o.DoRerank
	} else {
		m.DoRerankFunc = func(_ rm.CallOptions) (rm.RerankResult, error) {
			panic("not implemented")
		}
	}

	return m
}

func (m *MockRerankingModelV4) SpecificationVersion() string { return "v4" }
func (m *MockRerankingModelV4) Provider() string             { return m.ProviderID }
func (m *MockRerankingModelV4) ModelID() string              { return m.ModelIDVal }

func (m *MockRerankingModelV4) DoRerank(options rm.CallOptions) (rm.RerankResult, error) {
	return m.DoRerankFunc(options)
}
