// Ported from: packages/ai/src/test/mock-reranking-model-v3.ts
package testutil

import (
	rm "github.com/brainlet/brainkit/ai-kit/provider/rerankingmodel"
)

// MockRerankingModelV3 is a test double for a RerankingModel with specificationVersion "v3".
type MockRerankingModelV3 struct {
	ProviderID string
	ModelIDVal string

	DoRerankFunc func(options rm.CallOptions) (rm.RerankResult, error)
}

// MockRerankingModelV3Options configures a MockRerankingModelV3.
type MockRerankingModelV3Options struct {
	Provider string
	ModelID  string
	DoRerank func(options rm.CallOptions) (rm.RerankResult, error)
}

// NewMockRerankingModelV3 creates a new MockRerankingModelV3 with the given options.
func NewMockRerankingModelV3(opts ...MockRerankingModelV3Options) *MockRerankingModelV3 {
	var o MockRerankingModelV3Options
	if len(opts) > 0 {
		o = opts[0]
	}

	m := &MockRerankingModelV3{
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

func (m *MockRerankingModelV3) SpecificationVersion() string { return "v3" }
func (m *MockRerankingModelV3) Provider() string             { return m.ProviderID }
func (m *MockRerankingModelV3) ModelID() string              { return m.ModelIDVal }

func (m *MockRerankingModelV3) DoRerank(options rm.CallOptions) (rm.RerankResult, error) {
	return m.DoRerankFunc(options)
}
